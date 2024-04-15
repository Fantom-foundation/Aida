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

package generator

import (
	"errors"

	"github.com/Fantom-foundation/Aida/stochastic/statistics"
)

// IndirectAccess data structure for random access indices permitting deletion without reuse.
type IndirectAccess struct {
	randAcc *RandomAccess

	// translation table for converting compact index space to sparse
	// permitting index deletion without later reuse.
	translation []int64

	// counter for introducing new index values
	ctr int64
}

// NewIndirectAccess creates a new indirect index access-generator.
func NewIndirectAccess(ra *RandomAccess) *IndirectAccess {
	t := make([]int64, ra.numElem)
	for i := int64(0); i < ra.numElem; i++ {
		t[i] = i + 1 // shifted by one because of zero value
	}
	return &IndirectAccess{
		randAcc:     ra,
		ctr:         ra.numElem,
		translation: t,
	}
}

// NextIndex returns the next index value based on the provided class.
func (a *IndirectAccess) NextIndex(class int) int64 {
	v := a.randAcc.NextIndex(class)
	if v == -1 {
		return -1
	} else if class == statistics.ZeroValueID {
		return v
	} else if class == statistics.NewValueID {
		if v != a.randAcc.numElem {
			panic("unexpected nextIndex result.")
		}
		a.ctr++
		v := a.ctr
		a.translation = append(a.translation, v)
		return v
	} else {
		return a.translation[v-1]
	}
}

// DeleteIndex deletes an indirect index.
func (a *IndirectAccess) DeleteIndex(k int64) error {
	if k == 0 {
		return nil
	}

	// find index in translation table
	i := a.findIndex(k)
	if i < 0 {
		return errors.New("index not found")
	}

	// delete index i from the translation table and the random access generator.
	a.translation = append(a.translation[:i], a.translation[i+1:]...)
	if err := a.randAcc.DeleteIndex(i); err != nil {
		return err
	}

	return nil
}

// findIndex finds the index in the translation table for a given index k.
func (a *IndirectAccess) findIndex(k int64) int64 {
	for i := int64(0); i < int64(len(a.translation)); i++ {
		if a.translation[i] == k {
			return i
		}
	}
	return -1
}

// NumElem returns the number of indexes
func (a *IndirectAccess) NumElem() int64 {
	return a.randAcc.numElem
}
