package executor

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Norma/driver/rpc"
	"github.com/Fantom-foundation/Norma/load/app"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

const testTreasureAccountPrivateKey = "1234567890123456789012345678901234567890123456789012345678901234"

// normaTxProvider is a Provider that generates transactions using the norma
// transactions generator.
type normaTxProvider struct {
	cfg     *utils.Config
	stateDb state.StateDB
}

// NewNormaTxProvider creates a new norma tx provider.
func NewNormaTxProvider(cfg *utils.Config, stateDb state.StateDB) Provider[txcontext.TxContext] {
	return normaTxProvider{
		cfg:     cfg,
		stateDb: stateDb,
	}
}

// Run runs the norma tx provider.
func (r normaTxProvider) Run(from int, to int, consumer Consumer[txcontext.TxContext]) error {
	wg := sync.WaitGroup{}
	fakeRpc := newFakeRpcClient(r.stateDb)
	txChan := fakeRpc.OutTxs()

	currentBlock := from + 1
	currentTx := 0

	wg.Add(1)
	// listen for transactions emitted by the RPC client and apply them to the
	// consumer, this goroutine will finish when the contract is deployed and
	// accounts are funded
	go func() {
		defer wg.Done()
		for tx := range txChan {
			data := newNormaTx(tx)
			if err := consumer(TransactionInfo[txcontext.TxContext]{Block: currentBlock, Transaction: currentTx, Data: data}); err != nil {
				fmt.Printf("failed to consume transaction; %v\n", err)
			}
			//currentTx++
			currentBlock++
		}
	}()

	privateKey, err := crypto.HexToECDSA(testTreasureAccountPrivateKey)
	if err != nil {
		return err
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("failed to cast public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	amount := big.NewInt(params.Ether)
	amount = amount.Mul(amount, big.NewInt(100000))

	r.stateDb.BeginBlock(0)
	r.stateDb.BeginTransaction(0)
	r.stateDb.CreateAccount(fromAddress)
	r.stateDb.AddBalance(fromAddress, amount)
	r.stateDb.EndTransaction()
	r.stateDb.EndBlock()

	// initialize the norma application
	primaryAccount, err := app.NewAccount(0, testTreasureAccountPrivateKey, int64(r.cfg.ChainID))
	if err != nil {
		return err
	}

	app, err := app.NewCounterApplication(fakeRpc, primaryAccount, 0, 0, 0)
	if err != nil {
		return err
	}

	user, err := app.CreateUser(fakeRpc)
	if err != nil {
		return err
	}

	if err = app.WaitUntilApplicationIsDeployed(fakeRpc); err != nil {
		return err
	}

	// from now on, we don't need the rpc anymore, it was needed just for the
	// deployment of the contract, so we can close it so that the goroutine
	// listening for transactions emitted by the RPC client can finish
	wg.Wait()
	fakeRpc.Close()

	for i := 0; i < 10; i++ {
		tx, err := user.GenerateTx()
		if err != nil {
			return err
		}
		data := newNormaTx(tx)
		if err := consumer(TransactionInfo[txcontext.TxContext]{Block: 1, Transaction: 1, Data: data}); err != nil {
			fmt.Printf("failed to consume transaction; %v\n", err)
		}
	}

	return nil
}

func (r normaTxProvider) Close() {
	// nothing to do
}

func newNormaTx(tx *types.Transaction) txcontext.TxContext {
	return &normaTx{tx: tx}
}

type normaTx struct {
	txcontext.NilTxContext
	tx *types.Transaction
}

//func (g normaTx) GetOutputState() txcontext.WorldState {
//	//TODO implement me
//	panic("implement me")
//}

func (ntx normaTx) GetBlockEnvironment() txcontext.BlockEnvironment {
	return normaTxBlockEnv{tx: ntx.tx}
}

// GetCoinbase returns the coinbase address.
func (e normaTxBlockEnv) GetCoinbase() common.Address {
	return common.HexToAddress("0x1")
}

// GetDifficulty returns the current difficulty level.
func (e normaTxBlockEnv) GetDifficulty() *big.Int {
	return big.NewInt(1)
}

// GetGasLimit returns the maximum amount of gas that can be used in a block.
func (e normaTxBlockEnv) GetGasLimit() uint64 {
	return 1_000_000_000_000
}

// GetNumber returns the current block number.
func (e normaTxBlockEnv) GetNumber() uint64 {
	// not used
	return 0
}

// GetTimestamp returns the timestamp of the current block.
func (e normaTxBlockEnv) GetTimestamp() uint64 {
	// use current timestamp as the block timestamp
	// since we don't have a real block
	return uint64(time.Now().Unix())
}

// GetBlockHash returns the hash of the block with the given number.
func (e normaTxBlockEnv) GetBlockHash(blockNumber uint64) common.Hash {
	// transform the block number into a hash
	// we don't have real block hashes, so we just use the block number
	return common.BigToHash(big.NewInt(int64(blockNumber)))
}

// GetBaseFee returns the base fee for transactions in the current block.
func (e normaTxBlockEnv) GetBaseFee() *big.Int {
	return big.NewInt(0)
}

type normaTxBlockEnv struct {
	tx *types.Transaction
}

func (ntx normaTx) GetMessage() core.Message {

	if ntx.tx.To() != nil {
		fmt.Printf("receiver: %s\n", ntx.tx.To().String())
		fmt.Printf("Value %d\n", ntx.tx.Value().Uint64())
	}

	// extract sender from tx by passing it through the signer
	// we expect that the tx is signed
	sender, _ := types.Sender(types.NewEIP155Signer(ntx.tx.ChainId()), ntx.tx)
	fmt.Printf("sender: %s\n", sender.String())
	return types.NewMessage(
		sender,
		ntx.tx.To(),
		ntx.tx.Nonce(),
		ntx.tx.Value(),
		ntx.tx.Gas(),
		ntx.tx.GasPrice(),
		ntx.tx.GasFeeCap(),
		ntx.tx.GasTipCap(),
		ntx.tx.Data(),
		ntx.tx.AccessList(),
		false,
	)
}

// NormaRpcClient is an interface that abstracts the RPC client used by the norma
// transactions generator.
type NormaRpcClient interface {
	rpc.RpcClient
	// OutTxs returns a channel to which the RPC client will send transactions.
	OutTxs() <-chan *types.Transaction
}

// fakeRpcClient is a fake RPC client that generates fake data. It is used to provide
// data for norma transactions generator.
type fakeRpcClient struct {
	// outTxs is a channel to which the RPC client will send transactions.
	outTxs chan *types.Transaction
	// stateDb is a state database.
	stateDb state.StateDB
}

// newFakeRpcClient creates a new fakeRpcClient.
func newFakeRpcClient(stateDb state.StateDB) fakeRpcClient {
	return fakeRpcClient{
		outTxs:  make(chan *types.Transaction, 1000),
		stateDb: stateDb,
	}
}

func (f fakeRpcClient) OutTxs() <-chan *types.Transaction {
	return f.outTxs
}

// SendTransaction injects the transaction into the pending pool for execution.
func (f fakeRpcClient) SendTransaction(_ context.Context, tx *types.Transaction) error {
	fmt.Print("SendTransaction\n")
	// create account if it doesn't exist
	// this is necessary because the receiver of the transaction must exist
	// in the state database, otherwise the receiver won't be able to receive
	if tx.To() != nil && !f.stateDb.Exist(*tx.To()) {
		f.stateDb.CreateAccount(*tx.To())
	}
	f.outTxs <- tx
	return nil
}

func (f fakeRpcClient) Call(result interface{}, method string, args ...interface{}) error {
	// todo implement me
	fmt.Print("Call\n")
	return nil
}

func (f fakeRpcClient) NonceAt(_ context.Context, account common.Address, _ *big.Int) (uint64, error) {
	nonce := f.stateDb.GetNonce(account)
	fmt.Printf("NonceAt: %d, Addr: %s\n", nonce, account.String())
	return nonce, nil
}

func (f fakeRpcClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	balance := f.stateDb.GetBalance(account)
	fmt.Printf("BalanceAt: %d, Addr: %s\n", balance, account.String())
	return balance, nil
}

func (f fakeRpcClient) Close() {
	close(f.outTxs)
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
	// this method is called to obtain GasFeeCap, which was introduced in EIP-1559
	// since this is ethereum thing, we can just return an empty header
	return &types.Header{}, nil
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
	// use lower gas price, so we don't run out of gas
	// too quickly since estimation is overestimating
	return big.NewInt(1), nil
}

// SuggestGasTipCap retrieves the currently suggested 1559 priority fee to allow
// a timely execution of a transaction.
func (f fakeRpcClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	// not used
	return big.NewInt(0), nil
}

// EstimateGas tries to estimate the gas needed to execute a specific
// transaction based on the current pending state of the backend blockchain.
// There is no guarantee that this is the true gas limit requirement as other
// transactions may be added or removed by miners, but it should provide a basis
// for setting a reasonable default.
func (f fakeRpcClient) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	// use more gas than should be needed
	// TODO: use the vm for gas estimation
	return 500_000, nil
}

// FilterLogs executes a log filter operation, blocking during execution and
// returning all the results in one batch.
func (f fakeRpcClient) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	// not used
	return nil, nil
}

// SubscribeFilterLogs creates a background log filtering operation, returning
// a subscription immediately, which can be used to stream the found events.
func (f fakeRpcClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	// not used
	return nil, nil
}
