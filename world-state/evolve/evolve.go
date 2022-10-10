package evolve

import (
	"github.com/Fantom-foundation/Aida-Testing/world-state/db"
	"github.com/Fantom-foundation/Aida-Testing/world-state/logger"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

const (
	flagUntilBlock       = "untilblock"
	flagSubstateDBPath   = "substate-db"
	flagWorldStateDBPath = "db"
	flagWorkers          = "workers"
)

// CmdEvolveState evolves state of
var CmdEvolveState = cli.Command{
	Action:      evolveState,
	Name:        "evolve",
	Usage:       "",
	Description: ``,
	ArgsUsage:   "",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:     flagUntilBlock,
			Usage:    "Evolve database only until given block is reached",
			Required: true,
		},
		&cli.PathFlag{
			Name:     flagSubstateDBPath,
			Usage:    "Input SubState database path",
			Required: true,
		},
		&cli.IntFlag{
			Name:  flagWorkers,
			Usage: "Number of account processing threads",
			Value: 5,
		},
	},
}

// evolveState dumps state from given EVM trie into an output account-state database
func evolveState(ctx *cli.Context) error {
	// try to open state DB
	stateDB, err := db.OpenStateSnapshotDB(ctx.Path(flagWorldStateDBPath))
	if err != nil {
		return err
	}
	defer db.MustCloseSnapshotDB(stateDB)

	// try to open sub state DB
	subDB, err := db.OpenSubstateDB(ctx.Path(flagSubstateDBPath))
	if err != nil {
		return err
	}
	defer db.MustCloseSubstateDB(subDB)
	substateDB := substate.NewSubstateDB(subDB.Backend)

	// evolution until given block
	targetBlock := ctx.Uint64(flagUntilBlock)

	// make logger
	log := logger.New(ctx.App.Writer, "info")

	blockNumber, err := stateDB.GetBlockNumber()
	if err != nil {
		return err
	}

	log.Info("starting block number", blockNumber, "target block", targetBlock)

	task := func(block uint64, tx int, recording *substate.Substate, taskPool *substate.SubstateTaskPool) error {
		return evolve(block, tx, recording, stateDB, log)
	}

	sst := substate.SubstateTaskPool{
		Name:     "evolve",
		TaskFunc: task,

		First: blockNumber,
		Last:  targetBlock,

		Workers: ctx.Int(flagWorkers),

		DB: substateDB,
	}
	err = sst.Execute()

	log.Info("done")
	return nil
}

func evolve(block uint64, tx int, recording *substate.Substate, stateDB *db.StateSnapshotDB, log *logging.Logger) error {
	log.Info("block", block, "tx", tx, recording.InputAlloc)

	//check state of accounts before transaction
	//for address, account := range recording.InputAlloc {
	//	acc, err := stateDB.GetAccount(address)
	//	if err != nil {
	//		log.Errorf("Account %s was not found in snapshot database; %s", address.String(), err.Error())
	//	}
	//	if !acc.Equal(account) {
	//		log.Errorf("Account %s data are not matching in snapshot and substate databases.")
	//		return nil
	//	}
	//}

	for address, account := range recording.OutputAlloc {
		addrHash := crypto.Keccak256Hash(address.Bytes())
		_, err := stateDB.GetAccount(addrHash)
		if err != nil {
			log.Errorf("Account %s was not found in snapshot database; %s", address.String(), err.Error())
			break
		}

		// no need to insert if codeHash is already in database?
		err = stateDB.PutCode(account.Code)
		if err != nil {
			return err
		}

		newAccount := types.Account{Hash: addrHash, Storage: account.Storage, Code: account.Code}
		newAccount.Nonce = account.Nonce
		newAccount.Balance = account.Balance
		newAccount.CodeHash = account.CodeHash().Bytes()

		err = stateDB.PutAccount(&newAccount)
		if err != nil {
			log.Errorf("Unable to update account %s in database; %s", address.String(), err.Error())
			break
		}

	}
	return nil
}
