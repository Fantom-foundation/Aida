package utildb

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/state/proxy"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const channelSize = 100000 // size of deletion channel

// readAccounts reads contracts which were suicided or created and adds them to lists
func readAccounts(ch chan proxy.ContractLiveliness, deleteHistory *map[common.Address]bool) ([]common.Address, []common.Address) {
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
			(*deleteHistory)[addr] = true // meta list
			des[addr] = true
		} else {
			// if a contract was suicided before resurrected in the same tx,
			// only keep the last action.
			if _, found := des[addr]; found {
				delete(des, addr)
			}
			// an account is considered as resurrected if it was recently deleted.
			if recentlyDeleted, found := (*deleteHistory)[addr]; found && recentlyDeleted {
				(*deleteHistory)[addr] = false
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
func genDeletedAccountsTask(
	tx *substate.Transaction,
	processor *executor.TxProcessor,
	ddb *substate.DestroyedAccountDB,
	deleteHistory *map[common.Address]bool,
	cfg *utils.Config,
) error {
	ch := make(chan proxy.ContractLiveliness, channelSize)
	var statedb state.StateDB
	var err error
	ss := substatecontext.NewTxContext(tx.Substate)

	statedb, err = state.MakeOffTheChainStateDB(ss.GetInputState(), tx.Block, state.NewChainConduit(cfg.ChainID == utils.EthereumChainID, utils.GetChainConfig(cfg.ChainID)))
	if err != nil {
		return err
	}

	//wrapper
	statedb = proxy.NewDeletionProxy(statedb, ch, cfg.LogLevel)

	_, err = processor.ProcessTransaction(statedb, int(tx.Block), tx.Transaction, ss)
	if err != nil {
		return nil
	}

	close(ch)
	des, res := readAccounts(ch, deleteHistory)
	if len(des)+len(res) > 0 {
		// if transaction completed successfully, put destroyed accounts
		// and resurrected accounts to a database
		if tx.Substate.Result.Status == types.ReceiptStatusSuccessful {
			err = ddb.SetDestroyedAccounts(tx.Block, tx.Transaction, des, res)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// GenDeletedAccountsAction replays transactions and record self-destructed accounts and resurrected accounts.
func GenDeletedAccountsAction(cfg *utils.Config, ddb *substate.DestroyedAccountDB, firstBlock uint64, lastBlock uint64) error {
	var err error

	err = utils.StartCPUProfile(cfg)
	if err != nil {
		return err
	}

	log := logger.NewLogger(cfg.LogLevel, "Generate Deleted Accounts")

	log.Noticef("Generate deleted accounts from block %v to block %v", firstBlock, lastBlock)

	start := time.Now()
	sec := time.Since(start).Seconds()
	lastSec := time.Since(start).Seconds()
	txCount := uint64(0)
	lastTxCount := uint64(0)
	var deleteHistory = make(map[common.Address]bool)

	iter := substate.NewSubstateIterator(firstBlock, cfg.Workers)
	defer iter.Release()

	processor := executor.MakeTxProcessor(cfg)

	for iter.Next() {
		tx := iter.Value()
		if tx.Block > lastBlock {
			break
		}

		if tx.Transaction < utils.PseudoTx {
			err = genDeletedAccountsTask(tx, processor, ddb, &deleteHistory, cfg)
			if err != nil {
				return err
			}

			txCount++
			sec = time.Since(start).Seconds()
			diff := sec - lastSec
			if diff >= 30 {
				numTx := txCount - lastTxCount
				lastTxCount = txCount
				log.Infof("aida-vm: gen-del-acc: Elapsed time: %.0f s, at block %v (~%.1f Tx/s)", sec, tx.Block, float64(numTx)/diff)
				lastSec = sec
			}
		}
	}

	utils.StopCPUProfile(cfg)

	// explicitly set to nil to release memory as soon as possible
	deleteHistory = nil

	return err
}
