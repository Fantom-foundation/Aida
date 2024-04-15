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

package utils

import (
	xmath "github.com/Fantom-foundation/Aida/utils/math"
)

type Interval struct {
	first uint64
	last  uint64
	start uint64
	end   uint64
}

func NewInterval(first, last, interval uint64) *Interval {
	f := first - (first % interval)
	return &Interval{first, last, f, f + interval - 1}
}

func (i *Interval) Start() uint64 {
	return xmath.Max(i.first, i.start)
}

func (i *Interval) End() uint64 {
	return xmath.Min(i.last, i.end)
}

func (i *Interval) Next() *Interval {
	interval := i.end - i.start + 1
	i.start += interval
	i.end += interval
	return i
}
