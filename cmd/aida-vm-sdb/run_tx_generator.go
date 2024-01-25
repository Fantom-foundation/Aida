package main

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"os"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Norma/load/app"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/urfave/cli/v2"
)

const testTreasureAccountPrivateKey = "1234567890123456789012345678901234567890123456789012345678901234"

type fakeRpcClient struct {
}

func (f fakeRpcClient) Call(result interface{}, method string, args ...interface{}) error {
	// todo implement me
	fmt.Print("Call\n")
	return nil
}

func (f fakeRpcClient) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	// todo implement me
	fmt.Print("NonceAt\n")
	return 0, nil
}

func (f fakeRpcClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	// todo implement me
	fmt.Print("BalanceAt\n")
	return big.NewInt(0), nil
}

func (f fakeRpcClient) Close() {
	fmt.Print("Close\n")
	// todo implement me
}

func (f fakeRpcClient) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	// todo implement me
	fmt.Print("CodeAt\n")
	return nil, nil
}

func (f fakeRpcClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	// todo implement me
	fmt.Print("CallContract\n")
	return nil, nil
}

// HeaderByNumber returns a block header from the current canonical chain. If
// number is nil, the latest known header is returned.
func (f fakeRpcClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	// todo implement me
	fmt.Print("HeaderByNumber\n")
	return nil, nil
}

// PendingCodeAt returns the code of the given account in the pending state.
func (f fakeRpcClient) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	// todo implement me
	fmt.Print("PendingCodeAt\n")
	return nil, nil
}

// PendingNonceAt retrieves the current pending nonce associated with an account.
func (f fakeRpcClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	// todo implement me
	fmt.Print("PendingNonceAt\n")
	return 1, nil
}

// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
// execution of a transaction.
func (f fakeRpcClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	// todo implement me
	fmt.Print("SuggestGasPrice\n")
	return big.NewInt(27_000), nil
}

// SuggestGasTipCap retrieves the currently suggested 1559 priority fee to allow
// a timely execution of a transaction.
func (f fakeRpcClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	fmt.Print("SuggestGasTipCap\n")
	return big.NewInt(0), nil
}

// EstimateGas tries to estimate the gas needed to execute a specific
// transaction based on the current pending state of the backend blockchain.
// There is no guarantee that this is the true gas limit requirement as other
// transactions may be added or removed by miners, but it should provide a basis
// for setting a reasonable default.
func (f fakeRpcClient) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	// todo implement me
	fmt.Print("EstimateGas\n")
	return 27_000, nil
}

// SendTransaction injects the transaction into the pending pool for execution.
func (f fakeRpcClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	fmt.Print("SendTransaction\n")
	return nil
}

// FilterLogs executes a log filter operation, blocking during execution and
// returning all the results in one batch.
//
// TODO(karalabe): Deprecate when the subscription one can return past data too.
func (f fakeRpcClient) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	// todo implement me
	fmt.Print("FilterLogs\n")
	return nil, nil
}

// SubscribeFilterLogs creates a background log filtering operation, returning
// a subscription immediately, which can be used to stream the found events.
func (f fakeRpcClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	// todo implement me
	fmt.Print("SubscribeFilterLogs\n")
	return nil, nil
}

// RunTxGenerator performs sequential block processing on a StateDb using transaction generator
func RunTxGenerator(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	cfg.StateValidationMode = utils.SubsetCheck

	primaryAccount, err := app.NewAccount(0, testTreasureAccountPrivateKey, int64(cfg.ChainID))
	if err != nil {
		return err
	}

	_ = executor.MakeLiveDbProcessor(cfg)

	rpc := fakeRpcClient{}

	_, _ = app.NewCounterApplication(rpc, primaryAccount, 0, 0, 0)

	fmt.Print("RunTxGenerator")
	os.Exit(1)

	// todo init the provider (the generator) here and pass it to runTransactions

	return runTransactions(cfg, nil, nil, false)
}
func newGenerateData() txcontext.Transaction {
	return &generateData{}
}

type generateData struct {
}

func (g generateData) GetBlockEnvironment() txcontext.BlockEnvironment {
	//TODO implement me
	panic("implement me")
}

func (g generateData) GetMessage() types.Message {
	//TODO implement me
	panic("implement me")
}

type txProcessor struct {
	cfg *utils.Config
}

func (p txProcessor) Process(state executor.State[txcontext.Transaction], ctx *executor.Context) error {
	// todo apply data onto StateDb
	return nil
}

func runTransactions(
	cfg *utils.Config,
	provider executor.Provider[txcontext.Transaction],
	stateDb state.StateDB,
	disableStateDbExtension bool,
) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension[txcontext.Transaction]{
		// todo choose extensions
	}

	return executor.NewExecutor(provider, cfg.LogLevel).Run(
		executor.Params{
			From:  0,
			To:    math.MaxInt,
			State: stateDb,
		},
		txProcessor{cfg},
		extensionList,
	)
}
