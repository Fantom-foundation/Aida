package trace

import (
	"bufio"
	"fmt"
	"github.com/Fantom-foundation/Aida/world-state/simulation"
	common_eth "github.com/ethereum/go-ethereum/common"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/Fantom-foundation/Aida/tracer"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// StochasticReplayCommand data structure for the StochasticReplay app
var StochasticReplayCommand = cli.Command{
	Action:    traceStochasticReplayAction,
	Name:      "stochastic-replay-new",
	Usage:     "executes storage trace",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&cpuProfileFlag,
		&disableProgressFlag,
		&profileFlag,
		&stateDbImplementation,
		&stateDbVariant,
		&substate.WorkersFlag,
		&traceDirectoryFlag,
		&traceDebugFlag,
		&numberOfBlocksFlag,
	},
	Description: `
The trace StochasticReplay command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to StochasticReplay storage traces.`,
}

// traceStochasticReplayTask simulates storage operations from storage traces on stateDB.
func traceStochasticReplayTask(cfg *TraceConfig) error {

	dCtx := dict.NewDictionaryContext()
	// TODO load data into dictionary

	// create a directory for the store to place all its files, and
	// instantiate the state DB under testing.
	log.Printf("Create stateDB database")
	stateDirectory, err := ioutil.TempDir("", "state_db_*")
	if err != nil {
		return err
	}
	db, err := makeStateDB(stateDirectory, cfg.impl, cfg.variant)
	if err != nil {
		return err
	}

	// progress message setup
	var (
		start   time.Time
		sec     float64
		lastSec float64
	)
	if cfg.enableProgress {
		start = time.Now()
		sec = time.Since(start).Seconds()
		lastSec = time.Since(start).Seconds()
	}

	transitions, err := loadTransitions()
	if err != nil {
		return err
	}

	distContract, distStorage, distValue, err := getGenerators(dCtx)
	if err != nil {
		return err
	}

	sc, err := simulation.NewStateContext(distContract, distStorage, distValue, transitions, dCtx)
	if err != nil {
		return err
	}

	for {
		op := sc.NextOperation()
		if op == nil {
			log.Fatalf("operation was null")
		}
		if op.GetId() == operation.BeginBlockID {
			block := op.(*operation.BeginBlock).BlockNumber
			if block > cfg.last {
				break
			}
			if cfg.enableProgress {
				// report progress
				sec = time.Since(start).Seconds()
				if sec-lastSec >= 15 {
					log.Printf("elapsed time: %.0f s, at block %v\n", sec, block)
					lastSec = sec
				}
			}
		}
		operation.Execute(op, db, dCtx)
		if cfg.debug {
			operation.Debug(dCtx, op)
		}
	}
	sec = time.Since(start).Seconds()

	log.Printf("Finished StochasticReplaying storage operations on StateDB database")

	// print profile statistics (if enabled)
	if operation.EnableProfiling {
		operation.PrintProfiling()
	}

	// close the DB and print disk usage
	log.Printf("Close StateDB database")
	start = time.Now()
	if err := db.Close(); err != nil {
		log.Printf("Failed to close database: %v", err)
	}

	// print progress summary
	if cfg.enableProgress {
		log.Printf("trace StochasticReplay: Total elapsed time: %.3f s, processed %v blocks\n", sec, cfg.last-cfg.first+1)
		log.Printf("trace StochasticReplay: Closing DB took %v\n", time.Since(start))
		log.Printf("trace StochasticReplay: Final disk usage: %v MiB\n", float32(getDirectorySize(stateDirectory))/float32(1024*1024))
	}

	return nil
}

func getGenerators(dCtx *dict.DictionaryContext) (simulation.Generator, simulation.Generator, simulation.Generator, error) {
	newContract, newStorage, newValue, err := loadNewOccurances()
	if err != nil {
		return simulation.Generator{}, simulation.Generator{}, simulation.Generator{}, err
	}

	lambdaContract, err := loadLambda("contract-distribution.dat")
	//if err != nil {
	//	return simulation.Generator{}, simulation.Generator{}, simulation.Generator{}, err
	//}
	lambdaStorage, err := loadLambda("storage-distribution.dat")
	//if err != nil {
	//	return simulation.Generator{}, simulation.Generator{}, simulation.Generator{}, err
	//}
	lambdaValue, err := loadLambda("value-distribution.dat")
	//if err != nil {
	//	return simulation.Generator{}, simulation.Generator{}, simulation.Generator{}, err
	//}

	gc := simulation.Generator{C: newContract, Size: 0, E: lambdaContract}
	gc.GetNew = func() []any {
		gc.Size++
		var address = common_eth.BytesToAddress(simulation.RetrieveValueAt(gc.Size))
		idx := dCtx.EncodeContract(address)
		return []any{idx}
	}
	gs := simulation.Generator{C: newStorage, Size: 0, E: lambdaStorage}
	gs.GetNew = func() []any {
		gs.Size++
		key := common_eth.BytesToHash(simulation.RetrieveValueAt(gs.Size))
		sIdx, pos := dCtx.EncodeStorage(common_eth.BytesToHash(key[:]))

		return []any{sIdx, pos}
	}
	gv := simulation.Generator{C: newValue, Size: 0, E: lambdaValue}
	gv.GetNew = func() []any {
		gv.Size++
		value := common_eth.BytesToHash(simulation.RetrieveValueAt(gv.Size))
		idx := dCtx.EncodeValue(value)

		return []any{idx}
	}

	return gc, gs, gv, nil
}

// TODO this calculation should be moved into stochastic record
func loadLambda(path string) (float64, error) {
	file, err := os.Open(dict.DictionaryContextDir + path)
	if err != nil {
		log.Fatalf("Cannot open %s file. Error: %v", path, err)
	}
	defer file.Close()

	// string is actually float32 but no need to parse
	occurances := make(map[string]uint32)

	for {
		rd := bufio.NewReader(file)
		line, err := rd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			err = fmt.Errorf("read file line error: %v", err)
			return 0, err
		}

		p := strings.Split(line, " - ")
		if len(p) != 2 {
			return 0, fmt.Errorf("file %s is in incorrect format", path)
		}

		occurances[strings.TrimSuffix(p[1], " \n")]++
	}

	s := make(map[uint32]uint32)

	// counting up occurances
	for _, occs := range occurances {
		s[occs]++
	}

	log.Print("hello", s, "\n")

	return 0, nil
}

func loadNewOccurances() ([]float32, []float32, []float32, error) {
	newContract := make([]float32, operation.NumProfiledOperations)
	newStorage := make([]float32, operation.NumProfiledOperations)
	newValue := make([]float32, operation.NumProfiledOperations)

	f, err := readFrequenciesFile()
	if err != nil {
		return nil, nil, nil, err
	}

	for i := 0; i < operation.NumProfiledOperations; i++ {
		newContract[i] = float32(f[1][i]) / float32(f[0][i])
		newStorage[i] = float32(f[2][i]) / float32(f[0][i])
		newValue[i] = float32(f[3][i]) / float32(f[0][i])
	}

	return newContract, newStorage, newValue, nil
}

func readFrequenciesFile() ([][]uint64, error) {
	file, err := os.Open(dict.DictionaryContextDir + "frequencies.dat")
	if err != nil {
		log.Fatalf("Cannot open frequencies.dat file. Error: %v", err)
	}
	defer file.Close()

	res := make([][]uint64, operation.NumProfiledOperations)

	i := 0
	for {
		rd := bufio.NewReader(file)
		line, err := rd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			err = fmt.Errorf("read file line error: %v", err)
			return nil, err
		}

		p := strings.Split(line, ",")
		if len(p) != operation.NumProfiledOperations {
			err = fmt.Errorf("frequencies.dat file doesn't contain correct number of operations")
			return nil, err
		}
		l := make([]uint64, operation.NumProfiledOperations)
		for k, s := range p {
			//TrimSuffix last item has new line
			j, err := strconv.ParseUint(strings.TrimSuffix(s, "\n"), 10, 64)
			if err != nil {
				return nil, err
			}
			l[k] = j
		}

		i++
	}

	if i != 4 {
		err = fmt.Errorf("incomplete data in frequencies.dat file")
		return nil, err
	}

	return res, nil
}

func loadTransitions() ([][]float64, error) {
	file, err := os.Open(dict.DictionaryContextDir + "stochastic-matrix.csv")
	if err != nil {
		log.Fatalf("Cannot open stochastic-matrix.dat file. Error: %v", err)
	}

	res := make([][]float64, operation.NumProfiledOperations)
	i := 0
	rd := bufio.NewReader(file)
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			err = fmt.Errorf("read file line error: %v", err)
			return nil, err
		}
		p := strings.Split(line, ",")
		if len(p) != operation.NumProfiledOperations {
			err = fmt.Errorf("stochastic-matrix file doesn't contain correct number of operations")
			return nil, err
		}
		l := make([]float64, operation.NumProfiledOperations)
		for k, s := range p {
			//TrimSuffix last item has new line
			j, err := strconv.ParseFloat(strings.TrimSuffix(s, "\n"), 64)
			if err != nil {
				return nil, err
			}
			l[k] = j
		}
		res[i] = l
		i++
	}

	if i != operation.NumProfiledOperations {
		err = fmt.Errorf("stochastic-matrix file doesn't contain correct number of rows")
		return nil, err
	}
	return res, nil
}

// traceStochasticReplayAction implements trace command for StochasticReplaying.
func traceStochasticReplayAction(ctx *cli.Context) error {
	var err error
	cfg, err := NewTraceConfig(ctx)
	if err != nil {
		return err
	}

	seed := ctx.Int64(stochasticSeedFlag.Name)
	if seed != -1 {
		rand.Seed(seed)
	} else {
		rand.Seed(time.Now().UnixNano())
	}

	operation.EnableProfiling = cfg.profile
	// set trace directory
	tracer.TraceDir = ctx.String(traceDirectoryFlag.Name) + "/"
	dict.DictionaryContextDir = ctx.String(traceDirectoryFlag.Name) + "/"

	// start CPU profiling if requested.
	if profileFileName := ctx.String(cpuProfileFlag.Name); profileFileName != "" {
		f, err := os.Create(profileFileName)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	err = traceStochasticReplayTask(cfg)

	return err
}
