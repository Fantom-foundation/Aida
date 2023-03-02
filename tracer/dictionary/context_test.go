package dictionary

import (
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestContextWriteReadEmpty writes and reads an empty dictionary
// context to a directory and validate file names.
func TestContextWriteReadEmpty(t *testing.T) {
	ContextDir = "./test_dictionary_context/"
	want := []string{"code-dictionary.dat", "contract-dictionary.dat",
		"storage-dictionary.dat"}
	have := []string{}

	if err := os.Mkdir(ContextDir, 0700); err != nil {
		t.Fatalf("Failed to create test directory")
	}
	defer os.RemoveAll(ContextDir)
	ctx1 := NewContext()
	ctx1.Write()
	files, err := ioutil.ReadDir(ContextDir)
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
	ctx2 := ReadContext()
	if ctx2 == nil {
		t.Fatalf("Failed to read dictonary context files")
	}
}

// TestContextEncodeContract encodes an address and check the returned index.
func TestContextEncodeContract(t *testing.T) {
	ctx := NewContext()
	encodedAddr := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	if idx := ctx.EncodeContract(encodedAddr); idx != 0 {
		t.Fatalf("Encoding contract failed")
	}
}

// TestContextDecodeContract encodes then decodes an address and
// compares whether the addresses are not the same.
func TestContextDecodeContract(t *testing.T) {
	ctx := NewContext()
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

// TestContextPrevContract fetches the last used addresses
// after encodeing and decoding, then compares whether they match the actual
// last used contract addresses.
func TestContextPrevContract(t *testing.T) {
	ctx := NewContext()
	encodedAddr1 := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	idx1 := ctx.EncodeContract(encodedAddr1)
	lastAddr := ctx.PrevContract()
	if encodedAddr1 != lastAddr {
		t.Fatalf("Failed to get last contract address (1) after encoding")
	}

	encodedAddr2 := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F274")
	idx2 := ctx.EncodeContract(encodedAddr2)
	lastAddr = ctx.PrevContract()
	if encodedAddr2 != lastAddr {
		t.Fatalf("Failed to get last contract address (2) after encoding")
	}

	decodedAddr1 := ctx.DecodeContract(idx1)
	lastAddr = ctx.PrevContract()
	if decodedAddr1 != lastAddr {
		t.Fatalf("Failed to get last contract address (1) after decoding")
	}

	decodedAddr2 := ctx.DecodeContract(idx2)
	lastAddr = ctx.PrevContract()
	if decodedAddr2 != lastAddr {
		t.Fatalf("Failed to get last contract address (2) after decoding")
	}
}

// TestContextEncodeStorage encodes a storage key and checks the returned index.
func TestContextEncodeStorage(t *testing.T) {
	ctx := NewContext()
	encodedKey := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	if idx, _ := ctx.EncodeStorage(encodedKey); idx != 0 {
		t.Fatalf("Encoding storage key failed")
	}
}

// TestContextDecodeStorage encodes then decodes a storage key and compares
// whether the storage keys are not matched.
func TestContextDecodeStorage(t *testing.T) {
	ctx := NewContext()
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

// TestContextReadStorageCache reads storage key from index-cache after
// encoding/decoding new storage key. ReadStorageCache doesn't update top index.
func TestContextReadStorageCache(t *testing.T) {
	ctx := NewContext()
	encodedKey1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	idx1, _ := ctx.EncodeStorage(encodedKey1)
	encodedKey2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F274")
	idx2, _ := ctx.EncodeStorage(encodedKey2)

	cachedKey := ctx.ReadStorageCache(1)
	if encodedKey1 != cachedKey {
		t.Fatalf("Failed to read storage key (1) from index-cache")
	}

	cachedKey = ctx.ReadStorageCache(0)
	if encodedKey2 != cachedKey {
		t.Fatalf("Failed to read storage key (2) from index-cache")
	}

	decodedKey1 := ctx.DecodeStorage(idx1)
	decodedKey2 := ctx.DecodeStorage(idx2)

	cachedKey = ctx.ReadStorageCache(1)
	if decodedKey1 != cachedKey {
		t.Fatalf("Failed to read storage key (1) from index-cache")
	}

	cachedKey = ctx.ReadStorageCache(0)
	if decodedKey2 != cachedKey {
		t.Fatalf("Failed to read storage key (2) from index-cache")
	}
}

// TestContextLookup reads storage key from index-cache after
// encoding/decoding new storage key. DecodeStorageCache updates top index.
func TestContextDecodeStorageCache(t *testing.T) {
	ctx := NewContext()
	encodedKey1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	idx1, _ := ctx.EncodeStorage(encodedKey1)
	encodedKey2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F274")
	idx2, _ := ctx.EncodeStorage(encodedKey2)

	cachedKey := ctx.DecodeStorageCache(1)
	if encodedKey1 != cachedKey {
		t.Fatalf("Failed to lookup storage key (1) from index-cache")
	}

	cachedKey = ctx.DecodeStorageCache(1)
	if encodedKey2 != cachedKey {
		t.Fatalf("Failed to lookup storage key (2) from index-cache")
	}

	decodedKey1 := ctx.DecodeStorage(idx1)
	decodedKey2 := ctx.DecodeStorage(idx2)

	cachedKey = ctx.DecodeStorageCache(1)
	if decodedKey1 != cachedKey {
		t.Fatalf("Failed to lookup storage key (1) from index-cache")
	}

	cachedKey = ctx.DecodeStorageCache(1)
	if decodedKey2 != cachedKey {
		t.Fatalf("Failed to lookup storage key (2) from index-cache")
	}
}

// TestContextSnapshot adds a new snapshot pair to the snapshot
// dictionary, then gets the replayed snapshot id from the dictionary.
func TestContextSnapshot(t *testing.T) {
	ctx := NewContext()
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

// TestContextEncodeCode encodes byte-code to code dictionary.
func TestContextEncodeCode(t *testing.T) {
	ctx := NewContext()
	encodedCode := []byte{0x99, 0xe0, 0x5, 0xed, 0xce, 0xdf, 0xf5}
	idx := ctx.EncodeCode(encodedCode)
	if idx != 0 {
		t.Fatalf("Encoding byte-code failed")
	}
}

// TestContextDecodeCode encodes then decodes byte-code, and compares
// whether the byte-code arrays are matches.
func TestContextDecodeCode(t *testing.T) {
	ctx := NewContext()
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
