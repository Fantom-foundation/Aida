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

package ethtest

import (
	"encoding/json"
	"math/big"
	"strings"
)

type BigInt struct {
	big.Int
}

func (i *BigInt) Convert() *big.Int {
	if i == nil {
		return new(big.Int)
	}
	return &i.Int
}

func (i *BigInt) UnmarshalJSON(b []byte) error {
	var val string
	err := json.Unmarshal(b, &val)
	if err != nil {
		return err
	}

	i.SetString(strings.TrimPrefix(val, "0x"), 16)

	return nil
}

func (i *BigInt) MarshalJSON() ([]byte, error) {
	str := "0x" + i.Text(16)
	return json.Marshal(str)
}
