package statedb

import (
	"bytes"
	"fmt"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core"
)

func NewTemporaryEthStatePrepper(cfg *utils.Config) executor.Extension[*ethtest.Data] {
	return &ethStatePrepper{
		cfg: cfg,
		log: logger.NewLogger(cfg.LogLevel, "EthStatePrepper"),
	}
}

type ethStatePrepper struct {
	extension.NilExtension[*ethtest.Data]
	cfg                           *utils.Config
	log                           logger.Logger
	failedPre, failedPost, passed uint64
	lastPre                       bool
}

// PreRun primes the state db with Pre Alloc

func (e *ethStatePrepper) PreTransaction(st executor.State[*ethtest.Data], ctx *executor.Context) error {

	primeCtx := utils.NewPrimeContext(e.cfg, ctx.State, e.log)

	alloc := make(substate.SubstateAlloc)

	for addr, acc := range st.Data.Pre {
		alloc[addr] = substate.NewSubstateAccount(acc.Nonce, acc.Balance, acc.Code)
		for k, v := range acc.Storage {
			alloc[addr].Storage[k] = v
		}
	}

	err := primeCtx.PrimeStateDB(alloc, ctx.State)
	if err != nil {
		return err
	}

	err = e.validate(st.Data.Pre, ctx.State)
	if err != nil {
		e.lastPre = false

		//return fmt.Errorf("pre alloc validation failed; %v", err)
	}

	e.lastPre = true

	return nil
}

func (e *ethStatePrepper) validate(genesisAlloc core.GenesisAlloc, db state.StateDB) error {
	alloc := make(substate.SubstateAlloc)

	for addr, acc := range genesisAlloc {
		alloc[addr] = substate.NewSubstateAccount(acc.Nonce, acc.Balance, acc.Code)
		for k, v := range acc.Storage {
			alloc[addr].Storage[k] = v
		}
	}

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

func (e *ethStatePrepper) PostTransaction(state executor.State[*ethtest.Data], ctx *executor.Context) error {
	err := e.validate(state.Data.Post, ctx.State)
	if err != nil {
		if e.lastPre {
			e.failedPost++
		} else {
			e.failedPre++
		}

		return nil
		//return fmt.Errorf("post alloc validation failed;\n%v", err)
	}

	if e.lastPre && state.Data.Post != nil {
		e.passed++
	}

	return nil
}

// doSubsetValidation validates whether the given alloc is contained in the db object.
// NB: We can only check what must be in the db (but cannot check whether db stores more).
func doSubsetValidation(alloc substate.SubstateAlloc, db state.VmStateDB, updateOnFail bool) error {
	var err string
	for addr, account := range alloc {
		if !db.Exist(addr) {
			err += fmt.Sprintf("  Account %v does not exist\n", addr.Hex())
			if updateOnFail {
				db.CreateAccount(addr)
			}
		}
		if balance := db.GetBalance(addr); account.Balance.Cmp(balance) != 0 {
			err += fmt.Sprintf("  Failed to validate balance for account %v\n"+
				"    have %v\n"+
				"    want %v\n",
				addr.Hex(), balance, account.Balance)
			if updateOnFail {
				db.SubBalance(addr, balance)
				db.AddBalance(addr, account.Balance)
			}
		}
		if nonce := db.GetNonce(addr); nonce != account.Nonce {
			err += fmt.Sprintf("  Failed to validate nonce for account %v\n"+
				"    have %v\n"+
				"    want %v\n",
				addr.Hex(), nonce, account.Nonce)
			if updateOnFail {
				db.SetNonce(addr, account.Nonce)
			}
		}
		if code := db.GetCode(addr); bytes.Compare(code, account.Code) != 0 {
			err += fmt.Sprintf("  Failed to validate code for account %v\n"+
				"    have len %v\n"+
				"    want len %v\n",
				addr.Hex(), len(code), len(account.Code))
			if updateOnFail {
				db.SetCode(addr, account.Code)
			}
		}
		for key, value := range account.Storage {
			if db.GetState(addr, key) != value {
				err += fmt.Sprintf("  Failed to validate storage for account %v, key %v\n"+
					"    have %v\n"+
					"    want %v\n",
					addr.Hex(), key.Hex(), db.GetState(addr, key).Hex(), value.Hex())
				if updateOnFail {
					db.SetState(addr, key, value)
				}
			}
		}
	}
	if len(err) > 0 {
		return fmt.Errorf(err)
	}
	return nil
}
