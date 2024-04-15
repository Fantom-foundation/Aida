// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package aidadb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/utils"
)

func TestAidaDbManager_NoManagerIsCreatedIfPathIsNotProvided(t *testing.T) {
	cfg := &utils.Config{}
	ext := MakeAidaDbManager[any](cfg)

	if _, ok := ext.(extension.NilExtension[any]); !ok {
		t.Errorf("manager is enabled although not set in configuration")
	}
}
