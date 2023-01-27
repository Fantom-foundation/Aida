package stochastic

// Classifications of entries
const (
	randomEntry   = iota // random access of previoulsy known entry / default class
	recentEntry          // recently accessed entry for a defined time-window
	previousEntry        // previously accessed entry
	newEntry             // never accessed entry

	numClasses
)

// Classifications as strings
// NB: Must follow order as defined above
var classText = []string{"random", "recent", "previous", "new"}

// AccessStats for tracking access classes
type AccessStats[T comparable] struct {
	// counting statistics for data
	stats Statistics[T]

	// queue of recent data accesses
	queue Queue[T]
}

// Acccess Distribution for JSON output
type AccessDistribution struct {
	Distribution StatisticsDistribution
	Queue        QueueDistribution
}

// NewAccessStats creates a new access.
func NewAccessStats[T comparable]() AccessStats[T] {
	return AccessStats[T]{NewStatistics[T](), NewQueue[T]()}
}

// Place new into access.
func (a *AccessStats[T]) Put(data T) {
	// Update distribution
	a.stats.Count(data)

	// Place data into queue
	a.queue.Place(data)
}

// Classify the access depending on previously placement.
func (a *AccessStats[T]) Classify(data T) int {
	switch a.queue.Find(data) {
	case -1:
		// data not found in queue
		if !a.stats.Exists(data) {
			return newEntry
		} else {
			return randomEntry
		}
	case 0:
		// previously accessed
		return previousEntry
	default:
		// data found in queue but not previously accessed
		return recentEntry
	}
}

// ProduceDistribution for JSON output
func (a *AccessStats[T]) ProduceDistribution() AccessDistribution {
	return AccessDistribution{a.stats.ProduceDistribution(), a.queue.ProduceDistribution()}
}
