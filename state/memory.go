package state

import (
	"fmt"
	"math/big"

	geth "github.com/Fantom-foundation/Aida/substate-cli/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/substate"
)

func MakeGethInMemoryStateDB(variant string, block uint64) (StateDB, error) {
	if variant != "" {
		return nil, fmt.Errorf("unkown variant: %v", variant)
	}
	return &gethInMemoryStateDB{gethStateDB{db: geth.MakeInMemoryStateDB(&substate.SubstateAlloc{}, block)}}, nil
}

type gethInMemoryStateDB struct {
	gethStateDB
}

func (s *gethInMemoryStateDB) BeginBlockApply() error {
	return nil
}

func (s *gethInMemoryStateDB) Close() error {
	// Nothing to do.
	return nil
}

func (s *gethInMemoryStateDB) PrepareSubstate(substate *substate.SubstateAlloc, block uint64) {
	s.db = geth.MakeInMemoryStateDB(substate, block)
}

func (s *gethInMemoryStateDB) StartBulkLoad() BulkLoad {
	return &gethInMemoryBulkLoad{}
}

type gethInMemoryBulkLoad struct{}

func (l *gethInMemoryBulkLoad) CreateAccount(addr common.Address) {
	// ignored
}

func (l *gethInMemoryBulkLoad) SetBalance(addr common.Address, value *big.Int) {
	// ignored
}

func (l *gethInMemoryBulkLoad) SetNonce(addr common.Address, nonce uint64) {
	// ignored
}

func (l *gethInMemoryBulkLoad) SetState(addr common.Address, key common.Hash, value common.Hash) {
	// ignored
}

func (l *gethInMemoryBulkLoad) SetCode(addr common.Address, code []byte) {
	// ignored
}

func (l *gethInMemoryBulkLoad) Close() error {
	// ignored
	return nil
}
