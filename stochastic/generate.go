package stochastic

import "github.com/Fantom-foundation/Aida/stochastic/statistics"

// TODO: Convert constants to command-line interface parameters
const (
	TransactionsPerBlock = 10
	BlocksPerEpoch       = 10
	OperationFrequency   = 10 // determines indirectly the length of a transaction
	NumContracts         = 1000
	NumKeys              = 1000
	NumValues            = 1000
	SnapshotDepth        = 100
)

// GenerateUniformRegistry produces the a uniformly distributed simulation file.
func GenerateUniformRegistry() *EventRegistry {
	r := NewEventRegistry()

	// generate a uniform distribution for contracts, storage keys/values, and snapshots
	for i := int64(0); i < NumContracts; i++ {
		for j := i - statistics.QueueLen - 1; j <= i; j++ {
			if j >= 0 {
				r.contracts.Place(toAddress(j))
			}
		}
	}
	for i := int64(0); i < NumKeys; i++ {
		for j := i - statistics.QueueLen - 1; j <= i; j++ {
			if j >= 0 {
				r.keys.Place(toHash(j))
			}
		}
	}
	for i := int64(0); i < NumValues; i++ {
		for j := i - statistics.QueueLen - 1; j <= i; j++ {
			if j >= 0 {
				r.values.Place(toHash(j))
			}
		}
	}
	for i := 0; i < SnapshotDepth; i++ {
		r.snapshotFreq[i] = 1
	}

	for i := 0; i < numArgOps; i++ {
		if IsValidArgOp(i) {
			r.argOpFreq[i] = 1 // set frequency to greater than zero to emit operation
			opI, _, _, _ := decodeArgOp(i)
			switch opI {
			case BeginEpochID:
				j := EncodeArgOp(BeginBlockID, statistics.NoArgID, statistics.NoArgID, statistics.NoArgID)
				r.transitFreq[i][j] = 1
			case BeginBlockID:
				j := EncodeArgOp(BeginTransactionID, statistics.NoArgID, statistics.NoArgID, statistics.NoArgID)
				r.transitFreq[i][j] = 1
			case FinaliseID:
				j := EncodeArgOp(EndTransactionID, statistics.NoArgID, statistics.NoArgID, statistics.NoArgID)
				r.transitFreq[i][j] = 1
			case EndTransactionID:
				j1 := EncodeArgOp(BeginTransactionID, statistics.NoArgID, statistics.NoArgID, statistics.NoArgID)
				j2 := EncodeArgOp(EndBlockID, statistics.NoArgID, statistics.NoArgID, statistics.NoArgID)
				r.transitFreq[i][j1] = TransactionsPerBlock - 1
				r.transitFreq[i][j2] = 1
			case EndBlockID:
				j1 := EncodeArgOp(BeginBlockID, statistics.NoArgID, statistics.NoArgID, statistics.NoArgID)
				j2 := EncodeArgOp(EndEpochID, statistics.NoArgID, statistics.NoArgID, statistics.NoArgID)
				r.transitFreq[i][j1] = BlocksPerEpoch - 1
				r.transitFreq[i][j2] = 1
			case EndEpochID:
				j := EncodeArgOp(BeginEpochID, statistics.NoArgID, statistics.NoArgID, statistics.NoArgID)
				r.transitFreq[i][j] = 1
			default:
				for j := 0; j < numArgOps; j++ {
					if IsValidArgOp(j) {
						opJ, _, _, _ := decodeArgOp(j)
						if opJ != BeginEpochID &&
							opJ != BeginBlockID &&
							opJ != BeginTransactionID &&
							opJ != FinaliseID &&
							opJ != EndTransactionID &&
							opJ != EndBlockID &&
							opJ != EndEpochID {
							r.transitFreq[i][j] = OperationFrequency
						} else if opJ == FinaliseID {
							r.transitFreq[i][j] = 1
						}
					}
				}
			}
		}
	}
	return &r
}
