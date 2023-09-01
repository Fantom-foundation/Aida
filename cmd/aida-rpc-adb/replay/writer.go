package replay

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/op/go-logging"
)

// logWriter receives any data mismatch from comparators and writes them into file
type logWriter struct {
	file   *os.File
	input  chan *comparatorError
	log    *logging.Logger
	closed chan any
	wg     *sync.WaitGroup
}

func newWriter(logLevel string, closed chan any, path string, wg *sync.WaitGroup) (*logWriter, chan *comparatorError, error) {
	now := time.Now()
	y, m, d := now.Date()
	var (
		hour   string
		minute string
	)

	if now.Hour() < 10 {
		hour = fmt.Sprintf("%v%v", 0, now.Hour())
	} else {
		hour = fmt.Sprintf("%v", now.Hour())
	}

	if now.Minute() < 10 {
		minute = fmt.Sprintf("%v%v", 0, now.Minute())
	} else {
		minute = fmt.Sprintf("%v", now.Minute())
	}

	fileName := fmt.Sprintf("/rpc-replay-log_%v-%v-%v_%v-%v.log", y, m.String(), d, hour, minute)

	filePath := filepath.Join(path, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot open file %v; %v", file, err)
	}

	input := make(chan *comparatorError, 100)

	return &logWriter{
		file:   file,
		input:  input,
		log:    logger.NewLogger(logLevel, "Error file Writer"),
		closed: closed,
		wg:     wg,
	}, input, nil

}

func (w *logWriter) write() {
	defer func() {
		if err := w.file.Close(); err != nil {
			w.log.Criticalf("cannot close api-log; %v", err)
		}
		w.wg.Done()
	}()

	var (
		compErr *comparatorError
		err     error
	)

	for {
		select {
		case <-w.closed:
			return
		case compErr = <-w.input:
			if _, err = w.file.WriteString(compErr.Error() + "\n\n\n\n"); err != nil {
				w.log.Errorf("cannot write into file; %v", err)
			}
		}
	}
}

func (w *logWriter) Start() {
	w.log.Info("starting counter")
	w.wg.Add(1)
	go w.write()
}
