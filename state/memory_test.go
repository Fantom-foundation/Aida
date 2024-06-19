package state

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestInMemoryDb_SelfDestruct6780OnlyDeletesContractsCreatedInSameTransaction(t *testing.T) {
	a := common.Address{1}
	b := common.Address{2}

	db := MakeInMemoryStateDB(nil, 12)
	db.CreateContract(a)

	if want, got := false, db.HasSelfDestructed(a); want != got {
		t.Errorf("invalid self-destruct state of contract %x, want %v, got %v", a, want, got)
	}
	if want, got := false, db.HasSelfDestructed(b); want != got {
		t.Errorf("invalid self-destruct state of contract %x, want %v, got %v", b, want, got)
	}

	db.Selfdestruct6780(a) // < this should work

	if want, got := true, db.HasSelfDestructed(a); want != got {
		t.Errorf("invalid self-destruct state of contract %x, want %v, got %v", a, want, got)
	}
	if want, got := false, db.HasSelfDestructed(b); want != got {
		t.Errorf("invalid self-destruct state of contract %x, want %v, got %v", b, want, got)
	}

	db.Selfdestruct6780(b) // < this should be ignored

	if want, got := true, db.HasSelfDestructed(a); want != got {
		t.Errorf("invalid self-destruct state of contract %x, want %v, got %v", a, want, got)
	}
	if want, got := false, db.HasSelfDestructed(b); want != got {
		t.Errorf("invalid self-destruct state of contract %x, want %v, got %v", b, want, got)
	}
}
