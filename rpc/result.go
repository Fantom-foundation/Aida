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

package rpc

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/txcontext"
)

type result struct {
	gasUsed uint64
	result  []byte
	err     error
}

func NewResult(res []byte, err error, gasUsed uint64) txcontext.Result {
	return &result{
		gasUsed: gasUsed,
		result:  res,
		err:     err,
	}
}

func (r *result) GetReceipt() txcontext.Receipt {
	// unused for rpc
	return nil
}

func (r *result) GetRawResult() ([]byte, error) {
	return r.result, r.err
}

func (r *result) GetGasUsed() uint64 {
	return r.gasUsed
}

func (r *result) String() string {
	return fmt.Sprintf("Result: %v\nError: %v\n; Gas Used: %v", string(r.result), r.err, r.gasUsed)
}
