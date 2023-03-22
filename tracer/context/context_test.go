package context

import (
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestDictionaryContextWriteReadEmpty writes and reads an empty context
// context to a directory and validate file names.
func TestDictionaryContextWriteReadEmpty(t *testing.T) {
	ContextDir = "./test_context_context/"
	want := []string{"code-context.dat"}
	have := []string{}

	if err := os.Mkdir(ContextDir, 0700); err != nil {
		t.Fatalf("Failed to create test directory")
	}

	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove test directory: %v", err)
		}
	}(ContextDir)

	ctx1 := NewContext()
	ctx1.Write()
	files, err := os.ReadDir(ContextDir)
	if err != nil {
		t.Fatalf("Dictionary context directory not found. %v", err)
	}
	for _, f := range files {
		have = append(have, f.Name())
	}
	sort.Strings(want)
	sort.Strings(have)
	if !reflect.DeepEqual(want, have) {
		t.Fatalf("Failed to write context context files.\n\twant %v\n\thave %v", want, have)
	}
	ctx2 := ReadContext()
	if ctx2 == nil {
		t.Fatalf("Failed to read dictonary context files")
	}
}

// TestDictionaryContextEncodeContract encodes an address and check the returned index.
func TestDictionaryContextEncodeContract(t *testing.T) {
	ctx := NewContext()
	encodedAddr := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	if addr := ctx.EncodeContract(encodedAddr); addr != encodedAddr {
		t.Fatalf("Encoding contract failed")
	}
}

// TestDictionaryContextDecodeContract encodes then decodes an address and
// compares whether the addresses are not the same.
func TestDictionaryContextDecodeContract(t *testing.T) {
	ctx := NewContext()
	encodedAddr := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	if addr := ctx.EncodeContract(encodedAddr); addr != encodedAddr {
		t.Fatalf("Encoding contract failed")
	}
	decodedAddr := ctx.DecodeContract(encodedAddr)
	if encodedAddr != decodedAddr {
		t.Fatalf("Decoding contract failed")
	}
}

// TestDictionaryContextPrevContract fetches the last used addresses
// after encodeing and decoding, then compares whether they match the actual
// last used contract addresses.
func TestDictionaryContextPrevContract(t *testing.T) {
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

// TestDictionaryContextEncodeKey encodes a storage key and checks the returned index.
func TestDictionaryContextEncodeKey(t *testing.T) {
	ctx := NewContext()
	encodedKey := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	if _, idx := ctx.EncodeKey(encodedKey); idx != -1 {
		t.Fatalf("Encoding storage key failed; position: %d", idx)
	}
}

// TestDictionaryContextDecodeKey encodes then decodes a storage key and compares
// whether the storage keys are not matched.
func TestDictionaryContextDecodeKey(t *testing.T) {
	ctx := NewContext()
	encodedKey := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	_, idx := ctx.EncodeKey(encodedKey)
	if idx != -1 {
		t.Fatalf("Encoding storage key failed; position: %d", idx)
	}
	decodedKey := ctx.DecodeKey(encodedKey)
	if encodedKey != decodedKey {
		t.Fatalf("Decoding storage key failed")
	}
}

// TestDictionaryContextReadKeyCache reads storage key from index-cache after
// encoding/decoding new storage key. ReadKeyCache doesn't update top index.
func TestDictionaryContextReadKeyCache(t *testing.T) {
	ctx := NewContext()
	encodedKey1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	idx1, _ := ctx.EncodeKey(encodedKey1)
	encodedKey2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F274")
	idx2, _ := ctx.EncodeKey(encodedKey2)

	cachedKey := ctx.ReadKeyCache(1)
	if encodedKey1 != cachedKey {
		t.Fatalf("Failed to read storage key (1) from index-cache")
	}

	cachedKey = ctx.ReadKeyCache(0)
	if encodedKey2 != cachedKey {
		t.Fatalf("Failed to read storage key (2) from index-cache")
	}

	decodedKey1 := ctx.DecodeKey(idx1)
	decodedKey2 := ctx.DecodeKey(idx2)

	cachedKey = ctx.ReadKeyCache(1)
	if decodedKey1 != cachedKey {
		t.Fatalf("Failed to read storage key (1) from index-cache")
	}

	cachedKey = ctx.ReadKeyCache(0)
	if decodedKey2 != cachedKey {
		t.Fatalf("Failed to read storage key (2) from index-cache")
	}
}

// TestDictionaryContextLookup reads storage key from index-cache after
// encoding/decoding new storage key. DecodeKeyCache updates top index.
func TestDictionaryContextDecodeKeyCache(t *testing.T) {
	ctx := NewContext()
	encodedKey1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	idx1, _ := ctx.EncodeKey(encodedKey1)
	encodedKey2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F274")
	idx2, _ := ctx.EncodeKey(encodedKey2)

	cachedKey := ctx.DecodeKeyCache(1)
	if encodedKey1 != cachedKey {
		t.Fatalf("Failed to lookup storage key (1) from index-cache")
	}

	cachedKey = ctx.DecodeKeyCache(1)
	if encodedKey2 != cachedKey {
		t.Fatalf("Failed to lookup storage key (2) from index-cache")
	}

	decodedKey1 := ctx.DecodeKey(idx1)
	decodedKey2 := ctx.DecodeKey(idx2)

	cachedKey = ctx.DecodeKeyCache(1)
	if decodedKey1 != cachedKey {
		t.Fatalf("Failed to lookup storage key (1) from index-cache")
	}

	cachedKey = ctx.DecodeKeyCache(1)
	if decodedKey2 != cachedKey {
		t.Fatalf("Failed to lookup storage key (2) from index-cache")
	}
}

// TestDictionaryContextSnapshot adds a new snapshot pair to the snapshot
// context, then gets the replayed snapshot id from the context.
func TestDictionaryContextSnapshot(t *testing.T) {
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

// TestDictionaryContextEncodeCode encodes byte-code to code context.
func TestDictionaryContextEncodeCode(t *testing.T) {
	ctx := NewContext()
	encodedCode := []byte{0x99, 0xe0, 0x5, 0xed, 0xce, 0xdf, 0xf5}
	idx := ctx.EncodeCode(encodedCode)
	if idx != 0 {
		t.Fatalf("Encoding byte-code failed")
	}
}

// TestDictionaryContextDecodeCode encodes then decodes byte-code, and compares
// whether the byte-code arrays are matches.
func TestDictionaryContextDecodeCode(t *testing.T) {
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
