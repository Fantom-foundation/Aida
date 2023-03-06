package utils

import (
	"testing"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/ethereum/go-ethereum/core/vm"
)

// TestVmImplsAreRegistered checks if interpreters are correctly registered
func TestVmImplsAreRegistered(t *testing.T) {
	checkedImpls := []string{"lfvm", "lfvm-si", "geth"}

	statedb := state.MakeGethInMemoryStateDB(nil, 0)
	defer statedb.Close()
	chainConfig := GetChainConfig(0xFA)

	for _, interpreterImpl := range checkedImpls {
		evm := vm.NewEVM(vm.BlockContext{}, vm.TxContext{}, statedb, chainConfig, vm.Config{
			InterpreterImpl: interpreterImpl,
		})
		if evm == nil {
			t.Errorf("Unable to create EVM with InterpreterImpl %s", interpreterImpl)
		}
	}
}
