package state

import (
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Aida/txcontext"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func MakeEmptyGethInMemoryStateDB(variant string) (StateDB, error) {
	if variant != "" {
		return nil, fmt.Errorf("unknown variant: %v", variant)
	}
	return MakeInMemoryStateDB(substatecontext.NewWorldState(make(substate.SubstateAlloc)), 0), nil
}

// MakeInMemoryStateDB creates a StateDB instance reflecting the state
// captured by the provided Substate allocation.
func MakeInMemoryStateDB(ws txcontext.WorldState, block uint64) StateDB {
	return &inMemoryStateDB{ws: ws, state: makeSnapshot(nil, 0), blockNum: block}
}

// inMemoryStateDB implements the interface of a state.StateDB and can be
// used as a fast, in-memory replacement of the state DB.
type inMemoryStateDB struct {
	ws               txcontext.WorldState
	state            *snapshot
	snapshot_counter int
	blockNum         uint64
}

type slot struct {
	addr common.Address
	key  common.Hash
}

type snapshot struct {
	parent *snapshot
	id     int

	touched           map[common.Address]int // Set of referenced accounts
	balances          map[common.Address]*big.Int
	nonces            map[common.Address]uint64
	codes             map[common.Address][]byte
	suicided          map[common.Address]int // Set of destructed accounts
	storage           map[slot]common.Hash
	accessed_accounts map[common.Address]int
	accessed_slots    map[slot]int
	logs              []*types.Log
	refund            uint64
	createdAccounts   map[common.Address]int
	touchedSlots      map[slot]int
}

func makeSnapshot(parent *snapshot, id int) *snapshot {
	var refund uint64
	if parent != nil {
		refund = parent.refund
	}
	return &snapshot{
		parent:            parent,
		id:                id,
		touched:           map[common.Address]int{},
		balances:          map[common.Address]*big.Int{},
		nonces:            map[common.Address]uint64{},
		codes:             map[common.Address][]byte{},
		suicided:          map[common.Address]int{},
		storage:           map[slot]common.Hash{},
		accessed_accounts: map[common.Address]int{},
		accessed_slots:    map[slot]int{},
		logs:              make([]*types.Log, 0),
		refund:            refund,
		createdAccounts:   map[common.Address]int{},
		touchedSlots:      map[slot]int{},
	}
}

func (db *inMemoryStateDB) CreateAccount(addr common.Address) {
	if db.blockNum > 46051750 {
		db.state.createdAccounts[addr] = 0
	}
}

func (db *inMemoryStateDB) SubBalance(addr common.Address, value *big.Int) {
	if value.Sign() == 0 {
		return
	}
	db.state.touched[addr] = 0
	db.state.balances[addr] = new(big.Int).Sub(db.GetBalance(addr), value)
}

func (db *inMemoryStateDB) AddBalance(addr common.Address, value *big.Int) {
	db.state.touched[addr] = 0
	db.state.balances[addr] = new(big.Int).Add(db.GetBalance(addr), value)
}

func (db *inMemoryStateDB) GetBalance(addr common.Address) *big.Int {
	for state := db.state; state != nil; state = state.parent {
		val, exists := state.balances[addr]
		if exists {
			return new(big.Int).Set(val)
		}
	}
	acc := db.ws.Get(addr)
	if acc == nil {
		return new(big.Int).Set(common.Big0)
	}
	return new(big.Int).Set(acc.GetBalance())
}

func (db *inMemoryStateDB) GetNonce(addr common.Address) uint64 {
	for state := db.state; state != nil; state = state.parent {
		val, exists := state.nonces[addr]
		if exists {
			return val
		}
	}
	acc := db.ws.Get(addr)
	if acc == nil {
		return 0
	}
	return acc.GetNonce()
}

func (db *inMemoryStateDB) SetNonce(addr common.Address, value uint64) {
	db.state.touched[addr] = 0
	db.state.nonces[addr] = value
}

func (db *inMemoryStateDB) GetCodeHash(addr common.Address) common.Hash {
	if !db.Exist(addr) {
		return common.Hash{}
	}
	return getHash(addr, db.GetCode(addr))
}

func (db *inMemoryStateDB) GetCode(addr common.Address) []byte {
	for state := db.state; state != nil; state = state.parent {
		val, exists := state.codes[addr]
		if exists {
			return val
		}
	}
	if !db.ws.Has(addr) {
		return []byte{}
	}
	return db.ws.Get(addr).GetCode()
}

func (db *inMemoryStateDB) SetCode(addr common.Address, code []byte) {
	db.state.touched[addr] = 0
	db.state.codes[addr] = code
}

func (db *inMemoryStateDB) GetCodeSize(addr common.Address) int {
	return len(db.GetCode(addr))
}

func (db *inMemoryStateDB) AddRefund(gas uint64) {
	db.state.refund += gas
}
func (db *inMemoryStateDB) SubRefund(gas uint64) {
	db.state.refund -= gas
}
func (db *inMemoryStateDB) GetRefund() uint64 {
	return db.state.refund
}

func (db *inMemoryStateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	if !db.ws.Has(addr) {
		return common.Hash{}
	}
	return db.ws.Get(addr).GetStorageAt(key)
}

func (db *inMemoryStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	//fmt.Printf("SLOAD: %v %v\n", addr, key)
	slot := slot{addr, key}
	for state := db.state; state != nil; state = state.parent {
		val, exists := state.storage[slot]
		if exists {
			return val
		}
	}

	if !db.ws.Has(addr) {
		db.state.storage[slot] = common.Hash{}
		return common.Hash{}
	}

	return db.ws.Get(addr).GetStorageAt(key)
}

func (db *inMemoryStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	db.state.touched[addr] = 0
	db.state.storage[slot{addr, key}] = value
}

func (db *inMemoryStateDB) Suicide(addr common.Address) bool {
	db.state.suicided[addr] = 0
	db.state.balances[addr] = new(big.Int) // Apparently when you die all your money is gone.
	return true
}
func (db *inMemoryStateDB) HasSuicided(addr common.Address) bool {
	for state := db.state; state != nil; state = state.parent {
		_, exists := state.suicided[addr]
		if exists {
			return true
		}
	}
	return false
}

func (db *inMemoryStateDB) Exist(addr common.Address) bool {
	for state := db.state; state != nil; state = state.parent {
		_, exists := state.touched[addr]
		if exists {
			return true
		}
	}
	return db.ws.Has(addr)
}

func (db *inMemoryStateDB) Empty(addr common.Address) bool {
	return db.GetNonce(addr) == 0 && db.GetBalance(addr).Sign() == 0 && db.GetCodeSize(addr) == 0
}

func (db *inMemoryStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	db.AddAddressToAccessList(sender)
	if dest != nil {
		db.AddAddressToAccessList(*dest)
		// If it's a create-tx, the destination will be added inside evm.create
	}
	for _, addr := range precompiles {
		db.AddAddressToAccessList(addr)
	}
	for _, el := range txAccesses {
		db.AddAddressToAccessList(el.Address)
		for _, key := range el.StorageKeys {
			db.AddSlotToAccessList(el.Address, key)
		}
	}
}
func (db *inMemoryStateDB) AddressInAccessList(addr common.Address) bool {
	for state := db.state; state != nil; state = state.parent {
		if _, present := state.accessed_accounts[addr]; present {
			return true
		}
	}
	return false
}
func (db *inMemoryStateDB) SlotInAccessList(addr common.Address, key common.Hash) (addressOk bool, slotOk bool) {
	addressOk = db.AddressInAccessList(addr)
	id := slot{addr, key}
	for state := db.state; state != nil; state = state.parent {
		if _, present := state.accessed_slots[id]; present {
			slotOk = true
			return
		}
	}
	return
}

func (db *inMemoryStateDB) AddAddressToAccessList(addr common.Address) {
	db.state.accessed_accounts[addr] = 0
}

func (db *inMemoryStateDB) AddSlotToAccessList(addr common.Address, key common.Hash) {
	db.AddAddressToAccessList(addr)
	db.state.accessed_slots[slot{addr, key}] = 0
	for state := db.state; state != nil; state = state.parent {
		if _, exists := state.createdAccounts[addr]; exists {
			state.touchedSlots[slot{addr, key}] = 0
		}
	}
}

func (db *inMemoryStateDB) RevertToSnapshot(id int) {
	for ; db.state != nil && db.state.id != id; db.state = db.state.parent {
		// nothing
	}
	if db.state == nil {
		panic(fmt.Errorf("unable to revert to snapshot %d", id))
	}
}

func (db *inMemoryStateDB) Snapshot() int {
	res := db.state.id
	db.snapshot_counter++
	db.state = makeSnapshot(db.state, db.snapshot_counter)
	return res
}

func (db *inMemoryStateDB) AddLog(log *types.Log) {
	db.state.logs = append(db.state.logs, log)
}

func (db *inMemoryStateDB) AddPreimage(common.Hash, []byte) {
	// ignored
	panic("not implemented")
}

func (db *inMemoryStateDB) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error {
	panic("not implemented")
	return nil
}

func (db *inMemoryStateDB) Prepare(common.Hash, int) {
	// nothing to do ...
}

func (db *inMemoryStateDB) Finalise(bool) {
	// nothing to do ...
}
func (db *inMemoryStateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	panic("not implemented")
}

func (db *inMemoryStateDB) Commit(deleteEmptyObjects bool) (common.Hash, error) {
	return common.Hash{}, nil
}

func collectLogs(s *snapshot) []*types.Log {
	if s == nil {
		return []*types.Log{}
	}
	logs := collectLogs(s.parent)
	logs = append(logs, s.logs...)
	return logs
}

func (db *inMemoryStateDB) GetLogs(txHash common.Hash, blockHash common.Hash) []*types.Log {
	// Since the in-memory stateDB is only to be used for a single
	// transaction, all logs are from the same transactions. But
	// those need to be collected in the right order (inverse order
	// snapshots).
	return collectLogs(db.state)
}

func (s *inMemoryStateDB) Error() error {
	// ignored
	return nil
}

func (db *inMemoryStateDB) getEffects() substate.SubstateAlloc {
	// todo this should return txcontext.WorldState
	// collect all modified accounts
	touched := map[common.Address]int{}
	for state := db.state; state != nil; state = state.parent {
		for addr := range state.touched {
			touched[addr] = 0
		}
	}

	// build state of all touched addresses
	res := make(substate.SubstateAlloc)
	for addr := range touched {
		cur := new(substate.SubstateAccount)
		cur.Nonce = db.GetNonce(addr)
		cur.Balance = db.GetBalance(addr)
		cur.Code = db.GetCode(addr)
		cur.Storage = make(map[common.Hash]common.Hash)

		reported := map[common.Hash]int{}
		for state := db.state; state != nil; state = state.parent {
			for key, value := range state.storage {
				if key.addr == addr {
					_, exist := reported[key.key]
					if !exist {
						reported[key.key] = 0
						cur.Storage[key.key] = value
					}
				}
			}
		}

		res[addr] = cur
	}

	return res
}

func (db *inMemoryStateDB) GetSubstatePostAlloc() txcontext.WorldState {
	// todo we should not copy the map
	// rn the inMemoryDb is broken and unused anyway, when fixed this should be reworked
	res := make(substate.SubstateAlloc)
	db.ws.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
		storage := make(map[common.Hash]common.Hash)
		acc.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
			storage[keyHash] = valueHash
		})
		res[addr] = &substate.SubstateAccount{
			Nonce:   acc.GetNonce(),
			Balance: acc.GetBalance(),
			Storage: storage,
			Code:    acc.GetCode(),
		}
	})

	// ... and extend with effects
	for key, value := range db.getEffects() {
		entry, exists := res[key]
		if !exists {
			res[key] = value
			continue
		}

		entry.Balance = new(big.Int).Set(value.Balance)
		entry.Nonce = value.Nonce
		entry.Code = value.Code
		for key, value := range value.Storage {
			entry.Storage[key] = value
		}
	}
	for state := db.state; state != nil; state = state.parent {
		for slot := range state.touchedSlots {
			if _, exist := res[slot.addr]; exist {
				if _, contain := res[slot.addr].Storage[slot.key]; !contain {
					res[slot.addr].Storage[slot.key] = common.Hash{}
				}
			}
		}
	}

	for key := range res {
		if db.HasSuicided(key) || db.Empty(key) {
			delete(res, key)
			continue
		}
	}

	return substatecontext.NewWorldState(res)
}

func (db *inMemoryStateDB) BeginTransaction(number uint32) error {
	// ignored
	return nil
}

func (db *inMemoryStateDB) EndTransaction() error {
	db.Finalise(true)
	return nil
}

func (db *inMemoryStateDB) BeginBlock(number uint64) error {
	db.blockNum = number
	return nil
}

func (db *inMemoryStateDB) EndBlock() error {
	// ignored
	return nil
}

func (db *inMemoryStateDB) BeginSyncPeriod(number uint64) {
	// ignored
}

func (db *inMemoryStateDB) EndSyncPeriod() {
	// ignored
}

func (s *inMemoryStateDB) GetHash() (common.Hash, error) {
	return common.Hash{}, nil // not a great hash function, but a valid one :)
}

func (db *inMemoryStateDB) Close() error {
	// Nothing to do.
	return nil
}

func (db *inMemoryStateDB) GetMemoryUsage() *MemoryUsage {
	// not supported yet
	return &MemoryUsage{uint64(0), nil}
}

func (db *inMemoryStateDB) GetArchiveState(block uint64) (NonCommittableStateDB, error) {
	return nil, fmt.Errorf("archive states are not (yet) supported by this DB implementation")
}

func (s *inMemoryStateDB) GetArchiveBlockHeight() (uint64, bool, error) {
	return 0, false, fmt.Errorf("archive states are not (yet) supported by this DB implementation")
}

func (db *inMemoryStateDB) PrepareSubstate(alloc txcontext.WorldState, block uint64) {
	db.ws = alloc
	db.state = makeSnapshot(nil, 0)
	db.blockNum = block
}

func (s *inMemoryStateDB) StartBulkLoad(uint64) (BulkLoad, error) {
	return &gethInMemoryBulkLoad{}, nil
}

func (s *inMemoryStateDB) GetShadowDB() StateDB {
	return nil
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
