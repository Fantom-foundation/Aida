package statistics

// Access for tracking access classes
type Access[T comparable] struct {
	// counting statistics for data accesses
	cstats Counting[T]

	// queuing statistics for data accesses
	qstats Queuing[T]
}

// AccessJSON is the JSON output for Access.
type AccessJSON struct {
	Counting CountingJSON
	Queuing  QueuingJSON
}

// NewAccess creates a new access.
func NewAccess[T comparable]() Access[T] {
	return Access[T]{NewCounting[T](), NewQueuing[T]()}
}

// Places an access into the access statistics.
func (a *Access[T]) Place(data T) {
	// don't place zero constant into queue/counting stats
	var zeroValue T
	if data == zeroValue {
		return
	}

	// Update counting statistics only if not found in queue
	if a.qstats.Find(data) == -1 {
		a.cstats.Place(data)
	}

	// Place data into queuing statistics
	a.qstats.Place(data)
}

// Classify an access depending on previous placements.
func (a *Access[T]) Classify(data T) int {
	// check zero value
	var zeroValue T
	if data == zeroValue {
		return ZeroValueID
	}
	switch a.qstats.Find(data) {
	case -1:
		// data not found in the queuing statistics
		// => check counting statistics
		if !a.cstats.Exists(data) {
			return NewValueID
		} else {
			return RandomValueID
		}
	case 0:
		// previous entry
		return PreviousValueID
	default:
		// data found in queuing statistics
		// but not previously accessed
		return RecentValueID
	}
}

// NewAccessJSON produces JSON output for an access statistics.
func (a *Access[T]) NewAccessJSON() AccessJSON {
	return AccessJSON{a.cstats.NewCountingJSON(), a.qstats.NewQueuingJSON()}
}
