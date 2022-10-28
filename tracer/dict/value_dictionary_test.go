package dict

import (
	"github.com/ethereum/go-ethereum/common"
	"math"
	"os"
	"testing"
)

// TestPositiveValueDictionarySimple1 encodes an value, and compares whether the 
// decoded value is the same, and its index is zero.
func TestPositiveValueDictionarySimple1(t *testing.T) {
	encodedValue := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	dict := NewValueDictionary()
	idx, err1 := dict.Encode(encodedValue)
	decodedValue, err2 := dict.Decode(idx)
	if encodedValue != decodedValue || err1 != nil || err2 != nil || idx != 0 {
		t.Fatalf("Encoding/decoding failed")
	}
}

// TestPositiveValueDictionarySimple2 encodes two valuees, and compares whether the 
// decoded valuees are the same, and their dictionary indices are zero and one.
func TestPositiveValueDictionarySimple2(t *testing.T) {
	encodedValue1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	encodedValue2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F273")
	dict := NewValueDictionary()
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

// TestPositiveValueDictionarySimple3 encodes one value twice and checks that its value
// is encoded only once, and its index is zero.
func TestPositiveValueDictionarySimple3(t *testing.T) {
	encodedValue1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	dict := NewValueDictionary()
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

// TestNegativeValueDictionaryOverflow checks whether dictionary overflows can be captured.
func TestNegativeValueDictionaryOverflow(t *testing.T) {
	encodedValue1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	encodedValue2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F273")
	dict := NewValueDictionary()
	// set limit to one storage
	ValueDictionaryLimit = 1
	_, err1 := dict.Encode(encodedValue1)
	if err1 != nil {
		t.Fatalf("Failed to encode a storage key")
	}
	_, err2 := dict.Encode(encodedValue2)
	if err2 == nil {
		t.Fatalf("Failed to report error when adding an exising storage key")
	}
	// reset limit
	ValueDictionaryLimit = math.MaxUint32
}

// TestNegativeValueDictionaryDecodingFailure1 checks whether invalid index for 
// Decode() can be captured (retrieving index 0 on an empty dictionary).
func TestNegativeValueDictionaryDecodingFailure1(t *testing.T) {
	dict := NewValueDictionary()
	_, err := dict.Decode(0)
	if err == nil {
		t.Fatalf("Failed to detect wrong index for Decode()")
	}
}

// TestNegativeValueDictionaryDecodingFailure2 checks whether invalid index for 
// Decode() can be captured (retrieving index MaxUint32 on an empty dictionary).
func TestNegativeValueDictionaryDecodingFailure2(t *testing.T) {
	dict := NewValueDictionary()
	_, err := dict.Decode(math.MaxUint32)
	if err == nil {
		t.Fatalf("Failed to detect wrong index for Decode()")
	}
}

// TestNegativeValueDictionaryReadFailure creates corrupted file and read file as dictionary.
func TestNegativeValueDictionaryReadFailure(t *testing.T) {
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
	rDict := NewValueDictionary()
	err = rDict.Read(filename)
	if err == nil {
		t.Fatalf("Failed to report error when reading a corrupted file")
	}
	os.Remove(filename)
}

// TestPositiveValueDictionaryReadWrite encodes two valuees, write them to file, and 
// read them from file. Check whether the newly created dictionary read from file is identical.
func TestPositiveValueDictionaryReadWrite(t *testing.T) {
	filename := "./test.dict"
	encodedValue1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	encodedValue2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F273")
	wDict := NewValueDictionary()
	idx1, err1 := wDict.Encode(encodedValue1)
	idx2, err2 := wDict.Encode(encodedValue2)
	err := wDict.Write(filename)
	if err != nil {
		t.Fatalf("Failed to write file")
	}
	rDict := NewValueDictionary()
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
