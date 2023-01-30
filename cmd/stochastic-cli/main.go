package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/stochastic-cli/stochastic"
	"github.com/urfave/cli/v2"
)

// initStochasticApp initializes a stochastic-cli app. This function is
// called by the main function and unit tests.
func initStochasticApp() *cli.App {
	return &cli.App{
		Name:      "Aida Stochastic-Test Manager",
		HelpName:  "stochastic",
		Copyright: "(c) 2022-23 Fantom Foundation",
		Flags:     []cli.Flag{},
		Commands: []*cli.Command{
			&stochastic.StochasticRecordCommand,
			&stochastic.StochasticEstimatorCommand,
		},
	}
}

// main implements "stochastic" cli stochasticApplication.
func main() {
	app := initStochasticApp()
	if err := app.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
