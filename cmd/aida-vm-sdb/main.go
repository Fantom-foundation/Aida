package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Carmen/go/database/mpt"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// RunVMApp data structure
var RunVMApp = cli.App{
	Name:      "Aida Storage Run VM Manager",
	Copyright: "(c) 2023 Fantom Foundation",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Commands: []*cli.Command{
		&RunSubstateCmd,
		&RunEthTestsCmd,
		&RunTxGeneratorCmd,
	},
	Description: `
The aida-vm-sdb command requires two arguments: <blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and last block of
the inclusive range of blocks.`,
}

var RunSubstateCmd = cli.Command{
	Action:    RunSubstate,
	Name:      "substate",
	Usage:     "Iterates over substates that are executed into a StateDb",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		// AidaDb
		&utils.AidaDbFlag,

		// StateDb
		&utils.CarmenSchemaFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.StateDbSrcFlag,
		&utils.DbTmpFlag,
		&utils.StateDbLoggingFlag,
		&utils.ValidateStateHashesFlag,

		// ArchiveDb
		&utils.ArchiveModeFlag,
		&utils.ArchiveQueryRateFlag,
		&utils.ArchiveMaxQueryAgeFlag,
		&utils.ArchiveVariantFlag,

		// ShadowDb
		&utils.ShadowDb,
		&utils.ShadowDbImplementationFlag,
		&utils.ShadowDbVariantFlag,

		// VM
		&utils.VmImplementation,

		// Profiling
		&utils.CpuProfileFlag,
		&utils.CpuProfilePerIntervalFlag,
		&utils.DiagnosticServerFlag,
		&utils.MemoryBreakdownFlag,
		&utils.MemoryProfileFlag,
		&utils.RandomSeedFlag,
		&utils.PrimeThresholdFlag,
		&utils.ProfileFlag,
		&utils.ProfileDepthFlag,
		&utils.ProfileFileFlag,
		&utils.ProfileSqlite3Flag,
		&utils.ProfileIntervalFlag,
		&utils.ProfileDBFlag,
		&utils.ProfileBlocksFlag,

		// RegisterRun
		&utils.RegisterRunFlag,
		&utils.OverwriteRunIdFlag,

		// Priming
		&utils.RandomizePrimingFlag,
		&utils.SkipPrimingFlag,
		&utils.UpdateBufferSizeFlag,

		// Utils
		&substate.WorkersFlag,
		&utils.ChainIDFlag,
		&utils.ContinueOnFailureFlag,
		&utils.SyncPeriodLengthFlag,
		&utils.KeepDbFlag,
		&utils.CustomDbNameFlag,
		//&utils.MaxNumTransactionsFlag,
		&utils.ValidateTxStateFlag,
		&utils.ValidateFlag,
		&logger.LogLevelFlag,
		&utils.NoHeartbeatLoggingFlag,
		&utils.TrackProgressFlag,
		&utils.ErrorLoggingFlag,
	},
	Description: `
The aida-vm-sdb substate command requires two arguments: <blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and last block of
the inclusive range of blocks.`,
}

var RunTxGeneratorCmd = cli.Command{
	Action: RunTxGenerator,
	Name:   "tx-generator",
	Usage:  "Generates transactions for specified block range and executes them over StateDb",
	Flags: []cli.Flag{
		// TxGenerator specific flags
		&utils.TxGeneratorTypeFlag,

		// StateDb
		&utils.CarmenSchemaFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.StateDbSrcFlag,
		&utils.DbTmpFlag,
		&utils.StateDbLoggingFlag,
		&utils.ValidateStateHashesFlag,

		// ShadowDb
		&utils.ShadowDb,
		&utils.ShadowDbImplementationFlag,
		&utils.ShadowDbVariantFlag,

		// RegisterRun
		&utils.RegisterRunFlag,
		&utils.OverwriteRunIdFlag,

		// VM
		&utils.VmImplementation,

		// Profiling
		&utils.CpuProfileFlag,
		&utils.CpuProfilePerIntervalFlag,
		&utils.DiagnosticServerFlag,
		&utils.MemoryBreakdownFlag,
		&utils.MemoryProfileFlag,

		// Utils
		&substate.WorkersFlag,
		&utils.ChainIDFlag,
		&utils.ContinueOnFailureFlag,
		&utils.KeepDbFlag,
		&utils.ValidateFlag,
		&logger.LogLevelFlag,
		&utils.NoHeartbeatLoggingFlag,
		&utils.BlockLengthFlag,
	},
	Description: `
The aida-vm-sdb tx-generator command requires two arguments: <blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and last block of
the inclusive range of blocks.`,
}

// main implements vm-sdb cli.
func main() {

	go func() {
		counter := 0
		fmt.Printf("MU, heap, mallocs, frees, live, sys, gcruns, cur_nodes, max_nodes\n")
		for {
			ticker := time.NewTicker(time.Second)
			select {
			case <-ticker.C:
				var stats runtime.MemStats
				runtime.ReadMemStats(&stats)
				fmt.Printf(
					"MU, %d, %d, %d, %d, %d, %d, %d, %d\n",
					stats.HeapAlloc,
					stats.Mallocs,
					stats.Frees,
					stats.Mallocs-stats.Frees,
					stats.Sys,
					stats.NumGC,
					mpt.GetCurrentNodeCount(),
					mpt.GetMaxNodeCount(),
				)
				if counter%10 == 0 {
					name := fmt.Sprintf("logs/heap_profile_%06d.dat", counter/10)
					f, err := os.Create(name)
					if err != nil {
						fmt.Printf("Failed to create heap profile: %v\n", err)
						continue
					}
					pprof.WriteHeapProfile(f)
					f.Close()
					fmt.Printf("Created profile %s\n", name)
				}
				counter++
			}
		}

	}()

	if err := RunVMApp.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
