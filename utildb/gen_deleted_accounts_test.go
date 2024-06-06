package utildb

import (
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state/proxy"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_errorHandlerAllErrorsAreMerged(t *testing.T) {
	abort := utils.MakeEvent()
	errChan := make(chan error, 2)
	encounteredErrors := errorHandler(abort, errChan)
	errChan <- fmt.Errorf("error1")
	errChan <- fmt.Errorf("error2")
	close(errChan)
	got := <-encounteredErrors
	assert.Equal(t, "error1\nerror2", got.Error())
}

func Test_errorHandlerSendsAbortSignal(t *testing.T) {
	abort := utils.MakeEvent()
	errChan := make(chan error, 2)
	encounteredErrors := errorHandler(abort, errChan)
	errChan <- fmt.Errorf("error1")
	close(errChan)
	got := <-encounteredErrors
	assert.Equal(t, "error1", got.Error())

	select {
	case <-abort.Wait():
	default:
		t.Errorf("abort signal should be sent")
	}
}

func Test_errorHandlerResultGetsClosed(t *testing.T) {
	abort := utils.MakeEvent()
	errChan := make(chan error, 2)
	encounteredErrors := errorHandler(abort, errChan)
	close(errChan)
	err, ok := <-encounteredErrors
	if !ok {
		t.Errorf("encounteredErrors channel should be open with result")
	}
	if err != nil {
		t.Errorf("encounteredErrors shouldn't have any errors")
	}

	_, ok = <-encounteredErrors
	if ok {
		t.Errorf("encounteredErrors channel should be closed")
	}
}

func Test_launchWorkersParallelAbortsOnSignal(t *testing.T) {
	ctrl := gomock.NewController(t)
	processor := executor.NewMockTxProcessor(ctrl)

	wg := &sync.WaitGroup{}
	cfg := &utils.Config{Workers: 2, ChainID: 250}

	errChan := make(chan error)
	abort := utils.MakeEvent()

	testTx := makeTestTx(4)

	// channel for each worker to get tasks for processing
	workerInputChannels := make([]chan *substate.Transaction, cfg.Workers)
	for i := 0; i < cfg.Workers; i++ {
		workerInputChannels[i] = make(chan *substate.Transaction)
		go func(workerId int, workerIn chan *substate.Transaction) {
			for k := 0; ; k++ {
				index := workerId + k*cfg.Workers
				if index >= len(testTx) {
					break
				}
				workerIn <- &substate.Transaction{Block: uint64(index), Substate: testTx[index]}
			}
			close(workerIn)
		}(i, workerInputChannels[i])
	}

	processor.EXPECT().ProcessTransaction(gomock.Any(), 0, gomock.Any(), substatecontext.NewTxContext(testTx[0])).Return(nil, nil)
	processor.EXPECT().ProcessTransaction(gomock.Any(), 1, gomock.Any(), substatecontext.NewTxContext(testTx[1])).Return(nil, nil)
	processor.EXPECT().ProcessTransaction(gomock.Any(), 2, gomock.Any(), substatecontext.NewTxContext(testTx[2])).Return(nil, nil)
	// block 3 is missing because of aborting

	outPut := txProcessor(wg, cfg, workerInputChannels, processor, abort, errChan, nil)

	_, ok := <-outPut[0]
	if !ok {
		t.Fatalf("output channel should be open")
	}

	// wait for second worker processing
	time.Sleep(100 * time.Millisecond)

	// abort before allowing processing of block 3
	abort.Signal()

	wg.Wait()
	ctrl.Finish()
}

func Test_launchWorkersParallelCorrectOutputOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	processor := executor.NewMockTxProcessor(ctrl)

	wg := &sync.WaitGroup{}
	cfg := &utils.Config{Workers: utils.GetRandom(2, 5), ChainID: 250}

	errChan := make(chan error)

	testSize := cfg.Workers*utils.GetRandom(100, 1000) + utils.GetRandom(0, cfg.Workers-1)
	testTx := makeTestTx(testSize)
	if len(testTx) != testSize {
		t.Fatalf("internal test error: testTx size is incorrect")
	}

	// channel for each worker to get tasks for processing
	workerInputChannels := make([]chan *substate.Transaction, cfg.Workers)
	for i := 0; i < cfg.Workers; i++ {
		workerInputChannels[i] = make(chan *substate.Transaction)
		go func(workerId int, workerIn chan *substate.Transaction) {
			for k := 0; ; k++ {
				index := workerId + k*cfg.Workers
				if index >= len(testTx) {
					break
				}

				workerIn <- &substate.Transaction{Block: uint64(index), Substate: testTx[index]}
			}

			close(workerIn)
		}(i, workerInputChannels[i])
	}

	for i := 0; i < len(testTx); i++ {
		processor.EXPECT().ProcessTransaction(gomock.Any(), i, gomock.Any(), substatecontext.NewTxContext(testTx[i])).Return(nil, nil)
	}

	outPut := txProcessor(wg, cfg, workerInputChannels, processor, utils.MakeEvent(), errChan, nil)

	var orderCheck uint64 = 0

loop:
	for {
		for i := 0; i < cfg.Workers; i++ {
			r, ok := <-outPut[i]
			if !ok {
				if int(orderCheck) != testSize {
					t.Fatalf("results are missing got: %d, expected: %d", orderCheck, testSize)
				}
				break loop
			}
			if orderCheck != r.tx.Block {
				t.Fatalf("results are in incorrect order")
			}
			orderCheck++
		}
	}

	wg.Wait()
	ctrl.Finish()
}

func Test_readAccounts(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		arr := make([]proxy.ContractLiveliness, 0)
		deleteHistory := make(map[common.Address]bool)
		del, res := readAccounts(arr, &deleteHistory)
		if len(del) != 0 || len(res) != 0 {
			t.Fatalf("should return empty arrays")
		}
		if len(deleteHistory) != 0 {
			t.Fatalf("deleteHistory should be empty")
		}
	})

	t.Run("Deletion", func(t *testing.T) {
		arr := make([]proxy.ContractLiveliness, 0)
		deleteHistory := make(map[common.Address]bool)

		arr = append(arr, proxy.ContractLiveliness{Addr: common.HexToAddress("0x1"), IsDeleted: true})
		del, res := readAccounts(arr, &deleteHistory)
		if len(del) != 1 || len(res) != 0 {
			t.Fatalf("should return empty arrays")
		}
		if !deleteHistory[common.HexToAddress("0x1")] {
			t.Fatalf("deleteHistory should have 0x1 deleted")
		}
	})

	t.Run("DeletionAndResurrection", func(t *testing.T) {
		arr := make([]proxy.ContractLiveliness, 0)
		deleteHistory := make(map[common.Address]bool)

		arr = append(arr, proxy.ContractLiveliness{Addr: common.HexToAddress("0x1"), IsDeleted: true})
		arr = append(arr, proxy.ContractLiveliness{Addr: common.HexToAddress("0x1"), IsDeleted: false})
		del, res := readAccounts(arr, &deleteHistory)
		if len(del) != 0 || len(res) != 1 {
			t.Fatalf("should return empty deletion array and 1 resurrected")
		}
		if deleteHistory[common.HexToAddress("0x1")] {
			t.Fatalf("deleteHistory should have 0x1 resurrected")
		}
	})

	t.Run("DeletionResurrectionDeletion", func(t *testing.T) {
		arr := make([]proxy.ContractLiveliness, 0)
		deleteHistory := make(map[common.Address]bool)

		arr = append(arr, proxy.ContractLiveliness{Addr: common.HexToAddress("0x1"), IsDeleted: true})
		arr = append(arr, proxy.ContractLiveliness{Addr: common.HexToAddress("0x1"), IsDeleted: false})
		arr = append(arr, proxy.ContractLiveliness{Addr: common.HexToAddress("0x1"), IsDeleted: true})
		del, res := readAccounts(arr, &deleteHistory)
		if len(del) != 1 || len(res) != 0 {
			t.Fatalf("should return empty deletion array and 1 resurrected")
		}
		if !deleteHistory[common.HexToAddress("0x1")] {
			t.Fatalf("deleteHistory should have 0x1 deleted")
		}
	})

	t.Run("DeletionResurrectionSplit", func(t *testing.T) {
		arr := make([]proxy.ContractLiveliness, 0)
		deleteHistory := make(map[common.Address]bool)
		arr = append(arr, proxy.ContractLiveliness{Addr: common.HexToAddress("0x1"), IsDeleted: true})
		_, _ = readAccounts(arr, &deleteHistory)

		// second run
		arr2 := make([]proxy.ContractLiveliness, 0)
		arr2 = append(arr, proxy.ContractLiveliness{Addr: common.HexToAddress("0x1"), IsDeleted: false})
		del, res := readAccounts(arr2, &deleteHistory)
		if len(del) != 0 || len(res) != 1 {
			t.Fatalf("should return empty deletion array and 1 resurrected")
		}
		if deleteHistory[common.HexToAddress("0x1")] {
			t.Fatalf("deleteHistory should have 0x1 deleted")
		}
	})

	t.Run("ResurrectionDeletionResurrection", func(t *testing.T) {
		arr := make([]proxy.ContractLiveliness, 0)
		deleteHistory := make(map[common.Address]bool)
		deleteHistory[common.HexToAddress("0x1")] = false

		arr = append(arr, proxy.ContractLiveliness{Addr: common.HexToAddress("0x1"), IsDeleted: true})
		arr = append(arr, proxy.ContractLiveliness{Addr: common.HexToAddress("0x1"), IsDeleted: false})
		del, res := readAccounts(arr, &deleteHistory)
		if len(del) != 0 || len(res) != 1 {
			t.Fatalf("should return empty deletion array and 1 resurrected")
		}
		if deleteHistory[common.HexToAddress("0x1")] {
			t.Fatalf("deleteHistory should have 0x1 ressurected")
		}
	})
}

func Test_resultCollectorCorrectResultOrder(t *testing.T) {
	wg := &sync.WaitGroup{}
	cfg := &utils.Config{Workers: 100, ChainID: 250}

	// channel for each worker to get tasks for processing
	workerOutputChannels := make([]chan txLivelinessResult, cfg.Workers)
	for i := 0; i < cfg.Workers; i++ {
		workerOutputChannels[i] = make(chan txLivelinessResult)
		go func(workerId int, workerOut chan txLivelinessResult) {
			workerOut <- txLivelinessResult{liveliness: []proxy.ContractLiveliness{{Addr: common.HexToAddress(fmt.Sprintf("0x%x", workerId)), IsDeleted: true}}}
			close(workerOutputChannels[workerId])
		}(i, workerOutputChannels[i])
	}

	res := resultCollector(wg, cfg, workerOutputChannels, utils.MakeEvent())

	currentBlk := 0
	for r := range res {
		if !r.liveliness[0].IsDeleted {
			t.Fatalf("results are in incorrect order")
		}

		if r.liveliness[0].Addr != common.HexToAddress(fmt.Sprintf("0x%x", currentBlk)) {
			t.Fatalf("results are in incorrect order liveliness")
		}
		currentBlk++
	}
}

func Test_resultCollectorAbortsOnSignal(t *testing.T) {
	wg := &sync.WaitGroup{}
	cfg := &utils.Config{Workers: 100, ChainID: 250}

	abort := utils.MakeEvent()

	// channel for each worker to get tasks for processing
	workerOutputChannels := make([]chan txLivelinessResult, cfg.Workers)
	for i := 0; i < cfg.Workers; i++ {
		workerOutputChannels[i] = make(chan txLivelinessResult)
		go func(workerId int, workerOut chan txLivelinessResult) {
			workerOut <- txLivelinessResult{liveliness: []proxy.ContractLiveliness{{Addr: common.HexToAddress(fmt.Sprintf("0x%x", workerId)), IsDeleted: true}}}
			close(workerOutputChannels[workerId])
		}(i, workerOutputChannels[i])
	}

	res := resultCollector(wg, cfg, workerOutputChannels, abort)

	currentBlk := 0
	for range res {
		if currentBlk == 50 {
			abort.Signal()
			break
		}
		currentBlk++
	}
}

// makeTestTx creates dummy substates that will be processed without crashing.
func makeTestTx(count int) []*substate.Substate {
	testTxArr := make([]*substate.Substate, count)
	for i := 0; i < count; i++ {
		var testTx = &substate.Substate{
			Env: &substate.SubstateEnv{},
			Message: &substate.SubstateMessage{
				Gas:      10000,
				GasPrice: big.NewInt(0),
			},
			Result: &substate.SubstateResult{
				GasUsed: 1,
			},
		}
		testTxArr[i] = testTx
	}
	return testTxArr
}
