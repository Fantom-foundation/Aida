package extension

import (
	"log"
	"math"
	"net/http"
	_ "net/http/pprof"
	"runtime"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeDiagnosticServer creates an extension which runs a background
// HTTP server for real-time diagnosing aida processes.
func MakeDiagnosticServer(config *utils.Config) executor.Extension {
	return makeDiagnosticServer(config, logger.NewLogger(config.LogLevel, "Diagnostic-Server"))
}

func makeDiagnosticServer(config *utils.Config, logger logger.Logger) executor.Extension {
	if config.DiagnosticServer < 1 || config.DiagnosticServer > math.MaxUint16 {
		return NilExtension{}
	}
	return &diagnosticServer{
		port: config.DiagnosticServer,
		log:  logger,
	}
}

type diagnosticServer struct {
	NilExtension
	port int64
	log  logger.Logger
}

func (e *diagnosticServer) PreRun(executor.State, *executor.Context) error {
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
