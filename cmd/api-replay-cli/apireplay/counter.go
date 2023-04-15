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
	stats   map[string]uint64
	closed  chan any
	ticker  *time.Ticker
	input   chan requestLog
	start   time.Time
	builder *strings.Builder
	total   uint64
	log     *logging.Logger
	wg      *sync.WaitGroup
}

// requestLog transfers information from ReplayExecutor whether request was or was not executed for statistics purpose
type requestLog struct {
	method   string
	executed bool
}

// newCounter returns a new instance of requestCounter
func newCounter(closed chan any, logFrequency time.Duration, input chan requestLog, log *logging.Logger, wg *sync.WaitGroup) *requestCounter {
	return &requestCounter{
		stats:   make(map[string]uint64),
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

			// was request executed?
			if req.executed {
				c.stats[req.method]++
			}

			c.total++

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
	c.builder.WriteString(fmt.Sprintf("\tTotal read requests:%v\n\n", c.total))

	var executed uint64
	for m, count := range c.stats {
		c.builder.WriteString(fmt.Sprintf("\t%v: %v\n", m, count))
		// executed requests
		executed += count
	}

	c.builder.WriteString(fmt.Sprintf("Total executed requests:%v\n", executed))

	c.log.Notice(c.builder.String())
}
