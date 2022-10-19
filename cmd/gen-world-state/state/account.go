// Package state implements executable entry points to the world state generator app.
package state

import (
	"fmt"
	"github.com/Fantom-foundation/Aida/cmd/gen-world-state/flags"
	"github.com/Fantom-foundation/Aida/world-state/db/snapshot"
	"github.com/Fantom-foundation/Aida/world-state/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"io"
	"log"
	"math/big"
)

// CmdAccount defines a CLI command set for managing single account data in the state dump database.
var CmdAccount = cli.Command{
	Name:    "account",
	Aliases: []string{"a"},
	Usage:   `Provides information and management function for individual accounts in state dump database.`,
	Subcommands: []*cli.Command{
		&cmdAccountInfo,
	},
}

// cmdAccountInfo is the sub-command for providing details account information.
// build/gen-world-state --db=<path> account info "0xFC00FACE00000000000000000000000000000000"
var cmdAccountInfo = cli.Command{
	Action:      accountInfo,
	Aliases:     []string{"i"},
	Name:        "info",
	Usage:       "Provides detailed information about the target account.",
	Description: "Command provides detailed information about the account specified as an argument.",
	ArgsUsage:   "<address|hash>",
	Flags: []cli.Flag{
		&flags.IsStorageIncluded,
	},
}

// balanceDecimals represents a decimal correction we do for the displayed balance (6 digits).
var balanceDecimals = big.NewInt(1_000_000_000_000)

// accountInfo sends detailed account information to the console output stream.
func accountInfo(ctx *cli.Context) error {
	// check if we have an address
	if ctx.Args().Len() < 1 {
		return fmt.Errorf("please provide account address, or account key hash")
	}

	// try to open output DB
	snapDB, err := snapshot.OpenStateDB(ctx.Path(flags.StateDBPath.Name))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(snapDB)

	// try to get the account
	var acc *types.Account
	var addr common.Address

	// regular address provided
	if common.IsHexAddress(ctx.Args().Get(0)) {
		addr = common.HexToAddress(ctx.Args().Get(0))
		acc, err = snapDB.Account(addr)
	} else {
		acc, err = snapDB.AccountByHash(common.HexToHash(ctx.Args().Get(0)))
	}

	if err != nil {
		return err
	}

	// display base account information
	baseInfo(ctx.App.Writer, addr, acc)

	// display the storage content if requested
	if ctx.Bool(flags.IsStorageIncluded.Name) {
		accStorage(ctx.App.Writer, acc)
	}
	return nil
}

// baseInfo sends formatted base account information to the output writer.
func baseInfo(w io.Writer, addr common.Address, acc *types.Account) {
	bold := color.New(color.Bold).SprintfFunc()
	colored := color.New(color.FgBlue, color.Bold).SprintfFunc()
	m := message.NewPrinter(language.English)

	output(w, "Account:\t%s\n", colored(addr.String()))
	output(w, "DB Hash:\t%s\n", colored(acc.Hash.String()))

	balance := float64(new(big.Int).Div(acc.Balance, balanceDecimals).Int64()) / 1000000.0
	output(w, "Balance:\t%s\n", bold(m.Sprintf("%0.6f FTM", balance)))

	output(w, "Nonce:\t\t%s\n", bold(m.Sprintf("%d", acc.Nonce)))

	hash := common.Hash{}
	hash.SetBytes(acc.CodeHash)
	output(w, "Code Hash:\t%s\n", bold(hash.String()))
	output(w, "Code Length:\t%s bytes\n", bold(m.Sprintf("%d", len(acc.Code))))
	output(w, "Storage Items:\t%s\n", bold(m.Sprintf("%d", len(acc.Storage))))
}

// accStorage sends formatted table of account storage content into the output writer.
func accStorage(w io.Writer, acc *types.Account) {
	tbl := tablewriter.NewWriter(w)
	tbl.SetHeader([]string{"Key", "Value"})
	tbl.SetBorder(true)

	for k, v := range acc.Storage {
		tbl.Append([]string{k.String(), v.String()})
	}

	tbl.Render()
}

// output the given message with formatting.
func output(w io.Writer, format string, a ...any) {
	_, err := fmt.Fprintf(w, format, a...)
	if err != nil {
		log.Println("output error", err.Error())
	}
}
