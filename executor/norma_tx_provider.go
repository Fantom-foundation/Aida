// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package executor

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/txcontext/txgenerator"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Norma/load/app"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// treasureAccountPrivateKey is the private key of the treasure account.
const treasureAccountPrivateKey = "1234567890123456789012345678901234567890123456789012345678901234"

// normaConsumer is a consumer of norma transactions.
type normaConsumer func(*types.Transaction, *common.Address) error

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
func (p normaTxProvider) Run(from int, to int, consumer Consumer[txcontext.TxContext]) error {
	// initialize the treasure account
	primaryAccount, err := p.initializeTreasureAccount(from)
	if err != nil {
		return err
	}

	// define the current block and transaction numbers,
	// we start from the next block after the `from` block
	// because on the `from` block we initialized and funded
	// the treasure account
	currentBlock := from + 1
	nextTxNumber := 0

	// define norma consumer that will be used to consume transactions
	// this is the only place that is responsible for incrementing block and tx numbers
	nc := func(tx *types.Transaction, sender *common.Address) error {
		data, err := txgenerator.NewNormaTxContext(tx, uint64(currentBlock), sender)
		if err != nil {
			return err
		}
		err = consumer(TransactionInfo[txcontext.TxContext]{Block: currentBlock, Transaction: nextTxNumber, Data: data})
		if err != nil {
			return err
		}
		// increment the transaction number for next transaction
		// if we reached the maximum number of transactions per block, increment the block number
		nextTxNumber++
		// greater or equal, because transactions are indexed from 0
		if uint64(nextTxNumber) >= p.cfg.BlockLength {
			currentBlock++
			nextTxNumber = 0
		}
		return nil
	}

	fakeRpc := newFakeRpcClient(p.stateDb, nc)
	defer fakeRpc.Close()

	// initialize the list of app types
	appTypes := p.cfg.TxGeneratorType
	if len(appTypes) == 1 && appTypes[0] == "all" {
		appTypes = []string{"erc20", "counter", "store", "uniswap"}
	}

	// create users for each app type
	users := make([]app.User, 0)
	for ix, appType := range appTypes {
		application, err := app.NewApplication(appType, fakeRpc, primaryAccount, 1, uint32(ix), uint32(ix))
		if err != nil {
			return err
		}
		user, err := application.CreateUser(fakeRpc)
		if err != nil {
			return err
		}
		if err = application.WaitUntilApplicationIsDeployed(fakeRpc); err != nil {
			return err
		}
		users = append(users, user)
	}

	// generate transactions until the `to` block is reached
	// `currentBlock` is incremented in the `nc` function
	shouldBreak := false
	for {
		for _, user := range users {
			if currentBlock > to {
				shouldBreak = true
				break
			}
			// generate tx
			tx, err := user.GenerateTx()
			if err != nil {
				return err
			}
			// apply tx to the consumer
			addr := user.SenderAddress()
			if err = nc(tx, &addr); err != nil {
				return err
			}
		}
		if shouldBreak {
			break
		}
	}

	return nil
}

func (p normaTxProvider) Close() {
	// nothing to do
}

// initializeTreasureAccount initializes the treasure account.
// The treasure account is an account with a lot of ether that is used to fund
// the accounts and deploy the contract.
func (p normaTxProvider) initializeTreasureAccount(blkNumber int) (*app.Account, error) {
	// extract the address from the treasure account private key
	privateKey, err := crypto.HexToECDSA(treasureAccountPrivateKey)
	if err != nil {
		return nil, err
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// fund the treasure account directly in the state database
	amount := uint256.NewInt(0).Mul(uint256.NewInt(params.Ether), uint256.NewInt(2_000_000_000))
	// we need to begin and end the block and transaction to be able to create an account
	// and add balance to it (otherwise the account would not be funded for geth storage implementation)
	err = p.stateDb.BeginBlock(uint64(blkNumber))
	if err != nil {
		return nil, fmt.Errorf("cannot begin block; %w", err)
	}
	err = p.stateDb.BeginTransaction(uint32(0))
	if err != nil {
		return nil, fmt.Errorf("cannot begin transaction; %w", err)
	}
	p.stateDb.CreateAccount(fromAddress)
	p.stateDb.AddBalance(fromAddress, amount, 0)
	err = p.stateDb.EndTransaction()
	if err != nil {
		return nil, fmt.Errorf("cannot end transaction; %w", err)
	}
	err = p.stateDb.EndBlock()
	if err != nil {
		return nil, fmt.Errorf("cannot end block; %w", err)
	}

	return app.NewAccount(0, treasureAccountPrivateKey, int64(p.cfg.ChainID))
}

// fakeRpcClient is a fake RPC client that generates fake data. It is used to provide
// data for norma transactions generator.
type fakeRpcClient struct {
	// stateDb is a state database.
	stateDb state.StateDB
	// consumer is a consumer of transactions.
	consumer normaConsumer
	// pendingCodes is a map of pending codes.
	pendingCodes map[common.Address][]byte
}

// newFakeRpcClient creates a new fakeRpcClient.
func newFakeRpcClient(stateDb state.StateDB, consumer normaConsumer) fakeRpcClient {
	return fakeRpcClient{
		stateDb:      stateDb,
		consumer:     consumer,
		pendingCodes: make(map[common.Address][]byte),
	}
}

// SendTransaction injects the transaction into the pending pool for execution.
func (f fakeRpcClient) SendTransaction(_ context.Context, tx *types.Transaction) error {
	// if the transaction is a contract deployment, we need to store the code
	// in the pending codes map
	if tx.To() == nil {
		// extract sender from tx
		sender, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
		if err != nil {
			return err
		}
		// get the expected contract address
		contractAddress := crypto.CreateAddress(sender, tx.Nonce())
		// store the code in the pending codes map
		f.pendingCodes[contractAddress] = tx.Data()
	}
	return f.consumer(tx, nil)
}

func (f fakeRpcClient) Call(_ interface{}, _ string, _ ...interface{}) error {
	// not used
	return nil
}

func (f fakeRpcClient) NonceAt(_ context.Context, account common.Address, _ *big.Int) (uint64, error) {
	err := f.stateDb.BeginTransaction(uint32(0))
	if err != nil {
		return 0, err
	}
	nonce := f.stateDb.GetNonce(account)
	err = f.stateDb.EndTransaction()
	if err != nil {
		return 0, err
	}
	return nonce, nil
}

func (f fakeRpcClient) BalanceAt(_ context.Context, account common.Address, _ *big.Int) (*big.Int, error) {
	err := f.stateDb.BeginTransaction(uint32(0))
	if err != nil {
		return nil, err
	}
	balance := f.stateDb.GetBalance(account)
	err = f.stateDb.EndTransaction()
	if err != nil {
		return nil, err
	}
	return balance.ToBig(), nil
}

func (f fakeRpcClient) Close() {
	// do nothing
}

func (f fakeRpcClient) CodeAt(_ context.Context, address common.Address, _ *big.Int) ([]byte, error) {
	err := f.stateDb.BeginTransaction(uint32(0))
	if err != nil {
		return nil, err
	}
	code := f.stateDb.GetCode(address)
	err = f.stateDb.EndTransaction()
	if err != nil {
		return nil, err
	}
	return code, nil
}

func (f fakeRpcClient) CallContract(_ context.Context, _ ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	// not used
	return nil, nil
}

// HeaderByNumber returns a block header from the current canonical chain. If
// number is nil, the latest known header is returned.
func (f fakeRpcClient) HeaderByNumber(_ context.Context, _ *big.Int) (*types.Header, error) {
	// this method is called to obtain GasFeeCap, which was introduced in EIP-1559
	// since this is an ethereum thing, we can just return an empty header
	return &types.Header{}, nil
}

// PendingCodeAt returns the code of the given account in the pending state.
func (f fakeRpcClient) PendingCodeAt(_ context.Context, address common.Address) ([]byte, error) {
	return f.pendingCodes[address], nil
}

// PendingNonceAt retrieves the current pending nonce associated with an account.
func (f fakeRpcClient) PendingNonceAt(_ context.Context, _ common.Address) (uint64, error) {
	// not used
	return 0, nil
}

// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
// execution of a transaction.
func (f fakeRpcClient) SuggestGasPrice(_ context.Context) (*big.Int, error) {
	// use lower gas price, so we don't run out of gas
	// too quickly since estimation is overestimating
	return big.NewInt(1), nil
}

// SuggestGasTipCap retrieves the currently suggested 1559 priority fee to allow
// a timely execution of a transaction.
func (f fakeRpcClient) SuggestGasTipCap(_ context.Context) (*big.Int, error) {
	// not used
	return big.NewInt(0), nil
}

// EstimateGas tries to estimate the gas needed to execute a specific
// transaction based on the current pending state of the backend blockchain.
// There is no guarantee that this is the true gas limit requirement as other
// transactions may be added or removed by miners, but it should provide a basis
// for setting a reasonable default.
func (f fakeRpcClient) EstimateGas(_ context.Context, _ ethereum.CallMsg) (gas uint64, err error) {
	// use more gas than should be needed
	// it is only used for contract deployment
	return 1_200_000, nil
}

// FilterLogs executes a log filter operation, blocking during execution and
// returning all the results in one batch.
func (f fakeRpcClient) FilterLogs(_ context.Context, _ ethereum.FilterQuery) ([]types.Log, error) {
	// not used
	return nil, nil
}

// SubscribeFilterLogs creates a background log filtering operation, returning
// a subscription immediately, which can be used to stream the found events.
func (f fakeRpcClient) SubscribeFilterLogs(_ context.Context, _ ethereum.FilterQuery, _ chan<- types.Log) (ethereum.Subscription, error) {
	// not used
	return nil, nil
}
