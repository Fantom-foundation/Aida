package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"os"
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
	GetCommittedStateID
	SnapshotID
	RevertToSnapshotID
	CreateAccountID
	GetBalanceID
	GetCodeHashID
	SuicideID
	ExistID
	FinaliseID
	EndTransactionID
	BeginBlockID
	EndBlockID

	// Number of operations (must be last)
	NumOperations
)

// Dictionary data structure contains label and read function of an operation
type OperationDictionary struct {
	label    string
	readfunc func(*os.File) (Operation, error)
}

// opDict contains a dictionary of operation's label and read function.
var opDict = map[byte]OperationDictionary{
	GetStateID:          {label: "GetState", readfunc: ReadGetState},
	GetStateLclsID:      {label: "GetStateLcls", readfunc: ReadGetStateLcls},
	GetStateLcID:        {label: "GetStateLc", readfunc: ReadGetStateLc},
	GetStateLccsID:      {label: "GetStateLccs", readfunc: ReadGetStateLccs},
	SetStateID:          {label: "SetState", readfunc: ReadSetState},
	GetCommittedStateID: {label: "GetCommittedState", readfunc: ReadGetCommittedState},
	SnapshotID:          {label: "Snapshot", readfunc: ReadSnapshot},
	RevertToSnapshotID:  {label: "RevertToSnapshot", readfunc: ReadRevertToSnapshot},
	CreateAccountID:     {label: "CreateAccount", readfunc: ReadCreateAccount},
	GetBalanceID:        {label: "GetBalance", readfunc: ReadGetBalance},
	GetCodeHashID:       {label: "GetCodeHash", readfunc: ReadGetCodeHash},
	SuicideID:           {label: "Suicide", readfunc: ReadSuicide},
	ExistID:             {label: "Exist", readfunc: ReadExist},
	FinaliseID:          {label: "Finalise", readfunc: ReadFinalise},
	EndTransactionID:    {label: "EndTransaction", readfunc: ReadEndTransaction},
	BeginBlockID:        {label: "BeginBlock", readfunc: ReadBeginBlock},
	EndBlockID:          {label: "EndBlock", readfunc: ReadEndBlock},
}

// Profiling data structures for executed operations.
var (
	operationFrequency   = map[byte]uint64{}        // operation frequency stats
	operationDuration    = map[byte]time.Duration{} // accumulated operation duration
	operationMinDuration = map[byte]time.Duration{} // min runtime observerd
	operationMaxDuration = map[byte]time.Duration{} // max runtime observerd
	operationVariance    = map[byte]float64{}       // duration variance
)

// Get a label of a state operation.
func getLabel(i byte) string {
	if i < 0 || i >= NumOperations {
		log.Fatalf("getLabel failed; index is out-of-bound")
	}
	return opDict[i].label
}

// Operation interface.
type Operation interface {
	GetOpId() byte                                  // obtain operation identifier
	Write(*os.File) error                           // write operation
	Execute(state.StateDB, *dict.DictionaryContext) // execute operation
	Debug(*dict.DictionaryContext)                  // print debug message for operation
}

// Read an operation from file.
func Read(f *os.File) Operation {
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
func Write(f *os.File, op Operation) {
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

// Execute an operation.
func Execute(op Operation, db state.StateDB, ctx *dict.DictionaryContext) {
	var start time.Time
	if Profiling {
		start = time.Now()
	}
	op.Execute(db, ctx)
	if Profiling {
		elapsed := time.Since(start)
		op := op.GetOpId()
		// check if measuring this operation first time
		_, opExists := operationFrequency[op]
		// compute old mean of operation
		oldMean := float64(0.0)
		if opExists {
			// express in seconds
			oldMean = float64(operationDuration[op]) / float64(operationFrequency[op])
		}
		// update accumulated duration and frequency
		operationDuration[op] += elapsed
		operationFrequency[op]++
		// update min/max
		if opExists {
			if operationMaxDuration[op] < elapsed {
				operationMaxDuration[op] = elapsed
			}
			if operationMinDuration[op] > elapsed {
				operationMinDuration[op] = elapsed
			}
		} else {
			operationMinDuration[op] = elapsed
			operationMaxDuration[op] = elapsed
		}
		// update variance
		if opExists {
			n := float64(operationFrequency[op])
			newMean := float64(operationDuration[op]) / n
			operationVariance[op] = (n-2)*operationVariance[op]/(n-1) + (newMean-oldMean)*(newMean-oldMean)/n
		} else {
			operationVariance[op] = 0.0
		}
	}
}

// Print debug information of an operation.
func Debug(ctx *dict.DictionaryContext, op Operation) {
	fmt.Printf("%v:\n", getLabel(op.GetOpId()))
	op.Debug(ctx)
}

// PrintProfiling prints replay profiling information for executed operation.
func PrintProfiling() {
	fmt.Printf("op, n, mean(us), std(us), min(us), max(us)\n")
	total := float64(0)
	timeUnit := float64(time.Microsecond)
	for op, n := range operationFrequency {
		total += float64(operationDuration[op])
		mean := float64(operationDuration[op]) / float64(n) / timeUnit
		variance := operationVariance[op]
		std := math.Sqrt(variance) / timeUnit
		min := float64(operationMinDuration[op]) / timeUnit
		max := float64(operationMaxDuration[op]) / timeUnit
		fmt.Printf("%v, %v, %v, %v, %v, %v\n", getLabel(op), n, mean, std, min, max)
	}
	fmt.Printf("Total StateDB net execution time=%v (s)\n", total/float64(time.Second))
}
