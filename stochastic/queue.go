package stochastic

// queueLength sets the length of FIFO queue.
const queueLength = 256

// Queue data structure for a generic FIFO queue.
type Queue[T comparable] struct {
	// queue structure
	top  int            // index of first entry in queue
	rear int            // index of last entry in queue
	data [queueLength]T // queue data

	// counting statistics for find operations
	totalFinds  uint64              // number of total find operations
	failedFinds uint64              // number of failed finds
	posFreq     [queueLength]uint64 // position counters for successful finds
}

// QueueDistribution for JSON output.
type QueueDistribution struct {
	TotalFinds          uint64    // total number of find operations
	FailingProbability  float64   // probability of a failing find operation
	PositionProbability []float64 // probability of a position in the queue
}

// NewQueue creates a new queue.
func NewQueue[T comparable]() Queue[T] {
	return Queue[T]{
		top:         -1,
		rear:        -1,
		data:        [queueLength]T{},
		totalFinds:  0,
		failedFinds: 0,
		posFreq:     [queueLength]uint64{},
	}
}

// Place a new item into queue.
func (q *Queue[T]) Place(item T) {
	// is the queue empty => initialize top/rear
	if q.top == -1 {
		q.top, q.rear = 0, 0
		q.data[q.top] = item
		return
	}

	// put new item into the queue
	q.top = (q.top + 1) % queueLength
	q.data[q.top] = item

	// update rear of queue
	if q.top == q.rear {
		q.rear = (q.rear + 1) % queueLength
	}
}

// Find the index position of the item.
func (q *Queue[T]) Find(item T) int {

	// count total number of find operations
	q.totalFinds++

	// if queue is empty, return -1
	if q.top == -1 {
		q.failedFinds++
		return -1
	}

	// for non-empty queues, find item by iterating from top
	i := q.top
	for {
		// if found, return position in the FIFO queue
		if q.data[i] == item {
			idx := (q.top - i + queueLength) % queueLength
			q.posFreq[idx]++
			return idx
		}

		// if rear of queue reached, return not found
		if i == q.rear {
			q.failedFinds++
			return -1
		}

		// go one element back
		i = (i - 1 + queueLength) % queueLength
	}
}

// ProduceDistribution returns the distribution of the queue
func (q *Queue[T]) ProduceDistribution() QueueDistribution {
	// compute probabilities of queue
	failingProbability := float64(0.0)
	positionProbability := make([]float64, queueLength)
	if q.totalFinds > 0 {
		failingProbability = float64(q.failedFinds) / float64(q.totalFinds)
		if d := q.totalFinds - q.failedFinds; d != 0 {
			for i := 0; i < queueLength; i++ {
				positionProbability[i] = float64(q.posFreq[i]) / float64(d)
			}
		}
	}

	// create and populate new distribution
	return QueueDistribution{
		TotalFinds:          q.totalFinds,
		FailingProbability:  failingProbability,
		PositionProbability: positionProbability,
	}
}
