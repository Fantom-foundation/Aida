package statistics

// IDs for argument classes
const (
	NoArgID         = iota // default label (for no argument)
	ZeroValueID            // zero value access
	NewValueID             // newly occurring value access
	PreviousValueID        // value that was previously accessed
	RecentValueID          // value that recently accessed (time-window is fixed to statistics.QueueLen)
	RandomValueID          // random access (everything else)

	NumClasses
)

// number of points on the ecdf
const NumDistributionPoints = 100

// QueueLen sets the length of queuing statistics.
// NB: must be greater than one.
const QueueLen = 32
