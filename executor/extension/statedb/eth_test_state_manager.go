package statedb

import (
	"bytes"
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
)

func MakeTemporaryEthStateManager(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	return makeTemporaryEthStateManager(logger.NewLogger(cfg.LogLevel, "EthStatePrepper"), cfg)
}

func makeTemporaryEthStateManager(log logger.Logger, cfg *utils.Config) *ethStateManager {
	return &ethStateManager{
		cfg: cfg,
		log: log,
	}
}

type ethStateManager struct {
	extension.NilExtension[txcontext.TxContext]
	cfg *utils.Config
	log logger.Logger
}

func (e *ethStateManager) PreTransaction(st executor.State[txcontext.TxContext], ctx *executor.Context) error {
	var err error

	ctx.State, ctx.StateDbPath, err = utils.PrepareStateDB(e.cfg)
	if err != nil {
		return fmt.Errorf("failed to prepare statedb; %v", err)
	}

	primeCtx := utils.NewPrimeContext(e.cfg, ctx.State, e.log)

	err = primeCtx.PrimeStateDB(st.Data.GetInputState(), ctx.State)
	if err != nil {
		return err
	}

	err = e.validate(st.Data.GetInputState(), ctx.State)
	if err != nil {
		return fmt.Errorf("pre alloc validation failed; %v", err)
	}

	ctx.State.BeginBlock(st.Data.GetBlockEnvironment().GetNumber())

	return nil
}

func (e *ethStateManager) validate(alloc txcontext.WorldState, db state.StateDB) error {

	var err error
	switch e.cfg.StateValidationMode {
	case utils.SubsetCheck:
		err = doSubsetValidation(alloc, db, e.cfg.UpdateOnFailure)
	case utils.EqualityCheck:
		vmAlloc := db.GetSubstatePostAlloc()
		isEqual := alloc.Equal(vmAlloc)
		if !isEqual {
			err = fmt.Errorf("inconsistent output: alloc")
			//v.printAllocationDiffSummary(&expectedAlloc, &vmAlloc)
		}
	}

	return err
}

func (e *ethStateManager) PostTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	defer os.RemoveAll(ctx.StateDbPath)

	ctx.State.EndBlock()

	want := state.Data.GetStateHash()
	got := ctx.State.GetHash()

	// cast state.Data to stJSON
	c := state.Data.(*ethtest.StJSON)

	if got != want {
		err := fmt.Errorf("%v - (%v) FAIL\ndifferent hashes\ngot: %v\nwant:%v", c.TestLabel, c.UsedNetwork, got.Hex(), want.Hex())
		if e.cfg.ContinueOnFailure {
			e.log.Error(err)
		} else {
			return err
		}
	} else {
		e.log.Noticef("%v - (%v) PASS\nblock: %v; tx: %v\nhash:%v", c.TestLabel, c.UsedNetwork, state.Block, state.Transaction, got.Hex())
	}

	return nil
}

// doSubsetValidation validates whether the given alloc is contained in the db object.
// NB: We can only check what must be in the db (but cannot check whether db stores more).
func doSubsetValidation(alloc txcontext.WorldState, db state.VmStateDB, updateOnFail bool) error {
	var err string

	alloc.ForEachAccount(func(addr common.Address, acc txcontext.Account) {
		if !db.Exist(addr) {
			err += fmt.Sprintf("  Account %v does not exist\n", addr.Hex())
			if updateOnFail {
				db.CreateAccount(addr)
			}
		}
		accBalance := acc.GetBalance()

		if balance := db.GetBalance(addr); accBalance.Cmp(balance) != 0 {
			err += fmt.Sprintf("  Failed to validate balance for account %v\n"+
				"    have %v\n"+
				"    want %v\n",
				addr.Hex(), balance, accBalance)
			if updateOnFail {
				db.SubBalance(addr, balance)
				db.AddBalance(addr, accBalance)
			}
		}
		if nonce := db.GetNonce(addr); nonce != acc.GetNonce() {
			err += fmt.Sprintf("  Failed to validate nonce for account %v\n"+
				"    have %v\n"+
				"    want %v\n",
				addr.Hex(), nonce, acc.GetNonce())
			if updateOnFail {
				db.SetNonce(addr, acc.GetNonce())
			}
		}
		if code := db.GetCode(addr); bytes.Compare(code, acc.GetCode()) != 0 {
			err += fmt.Sprintf("  Failed to validate code for account %v\n"+
				"    have len %v\n"+
				"    want len %v\n",
				addr.Hex(), len(code), len(acc.GetCode()))
			if updateOnFail {
				db.SetCode(addr, acc.GetCode())
			}
		}

		// validate Storage
		acc.ForEachStorage(func(keyHash common.Hash, valueHash common.Hash) {
			if db.GetState(addr, keyHash) != valueHash {
				err += fmt.Sprintf("  Failed to validate storage for account %v, key %v\n"+
					"    have %v\n"+
					"    want %v\n",
					addr.Hex(), keyHash.Hex(), db.GetState(addr, keyHash).Hex(), valueHash.Hex())
				if updateOnFail {
					db.SetState(addr, keyHash, valueHash)
				}
			}
		})

	})

	if len(err) > 0 {
		return fmt.Errorf(err)
	}
	return nil
}
