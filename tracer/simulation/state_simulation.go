package simulation

import (
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Carmen/go/common"
	"math/big"
	"math/rand"
)

var KeysCacheSize = 256

// StateContext wraps current state transition of the simulation
type StateContext struct {
	address         common.Address      // Current account address
	key             common.Key          // Current contract slot address
	value           common.Value        // Last returned slot value
	balance         *big.Int            // Last returned account balance
	nonce           uint64              // Last returned account nonce
	usedKeys        []common.Key        // A cache of recently used contract slot keys
	distContract    StochasticGenerator // Probabilistic distribution used to generate next address
	distStorage     StochasticGenerator // Probabilistic distribution used to generate next storage
	distValue       StochasticGenerator // Probabilistic distribution used to generate next value
	dCtx            *dict.DictionaryContext
	transitions     [][]float64
	opDict          map[byte]func(sc *StateContext) operation.Operation
	currentBlock    uint64
	opId            byte
	nonces          map[uint32]uint64
	balances        map[uint32]*big.Int
	snapshotCounter int32 // Last returned snapshotCounter
	totalSnap       uint32
	usedPositions   map[int]bool
}

// NewStateContext creates a new context, which contains current state of Transitions
func NewStateContext(distContract StochasticGenerator, distStorage StochasticGenerator, distValue StochasticGenerator, t [][]float64, dCtx *dict.DictionaryContext) (StateContext, error) {
	sc := StateContext{
		address:         common.Address{},
		key:             common.Key{},
		value:           common.Value{},
		snapshotCounter: 0,
		balance:         &big.Int{},
		distContract:    distContract,
		distStorage:     distStorage,
		distValue:       distValue,
		usedKeys:        make([]common.Key, 0, KeysCacheSize),
		dCtx:            dCtx,
		transitions:     t,
		currentBlock:    0,
		opId:            operation.EndBlockID,
		opDict:          make(map[byte]func(sc *StateContext) operation.Operation, operation.NumProfiledOperations),
		nonces:          make(map[uint32]uint64),
		balances:        make(map[uint32]*big.Int),
		totalSnap:       0,
		usedPositions:   make(map[int]bool, 256),
	}

	err := sc.initOpDictionary()
	if err != nil {
		return StateContext{}, err
	}

	return sc, nil
}

func (sc *StateContext) initOpDictionary() error {
	sc.opDict[operation.AddBalanceID] = func(sc *StateContext) operation.Operation {
		idx := sc.getNextAddress()
		nw := sc.getNextBalance(idx)
		cur, ok := sc.balances[idx]
		if !ok {
			sc.balances[idx] = nw
		} else {
			sc.balances[idx] = new(big.Int).Add(cur, nw)
		}
		return operation.NewAddBalance(idx, nw)
	}
	sc.opDict[operation.BeginBlockID] = func(sc *StateContext) operation.Operation {
		{
			sc.currentBlock++
			return operation.NewBeginBlock(sc.currentBlock)
		}
	}
	sc.opDict[operation.CreateAccountID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewCreateAccount(sc.getNextAddress())
		}
	}
	sc.opDict[operation.EmptyID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewEmpty(sc.getNextAddress())
		}
	}
	sc.opDict[operation.EndBlockID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewEndBlock()
		}
	}
	sc.opDict[operation.EndTransactionID] = func(sc *StateContext) operation.Operation {
		{
			sc.snapshotCounter = 0
			return operation.NewEndTransaction()
		}
	}
	sc.opDict[operation.ExistID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewExist(sc.getNextAddress())
		}
	}
	sc.opDict[operation.FinaliseID] = func(sc *StateContext) operation.Operation {
		{
			//TODO solve deleteEmptyObjects
			return operation.NewFinalise(false)
		}
	}
	sc.opDict[operation.GetBalanceID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewGetBalance(sc.getNextAddress())
		}
	}
	sc.opDict[operation.GetCodeHashID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewGetCodeHash(sc.getNextAddress())
		}
	}
	sc.opDict[operation.GetCodeHashLcID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewGetCodeHashLc()
		}
	}
	sc.opDict[operation.GetCodeID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewGetCode(sc.getNextAddress())
		}
	}
	sc.opDict[operation.GetCodeSizeID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewGetCodeSize(sc.getNextAddress())
		}
	}
	sc.opDict[operation.GetCommittedStateID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewGetCommittedState(sc.getNextAddress(), sc.getNextKey())
		}
	}
	sc.opDict[operation.GetCommittedStateLclsID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewGetCommittedStateLcls()
		}
	}
	sc.opDict[operation.GetNonceID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewGetNonce(sc.getNextAddress())
		}
	}
	sc.opDict[operation.GetStateID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewGetState(sc.getNextAddress(), sc.getNextKey())
		}
	}
	sc.opDict[operation.GetStateLcID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewGetStateLc(sc.getNextKey())
		}
	}
	sc.opDict[operation.GetStateLccsID] = func(sc *StateContext) operation.Operation {
		{

			////ctx.StorageIndexCache.Get(sPos) could be used aswell

			//var p int
			//if len(sc.usedPositions) == 0 {
			//	//TODO fix
			//	p = 0
			//} else {
			//	p = rand.Intn(len(sc.usedPositions))
			//}

			//return operation.NewGetStateLccs(p)

			return operation.NewGetStateLccs(0)
		}
	}
	sc.opDict[operation.GetStateLclsID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewGetStateLcls()
		}
	}
	sc.opDict[operation.HasSuicidedID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewHasSuicided(sc.getNextAddress())
		}
	}
	sc.opDict[operation.RevertToSnapshotID] = func(sc *StateContext) operation.Operation {
		{

			s := sc.snapshotCounter
			var id int
			if s > 0 {
				id = rand.Intn(int(s))
			}

			//update remaining usable snapshots count
			sc.snapshotCounter = int32(id)

			return operation.NewRevertToSnapshot(id)
		}
	}
	sc.opDict[operation.SetCodeID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewSetCode(sc.getNextAddress(), sc.getNextCode())
		}
	}
	sc.opDict[operation.SetNonceID] = func(sc *StateContext) operation.Operation {
		{
			idx := sc.getNextAddress()
			return operation.NewSetNonce(idx, sc.getNextNonce(idx))
		}
	}
	sc.opDict[operation.SetStateID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewSetState(sc.getNextAddress(), sc.getNextKey(), sc.getNextValue())
		}
	}
	sc.opDict[operation.SetStateLclsID] = func(sc *StateContext) operation.Operation {
		{
			return operation.NewSetStateLcls(sc.getNextValue())
		}
	}
	sc.opDict[operation.SnapshotID] = func(sc *StateContext) operation.Operation {
		{
			op := operation.NewSnapshot(sc.snapshotCounter)
			sc.snapshotCounter++
			return op
		}
	}
	sc.opDict[operation.SubBalanceID] = func(sc *StateContext) operation.Operation {
		{
			idx := sc.getNextAddress()
			nw := sc.getNextBalance(idx)
			cur, ok := sc.balances[idx]
			if !ok {
				// No balance left can't sub anything
				nw.SetUint64(0)
			} else {
				n := new(big.Int).Sub(cur, nw)
				if n.Sign() == -1 {
					// generated sub value was too big, reduce to current value for result to be zero
					nw = cur
					n.SetUint64(0)
				}
				sc.balances[idx] = n
			}
			return operation.NewSubBalance(idx, nw)
		}
	}
	sc.opDict[operation.SuicideID] = func(sc *StateContext) operation.Operation {
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
	sc.nonces[idx]++
	return sc.nonces[idx]
}

// getNextBalance generates a new balance using the current random probabilistic distribution
func (sc *StateContext) getNextBalance(idx uint32) *big.Int {
	var balance = new(big.Int)
	v := rand.Uint64()
	balance.SetUint64(v)
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
	r := sc.distStorage.GetNext(sc.opId)
	i := r[0].(uint32)
	pos := r[1].(int)
	sc.usedPositions[pos] = true

	return i
}

// getNextCode
func (sc *StateContext) getNextCode() uint32 {
	// TODO make code generation more realistic
	buffer := make([]byte, 64)
	rand.Read(buffer)

	idxC := sc.dCtx.EncodeCode(buffer)
	return idxC
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
	return opF(sc)
}

func (sc *StateContext) getNextOp(op byte) byte {
	// determine next state
	p := rand.Float64()

	sum := 0.0

	for i := 0; i < operation.NumProfiledOperations; i++ {
		sum += sc.transitions[op][i]
		if p <= sum {
			// preventing RevertToSnapshotID being called when no snapshot is available
			if i == operation.RevertToSnapshotID && sc.snapshotCounter == 0 {
				return sc.getNextOpSkip(op, i)
			}

			return byte(i)
		}
	}
	return byte(255)
}

// getNextOpSkip returns operation while skipping the skip operation and appropriating chance for the rest of the operations
func (sc *StateContext) getNextOpSkip(op byte, skip int) byte {
	modifier := sc.transitions[op][skip]
	if modifier == 0 {
		// could return getNextOp - but there might be deadlock
		return byte(255)
	}

	//error if skipped operation is 100% chance
	if modifier == 1 {
		return byte(255)
	}

	// determine next state
	// modified max probability maximum to one without skipped operation
	p := rand.Float64() * (1 - modifier)

	sum := 0.0

	for i := 0; i < operation.NumProfiledOperations; i++ {
		sum += sc.transitions[op][i]
		if p <= sum {
			return byte(i)
		}
	}
	return byte(255)
}
