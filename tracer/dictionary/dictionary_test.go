package dict

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestDictionarySimple1 encodes an value, and compares whether the
// decoded value is the same, and its index is zero.
func TestDictionarySimple1(t *testing.T) {
	encodedValue := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	dict := NewDictionary[common.Hash]()
	idx, err1 := dict.Encode(encodedValue)
	decodedValue, err2 := dict.Decode(idx)
	if encodedValue != decodedValue || err1 != nil || err2 != nil || idx != 0 {
		t.Fatalf("Encoding/decoding failed")
	}
}

// TestDictionarySimple2 encodes two valuees, and compares whether the
// decoded valuees are the same, and their dictionary indices are zero and one.
func TestDictionarySimple2(t *testing.T) {
	encodedValue1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	encodedValue2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F273")
	dict := NewDictionary[common.Hash]()
	idx1, err1 := dict.Encode(encodedValue1)
	idx2, err2 := dict.Encode(encodedValue2)
	decodedValue1, err3 := dict.Decode(idx1)
	decodedValue2, err4 := dict.Decode(idx2)
	if encodedValue1 != decodedValue1 || err1 != nil || err3 != nil || idx1 != 0 {
		t.Fatalf("Encoding/decoding value (1) failed")
	}
	if encodedValue2 != decodedValue2 || err2 != nil || err4 != nil || idx2 != 1 {
		t.Fatalf("Encoding/decoding value (2) failed")
	}
}

// TestDictionarySimple3 encodes one value twice and checks that its value
// is encoded only once, and its index is zero.
func TestDictionarySimple3(t *testing.T) {
	encodedValue1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	dict := NewDictionary[common.Hash]()
	idx1, err1 := dict.Encode(encodedValue1)
	idx2, err2 := dict.Encode(encodedValue1)
	decodedValue1, err3 := dict.Decode(idx1)
	decodedValue2, err4 := dict.Decode(idx2)
	if encodedValue1 != decodedValue1 || err1 != nil || err3 != nil || idx1 != 0 {
		t.Fatalf("Encoding/decoding value (1) failed")
	}
	if encodedValue1 != decodedValue2 || err2 != nil || err4 != nil || idx2 != 0 {
		t.Fatalf("Encoding/decoding value (2) failed")
	}
}

// TestDictionaryOverflow checks whether dictionary overflows can be captured.
func TestDictionaryOverflow(t *testing.T) {
	encodedValue1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	encodedValue2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F273")
	dict := NewDictionary[common.Hash]()
	// set limit to one storage
	DictionaryLimit = 1
	_, err1 := dict.Encode(encodedValue1)
	if err1 != nil {
		t.Fatalf("Failed to encode a storage key")
	}
	_, err2 := dict.Encode(encodedValue2)
	if err2 == nil {
		t.Fatalf("Failed to report error when adding an exising storage key")
	}
	// reset limit
	DictionaryLimit = math.MaxUint32
}

// TestDictionaryDecodingFailure1 checks whether invalid index for
// Decode() can be captured (retrieving index 0 on an empty dictionary).
func TestDictionaryDecodingFailure1(t *testing.T) {
	dict := NewDictionary[common.Hash]()
	_, err := dict.Decode(0)
	if err == nil {
		t.Fatalf("Failed to detect wrong index for Decode()")
	}
}

// TestDictionaryDecodingFailure2 checks whether invalid index for
// Decode() can be captured (retrieving index MaxUint32 on an empty dictionary).
func TestDictionaryDecodingFailure2(t *testing.T) {
	dict := NewDictionary[common.Hash]()
	_, err := dict.Decode(math.MaxUint32)
	if err == nil {
		t.Fatalf("Failed to detect wrong index for Decode()")
	}
}

// TestDictionaryReadFailure creates corrupted file and read file as dictionary.
func TestDictionaryReadFailure(t *testing.T) {
	filename := "./test.dict"
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file")
	}
	// write corrupted entry
	data := []byte("hello")
	if _, err := f.Write(data); err != nil {
		t.Fatalf("Failed to write data")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file")
	}
	rDict := NewDictionary[common.Hash]()
	err = rDict.Read(filename, 4711)
	if err == nil {
		t.Fatalf("Failed to report error when reading a corrupted file")
	}
	os.Remove(filename)
}

// TestDictionaryReadWrite encodes two valuees, write them to file, and
// read them from file. Check whether the newly created dictionary read from file is identical.
func TestDictionaryReadWrite(t *testing.T) {
	filename := "./test.dict"
	encodedValue1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	encodedValue2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F273")
	wDict := NewDictionary[common.Hash]()
	idx1, err1 := wDict.Encode(encodedValue1)
	idx2, err2 := wDict.Encode(encodedValue2)
	err := wDict.Write(filename, 4711)
	if err != nil {
		t.Fatalf("Failed to write file")
	}
	rDict := NewDictionary[common.Hash]()
	err = rDict.Read(filename, 4711)
	if err != nil {
		t.Fatalf("Failed to read file")
	}
	decodedValue1, err3 := rDict.Decode(idx1)
	decodedValue2, err4 := rDict.Decode(idx2)
	if encodedValue1 != decodedValue1 || err1 != nil || err3 != nil || idx1 != 0 {
		t.Fatalf("Encoding/Decoding failed")
	}
	if encodedValue2 != decodedValue2 || err2 != nil || err4 != nil || idx2 != 1 {
		t.Fatalf("Encoding/Decoding failed")
	}
	os.Remove(filename)
}

// TestCodeDictionarySimple1 encodes byte-code, and compares whether the
// decoded bytecode is the same, and its index is zero.
func TestCodeDictionarySimple1(t *testing.T) {
	encodedCode := []byte{0x1, 0x0, 0x02, 0x5, 0x7}
	dict := NewDictionary[string]()
	idx, err1 := dict.Encode(string(encodedCode))
	decodedCode, err2 := dict.Decode(idx)
	if !reflect.DeepEqual(encodedCode, []byte(decodedCode)) || err1 != nil || err2 != nil || idx != 0 {
		t.Fatalf("Encoding/Decoding failed %v %v", encodedCode, decodedCode)
	}
}

// TestCodeDictionarySimple2 encoded two byte-codes, and compare whether the decoded
// bytecode are not the same, and their indices are zero and one.
func TestCodeDictionarySimple2(t *testing.T) {
	encodedCode1 := []byte{0x1, 0x0, 0x2, 0x0, 0x5}
	encodedCode2 := []byte{0x1, 0x0, 0x2}
	dict := NewDictionary[string]()
	idx1, err1 := dict.Encode(string(encodedCode1))
	idx2, err2 := dict.Encode(string(encodedCode2))
	decodedCode1, err3 := dict.Decode(idx1)
	decodedCode2, err4 := dict.Decode(idx2)
	if !reflect.DeepEqual(encodedCode1, []byte(decodedCode1)) || err1 != nil || err3 != nil || idx1 != 0 {
		t.Fatalf("Encoding/decoding byte-code (1) failed")
	}
	if !reflect.DeepEqual(encodedCode2, []byte(decodedCode2)) || err2 != nil || err4 != nil || idx2 != 1 {
		t.Fatalf("Encoding/decoding byte-code (2) failed")
	}
}

// TestCodeDictionarySimple3 encodes the same byte code twice and check that it
// is encoded only once, and its index is zero.
func TestCodeDictionarySimple3(t *testing.T) {
	encodedCode := []byte{0x1, 0x02, 0x3, 0x4}
	dict := NewDictionary[string]()
	idx1, err1 := dict.Encode(string(encodedCode))
	idx2, err2 := dict.Encode(string(encodedCode))
	decodedCode1, err3 := dict.Decode(idx1)
	decodedCode2, err4 := dict.Decode(idx2)
	if !reflect.DeepEqual(encodedCode, []byte(decodedCode1)) || err1 != nil || err3 != nil || idx1 != 0 {
		t.Fatalf("Encoding/decoding byte-code (1) failed")
	}
	if !reflect.DeepEqual(encodedCode, []byte(decodedCode2)) || err2 != nil || err4 != nil || idx2 != 0 {
		t.Fatalf("Encoding/decoding byte-code (2) failed")
	}
}

// TestCodeDictionaryOverflow checks whether dictionary overflows can be captured.
func TestCodeDictionaryOverflow(t *testing.T) {
	encodedCode1 := []byte{0x1, 0x0, 0x2, 0x0, 0x5}
	encodedCode2 := []byte{0x1, 0x0, 0x2}
	dict := NewDictionary[string]()
	// set limit to one storage
	DictionaryLimit = 1
	_, err1 := dict.Encode(string(encodedCode1))
	if err1 != nil {
		t.Fatalf("Failed to encode a storage key")
	}
	_, err2 := dict.Encode(string(encodedCode2))
	if err2 == nil {
		t.Fatalf("Failed to report error when adding an exising storage key")
	}
	// reset limit
	DictionaryLimit = math.MaxInt - 1
}

// TestCodeDictionaryDecodingFailure1 checks whether invalid index for Decode() can be captured.
// (Retrieving index 0 on an empty dictionary)
func TestCodeDictionaryDecodingFailure1(t *testing.T) {
	dict := NewDictionary[string]()
	_, err := dict.Decode(0)
	if err == nil {
		t.Fatalf("Failed to detect wrong index for Decode()")
	}
}

// TestCodeDictionaryDecodingFailure2 checks whether invalid index for Decode() can be captured.
// (Retrieving index MaxUint32 on an empty dictionary)
func TestCodeDictionaryDecodingFailure2(t *testing.T) {
	dict := NewDictionary[string]()
	_, err := dict.Decode(math.MaxUint32)
	if err == nil {
		t.Fatalf("Failed to detect wrong index for Decode()")
	}
}

// TestCodeDictionaryReadFailure creates corrupted file and read file as dictionary.
func TestCodeDictionaryReadFailure(t *testing.T) {
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
	rDict := NewDictionary[string]()
	err = rDict.Read(filename, 4711)
	if err == nil {
		t.Fatalf("Failed to report error when reading a corrupted file")
	}
	os.Remove(filename)
}

// TestCodeDictionaryReadWrite encodes two byte codes, writes them to file,
// and reads them from file. Check whether the newly created dictionary (read from
// file) is identical.
func TestCodeDictionaryReadWrite(t *testing.T) {
	filename := "./test.dict"
	encodedCode1 := []byte{0x1, 0x0, 0x2, 0x0, 0x5}
	encodedCode2 := []byte{0x1, 0x0, 0x2}
	wDict := NewDictionary[string]()
	idx1, err1 := wDict.Encode(string(encodedCode1))
	idx2, err2 := wDict.Encode(string(encodedCode2))
	err := wDict.WriteString(filename, 4711)
	if err != nil {
		t.Fatalf("Failed to write file. Error: %v", err)
	}
	rDict := NewDictionary[string]()
	err = rDict.ReadString(filename, 4711)
	if err != nil {
		t.Fatalf("Failed to read file")
	}
	decodedCode1, err3 := rDict.Decode(idx1)
	decodedCode2, err4 := rDict.Decode(idx2)
	if !reflect.DeepEqual(encodedCode1, []byte(decodedCode1)) || err1 != nil || err3 != nil || idx1 != 0 {
		fmt.Printf("%v %v\n", encodedCode1, decodedCode1)
		t.Fatalf("Encoding/decoding byte-code (1) failed")
	}
	if !reflect.DeepEqual(encodedCode2, []byte(decodedCode2)) || err2 != nil || err4 != nil || idx2 != 1 {
		fmt.Printf("%v %v\n", encodedCode2, decodedCode2)
		t.Fatalf("Encoding/decoding byte-code (2) failed")
	}
	os.Remove(filename)
}
