package stochastic

// Classifications of entries
const (
	randomEntry   = iota // random access of previoulsy known entry / default class
	recentEntry          // recently accessed entry for a defined time-window
	previousEntry        // previously accessed entry
	newEntry             // never accessed entry

	numClasses
)
const defaultEntry = randomEntry // default classifier if no simulation argument exists

// AccessStats for tracking access classes
type AccessStats[T comparable] struct {
	// counting statistics for data
	stats *Statistics[T]

	// queue of recent data accesses
	queue *Queue[T]
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

// Write access statistics to file.
func (a *AccessStats[T]) WriteStats(prefix string) {
	a.stats.WriteStats(prefix + "_stats.csv")
	a.queue.WriteStats(prefix + "_queue.csv")
}
