package statedb

import (
	"bytes"
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

func NewTemporaryEthStatePrepper(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	return &ethStatePrepper{
		cfg: cfg,
		log: logger.NewLogger(cfg.LogLevel, "EthStatePrepper"),
	}
}

type ethStatePrepper struct {
	extension.NilExtension[txcontext.TxContext]
	cfg                           *utils.Config
	log                           logger.Logger
	failedPre, failedPost, passed uint64
	lastPre                       bool
}

// PreRun primes the state db with pre Alloc
func (e *ethStatePrepper) PreTransaction(st executor.State[txcontext.TxContext], ctx *executor.Context) error {
	primeCtx := utils.NewPrimeContext(e.cfg, ctx.State, e.log)

	err := primeCtx.PrimeStateDB(st.Data.GetInputState(), ctx.State)
	if err != nil {
		return err
	}

	err = e.validate(st.Data.GetInputState(), ctx.State)
	if err != nil {
		e.lastPre = false

		//return fmt.Errorf("pre alloc validation failed; %v", err)
	}

	e.lastPre = true

	return nil
}

func (e *ethStatePrepper) validate(alloc txcontext.WorldState, db state.StateDB) error {
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
func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewLegacyKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

func (e *ethStatePrepper) PostTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	h := ctx.State.GetHash()
	fmt.Println(h.Hex())
	logs := ctx.ExecutionResult.GetLogs()
	fmt.Println(logs)
	//err := e.validate(state.Data.GetOutputState(), ctx.State)
	//if err != nil {
	//	if e.lastPre {
	//		e.failedPost++
	//	} else {
	//		e.failedPre++
	//	}
	//	return fmt.Errorf("post alloc validation failed;\n%v", err)
	//}

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
