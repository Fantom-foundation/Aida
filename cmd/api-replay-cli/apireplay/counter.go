package apireplay

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/op/go-logging"
)

// requestCounter serves as a counting mechanism for statistics logging.
// Since each ReplayExecutor runs in own thread, need another thread for counting how many requests we executed,
// so we do not slow down the execution process with mutex
type requestCounter struct {
	closed  chan any
	ticker  *time.Ticker
	input   chan requestLog
	start   time.Time
	builder *strings.Builder
	total   uint64
	log     *logging.Logger
	wg      *sync.WaitGroup
	stats   map[reqLogType]map[string]uint64
}

// reqLogType represents what happened to the request
type reqLogType byte

const (
	executed                reqLogType = iota // the request got executed successfully
	outOfStateDBRange                         // the request was not executed due to not being in StateDBs block range
	noSubstateForGivenBlock                   // the request was not executed due to no having substate for given block
	noMatchingData
	statisticsLogFrequency = 10 * time.Second // how often will the app log statistics info
)

// todo why not executed - statedb out of range; no substate..
// requestLog transfers information from ReplayExecutor whether request was or was not executed for statistics purpose
type requestLog struct {
	method  string
	logType reqLogType
}

// newCounter returns a new instance of requestCounter
func newCounter(closed chan any, input chan requestLog, log *logging.Logger, wg *sync.WaitGroup) *requestCounter {
	m := map[reqLogType]map[string]uint64{}
	return &requestCounter{
		stats:   m,
		closed:  closed,
		ticker:  time.NewTicker(statisticsLogFrequency),
		input:   input,
		builder: new(strings.Builder),
		log:     log,
		wg:      wg,
	}
}

// Start requestCounter
func (c *requestCounter) Start() {
	c.log.Info("starting counter")
	c.wg.Add(1)
	go c.count()
}

// count is counters thread in which he reads requests from executor
func (c *requestCounter) count() {
	var (
		req requestLog
		ok  bool
	)

	defer func() {
		c.wg.Done()
		c.logStats()
	}()

	c.start = time.Now()

	for {
		select {
		case <-c.closed:
			return
		case <-c.ticker.C:
			c.logStats()
		case req, ok = <-c.input:
			if !ok {
				return
			}

			c.addStat(req)
			if req.logType == executed {
				c.total++
			}

		}
	}
}

// logStats about time, executed and total read requests. Frequency of logging depends on statisticsLogFrequency
func (c *requestCounter) logStats() {
	defer c.builder.Reset()

	// how long has replayer been running
	elapsed := time.Since(c.start)
	c.builder.WriteString(fmt.Sprintf("Elapsed time: %v\n\n", elapsed))

	// total requests
	c.builder.WriteString(fmt.Sprintf("Total read requests: %v\n\n", c.total))

	c.addExecuted()

	c.addOutOfDbRange()

	c.addNoSubstate()

	c.log.Notice(c.builder.String())
}

// addStat to given method and reqLogType
func (c *requestCounter) addStat(req requestLog) {
	if _, ok := c.stats[req.logType]; !ok {
		c.stats[req.logType] = make(map[string]uint64)
	}
	c.stats[req.logType][req.method]++
}

// addUnmatchedResults requests to counters string builder
func (c *requestCounter) addUnmatchedResults() {
	c.builder.WriteString(fmt.Sprintf("\nUnmatched results:\n"))

	var unmatchedResult uint64
	for method, count := range c.stats[noMatchingData] {
		c.builder.WriteString(fmt.Sprintf("\t%v: %v\n", method, count))
		// executed requests
		unmatchedResult += count
	}

	c.builder.WriteString(fmt.Sprintf("\n\tTotal: %v\n", unmatchedResult))
}

// addNoSubstate requests to counters string builder
func (c *requestCounter) addNoSubstate() {
	c.builder.WriteString(fmt.Sprintf("\nSkipped requests (non-existing substate):\n"))

	var noSubstate uint64
	for method, count := range c.stats[noSubstateForGivenBlock] {
		c.builder.WriteString(fmt.Sprintf("\t%v: %v\n", method, count))
		// executed requests
		noSubstate += count
	}

	c.builder.WriteString(fmt.Sprintf("\n\tTotal: %v\n", noSubstate))
}

// addOutOfDbRange requests to counters string builder
func (c *requestCounter) addOutOfDbRange() {
	c.builder.WriteString(fmt.Sprintf("\nSkipped requests (out of given Db range):\n"))

	var outOfRange uint64
	for m, count := range c.stats[outOfStateDBRange] {
		c.builder.WriteString(fmt.Sprintf("\t%v: %v\n", m, count))
		// executed requests
		outOfRange += count
	}

	c.builder.WriteString(fmt.Sprintf("\n\tTotal: %v\n", outOfRange))
}

// addExecuted requests to counters string builder
func (c *requestCounter) addExecuted() {
	c.builder.WriteString(fmt.Sprintf("\nExecuted requests:\n"))
	var exc uint64
	for m, count := range c.stats[executed] {
		c.builder.WriteString(fmt.Sprintf("\t%v: %v\n", m, count))
		// executed requests
		exc += count
	}

	c.builder.WriteString(fmt.Sprintf("\n\tTotal: %v\n\n", exc))
}
