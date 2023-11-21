package tracker

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

type errorLogger[T any] struct {
	extension.NilExtension[T]
	cfg    *utils.Config
	file   *os.File
	log    logger.Logger
	wg     *sync.WaitGroup
	errors []error
}

func MakeErrorLogger[T any](cfg *utils.Config) executor.Extension[T] {
	return makeErrorLogger[T](cfg, logger.NewLogger("critical", "Error-Logger"))
}

func makeErrorLogger[T any](cfg *utils.Config, log logger.Logger) *errorLogger[T] {
	return &errorLogger[T]{
		cfg: cfg,
		log: log,
		wg:  new(sync.WaitGroup),
	}
}

func (l *errorLogger[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	ctx.ErrorInput = make(chan error, l.cfg.Workers*10)

	l.wg.Add(1)
	go l.doLogging(ctx.ErrorInput)

	if l.cfg.ErrorLogging == "" {
		return nil
	}

	l.log.Noticef("Creating log-file %v in which any processing error will be recorded.", l.cfg.ErrorLogging)

	var err error
	l.file, err = os.Create(l.cfg.ErrorLogging)
	if err != nil {
		return fmt.Errorf("cannot create log file %v; %v", l.cfg.ErrorLogging, err)
	}

	return nil
}

// PostRun closes the file and logging thread.
func (l *errorLogger[T]) PostRun(_ executor.State[T], ctx *executor.Context, _ error) error {
	close(ctx.ErrorInput)
	l.wg.Wait()

	if l.file != nil {
		err := l.file.Close()
		if err != nil {
			l.log.Errorf("cannot close log-file; %v", err)
		}
	}

	if len(l.errors) != 0 {
		return fmt.Errorf("total %v errors occurred: %v", len(l.errors), errors.Join(l.errors...))
	}

	return nil
}

func (l *errorLogger[T]) doLogging(input chan error) {
	defer l.wg.Done()

	var numberOfErrors int
	for {
		in := <-input
		if in == nil {
			return
		}
		numberOfErrors++
		l.log.Errorf("New error: \n\t%v", in)
		l.log.Warningf("Total number of errors %v", numberOfErrors)
		if l.file != nil {
			_, err := l.file.WriteString(in.Error())
			if err != nil {
				l.log.Errorf("cannot write into log-file; %v", err)
			}
		}
		l.errors = append(l.errors, in)
	}
}
