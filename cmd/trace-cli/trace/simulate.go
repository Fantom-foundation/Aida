package trace

import (
	"github.com/Fantom-foundation/Aida/world-state/simulation"
	"github.com/Fantom-foundation/Carmen/go/state"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// SimulateCommand simulates traffic by using Markov chains
var SimulateCommand = cli.Command{
	Action:    SimulateAction,
	Name:      "simulate",
	Usage:     "",
	ArgsUsage: "<blockNum>",
	Flags: []cli.Flag{
		&numberOfBlocksFlag,
		&substate.WorkersFlag,
	},
	Description: `
The simulate command requires two arguments:
<blockNum>`,
}

func SimulateAction(ctx *cli.Context) error {
	stateDB, err := state.NewMemory()
	if err != nil {
		return err
	}
	t := simulation.InitTransitions()

	// number of blocks to be generated
	n := ctx.Uint(numberOfBlocksFlag.Name)

	workers := ctx.Int(substate.WorkersFlag.Name)

	simulation.Simulate(ctx.Context, state.CreateStateDBUsing(stateDB), t, n, workers)
	return nil
}
