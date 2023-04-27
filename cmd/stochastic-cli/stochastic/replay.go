package stochastic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// StochasticReplayCommand data structure for the replay app.
var StochasticReplayCommand = cli.Command{
	Action:    stochasticReplayAction,
	Name:      "replay",
	Usage:     "Simulates StateDB operations using a random generator with realistic distributions",
	ArgsUsage: "<simulation-length> <simulation-file>",
	Flags: []cli.Flag{
		&utils.CarmenSchemaFlag,
		&utils.ContinueOnFailureFlag,
		&utils.CpuProfileFlag,
		&utils.DebugFromFlag,
		&utils.QuietFlag,
		&utils.MemoryBreakdownFlag,
		&utils.RandomSeedFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.DbTmpFlag,
		&utils.StateDbLoggingFlag,
		&utils.TraceFileFlag,
		&utils.TraceDebugFlag,
		&utils.TraceFlag,
		&utils.ShadowDbImplementationFlag,
		&utils.ShadowDbVariantFlag,
		&utils.AidaDbFlag,
	},
	Description: `
The stochastic replay command requires two argument:
<simulation-length> <simulation.json> 

<simulation-length> determines the number of blocks
<simulation.json> contains the simulation parameters produced by the stochastic estimator.`,
}

// stochasticReplayAction implements the replay command. The user
// provides simulation file and simulation as arguments.
func stochasticReplayAction(ctx *cli.Context) error {
	// parse command-line arguments
	if ctx.Args().Len() != 2 {
		return fmt.Errorf("missing simulation file and simulation length as parameter")
	}
	simLength, perr := strconv.ParseInt(ctx.Args().Get(0), 10, 64)
	if perr != nil {
		return fmt.Errorf("simulation length is not an integer. Error: %v", perr)
	}

	// process configuration
	cfg, err := utils.NewConfig(ctx, utils.LastBlockArg)
	if err != nil {
		return err
	}
	if cfg.DbImpl == "memory" {
		return fmt.Errorf("db-impl memory is not supported")
	}

	// start CPU profiling if requested.
	if err := utils.StartCPUProfile(cfg); err != nil {
		return err
	}
	defer utils.StopCPUProfile(cfg)

	// read simulation file
	simulation, serr := readSimulation(ctx.Args().Get(1))
	if serr != nil {
		return fmt.Errorf("failed reading simulation. Error: %v", serr)
	}

	// create a directory for the store to place all its files, and
	// instantiate the state DB under testing.
	log.Printf("Create stateDB database")
	db, stateDirectory, _, err := utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}
	defer os.RemoveAll(stateDirectory)

	// Enable tracing if debug flag is set
	if cfg.Trace {
		rCtx := context.NewRecord(cfg.TraceFile)
		defer rCtx.Close()
		db = tracer.NewProxyRecorder(db, rCtx)
	}

	// run simulation.
	fmt.Printf("stochastic replay: run simulation ...\n")

	runErr := stochastic.RunStochasticReplay(db, simulation, int(simLength), cfg)

	// print memory usage after simulation
	if cfg.MemoryBreakdown {
		if usage := db.GetMemoryUsage(); usage != nil {
			log.Printf("stochastic replay: state DB memory usage: %d byte\n%s\n", usage.UsedBytes, usage.Breakdown)
		} else {
			log.Printf("Utilized storage solution does not support memory breakdowns.\n")
		}
	}

	// close the DB and print disk usage
	start := time.Now()
	if err := db.Close(); err != nil {
		log.Printf("Failed to close database: %v", err)
	}
	log.Printf("stochastic replay: Closing DB took %v\n", time.Since(start))
	log.Printf("stochastic replay: Final disk usage: %v MiB\n", float32(utils.GetDirectorySize(stateDirectory))/float32(1024*1024))

	return runErr
}

// readSimulation reads the simulation file in JSON format (generated by the estimator).
func readSimulation(filename string) (*stochastic.EstimationModelJSON, error) {
	// open simulation file and read JSON
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed opening simulation file")
	}
	defer file.Close()

	// read file into memory
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed reading simulation file")
	}

	// convert text to JSON object
	var simulation stochastic.EstimationModelJSON
	err = json.Unmarshal(contents, &simulation)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshalling JSON")
	}

	return &simulation, nil
}
