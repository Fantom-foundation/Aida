package profile

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

// GetKeyStatsCommand computes usage statistics of accessed storage locations
var GetKeyStatsCommand = cli.Command{
	Action:    getKeyStatsAction,
	Name:      "key-stats",
	Usage:     "computes usage statistics of accessed storage locations",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.WorkersFlag,
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The aida-profile key-stats command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to be analysed.

Statistics on the usage of accessed storage locations are printed to the console.
`,
}

// getKeyStatsAction collects statistical information on the usage
// of keys (=addresses of storage locations) in transactions.
func getKeyStatsAction(ctx *cli.Context) error {
	return getReferenceStatsActionWithConsumer(ctx, "key-stats", func(info *TransactionInfo) []common.Hash {
		keys := []common.Hash{}
		for _, account := range info.st.InputSubstate {
			for key := range account.Storage {
				keys = append(keys, common.Hash(key))
			}
		}
		for _, account := range info.st.InputSubstate {
			for key := range account.Storage {
				keys = append(keys, common.Hash(key))
			}
		}
		return keys
	}, printKeyValueDistribution)
}

func printKeyValueDistribution(stats *AccessStatistics[common.Hash]) {

	counts := [common.HashLength + 1]int64{}
	accesses := [common.HashLength + 1]int64{}
	for key, value := range stats.accesses {
		length := getLength(&key)
		counts[length]++
		accesses[length] += int64(value)
	}
	fmt.Printf("Key length distribution:\n")
	for i, c := range counts {
		fmt.Printf("%d, %d, %d\n", i, c, accesses[i])
	}
	fmt.Printf("------------------------\n")
}

func getLength(h *common.Hash) int {
	res := common.HashLength
	for res > 0 && h[common.HashLength-res] == 0 {
		res--
	}
	return res
}
