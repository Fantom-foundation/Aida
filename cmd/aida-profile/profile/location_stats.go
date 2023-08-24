package profile

import (
	"sync"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

// GetLocationStatsCommand computes usage statistics of accessed storage locations
var GetLocationStatsCommand = cli.Command{
	Action:    getLocationStatsAction,
	Name:      "location-stats",
	Usage:     "computes usage statistics of accessed storage locations",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&substate.WorkersFlag,
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The aida-profile location-stats command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to be analysed.

Statistics on the usage of accessed storage locations are printed to the console.
`,
}

type Index[T comparable] struct {
	index map[T]int
	mu    sync.Mutex
}

func (i *Index[T]) Get(value *T) int {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.index == nil {
		i.index = map[T]int{}
	}
	v, present := i.index[*value]
	if present {
		return v
	}
	v = len(i.index)
	i.index[*value] = v
	return v
}

type Location struct {
	address_id int
	key_id     int
}

// getLocationStatsAction collects statistical information on the usage
// of storage locations identified by a contracts address and the memory
// location key.
func getLocationStatsAction(ctx *cli.Context) error {
	var address_index Index[common.Address]
	var key_index Index[common.Hash]
	return getReferenceStatsAction(ctx, "location-stats", func(info *TransactionInfo) []Location {
		locations := []Location{}
		for address, account := range info.st.InputAlloc {
			address_id := address_index.Get(&address)
			for key := range account.Storage {
				key_id := key_index.Get(&key)
				locations = append(locations, Location{address_id, key_id})
			}
		}
		for address, account := range info.st.OutputAlloc {
			address_id := address_index.Get(&address)
			for key := range account.Storage {
				key_id := key_index.Get(&key)
				locations = append(locations, Location{address_id, key_id})
			}
		}
		return locations
	})
}
