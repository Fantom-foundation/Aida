package stochastic

import (
	"log"

	"github.com/Fantom-foundation/Aida/stochastic/statistics"
	"github.com/Fantom-foundation/Aida/utils"
)

// GenerateUniformRegistry produces a uniformly distributed simulation file.
func GenerateUniformRegistry(cfg *utils.Config) *EventRegistry {
	r := NewEventRegistry()

	// generate a uniform distribution for contracts, storage keys/values, and snapshots

	log.Printf("number of contract addresses for priming: %v\n", cfg.ContractNumber)
	for i := int64(0); i < cfg.ContractNumber; i++ {
		for j := i - statistics.QueueLen - 1; j <= i; j++ {
			if j >= 0 {
				r.contracts.Place(toAddress(j))
			}
		}
	}

	log.Printf("number of storage keys for priming: %v\n", cfg.KeysNumber)
	for i := int64(0); i < cfg.KeysNumber; i++ {
		for j := i - statistics.QueueLen - 1; j <= i; j++ {
			if j >= 0 {
				r.keys.Place(toHash(j))
			}
		}
	}

	log.Printf("number of storage values for priming: %v\n", cfg.ValuesNumber)
	for i := int64(0); i < cfg.ValuesNumber; i++ {
		for j := i - statistics.QueueLen - 1; j <= i; j++ {
			if j >= 0 {
				r.values.Place(toHash(j))
			}
		}
	}

	log.Printf("snapshot depth: %v\n", cfg.KeysNumber)
	for i := 0; i < cfg.SnapshotDepth; i++ {
		r.snapshotFreq[i] = 1
	}

	for i := 0; i < numArgOps; i++ {
		if IsValidArgOp(i) {
			r.argOpFreq[i] = 1 // set frequency to greater than zero to emit operation
			opI, _, _, _ := DecodeArgOp(i)
			switch opI {
			case BeginSyncPeriodID:
				j := EncodeArgOp(BeginBlockID, statistics.NoArgID, statistics.NoArgID, statistics.NoArgID)
				r.transitFreq[i][j] = 1
			case BeginBlockID:
				j := EncodeArgOp(BeginTransactionID, statistics.NoArgID, statistics.NoArgID, statistics.NoArgID)
				r.transitFreq[i][j] = 1
			case EndTransactionID:
				j1 := EncodeArgOp(BeginTransactionID, statistics.NoArgID, statistics.NoArgID, statistics.NoArgID)
				j2 := EncodeArgOp(EndBlockID, statistics.NoArgID, statistics.NoArgID, statistics.NoArgID)
				r.transitFreq[i][j1] = cfg.BlockLength - 1
				r.transitFreq[i][j2] = 1
			case EndBlockID:
				j1 := EncodeArgOp(BeginBlockID, statistics.NoArgID, statistics.NoArgID, statistics.NoArgID)
				j2 := EncodeArgOp(EndSyncPeriodID, statistics.NoArgID, statistics.NoArgID, statistics.NoArgID)
				r.transitFreq[i][j1] = cfg.SyncPeriodLength - 1
				r.transitFreq[i][j2] = 1
			case EndSyncPeriodID:
				j := EncodeArgOp(BeginSyncPeriodID, statistics.NoArgID, statistics.NoArgID, statistics.NoArgID)
				r.transitFreq[i][j] = 1
			default:
				for j := 0; j < numArgOps; j++ {
					if IsValidArgOp(j) {
						opJ, _, _, _ := DecodeArgOp(j)
						if opJ != BeginSyncPeriodID &&
							opJ != BeginBlockID &&
							opJ != BeginTransactionID &&
							opJ != EndTransactionID &&
							opJ != EndBlockID &&
							opJ != EndSyncPeriodID {
							r.transitFreq[i][j] = cfg.TransactionLength - 1
						} else if opJ == EndTransactionID {
							r.transitFreq[i][j] = 1
						}
					}
				}
			}
		}
	}
	return &r
}
