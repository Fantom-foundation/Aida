package stochastic

// Classifications of data entries in an access statistics
const (
	noArgEntry    = iota // default label (for no argument)
	zeroEntry            // zero value access
	newEntry             // newly occurring value access
	previousEntry        // value that was previously accessed
	recentEntry          // value that recently accessed (time-window is fixed to qstatsLen)
	randomEntry          // random access (everything else)

	numClasses
)

// classTest maps ids to character code for a classification (NB: Must follow order as defined above)
var classText = []string{
	"",  // no argument entry
	"z", // zero value entry
	"n", // new entry
	"p", // previous entry
	"q", // recent entry
	"r", // random entry
}

// AccessStats for tracking access classes
type AccessStats[T comparable] struct {
	// counting statistics for data accesses
	cstats CountingStats[T]

	// queuing statistics for data accesses
	qstats QueuingStats[T]
}

// AccessStatsJSON is the JSON output for AccessStats.
type AccessStatsJSON struct {
	CountingStats CountingStatsJSON
	QueuingStats  QueuingStatsJSON
}

// NewAccessStats creates a new access.
func NewAccessStats[T comparable]() AccessStats[T] {
	return AccessStats[T]{NewCountingStats[T](), NewQueuingStats[T]()}
}

// Put an access into the access statistics.
func (a *AccessStats[T]) Put(data T) {
	// don't place zero constant into queue/counting stats
	var zeroValue T
	if data == zeroValue {
		return
	}

	// Update counting statistics
	a.cstats.Count(data)

	// Place data into queuing statistics
	a.qstats.Place(data)
}

// Classify an access depending on previous placements.
func (a *AccessStats[T]) Classify(data T) int {
	// check zero value
	var zeroValue T
	if data == zeroValue {
		return zeroEntry
	}
	switch a.qstats.Find(data) {
	case -1:
		// data not found in the queuing statistics
		// => check counting statistics
		if !a.cstats.Exists(data) {
			return newEntry
		} else {
			return randomEntry
		}
	case 0:
		// previous entry
		return previousEntry
	default:
		// data found in queuing statistics
		// but not previously accessed
		return recentEntry
	}
}

// NewAccessStatsJSON produces JSON output for an access statistics.
func (a *AccessStats[T]) NewAccessStatsJSON() AccessStatsJSON {
	return AccessStatsJSON{a.cstats.NewCountingStatsJSON(), a.qstats.NewQueuingStatsJSON()}
}
