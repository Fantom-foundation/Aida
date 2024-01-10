package main

import (
	"fmt"
	"os"

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
	},
	Description: `
The aida-vm-sdb command requires two arguments: <blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and last block of
the inclusive range of blocks.`,
}

// main implements vm-sdb cli.
func main() {
	if err := RunVMApp.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
