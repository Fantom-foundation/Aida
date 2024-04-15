// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package profile

import (
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
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
		&substate.WorkersFlag,
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
		for address := range info.st.InputAlloc {
			addresses = append(addresses, address)
		}
		for address := range info.st.OutputAlloc {
			addresses = append(addresses, address)
		}
		return addresses
	})
}
