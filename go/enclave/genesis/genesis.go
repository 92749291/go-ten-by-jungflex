package genesis

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/obscuronet/go-obscuro/go/common"
	"github.com/obscuronet/go-obscuro/go/enclave/db"

	gethcommon "github.com/ethereum/go-ethereum/common"
)

// Account specifies the address that's prefunded and the amount it's funded with
type Account struct {
	Address gethcommon.Address
	Amount  *big.Int
}

// Genesis holds a range of prefunded accounts
type Genesis struct {
	Accounts []Account
}

// New creates a new Genesis given a json string
// if the string is empty then it defaults to the testnet genesis
func New(genesisJSON string) (*Genesis, error) {
	// defaults to the testnet genesis
	if genesisJSON == "" {
		return &TestnetGenesis, nil
	}

	genesis := &Genesis{}
	err := json.Unmarshal([]byte(genesisJSON), genesis)
	if err != nil {
		return nil, err
	}
	return genesis, nil
}

func (g Genesis) CommitGenesisState(storage db.Storage) error {
	stateDB, err := g.applyAllocations(storage)
	if err != nil {
		return err
	}
	_, err = stateDB.Commit(true)
	if err != nil {
		return err
	}
	return nil
}

func (g Genesis) GetGenesisRoot(storage db.Storage) (*common.StateRoot, error) {
	stateDB, err := g.applyAllocations(storage)
	if err != nil {
		return nil, err
	}
	stateHash := stateDB.IntermediateRoot(true)
	return &stateHash, nil
}

// Applies the faucet preallocation on top of an empty state DB.
func (g Genesis) applyAllocations(storage db.Storage) (*state.StateDB, error) {
	s, err := storage.EmptyStateDB()
	if err != nil {
		return nil, fmt.Errorf("could not initialise empty state DB. Cause: %w", err)
	}

	// set the accounts funds
	for _, acc := range g.Accounts {
		s.SetBalance(acc.Address, acc.Amount)
	}

	return s, nil
}
