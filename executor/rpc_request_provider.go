package executor

import (
	"errors"
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

func OpenRpcRecording(cfg *utils.Config, ctx *cli.Context) (Provider[*rpc.RequestAndResults], error) {
	iter, err := rpc.NewFileReader(ctx.Context, cfg.RpcRecordingFile)
	if err != nil {
		return nil, fmt.Errorf("cannot open rpc recording file; %v", err)
	}
	return openRpcRecording(iter, cfg, ctx), nil
}

func openRpcRecording(iter rpc.Iterator, cfg *utils.Config, ctxt *cli.Context) Provider[*rpc.RequestAndResults] {
	return rpcRequestProvider{
		ctxt:     ctxt,
		fileName: cfg.RpcRecordingFile,
		iter:     iter,
		log:      logger.NewLogger(cfg.LogLevel, "rpc-provider"),
	}
}

type rpcRequestProvider struct {
	ctxt     *cli.Context
	fileName string
	iter     rpc.Iterator
	log      logger.Logger
}

func (r rpcRequestProvider) Run(from int, to int, consumer Consumer[*rpc.RequestAndResults]) error {
	if err := r.processFirst(from, consumer); err != nil {
		return err
	}

	for r.iter.Next() {
		if r.iter.Error() != nil {
			return fmt.Errorf("iterator returned error; %v", r.iter.Error())
		}

		req := r.iter.Value()

		if req == nil {
			return errors.New("iterator returned nil request")
		}

		// get logs is not yet implemented, skip these for now
		if req.Query.MethodBase == "getLogs" {
			continue
		}

		req.DecodeInfo()

		// are we skipping requests?
		if req.RecordedBlock < from {
			continue
		}

		if req.RecordedBlock >= to {
			return nil
		}

		if err := consumer(TransactionInfo[*rpc.RequestAndResults]{req.RecordedBlock, 0, req}); err != nil {
			return err
		}
	}

	return nil
}

func (r rpcRequestProvider) Close() {
	r.iter.Close()
}

// processFirst takes first request and logs information about run.
func (r rpcRequestProvider) processFirst(from int, consumer Consumer[*rpc.RequestAndResults]) error {
	if r.iter.Next() {
		if r.iter.Error() != nil {
			return fmt.Errorf("iterator returned error; %v", r.iter.Error())
		}

		req := r.iter.Value()
		if req == nil {
			return errors.New("iterator returned nil request")
		}

		req.DecodeInfo()
		r.log.Noticef("First block of recording: %v", req.RecordedBlock)

		// are we skipping requests?
		if req.RecordedBlock < from {
			r.log.Noticef("Skipping %v blocks. This might take a while, skip rate is ~50k Req/s "+
				"and there is up to 2500 Requests in a block.", from-req.RecordedBlock)
			return nil
		}

		// get logs is not yet implemented, skip these for now
		if req.Query.MethodBase == "getLogs" {
			return nil
		}

		if err := consumer(TransactionInfo[*rpc.RequestAndResults]{req.RecordedBlock, 0, req}); err != nil {
			return err
		}
	} else {
		r.log.Critical("Iterator returned no requests.")
	}

	return nil
}
