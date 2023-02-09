package stochastic

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

// Places an access into the access statistics.
func (a *AccessStats[T]) Place(data T) {
	// don't place zero constant into queue/counting stats
	var zeroValue T
	if data == zeroValue {
		return
	}

	// Update counting statistics
	a.cstats.Place(data)

	// Place data into queuing statistics
	a.qstats.Place(data)
}

// Classify an access depending on previous placements.
func (a *AccessStats[T]) Classify(data T) int {
	// check zero value
	var zeroValue T
	if data == zeroValue {
		return zeroValueID
	}
	switch a.qstats.Find(data) {
	case -1:
		// data not found in the queuing statistics
		// => check counting statistics
		if !a.cstats.Exists(data) {
			return newValueID
		} else {
			return randomValueID
		}
	case 0:
		// previous entry
		return previousValueID
	default:
		// data found in queuing statistics
		// but not previously accessed
		return recentValueID
	}
}

// NewAccessStatsJSON produces JSON output for an access statistics.
func (a *AccessStats[T]) NewAccessStatsJSON() AccessStatsJSON {
	return AccessStatsJSON{a.cstats.NewCountingStatsJSON(), a.qstats.NewQueuingStatsJSON()}
}
