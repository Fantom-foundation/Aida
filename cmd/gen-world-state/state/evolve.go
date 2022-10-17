package state

import (
	"fmt"
	"github.com/Fantom-foundation/Aida-Testing/cmd/gen-world-state/flags"
	"github.com/Fantom-foundation/Aida-Testing/world-state/db/snapshot"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
	"time"
)

// CmdEvolveState evolves state of World State database to given target block by using substateDB data about accounts
var CmdEvolveState = cli.Command{
	Action:      evolveState,
	Name:        "evolve",
	Aliases:     []string{"e"},
	Usage:       "Evolves world state snapshot database into selected target block",
	Description: `The evolve evolves state of stored accounts in world state snapshot database.`,
	ArgsUsage:   "<target> <substatedir> <workers>",
	Flags: []cli.Flag{
		&flags.TargetBlock,
		&flags.SubstateDBPath,
		&flags.Workers,
	},
}

// evolveState dumps state from given EVM trie into an output account-state database
func evolveState(ctx *cli.Context) error {
	// try to open state DB
	stateDB, err := snapshot.OpenStateDB(ctx.Path(flags.StateDBPath.Name))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(stateDB)

	// evolution until given block
	targetBlock := ctx.Uint64(flags.TargetBlock.Name)

	// make logger
	log := Logger(ctx, "evolve")

	// call evolveState with prepared arguments
	err = EvolveState(stateDB, ctx.Path(flags.SubstateDBPath.Name), targetBlock, ctx.Int(flags.Workers.Name), log)

	log.Info("done")
	return err
}

// EvolveState evolves stateDB to target block
func EvolveState(stateDB *snapshot.StateDB, substateDBPath string, targetBlock uint64, workers int, log *logging.Logger) error {
	// retrieving block number from world state database
	currentBlock, err := stateDB.GetBlockNumber()
	if err != nil {
		return err
	}
	log.Infof("Database is currently at block %d", currentBlock)

	if currentBlock == targetBlock {
		log.Info("World state database is already at target block %d", targetBlock)
		return nil
	}

	if currentBlock > targetBlock {
		err = fmt.Errorf("target block %d can't be lower than current block in database", targetBlock)
		log.Error(err.Error())
		return err
	}

	// try to open sub state DB
	substate.SetSubstateDirectory(substateDBPath)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	// database has already current block completed therefore starting at following block
	startingBlock := currentBlock + 1

	// evolution of stateDB
	lastProcessedBlock, err := evolution(stateDB, startingBlock, targetBlock, workers, log)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	// if evolution to desired state didn't complete successfully
	if lastProcessedBlock != targetBlock {
		log.Infof("last processed block was %d, substateDB didn't contain data for other blocks till target %d", lastProcessedBlock, targetBlock)
	}

	// insert new block number into database
	err = stateDB.PutBlockNumber(lastProcessedBlock)
	if err != nil {
		log.Errorf("Unable to insert block number into db; %s", err.Error())
		return err
	}

	// log last processed block
	log.Infof("Database was successfully evolved to %d block", lastProcessedBlock)
	return nil
}

// evolution iterates trough Substates between first and target blocks
// anticipates that SubstateDB is already open
func evolution(stateDB *snapshot.StateDB, firstBlock uint64, targetBlock uint64, workers int, log *logging.Logger) (uint64, error) {
	log.Info("starting evolution block number", firstBlock, "target block", targetBlock)

	// contains last block id
	var lastProcessedBlock uint64 = 0

	// iterator starting from first block - current block of stateDB
	iter := substate.NewSubstateIterator(firstBlock, workers)
	defer iter.Release()

	// timer for printing progress
	tick := time.NewTicker(20 * time.Second)
	defer func() {
		tick.Stop()
	}()

	// iteration trough substates
	for iter.Next() {
		tx := iter.Value()
		if tx.Block > targetBlock {
			break
		}

		// print progress
		select {
		case <-tick.C:
			log.Infof("evolving %d/%d", tx.Block, targetBlock)
		default:
		}

		// evolution of database by single Substate Output values
		err := evolveSubstate(&tx.Substate.OutputAlloc, stateDB, log)
		if err != nil {
			return 0, err
		}
		lastProcessedBlock = tx.Block
	}

	return lastProcessedBlock, nil
}

// evolveSubstate evolves world state db supplied substate.substateOut containing data of accounts at the end of one transaction
func evolveSubstate(substateOut *substate.SubstateAlloc, stateDB *snapshot.StateDB, log *logging.Logger) error {
	for address, substateAccount := range *substateOut {
		// get account stored in state snapshot database
		acc, err := stateDB.Account(address)
		if err != nil {
			// account was not found in database therefore we need to create new instance
			addrHash := crypto.Keccak256Hash(address.Bytes())
			acc = &types.Account{Hash: addrHash}

			if len(substateAccount.Storage) > 0 {
				acc.Storage = make(map[common.Hash]common.Hash, len(substateAccount.Storage))
			}
		}

		// updating account data
		acc.Code = substateAccount.Code
		acc.Nonce = substateAccount.Nonce
		acc.Balance = substateAccount.Balance

		// overwriting all changed values in storage
		for keyRaw, value := range substateAccount.Storage {
			// generation of key
			// keyRaw consists of unhashed ordered keys
			// eg. keyRaw=0x0000000000000000000000000000000000000000000000000000000000000001 (substate record key)
			// 	   key=0xb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6 (snapshot record key)
			key := common.BytesToHash(crypto.Keccak256(keyRaw.Bytes()))
			if value == snapshot.ZeroHash {
				if _, found := acc.Storage[key]; found {
					// removing key with empty value from storage
					delete(acc.Storage, key)
				}
				continue
			}
			// storing new value or updating old value
			acc.Storage[key] = value
		}

		// inserting updated account into database
		err = stateDB.PutAccount(acc)
		if err != nil {
			log.Errorf("Unable to insert account %s in database; %s", address.String(), err.Error())
			break
		}

	}
	return nil
}
