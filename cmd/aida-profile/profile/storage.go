package profile

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/db"
	"github.com/Fantom-foundation/Substate/substate"
	substatetypes "github.com/Fantom-foundation/Substate/types"
	"github.com/urfave/cli/v2"
)

// GetStorageUpdateSizeCommand returns changes in storage size by transactions in the specified block range
var GetStorageUpdateSizeCommand = cli.Command{
	Action:    getStorageUpdateSizeAction,
	Name:      "storage-size",
	Usage:     "returns changes in storage size by transactions in the specified block range",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.WorkersFlag,
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The util-db storage-size command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to vm transactions.

Output log format: (block, timestamp, transaction, account, storage update size, storage size in input substate, storage size in output substate)`,
}

// computeStorageSize computes the number of non-zero storage entries
func computeStorageSizes(inUpdateSet map[substatetypes.Hash]substatetypes.Hash, outUpdateSet map[substatetypes.Hash]substatetypes.Hash) (int64, uint64, uint64) {
	deltaSize := int64(0)
	inUpdateSize := uint64(0)
	outUpdateSize := uint64(0)
	wordSize := uint64(32) //bytes
	for address, outValue := range outUpdateSet {
		if inValue, found := inUpdateSet[address]; found {
			if (inValue == substatetypes.Hash{} && outValue != substatetypes.Hash{}) {
				// storage increases by one new cell
				// (cell is empty in in-storage)
				deltaSize++
			} else if (inValue != substatetypes.Hash{} && outValue == substatetypes.Hash{}) {
				// storage shrinks by one new cell
				// (cell is empty in out-storage)
				deltaSize--
			}
		} else {
			// storage increases by one new cell
			// (cell is not found in in-storage but found in out-storage)
			if (outValue != substatetypes.Hash{}) {
				deltaSize++
			}
		}
		// compute update size
		if (outValue != substatetypes.Hash{}) {
			outUpdateSize++
		}
	}
	for address, inValue := range inUpdateSet {
		if _, found := outUpdateSet[address]; !found {
			// storage shrinks by one cell
			// (The cell does not exist for an address in in-storage)
			if (inValue != substatetypes.Hash{}) {
				deltaSize--
			}
		}
		if (inValue != substatetypes.Hash{}) {
			inUpdateSize++
		}
	}
	return deltaSize * int64(wordSize), inUpdateSize * wordSize, outUpdateSize * wordSize
}

// getStorageUpdateSizeTask replays storage access of accounts in each transaction
func getStorageUpdateSizeTask(block uint64, tx int, st *substate.Substate, taskPool *db.SubstateTaskPool) error {

	timestamp := st.Env.Timestamp
	for wallet, outputAccount := range st.OutputSubstate {
		var (
			deltaSize     int64
			inUpdateSize  uint64
			outUpdateSize uint64
		)
		// account exists in both input substate and output substate
		if inputAccount, found := st.InputSubstate[wallet]; found {
			deltaSize, inUpdateSize, outUpdateSize = computeStorageSizes(inputAccount.Storage, outputAccount.Storage)
			// account exists in output substate but not input substate
		} else {
			deltaSize, inUpdateSize, outUpdateSize = computeStorageSizes(map[substatetypes.Hash]substatetypes.Hash{}, outputAccount.Storage)
		}
		fmt.Printf("metric: %v,%v,%v,%v,%v,%v,%v\n", block, timestamp, tx, wallet.String(), deltaSize, inUpdateSize, outUpdateSize)
	}
	// account exists in input substate but not output substate
	for wallet, inputAccount := range st.InputSubstate {
		if _, found := st.OutputSubstate[wallet]; !found {
			deltaSize, inUpdateSize, outUpdateSize := computeStorageSizes(inputAccount.Storage, map[substatetypes.Hash]substatetypes.Hash{})
			fmt.Printf("metric: %v,%v,%v,%v,%v,%v,%v\n", block, timestamp, tx, wallet.String(), deltaSize, inUpdateSize, outUpdateSize)
		}
	}
	return nil
}

// func getStorageUpdateSizeAction for vm-storage command
func getStorageUpdateSizeAction(ctx *cli.Context) error {
	var err error

	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	log := logger.NewLogger(cfg.LogLevel, "Substate Replay")

	log.Infof("chain-id: %v\n", cfg.ChainID)

	sdb, err := db.NewReadOnlySubstateDB(cfg.AidaDb)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %w", err)
	}
	defer sdb.Close()

	taskPool := sdb.NewSubstateTaskPool("aida-vm storage", getStorageUpdateSizeTask, cfg.First, cfg.Last, ctx)
	err = taskPool.Execute()
	return err
}
