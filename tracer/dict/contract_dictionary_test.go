package dict

import (
	"github.com/ethereum/go-ethereum/common"
	"math"
	"os"
	"testing"
)

// TestPositiveContractDictionarySimple1 encodes an address, and compares whether the decoded
// address is the same, and its index is zero.
func TestPositiveContractDictionarySimple1(t *testing.T) {
	encodedAddr := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	dict := NewContractDictionary()
	idx, err1 := dict.Encode(encodedAddr)
	decodedAddr, err2 := dict.Decode(idx)
	if encodedAddr != decodedAddr || err1 != nil || err2 != nil || idx != 0 {
		t.Fatalf("Encoding/decoding address failed")
	}
}

// TestPositiveContractDictionarySimple2 encodes two addresses, and compares whether
// the decoded addresses are the same, and their dictionary indices are zero and one.
func TestPositiveContractDictionarySimple2(t *testing.T) {
	encodedAddr1 := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	encodedAddr2 := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F273")
	dict := NewContractDictionary()
	idx1, err1 := dict.Encode(encodedAddr1)
	idx2, err2 := dict.Encode(encodedAddr2)
	decodedAddr1, err3 := dict.Decode(idx1)
	decodedAddr2, err4 := dict.Decode(idx2)
	if encodedAddr1 != decodedAddr1 || err1 != nil || err3 != nil || idx1 != 0 {
		t.Fatalf("Encoding/decoding address (1) failed")
	}
	if encodedAddr2 != decodedAddr2 || err2 != nil || err4 != nil || idx2 != 1 {
		t.Fatalf("Encoding/decoding address (2) failed")
	}
}

// TestPositiveContractDictionarySimple3 encodes one address twice and check that its address
// is encoded only once, and its index is zero.
func TestPositiveContractDictionarySimple3(t *testing.T) {
	encodedAddr1 := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	dict := NewContractDictionary()
	idx1, err1 := dict.Encode(encodedAddr1)
	idx2, err2 := dict.Encode(encodedAddr1)
	decodedAddr1, err3 := dict.Decode(idx1)
	decodedAddr2, err4 := dict.Decode(idx2)
	if encodedAddr1 != decodedAddr1 || err1 != nil || err3 != nil || idx1 != 0 {
		t.Fatalf("Encoding/decoding address (1) failed")
	}
	if encodedAddr1 != decodedAddr2 || err2 != nil || err4 != nil || idx2 != 0 {
		t.Fatalf("Encoding/decoding address (2) failed")
	}
}

// TestNegativeContractDictionaryOverflow checks whether dictionary overflows can be captured.
func TestNegativeContractDictionaryOverflow(t *testing.T) {
	encodedAddr1 := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	encodedAddr2 := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F273")
	dict := NewContractDictionary()
	// set limit to one contract
	ContractDictionaryLimit = 1
	_, err1 := dict.Encode(encodedAddr1)
	if err1 != nil {
		t.Fatalf("Failed to encode contract")
	}
	_, err2 := dict.Encode(encodedAddr2)
	if err2 == nil {
		t.Fatalf("Failed to report error when adding an existing address")
	}
	// reset limit
	ContractDictionaryLimit = math.MaxUint32
}

// TestNegativeContractDictionaryDecodingFailure1 checks whether invalid index for Decode()
// can be captured (retrieving index 0 on an empty dictionary).
func TestNegativeContractDictionaryDecodingFailure1(t *testing.T) {
	dict := NewContractDictionary()
	_, err := dict.Decode(0)
	if err == nil {
		t.Fatalf("Failed to detect wrong index for Decode()")
	}
}

// TestNegativeContractDictionaryDecodingFailure2 checks whether invalid index for
// Decode() can be captured (retrieving index MaxUint32 on an empty dictionary).
func TestNegativeContractDictionaryDecodingFailure2(t *testing.T) {
	dict := NewContractDictionary()
	_, err := dict.Decode(math.MaxUint32)
	if err == nil {
		t.Fatalf("Failed to detect wrong index for Decode()")
	}
}

// TestNegativeContractDictionaryReadFailure creates corrupted file and read file as dictionary.
func TestNegativeContractDictionaryReadFailure(t *testing.T) {
	filename := "./test.dict"
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file")
	}
	defer os.Remove(filename)
	// write corrupted entry
	data := []byte("hello")
	if _, err := f.Write(data); err != nil {
		t.Fatalf("Failed to write data")
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("Failed to close file")
	}
	rDict := NewContractDictionary()
	err = rDict.Read(filename)
	if err == nil {
		t.Fatalf("Failed to report error when reading a corrupted file")
	}
}

// TestPositiveContractDictionaryReadWrite encodes two addresses, write them to file,
// and read them from file. Check whether the newly created dictionary (read from
// file) is identical.
func TestPositiveContractDictionaryReadWrite(t *testing.T) {
	filename := "./test.dict"
	encodedAddr1 := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	encodedAddr2 := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F273")
	wDict := NewContractDictionary()
	idx1, err1 := wDict.Encode(encodedAddr1)
	idx2, err2 := wDict.Encode(encodedAddr2)
	err := wDict.Write(filename)
	if err != nil {
		t.Fatalf("Failed to write file")
	}
	defer os.Remove(filename)
	rDict := NewContractDictionary()
	err = rDict.Read(filename)
	if err != nil {
		t.Fatalf("Failed to read file")
	}
	decodedAddr1, err3 := rDict.Decode(idx1)
	decodedAddr2, err4 := rDict.Decode(idx2)
	if encodedAddr1 != decodedAddr1 || err1 != nil || err3 != nil || idx1 != 0 {
		t.Fatalf("Encoding/decoding address (1) failed")
	}
	if encodedAddr2 != decodedAddr2 || err2 != nil || err4 != nil || idx2 != 1 {
		t.Fatalf("Encoding/decoding address (2) failed")
	}
}
