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

// Queuing data structure for a generic FIFO queue.
type Queuing[T comparable] struct {
	// queue structure
	top  int         // index of first entry in queue
	rear int         // index of last entry in queue
	data [QueueLen]T // queue data

	// counting statistics for queue
	// (counter for each position counting successful finds)
	freq [QueueLen]uint64
}

// QueuingJSON is the JSON output for queuing statistics.
type QueuingJSON struct {
	// probability of a position in the queue
	Distribution []float64 `json:"distribution"`
}

// NewQueuing creates a new queue.
func NewQueuing[T comparable]() Queuing[T] {
	return Queuing[T]{
		top:  -1,
		rear: -1,
		data: [QueueLen]T{},
		freq: [QueueLen]uint64{},
	}
}

// Place a new item into the queue.
func (q *Queuing[T]) Place(item T) {
	// is the queue empty => initialize top/rear
	if q.top == -1 {
		q.top, q.rear = 0, 0
		q.data[q.top] = item
		return
	}

	// put new item into the queue
	q.top = (q.top + 1) % QueueLen
	q.data[q.top] = item

	// update rear of queue
	if q.top == q.rear {
		q.rear = (q.rear + 1) % QueueLen
	}
}

// Find the index position of an item.
func (q *Queuing[T]) Find(item T) int {

	// if queue is empty, return -1
	if q.top == -1 {
		return -1
	}

	// for non-empty queues, find item by iterating from top
	i := q.top
	for {
		// if found, return position in the FIFO queue
		if q.data[i] == item {
			idx := (q.top - i + QueueLen) % QueueLen
			q.freq[idx]++
			return idx
		}

		// if rear of queue reached, return not found
		if i == q.rear {
			return -1
		}

		// go one element back
		i = (i - 1 + QueueLen) % QueueLen
	}
}

// NewQueuingJSON produces JSON output for for a queuing statistics.
func (q *Queuing[T]) NewQueuingJSON() QueuingJSON {
	// Compute total frequency over all positions
	total := uint64(0)
	for i := 0; i < QueueLen; i++ {
		total += q.freq[i]
	}

	// compute index probabilities
	dist := make([]float64, QueueLen)
	if total > 0 {
		for i := 0; i < QueueLen; i++ {
			dist[i] = float64(q.freq[i]) / float64(total)
		}
	}

	// populate new index probabilities
	return QueuingJSON{
		Distribution: dist,
	}
}
