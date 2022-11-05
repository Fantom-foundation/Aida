package dict

import (
	"github.com/ethereum/go-ethereum/common"
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"testing"
)

// TestDictionaryContextWriteReadEmpty writes and reads an empty dictionary
// context to a directory and validate file names.
func TestDictionaryContextWriteReadEmpty(t *testing.T) {
	DictionaryContextDir = "./test_dictionary_context/"
	want := []string{"code-dictionary.dat", "contract-dictionary.dat",
		"storage-dictionary.dat", "value-dictionary.dat"}
	have := []string{}

	if err := os.Mkdir(DictionaryContextDir, 0700); err != nil {
		t.Fatalf("Failed to create test directory")
	}
	defer os.RemoveAll(DictionaryContextDir)
	ctx1 := NewDictionaryContext()
	ctx1.Write()
	files, err := ioutil.ReadDir(DictionaryContextDir)
	if err != nil {
		t.Fatalf("Dictionary context directory not found. %v", err)
	}
	for _, f := range files {
		have = append(have, f.Name())
	}
	sort.Strings(want)
	sort.Strings(have)
	if !reflect.DeepEqual(want, have) {
		t.Fatalf("Failed to write dictionary context files.\n\twant %v\n\thave %v", want, have)
	}
	ctx2 := ReadDictionaryContext()
	if ctx2 == nil {
		t.Fatalf("Failed to read dictonary context files")
	}
}

// TestDictionaryContextEncodeContract encodes an address and check the returned index.
func TestDictionaryContextEncodeContract(t *testing.T) {
	ctx := NewDictionaryContext()
	encodedAddr := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	if idx := ctx.EncodeContract(encodedAddr); idx != 0 {
		t.Fatalf("Encoding contract failed")
	}
}

// TestDictionaryContextDecodeContract encodes then decodes an address and
// compares whether the addresses are not the same.
func TestDictionaryContextDecodeContract(t *testing.T) {
	ctx := NewDictionaryContext()
	encodedAddr := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	idx := ctx.EncodeContract(encodedAddr)
	if idx != 0 {
		t.Fatalf("Encoding contract failed")
	}
	decodedAddr := ctx.DecodeContract(idx)
	if encodedAddr != decodedAddr {
		t.Fatalf("Decoding contract failed")
	}
}

// TestDictionaryContextLastContractAddress fetches the last used addresses
// after encodeing and decoding, then compares whether they match the actual
// last used contract addresses.
func TestDictionaryContextLastContractAddress(t *testing.T) {
	ctx := NewDictionaryContext()
	encodedAddr1 := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	idx1 := ctx.EncodeContract(encodedAddr1)
	lastAddr := ctx.LastContractAddress()
	if encodedAddr1 != lastAddr {
		t.Fatalf("Failed to get last contract address (1) after encoding")
	}

	encodedAddr2 := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F274")
	idx2 := ctx.EncodeContract(encodedAddr2)
	lastAddr = ctx.LastContractAddress()
	if encodedAddr2 != lastAddr {
		t.Fatalf("Failed to get last contract address (2) after encoding")
	}

	decodedAddr1 := ctx.DecodeContract(idx1)
	lastAddr = ctx.LastContractAddress()
	if decodedAddr1 != lastAddr {
		t.Fatalf("Failed to get last contract address (1) after decoding")
	}

	decodedAddr2 := ctx.DecodeContract(idx2)
	lastAddr = ctx.LastContractAddress()
	if decodedAddr2 != lastAddr {
		t.Fatalf("Failed to get last contract address (2) after decoding")
	}
}

// TestDictionaryContextEncodeStorage encodes a storage key and checks the returned index.
func TestDictionaryContextEncodeStorage(t *testing.T) {
	ctx := NewDictionaryContext()
	encodedKey := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	if idx, _ := ctx.EncodeStorage(encodedKey); idx != 0 {
		t.Fatalf("Encoding storage key failed")
	}
}

// TestDictionaryContextDecodeStorage encodes then decodes a storage key and compares
// whether the storage keys are not matched.
func TestDictionaryContextDecodeStorage(t *testing.T) {
	ctx := NewDictionaryContext()
	encodedKey := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	idx, _ := ctx.EncodeStorage(encodedKey)
	if idx != 0 {
		t.Fatalf("Encoding storage key failed")
	}
	decodedKey := ctx.DecodeStorage(idx)
	if encodedKey != decodedKey {
		t.Fatalf("Decoding storage key failed")
	}
}

// TestDictionaryContextReadStorage reads storage key from index-cache after
// encoding/decoding new storage key. ReadStorage doesn't update top index.
func TestDictionaryContextReadStorage(t *testing.T) {
	ctx := NewDictionaryContext()
	encodedKey1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	idx1, _ := ctx.EncodeStorage(encodedKey1)
	encodedKey2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F274")
	idx2, _ := ctx.EncodeStorage(encodedKey2)

	cachedKey := ctx.ReadStorage(1)
	if encodedKey1 != cachedKey {
		t.Fatalf("Failed to read storage key (1) from index-cache")
	}

	cachedKey = ctx.ReadStorage(0)
	if encodedKey2 != cachedKey {
		t.Fatalf("Failed to read storage key (2) from index-cache")
	}

	decodedKey1 := ctx.DecodeStorage(idx1)
	decodedKey2 := ctx.DecodeStorage(idx2)

	cachedKey = ctx.ReadStorage(1)
	if decodedKey1 != cachedKey {
		t.Fatalf("Failed to read storage key (1) from index-cache")
	}

	cachedKey = ctx.ReadStorage(0)
	if decodedKey2 != cachedKey {
		t.Fatalf("Failed to read storage key (2) from index-cache")
	}
}

// TestDictionaryContextLookup reads storage key from index-cache after
// encoding/decoding new storage key. LookupStorage updates top index.
func TestDictionaryContextLookupStorage(t *testing.T) {
	ctx := NewDictionaryContext()
	encodedKey1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	idx1, _ := ctx.EncodeStorage(encodedKey1)
	encodedKey2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F274")
	idx2, _ := ctx.EncodeStorage(encodedKey2)

	cachedKey := ctx.LookupStorage(1)
	if encodedKey1 != cachedKey {
		t.Fatalf("Failed to lookup storage key (1) from index-cache")
	}

	cachedKey = ctx.LookupStorage(1)
	if encodedKey2 != cachedKey {
		t.Fatalf("Failed to lookup storage key (2) from index-cache")
	}

	decodedKey1 := ctx.DecodeStorage(idx1)
	decodedKey2 := ctx.DecodeStorage(idx2)

	cachedKey = ctx.LookupStorage(1)
	if decodedKey1 != cachedKey {
		t.Fatalf("Failed to lookup storage key (1) from index-cache")
	}

	cachedKey = ctx.LookupStorage(1)
	if decodedKey2 != cachedKey {
		t.Fatalf("Failed to lookup storage key (2) from index-cache")
	}
}

// TestDictionaryContextEncodeValue encodes a value and compares the returned index.
func TestDictionaryContextEncodeValue(t *testing.T) {
	ctx := NewDictionaryContext()
	encodedValue := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	if idx := ctx.EncodeValue(encodedValue); idx != 0 {
		t.Fatalf("Encoding value failed")
	}
}

// TestDictionaryContextDecodeValue encodes then decodes a storage value and
// compares whether the values are the same.
func TestDictionaryContextDecodeValue(t *testing.T) {
	ctx := NewDictionaryContext()
	encodedValue := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	idx := ctx.EncodeValue(encodedValue)
	if idx != 0 {
		t.Fatalf("Encoding value failed")
	}
	decodedValue := ctx.DecodeValue(idx)
	if encodedValue != decodedValue {
		t.Fatalf("Decoding value failed")
	}
}

// TestDictionaryContextSnapshot adds a new snapshot pair to the snapshot
// dictionary, then gets the replayed snapshot id from the dictionary.
func TestDictionaryContextSnapshot(t *testing.T) {
	ctx := NewDictionaryContext()
	recordedID := int32(39)
	replayedID1 := int32(50)
	ctx.AddSnapshot(recordedID, replayedID1)
	if ctx.GetSnapshot(recordedID) != replayedID1 {
		t.Fatalf("Failed to retrieve snapshot id")
	}
	replayedID2 := int32(8)
	ctx.AddSnapshot(recordedID, replayedID2)
	if ctx.GetSnapshot(recordedID) != replayedID2 {
		t.Fatalf("Failed to retrieve snapshot id")
	}
}

// TestDictionaryContextEncodeCode encodes byte-code to code dictionary.
func TestDictionaryContextEncodeCode(t *testing.T) {
	ctx := NewDictionaryContext()
	encodedCode := []byte{0x99, 0xe0, 0x5, 0xed, 0xce, 0xdf, 0xf5}
	idx := ctx.EncodeCode(encodedCode)
	if idx != 0 {
		t.Fatalf("Encoding byte-code failed")
	}
}

// TestDictionaryContextDecodeCode encodes then decodes byte-code, and compares
// whether the byte-code arrays are matches.
func TestDictionaryContextDecodeCode(t *testing.T) {
	ctx := NewDictionaryContext()
	encodedCode := []byte{0x99, 0xe0, 0x5, 0xed, 0xce, 0xdf, 0xf5}
	idx := ctx.EncodeCode(encodedCode)
	if idx != 0 {
		t.Fatalf("Encoding byte-code failed")
	}
	decodedCode := ctx.DecodeCode(idx)
	if !reflect.DeepEqual(encodedCode, decodedCode) {
		t.Fatalf("Decoding byte-code failed")
	}
}
