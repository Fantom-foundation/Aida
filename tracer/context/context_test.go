package context

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestContextEncodeContract encodes an address and check the returned index.
func TestContextEncodeContract(t *testing.T) {
	ctx := NewContext()
	encodedAddr := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	if addr := ctx.EncodeContract(encodedAddr); addr != encodedAddr {
		t.Fatalf("Encoding contract failed")
	}
}

// TestContextDecodeContract encodes then decodes an address and
// compares whether the addresses are not the same.
func TestContextDecodeContract(t *testing.T) {
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

// TestContextPrevContract fetches the last used addresses
// after encoding and decoding, then compares whether they match the actual
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

// TestContextEncodeKey encodes a storage key and checks the returned index.
func TestContextEncodeKey(t *testing.T) {
	ctx := NewContext()
	encodedKey := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	if _, idx := ctx.EncodeKey(encodedKey); idx != -1 {
		t.Fatalf("Encoding storage key failed; position: %d", idx)
	}
}

// TestContextDecodeKey encodes then decodes a storage key and compares
// whether the storage keys are not matched.
func TestContextDecodeKey(t *testing.T) {
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

// TestContextReadKeyCache reads storage key from index-cache after
// encoding/decoding new storage key. ReadKeyCache doesn't update top index.
func TestContextReadKeyCache(t *testing.T) {
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

// TestContextLookup reads storage key from index-cache after
// encoding/decoding new storage key. DecodeKeyCache updates top index.
func TestContextDecodeKeyCache(t *testing.T) {
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

// TestContextSnapshot adds a new snapshot pair to the snapshot
// context, then gets the replayed snapshot id from the context.
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
