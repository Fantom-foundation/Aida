// Package state implements executable entry points to the world state generator app.
package state

import (
	"context"
	"fmt"
	"github.com/Fantom-foundation/Aida/cmd/gen-world-state/flags"
	"github.com/Fantom-foundation/Aida/world-state/db/snapshot"
	"github.com/Fantom-foundation/Aida/world-state/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"io"
	"log"
	"math/big"
	"reflect"
	"time"
)

// CmdAccount defines a CLI command set for managing single account data in the state dump database.
var CmdAccount = cli.Command{
	Name:    "account",
	Aliases: []string{"a"},
	Usage:   `Provides information and management function for individual accounts in state dump database.`,
	Subcommands: []*cli.Command{
		&cmdAccountInfo,
		&cmdAccountCollect,
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

// cmdAccountCollect collects known accounts from the SubState database.
var cmdAccountCollect = cli.Command{
	Action:      collectAccounts,
	Name:        "collect",
	Usage:       "Collects known account addresses from substate database.",
	Description: "Command updates internal map of account hashes for the known accounts in substate database.",
	Aliases:     []string{"c"},
	Flags: []cli.Flag{
		&flags.SubstateDBPath,
		&flags.StartingBlock,
		&flags.EndingBlock,
		&flags.Workers,
	},
}

// balanceDecimals represents a decimal correction we do for the displayed balance (6 digits).
var balanceDecimals = big.NewInt(1_000_000_000_000)

// collectAccounts collects known accounts from the substate database.
func collectAccounts(ctx *cli.Context) error {
	// try to open state DB
	stateDB, err := snapshot.OpenStateDB(ctx.Path(flags.StateDBPath.Name))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(stateDB)

	// try to open sub state DB
	substate.SetSubstateDirectory(ctx.Path(flags.SubstateDBPath.Name))
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	workers := ctx.Int(flags.Workers.Name)
	iter := substate.NewSubstateIterator(ctx.Uint64(flags.StartingBlock.Name), workers)
	defer iter.Release()

	// load raw accounts
	accounts, storage := snapshot.CollectAccounts(ctx.Context, &iter, ctx.Uint64(flags.EndingBlock.Name), workers)

	// filter uniqueAccount addresses before writing
	uniqueAccount := make(chan any, cap(accounts))
	go snapshot.FilterUnique(ctx.Context, accounts, uniqueAccount)

	// filter uniqueStorage hashes before writing
	uniqueStorage := make(chan any, cap(storage))
	go snapshot.FilterUnique(ctx.Context, storage, uniqueStorage)

	// write found addresses
	errAcc := snapshot.WriteAccounts(ctx.Context, collectProgressFactory(ctx.Context, uniqueAccount, "account", Logger(ctx, "addr")), stateDB)

	// write found storage hashes
	errStorage := snapshot.WriteAccounts(ctx.Context, collectProgressFactory(ctx.Context, uniqueStorage, "storage", Logger(ctx, "storage")), stateDB)

	// check for any error in above execution threads;
	// this will block until all threads above close their error channels
	return getChannelError(errAcc, errStorage)
}

// collectProgressFactory observes progress in scanning.
func collectProgressFactory(ctx context.Context, in <-chan any, label string, log *logging.Logger) <-chan any {
	out := make(chan any, cap(in))
	go collectProgress(ctx, in, out, label, log)
	return out
}

// collectProgress reports progress on collector stream.
func collectProgress(ctx context.Context, in <-chan any, out chan<- any, label string, log *logging.Logger) {
	var count int
	var last string
	var err error
	tick := time.NewTicker(2 * time.Second)

	defer func() {
		tick.Stop()
		close(out)
	}()

	ctxDone := ctx.Done()
	for {
		select {
		case <-ctxDone:
			return
		case <-tick.C:
			log.Infof("observed %d %s; last one is %s", count, label, last)
		case adr, open := <-in:
			if !open {
				log.Noticef("found %d %s", count, label)
				return
			}

			last, err = getType(adr)
			if err != nil {
				log.Warning(err)
				continue
			}

			out <- adr
			count++
		}
	}
}

func getType(adr any) (string, error) {
	switch d := adr.(type) {
	case common.Address:
		return d.String(), nil
	case common.Hash:
		return d.String(), nil
	}
	return "", fmt.Errorf("unexpected type while writting to database %s", reflect.TypeOf(adr))
}

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
