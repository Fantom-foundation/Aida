package profiler

import (
	"fmt"
	"log"
	"math"
	"net/http"
	_ "net/http/pprof"
	"runtime"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeDiagnosticServer creates an extension which runs a background
// HTTP server for real-time diagnosing aida processes.
func MakeDiagnosticServer[T any](cfg *utils.Config) executor.Extension[T] {
	return makeDiagnosticServer[T](cfg, logger.NewLogger(cfg.LogLevel, "Diagnostic-Server"))
}

func makeDiagnosticServer[T any](cfg *utils.Config, log logger.Logger) executor.Extension[T] {
	if cfg.DiagnosticServer < 1 || cfg.DiagnosticServer > math.MaxUint16 {
		return extension.NilExtension[T]{}
	}
	return &diagnosticServer[T]{
		port: cfg.DiagnosticServer,
		log:  log,
	}
}

type diagnosticServer[T any] struct {
	extension.NilExtension[T]
	port int64
	log  logger.Logger
}

func (e *diagnosticServer[T]) PreRun(executor.State[T], *executor.Context) error {
	e.log.Infof("Starting diagnostic server at port http://localhost:%d (see https://pkg.go.dev/net/http/pprof#hdr-Usage_examples for usage examples)", e.port)
	e.log.Warning("Block and mutex sampling rate is set to 100%% for diagnostics, which may impact overall performance")
	go func() {
		addr := fmt.Sprintf("localhost:%d", e.port)
		log.Println(http.ListenAndServe(addr, nil))
	}()
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)
	return nil
}
