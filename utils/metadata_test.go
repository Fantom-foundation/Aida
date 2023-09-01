package utils

import "testing"

func TestDownloadPatchesJson(t *testing.T) {
	AidaDbRepositoryUrl = AidaDbRepositoryMainnetUrl

	patches, err := DownloadPatchesJson()
	if err != nil {
		t.Fatal(err)
	}

	if len(patches) == 0 {
		t.Fatal("patches.json are empty; are you connected to the internet?")
	}
}

func TestGetPatchFirstBlock_Positive(t *testing.T) {
	AidaDbRepositoryUrl = AidaDbRepositoryMainnetUrl

	patches, err := DownloadPatchesJson()
	if err != nil {
		t.Fatalf("cannot download patches.json; %v", err)
	}

	for _, p := range patches {
		firstBlock, err := getPatchFirstBlock(p.ToBlock)
		if err != nil {
			t.Fatalf("getPatchFirstBlock returned an err; %v", err)
		}

		// returned block needs to match the block in patch
		if firstBlock != p.FromBlock {
			t.Fatalf("first blocks are different; expected: %v, real: %v", firstBlock, p.FromBlock)
		}
	}
}
