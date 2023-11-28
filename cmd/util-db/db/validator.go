package db

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/Fantom-foundation/Aida/cmd/util-db/flags"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utildb"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/urfave/cli/v2"
)

var GenerateDbHashCommand = cli.Command{
	Action: generateDbHashCmd,
	Name:   "generate-db-hash",
	Usage:  "Generates new db-hash. Note that this will overwrite the current AidaDb hash.",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
}

var ValidateCommand = cli.Command{
	Action: validateCmd,
	Name:   "validate",
	Usage:  "Validates AidaDb using md5 DbHash.",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
}

// validateCmd calculates the dbHash for given AidaDb and saves it.
func generateDbHashCmd(ctx *cli.Context) error {
	log := logger.NewLogger("INFO", "DbHashGenerateCMD")

	cfg, err := utils.NewConfig(ctx, utils.NoArgs)

	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open db; %v", err)
	}

	defer utildb.MustCloseDB(aidaDb)

	md := utils.NewAidaDbMetadata(aidaDb, "INFO")

	log.Noticef("Starting DbHash generation for %v; this may take several hours...", cfg.AidaDb)
	hash, err := utildb.GenerateDbHash(aidaDb, "INFO")
	if err != nil {
		return err
	}

	err = md.SetDbHash(hash)
	if err != nil {
		return fmt.Errorf("cannot set db-hash; %v", err)
	}

	return nil
}

// validateCmd calculates the dbHash for given AidaDb and compares it to expected hash either found in metadata or online
func validateCmd(ctx *cli.Context) error {
	log := logger.NewLogger("INFO", "ValidateCMD")

	cfg, err := utils.NewConfig(ctx, utils.NoArgs)

	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("cannot open db; %v", err)
	}

	defer utildb.MustCloseDB(aidaDb)

	md := utils.NewAidaDbMetadata(aidaDb, "INFO")

	md.ChainId = md.GetChainID()
	if md.ChainId == 0 {
		log.Warning("cannot find db-hash in your aida-db metadata, this operation is needed because db-hash was not found inside your aida-db; please make sure you specified correct chain-id with flag --%v", utils.ChainIDFlag.Name)
		md.ChainId = cfg.ChainID
	}

	// validation only makes sense if user has pure AidaDb
	dbType := md.GetDbType()
	if dbType != utils.GenType {
		return fmt.Errorf("validation cannot be performed - your db type (%v) cannot be validated; aborting", dbType)
	}

	// we need to make sure aida-db starts from beginning, otherwise validation is impossible
	// todo simplify condition once lachesis patch is ready for testnet
	md.FirstBlock = md.GetFirstBlock()
	if (md.ChainId == utils.MainnetChainID && md.FirstBlock != 0) || (md.ChainId == utils.TestnetChainID && md.FirstBlock != utildb.FirstOperaTestnetBlock) {
		return fmt.Errorf("validation cannot be performed - your db does not start at block 0; your first block: %v", md.FirstBlock)
	}

	var saveHash = false

	// if db hash is not present, look for it in patches.json
	expectedHash := md.GetDbHash()
	if len(expectedHash) == 0 {
		// we want to save the hash inside metadata
		saveHash = true
		expectedHash, err = utildb.FindDbHashOnline(md.ChainId, log, md)
		if err != nil {
			return fmt.Errorf("validation cannot be performed; %v", err)
		}
	}

	log.Noticef("Found DbHash for your Db: %v", hex.EncodeToString(expectedHash))

	log.Noticef("Starting DbHash calculation for %v; this may take several hours...", cfg.AidaDb)
	trueHash, err := utildb.GenerateDbHash(aidaDb, "INFO")
	if err != nil {
		return err
	}

	if bytes.Compare(expectedHash, trueHash) != 0 {
		return fmt.Errorf("hashes are different! expected: %v; your aida-db:%v", hex.EncodeToString(expectedHash), hex.EncodeToString(trueHash))
	}

	log.Noticef("Validation successful!")

	if saveHash {
		err = md.SetDbHash(trueHash)
		if err != nil {
			return err
		}
	}

	return nil
}

var PrintDbHashCommand = cli.Command{
	Action: printDbHash,
	Name:   "print-db-hash",
	Usage:  "Prints db-hash (md5) inside AidaDb. If this value is not present in metadata it iterates through all of data.",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&flags.ForceFlag,
	},
}

func printDbHash(ctx *cli.Context) error {
	var force = ctx.Bool(flags.ForceFlag.Name)

	aidaDb, err := rawdb.NewLevelDBDatabase(ctx.String(utils.AidaDbFlag.Name), 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("cannot open db; %v", err)
	}

	defer utildb.MustCloseDB(aidaDb)

	var dbHash []byte

	log := logger.NewLogger("INFO", "AidaDb-Db-Hash")

	md := utils.NewAidaDbMetadata(aidaDb, "INFO")

	// first try to extract from db
	dbHash = md.GetDbHash()
	if len(dbHash) != 0 && !force {
		log.Infof("Db-Hash (metadata): %v", hex.EncodeToString(dbHash))
		return nil
	}

	// if not found in db, we need to iterate and create the hash
	if dbHash, err = utildb.GenerateDbHash(aidaDb, "INFO"); err != nil {
		return err
	}

	fmt.Printf("Db-Hash (calculated): %v", hex.EncodeToString(dbHash))
	return nil
}

// PrintTableHashCommand calculates md5 of actual data stored
var PrintTableHashCommand = cli.Command{
	Action: printTableHash,
	Name:   "print-table-hash",
	Usage:  "Calculates md5 of decoded objects stored in AidaDb. Using []byte value from database, it decodes it and calculates md5 of the decoded objects.",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.DbComponentFlag,
		&logger.LogLevelFlag,
	},
	Description: `
Creates signatures of substates, updatesets, deletion and state-hashes using decoded objects from database rather than []byte value representation, because that is not deterministic.
`,
}

var PrintPrefixHashCommand = cli.Command{
	Action:    printPrefixHash,
	Name:      "print-prefix-hash",
	Usage:     "Prints state hash of db prefixes individually. Or just for single prefix specified in arg.",
	ArgsUsage: "<prefix>",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
}
