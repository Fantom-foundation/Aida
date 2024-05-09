// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package stochastic

import (
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/stochastic/statistics"
	"github.com/Fantom-foundation/Aida/utils"
)

// GenerateUniformRegistry produces a uniformly distributed simulation file.
func GenerateUniformRegistry(cfg *utils.Config, log logger.Logger) *EventRegistry {
	r := NewEventRegistry()

	// generate a uniform distribution for contracts, storage keys/values, and snapshots

	log.Infof("Number of contract addresses for priming: %v", cfg.ContractNumber)
	for i := int64(0); i < cfg.ContractNumber; i++ {
		for j := i - statistics.QueueLen - 1; j <= i; j++ {
			if j >= 0 {
				r.contracts.Place(toAddress(j))
			}
		}
	}

	log.Infof("Number of storage keys for priming: %v", cfg.KeysNumber)
	for i := int64(0); i < cfg.KeysNumber; i++ {
		for j := i - statistics.QueueLen - 1; j <= i; j++ {
			if j >= 0 {
				r.keys.Place(toHash(j))
			}
		}
	}

	log.Infof("Number of storage values for priming: %v", cfg.ValuesNumber)
	for i := int64(0); i < cfg.ValuesNumber; i++ {
		for j := i - statistics.QueueLen - 1; j <= i; j++ {
			if j >= 0 {
				r.values.Place(toHash(j))
			}
		}
	}

	log.Infof("Snapshot depth: %v", cfg.KeysNumber)
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
