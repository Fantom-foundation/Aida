package utils

//go:generate mockgen -source state_root.go -destination state_root_mocks.go -package utils

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/syndtr/goleveldb/leveldb"
)

const stateRootPrefix = "dbh0x"

type StateHashProvider interface {
	PreLoadStateHashes(first, last int) error
	LoadStateHashFromDb(number int) (common.Hash, error)
	GetStateHash(number int) common.Hash
	DeletePreLoadedStateHash(number int)
}

func MakeStateHashProvider(db ethdb.Database, log logger.Logger) StateHashProvider {
	return &stateHashProvider{
		db:  db,
		log: log,
	}
}

type stateHashProvider struct {
	db     ethdb.Database
	log    logger.Logger
	hashes map[int]common.Hash
}

// PreLoadStateHashes and return map with block number to hash starting with 0x prefix.
func (p *stateHashProvider) PreLoadStateHashes(first, last int) error {
	p.hashes = make(map[int]common.Hash)

	numberOfHashes := last - first
	sizeInMB := float64(numberOfHashes*common.HashLength) / 100000000

	p.log.Noticef("Preloading %v state hashes - this requires ~%.2f MB of memory.", numberOfHashes, sizeInMB)

	var err error
	for i := first; i <= last; i++ {

		p.hashes[i], err = p.LoadStateHashFromDb(i)
		if err != nil {
			if errors.Is(err, leveldb.ErrNotFound) {
				return errors.New("your aida-db does not contain state hashes - please update it")
			}
			return fmt.Errorf("cannot load state hash for block %v; %v", i, err)
		}
	}
	return nil
}

func (p *stateHashProvider) LoadStateHashFromDb(number int) (common.Hash, error) {
	hex := strconv.FormatUint(uint64(number), 16)
	stateRoot, err := p.db.Get([]byte(stateRootPrefix + hex))
	if err != nil {
		return common.Hash{}, err
	}

	return common.Hash(stateRoot), nil
}

func (p *stateHashProvider) GetStateHash(number int) common.Hash {
	return p.hashes[number]
}

func (p *stateHashProvider) DeletePreLoadedStateHash(number int) {
	delete(p.hashes, number)
}
