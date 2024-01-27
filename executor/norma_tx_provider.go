package executor

import (
	"context"
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Norma/driver/rpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

type normaTxProvider struct {
	rpc NormaRpcClient
}

func NewNormaTxProvider(inTxs chan txcontext.TxContext) Provider[txcontext.TxContext] {
	return normaTxProvider{
		inTxs: inTxs,
	}
}

func (r normaTxProvider) Run(from int, to int, consumer Consumer[txcontext.TxContext]) error {

	//if err := consumer(TransactionInfo[*rpc.RequestAndResults]{blockNumber, 0, req}); err != nil {
	//return err
	//}

	return nil
}

func (r normaTxProvider) Close() {

}

func newGenerateData(tx *types.Transaction) txcontext.TxContext {
	return &generateData{tx: tx}
}

type generateData struct {
	txcontext.NilTxContext
	tx *types.Transaction
}

func (g generateData) GetOutputState() txcontext.WorldState {
	//TODO implement me
	panic("implement me")
}

func (g generateData) GetBlockEnvironment() txcontext.BlockEnvironment {
	//TODO implement me
	panic("implement me")
}

func (g generateData) GetMessage() core.Message {
	//TODO implement me
	panic("implement me")
}

// NormaRpcClient is an interface that abstracts the RPC client used by the norma
// transactions generator.
type NormaRpcClient interface {
	rpc.RpcClient
}

// fakeRpcClient is a fake RPC client that generates fake data. It is used to provide
// data for norma transactions generator.
type fakeRpcClient struct {
	// outTxs is a channel to which the RPC client will send transactions.
	outTxs chan<- txcontext.TxContext
}

// newFakeRpcClient creates a new fakeRpcClient.
func newFakeRpcClient() fakeRpcClient {
	return fakeRpcClient{
		outTxs: make(chan<- txcontext.TxContext, 1000),
	}
}

// SendTransaction injects the transaction into the pending pool for execution.
func (f fakeRpcClient) SendTransaction(_ context.Context, tx *types.Transaction) error {
	fmt.Print("SendTransaction\n")

	f.outTxs <- newGenerateData(tx)

	return nil
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
	return &types.Header{
		Number: big.NewInt(1),
	}, nil
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

// FilterLogs executes a log filter operation, blocking during execution and
// returning all the results in one batch.
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
