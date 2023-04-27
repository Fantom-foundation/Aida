package db

import (
	"fmt"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/utils"
	"github.com/syndtr/goleveldb/leveldb"
	leveldb_opt "github.com/syndtr/goleveldb/leveldb/opt"
	leveldb_util "github.com/syndtr/goleveldb/leveldb/util"
	"github.com/urfave/cli/v2"
)

var CompactCommand = cli.Command{
	Action:    compact,
	Name:      "compact",
	Usage:     "Compat LevelDB - discarding deleted and overwritten versions",
	ArgsUsage: "<dbPath>",
	Flags: []cli.Flag{
		&utils.LogLevel,
	},
	Description: `
The substate-cli db compact command requires one argument:
	<dbPath>
<dbPath> is the target LevelDB instance to compact.`,
}

func compact(ctx *cli.Context) error {
	var err error
	if ctx.Args().Len() != 1 {
		return fmt.Errorf("substate-cli db compact: command requires exactly one arguments")
	}

	log := utils.NewLogger(ctx.String(utils.LogLevel.Name), "Substate DB")

	dbPath := ctx.Args().Get(0)
	dbOpt := &leveldb_opt.Options{
		BlockCacheCapacity:     1 * leveldb_opt.GiB,
		OpenFilesCacheCapacity: 50,

		ErrorIfMissing: true,
		ReadOnly:       false,
	}
	db, err := leveldb.OpenFile(dbPath, dbOpt)
	if err != nil {
		return fmt.Errorf("substate-cli db compact: error opening dbPath %s: %v", dbPath, err)
	}

	start := time.Now()
	log.Notice("Compaction begin")
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err = db.CompactRange(leveldb_util.Range{})
		if err != nil {
			panic(fmt.Errorf("substate-cli db compact: error compacting dbPath %s: %v", dbPath, err))
		}
		wg.Done()
	}()
	wg.Wait()
	duration := time.Since(start)
	log.Notice("Compaction completed")
	log.Infof("Elapsed time: %v", duration.Round(1*time.Millisecond))

	return nil
}
