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

package dbcomponent

import (
	"fmt"
)

type DbComponent string

const (
	All       DbComponent = "all"
	Substate  DbComponent = "substate"
	Delete    DbComponent = "delete"
	Update    DbComponent = "update"
	StateHash DbComponent = "state-hash"
)

// ParseDbComponent parses string to DbComponent
func ParseDbComponent(s string) (DbComponent, error) {
	switch s {
	case "all":
		return All, nil
	case "substate":
		return Substate, nil
	case "delete":
		return Delete, nil
	case "update":
		return Update, nil
	case "state-hash":
		return StateHash, nil
	default:
		return "", fmt.Errorf("invalid db component: %v. Usage: (\"all\", \"substate\", \"delete\", \"update\", \"state-hash\")", s)
	}
}
