package txcontext

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Result interface {
	GetReceipt() Receipt
	GetRawResult() ([]byte, error)
	GetGasUsed() uint64
}

// Receipt represents an interface for managing and retrieving the result of a blockchain transaction or contract execution.
type Receipt interface {
	// GetStatus returns the status code indicating the success or failure of the transaction or execution.
	GetStatus() uint64

	// GetBloom returns the Bloom filter associated with the transaction or execution result.
	GetBloom() types.Bloom

	// GetLogs returns the logs generated during the transaction or contract execution.
	GetLogs() []*types.Log

	// GetContractAddress returns the address of the contract created, if any.
	GetContractAddress() common.Address

	// GetGasUsed returns the amount of gas used during the transaction or contract execution.
	GetGasUsed() uint64

	// Equal checks if the current result is equal to the provided result.
	// Note: Have a look at ReceiptEqual.
	Equal(y Receipt) bool
}

func ReceiptEqual(x, y Receipt) bool {
	if x == y {
		return true
	}

	if (x == nil || y == nil) && x != y {
		return false
	}

	rLogs := x.GetLogs()
	yLogs := y.GetLogs()

	equal := x.GetStatus() == y.GetStatus() &&
		x.GetBloom() == y.GetBloom() &&
		(len(rLogs)) == len(yLogs) &&
		x.GetContractAddress() == y.GetContractAddress() &&
		x.GetGasUsed() == y.GetGasUsed()
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
