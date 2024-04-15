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

package statistics

import (
	"encoding/json"
	"testing"
)

// TestQueuingSimple tests for existence/non-existence of elements.
func TestQueuingSimple(t *testing.T) {
	// create index queue
	queue := NewQueuing[int]()

	// place first element
	queue.Place(0)

	// find first element
	pos := queue.Find(0)
	if pos != 0 {
		t.Fatalf("element cannot be found")
	}

	// unknown element must not be found
	pos = queue.Find(1)
	if pos != -1 {
		t.Fatalf("element must not be found")
	}
}

// TestQueuingSimple1 tests for existence/non-existence of elements.
func TestQueuingSimple1(t *testing.T) {
	// create index queue
	queue := NewQueuing[int]()

	// find first element
	pos := queue.Find(0)
	if pos != -1 {
		t.Fatalf("Queue must be empty")
	}

	// place first element
	queue.Place(0)

	// place second element
	queue.Place(1)

	// find first element
	pos = queue.Find(1)
	if pos != 0 {
		t.Fatalf("first element cannot be found")
	}
	pos = queue.Find(0)
	if pos != 1 {
		t.Fatalf("second element cannot be found")
	}
}

// TestQueuingSimple2 tests for existence/non-existence of elements.
func TestQueuingSimple2(t *testing.T) {
	// create index queue
	queue := NewQueuing[int]()

	// place first element
	for i := 0; i < QueueLen+1; i++ {
		queue.Place(i)
	}

	// find first element
	pos := queue.Find(0)
	if pos != -1 {
		t.Fatalf("first element must not be found")
	}
	pos = queue.Find(1)
	if pos != QueueLen-1 {
		t.Fatalf("second element must be found: %v", pos)
	}
	pos = queue.Find(QueueLen)
	if pos != 0 {
		t.Fatalf("last element must be found")
	}

	queue.Place(QueueLen + 1)

	pos = queue.Find(1)
	if pos != -1 {
		t.Fatalf("second element must not be found")
	}
	pos = queue.Find(2)
	if pos != QueueLen-1 {
		t.Fatalf("third element must be found: %v", pos)
	}
	pos = queue.Find(QueueLen + 1)
	if pos != 0 {
		t.Fatalf("last element must be found")
	}
}

// TestQueuingJSON tests JSON output of distribution.
func TestQueuingJSON(t *testing.T) {
	// create index queue
	queue := NewQueuing[int]()

	// place first element
	for i := 0; i < 300; i++ {
		queue.Place(i)
		// find first element
		pos := queue.Find(i)
		if pos != 0 {
			t.Fatalf("first element must be found")
		}
		pos = queue.Find(i - 1)
		pos = queue.Find(i - 2)
		pos = queue.Find(i - 3)
	}

	// produce distribution in JSON format
	jOut, err := json.Marshal(queue.NewQueuingJSON())
	if err != nil {
		t.Fatalf("Marshalling failed to produce distribution")
	}
	expected := `{"distribution":[0.25125628140703515,0.25041876046901174,0.24958123953098826,0.24874371859296482,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}`
	if string(jOut) != expected {
		t.Fatalf("produced wrong JSON output %v", string(jOut))
	}
}
