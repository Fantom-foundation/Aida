package utildb

import (
	"fmt"
	"math/big"
	"sync"
	"testing"

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
	stopChan := make(chan struct{})
	errChan := make(chan error, 2)
	encounteredErrors := errorHandler(stopChan, errChan)
	errChan <- fmt.Errorf("error1")
	errChan <- fmt.Errorf("error2")
	close(errChan)
	got := <-encounteredErrors
	assert.Equal(t, "error1\nerror2", got.Error())
}

func Test_errorHandlerResultGetsClosed(t *testing.T) {
	stopChan := make(chan struct{})
	errChan := make(chan error, 2)
	encounteredErrors := errorHandler(stopChan, errChan)
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

func Test_launchWorkersParallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	processor := executor.NewMockTxProcessor(ctrl)

	wg := &sync.WaitGroup{}
	cfg := &utils.Config{Workers: 3, ChainID: 250}

	stopChan := make(chan struct{})
	errChan := make(chan error)

	testTx := makeTestTx(6)

	// channel for each worker to get tasks for processing
	workerInputChannels := make(map[int]chan *substate.Transaction)
	for i := 0; i < cfg.Workers; i++ {
		workerInputChannels[i] = make(chan *substate.Transaction)
		go func(workerId int) {
			// 2 transactions for each worker
			workerInputChannels[workerId] <- &substate.Transaction{Block: uint64(workerId), Substate: testTx[workerId]}
			workerInputChannels[workerId] <- &substate.Transaction{Block: uint64(workerId + cfg.Workers), Substate: testTx[workerId+cfg.Workers]}
			close(workerInputChannels[workerId])
		}(i)
	}

	processor.EXPECT().ProcessTransaction(gomock.Any(), 0, gomock.Any(), substatecontext.NewTxContext(testTx[0])).Return(nil, nil)
	processor.EXPECT().ProcessTransaction(gomock.Any(), 1, gomock.Any(), substatecontext.NewTxContext(testTx[1])).Return(nil, nil)
	processor.EXPECT().ProcessTransaction(gomock.Any(), 2, gomock.Any(), substatecontext.NewTxContext(testTx[2])).Return(nil, nil)
	processor.EXPECT().ProcessTransaction(gomock.Any(), 3, gomock.Any(), substatecontext.NewTxContext(testTx[3])).Return(nil, nil)
	processor.EXPECT().ProcessTransaction(gomock.Any(), 4, gomock.Any(), substatecontext.NewTxContext(testTx[4])).Return(nil, nil)
	processor.EXPECT().ProcessTransaction(gomock.Any(), 5, gomock.Any(), substatecontext.NewTxContext(testTx[5])).Return(nil, nil)

	outPut := launchWorkers(wg, cfg, workerInputChannels, processor, stopChan, errChan)

	var orderCheck uint64 = 0
	// 2 transactions for each worker
	for k := 0; k < 2; k++ {
		for i := 0; i < cfg.Workers; i++ {
			r, ok := <-outPut[i]
			if !ok {
				t.Fatalf("results channel should be open")
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
		deleteHistory[common.HexToAddress("0x1")] = true

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
}

// TODO trace why this test fails
//func Test_resultCollector(t *testing.T) {
//
//	wg := &sync.WaitGroup{}
//	cfg := &utils.Config{Workers: 100, ChainID: 250}
//	stopChan := make(chan struct{})
//
//	// channel for each worker to get tasks for processing
//	workerOutputChannels := make(map[int]chan txLivelinessResult)
//	for i := 0; i < cfg.Workers; i++ {
//		workerOutputChannels[i] = make(chan txLivelinessResult)
//		go func(workerId int) {
//			// make proxy.ContractLiveliness map
//			cll := make([]proxy.ContractLiveliness, 0)
//			cll = append(cll, proxy.ContractLiveliness{Addr: common.HexToAddress(fmt.Sprintf("0x%x", workerId)), IsDeleted: true})
//			workerOutputChannels[workerId] <- txLivelinessResult{liveliness: cll}
//			close(workerOutputChannels[workerId])
//		}(i)
//	}
//
//	res := resultCollector(wg, cfg, workerOutputChannels, stopChan)
//
//	currentBlk := 0
//	for r := range res {
//		if !r.liveliness[0].IsDeleted {
//			t.Fatalf("results are in incorrect order")
//		}
//
//		if r.liveliness[0].Addr != common.HexToAddress(fmt.Sprintf("0x%x", currentBlk)) {
//			t.Fatalf("results are in incorrect order liveliness")
//		}
//		currentBlk++
//	}
//}
