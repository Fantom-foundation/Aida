// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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
