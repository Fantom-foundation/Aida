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

package register

import (
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/utils"
)

func TestIdentity_SameIdIfSameRun(t *testing.T) {
	cfg := &utils.Config{}
	cfg.DbImpl = "carmen"
	cfg.DbVariant = "go-file"
	cfg.CarmenSchema = 5
	cfg.VmImpl = "lfvm"

	timestamp := time.Now().Unix()

	info := &RunIdentity{timestamp, cfg}

	//Same info = same id
	i := info.GetId()
	j := info.GetId()
	if i != j {
		t.Errorf("Same Info but ID doesn't matched: %s vs %s", i, j)
	}

	//Same timestamp, cfg = same id
	info2 := &RunIdentity{timestamp, cfg}
	k := info2.GetId()
	if i != k {
		t.Errorf("Same timestamp, cfg but ID doesn't matched: %s vs %s", i, k)
	}
}

func TestIdentity_DiffIdIfDiffRun(t *testing.T) {
	cfg := &utils.Config{}
	cfg.DbImpl = "carmen"
	cfg.DbVariant = "go-file"
	cfg.CarmenSchema = 5
	cfg.VmImpl = "lfvm"

	cfg2 := &utils.Config{}
	cfg2.DbImpl = "carmen"
	cfg2.DbVariant = "go-file"
	cfg2.CarmenSchema = 3
	cfg2.VmImpl = "geth"

	timestamp := time.Now().Unix()
	timestamp2 := timestamp + 10_000

	info := &RunIdentity{timestamp, cfg}
	info2 := &RunIdentity{timestamp2, cfg}
	info3 := &RunIdentity{timestamp, cfg2}

	//Different timestamp = Different Id
	if info.GetId() == info2.GetId() {
		t.Errorf("Different timestamp but ID still matched: %s vs %s", info.GetId(), info2.GetId())
	}

	//Different cfg = Different Id
	if info.GetId() == info3.GetId() {
		t.Errorf("Different cfg but ID still matched: %s vs %s", info.GetId(), info3.GetId())
	}

	//Different everything = different Id
	if info2.GetId() == info3.GetId() {
		t.Errorf("Different info but ID still matched: %s vs %s", info2.GetId(), info3.GetId())
	}
}

func TestIdentity_OverwriteRunIdWorks(t *testing.T) {
	cfg := &utils.Config{}
	cfg.DbImpl = "carmen"
	cfg.DbVariant = "go-file"
	cfg.CarmenSchema = 5
	cfg.VmImpl = "lfvm"
	cfg.OverwriteRunId = "DummyTest"

	timestamp := time.Now().Unix()

	info := &RunIdentity{timestamp, cfg}

	s := info.GetId()
	if s != cfg.OverwriteRunId {
		t.Errorf("RunId should be overwritten as %s but is %s", s, cfg.OverwriteRunId)
	}
}
