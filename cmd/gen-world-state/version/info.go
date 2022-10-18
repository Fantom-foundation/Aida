// Package version provides build time information from the make process.
//
// NOTE: Manual build will not inject versioning data into the binary application
// and only default values will be provided by the help subcommand.
package version

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"strings"
)

// markReset represents a token for terminal style reset.
const markReset = "\033[0m"

// markEnable represents a token for marking text in terminal style.
const markEnable = "\033[1;34m"

// CmdVersion defines the version output subcommand for the World State CLI app
var CmdVersion = cli.Command{
	Name:    "version",
	Aliases: []string{"v"},
	Usage:   "Provides information about the application version and build details",
	Action: func(_ *cli.Context) error {
		fmt.Print(Long())
		return nil
	},
}

// Long returns a long version of preformatted app build information
// suitable for direct printing into terminal.
func Long() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("%sApp Version:%s\t%s\n", markEnable, markReset, Version))
	b.WriteString(fmt.Sprintf("%sCommit Hash:%s\t%s\n", markEnable, markReset, Commit))
	b.WriteString(fmt.Sprintf("%sCommit Time:%s\t%s\n", markEnable, markReset, CommitTime))
	b.WriteString(fmt.Sprintf("%sBuild Time:%s\t%s\n", markEnable, markReset, Time))
	b.WriteString(fmt.Sprintf("%sCompiler:%s\t%s\n", markEnable, markReset, Compiler))

	return b.String()
}
