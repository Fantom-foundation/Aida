package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

var Profiling = false

// Operation IDs of the StateDB interface
const (
	GetStateID = iota
	GetStateLclsID
	GetStateLcID
	GetStateLccsID
	SetStateID
	SetStateLclsID
	GetCommittedStateID
	GetCommittedStateLclsID
	GetNonceID
	SetNonceID
	SnapshotID
	RevertToSnapshotID
	CreateAccountID
	AddBalanceID
	GetBalanceID
	SubBalanceID
	GetCodeID
	GetCodeHashID
	GetCodeHashLcID
	GetCodeSizeID
	SetCodeID
	SuicideID
	ExistID
	FinaliseID
	EndTransactionID
	BeginBlockID
	EndBlockID

	// Number of operations (must be last)
	NumOperations
)

// OperationDictionary data structure contains a label and a read function for an operation
type OperationDictionary struct {
	label    string                             // operation's label
	readfunc func(io.Reader) (Operation, error) // operation's read-function
}

// opDict relates an operation's id with its label and read-function.
var opDict = map[byte]OperationDictionary{
	GetStateID:              {label: "GetState", readfunc: ReadGetState},
	GetStateLclsID:          {label: "GetStateLcls", readfunc: ReadGetStateLcls},
	GetStateLcID:            {label: "GetStateLc", readfunc: ReadGetStateLc},
	GetStateLccsID:          {label: "GetStateLccs", readfunc: ReadGetStateLccs},
	SetStateID:              {label: "SetState", readfunc: ReadSetState},
	SetStateLclsID:          {label: "SetStateLcls", readfunc: ReadSetStateLcls},
	GetCommittedStateID:     {label: "GetCommittedState", readfunc: ReadGetCommittedState},
	GetCommittedStateLclsID: {label: "GetCommittedStateLcls", readfunc: ReadGetCommittedStateLcls},
	SnapshotID:              {label: "Snapshot", readfunc: ReadSnapshot},
	RevertToSnapshotID:      {label: "RevertToSnapshot", readfunc: ReadRevertToSnapshot},
	CreateAccountID:         {label: "CreateAccount", readfunc: ReadCreateAccount},
	AddBalanceID:            {label: "AddBalance", readfunc: ReadAddBalance},
	GetBalanceID:            {label: "GetBalance", readfunc: ReadGetBalance},
	SubBalanceID:            {label: "SubBalance", readfunc: ReadSubBalance},
	GetNonceID:              {label: "GetNonce", readfunc: ReadGetNonce},
	SetNonceID:              {label: "SetNonce", readfunc: ReadSetNonce},
	GetCodeID:               {label: "GetCode", readfunc: ReadGetCode},
	GetCodeSizeID:           {label: "GetCodeSize", readfunc: ReadGetCodeSize},
	SetCodeID:               {label: "SetCode", readfunc: ReadSetCode},
	GetCodeHashID:           {label: "GetCodeHash", readfunc: ReadGetCodeHash},
	GetCodeHashLcID:         {label: "GetCodeLcHash", readfunc: ReadGetCodeHashLc},
	SuicideID:               {label: "Suicide", readfunc: ReadSuicide},
	ExistID:                 {label: "Exist", readfunc: ReadExist},
	FinaliseID:              {label: "Finalise", readfunc: ReadFinalise},
	EndTransactionID:        {label: "EndTransaction", readfunc: ReadEndTransaction},
	BeginBlockID:            {label: "BeginBlock", readfunc: ReadBeginBlock},
	EndBlockID:              {label: "EndBlock", readfunc: ReadEndBlock},
}

// Profiling data structures for executing operations.
var (
	opFrequencyy  [NumOperations]uint64        // operation frequency stats
	opDuration    [NumOperations]time.Duration // accumulated operation duration
	opMinDuration [NumOperations]time.Duration // min runtime observerd
	opMaxDuration [NumOperations]time.Duration // max runtime observerd
	opVariance    [NumOperations]float64       // duration variance
)

// getLabel retrieves a label of a state operation.
func getLabel(i byte) string {
	if i < 0 || i >= NumOperations {
		log.Fatalf("getLabel failed; index is out-of-bound")
	}
	if _, ok := opDict[i]; !ok {
		log.Fatalf("operation is not defined")
	}

	return opDict[i].label
}

// Operation interface.
type Operation interface {
	GetOpId() byte                                                // get operation identifier
	Write(io.Writer) error                                        // write operation to a file
	Execute(state.StateDB, *dict.DictionaryContext) time.Duration // execute operation on a stateDB instance
	Debug(*dict.DictionaryContext)                                // print debug message for operation
}

// Read an operation from file.
func Read(f io.Reader) Operation {
	var (
		op Operation
		ID byte
	)

	// read ID from file
	err := binary.Read(f, binary.LittleEndian, &ID)
	if err == io.EOF {
		return nil
	} else if err != nil {
		log.Fatalf("Cannot read ID from file. Error:%v", err)
	}
	if ID >= NumOperations {
		log.Fatalf("ID out of range %v", ID)
	}

	// read state operation
	op, err = opDict[ID].readfunc(f)
	if err != nil {
		log.Fatalf("Failed to read operation %v. Error %v", getLabel(ID), err)
	}
	if op.GetOpId() != ID {
		log.Fatalf("Generated object of type %v has wrong ID (%v) ", getLabel(op.GetOpId()), getLabel(ID))
	}
	return op
}

// Write an operation to file.
func Write(f io.Writer, op Operation) {
	// write ID to file
	ID := op.GetOpId()
	if err := binary.Write(f, binary.LittleEndian, &ID); err != nil {
		log.Fatalf("Failed to write ID for operation %v. Error: %v", getLabel(ID), err)
	}

	// write details of operation to file
	if err := op.Write(f); err != nil {
		log.Fatalf("Failed to write operation %v. Error: %v", getLabel(ID), err)
	}
}

// Execute an operation and profile it.
func Execute(op Operation, db state.StateDB, ctx *dict.DictionaryContext) {
	elapsed := op.Execute(db, ctx)
	if Profiling {
		op := op.GetOpId()
		n := opFrequencyy[op]
		duration := opDuration[op]
		// update min/max values
		if n > 0 {
			if opMaxDuration[op] < elapsed {
				opMaxDuration[op] = elapsed
			}
			if opMinDuration[op] > elapsed {
				opMinDuration[op] = elapsed
			}
		} else {
			opMinDuration[op] = elapsed
			opMaxDuration[op] = elapsed
		}
		// compute previous mean
		prevMean := float64(0.0)
		if n > 0 {
			prevMean = float64(opDuration[op]) / float64(n)
		}
		// update variance
		newDuration := duration + elapsed
		if n > 0 {
			newMean := float64(newDuration) / float64(n+1)
			opVariance[op] = float64(n-1)*opVariance[op]/float64(n) +
				(newMean-prevMean)*(newMean-prevMean)/float64(n+1)
		} else {
			opVariance[op] = 0.0
		}

		// update execution frequency
		opFrequencyy[op] = n + 1

		// update accumulated duration and frequency
		opDuration[op] = newDuration
	}
}

// Debug prints debug information of an operation.
func Debug(ctx *dict.DictionaryContext, op Operation) {
	fmt.Printf("%v:\n", getLabel(op.GetOpId()))
	op.Debug(ctx)
}

// PrintProfiling prints replay profiling information for executed operation.
func PrintProfiling() {
	timeUnit := float64(time.Microsecond)
	tuStr := "us"
	fmt.Printf("op, n, mean(%v), std(%v), min(%v), max(%v)\n", tuStr, tuStr, tuStr, tuStr)
	total := float64(0)
	for op := byte(0); op < NumOperations; op++ {
		n := opFrequencyy[op]
		mean := (float64(opDuration[op]) / float64(n)) / timeUnit
		std := math.Sqrt(opVariance[op]) / timeUnit
		min := float64(opMinDuration[op]) / timeUnit
		max := float64(opMaxDuration[op]) / timeUnit
		fmt.Printf("%v, %v, %v, %v, %v, %v\n", getLabel(op), n, mean, std, min, max)

		total += float64(opDuration[op])
	}
	sec := total / float64(time.Second)
	tps := float64(opFrequencyy[EndTransactionID]) / sec
	fmt.Printf("Total StateDB net execution time=%v (s) / ~%.1f Tx/s\n", sec, tps)
}
