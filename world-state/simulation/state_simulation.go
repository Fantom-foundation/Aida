package simulation

import (
	"encoding/binary"
	"fmt"
	"github.com/Fantom-foundation/Carmen/go/common"
	"github.com/Fantom-foundation/Carmen/go/state"
	"math"
	"math/big"
	"math/rand"
	"time"
)

const KeysCacheSize = 256

func main() {
	memState, err := state.NewMemory("")
	if err != nil {
		panic(err)
	}
	// TODO - how to generate max range for Address and Keys?
	dist := common.Exponential.GetDistribution(math.MaxInt)
	t := initTransitions()

	// TODO loop from here N-times for N blocks
	// simulate one block processing
	simulate(state.CreateStateDBUsing(memState), dist, t)
}

// simulate executes simulation from StartBlock and runs the Markov Chain until EndBlock is reached
func simulate(stateDB state.StateDB, dist common.Distribution, transitions transitions) {
	sc := newStateContext(stateDB, dist)
	n := len(transitions.ops)

	// run Markov chain
	var previousState, currentState, steps int
	for !sc.finished {
		steps++
		// execute current state
		sc.perform(transitions.ops[currentState])

		// determine next state
		p := rand.Float64()
		sum := 0.0
		for i := 0; i < n; i++ {
			sum += transitions.probabilities[i][currentState]
			if p <= sum {
				currentState = i
				break
			}
		}
		fmt.Printf("Step: %3.0d. (p %f) %25s -> %25s, address: %x, key: %x, value: %x, balance: %x, nonce: %x \n", steps, p, transitions.labels[previousState], transitions.labels[currentState], sc.address, sc.key, sc.value, sc.balance, sc.nonce)
		previousState = currentState
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
	finished     bool                // Flag switched to true once the End block is reached during the simulation
}

// newStateContext creates a new context, which contains current state of transitions
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
		finished:     false,
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
// It is a shortcut for getting an operation from the transitions array and passing it the stateContext
func (sc *stateContext) perform(op op) {
	op(sc)
}

// transitions contains probabilities of transitions between operations
type transitions struct {
	ops           []op        // operations are in array indexed in the same order as probabilities in the matrix
	probabilities [][]float64 // probabilities in the matrix
	labels        []string    // name of ops for debug purposes
}

// op is an operation to transit the state from one to another
type op func(c *stateContext)

// initTransitions creates stochastic matrix of probabilities between states and operations to perform on each transition
func initTransitions() transitions {
	BeginBlock := func(c *stateContext) {
		// regenerate both address and key
		c.address = c.getNextAddress()
		c.key = c.getNextKey()
	}
	GetState := func(c *stateContext) {
		// regenerate both address and key
		c.address = c.getNextAddress()
		c.key = c.getNextKey()
		c.value = c.stateDB.GetState(c.address, c.key)
	}
	GetStateLcls := func(c *stateContext) {
		// the same address and key
		c.value = c.stateDB.GetState(c.address, c.key)
	}
	GetStateLc := func(c *stateContext) {
		// the same address and a new key
		c.key = c.getNextKey()
		c.value = c.stateDB.GetState(c.address, c.key)
	}
	GetStateLccs := func(c *stateContext) {
		// the same address, the key random from the cache of already used ones
		c.key = c.getUsedKey()
		c.value = c.stateDB.GetState(c.address, c.key)
	}
	SetStateLcls := func(c *stateContext) {
		// the same address and key
		c.stateDB.SetState(c.address, c.key, c.getNextValue())
	}
	GetCommittedStateLcls := func(c *stateContext) {
		// the same address and key
		c.value = c.stateDB.GetCommittedState(c.address, c.key)
	}
	Snapshot := func(c *stateContext) {
		c.snapshot = c.stateDB.Snapshot()
	}
	RevertToSnapshot := func(c *stateContext) {
		c.stateDB.RevertToSnapshot(c.snapshot)
	}
	GetBalance := func(c *stateContext) {
		c.balance = c.stateDB.GetBalance(c.address)
	}
	AddBalance := func(c *stateContext) {
		c.stateDB.AddBalance(c.address, c.getNextBalance())
	}
	SubBalance := func(c *stateContext) {
		// prevent sub more than the current balance is
		newBalance := int64(math.Min(float64(c.balance.Int64()), float64(c.getNextBalance().Int64())))
		c.stateDB.SubBalance(c.address, big.NewInt(newBalance))
	}
	GetCodeHash := func(c *stateContext) {
		//  TODO CodeHash not implemented yet
	}
	SetCodeHash := func(c *stateContext) {
		//  TODO CodeHash not implemented yet
	}
	CreateAccount := func(c *stateContext) {
		c.address = c.getNextAddress()
		c.stateDB.CreateAccount(c.address)
	}
	Exist := func(c *stateContext) {
		c.address = c.getNextAddress()
		_ = c.stateDB.Exist(c.address)
	}
	Empty := func(c *stateContext) {
		c.address = c.getNextAddress()
		_ = c.stateDB.Empty(c.address)
	}
	Suicide := func(c *stateContext) {
		c.stateDB.Suicide(c.address)
	}
	HasSuicided := func(c *stateContext) {
		_ = c.stateDB.HasSuicided(c.address)
	}
	GetNonce := func(c *stateContext) {
		c.nonce = c.stateDB.GetNonce(c.address)
	}
	SetNonce := func(c *stateContext) {
		c.stateDB.SetNonce(c.address, c.getNextNonce())
	}
	AbortTransaction := func(c *stateContext) {
		c.stateDB.AbortTransaction()
	}
	EndTransaction := func(c *stateContext) {
		c.stateDB.EndTransaction()
	}
	Finalise := func(c *stateContext) {
		c.stateDB.EndTransaction()
	}
	EndBlock := func(c *stateContext) {
		c.finished = true
	}

	ops := []op{
		BeginBlock,
		GetState,
		GetStateLcls,
		GetStateLc,
		GetStateLccs,
		SetStateLcls,
		GetCommittedStateLcls,
		Snapshot,
		RevertToSnapshot,
		GetBalance,
		AddBalance,
		SubBalance,
		GetCodeHash,
		SetCodeHash,
		CreateAccount,
		Exist,
		Empty,
		Suicide,
		HasSuicided,
		GetNonce,
		SetNonce,
		AbortTransaction,
		EndTransaction,
		Finalise,
		EndBlock,
	}

	labels := []string{
		"BeginBlock",
		"GetState",
		"GetStateLcls",
		"GetStateLc",
		"GetStateLccs",
		"SetStateLcls",
		"GetCommittedStateLcls",
		"Snapshot",
		"RevertToSnapshot",
		"GetBalance",
		"AddBalance",
		"SubBalance",
		"GetCodeHash",
		"SetCodeHash",
		"CreateAccount",
		"Exist",
		"Empty",
		"Suicide",
		"HasSuicided",
		"GetNonce",
		"SetNonce",
		"AbortTransaction",
		"EndTransaction",
		"Finalise",
		"EndBlock",
	}

	probabilities := [][]float64{
		// 	BeginBlock, 	 GetState,				 GetStateLcls,		 GetStateLc, 		GetStateLccs, 			SetStateLcls, 		GetCommittedStateLcls, 	Snapshot, 			RevertToSnapshot, 		GetBalance,		 AddBalance, 		SubBalance, 			GetCodeHash, 		SetCodeHash, 		CreateAccount,		Exist, 				Empty, 				Suicide, 				HasSuicided, 		GetNonce, 			SetNonce,			 AbortTransaction, 		EndTransaction, 	Finalise,			 EndBlock
		{0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 1.00000000000000000}, // BeginBlock
		{0.00000000000000000, 0.01672955450395047, 0.00692793386092787, 0.06270554854003470, 0.03099525535429102, 0.08934384613087239, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.02187741913769311, 0.00000000000000000, 0.30877573131094260, 0.01121985050473915, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // GetState
		{0.00000000000000000, 0.15142070920803019, 0.02760941633459465, 0.16463178277880092, 0.54352355177647571, 0.00674222739897859, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.04842202102476075, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // GetStateLcls
		{0.00000000000000000, 0.22233108882820155, 0.51163461288030154, 0.10733402359167486, 0.06154651132608271, 0.35536316936953083, 0.00000000000000000, 0.00027860156676723, 0.00000000000000000, 0.00064878015814016, 0.00000000000000000, 0.00000000000000000, 0.30455611277529088, 0.00000000000000000, 0.00325027085590466, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // GetStateLc
		{0.00000000000000000, 0.19740874314661552, 0.05072423325780323, 0.53025622826073848, 0.22139175363362265, 0.38661036944120897, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00016219503953504, 0.00000000000000000, 0.00000000000000000, 0.42018855717485853, 0.00000000000000000, 0.00216684723726977, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // GetStateLccs
		{0.00000000000000000, 0.00481561008151098, 0.00899477986508760, 0.00186231930248304, 0.00274596541176503, 0.00000000000000000, 0.99979963626275392, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // SetStateLcls
		{0.00000000000000000, 0.00181367132940024, 0.38439614982494130, 0.04014237008987715, 0.04020175331940769, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // GetCommittedStateLcls
		{1.00000000000000000, 0.34513956930517625, 0.00896315395863331, 0.05755962092068518, 0.08727115247212025, 0.10261177313860766, 0.00000000000000000, 0.00000000000000000, 0.00010873504893077, 0.78471311752382245, 0.00000000000000000, 0.00000000000000000, 0.02915867299428943, 0.00000000000000000, 0.28819068255687974, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.77261146496815292, 0.00000000000000000}, // Snapshot
		{0.00000000000000000, 0.00169901394650712, 0.00050415415583014, 0.03298079623194747, 0.00970650957740822, 0.00175835498611236, 0.00020036373724608, 0.00000366581008904, 0.01080101486045669, 0.00109481651686153, 0.00000000000000000, 0.00000000000000000, 0.00001495891906851, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // RevertToSnapshot
		{0.00000000000000000, 0.04644666347015781, 0.00019533648104120, 0.00241877548629619, 0.00222409536833506, 0.01280500552519189, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.21327296073528418, 0.00000000000000000, 0.00000000000000000, 0.17578225797403879, 0.00000000000000000, 0.00000000000000000, 0.00137936348925021, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // GetBalance
		{0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // AddBalance
		{0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // SubBalance
		{0.00000000000000000, 0.00012508078133795, 0.00000000000000000, 0.00000344554912578, 0.00000273230389230, 0.00005599856643670, 0.00000000000000000, 0.52397989669747169, 0.00000000000000000, 0.00009461377306211, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.95880403791323110, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // GetCodeHash
		{0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // SetCodeHash
		{0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00002566067062330, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00705864221314634, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // CreateAccount
		{0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.47571217525504872, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // Exist
		{0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // Empty
		{0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // Suicide
		{0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // HasSuicided
		{0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // GetNonce
		{0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // SetNonce
		{0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // AbortTransaction
		{0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // EndTransaction
		{0.00000000000000000, 0.01207029539911193, 0.00005022938083917, 0.00010508924833623, 0.00039071945659940, 0.04470925544306066, 0.00000000000000000, 0.00000000000000000, 0.98909025009061258, 0.00001351625329459, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.39761646803900325, 0.02153810587963320, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000}, // Finalise
		{0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.00000000000000000, 0.22738853503184714, 0.00000000000000000}, // EndBlock
	}

	return transitions{ops, probabilities, labels}
}
