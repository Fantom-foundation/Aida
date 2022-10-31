package operation

import (
	"bytes"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/ethereum/go-ethereum/common"
	"io"
	"os"
	"reflect"
	"testing"
)

// TestGetCodeSimple creates a new GetCode object and checks its ID.
func TestGetCodeSimple(t *testing.T) {
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(common.HexToAddress("0x213129039821098302981"))

	// create new operation
	op := NewGetCode(cIdx)
	if op == nil {
		t.Fatalf("Failed to create operation")
	}

	// check id
	if op.GetId() != GetCodeID {
		t.Fatalf("Wrong ID returned")
	}
}

// TestGetCodeReadWrite writes a new GetCode object into a buffer, reads from it,
// and checks equality.
func TestGetCodeReadWrite(t *testing.T) {
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(common.HexToAddress("0x213129039821098302981"))

	// create new operation and write to memory buffer
	op1 := NewGetCode(cIdx)
	if op1 == nil {
		t.Fatalf("Failed to create operation")
	}
	if op1.GetId() != GetCodeID {
		t.Fatalf("Wrong ID returned")
	}
	op1Buffer := bytes.NewBufferString("")
	op1.Write(op1Buffer)

	// read object from buffer
	op2Buffer := bytes.NewBufferString(op1Buffer.String())
	op2, err := ReadGetCode(op2Buffer)
	if err != nil {
		t.Fatalf("Failed to read operation. Error: %v", err)
	}
	if op2 == nil {
		t.Fatalf("Failed to create newly read operation from buffer")
	}
	// check equivalence
	if !reflect.DeepEqual(op1, op2) {
		t.Fatalf("Operations are not the same")
	}
}

// TestGetCodeDebug creates a new GetCode object and checks its Debug message.
func TestGetCodeDebug(t *testing.T) {
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(common.HexToAddress("0x213129039821098302981"))

	// create new operation
	op := NewGetCode(cIdx)
	if op == nil {
		t.Fatalf("Failed to create operation")
	}
	if op.GetId() != GetCodeID {
		t.Fatalf("Wrong ID returned")
	}

	// divert stdout to a buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// print debug message
	op.Debug(dict)

	// restore stdout
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// check debug message
	if buf.String() != "\tcontract: 0x0000000000000000000213129039821098302981\n" {
		t.Fatalf("Wrong debug message: %v", buf.String())
	}
}

// TestGetCodeExecute creates a new GetCode object and checks its execution signature.
func TestGetCodeExecute(t *testing.T) {
	// create dictionary context
	dict := dict.NewDictionaryContext()
	cIdx := dict.EncodeContract(common.HexToAddress("0x213129039821098302981"))

	// create new operation
	op := NewGetCode(cIdx)
	if op == nil {
		t.Fatalf("Failed to create operation")
	}
	// check id
	if op.GetId() != GetCodeID {
		t.Fatalf("Wrong ID returned")
	}

	// check execution
	mock := NewMockStateDB()
	op.Execute(mock, dict)
	if mock.GetSignature() != "GetCode: 0x0000000000000000000213129039821098302981" {
		t.Fatalf("Execution signature fails: %v", mock.GetSignature())
	}
}
