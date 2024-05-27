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

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestInterval_WithKnownIntervals(t *testing.T) {
	type argument struct {
		first    uint64
		last     uint64
		interval uint64
		random   bool
	}

	type testcase struct {
		args argument
	}

	tests := []testcase{
		{args: argument{0, 5, 1, true}},
		{args: argument{1, 100, 1, false}},
		{args: argument{1, 100, 1, true}},
		{args: argument{1, 100, 10, true}},
		{args: argument{1, 100, 100, true}},
		{args: argument{1, 100, 1000, true}},
	}

	for _, test := range tests {
		name := fmt.Sprintf("Interval Random [%v]", test.args)
		t.Run(name, func(t *testing.T) {
			var r *rand.Rand
			if test.args.random {
				r = rand.New(rand.NewSource(time.Now().UnixNano()))
			} else {
				r = rand.New(rand.NewSource(0))
			}

			i := NewInterval(test.args.first, test.args.last, test.args.interval)
			c := 1

			if i.Start() != test.args.first {
				t.Fatalf("First interval started at %d, should exactly at first %d", i.start, test.args.first)
			}
			if i.end-i.start+1 != test.args.interval {
				t.Fatalf("First interval has incorrect shape %d, should be %d", i.end-i.start+1, test.args.interval)
			}

			for b := test.args.first; b <= test.args.last; b += uint64(1 + r.Intn(3)) {
				for b > i.End() {
					i.Next()
					c += 1

					if i.end-i.start+1 != test.args.interval {
						t.Fatalf("Interval #%d has incorrect shape %d, should be %d", c, i.end-i.start+1, test.args.interval)
					}
				}
			}
		})
	}
}
