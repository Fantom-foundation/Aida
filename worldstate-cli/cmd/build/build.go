// Package build provides build time information from the make process.
//
// NOTE: Manual build will not inject versioning data into the binary application
// and only default values will be provided by the help subcommand.
package build

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"strings"
	"time"
)

// Version represents the version of the app.
var Version = "0.0"

// Commit represents the GitHub commit hash the app was built from.
var Commit = "0000000000000000000000000000000000000000"

// CommitTime represents the GitHub commit time stamp the app was built from.
var CommitTime = time.RFC1123Z

// Time represents the time of the app build.
var Time = "Mon, 01 Jan 2000 00:00:00"

// Compiler represents the information about the compiler used to build the app.
var Compiler = "go version unknown"

// MarkReset represents a token for terminal style reset.
var MarkReset = "\033[0m"

// MarkEnable represents a token for marking text in terminal style.
var MarkEnable = "\033[1;34m"

// CmdVersion defines the version output subcommand for the World State CLI app
var CmdVersion = cli.Command{
	Name:    "version",
	Aliases: []string{"v"},
	Usage:   "Provides information about the application version and build details",
	Action: func(_ *cli.Context) error {
		fmt.Print(VersionInfo())
		return nil
	},
}

// VersionInfo returns a preformatted app version information suitable for direct printing into terminal.
func VersionInfo() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("%sApp Version:%s\t%s\n", MarkEnable, MarkReset, Version))
	b.WriteString(fmt.Sprintf("%sCommit Hash:%s\t%s\n", MarkEnable, MarkReset, Commit))
	b.WriteString(fmt.Sprintf("%sCommit Time:%s\t%s\n", MarkEnable, MarkReset, CommitTime))
	b.WriteString(fmt.Sprintf("%sBuild Time:%s\t%s\n", MarkEnable, MarkReset, Time))
	b.WriteString(fmt.Sprintf("%sCompiler:%s\t%s\n", MarkEnable, MarkReset, Compiler))

	return b.String()
}
