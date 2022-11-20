package simulation

import (
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Carmen/go/common"
	"math/big"
	"math/rand"
	"time"
)

var KeysCacheSize = 256

// StateContext wraps current state transition of the simulation
type StateContext struct {
	address      common.Address // Current account address
	key          common.Key     // Current contract slot address
	value        common.Value   // Last returned slot value
	snapshot     int            // Last returned snapshot
	balance      *big.Int       // Last returned account balance
	nonce        uint64         // Last returned account nonce
	distContract Generator      // Probabilistic distribution used to generate next address
	distStorage  Generator      // Probabilistic distribution used to generate next storage
	distValue    Generator      // Probabilistic distribution used to generate next value
	usedKeys     []common.Key   // A cache of recently used contract slot keys
	dCtx         *dict.DictionaryContext
	transitions  [][]float64
	opDict       map[byte]func() operation.Operation
	currentBlock uint64
	opId         byte
}

// NewStateContext creates a new context, which contains current state of Transitions
func NewStateContext(distContract Generator, distStorage Generator, distValue Generator, t [][]float64, dCtx *dict.DictionaryContext) (StateContext, error) {
	rand.Seed(time.Now().UnixNano())
	sc := StateContext{
		address:      common.Address{},
		key:          common.Key{},
		value:        common.Value{},
		snapshot:     0,
		balance:      &big.Int{},
		nonce:        0,
		distContract: distContract,
		distStorage:  distStorage,
		distValue:    distValue,
		usedKeys:     make([]common.Key, 0, KeysCacheSize),
		dCtx:         dCtx,
		transitions:  t,
		currentBlock: 0,
		opId:         operation.EndBlockID,
		opDict:       make(map[byte]func() operation.Operation, operation.NumProfiledOperations),
	}

	err := sc.initOpDictionary()
	if err != nil {
		return StateContext{}, err
	}

	return sc, nil
}

func (sc *StateContext) initOpDictionary() error {
	sc.opDict[operation.AddBalanceID] = func() operation.Operation {
		return operation.NewAddBalance(sc.getNextAddress(), sc.getNextBalance())
	}
	sc.opDict[operation.BeginBlockID] = func() operation.Operation {
		{
			sc.currentBlock++
			return operation.NewBeginBlock(sc.currentBlock)
		}
	}
	sc.opDict[operation.CreateAccountID] = func() operation.Operation {
		{
			return operation.NewCreateAccount(sc.getNextAddress())
		}
	}
	sc.opDict[operation.EmptyID] = func() operation.Operation {
		{
			return operation.NewEmpty(sc.getNextAddress())
		}
	}
	sc.opDict[operation.EndBlockID] = func() operation.Operation {
		{
			return operation.NewEndBlock()
		}
	}
	sc.opDict[operation.EndTransactionID] = func() operation.Operation {
		{
			return operation.NewEndTransaction()
		}
	}
	sc.opDict[operation.ExistID] = func() operation.Operation {
		{
			return operation.NewExist(sc.getNextAddress())
		}
	}
	sc.opDict[operation.FinaliseID] = func() operation.Operation {
		{
			//TODO solve deleteEmptyObjects
			return operation.NewFinalise(false)
		}
	}
	sc.opDict[operation.GetBalanceID] = func() operation.Operation {
		{
			return operation.NewGetBalance(sc.getNextAddress())
		}
	}
	sc.opDict[operation.GetCodeHashID] = func() operation.Operation {
		{
			return operation.NewGetCodeHash(sc.getNextAddress())
		}
	}
	sc.opDict[operation.GetCodeHashLcID] = func() operation.Operation {
		{
			return operation.NewGetCodeHashLc()
		}
	}
	sc.opDict[operation.GetCodeID] = func() operation.Operation {
		{
			return operation.NewGetCode(sc.getNextAddress())
		}
	}
	sc.opDict[operation.GetCodeSizeID] = func() operation.Operation {
		{
			return operation.NewGetCodeSize(sc.getNextAddress())
		}
	}
	sc.opDict[operation.GetCommittedStateID] = func() operation.Operation {
		{
			return operation.NewGetCommittedState(sc.getNextAddress(), sc.getNextKey())
		}
	}
	sc.opDict[operation.GetCommittedStateLclsID] = func() operation.Operation {
		{
			return operation.NewGetCommittedStateLcls()
		}
	}
	sc.opDict[operation.GetNonceID] = func() operation.Operation {
		{
			return operation.NewGetNonce(sc.getNextAddress())
		}
	}
	sc.opDict[operation.GetStateID] = func() operation.Operation {
		{
			return operation.NewGetState(sc.getNextAddress(), sc.getNextKey())
		}
	}
	sc.opDict[operation.GetStateLcID] = func() operation.Operation {
		{
			return operation.NewGetStateLc(sc.getNextKey())
		}
	}
	sc.opDict[operation.GetStateLccsID] = func() operation.Operation {
		{
			// TODO pos cache
			return operation.NewGetStateLccs(0)
		}
	}
	sc.opDict[operation.GetStateLclsID] = func() operation.Operation {
		{
			return operation.NewGetStateLcls()
		}
	}
	sc.opDict[operation.HasSuicidedID] = func() operation.Operation {
		{
			return operation.NewHasSuicided(sc.getNextAddress())
		}
	}
	sc.opDict[operation.RevertToSnapshotID] = func() operation.Operation {
		{
			// TODO revert to snapshotID
			return operation.NewRevertToSnapshot(0)
		}
	}
	sc.opDict[operation.SetCodeID] = func() operation.Operation {
		{
			return operation.NewSetCode(sc.getNextAddress(), sc.getNextCode())
		}
	}
	sc.opDict[operation.SetNonceID] = func() operation.Operation {
		{
			idx := sc.getNextAddress()
			return operation.NewSetNonce(idx, sc.getNextNonce(idx))
		}
	}
	sc.opDict[operation.SetStateID] = func() operation.Operation {
		{
			return operation.NewSetState(sc.getNextAddress(), sc.getNextKey(), sc.getNextValue())
		}
	}
	sc.opDict[operation.SetStateLclsID] = func() operation.Operation {
		{
			return operation.NewSetStateLcls(sc.getNextValue())
		}
	}
	sc.opDict[operation.SnapshotID] = func() operation.Operation {
		{
			// TODO use snapshotID
			return operation.NewSnapshot(rand.Int31())
		}
	}
	sc.opDict[operation.SubBalanceID] = func() operation.Operation {
		{
			return operation.NewSubBalance(sc.getNextAddress(), sc.getNextBalance())
		}
	}
	sc.opDict[operation.SuicideID] = func() operation.Operation {
		{
			return operation.NewSuicide(sc.getNextAddress())
		}
	}

	if len(sc.opDict) != operation.NumProfiledOperations {
		return fmt.Errorf("incompatible number of profiled operations")
	}
	return nil
}

// getNextNonce generates a new nonce using the current random probabilistic distribution
func (sc *StateContext) getNextNonce(idx uint32) uint64 {
	// TODO set realistic nonce in sequence
	//return uint64(sc.distribution.GetNext())
	return rand.Uint64()
}

// getNextBalance generates a new balance using the current random probabilistic distribution
func (sc *StateContext) getNextBalance() *big.Int {
	//nextVal := sc.distribution.GetNext()
	//balance.SetInt64(int64(nextVal))
	var balance = new(big.Int)
	balance.SetInt64(rand.Int63())
	return balance
}

// getNextValue generates a new value using the current random probabilistic distribution
func (sc *StateContext) getNextValue() uint64 {
	return (sc.distValue.GetNext(sc.opId)[0]).(uint64)
}

// getNextAddress generates a new address using the current random probabilistic distribution
func (sc *StateContext) getNextAddress() uint32 {
	return (sc.distContract.GetNext(sc.opId)[0]).(uint32)
}

// getNextKey generates a new key using the current random probabilistic distribution
func (sc *StateContext) getNextKey() uint32 {
	r := sc.distContract.GetNext(sc.opId)
	i := r[0].(uint32)
	//pos := r[1].(int)
	//TODO put pos into cache

	//TODO usedKeys cache for GetStateLccs
	//TODO maybe return key or retrieve it by idx from dict
	//if len(sc.usedKeys) < KeysCacheSize {
	//	sc.usedKeys = append(sc.usedKeys, key)
	//}
	return i
}

// getNextCode
func (sc *StateContext) getNextCode() uint32 {
	// TODO make code generation more realistic
	buffer := make([]byte, 32)
	rand.Read(buffer)

	idxC := sc.dCtx.EncodeCode(buffer)
	return idxC
}

// getUsedKey assigns a new key using one of the already used keys selected by a uniform random probabilistic distribution
func (sc *StateContext) getUsedKey() (key common.Key) {
	if len(sc.usedKeys) == 1 {
		key = sc.usedKeys[0]
	}
	if len(sc.usedKeys) > 1 {
		next := rand.Intn(len(sc.usedKeys))
		key = sc.usedKeys[next]
	}
	return
}

func (sc *StateContext) NextOperation() operation.Operation {
	sc.opId = sc.getNextOp(sc.opId)
	if sc.opId > operation.NumProfiledOperations-1 {
		return nil
	}
	op := sc.encodeIntoOperation()
	return op
}

func (sc *StateContext) encodeIntoOperation() operation.Operation {
	opF, ok := sc.opDict[sc.opId]
	if !ok {
		return nil
	}
	return opF()
}

func (sc *StateContext) getNextOp(op byte) byte {
	// determine next state
	p := rand.Float64()

	sum := 0.0

	for i := 0; i < operation.NumProfiledOperations; i++ {
		sum += sc.transitions[i][op]
		if p <= sum {
			return byte(i)
		}
	}
	return byte(255)
}
