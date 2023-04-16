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

type reqLogType byte

const (
	executed reqLogType = iota
	outOfStateDBRange
	noSubstateForGivenBlock
)

// todo why not executed - statedb out of range; no substate..
// requestLog transfers information from ReplayExecutor whether request was or was not executed for statistics purpose
type requestLog struct {
	method  string
	logType reqLogType
}

// newCounter returns a new instance of requestCounter
func newCounter(closed chan any, logFrequency time.Duration, input chan requestLog, log *logging.Logger, wg *sync.WaitGroup) *requestCounter {
	m := map[reqLogType]map[string]uint64{}
	return &requestCounter{
		stats:   m,
		closed:  closed,
		ticker:  time.NewTicker(logFrequency),
		input:   input,
		builder: new(strings.Builder),
		log:     log,
		wg:      wg,
	}
}

// Start requestCounter
func (c *requestCounter) Start() {
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
		c.logStats()
		c.wg.Done()
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
			// todo
			//c.addStat(req)
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
	//elapsed := time.Since(c.start)
	//c.builder.WriteString(fmt.Sprintf("Elapsed time: %v\n\n", elapsed))
	//
	//// total requests
	//c.builder.WriteString(fmt.Sprintf("\tTotal read requests: %v\n\n", c.total))
	//
	//var exc uint64
	//for m, count := range c.stats[executed] {
	//	c.builder.WriteString(fmt.Sprintf("\t%v: %v\n", m, count))
	//	// executed requests
	//	exc += count
	//}
	//
	//c.builder.WriteString(fmt.Sprintf("\n\tTotal executed requests: %v\n", exc))
	//
	//var outOfRange uint64
	//for m, count := range c.stats[outOfStateDBRange] {
	//	c.builder.WriteString(fmt.Sprintf("\t%v: %v\n", m, count))
	//	// executed requests
	//	outOfRange += count
	//}
	//
	//c.builder.WriteString(fmt.Sprintf("\n\tTotal skipped due to not being in StateDB block range: %v\n", outOfRange))
	//
	//var noSubstate uint64
	//for m, count := range c.stats[noSubstateForGivenBlock] {
	//	c.builder.WriteString(fmt.Sprintf("\t%v: %v\n", m, count))
	//	// executed requests
	//	noSubstate += count
	//}
	//
	//c.builder.WriteString(fmt.Sprintf("\n\tTotal skipped due to non-existing substate for given block: %v\n", noSubstate))

	c.builder.WriteString(fmt.Sprintf("Requests sent: %v\n", c.total))

	c.log.Notice(c.builder.String())
}

// addStat to given method and reqLogType
func (c *requestCounter) addStat(req requestLog) {
	c.stats[req.logType][req.method]++
}
