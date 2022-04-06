package enclave

import (
	"sync"

	"github.com/obscuronet/obscuro-playground/go/obscuronode/nodecommon"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/common"

	"github.com/obscuronet/obscuro-playground/go/obscurocommon"
)

// TODO - Further generify this interface's methods
// TODO - Put this in a separate package to enclave

// DB lives purely in the encrypted memory space of an enclave.
// Unlike Storage, methods in this class should have minimal logic, to map them more easily to our chosen datastore.
type DB interface {
	// FetchBlockAndHeight returns the L1 block with the given hash, its height and true, or (nil, false) if no such block is stored
	FetchBlockAndHeight(hash obscurocommon.L1RootHash) (*blockAndHeight, bool)
	// StoreBlock persists the L1 block and its height in the chain
	StoreBlock(b *types.Block, height uint64)
	// FetchHeadBlock returns the L1 block at the head of the chain
	FetchHeadBlock() obscurocommon.L1RootHash

	// FetchRollup returns the rollup with the given hash and true, or (nil, false) if no such rollup is stored
	FetchRollup(hash obscurocommon.L2RootHash) (*Rollup, bool)
	// FetchRollups returns all the proposed rollups with the given height
	FetchRollups(height uint64) []*Rollup
	// StoreRollup persists the rollup
	StoreRollup(rollup *Rollup)

	// FetchBlockState returns the state after ingesting the L1 block with the given hash
	FetchBlockState(hash obscurocommon.L1RootHash) (*blockState, bool)
	// SetBlockState persists the state after ingesting the L1 block with the given hash
	SetBlockState(hash obscurocommon.L1RootHash, state *blockState)
	// SetBlockStateNewRollup persists the state after ingesting the L1 block with the given hash that contains a new rollup
	SetBlockStateNewRollup(hash obscurocommon.L1RootHash, state *blockState)
	// FetchRollupState returns the state after adding the rollup with the given hash
	FetchRollupState(hash obscurocommon.L2RootHash) State
	// SetRollupState persists the state after adding the rollup with the given hash
	SetRollupState(hash obscurocommon.L2RootHash, state State)

	// FetchMempoolTxs returns all L2 transactions in the mempool
	FetchMempoolTxs() []nodecommon.L2Tx
	// AddMempoolTx adds an L2 transaction to the mempool
	AddMempoolTx(tx nodecommon.L2Tx)
	// RemoveMempoolTxs removes any L2 transactions whose hash is keyed in the map from the mempool
	RemoveMempoolTxs(remove map[common.Hash]common.Hash)
	// FetchRollupTxs returns all transactions in a given rollup keyed by hash and true, or (nil, false) if the rollup is unknown
	FetchRollupTxs(r *Rollup) (map[common.Hash]nodecommon.L2Tx, bool)
	// StoreRollupTxs overwrites the transactions associated with a given rollup
	StoreRollupTxs(*Rollup, map[common.Hash]nodecommon.L2Tx)

	// FetchSecret returns the enclave's secret
	FetchSecret() SharedEnclaveSecret
	// StoreSecret stores a secret in the enclave
	StoreSecret(secret SharedEnclaveSecret)

	// StoreGenesisRollup stores the rollup genesis
	StoreGenesisRollup(rol *Rollup)
	// FetchGenesisRollup returns the rollup genesis
	FetchGenesisRollup() *Rollup
}

type blockAndHeight struct {
	b      *types.Block
	height uint64
}

type inMemoryDB struct {
	rollupGenesisHash common.Hash // TODO add lock protection, not needed atm

	stateMutex sync.RWMutex // Controls access to `statePerBlock`, `statePerRollup`, `headBlock`, `rollupsByHeight` and `rollups`
	mpMutex    sync.RWMutex // Controls access to `mempool`
	blockMutex sync.RWMutex // Controls access to `blockCache`
	txMutex    sync.RWMutex // Controls access to `txsPerRollupCache`

	statePerBlock     map[obscurocommon.L1RootHash]*blockState
	statePerRollup    map[obscurocommon.L2RootHash]State
	headBlock         obscurocommon.L1RootHash
	rollupsByHeight   map[uint64][]*Rollup
	rollups           map[obscurocommon.L2RootHash]*Rollup
	mempool           map[common.Hash]nodecommon.L2Tx
	blockCache        map[obscurocommon.L1RootHash]*blockAndHeight
	txsPerRollupCache map[obscurocommon.L2RootHash]map[common.Hash]nodecommon.L2Tx

	sharedEnclaveSecret SharedEnclaveSecret
}

func NewInMemoryDB() DB {
	return &inMemoryDB{
		statePerBlock:     make(map[obscurocommon.L1RootHash]*blockState),
		stateMutex:        sync.RWMutex{},
		rollupsByHeight:   make(map[uint64][]*Rollup),
		rollups:           make(map[obscurocommon.L2RootHash]*Rollup),
		mempool:           make(map[common.Hash]nodecommon.L2Tx),
		mpMutex:           sync.RWMutex{},
		statePerRollup:    make(map[obscurocommon.L2RootHash]State),
		blockCache:        map[obscurocommon.L1RootHash]*blockAndHeight{},
		blockMutex:        sync.RWMutex{},
		txsPerRollupCache: make(map[obscurocommon.L2RootHash]map[common.Hash]nodecommon.L2Tx),
		txMutex:           sync.RWMutex{},
	}
}

func (db *inMemoryDB) StoreGenesisRollup(rol *Rollup) {
	db.rollupGenesisHash = rol.Hash()
	db.StoreRollup(rol)
}

func (db *inMemoryDB) FetchGenesisRollup() *Rollup {
	r, _ := db.FetchRollup(db.rollupGenesisHash)
	return r
}

func (db *inMemoryDB) FetchBlockState(hash obscurocommon.L1RootHash) (*blockState, bool) {
	db.stateMutex.RLock()
	defer db.stateMutex.RUnlock()

	val, found := db.statePerBlock[hash]
	return val, found
}

func (db *inMemoryDB) SetBlockState(hash obscurocommon.L1RootHash, state *blockState) {
	db.stateMutex.Lock()
	defer db.stateMutex.Unlock()

	db.statePerBlock[hash] = state
	// todo - is there any other logic here?
	db.headBlock = hash
}

func (db *inMemoryDB) SetBlockStateNewRollup(hash obscurocommon.L1RootHash, state *blockState) {
	db.stateMutex.Lock()
	defer db.stateMutex.Unlock()

	db.statePerBlock[hash] = state
	db.rollups[state.head.Hash()] = state.head
	db.statePerRollup[state.head.Hash()] = state.state
	// todo - is there any other logic here?
	db.headBlock = hash
}

func (db *inMemoryDB) SetRollupState(hash obscurocommon.L2RootHash, state State) {
	db.stateMutex.Lock()
	defer db.stateMutex.Unlock()

	db.statePerRollup[hash] = state
}

func (db *inMemoryDB) FetchHeadBlock() obscurocommon.L1RootHash {
	return db.headBlock
}

// TODO - Pull this logic into the storage layer.
func (db *inMemoryDB) StoreRollup(rollup *Rollup) {
	db.stateMutex.Lock()
	defer db.stateMutex.Unlock()

	db.rollups[rollup.Hash()] = rollup
	val, found := db.rollupsByHeight[rollup.Header.Height]
	if found {
		db.rollupsByHeight[rollup.Header.Height] = append(val, rollup)
	} else {
		db.rollupsByHeight[rollup.Header.Height] = []*Rollup{rollup}
	}
}

func (db *inMemoryDB) FetchRollup(hash obscurocommon.L2RootHash) (*Rollup, bool) {
	db.stateMutex.RLock()
	defer db.stateMutex.RUnlock()

	r, f := db.rollups[hash]
	return r, f
}

func (db *inMemoryDB) FetchRollups(height uint64) []*Rollup {
	db.stateMutex.RLock()
	defer db.stateMutex.RUnlock()

	return db.rollupsByHeight[height]
}

func (db *inMemoryDB) FetchRollupState(hash obscurocommon.L2RootHash) State {
	db.stateMutex.RLock()
	defer db.stateMutex.RUnlock()

	return db.statePerRollup[hash]
}

func (db *inMemoryDB) AddMempoolTx(tx nodecommon.L2Tx) {
	db.mpMutex.Lock()
	defer db.mpMutex.Unlock()

	db.mempool[tx.Hash()] = tx
}

func (db *inMemoryDB) FetchMempoolTxs() []nodecommon.L2Tx {
	db.mpMutex.RLock()
	defer db.mpMutex.RUnlock()

	mpCopy := make([]nodecommon.L2Tx, 0)
	for _, tx := range db.mempool {
		mpCopy = append(mpCopy, tx)
	}
	return mpCopy
}

func (db *inMemoryDB) RemoveMempoolTxs(toRemove map[common.Hash]common.Hash) {
	db.mpMutex.Lock()
	defer db.mpMutex.Unlock()

	r := make(map[common.Hash]nodecommon.L2Tx)
	for id, t := range db.mempool {
		_, f := toRemove[id]
		if !f {
			r[id] = t
		}
	}
	db.mempool = r
}

func (db *inMemoryDB) StoreBlock(b *types.Block, height uint64) {
	db.blockMutex.Lock()
	defer db.blockMutex.Unlock()

	db.blockCache[b.Hash()] = &blockAndHeight{b: b, height: height}
}

func (db *inMemoryDB) FetchBlockAndHeight(hash obscurocommon.L1RootHash) (*blockAndHeight, bool) {
	db.blockMutex.RLock()
	defer db.blockMutex.RUnlock()

	val, f := db.blockCache[hash]
	return val, f
}

func (db *inMemoryDB) FetchRollupTxs(r *Rollup) (map[common.Hash]nodecommon.L2Tx, bool) {
	db.txMutex.RLock()
	defer db.txMutex.RUnlock()

	val, found := db.txsPerRollupCache[r.Hash()]
	return val, found
}

func (db *inMemoryDB) StoreRollupTxs(r *Rollup, newMap map[common.Hash]nodecommon.L2Tx) {
	db.txMutex.Lock()
	defer db.txMutex.Unlock()

	db.txsPerRollupCache[r.Hash()] = newMap
}

func (db *inMemoryDB) StoreSecret(secret SharedEnclaveSecret) {
	db.sharedEnclaveSecret = secret
}

func (db *inMemoryDB) FetchSecret() SharedEnclaveSecret {
	return db.sharedEnclaveSecret
}