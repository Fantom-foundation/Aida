package transaction

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Result represents an interface for managing and retrieving the result of a blockchain transaction or contract execution.
type Result interface {
	// GetStatus returns the status code indicating the success or failure of the transaction or execution.
	GetStatus() uint64

	// SetStatus sets the status code indicating the success or failure of the transaction or execution.
	SetStatus(status uint64)

	// GetBloom returns the Bloom filter associated with the transaction or execution result.
	GetBloom() types.Bloom

	// SetBloom sets the Bloom filter associated with the transaction or execution result.
	SetBloom(bloom types.Bloom)

	// GetLogs returns the logs generated during the transaction or contract execution.
	GetLogs() []*types.Log

	// SetLogs sets the logs generated during the transaction or contract execution.
	SetLogs(logs []*types.Log)

	// GetContractAddress returns the address of the contract created, if any.
	GetContractAddress() common.Address

	// SetContractAddress sets the address of the contract created, if any.
	SetContractAddress(addr common.Address)

	// GetGasUsed returns the amount of gas used during the transaction or contract execution.
	GetGasUsed() uint64

	// SetGasUsed sets the amount of gas used during the transaction or contract execution.
	SetGasUsed(gasUsed uint64)

	// Equal checks if the current result is equal to the provided result.
	// Note: Have a look at resultEqual.
	Equal(y Result) bool
}

type resultEqual struct{}

func (r *resultEqual) GetStatus() uint64 { return 0 }

func (r *resultEqual) SetStatus(uint64) {}

func (r *resultEqual) GetBloom() types.Bloom { return types.Bloom{} }

func (r *resultEqual) SetBloom(types.Bloom) {}

func (r *resultEqual) GetLogs() []*types.Log { return nil }

func (r *resultEqual) SetLogs([]*types.Log) {}

func (r *resultEqual) GetContractAddress() common.Address { return common.Address{} }

func (r *resultEqual) SetContractAddress(common.Address) {}

func (r *resultEqual) GetGasUsed() uint64 { return 0 }

func (r *resultEqual) SetGasUsed(uint64) {}

func (r *resultEqual) Equal(y Result) bool {
	if r == y {
		return true
	}

	if (r == nil || y == nil) && r != y {
		return false
	}

	rLogs := r.GetLogs()
	yLogs := y.GetLogs()

	equal := r.GetStatus() == y.GetStatus() &&
		r.GetBloom() == y.GetBloom() &&
		(len(rLogs)) == len(yLogs) &&
		r.GetContractAddress() == y.GetContractAddress() &&
		r.GetGasUsed() == y.GetGasUsed()
	if !equal {
		return false
	}

	for i, log := range rLogs {
		yLog := yLogs[i]

		equal := log.Address == yLog.Address &&
			len(log.Topics) == len(yLog.Topics) &&
			bytes.Equal(log.Data, yLog.Data)
		if !equal {
			return false
		}

		for i, xt := range log.Topics {
			yt := yLog.Topics[i]
			if xt != yt {
				return false
			}
		}
	}

	return true
}
