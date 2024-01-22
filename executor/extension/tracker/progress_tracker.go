package tracker

import (
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

const (
	ProgressTrackerDefaultReportFrequency = 100_000 // in blocks
)

func newProgressTracker[T any](cfg *utils.Config, reportFrequency int, log logger.Logger) *progressTracker[T] {
	return &progressTracker[T]{
		cfg:             cfg,
		log:             log,
		reportFrequency: reportFrequency,
	}
}

type progressTracker[T any] struct {
	extension.NilExtension[T]
	cfg                 *utils.Config
	log                 logger.Logger
	reportFrequency     int
	startOfRun          time.Time
	startOfLastInterval time.Time

	lock sync.Mutex
}

func (t *progressTracker[T]) PreRun(executor.State[T], *executor.Context) error {
	now := time.Now()
	t.startOfRun = now
	t.startOfLastInterval = now
	return nil
}
