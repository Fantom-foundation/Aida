package state

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
)

func TestStateDB_AccountsAreImplicitlyCreated(t *testing.T) {
	address := common.Address{0x1}
	change := map[string]func(StateDB){
		"set zero balance": func(state StateDB) {
			state.AddBalance(address, uint256.NewInt(0), tracing.BalanceChangeTouchAccount)
		},
		"set non-zero balance": func(state StateDB) {
			state.AddBalance(address, uint256.NewInt(1), tracing.BalanceChangeTouchAccount)
		},
		"set zero nonce": func(state StateDB) {
			state.SetNonce(address, 0)
		},
		"set non-zero nonce": func(state StateDB) {
			state.SetNonce(address, 1)
		},
		"set empty code": func(state StateDB) {
			state.SetCode(address, nil)
		},
		"set code": func(state StateDB) {
			state.SetCode(address, []byte{0x1})
		},
	}

	for name, change := range change {
		t.Run(name, func(t *testing.T) {
			testForAllImplementations(t, func(t *testing.T, state StateDB) {
				if err := state.BeginTransaction(0); err != nil {
					t.Fatalf("failed to begin transaction: %v", err)
				}
				if state.Exist(address) {
					t.Errorf("the initial state should not contain any accounts")
				}
				change(state)
				if !state.Exist(address) {
					t.Errorf("the account should be created implicitly")
				}
				if err := state.EndTransaction(); err != nil {
					t.Fatalf("failed to end transaction: %v", err)
				}
			})
		})
	}
}

func TestStateDB_TouchedEmptyAccountsAreImplicitlyDestroyedAtEndOfTransaction(t *testing.T) {
	address := common.Address{0x1}
	change := map[string]func(StateDB){
		"set zero balance": func(state StateDB) {
			state.AddBalance(address, uint256.NewInt(0), tracing.BalanceChangeTouchAccount)
		},
		"set zero nonce": func(state StateDB) {
			state.SetNonce(address, 0)
		},
		"set empty code": func(state StateDB) {
			state.SetCode(address, nil)
		},
		"non-zero and reverted balance": func(state StateDB) {
			state.AddBalance(address, uint256.NewInt(1), tracing.BalanceChangeTouchAccount)
			state.SubBalance(address, uint256.NewInt(1), tracing.BalanceChangeTouchAccount)
		},
	}

	for name, change := range change {
		t.Run(name, func(t *testing.T) {
			testForAllImplementations(t, func(t *testing.T, state StateDB) {
				if err := state.BeginTransaction(0); err != nil {
					t.Fatalf("failed to begin transaction: %v", err)
				}
				change(state)
				if !state.Exist(address) {
					t.Errorf("the account should be created implicitly")
				}
				if err := state.EndTransaction(); err != nil {
					t.Fatalf("failed to end transaction: %v", err)
				}
				if err := state.BeginTransaction(1); err != nil {
					t.Fatalf("failed to begin transaction: %v", err)
				}
				if state.Exist(address) {
					t.Errorf("the account should be destroyed implicitly")
				}
				if err := state.EndTransaction(); err != nil {
					t.Fatalf("failed to end transaction: %v", err)
				}
			})
		})
	}
}

func TestStateDB_RevertsDeleteImplicitlyCreatedAccounts(t *testing.T) {
	testForAllImplementations(t, func(t *testing.T, state StateDB) {
		address := common.Address{0x1}
		if err := state.BeginTransaction(0); err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}
		if state.Exist(address) {
			t.Errorf("the initial state should not contain any accounts")
		}

		s := state.Snapshot()
		state.SetNonce(address, 1)
		if !state.Exist(address) {
			t.Errorf("the account should be created implicitly")
		}

		state.RevertToSnapshot(s)
		if state.Exist(address) {
			t.Errorf("a revert should delete implicitly created accounts")
		}
		if err := state.EndTransaction(); err != nil {
			t.Fatalf("failed to end transaction: %v", err)
		}
	})
}

func TestStateDB_SelfDestructedAccountsExistTillEndOfTransaction(t *testing.T) {
	testForAllImplementations(t, func(t *testing.T, state StateDB) {
		address := common.Address{0x1}
		if err := state.BeginTransaction(0); err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}

		if state.Exist(address) {
			t.Errorf("the initial state should not contain any accounts")
		}

		state.SetNonce(address, 1)

		if !state.Exist(address) {
			t.Errorf("the account should be created implicitly")
		}

		state.SelfDestruct(address)

		if !state.Exist(address) {
			t.Errorf("the account should exist until the end of the transaction")
		}

		state.SetNonce(address, 2)

		if !state.Exist(address) {
			t.Errorf("the account should exist until the end of the transaction")
		}

		if err := state.EndTransaction(); err != nil {
			t.Fatalf("failed to end transaction: %v", err)
		}

		if err := state.BeginTransaction(1); err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}

		if state.Exist(address) {
			t.Errorf("the self-destructed account should be deleted at the end of the transaction")
		}

		if err := state.EndTransaction(); err != nil {
			t.Fatalf("failed to end transaction: %v", err)
		}
	})
}

func TestStateDB_SelfDestruct6780CanDeleteExplicitlyCreatedAccountInSameTransaction(t *testing.T) {
	testForAllImplementations(t, func(t *testing.T, state StateDB) {
		address := common.Address{0x1}
		if err := state.BeginTransaction(0); err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}

		if state.Exist(address) {
			t.Errorf("the initial state should not contain any accounts")
		}

		state.CreateAccount(address)

		if !state.Exist(address) {
			t.Errorf("the explicitly created account should exist")
		}

		state.Selfdestruct6780(address)

		if !state.Exist(address) {
			t.Errorf("the account should exist until the end of the transaction")
		}

		if err := state.EndTransaction(); err != nil {
			t.Fatalf("failed to end transaction: %v", err)
		}

		if err := state.BeginTransaction(1); err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}

		if state.Exist(address) {
			t.Errorf("the self-destructed account should be deleted at the end of the transaction")
		}

		if err := state.EndTransaction(); err != nil {
			t.Fatalf("failed to end transaction: %v", err)
		}
	})
}

func TestStateDB_SelfDestruct6780CanNotDeletePreExistingAccounts(t *testing.T) {
	testForAllImplementations(t, func(t *testing.T, state StateDB) {
		address := common.Address{0x1}
		if err := state.BeginTransaction(0); err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}

		if state.Exist(address) {
			t.Errorf("the initial state should not contain any accounts")
		}

		state.CreateAccount(address)
		state.SetNonce(address, 1)

		if !state.Exist(address) {
			t.Errorf("the explicitly created account should exist")
		}

		if err := state.EndTransaction(); err != nil {
			t.Fatalf("failed to end transaction: %v", err)
		}

		if err := state.BeginTransaction(1); err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}

		if !state.Exist(address) {
			t.Errorf("the account created in the previous transaction should still exist")
		}

		state.Selfdestruct6780(address)

		if !state.Exist(address) {
			t.Errorf("the account should exist until the end of the transaction")
		}

		if err := state.EndTransaction(); err != nil {
			t.Fatalf("failed to end transaction: %v", err)
		}

		if err := state.BeginTransaction(2); err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}

		if !state.Exist(address) {
			t.Errorf("the self-destructed of a pre-existing account in transaction 1 should not be carried out")
		}

		if err := state.EndTransaction(); err != nil {
			t.Fatalf("failed to end transaction: %v", err)
		}
	})
}

func testForAllImplementations(t *testing.T, test func(t *testing.T, state StateDB)) {
	impls := map[string]func(*testing.T) (StateDB, error){
		"geth": func(t *testing.T) (StateDB, error) {
			return MakeGethStateDB(t.TempDir(), "", common.Hash{}, false, nil)
		},
		"carmen": func(t *testing.T) (StateDB, error) {
			return MakeCarmenStateDB(t.TempDir(), "go-file", 5, "none", 0, 0, 0, 0)
		},
		/* TODO: fix the in-memory state DB to work with the tests
		"memory": func(t *testing.T) (StateDB, error) {
			return MakeEmptyGethInMemoryStateDB("")
		},
		*/
	}

	for impl, make := range impls {
		t.Run(impl, func(t *testing.T) {
			db, err := make(t)
			if err != nil {
				t.Fatalf("failed to create DB: %v", err)
			}
			defer func() {
				if err := db.Close(); err != nil {
					t.Fatalf("failed to close DB: %v", err)
				}
			}()

			if err := db.BeginBlock(1); err != nil {
				t.Fatalf("failed to begin block: %v", err)
			}

			test(t, db)

			if err := db.EndBlock(); err != nil {
				t.Fatalf("failed to end block: %v", err)
			}
		})
	}
}
