package executor

import (
	"errors"
	"testing"

	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

func TestNewTransactionResult_ExecutionResult_Error_IsAdded(t *testing.T) {
	injectedErr := errors.New("injected err")
	msg := &core.Message{To: &(common.Address{1})}
	result := &evmcore.ExecutionResult{Err: injectedErr}

	tr := newTransactionResult(nil, msg, result, nil, common.Address{2})

	if got, want := tr.err, injectedErr; !errors.Is(got, want) {
		t.Errorf("unexpected error, got: %v, want: %v", got, want)
	}
}
