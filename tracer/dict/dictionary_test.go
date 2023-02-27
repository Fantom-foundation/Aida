package dict

import (
	"math"
	"os"
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
	err = rDict.Read(filename)
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
	err := wDict.Write(filename)
	if err != nil {
		t.Fatalf("Failed to write file")
	}
	rDict := NewDictionary[common.Hash]()
	err = rDict.Read(filename)
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
