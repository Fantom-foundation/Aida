package db

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

var ExtractEthereumGenesisCommand = cli.Command{
	Action: extractEthereumGenesis,
	Name:   "extract-ethereum-genesis",
	Usage:  "Extracts substateAlloc from json into first updateset",
	Flags: []cli.Flag{
		&utils.ChainIDFlag,
		&utils.UpdateDbFlag,
		&logger.LogLevelFlag,
	},
	Description: `
Extracts substateAlloc from ethereum genesis.json into first updateset.`}

func extractEthereumGenesis(ctx *cli.Context) error {
	// process arguments and flags
	if ctx.Args().Len() != 1 {
		return fmt.Errorf("ethereum-update command requires exactly 1 arguments")
	}
	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}
	log := logger.NewLogger(cfg.LogLevel, "Ethereum Update")

	log.Notice("Load Ethereum initial world state")
	ws, err := loadEthereumGenesisWorldState(ctx.Args().Get(0))
	if err != nil {
		return err
	}

	udb, err := substate.OpenUpdateDB(cfg.UpdateDb)
	if err != nil {
		return err
	}
	defer udb.Close()

	log.Noticef("PutUpdateSet(0, %v, []common.Address{})", ws)
	udb.PutUpdateSet(0, &ws, []common.Address{})

	return nil
}

// loadEthereumGenesisWorldState loads opera initial world state from worldstate-db as SubstateAlloc
func loadEthereumGenesisWorldState(genesis string) (substate.SubstateAlloc, error) {
	var jsData map[string]interface{}
	// Read the content of the JSON file
	jsonData, err := ioutil.ReadFile(genesis)
	if err != nil {
		return nil, fmt.Errorf("failed to read genesis file: %v", err)
	}

	// Unmarshal JSON data
	if err := json.Unmarshal(jsonData, &jsData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal genesis file: %v", err)
	}

	// get field alloc
	alloc, ok := jsData["alloc"]
	if !ok {
		return nil, fmt.Errorf("failed to get alloc field from genesis file")
	}

	ssAccounts := make(substate.SubstateAlloc)

	// loop over all the accounts
	for k, v := range alloc.(map[string]interface{}) {
		// Convert the string key back to a common.Address
		address := common.HexToAddress(k)

		balance, _ := new(big.Int).SetString(v.(map[string]interface{})["balance"].(string), 10)
		ssAccounts[address] = substate.NewSubstateAccount(0, balance, []byte{})
	}

	return ssAccounts, err
}