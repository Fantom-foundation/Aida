package simulation

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/Fantom-foundation/Carmen/go/common"
	"github.com/Fantom-foundation/Carmen/go/state"
	"math/big"
	"math/rand"
	"time"
)

const KeysCacheSize = 256
const OpsChanSize = 1000

// Simulate executes simulation from StartBlock and runs the Markov Chain until EndBlock is reached
func Simulate(ctx context.Context, stateDB state.StateDB, dist common.Distribution, transitions Transitions, num uint) {
	// simulate one block processing
	ops := generateOperationsFactory(ctx, transitions, num)

	simulate(ctx, stateDB, dist, transitions, ops)
}

// simulate
func simulate(ctx context.Context, stateDB state.StateDB, dist common.Distribution, transitions Transitions, ops chan byte) {
	sc := newStateContext(stateDB, dist)

	// run Markov chain
	var previousState, currentState byte
	var steps uint
	var ok bool
	var blockNum = 1

	ctxDone := ctx.Done()
	for {
		select {
		case <-ctxDone:
			return
		case currentState, ok = <-ops:
			if !ok {
				return
			}
			steps++

			// execute current state
			sc.perform(transitions.ops[currentState])

			fmt.Printf("Block: %3.0d - Step: %3.0d. %25s -> %25s, address: %x, key: %x, value: %x, balance: %x, nonce: %x \n", blockNum, steps, transitions.labels[previousState], transitions.labels[currentState], sc.address, sc.key, sc.value, sc.balance, sc.nonce)

			if currentState == transitions.endBlockOpIdx {
				steps = 0
				blockNum++
			}

			previousState = currentState
		}
	}
}

func generateOperationsFactory(ctx context.Context, transitions Transitions, n uint) chan byte {
	ops := make(chan byte, OpsChanSize)
	go generateOperations(ctx, transitions, n, ops)
	return ops
}

func generateOperations(ctx context.Context, transitions Transitions, n uint, ops chan byte) {
	defer close(ops)

	opLen := byte(len(transitions.ops))
	ctxDone := ctx.Done()
	// initializing current state to 0 - BeginBlock
	currentState := transitions.beginBlockOpIdx
	ops <- currentState
	for {
		// determine next state
		p := rand.Float64()

		sum := 0.0

		var i byte = 0
		for ; i < opLen; i++ {
			sum += transitions.probabilities[i][currentState]
			if p <= sum {
				currentState = i

				select {
				case <-ctxDone:
					return
				case ops <- currentState:
				}
				break
			}
		}
		// operations for another block generated
		if currentState == transitions.endBlockOpIdx {
			n--
			if n == 0 {
				// desired number of blocks was generated
				break
			}
		}
	}
}

// stateContext wraps current state transition of the simulation
type stateContext struct {
	stateDB      state.StateDB       // StateDB used for simulation
	address      common.Address      // Current account address
	key          common.Key          // Current contract slot address
	value        common.Value        // Last returned slot value
	snapshot     int                 // Last returned snapshot
	balance      *big.Int            // Last returned account balance
	nonce        uint64              // Last returned account nonce
	distribution common.Distribution // Probabilistic distribution used to generate next address
	usedKeys     []common.Key        // A cache of recently used contract slot keys
}

// newStateContext creates a new context, which contains current state of Transitions
func newStateContext(stateDB state.StateDB, randDistribution common.Distribution) stateContext {
	rand.Seed(time.Now().UnixNano())
	return stateContext{
		stateDB:      stateDB,
		address:      common.Address{},
		key:          common.Key{},
		value:        common.Value{},
		snapshot:     0,
		balance:      &big.Int{},
		nonce:        0,
		distribution: randDistribution,
		usedKeys:     make([]common.Key, 0, KeysCacheSize),
	}
}

// getNextValue generates a new value using the current random probabilistic distribution
func (sc *stateContext) getNextValue() (value common.Value) {
	// TODO generate within the whole 20B address space
	nextVal := sc.distribution.GetNext()
	binary.BigEndian.PutUint32(value[:], nextVal)
	return value
}

// getNextNonce generates a new nonce using the current random probabilistic distribution
func (sc *stateContext) getNextNonce() uint64 {
	return uint64(sc.distribution.GetNext())
}

// getNextBalance generates a new balance using the current random probabilistic distribution
func (sc *stateContext) getNextBalance() (balance *big.Int) {
	// TODO generate within the whole 20B address space
	nextVal := sc.distribution.GetNext()
	balance.SetInt64(int64(nextVal))
	return balance
}

// getNextAddress generates a new address using the current random probabilistic distribution
func (sc *stateContext) getNextAddress() (address common.Address) {
	// TODO generate within the whole 20B address space
	nextVal := sc.distribution.GetNext()
	binary.BigEndian.PutUint32(address[:], nextVal)
	return address
}

// getNextKey generates a new key using the current random probabilistic distribution
func (sc *stateContext) getNextKey() (key common.Key) {
	// TODO generate within the whole 32B address space
	nextVal := sc.distribution.GetNext()
	binary.BigEndian.PutUint32(key[:], nextVal)
	if len(sc.usedKeys) < KeysCacheSize {
		sc.usedKeys = append(sc.usedKeys, key)
	}

	return key
}

// getUsedKey assigns a new key using one of the already used keys selected by a uniform random probabilistic distribution
func (sc *stateContext) getUsedKey() (key common.Key) {
	if len(sc.usedKeys) == 1 {
		key = sc.usedKeys[0]
	}
	if len(sc.usedKeys) > 1 {
		next := rand.Intn(len(sc.usedKeys))
		key = sc.usedKeys[next]
	}
	return
}

// perform executes operation for the given state index.
// It is a shortcut for getting an operation from the Transitions array and passing it the stateContext
func (sc *stateContext) perform(op op) {
	op(sc)
}
