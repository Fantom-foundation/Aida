package utils

//go:generate mockgen -source state_root.go -destination state_root_mocks.go -package utils

import (
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

const stateRootPrefix = "dbh0x"

type StateHashProvider interface {
	GetStateHash(blockNumber int) (common.Hash, error)
}

func MakeStateRootProvider(db ethdb.Database) StateHashProvider {
	return &StateRootProvider{db}
}

type StateRootProvider struct {
	db ethdb.Database
}

func (p *StateRootProvider) GetStateHash(number int) (common.Hash, error) {
	hex := strconv.FormatUint(uint64(number), 16)
	stateRoot, err := p.db.Get([]byte(stateRootPrefix + hex))
	if err != nil {
		return common.Hash{}, err
	}

	return common.Hash(stateRoot), nil
}
