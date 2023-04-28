package state

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/cmd/worldstate-cli/flags"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/world-state/db/opera"
	"github.com/urfave/cli/v2"
)

// CmdRoot retrieves root hash for given block number
var CmdRoot = cli.Command{
	Action:      root,
	Name:        "root",
	Aliases:     []string{"r"},
	Usage:       "Retrieve root hash of given block number.",
	Description: `Searches opera database for root hash for supplied block number.`,
	ArgsUsage:   "<target>",
	Flags: []cli.Flag{
		&flags.SourceDBType,
		&flags.SourceDBPath,
		&flags.SourceTableName,
		&flags.TargetBlock,
	},
}

// root retrieves root hash of given block number
func root(ctx *cli.Context) error {
	// open the source trie DB
	store, err := opera.Connect(ctx.String(flags.SourceDBType.Name), ctx.Path(flags.SourceDBPath.Name), ctx.Path(flags.SourceTableName.Name))
	if err != nil {
		return err
	}
	defer opera.MustCloseStore(store)

	// make logger
	log := utils.NewLogger(ctx.String(utils.LogLevelFlag.Name), "root")

	targetBlock := ctx.Uint64(flags.TargetBlock.Name)

	if targetBlock == 0 {
		err = fmt.Errorf("supplied target block can't be %d", targetBlock)
		log.Error(err)
		return err
	}

	//look up root hash from block number
	rootHash, err := opera.RootByBlockNumber(store, targetBlock)
	if err != nil {
		log.Errorf("unable to find root hash for block number %d; %s", targetBlock, err.Error())
		return err
	}

	log.Infof("block %d has root hash %s", targetBlock, rootHash)
	log.Info("done")
	return nil
}
