package dict

import (
	"github.com/ethereum/go-ethereum/common"
	"math"
	"os"
	"testing"
)

// TestStorageDictionarySimple1 encodes an address, and compares whether
// the decoded address is the same, and its index is zero.
func TestStorageDictionarySimple1(t *testing.T) {
	encodedAddr := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	dict := NewStorageDictionary()
	idx, err1 := dict.Encode(encodedAddr)
	decodedAddr, err2 := dict.Decode(idx)
	if encodedAddr != decodedAddr || err1 != nil || err2 != nil || idx != 0 {
		t.Fatalf("Encoding/Decoding failed")
	}
}

// TestStorageDictionarySimple2 encodes two addresses, and compares whether
// the decoded addresses are the same, and their dictionary indices are zero and one.
func TestStorageDictionarySimple2(t *testing.T) {
	encodedAddr1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	encodedAddr2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F273")
	dict := NewStorageDictionary()
	idx1, err1 := dict.Encode(encodedAddr1)
	idx2, err2 := dict.Encode(encodedAddr2)
	decodedAddr1, err3 := dict.Decode(idx1)
	decodedAddr2, err4 := dict.Decode(idx2)
	if encodedAddr1 != decodedAddr1 || err1 != nil || err3 != nil || idx1 != 0 {
		t.Fatalf("Encoding/Decoding failed")
	}
	if encodedAddr2 != decodedAddr2 || err2 != nil || err4 != nil || idx2 != 1 {
		t.Fatalf("Encoding/Decoding failed")
	}
}

// TestStorageDictionarySimple3 encodes one address twice and check that its address
// is encoded only once, and its index is zero.
func TestStorageDictionarySimple3(t *testing.T) {
	encodedAddr1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	dict := NewStorageDictionary()
	idx1, err1 := dict.Encode(encodedAddr1)
	idx2, err2 := dict.Encode(encodedAddr1)
	decodedAddr1, err3 := dict.Decode(idx1)
	decodedAddr2, err4 := dict.Decode(idx2)
	if encodedAddr1 != decodedAddr1 || err1 != nil || err3 != nil || idx1 != 0 {
		t.Fatalf("Encoding/Decoding failed")
	}
	if encodedAddr1 != decodedAddr2 || err2 != nil || err4 != nil || idx2 != 0 {
		t.Fatalf("Encoding/Decoding failed")
	}
}

// TestStorageDictionaryOverflow checks whether dictionary overflows can be captured.
func TestStorageDictionaryOverflow(t *testing.T) {
	encodedAddr1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	encodedAddr2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F273")
	dict := NewStorageDictionary()
	// set limit to one storage
	StorageDictionaryLimit = 1
	_, err1 := dict.Encode(encodedAddr1)
	if err1 != nil {
		t.Fatalf("Failed to encode a storage key")
	}
	_, err2 := dict.Encode(encodedAddr2)
	if err2 == nil {
		t.Fatalf("Failed to report error when adding an exising storage key")
	}
	// reset limit
	StorageDictionaryLimit = math.MaxUint32
}

// TestStorageDictionaryDecodingFailure1 checks whether invalid index for Decode()
// can be captured (retrieving index 0 on an empty dictionary).
func TestStorageDictionaryDecodingFailure1(t *testing.T) {
	dict := NewStorageDictionary()
	_, err := dict.Decode(0)
	if err == nil {
		t.Fatalf("Failed to detect wrong index for Decode()")
	}
}

// TestStorageDictionaryDecodingFailure2 checks whether invalid
// index for Decode() can be captured (retrieving index MaxUint32 on an
// empty dictionary).
func TestStorageDictionaryDecodingFailure2(t *testing.T) {
	dict := NewStorageDictionary()
	_, err := dict.Decode(math.MaxUint32)
	if err == nil {
		t.Fatalf("Failed to detect wrong index for Decode()")
	}
}

// TestStorageDictionaryReadFailure creates corrupted file and
// reads file as dictionary.
func TestStorageDictionaryReadFailure(t *testing.T) {
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
	rDict := NewStorageDictionary()
	err = rDict.Read(filename)
	if err == nil {
		t.Fatalf("Failed to report error when reading a corrupted file")
	}
	os.Remove(filename)
}

// TestStorageDictionaryReadWrite encodes two addresses, writes them to file,
// and reads them from file. Check whether the newly created dictionary read from
// file is identical.
func TestStorageDictionaryReadWrite(t *testing.T) {
	filename := "./test.dict"
	encodedAddr1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	encodedAddr2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F273")
	wDict := NewStorageDictionary()
	idx1, err1 := wDict.Encode(encodedAddr1)
	idx2, err2 := wDict.Encode(encodedAddr2)
	err := wDict.Write(filename)
	if err != nil {
		t.Fatalf("Failed to write file")
	}
	rDict := NewStorageDictionary()
	err = rDict.Read(filename)
	if err != nil {
		t.Fatalf("Failed to read file")
	}
	decodedAddr1, err3 := rDict.Decode(idx1)
	decodedAddr2, err4 := rDict.Decode(idx2)
	if encodedAddr1 != decodedAddr1 || err1 != nil || err3 != nil || idx1 != 0 {
		t.Fatalf("Encoding/Decoding failed")
	}
	if encodedAddr2 != decodedAddr2 || err2 != nil || err4 != nil || idx2 != 1 {
		t.Fatalf("Encoding/Decoding failed")
	}
	os.Remove(filename)
}
