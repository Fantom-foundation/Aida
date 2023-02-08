package stochastic

// qstatsLen sets the length of queuing statistics.
const qstatsLen = 32

// QueuingStats data structure for a generic FIFO queue.
type QueuingStats[T comparable] struct {
	// queue structure
	top  int          // index of first entry in queue
	rear int          // index of last entry in queue
	data [qstatsLen]T // queue data

	// counting statistics for queue
	// (counter for each position counting successful finds)
	freq [qstatsLen]uint64
}

// QueuingStatsJSON is the JSON output for queuing statistics.
type QueuingStatsJSON struct {
	// probability of a position in the queue
	Distribution []float64 `json:"distribution"`
}

// NewQueuingStats creates a new queue.
func NewQueuingStats[T comparable]() QueuingStats[T] {
	return QueuingStats[T]{
		top:  -1,
		rear: -1,
		data: [qstatsLen]T{},
		freq: [qstatsLen]uint64{},
	}
}

// Place a new item into the queue.
func (q *QueuingStats[T]) Place(item T) {
	// is the queue empty => initialize top/rear
	if q.top == -1 {
		q.top, q.rear = 0, 0
		q.data[q.top] = item
		return
	}

	// put new item into the queue
	q.top = (q.top + 1) % qstatsLen
	q.data[q.top] = item

	// update rear of queue
	if q.top == q.rear {
		q.rear = (q.rear + 1) % qstatsLen
	}
}

// Find the index position of an item.
func (q *QueuingStats[T]) Find(item T) int {

	// if queue is empty, return -1
	if q.top == -1 {
		return -1
	}

	// for non-empty queues, find item by iterating from top
	i := q.top
	for {
		// if found, return position in the FIFO queue
		if q.data[i] == item {
			idx := (q.top - i + qstatsLen) % qstatsLen
			q.freq[idx]++
			return idx
		}

		// if rear of queue reached, return not found
		if i == q.rear {
			return -1
		}

		// go one element back
		i = (i - 1 + qstatsLen) % qstatsLen
	}
}

// NewQueuingStatsJSON produces JSON output for for a queuing statistics.
func (q *QueuingStats[T]) NewQueuingStatsJSON() QueuingStatsJSON {
	// Compute total frequency over all positions
	total := uint64(0)
	for i := 0; i < qstatsLen; i++ {
		total += q.freq[i]
	}

	// compute index probabilities
	dist := make([]float64, qstatsLen)
	if total > 0 {
		for i := 0; i < qstatsLen; i++ {
			dist[i] = float64(q.freq[i]) / float64(total)
		}
	}

	// populate new index probabilities
	return QueuingStatsJSON{
		Distribution: dist,
	}
}
