package main

import (
	"fmt"
	"log"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// RunVmAdb performs block processing on an ArchiveDb
func RunVmAdb(ctx *cli.Context) error {
	config, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	config.SrcDbReadonly = true

	// executing archive blocks always calls ArchiveDb with block -1
	// this condition prevents an incorrect call for block that does not exist (block number -1 in this case)
	// there is nothing before block 0 so running this app on this block does nothing
	if config.First == 0 {
		config.First = 1
	}

	substateDb, err := executor.OpenSubstateDb(config, ctx)
	if err != nil {
		return err
	}
	defer substateDb.Close()

	db, _, err := utils.PrepareStateDB(config)
	if err != nil {
		return err
	}

	p := makeTxProcessor(config, db)
	go p.unite(int(config.First), int(config.Last))

	// start workers
	for i := 0; i < config.Workers; i++ {
		go p.process()
	}

	return run(config, substateDb, db, p)
}

// makeTxProcessor which processes united transactions by block. United transactions are processed in parallel.
func makeTxProcessor(config *utils.Config, db state.StateDB) *txProcessor {
	return &txProcessor{
		config:    config,
		stateCh:   make(chan executor.State, 10*config.Workers),
		archiveCh: make(chan state.StateDB, 2*config.Workers),
		toProcess: make(chan unitedStates, 2*config.Workers),
		db:        db,
	}
}

type txProcessor struct {
	config    *utils.Config
	stateCh   chan executor.State
	archiveCh chan state.StateDB
	toProcess chan unitedStates
	db        state.StateDB
}

// unitedStates are all united with correct archive state by block number
type unitedStates struct {
	states  []executor.State
	archive state.StateDB
}

type archiveGetter struct {
	extension.NilExtension
}

// PreBlock sends needed archive to the processor.
func (r *archiveGetter) PreBlock(state executor.State, context *executor.Context) error {
	var err error
	context.Archive, err = context.State.GetArchiveState(uint64(state.Block) - 1)
	if err != nil {
		return err
	}

	return nil
}

// unite transactions by blocks and wait until archive arrives then send transactions to process with given archive.
func (r *txProcessor) unite(first, last int) {
	var (
		united  unitedStates
		archive state.StateDB
		firstB  = true
		block   = first
	)

	defer close(r.toProcess)

	for {
		select {
		// when archive arrives it means we united all transactions for previous block
		// hence we can send transactions to process with correct archive state
		case archive = <-r.archiveCh:
			// save first archive - translations have not been united for first block yet
			if firstB {
				firstB = false
				united.archive = archive
				continue
			}

			r.toProcess <- united

			if united.states[0].Block == last {
				return
			}

			// reset states and assign archive for next block
			united.states = []executor.State{}
			united.archive = archive

		case s := <-r.stateCh:
			if s.Block > block {
				r.toProcess <- united
				united.states = []executor.State{}
				block = s.Block
			}
			united.states = append(united.states, s)

		}
	}
}

// process united transactions by block.
func (r *txProcessor) process() {
	var u unitedStates
	for {
		u = <-r.toProcess
		archive, err := r.db.GetArchiveState(uint64(u.states[0].Block) - 1)
		if err != nil {
			log.Fatal(err)
		}
		//r.archiveCh <- archive
		for _, s := range u.states {
			fmt.Println(s.Block)
			fmt.Println(s.Transaction)
			_, err := utils.ProcessTx(
				archive,
				r.config,
				uint64(s.Block),
				s.Transaction,
				s.Substate,
			)
			if err != nil {
				log.Fatal(err)
			}
		}
		archive.Close()
	}
}

func (r *txProcessor) Process(state executor.State, context *executor.Context) error {
	r.stateCh <- state
	return nil
}

func run(config *utils.Config, provider executor.SubstateProvider, stateDb state.StateDB, p executor.Processor) error {
	// order of extensionList has to be maintained
	var extensionList = []executor.Extension{extension.MakeCpuProfiler(config)}

	//if !disableStateDbExtension {
	//	extensionList = append(extensionList, extension.MakeStateDbManager(config))
	//}

	extensionList = append(extensionList, []executor.Extension{
		&archiveGetter{},
		extension.MakeProgressTracker(config, 0),
		extension.MakeStateDbPreparator(),
		extension.MakeBeginOnlyEmitter(),
	}...)
	return executor.NewExecutor(provider).Run(
		executor.Params{
			From:       int(config.First),
			To:         int(config.Last) + 1,
			State:      stateDb,
			NumWorkers: config.Workers,
		},
		p,
		extensionList,
	)
}
