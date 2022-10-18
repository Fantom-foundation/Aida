package snapshot

import (
	"math/rand"
	"testing"
	"time"
)

func TestStateDB_PutGetBlock(t *testing.T) {
	db, _, _ := makeTestDB(t)

	rand.Seed(time.Now().UnixNano())
	bln := uint64(rand.Int63())

	// try to store
	err := db.PutBlockNumber(bln)
	if err != nil {
		t.Fatalf("failed block put; expected to store %d, got error %s", bln, err.Error())
	}

	// try to read back
	found, err := db.GetBlockNumber()
	if err != nil {
		t.Errorf("failed block get; expected to load %d, got error %s", bln, err.Error())
	}

	if found != bln {
		t.Errorf("failed block get; expected to load %d, got %d", bln, found)
	}
}

func TestStateDB_Account(t *testing.T) {
	db, nodes, addr := makeTestDB(t)

	for h, a := range addr {
		ac, err := db.Account(a)
		if err != nil {
			t.Errorf("failed account get; expected to load %s, got error %s", a.String(), err.Error())
		}

		bc, ok := nodes[h]
		if !ok {
			t.Errorf("failed account check; expected to load %s, account not found", a.String())
		}

		if !ac.IsIdentical(&bc) {
			err = ac.IsDifferent(&bc)
			t.Errorf("failed account check; expected to load identical account %s, got %s", a.String(), err.Error())
		}
	}
}
