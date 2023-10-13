package blockprofile

import (
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/profile/graphutil"
	"github.com/ethereum/go-ethereum/common"

	substate "github.com/Fantom-foundation/Substate"
)

// checkContext returns true if the context is consistent; otherwise false.
func (ctx *Context) checkContext() bool {
	return ctx.n == len(ctx.txAddresses) && ctx.n == len(ctx.tCompletion) && ctx.n == len(ctx.txDependencies)
}

// TestInterfere tests the interfere function
func TestInterfere(t *testing.T) {
	u := AddressSet{}
	v := AddressSet{}
	if interfere(u, v) {
		t.Errorf("Empty address sets must not interfere")
	}
	addr1 := common.HexToAddress("0xFC00FACE00000000000000000000000000000000")
	// Both sets u,v = {} and do not interfere
	u[addr1] = struct{}{}
	if interfere(u, v) || interfere(v, u) {
		t.Errorf("Empty address set with non-empty must not interfere")
	}
	v[addr1] = struct{}{}
	// Both sets u,v = {addr1} and do interfere
	if !interfere(u, v) || !interfere(v, u) {
		t.Errorf("Identical address sets interfere")
	}
	addr2 := common.HexToAddress("0xFC00FACE22200000000000000000000000000000")
	v[addr2] = struct{}{}
	addr3 := common.HexToAddress("0xFC00FACE44200000000000000000000000000000")
	v[addr3] = struct{}{}
	// Both sets u = {addr1}, v={addr1,addr2, addr3} and interfere
	if !interfere(u, v) || !interfere(v, u) {
		t.Errorf("Identical address sets interfere")
	}
	delete(v, addr1)
	// Both sets u = {addr1}, v={addr2, addr3} and do not interfere
	if interfere(u, v) || interfere(v, u) {
		t.Errorf("Disjoint address sets interfere")
	}
}

// TestConsistency checks the context consistency
func TestConsistency(t *testing.T) {
	u := AddressSet{}
	addr := common.HexToAddress("0xFC00FACE00000000000000000000000000000000")
	u[addr] = struct{}{}
	ctx := NewContext()
	ctx.n = 0
	if !ctx.checkContext() {
		t.Errorf("Consistent context state was not captured")
	}

	ctx = NewContext()
	ctx.tCompletion = TxTime{1}
	if ctx.checkContext() {
		t.Errorf("Inconsistent context state was not captured")
	}

	ctx = NewContext()
	ctx.txAddresses = TxAddresses{AddressSet{}}
	if ctx.checkContext() {
		t.Errorf("Inconsistent context state was not captured")
	}

	ctx = NewContext()
	ctx.txDependencies = graphutil.StrictPartialOrder{}
	ctx.txDependencies = append(ctx.txDependencies, graphutil.OrdinalSet{1: struct{}{}})
	if ctx.checkContext() {
		t.Errorf("Inconsistent context state was not captured")
	}

}

// TestCheckContext tests the consistency check
func TestCheckContext(t *testing.T) {
	ctx := NewContext()
	ctx.n = 1
	ctx.tCompletion = TxTime{1, 1}
	ctx.txAddresses = TxAddresses{AddressSet{}}
	ctx.txDependencies = graphutil.StrictPartialOrder{graphutil.OrdinalSet{}, graphutil.OrdinalSet{}}
	if ctx.checkContext() {
		t.Errorf("Inconsistent context was not caught")
	}
}

// TestEarliestToRunFirst tests the computation to calculate the earliest time to run for an empty block
func TestEarliestToRunFirst(t *testing.T) {
	ctx := NewContext()
	earliest := ctx.earliestTimeToRun(AddressSet{})
	if earliest != 0 {
		t.Errorf("Unexpected result")
	}
}

// TestEarliestToRunSimple tests the computation of the earliest time to run for a block with one transaction
func TestEarliestToRunSimple(t *testing.T) {
	ctx := NewContext()
	ctx.n = 1
	ctx.tCompletion = TxTime{1}
	addr := common.HexToAddress("0xFC00FACE00000000000000000000000000000000")
	ctx.txAddresses = TxAddresses{AddressSet{addr: struct{}{}}}
	ctx.txDependencies = graphutil.StrictPartialOrder{graphutil.OrdinalSet{}}
	earliest := ctx.earliestTimeToRun(AddressSet{})
	if earliest != 0 {
		t.Errorf("Unexpected result")
	}
}

// TestEarliestToRunSimple2 tests the computation of the earliest time to run for a block with one transaction
func TestEarliestToRunSimple2(t *testing.T) {
	ctx := NewContext()
	ctx.n = 1
	ctx.tCompletion = TxTime{100}
	addr := common.HexToAddress("0xFC00FACE00000000000000000000000000000000")
	ctx.txAddresses = TxAddresses{AddressSet{addr: struct{}{}}}
	ctx.txDependencies = graphutil.StrictPartialOrder{graphutil.OrdinalSet{}}
	earliest := ctx.earliestTimeToRun(AddressSet{addr: struct{}{}})
	if earliest != 100 {
		t.Errorf("Unexpected result")
	}
}

// TestEarliestToRunSimple3 tests the computation of the earliest time to run for a block with two transaction
func TestEarliestToRunSimple3(t *testing.T) {
	ctx := NewContext()
	ctx.n = 2
	ctx.tCompletion = TxTime{100, 50}
	addr := common.HexToAddress("0xFC00FACE00000000000000000000000000000000")
	ctx.txAddresses = TxAddresses{AddressSet{addr: struct{}{}}, AddressSet{addr: struct{}{}}}
	ctx.txDependencies = graphutil.StrictPartialOrder{graphutil.OrdinalSet{}}
	earliest := ctx.earliestTimeToRun(AddressSet{addr: struct{}{}})
	if earliest != 100 {
		t.Errorf("Unexpected result")
	}
}

// TestDependenciesEmpty tests finding the dependencies for an empty block
func TestDependenciesEmpty(t *testing.T) {
	ctx := NewContext()
	dependentOn := ctx.dependencies(AddressSet{})
	if len(dependentOn) != 0 {
		t.Errorf("Unexpected result")
	}
}

// TestDependenciesSimple test finding the dependencies for a block with one transaction
func TestDependenciesSmple(t *testing.T) {
	ctx := NewContext()
	ctx.n = 1
	ctx.tCompletion = TxTime{100}
	addr := common.HexToAddress("0xFC00FACE00000000000000000000000000000000")
	ctx.txAddresses = TxAddresses{AddressSet{addr: struct{}{}}}
	ctx.txDependencies = graphutil.StrictPartialOrder{graphutil.OrdinalSet{}}
	dependentOn := ctx.dependencies(AddressSet{})
	if len(dependentOn) != 0 {
		t.Errorf("Unexpected result")
	}
}

// TestDependenciesSimple2 tests finding the dependencies for a block with one transaction
func TestDependenciesSimple2(t *testing.T) {
	ctx := NewContext()
	ctx.n = 1
	ctx.tCompletion = TxTime{100}
	addr := common.HexToAddress("0xFC00FACE00000000000000000000000000000000")
	ctx.txAddresses = TxAddresses{AddressSet{addr: struct{}{}}}
	ctx.txDependencies = graphutil.StrictPartialOrder{graphutil.OrdinalSet{}}
	dependentOn := ctx.dependencies(AddressSet{addr: struct{}{}})
	if _, ok := dependentOn[0]; !ok {
		t.Errorf("Unexpected result")
	}
	if len(dependentOn) != 1 {
		t.Errorf("Unexpected result")
	}
}

func TestDependenciesSimple3(t *testing.T) {
	ctx := NewContext()
	ctx.n = 2
	ctx.tCompletion = TxTime{100, 50}
	addr := common.HexToAddress("0xFC00FACE00000000000000000000000000000000")
	ctx.txAddresses = TxAddresses{AddressSet{addr: struct{}{}}, AddressSet{addr: struct{}{}}}
	ctx.txDependencies = graphutil.StrictPartialOrder{graphutil.OrdinalSet{}, graphutil.OrdinalSet{0: struct{}{}}}
	dependentOn := ctx.dependencies(AddressSet{addr: struct{}{}})
	if _, ok := dependentOn[0]; !ok {
		t.Errorf("Unexpected result")
	}
	if _, ok := dependentOn[1]; !ok {
		t.Errorf("Unexpected result")
	}
	if len(dependentOn) != 2 {
		t.Errorf("Unexpected result")
	}
}

// TestFindTxAddresses tests finding contract/wallet addresses of a transaction
func TestFindTxAddresses(t *testing.T) {

	// test substate.Transaction with empty fields
	testTransaction := executor.State{
		Substate: &substate.Substate{
			InputAlloc:  substate.SubstateAlloc{},
			OutputAlloc: substate.SubstateAlloc{},
			Message:     &substate.SubstateMessage{},
		},
	}

	addresses := findTxAddresses(testTransaction)
	if len(addresses) != 0 {
		t.Errorf("Unexpected result")
	}

	// test substate.Transaction with 3 addresses
	addr1 := common.HexToAddress("0xFC00FACE00000000000000000000000000000001")
	addr2 := common.HexToAddress("0xFC00FACE00000000000000000000000000000002")
	addr3 := common.HexToAddress("0xFC00FACE00000000000000000000000000000003")
	addrs := []common.Address{addr1, addr2, addr3}
	testTransaction = executor.State{
		Substate: &substate.Substate{
			InputAlloc:  substate.SubstateAlloc{addr1: &substate.SubstateAccount{}},
			OutputAlloc: substate.SubstateAlloc{addr2: &substate.SubstateAccount{}, addr3: &substate.SubstateAccount{}},
			Message:     &substate.SubstateMessage{},
		},
	}
	addresses = findTxAddresses(testTransaction)
	if len(addresses) != 3 {
		t.Errorf("Unexpected result")
	}
	for _, addr := range addrs {
		if _, ok := addresses[addr]; !ok {
			t.Errorf("Unexpected result")
		}
	}

	// test if substate.Message.To == nil and substate.Message.From == zero
	var zero common.Address
	testTransaction = executor.State{
		Substate: &substate.Substate{
			InputAlloc:  substate.SubstateAlloc{addr1: &substate.SubstateAccount{}},
			OutputAlloc: substate.SubstateAlloc{addr2: &substate.SubstateAccount{}, addr1: &substate.SubstateAccount{}},
			Message: &substate.SubstateMessage{
				From: zero,
				To:   nil,
			},
		},
	}

	addresses = findTxAddresses(testTransaction)
	if len(addresses) != 2 {
		t.Errorf("Unexpected result")
	}

	if _, ok := addresses[zero]; ok {
		t.Errorf("Unexpected result")
	}
}

// TestProcessTransaction tests RecordTransaction
func TestRecordTransaction(t *testing.T) {
	ctx := NewContext()

	// construct first transaction
	addr1 := common.HexToAddress("0xFC00FACE00000000000000000000000000000001")
	addr2 := common.HexToAddress("0xFC00FACE00000000000000000000000000000002")
	addr3 := common.HexToAddress("0xFC00FACE00000000000000000000000000000003")
	tx := executor.State{
		Substate: &substate.Substate{
			InputAlloc:  substate.SubstateAlloc{addr1: &substate.SubstateAccount{}},
			OutputAlloc: substate.SubstateAlloc{addr2: &substate.SubstateAccount{}, addr3: &substate.SubstateAccount{}},
			Message: &substate.SubstateMessage{
				From: addr1,
				To:   &addr2,
			},
			Result: &substate.SubstateResult{
				GasUsed: 11111,
			},
		},
		Transaction: 1,
		Block:       0,
	}

	tTransaction1 := time.Duration(50)
	err := ctx.RecordTransaction(tx, tTransaction1)
	if err != nil {
		t.Errorf("Unexpected error occurred: err: %q", err)
	}
	if ctx.n != 1 {
		t.Errorf("Unexpected number of transactions")
	}
	if ctx.tCritical != tTransaction1 || ctx.tSequential != tTransaction1 {
		t.Errorf("Unexpected sequential and critial path time")
	}
	if len(ctx.tCompletion) != 1 || ctx.tCompletion[0] != 50 {
		fmt.Printf("%v\n", ctx.tCompletion)
		t.Errorf("Unexpected completion time")
	}
	if ctx.tOverheads == 0 {
		t.Errorf("RecordTransaction cannot be executed in zero time")
	}
	if len(ctx.tTransactions) != 1 {
		t.Errorf("invalid length of ctx.tTransactions")
	}
	if len(ctx.gasTransactions) != 1 {
		t.Errorf("invalid length of ctx.gasTransactions")
	}

	checkAddr := func(s AddressSet) bool {
		firstAddr := false
		secondAddr := false
		thirdAddr := false
		for key := range s {
			if key == addr1 {
				firstAddr = true
			} else if key == addr2 {
				secondAddr = true
			} else if key == addr3 {
				thirdAddr = true
			}
		}
		return firstAddr && secondAddr && thirdAddr
	}

	if len(ctx.txAddresses) == 1 && len(ctx.txAddresses[0]) == 3 {
		if !checkAddr(ctx.txAddresses[0]) {
			t.Errorf("Unexpected addresses")
		}
	} else {
		t.Errorf("Unexpected number of transaction addresses")
	}
	if len(ctx.txDependencies) != 1 || len(ctx.txDependencies[0]) != 0 {
		t.Errorf("Unexpected dependencies")
	}

	// construct second transaction
	tx2 := executor.State{
		Substate: &substate.Substate{
			InputAlloc:  substate.SubstateAlloc{addr1: &substate.SubstateAccount{}},
			OutputAlloc: substate.SubstateAlloc{addr2: &substate.SubstateAccount{}, addr3: &substate.SubstateAccount{}},
			Message: &substate.SubstateMessage{
				From: addr1,
				To:   &addr2,
			},
			Result: &substate.SubstateResult{
				GasUsed: 22222,
			},
		},
		Transaction: 2,
		Block:       0,
	}

	tTransaction2 := time.Duration(100)
	err = ctx.RecordTransaction(tx2, tTransaction2)
	if err != nil {
		t.Errorf("Unexpected error occurred: err: %q", err)
	}
	if ctx.n != 2 {
		t.Errorf("Unexpected number of transactions")
	}
	if ctx.tCritical != tTransaction1+tTransaction2 || ctx.tSequential != tTransaction1+tTransaction2 {
		t.Errorf("Unexpected sequential and critial path time")
	}
	if len(ctx.tCompletion) != 2 || ctx.tCompletion[0] != tTransaction1 || ctx.tCompletion[1] != tTransaction1+tTransaction2 {
		t.Errorf("Unexpected completion time")
	}
	if ctx.tOverheads == 0 {
		t.Errorf("RecordTransaction cannot be executed in zero time")
	}
	if len(ctx.tTransactions) != 2 {
		t.Errorf("invalid length of ctx.tTransactions")
	}

	if len(ctx.gasTransactions) != 2 {
		t.Errorf("invalid length of ctx.gasTransactions")
	}

	if len(ctx.txAddresses) == 2 && len(ctx.txAddresses[0]) == 3 && len(ctx.txAddresses[0]) == 3 {
		if !checkAddr(ctx.txAddresses[0]) {
			t.Errorf("Unexpected addresses in first transaction")
		}
		if !checkAddr(ctx.txAddresses[1]) {
			t.Errorf("Unexpected addresses in second transaction")
		}
	} else {
		t.Errorf("Unexpected number of transaction addresses")
	}
	if len(ctx.txDependencies) != 2 || len(ctx.txDependencies[0]) != 0 || len(ctx.txDependencies[1]) != 1 {
		fmt.Printf("%v\n", ctx.txDependencies)
		t.Errorf("Unexpected dependencies")
	}
}

// TestGetProfileDataEmpty tests the retrieval of profile data from an empty block
func TestGetProfileDataWith2Transactions(t *testing.T) {
	ctx := NewContext()
	tBlock := time.Duration(100)
	ctx.tOverheads = time.Duration(50)

	pd, err := ctx.GetProfileData(0, tBlock)
	if err != nil {
		t.Errorf("error occurred while processing a block, err: %q", err)
	}

	expTBlock := tBlock - ctx.tOverheads
	if pd.tBlock != expTBlock.Nanoseconds() {
		t.Errorf("tBlock does not match expected one")
	}

	expTCommit := expTBlock - ctx.tSequential
	if pd.tCommit != expTCommit.Nanoseconds() {
		t.Errorf("tCommit does not match expected one")
	}

	expSpeedup := float64(expTBlock) / float64(expTCommit+ctx.tCritical)
	if pd.speedup != expSpeedup {
		t.Errorf("speed up does not match expected one")
	}

	if pd.ubNumProc != int64(len(graphutil.MinChainCover(ctx.txDependencies))) {
		t.Errorf("ubNumProc does not match expected one")
	}

	if pd.numTx != 0 || len(ctx.tTransactions) != 0 {
		t.Errorf("Number of transactions should be zero")
	}

	// check for errors
	ctx.tOverheads = 200
	_, err = ctx.GetProfileData(0, time.Duration(100))
	if !errors.Is(err, errBlockOverheadTime) {
		t.Errorf("Error does not match expected one")
	}

	ctx.tOverheads = 0
	ctx.tSequential = 200
	_, err = ctx.GetProfileData(0, time.Duration(100))
	if !errors.Is(err, errBlockTxsTime) {
		t.Errorf("Error does not match expected one")
	}

	ctx = NewContext()
	tBlock = time.Duration(100)
	ctx.tOverheads = time.Duration(50)
	ctx.tTransactions = []time.Duration{11222223}
	ctx.gasTransactions = []uint64{921929192818, 23892818218}
	_, err = ctx.GetProfileData(0, tBlock)
	if !errors.Is(err, errInvalidLen) {
		t.Errorf("Error does not match expected one")
	}

	ctx = NewContext()
	tBlock = time.Duration(100)
	ctx.tOverheads = time.Duration(50)
	ctx.tTransactions = []time.Duration{11222223, 29282388, 92382837}
	ctx.gasTransactions = []uint64{111111, 111111, 111111}
	ctx.gasBlock = uint64(333333)
	expBlockGas := uint64(333333)
	pd, err = ctx.GetProfileData(0, tBlock)
	if err != nil {
		t.Errorf("error occurred while processing a block, err: %q", err)
	}
	if pd.gasBlock != expBlockGas {
		t.Errorf("error occurred while computing block gas, got: %d, expected: %d", pd.gasBlock, expBlockGas)
	}
}

func TestGetTransactionType(t *testing.T) {
	toAddr := common.HexToAddress("0xabcdef0000000000000000000000000000000001")
	fromAddr1 := common.HexToAddress("0xabcdef0000000000000000000000000000000002")
	fromAddr2 := common.HexToAddress("0x0000000000000000000000000000000000000000")

	testTransaction := executor.State{
		Substate: &substate.Substate{
			InputAlloc: substate.SubstateAlloc{},
			Message:    &substate.SubstateMessage{},
		},
		Transaction: 0,
	}

	// expect create type, to address is nil
	if tt := getTransactionType(testTransaction); tt != CreateTx {
		t.Errorf("incorrect transaction type, got: %v, expected %v", TypeLabel[tt], TypeLabel[CreateTx])
	}

	// expect transafer type
	testTransaction.Substate.Message.To = &toAddr
	// to address doesn't exist in input substate
	if tt := getTransactionType(testTransaction); tt != TransferTx {
		t.Errorf("incorrect transaction type, got: %v, expected %v", TypeLabel[tt], TypeLabel[TransferTx])
	}
	// to address exists in input substate but doesn't have byte-code
	testTransaction.Substate.InputAlloc[toAddr] = substate.NewSubstateAccount(1, big.NewInt(1), []byte{})
	if tt := getTransactionType(testTransaction); tt != TransferTx {
		t.Errorf("incorrect transaction type, got: %v, expected %v", TypeLabel[tt], TypeLabel[TransferTx])
	}

	// expect call type
	// to address exists in input substate and has byte-code
	testTransaction.Substate.InputAlloc[toAddr].Code = []byte{1, 2, 3, 4}
	testTransaction.Substate.Message.From = fromAddr1
	if tt := getTransactionType(testTransaction); tt != CallTx {
		t.Errorf("incorrect transaction type, got: %v, expected %v", TypeLabel[tt], TypeLabel[CallTx])
	}

	// expect epoch sealing type
	// from address 0 to an sfc address (with byte-code
	testTransaction.Substate.Message.From = fromAddr2
	testTransaction.Substate.InputAlloc[toAddr] = substate.NewSubstateAccount(1, big.NewInt(1), []byte{1, 2, 3, 4})
	if tt := getTransactionType(testTransaction); tt != MaintenanceTx {
		t.Errorf("incorrect transaction type, got: %v, expected %v", TypeLabel[tt], TypeLabel[MaintenanceTx])
	}
}
