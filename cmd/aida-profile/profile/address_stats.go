package profile

import (
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

// GetAddressStatsCommand computes usage statistics of addresses
var GetAddressStatsCommand = cli.Command{
	Action:    getAddressStatsAction,
	Name:      "address-stats",
	Usage:     "computes usage statistics of addresses",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.WorkersFlag,
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
	},
	Description: `
The aida-profile address-stats command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to be analysed.

Statistics on the usage of addresses are printed to the console.
`,
}

// getAddressStatsAction collects statistical information on the usage
// of addresses in transactions.
func getAddressStatsAction(ctx *cli.Context) error {
	return getReferenceStatsAction(ctx, "address-stats", func(info *TransactionInfo) []common.Address {
		addresses := []common.Address{}
		for address := range info.st.InputSubstate {
			addresses = append(addresses, common.Address(address))
		}
		for address := range info.st.OutputSubstate {
			addresses = append(addresses, common.Address(address))
		}
		return addresses
	})
}
