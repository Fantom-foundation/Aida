package trace

import (
	"github.com/Fantom-foundation/Aida/world-state/simulation"
	"github.com/Fantom-foundation/Carmen/go/common"
	"github.com/Fantom-foundation/Carmen/go/state"
	"github.com/urfave/cli/v2"
	"math"
)

// SimulateCommand simulates traffic by using Markov chains
var SimulateCommand = cli.Command{
	Action:    SimulateAction,
	Name:      "simulate",
	Usage:     "",
	ArgsUsage: "<blockNum>",
	Flags: []cli.Flag{
		&numberOfBlocksFlag,
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
	// TODO - how to generate max range for Address and Keys?
	dist := common.Exponential.GetDistribution(math.MaxInt)
	t := simulation.InitTransitions()

	blocksNum := ctx.Uint(numberOfBlocksFlag.Name)
	simulation.Simulate(ctx.Context, state.CreateStateDBUsing(stateDB), dist, t, blocksNum)
	return nil
}
