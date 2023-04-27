// Package state implements executable entry points to the world state generator app.
package state

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/Fantom-foundation/Aida/cmd/worldstate-cli/flags"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/world-state/db/snapshot"
	"github.com/Fantom-foundation/Aida/world-state/types"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// CmdAccount defines a CLI command set for managing single account data in the state dump database.
var CmdAccount = cli.Command{
	Name:    "account",
	Aliases: []string{"a"},
	Usage:   `Provides information and management function for individual accounts in state dump database.`,
	Subcommands: []*cli.Command{
		&cmdAccountInfo,
		&cmdAccountCollect,
		&cmdAccountImport,
		&cmdUnknown,
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

// cmdUnknown is command for storage and account unknown searches
var cmdUnknown = cli.Command{
	Name:        "unknown",
	Usage:       "Lists unknown account addresses or storages from the world state database.",
	Description: "Command scans for addresses in the world state database and shows those not available in the address map.",
	Aliases:     []string{"u"},
	Subcommands: []*cli.Command{
		&cmdUnknownStorage,
		&cmdUnknownAddress,
	},
}

// cmdUnknownStorage scans the account map vs. account hashes and provides a list of unknown accounts
// in the world state.
var cmdUnknownStorage = cli.Command{
	Action:      listUnknownStorages,
	Name:        "storage",
	Usage:       "Lists unknown account storages from the world state database.",
	Description: "Command scans for storage keys in the world state database and shows those not available in the address map.",
	Flags: []cli.Flag{
		&flags.IsVerbose,
	},
}

// cmdUnknownAddress scans the account map vs. account hashes and provides a list of unknown storage keys
// in the world state.
var cmdUnknownAddress = cli.Command{
	Action:      listUnknownAddress,
	Name:        "address",
	Usage:       "Lists unknown account addresses from the world state database.",
	Description: "Command scans for addresses in the world state database and shows those not available in the address map.",
	Flags: []cli.Flag{
		&flags.IsVerbose,
	},
}

// cmdAccountImport imports accounts from a simple CSV account list and fills the HASH to Account Address mapping.
// build/gen-world-state --db=<path> account import <csv file path>
var cmdAccountImport = cli.Command{
	Action:      accountImport,
	Name:        "import",
	Aliases:     []string{"csv"},
	Usage:       "Imports account addresses or storages for hash mapping from a CSV file.",
	Description: "Command imports account hash to account address mapping from a CSV file.",
	ArgsUsage:   "<csv file path|- for stdin>",
	Flags: []cli.Flag{
		&flags.IsVerbose,
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

	// try to open substate DB
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
	errAcc := snapshot.WriteAccounts(ctx.Context, collectProgressFactory(ctx.Context, uniqueAccount, "account", utils.NewLogger(ctx.String(utils.LogLevelFlag.Name), "addr")), stateDB)

	// write found storage hashes
	errStorage := snapshot.WriteAccounts(ctx.Context, collectProgressFactory(ctx.Context, uniqueStorage, "storage", utils.NewLogger(ctx.String(utils.LogLevelFlag.Name), "storage")), stateDB)

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

// listUnknownAddress implements unknown accounts scan.
func listUnknownAddress(ctx *cli.Context) error {
	// try to open output DB
	db, err := snapshot.OpenStateDB(ctx.Path(flags.StateDBPath.Name))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(db)

	// out what we do
	_, err = fmt.Fprintf(ctx.App.Writer, "Unknown Account Hashes\n----------------------------------------\n")
	if err != nil {
		return fmt.Errorf("could not write output; %s", err.Error())
	}

	// we want an iterator of all the known addresses
	ite := db.NewAccountIterator(ctx.Context)
	defer ite.Release()

	// iterate all known addresses
	var ah common.Hash
	var all, missing int

	verbose := ctx.Bool(flags.IsVerbose.Name)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()

	for ite.Next() {
		all++
		ah.SetBytes(ite.Key())

		_, err := db.HashToAccountAddress(ah)
		if err != nil {
			err = nil
			missing++

			// display unknown account hash
			if verbose {
				_, err = fmt.Fprintln(ctx.App.Writer, ah.String())
			}
		}

		// display progress in non-verbose mode
		select {
		case <-tick.C:
			if !verbose {
				_, err = fmt.Fprintf(ctx.App.Writer, "\rChecked: %10d  Missing: %10d", all, missing)
			}
		default:
		}

		// output error reached?
		if err != nil {
			return fmt.Errorf("could not finish scan; %s", err.Error())
		}
	}

	// out total
	_, err = fmt.Fprintf(ctx.App.Writer, "\r----------------------------------------\nAccounts Checked:%23d\nUnknown Hashes:%25d\n", all, missing)
	if err != nil {
		return fmt.Errorf("could not write output; %s", err.Error())
	}

	return nil
}

// listUnknownStorages implements unknown storages scan.
func listUnknownStorages(ctx *cli.Context) error {
	// try to open output DB
	db, err := snapshot.OpenStateDB(ctx.Path(flags.StateDBPath.Name))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(db)

	// out what we do
	_, err = fmt.Fprintf(ctx.App.Writer, "Unknown Storage Hashes\n----------------------------------------\n")
	if err != nil {
		return fmt.Errorf("could not write output; %s", err.Error())
	}

	// we want an iterator of all the known storages
	ite := db.NewAccountIterator(ctx.Context)
	defer ite.Release()

	// iterate all known addresses
	var all, storagesCount, missing uint64

	verbose := ctx.Bool(flags.IsVerbose.Name)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()

	for ite.Next() {
		all++
		acc := ite.Value()

		for h := range acc.Storage {
			storagesCount++
			_, err := db.HashToStorage(h)
			if err != nil {
				err = nil
				missing++

				// display unknown storage hash
				if verbose {
					_, err = fmt.Fprintln(ctx.App.Writer, h.String())
				}
			}
		}

		// display progress in non-verbose mode
		select {
		case <-tick.C:
			if !verbose {
				_, err = fmt.Fprintf(ctx.App.Writer, "\rChecked: %10d  Missing: %10d", storagesCount, missing)
			}
		default:
		}

		// output error reached?
		if err != nil {
			return fmt.Errorf("could not finish scan; %s", err.Error())
		}
	}

	// out total
	_, err = fmt.Fprintf(ctx.App.Writer, "\r----------------------------------------\nAccounts Checked:%23d\nStorages Checked:%23d\nUnknown Storage Hashes:%25d\n", all, storagesCount, missing)
	if err != nil {
		return fmt.Errorf("could not write output; %s", err.Error())
	}

	return nil
}

// accountImport implements entry point for account addresses import from given CSV file.
// The file is expected to contain only account addresses in hex format, non-address lines are ignored.
func accountImport(ctx *cli.Context) error {
	// check if we have a CSV file path
	if ctx.Args().Len() < 1 {
		return fmt.Errorf("please provide path to accounts list, use single dash for stdin")
	}

	// we prep the reader
	var re io.Reader
	var err error

	// where do we the address data?
	switch ctx.Args().Get(0) {
	case "-":
		// standard input pipe
		re, err = stdinReader()
	default:
		// a given CSV file
		re, err = os.Open(ctx.Args().Get(0))
	}

	// any error open the input file?
	if err != nil {
		return err
	}

	// try to open output DB
	db, err := snapshot.OpenStateDB(ctx.Path(flags.StateDBPath.Name))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(db)

	return importCsv(ctx.App.Writer, re, db)
}

// stdinReader opens standard input for reading, if possible.
func stdinReader() (io.Reader, error) {
	// get the standard input stats
	info, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}

	// check if we have the right access mode
	if info.Mode()&os.ModeCharDevice != 0 || info.Size() <= 0 {
		return nil, fmt.Errorf("please provide valid account address pipe")
	}

	return bufio.NewReader(os.Stdin), nil
}

// importCsv imports addresses or storages mapping from the given reader.
func importCsv(w io.Writer, r io.Reader, db *snapshot.StateDB) error {
	scan := bufio.NewScanner(r)
	scan.Split(bufio.ScanLines)

	var countAcc, countStorage int
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()

	// read all the lines
	for scan.Scan() {
		text := strings.TrimSpace(scan.Text())
		// skip non-address lines
		if common.IsHexAddress(text) {
			adr := common.HexToAddress(text)
			ha := db.AccountAddressToHash(adr)

			err := db.PutHashToAccountAddress(ha, adr)
			if err != nil {
				return err
			}
			countAcc++
		} else if isHash(text) {
			s := common.HexToHash(text)
			ha := db.StorageToHash(s)

			err := db.PutHashToStorage(ha, s)
			if err != nil {
				return err
			}
			countStorage++
		}

		select {
		case <-tick.C:
			// print progress
			_, err := fmt.Fprintf(w, "\rImported:%10d accounts, %10d storages.", countAcc, countStorage)
			if err != nil {
				return fmt.Errorf("could not write output; %s", err.Error())
			}
		default:
		}
	}

	// print total result
	_, err := fmt.Fprintf(w, "\rImport finished, %d accounts loaded, %d storages loaded.\n", countAcc, countStorage)
	if err != nil {
		return fmt.Errorf("could not write output; %s", err.Error())
	}

	return nil
}

func isHash(s string) bool {
	if has0xPrefix(s) {
		s = s[2:]
	}
	return len(s) == 2*common.HashLength && isHex(s)
}

// has0xPrefix validates str begins with '0x' or '0X'.
func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

// isHexCharacter returns bool of c being a valid hexadecimal.
func isHexCharacter(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

// isHex validates whether each byte is valid hexadecimal string.
func isHex(str string) bool {
	if len(str)%2 != 0 {
		return false
	}
	for _, c := range []byte(str) {
		if !isHexCharacter(c) {
			return false
		}
	}
	return true
}
