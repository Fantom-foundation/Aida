package replay

import (
	"fmt"
	"math/big"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/Fantom-foundation/go-opera/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

var GenDeletedAccountsCommand = cli.Command{
	Action:    genDeletedAccountsAction,
	Name:      "gen-deleted-accounts",
	Usage:     "executes full state transitions and record suicided accounts",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&substate.WorkersFlag,
		&substate.SubstateFlag,
		&utils.ChainIDFlag,
		&utils.DeletionDbFlag,
		&utils.LogLevelFlag,
	},
	Description: `
The substate-cli replay command requires two arguments:
<blockNumFirst> <blockNumLast>
<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay transactions.`,
}

const channelSize = 1000 //size of deletion channel

var DeleteHistory map[common.Address]bool //address recently and deleted

// readAccounts reads contracts which were suicided or created and adds them to lists
func readAccounts(ch chan ContractLiveness) ([]common.Address, []common.Address) {
	des := make(map[common.Address]bool)
	res := make(map[common.Address]bool)
	for contract := range ch {
		addr := contract.Addr
		if contract.IsDeleted {
			// if a contract was resurrected before suicided in the same tx,
			// only keep the last action.
			if _, found := res[addr]; found {
				delete(res, addr)
			}
			DeleteHistory[addr] = true // meta list
			des[addr] = true
		} else {
			// if a contract was suicided before resurrected in the same tx,
			// only keep the last action.
			if _, found := des[addr]; found {
				delete(des, addr)
			}
			// an account is considered as resurrected if it was recently deleted.
			if recentlyDeleted, found := DeleteHistory[addr]; found && recentlyDeleted {
				DeleteHistory[addr] = false
				res[addr] = true
			} else if found && !recentlyDeleted {
			}
		}
	}

	var deletedAccounts []common.Address
	var resurrectedAccounts []common.Address

	for addr := range des {
		deletedAccounts = append(deletedAccounts, addr)
	}
	for addr := range res {
		resurrectedAccounts = append(resurrectedAccounts, addr)
	}
	return deletedAccounts, resurrectedAccounts
}

// genDeletedAccountsTask process a transaction substate then records self-destructed accounts
// and resurrected accounts to a database.
func genDeletedAccountsTask(block uint64, tx int, recording *substate.Substate, ddb *substate.DestroyedAccountDB, log *logging.Logger) error {

	inputAlloc := recording.InputAlloc
	inputEnv := recording.Env
	inputMessage := recording.Message

	outputAlloc := recording.OutputAlloc
	outputResult := recording.Result

	var (
		vmConfig    vm.Config
		chainConfig *params.ChainConfig
	)

	vmConfig = opera.DefaultVMConfig
	vmConfig.NoBaseFee = true

	chainConfig = params.AllEthashProtocolChanges
	chainConfig.ChainID = big.NewInt(int64(chainID))
	switch chainID {
	case 250:
		chainConfig.LondonBlock = new(big.Int).SetUint64(37534833)
		chainConfig.BerlinBlock = new(big.Int).SetUint64(37455223)
	case 4002:
		chainConfig.LondonBlock = new(big.Int).SetUint64(7513335)
		chainConfig.BerlinBlock = new(big.Int).SetUint64(1559470)
	}

	var hashError error
	getHash := func(num uint64) common.Hash {
		if inputEnv.BlockHashes == nil {
			hashError = fmt.Errorf("getHash(%d) invoked, no blockhashes provided", num)
			return common.Hash{}
		}
		h, ok := inputEnv.BlockHashes[num]
		if !ok {
			hashError = fmt.Errorf("getHash(%d) invoked, blockhash for that block not provided", num)
		}
		return h
	}

	ch := make(chan ContractLiveness, channelSize)
	var statedb state.StateDB
	statedb = state.MakeGethInMemoryStateDB(&inputAlloc, block)
	//wrapper
	statedb = NewProxyDeletion(statedb, ch)

	// Apply Message
	var (
		gaspool   = new(evmcore.GasPool)
		blockHash = common.Hash{0x01}
		txHash    = common.Hash{0x02}
		txIndex   = tx
	)

	gaspool.AddGas(inputEnv.GasLimit)
	blockCtx := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    inputEnv.Coinbase,
		BlockNumber: new(big.Int).SetUint64(inputEnv.Number),
		Time:        new(big.Int).SetUint64(inputEnv.Timestamp),
		Difficulty:  inputEnv.Difficulty,
		GasLimit:    inputEnv.GasLimit,
		GetHash:     getHash,
	}
	// If currentBaseFee is defined, add it to the vmContext.
	if inputEnv.BaseFee != nil {
		blockCtx.BaseFee = new(big.Int).Set(inputEnv.BaseFee)
	}

	msg := inputMessage.AsMessage()

	vmConfig.Tracer = nil
	vmConfig.Debug = false
	statedb.Prepare(txHash, txIndex)

	txCtx := evmcore.NewEVMTxContext(msg)

	evm := vm.NewEVM(blockCtx, txCtx, statedb, chainConfig, vmConfig)

	snapshot := statedb.Snapshot()
	msgResult, err := evmcore.ApplyMessage(evm, msg, gaspool)

	if err != nil {
		statedb.RevertToSnapshot(snapshot)
		return err
	}

	if hashError != nil {
		return hashError
	}

	if chainConfig.IsByzantium(blockCtx.BlockNumber) {
		statedb.Finalise(true)
	} else {
		statedb.IntermediateRoot(chainConfig.IsEIP158(blockCtx.BlockNumber))
	}

	evmResult := &substate.SubstateResult{}
	if msgResult.Failed() {
		evmResult.Status = types.ReceiptStatusFailed
	} else {
		evmResult.Status = types.ReceiptStatusSuccessful
	}
	evmResult.Logs = statedb.GetLogs(txHash, blockHash)
	evmResult.Bloom = types.BytesToBloom(types.LogsBloom(evmResult.Logs))
	if to := msg.To(); to == nil {
		evmResult.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, msg.Nonce())
	}
	evmResult.GasUsed = msgResult.UsedGas

	evmAlloc := statedb.GetSubstatePostAlloc()

	r := outputResult.Equal(evmResult)
	a := outputAlloc.Equal(evmAlloc)
	if !(r && a) {
		if !r {
			log.Criticalf("inconsistent output: result")
			utils.PrintResultDiffSummary(outputResult, evmResult)
		}
		if !a {
			log.Criticalf("inconsistent output: alloc")
			utils.PrintAllocationDiffSummary(&outputAlloc, &evmAlloc)
		}
		return fmt.Errorf("inconsistent output")
	}

	close(ch)
	des, res := readAccounts(ch)
	if len(des)+len(res) > 0 {
		// if transaction completed successfully, put destroyed accounts
		// and resurrected accounts to a database
		if !msgResult.Failed() {
			ddb.SetDestroyedAccounts(block, tx, des, res)
		}
	}

	return nil
}

// genDeletedAccountsAction prepares config and arguments before GenDeletedAccountsAction
func genDeletedAccountsAction(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	return GenDeletedAccountsAction(cfg)
}

// GenDeletedAccountsAction replays transactions and record self-destructed accounts and resurrected accounts.
func GenDeletedAccountsAction(cfg *utils.Config) error {
	var err error

	log := utils.NewLogger(cfg.LogLevel, "Substate Replay")

	chainID = cfg.ChainID
	log.Infof("chain-id: %v", chainID)

	substate.SetSubstateDirectory(cfg.SubstateDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	ddb := substate.OpenDestroyedAccountDB(cfg.DeletionDb)
	defer ddb.Close()

	start := time.Now()
	sec := time.Since(start).Seconds()
	lastSec := time.Since(start).Seconds()
	txCount := uint64(0)
	lastTxCount := uint64(0)
	DeleteHistory = make(map[common.Address]bool)

	iter := substate.NewSubstateIterator(cfg.First, cfg.Workers)
	defer iter.Release()

	for iter.Next() {
		tx := iter.Value()
		if tx.Block > cfg.Last {
			break
		}

		err := genDeletedAccountsTask(tx.Block, tx.Transaction, tx.Substate, ddb, log)
		if err != nil {
			return err
		}
		txCount++
		sec = time.Since(start).Seconds()
		diff := sec - lastSec
		if diff >= 30 {
			numTx := txCount - lastTxCount
			lastTxCount = txCount
			log.Infof("substate-cli: gen-del-acc: Elapsed time: %.0f s, at block %v (~%.1f Tx/s)", sec, tx.Block, float64(numTx)/diff)
			lastSec = sec
		}
	}

	return err
}
