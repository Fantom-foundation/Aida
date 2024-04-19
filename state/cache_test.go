package state

import (
	"math/big"
	"testing"

	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	cc "github.com/Fantom-foundation/Carmen/go/common"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

var (
	testCode = []byte{1}
	testAddr = common.Address{0x1}
	testWs   = substate.SubstateAlloc{testAddr: substate.NewSubstateAccount(1, big.NewInt(1), testCode)}
)

func Test_InMemoryDbUsesCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	cache := NewMockCodeCache(ctrl)
	want := common.Hash(cc.Keccak256(testCode))

	cache.EXPECT().Get(testAddr, testCode).Return(want)

	db := MakeInMemoryStateDB(substatecontext.NewWorldState(testWs), 1, cache)

	got := db.GetCodeHash(testAddr)
	if got != want {
		t.Fatalf("unexpected code hash\n got: %v\n want: %v", got.String(), want.String())
	}
}

func Test_OffTheChainMemoryDbUsesCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	cache := NewMockCodeCache(ctrl)
	want := common.Hash(cc.Keccak256(testCode))

	cache.EXPECT().Get(testAddr, testCode).Return(want)

	db, err := MakeOffTheChainStateDB(substatecontext.NewWorldState(testWs), 1, nil, cache)
	if err != nil {
		t.Fatalf("cannot make off-the-chain-db; %v", err)
	}

	got := db.GetCodeHash(testAddr)
	if got != want {
		t.Fatalf("unexpected code hash\n got: %v\n want: %v", got.String(), want.String())
	}
}
