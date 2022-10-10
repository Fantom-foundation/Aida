// Package account implements information providers for individual accounts in the state dump database.
package account

import (
	"fmt"
	"github.com/Fantom-foundation/Aida-Testing/world-state/db"
	"github.com/Fantom-foundation/Aida-Testing/world-state/dump"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
	"io"
	"log"
	"math/big"
)

// flagWithStorage represents a flag for displaying the full storage content.
const flagWithStorage = "with-storage"

// cmdAccountInfo is the sub-command for providing details account information.
// build/gen-world-state --db=<path> account info "0xFC00FACE00000000000000000000000000000000"
var cmdAccountInfo = cli.Command{
	Action:      accountInfo,
	Name:        "info",
	Usage:       "Provides detailed information about the target account.",
	Description: "Command provides detailed information about the account specified as an argument.",
	ArgsUsage:   "<address>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  flagWithStorage,
			Usage: "display full storage content",
			Value: false,
		},
	},
}

// balanceDecimals represents a decimal correction we do for the displayed balance (4 digits => 18 - 4 = 14 decimals).
var balanceDecimals = big.NewInt(100_000_000_000_000)

// accountInfo sends detailed account information to the console output stream.
func accountInfo(ctx *cli.Context) error {
	// check if we have an address
	if ctx.Args().Len() < 1 || !common.IsHexAddress(ctx.Args().Get(0)) {
		return fmt.Errorf("valid account address not provided")
	}

	// try to open output DB
	snapDB, err := db.OpenStateSnapshotDB(ctx.Path(dump.FlagOutputDBPath))
	if err != nil {
		return err
	}
	defer db.MustCloseSnapshotDB(snapDB)

	// try to get the account
	addr := common.HexToAddress(ctx.Args().Get(0))
	acc, err := snapDB.Account(addr)
	if err != nil {
		return err
	}

	// display base account information
	baseInfo(ctx.App.Writer, addr, acc)

	// display the storage content if requested
	if ctx.Bool(flagWithStorage) {
		accStorage(ctx.App.Writer, acc)
	}

	return nil
}

// baseInfo sends formatted base account information to the output writer.
func baseInfo(w io.Writer, addr common.Address, acc *types.Account) {
	bold := color.New(color.Bold).SprintfFunc()

	output(w, "Account:\t%s\n", bold(addr.String()))
	output(w, "DB Hash:\t%s\n", bold(acc.Hash.String()))

	balance := float64(new(big.Int).Div(acc.Balance, balanceDecimals).Int64()) / 10000.0
	output(w, "Balance:\t%s\n", bold("%0.4f FTM", balance))

	output(w, "Nonce:\t\t%s\n", bold("%d", acc.Nonce))

	hash := common.Hash{}
	hash.SetBytes(acc.CodeHash)
	output(w, "Code Hash:\t%s\n", bold(hash.String()))
	output(w, "Code Length:\t%s bytes\n", bold("%d", len(acc.Code)))

	output(w, "Storage Root:\t%s\n", bold(acc.Root.String()))
	output(w, "Storage Items:\t%s\n", bold("%d", len(acc.Storage)))
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
