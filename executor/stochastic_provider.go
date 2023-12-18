package executor

import (
	"math/rand"

	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/urfave/cli/v2"
)

func OpenSimulations(e *stochastic.EstimationModelJSON, ctxt *cli.Context, rg *rand.Rand) (res Provider[stochastic.Data], err error) {
	return simulationsProvider{
		ctxt:       ctxt,
		simulation: e,
		rg:         rg,
	}, nil
}

type simulationsProvider struct {
	ctxt       *cli.Context
	simulation *stochastic.EstimationModelJSON
	rg         *rand.Rand
}

func (s simulationsProvider) Run(from int, to int, consumer Consumer[stochastic.Data]) error {
	operations, matrix, markovianState := stochastic.GetStochasticMatrix(s.simulation)
	block := from
	for {
		data := stochastic.DecodeOpcode(operations[markovianState])
		if data.Operation == stochastic.BeginBlockID {
			block++
		}
		if block >= to {
			return nil
		}

		err := consumer(TransactionInfo[stochastic.Data]{block, markovianState, data})
		if err != nil {
			return err
		}

		markovianState = stochastic.NextState(s.rg, matrix, markovianState)
	}

}

func (s simulationsProvider) Close() {
	// ignored
}
