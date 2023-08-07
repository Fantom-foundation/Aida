package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/rc-testing/test/itest/cmd/stvm"
	"github.com/Fantom-foundation/rc-testing/test/itest/cmd/stvmdb"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:     "Integration Tester",
		HelpName: "itest",
		Copyright: "(c) 2022 Fantom Foundation",
		Flags:     []cli.Flag{},
		Commands: []*cli.Command{
			&stvm.StVmCommand,
			&stvmdb.StVmDbCommand,
		},
	}
	if err := app.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
