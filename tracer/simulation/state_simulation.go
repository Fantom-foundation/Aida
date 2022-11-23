package simulation

import (
	"fmt"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"math/big"
	"math/rand"
)

// StateContext wraps current state transition of the simulation
type StateContext struct {
	distContract    *StochasticGenerator // Probabilistic distribution used to generate next address
	distStorage     *StochasticGenerator // Probabilistic distribution used to generate next storage
	distValue       *StochasticGenerator // Probabilistic distribution used to generate next value
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
func NewStateContext(distContract *StochasticGenerator, distStorage *StochasticGenerator, distValue *StochasticGenerator, t [][]float64, dCtx *dict.DictionaryContext) (StateContext, error) {
	sc := StateContext{
		snapshotCounter: 0,
		distContract:    distContract,
		distStorage:     distStorage,
		distValue:       distValue,
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

// initOpDictionary initializes dictionary with all operation creation functions
func (sc *StateContext) initOpDictionary() error {
	sc.opDict[operation.AddBalanceID] = func(sc *StateContext) operation.Operation {
		idx := sc.getNextAddress()
		nw := sc.getNextBalance()

		cur, ok := sc.balances[idx]
		if !ok {
			sc.balances[idx] = nw
		} else {
			v := new(big.Int).Add(cur, nw)
			if v.IsUint64() {
				// new value fits into uint64
				sc.balances[idx] = v
			} else {
				// prevent value from overflowing - by adding just zero
				nw.SetUint64(0)
			}
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
			nw := big.NewInt(0)
			//nw := sc.getNextBalance()
			cur, ok := sc.balances[idx]
			if !ok {
				// No balance left can't sub anything
				nw.SetUint64(0)
			} else {
				n := new(big.Int).Sub(cur, nw)

				if n.Sign() == -1 {
					// generated sub value was too big, reduce to current value for result to be zero
					nw = nw.Set(cur)
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
func (sc *StateContext) getNextBalance() *big.Int {
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
	r := sc.distContract.GetNext(sc.opId)[0]
	v, ok := r.(uint32)
	if ok {
		return v
	}
	return uint32(r.(uint64))
}

// getNextKey generates a new key using the current random probabilistic distribution
func (sc *StateContext) getNextKey() uint32 {
	r := sc.distStorage.GetNext(sc.opId)

	if len(r) < 2 {
		return uint32(r[0].(uint64))
	}
	i := r[0].(uint32)
	pos := r[1].(int)
	sc.usedPositions[pos] = true
	return i
}

// getNextCode generates next code
func (sc *StateContext) getNextCode() uint32 {
	// TODO make code generation more realistic
	buffer := make([]byte, 64)
	rand.Read(buffer)

	idxC := sc.dCtx.EncodeCode(buffer)
	return idxC
}

// NextOperation calculates next operation from current operation
func (sc *StateContext) NextOperation() operation.Operation {
	sc.opId = sc.getNextOp(sc.opId)
	if sc.opId > operation.NumProfiledOperations-1 {
		return nil
	}
	op := sc.encodeIntoOperation()
	return op
}

// encodeIntoOperation creates new operation instance from given opId
func (sc *StateContext) encodeIntoOperation() operation.Operation {
	opF, ok := sc.opDict[sc.opId]
	if !ok {
		return nil
	}
	return opF(sc)
}

// getNextOp returns next operation while skipping the skip operations and appropriating chance for the rest of the operations
func (sc *StateContext) getNextOp(op byte) byte {
	skipOps := sc.toBeSkipped()

	var modifier = 0.0
	for _, skip := range skipOps {
		modifier += sc.transitions[op][skip]
	}

	// error if skipped operation is 100% chance
	// rounding error - shouldn't matter since last return of function will return same error
	if modifier == 1 {
		return byte(dict.BYTE_MAX)
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
	return byte(dict.BYTE_MAX)
}

// toBeSkipped skipping operations which have to be skipped because their conditions aren't met
func (sc *StateContext) toBeSkipped() []int {
	var skipOps []int
	// preventing RevertToSnapshotID being called when no snapshot is available
	if sc.snapshotCounter == 0 {
		skipOps = append(skipOps, operation.RevertToSnapshotID)
	}

	// preventing GetStateLclsID being called when StorageIndexCache is empty
	// StorageIndexCache uses top always last value is at index 0
	// we need to only check the 0 index to see whether Cache isn't empty
	_, err := sc.dCtx.StorageIndexCache.Get(0)
	if err != nil {
		skipOps = append(skipOps, operation.GetStateLclsID)
		skipOps = append(skipOps, operation.GetStateLccsID)
	}

	return skipOps
}
