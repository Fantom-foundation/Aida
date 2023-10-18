package utils

//go:generate mockgen -source state_hash.go -destination state_hash_mocks.go -package utils

import (
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

const stateHashPrefix = "dbh0x"

type StateHashProvider interface {
	GetStateHash(blockNumber int) (common.Hash, error)
}

func MakeStateHashProvider(db ethdb.Database) StateHashProvider {
	return &stateHashProvider{db}
}

type stateHashProvider struct {
	db ethdb.Database
}

func (p *stateHashProvider) GetStateHash(number int) (common.Hash, error) {
	hex := strconv.FormatUint(uint64(number), 16)
	stateRoot, err := p.db.Get([]byte(stateHashPrefix + hex))
	if err != nil {
		return common.Hash{}, err
	}

	return common.Hash(stateRoot), nil
}
