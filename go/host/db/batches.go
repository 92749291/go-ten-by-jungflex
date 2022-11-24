package db

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/obscuronet/go-obscuro/go/common/errutil"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/obscuronet/go-obscuro/go/common"
)

// DB methods relating to batches.

// GetHeadBatchHeader returns the header of the node's current head batch.
func (db *DB) GetHeadBatchHeader() (*common.Header, error) {
	headBatchHash, err := db.readHeadBatchHash()
	if err != nil {
		return nil, err
	}
	return db.readBatchHeader(*headBatchHash)
}

// GetBatchHeader returns the batch header given the hash, or (nil, false) if no such header is found.
func (db *DB) GetBatchHeader(hash gethcommon.Hash) (*common.Header, error) {
	return db.readBatchHeader(hash)
}

// AddBatchHeader adds a batch's header to the known headers
func (db *DB) AddBatchHeader(batch *common.ExtBatch) error {
	b := db.kvStore.NewBatch()

	if err := db.writeBatchHeader(batch.Header); err != nil {
		return fmt.Errorf("could not write batch header. Cause: %w", err)
	}
	if err := db.writeBatch(batch); err != nil {
		return fmt.Errorf("could not write batch. Cause: %w", err)
	}
	if err := db.writeBatchTxHashes(b, batch.Header.Hash(), batch.TxHashes); err != nil {
		return fmt.Errorf("could not write batch transaction hashes. Cause: %w", err)
	}
	if err := db.writeBatchHash(b, batch.Header); err != nil {
		return fmt.Errorf("could not write batch hash. Cause: %w", err)
	}
	for _, txHash := range batch.TxHashes {
		if err := db.writeBatchNumber(b, batch.Header, txHash); err != nil {
			return fmt.Errorf("could not write batch number. Cause: %w", err)
		}
	}

	// There's a potential race here, but absolute accuracy of the number of transactions is not required.
	currentTotal, err := db.readTotalTransactions()
	if err != nil {
		return fmt.Errorf("could not retrieve total transactions. Cause: %w", err)
	}
	newTotal := big.NewInt(0).Add(currentTotal, big.NewInt(int64(len(batch.TxHashes))))
	err = db.writeTotalTransactions(b, newTotal)
	if err != nil {
		return fmt.Errorf("could not write total transactions. Cause: %w", err)
	}

	// update the head if the new height is greater than the existing one
	headBatchHeader, err := db.GetHeadBatchHeader()
	if err != nil && !errors.Is(err, errutil.ErrNotFound) {
		return fmt.Errorf("could not retrieve head batch header. Cause: %w", err)
	}
	if errors.Is(err, errutil.ErrNotFound) || headBatchHeader.Number.Int64() <= batch.Header.Number.Int64() {
		err = db.writeHeadBatchHash(batch.Header.Hash())
		if err != nil {
			return fmt.Errorf("could not write new head batch hash. Cause: %w", err)
		}
	}

	if err = b.Write(); err != nil {
		return fmt.Errorf("could not write batch to DB. Cause: %w", err)
	}

	return nil
}

// GetBatchHash returns the hash of a batch given its number, or (nil, false) if no such batch is found.
func (db *DB) GetBatchHash(number *big.Int) (*gethcommon.Hash, error) {
	return db.readBatchHash(number)
}

// GetBatchTxs returns the transaction hashes of the batch with the given hash, or (nil, false) if no such batch is
// found.
func (db *DB) GetBatchTxs(batchHash gethcommon.Hash) ([]gethcommon.Hash, error) {
	return db.readBatchTxHashes(batchHash)
}

// GetBatchNumber returns the number of the batch containing the given transaction hash, or (nil, false) if no such
// batch is found.
func (db *DB) GetBatchNumber(txHash gethcommon.Hash) (*big.Int, error) {
	return db.readBatchNumber(txHash)
}

// GetTotalTransactions returns the total number of batched transactions.
func (db *DB) GetTotalTransactions() (*big.Int, error) {
	return db.readTotalTransactions()
}

// GetBatch returns the batch with the given hash.
func (db *DB) GetBatch(batchHash gethcommon.Hash) (*common.ExtBatch, error) {
	return db.readBatch(batchHash)
}

// headerKey = batchHeaderPrefix  + hash
func batchHeaderKey(hash gethcommon.Hash) []byte {
	return append(batchHeaderPrefix, hash.Bytes()...)
}

// headerKey = batchPrefix  + hash
func batchKey(hash gethcommon.Hash) []byte {
	return append(batchPrefix, hash.Bytes()...)
}

// headerKey = batchHashPrefix + number
func batchHashKey(num *big.Int) []byte {
	return append(batchHashPrefix, []byte(num.String())...)
}

// headerKey = batchTxHashesPrefix + batch hash
func batchTxHashesKey(hash gethcommon.Hash) []byte {
	return append(batchTxHashesPrefix, hash.Bytes()...)
}

// headerKey = batchNumberPrefix + hash
func batchNumberKey(txHash gethcommon.Hash) []byte {
	return append(batchNumberPrefix, txHash.Bytes()...)
}

// Retrieves the batch header corresponding to the hash.
func (db *DB) readBatchHeader(hash gethcommon.Hash) (*common.Header, error) {
	// TODO - #1208 - Analyse this weird Has/Get pattern, here and in other part of the `db` package.
	f, err := db.kvStore.Has(batchHeaderKey(hash))
	if err != nil {
		return nil, err
	}
	if !f {
		return nil, errutil.ErrNotFound
	}
	data, err := db.kvStore.Get(batchHeaderKey(hash))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errutil.ErrNotFound
	}
	header := new(common.Header)
	if err := rlp.Decode(bytes.NewReader(data), header); err != nil {
		return nil, err
	}
	return header, nil
}

// Retrieves the hash of the head batch.
func (db *DB) readHeadBatchHash() (*gethcommon.Hash, error) {
	f, err := db.kvStore.Has(headBatch)
	if err != nil {
		return nil, err
	}
	if !f {
		return nil, errutil.ErrNotFound
	}
	value, err := db.kvStore.Get(headBatch)
	if err != nil {
		return nil, err
	}
	h := gethcommon.BytesToHash(value)
	return &h, nil
}

// Stores a batch header into the database.
func (db *DB) writeBatchHeader(header *common.Header) error {
	// Write the encoded header
	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		return err
	}
	key := batchHeaderKey(header.Hash())
	if err := db.kvStore.Put(key, data); err != nil {
		return err
	}
	return nil
}

// Stores the head batch header hash into the database.
func (db *DB) writeHeadBatchHash(val gethcommon.Hash) error {
	err := db.kvStore.Put(headBatch, val.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// Stores a batch's hash in the database, keyed by the batch's number.
func (db *DB) writeBatchHash(w ethdb.KeyValueWriter, header *common.Header) error {
	key := batchHashKey(header.Number)
	if err := w.Put(key, header.Hash().Bytes()); err != nil {
		return err
	}
	return nil
}

// Retrieves the hash for the batch with the given number, or (nil, false) if no such batch is found.
func (db *DB) readBatchHash(number *big.Int) (*gethcommon.Hash, error) {
	f, err := db.kvStore.Has(batchHashKey(number))
	if err != nil {
		return nil, err
	}
	if !f {
		return nil, errutil.ErrNotFound
	}
	data, err := db.kvStore.Get(batchHashKey(number))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errutil.ErrNotFound
	}
	hash := gethcommon.BytesToHash(data)
	return &hash, nil
}

// Returns the transaction hashes in the batch with the given hash, or (nil, false) if no such batch is found.
func (db *DB) readBatchTxHashes(batchHash common.L2RootHash) ([]gethcommon.Hash, error) {
	f, err := db.kvStore.Has(batchTxHashesKey(batchHash))
	if err != nil {
		return nil, err
	}
	if !f {
		return nil, errutil.ErrNotFound
	}

	data, err := db.kvStore.Get(batchTxHashesKey(batchHash))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errutil.ErrNotFound
	}

	txHashes := []gethcommon.Hash{}
	if err = rlp.Decode(bytes.NewReader(data), &txHashes); err != nil {
		return nil, err
	}

	return txHashes, nil
}

// Stores a batch's number in the database, keyed by the hash of a transaction in that rollup.
func (db *DB) writeBatchNumber(w ethdb.KeyValueWriter, header *common.Header, txHash gethcommon.Hash) error {
	key := batchNumberKey(txHash)
	if err := w.Put(key, header.Number.Bytes()); err != nil {
		return err
	}
	return nil
}

// Writes the transaction hashes against the batch containing them.
func (db *DB) writeBatchTxHashes(w ethdb.KeyValueWriter, batchHash common.L2RootHash, txHashes []gethcommon.Hash) error {
	data, err := rlp.EncodeToBytes(txHashes)
	if err != nil {
		return err
	}
	key := batchTxHashesKey(batchHash)
	if err = w.Put(key, data); err != nil {
		return err
	}
	return nil
}

// Retrieves the number of the batch containing the transaction with the given hash, or (nil, false) if no such batch
// is found.
func (db *DB) readBatchNumber(txHash gethcommon.Hash) (*big.Int, error) {
	f, err := db.kvStore.Has(batchNumberKey(txHash))
	if err != nil {
		return nil, err
	}
	if !f {
		return nil, errutil.ErrNotFound
	}
	data, err := db.kvStore.Get(batchNumberKey(txHash))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errutil.ErrNotFound
	}
	return big.NewInt(0).SetBytes(data), nil
}

// Retrieves the total number of rolled-up transactions.
func (db *DB) readTotalTransactions() (*big.Int, error) {
	f, err := db.kvStore.Has(totalTransactionsKey)
	if err != nil {
		return nil, err
	}
	if !f {
		return big.NewInt(0), nil
	}
	data, err := db.kvStore.Get(totalTransactionsKey)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return big.NewInt(0), nil
	}
	return big.NewInt(0).SetBytes(data), nil
}

// Stores the total number of transactions in the database.
func (db *DB) writeTotalTransactions(w ethdb.KeyValueWriter, newTotal *big.Int) error {
	err := w.Put(totalTransactionsKey, newTotal.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// Stores a batch into the database.
func (db *DB) writeBatch(batch *common.ExtBatch) error {
	// Write the encoded header
	data, err := rlp.EncodeToBytes(batch)
	if err != nil {
		return err
	}
	key := batchKey(batch.Header.Hash())
	if err := db.kvStore.Put(key, data); err != nil {
		return err
	}
	return nil
}

// Retrieves the batch corresponding to the hash.
func (db *DB) readBatch(hash gethcommon.Hash) (*common.ExtBatch, error) {
	f, err := db.kvStore.Has(batchKey(hash))
	if err != nil {
		return nil, err
	}
	if !f {
		return nil, errutil.ErrNotFound
	}
	data, err := db.kvStore.Get(batchKey(hash))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errutil.ErrNotFound
	}
	batch := new(common.ExtBatch)
	if err := rlp.Decode(bytes.NewReader(data), batch); err != nil {
		return nil, err
	}
	return batch, nil
}