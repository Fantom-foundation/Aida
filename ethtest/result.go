package ethtest

import (
	"errors"

	"github.com/Fantom-foundation/Aida/txcontext"
)

type stateTestResult struct {
	expectedErr string
}

func (s stateTestResult) GetReceipt() txcontext.Receipt {
	return nil
}

func (s stateTestResult) GetRawResult() ([]byte, error) {
	if s.expectedErr == "" {
		return nil, nil
	}
	return nil, errors.New(s.expectedErr)
}

func (s stateTestResult) GetGasUsed() uint64 {
	return 0
}

func (s stateTestResult) String() string {
	return ""
}
