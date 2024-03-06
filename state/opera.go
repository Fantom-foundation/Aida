package state

import (
	"errors"
	"fmt"
	"math/big"
	"path"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/go-opera/cmd/opera/launcher"
	"github.com/Fantom-foundation/go-opera/gossip"
	"github.com/Fantom-foundation/go-opera/integration"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	operaUtils "github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	geth "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

// operaStateDB implements db using Opera database
type operaStateDB struct {
	db            *geth.StateDB
	stateReader   *gossip.EvmStateReader
	svc           *gossip.Service
	stateRoot     common.Hash
	block         uint64
	isArchiveMode bool
	gdb           *gossip.Store
	log           logger.Logger
}

// newOperaStateDB returns a new operaStateDB instance
func newOperaStateDB(db *geth.StateDB, stateReader *gossip.EvmStateReader, store *gossip.Store, stateRoot common.Hash, logLevel string) *operaStateDB {
	return &operaStateDB{
		gdb:         store,
		db:          db,
		stateReader: stateReader,
		stateRoot:   stateRoot,
		log:         logger.NewLogger(logLevel, "Opera StateDb"),
	}
}

// MakeOperaStateDB creates gossip.Store, gossip.Service and gossip.EvmStateReader and returns new operaStateDB
func MakeOperaStateDB(pathToDb, dbVariant, logLevel string) (StateDB, error) {
	store, err := makeNewStore(pathToDb, dbVariant)
	if err != nil {
		return nil, err
	}

	stateReader := gossip.NewEvmStateReader(store)

	db, err := stateReader.StateAt(common.Hash{})
	if err != nil {
		return nil, err
	}
	return newOperaStateDB(db, stateReader, store, common.Hash{}, logLevel), nil
}

// makeNewStore reads gossip.Store from db and returns it
func makeNewStore(pathToDb, dbVariant string) (*gossip.Store, error) {
	cacheRatio := cachescale.Ratio{
		Base:   uint64(launcher.DefaultCacheSize - launcher.ConstantCacheSize),
		Target: uint64(launcher.DefaultCacheSize - launcher.ConstantCacheSize),
	}

	// todo we might be able to extract this information from the db
	dbCfg, err := setDBConfig(dbVariant, cacheRatio)
	if err != nil {
		return nil, fmt.Errorf("cannot create db config; %v", err)
	}

	// first check the db is present
	if err := integration.CheckStateInitialized(path.Join(pathToDb, "chaindata"), dbCfg); err != nil {
		return nil, err
	}

	dbsList, _ := integration.SupportedDBs(path.Join(pathToDb, "chaindata"), dbCfg.RuntimeCache)

	multiRawDbs, err := integration.MakeDirectMultiProducer(dbsList, dbCfg.Routing)
	if err != nil {
		return nil, err
	}

	return gossip.NewStore(multiRawDbs, gossip.DefaultStoreConfig(cacheRatio)), nil
}

func setDBConfig(dbPreset string, cacheRatio cachescale.Func) (integration.DBsConfig, error) {
	switch dbPreset {
	case "pbl-1":
		return integration.Pbl1DBsConfig(cacheRatio.U64, uint64(operaUtils.MakeDatabaseHandles())), nil
	case "ldb-1":
		return integration.Ldb1DBsConfig(cacheRatio.U64, uint64(operaUtils.MakeDatabaseHandles())), nil
	case "legacy-ldb":
		return integration.LdbLegacyDBsConfig(cacheRatio.U64, uint64(operaUtils.MakeDatabaseHandles())), nil
	case "legacy-pbl":
		return integration.PblLegacyDBsConfig(cacheRatio.U64, uint64(operaUtils.MakeDatabaseHandles())), nil
	default:
		return integration.DBsConfig{}, errors.New("unknown db preset")
	}
}

func (s *operaStateDB) CreateAccount(addr common.Address) {
	s.db.CreateAccount(addr)
}
func (s *operaStateDB) Exist(addr common.Address) bool {
	return s.db.Exist(addr)
}
func (s *operaStateDB) Empty(addr common.Address) bool {
	return s.db.Empty(addr)
}

func (s *operaStateDB) Suicide(addr common.Address) bool {
	return s.db.Suicide(addr)
}
func (s *operaStateDB) HasSuicided(addr common.Address) bool {
	return s.db.HasSuicided(addr)
}

func (s *operaStateDB) GetBalance(addr common.Address) *big.Int {
	return s.db.GetBalance(addr)
}
func (s *operaStateDB) AddBalance(addr common.Address, value *big.Int) {
	s.db.AddBalance(addr, value)
}
func (s *operaStateDB) SubBalance(addr common.Address, value *big.Int) {
	s.db.SubBalance(addr, value)
}

func (s *operaStateDB) GetNonce(addr common.Address) uint64 {
	return s.db.GetNonce(addr)
}
func (s *operaStateDB) SetNonce(addr common.Address, value uint64) {
	s.db.SetNonce(addr, value)
}

func (s *operaStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	return s.db.GetCommittedState(addr, key)
}
func (s *operaStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	return s.db.GetState(addr, key)
}
func (s *operaStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	s.db.SetState(addr, key, value)
}

func (s *operaStateDB) GetCodeHash(addr common.Address) common.Hash {
	return s.db.GetCodeHash(addr)
}
func (s *operaStateDB) GetCode(addr common.Address) []byte {
	return s.db.GetCode(addr)
}
func (s *operaStateDB) SetCode(addr common.Address, code []byte) {
	s.db.SetCode(addr, code)
}
func (s *operaStateDB) GetCodeSize(addr common.Address) int {
	return s.db.GetCodeSize(addr)
}

func (s *operaStateDB) AddRefund(amount uint64) {
	s.db.AddRefund(amount)
}
func (s *operaStateDB) SubRefund(amount uint64) {
	s.db.SubRefund(amount)
}
func (s *operaStateDB) GetRefund() uint64 {
	return s.db.GetRefund()
}

func (s *operaStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.db.PrepareAccessList(sender, dest, precompiles, txAccesses)
}
func (s *operaStateDB) AddressInAccessList(addr common.Address) bool {
	return s.db.AddressInAccessList(addr)
}
func (s *operaStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return s.db.SlotInAccessList(addr, slot)
}
func (s *operaStateDB) AddAddressToAccessList(addr common.Address) {
	s.db.AddAddressToAccessList(addr)
}
func (s *operaStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.db.AddSlotToAccessList(addr, slot)
}

func (s *operaStateDB) AddLog(log *types.Log) {
	s.db.AddLog(log)
}
func (s *operaStateDB) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	return s.db.GetLogs(hash, blockHash)
}

func (s *operaStateDB) Snapshot() int {
	return s.db.Snapshot()
}
func (s *operaStateDB) RevertToSnapshot(id int) {
	s.db.RevertToSnapshot(id)
}

func (s *operaStateDB) BeginTransaction(number uint32) error {
	// ignored
	return nil
}
func (s *operaStateDB) EndTransaction() error {
	s.Finalise(true)
	return nil
}

func (s *operaStateDB) BeginBlock(number uint64) error {
	if err := s.openStateDB(); err != nil {
		s.log.Fatalf("cannot begin block; %v", err)
	}
	s.block = number
	return nil
}
func (s *operaStateDB) EndBlock() error {
	var err error
	s.stateRoot, err = s.Commit(true)
	if err != nil {
		s.log.Fatalf("cannot commit; %v", err)
	}

	// todo trie commit/cap
	return nil
}

func (s *operaStateDB) BeginSyncPeriod(number uint64) {
	// ignored
}
func (s *operaStateDB) EndSyncPeriod() {

	// todo what to do at the end of an epoch

}

func (s *operaStateDB) GetHash() (common.Hash, error) {
	return common.Hash{}, nil // not supported
}

func (s *operaStateDB) Error() error {
	return nil
}

func (s *operaStateDB) Close() error {
	hash, err := s.Commit(true)
	if err != nil {
		return err
	}

	db := s.db.Database().TrieDB()

	if err = db.Commit(hash, true, nil); err != nil {
		return err
	}

	return db.DiskDB().Close()
}

func (s *operaStateDB) StartBulkLoad(block uint64) BulkLoad {
	s.log.Fatal("bulkload not yet implemented for opera statedb")
	return nil
}

func (s *operaStateDB) GetArchiveState(block uint64) (NonCommittableStateDB, error) {
	header := common.Hash(s.gdb.GetBlock(idx.Block(block)).Root)
	state, err := s.stateReader.StateAt(header)
	if err != nil {
		return nil, err
	}

	return newOperaStateDB(state, s.stateReader, nil, header, "INFO"), nil
}

func (s *operaStateDB) GetArchiveBlockHeight() (uint64, bool, error) {
	return 0, false, fmt.Errorf("retrieving of the Archive's block height is not (yet) supported by this DB implementation")
}

func (s *operaStateDB) GetMemoryUsage() *MemoryUsage {
	s.log.Warning("GetMemoryUsage is not yet implemented.")
	return &MemoryUsage{uint64(0), nil}
}

func (s *operaStateDB) Prepare(thash common.Hash, ti int) {
	s.db.Prepare(thash, ti)
}
func (s *operaStateDB) AddPreimage(hash common.Hash, preimage []byte) {
	s.db.AddPreimage(hash, preimage)
}
func (s *operaStateDB) Finalise(deleteEmptyObjects bool) {
	s.db.Finalise(deleteEmptyObjects)
}
func (s *operaStateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	return s.db.IntermediateRoot(deleteEmptyObjects)
}
func (s *operaStateDB) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	return s.db.Commit(deleteEmptyObjects)
}
func (s *operaStateDB) ForEachStorage(addr common.Address, cb func(common.Hash, common.Hash) bool) error {
	return s.db.ForEachStorage(addr, cb)
}

func (s *operaStateDB) GetSubstatePostAlloc() txcontext.WorldState {
	return substatecontext.NewWorldState(s.db.GetSubstatePostAlloc())
}

func (s *operaStateDB) PrepareSubstate(substate txcontext.WorldState, block uint64) {
	// ignored
}

func (s *operaStateDB) GetShadowDB() StateDB {
	return nil
}

func (s *operaStateDB) openStateDB() error {
	var err error

	// open new StateDb
	s.db, err = s.stateReader.StateAt(common.Hash{})
	return err
}

func (s *operaStateDB) Release() error {
	// nothing to do
	return nil
}
