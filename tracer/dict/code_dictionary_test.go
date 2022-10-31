package dict

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"testing"
)

// TestPositiveCodeDictionarySimple1 encodes byte-code, and compares whether the
// decoded bytecode is the same, and its index is zero.
func TestPositiveCodeDictionarySimple1(t *testing.T) {
	encodedCode := []byte{0x1, 0x0, 0x02, 0x5, 0x7}
	dict := NewCodeDictionary()
	idx, err1 := dict.Encode(encodedCode)
	decodedCode, err2 := dict.Decode(idx)
	if !reflect.DeepEqual(encodedCode, decodedCode) || err1 != nil || err2 != nil || idx != 0 {
		t.Fatalf("Encoding/Decoding failed")
	}
}

// TestPositiveCodeDictionarySimple2 encoded two byte-codes, and compare whether the decoded
// bytecode are not the same, and their indices are zero and one.
func TestPositiveCodeDictionarySimple2(t *testing.T) {
	encodedCode1 := []byte{0x1, 0x0, 0x2, 0x0, 0x5}
	encodedCode2 := []byte{0x1, 0x0, 0x2}
	dict := NewCodeDictionary()
	idx1, err1 := dict.Encode(encodedCode1)
	idx2, err2 := dict.Encode(encodedCode2)
	decodedCode1, err3 := dict.Decode(idx1)
	decodedCode2, err4 := dict.Decode(idx2)
	if !reflect.DeepEqual(encodedCode1, decodedCode1) || err1 != nil || err3 != nil || idx1 != 0 {
		t.Fatalf("Encoding/decoding byte-code (1) failed")
	}
	if !reflect.DeepEqual(encodedCode2, decodedCode2) || err2 != nil || err4 != nil || idx2 != 1 {
		t.Fatalf("Encoding/decoding byte-code (2) failed")
	}
}

// TestPositiveCodeDictionarySimple3 encodes the same byte code twice and check that it
// is encoded only once, and its index is zero.
func TestPositiveCodeDictionarySimple3(t *testing.T) {
	encodedCode := []byte{0x1, 0x02, 0x3, 0x4}
	dict := NewCodeDictionary()
	idx1, err1 := dict.Encode(encodedCode)
	idx2, err2 := dict.Encode(encodedCode)
	decodedCode1, err3 := dict.Decode(idx1)
	decodedCode2, err4 := dict.Decode(idx2)
	if !reflect.DeepEqual(encodedCode, decodedCode1) || err1 != nil || err3 != nil || idx1 != 0 {
		t.Fatalf("Encoding/decoding byte-code (1) failed")
	}
	if !reflect.DeepEqual(encodedCode, decodedCode2) || err2 != nil || err4 != nil || idx2 != 0 {
		t.Fatalf("Encoding/decoding byte-code (2) failed")
	}
}

// TestNegativeCodeDictionaryOverflow checks whether dictionary overflows can be captured.
func TestNegativeCodeDictionaryOverflow(t *testing.T) {
	encodedCode1 := []byte{0x1, 0x0, 0x2, 0x0, 0x5}
	encodedCode2 := []byte{0x1, 0x0, 0x2}
	dict := NewCodeDictionary()
	// set limit to one storage
	CodeDictionaryLimit = 1
	_, err1 := dict.Encode(encodedCode1)
	if err1 != nil {
		t.Fatalf("Failed to encode a storage key")
	}
	_, err2 := dict.Encode(encodedCode2)
	if err2 == nil {
		t.Fatalf("Failed to report error when adding an exising storage key")
	}
	// reset limit
	CodeDictionaryLimit = math.MaxUint32
}

// TestNegativeCodeDictionaryDecodingFailure1 checks whether invalid index for Decode() can be captured.
// (Retrieving index 0 on an empty dictionary)
func TestNegativeCodeDictionaryDecodingFailure1(t *testing.T) {
	dict := NewCodeDictionary()
	_, err := dict.Decode(0)
	if err == nil {
		t.Fatalf("Failed to detect wrong index for Decode()")
	}
}

// TestNegativeCodeDictionaryDecodingFailure2 checks whether invalid index for Decode() can be captured.
// (Retrieving index MaxUint32 on an empty dictionary)
func TestNegativeCodeDictionaryDecodingFailure2(t *testing.T) {
	dict := NewCodeDictionary()
	_, err := dict.Decode(math.MaxUint32)
	if err == nil {
		t.Fatalf("Failed to detect wrong index for Decode()")
	}
}

// TestNegativeCodeDictionaryReadFailure creates corrupted file and read file as dictionary.
func TestNegativeCodeDictionaryReadFailure(t *testing.T) {
	filename := "./test.dict"
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file")
	}
	// write corrupted entry
	data := []byte("hellodello")
	if _, err := f.Write(data); err != nil {
		t.Fatalf("Failed to write data")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file")
	}
	rDict := NewCodeDictionary()
	err = rDict.Read(filename)
	if err == nil {
		t.Fatalf("Failed to report error when reading a corrupted file")
	}
	os.Remove(filename)
}

// TestPositiveCodeDictionaryReadWrite encodes two byte codes, writes them to file,
// and reads them from file. Check whether the newly created dictionary (read from
// file) is identical.
func TestPositiveCodeDictionaryReadWrite(t *testing.T) {
	filename := "./test.dict"
	encodedCode1 := []byte{0x1, 0x0, 0x2, 0x0, 0x5}
	encodedCode2 := []byte{0x1, 0x0, 0x2}
	wDict := NewCodeDictionary()
	idx1, err1 := wDict.Encode(encodedCode1)
	idx2, err2 := wDict.Encode(encodedCode2)
	err := wDict.Write(filename)
	if err != nil {
		t.Fatalf("Failed to write file")
	}
	rDict := NewCodeDictionary()
	err = rDict.Read(filename)
	if err != nil {
		t.Fatalf("Failed to read file")
	}
	decodedCode1, err3 := rDict.Decode(idx1)
	decodedCode2, err4 := rDict.Decode(idx2)
	if !reflect.DeepEqual(encodedCode1, decodedCode1) || err1 != nil || err3 != nil || idx1 != 0 {
		fmt.Printf("%v %v\n", encodedCode1, decodedCode1)
		t.Fatalf("Encoding/decoding byte-code (1) failed")
	}
	if !reflect.DeepEqual(encodedCode2, decodedCode2) || err2 != nil || err4 != nil || idx2 != 1 {
		fmt.Printf("%v %v\n", encodedCode2, decodedCode2)
		t.Fatalf("Encoding/decoding byte-code (2) failed")
	}
	os.Remove(filename)
}
