package rpc

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/txcontext"
)

type result struct {
	gasUsed uint64
	result  []byte
	err     error
}

func NewResult(res []byte, err error, gasUsed uint64) txcontext.Result {
	return &result{
		gasUsed: gasUsed,
		result:  res,
		err:     err,
	}
}

func (r *result) GetReceipt() txcontext.Receipt {
	// unused for rpc
	return nil
}

func (r *result) GetRawResult() ([]byte, error) {
	return r.result, r.err
}

func (r *result) GetGasUsed() uint64 {
	return r.gasUsed
}

func (r *result) String() string {
	return fmt.Sprintf("Result: %v\nError: %v\n; Gas Used: %v", string(r.result), r.err, r.gasUsed)
}
