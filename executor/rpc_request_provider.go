package executor

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	}
}

type rpcRequestProvider struct {
	ctxt     *cli.Context
	fileName string
	iter     rpc.Iterator
}

func (r rpcRequestProvider) Run(from int, to int, consumer Consumer[*rpc.RequestAndResults]) error {
	var recordedBlockNumber int

	for r.iter.Next() {
		if r.iter.Error() != nil {
			return fmt.Errorf("iterator returned error; %v", r.iter.Error())
		}

		req := r.iter.Value()

		if req == nil {
			return nil
		}

		// get logs is not yet implemented, skip these for now
		if req.Query.MethodBase == "getLogs" {
			continue
		}

		if req.Response != nil {
			recordedBlockNumber = int(req.Response.BlockID)
			req.RecordedTimestamp = req.Response.Timestamp
		} else {
			recordedBlockNumber = int(req.Error.BlockID)
			req.RecordedTimestamp = req.Error.Timestamp
		}

		req.RequestedBlock = findRequestedBlockNumber(req, recordedBlockNumber)

		// are we skipping requests?
		if req.RequestedBlock < from {
			continue
		}

		if err := consumer(TransactionInfo[*rpc.RequestAndResults]{recordedBlockNumber, 0, req}); err != nil {
			return err
		}
	}

	return nil
}

func findRequestedBlockNumber(data *rpc.RequestAndResults, recordedBlockNumber int) int {
	l := len(data.Query.Params)
	if l < 2 {
		return recordedBlockNumber
	}

	str := data.Query.Params[l-1].(string)

	switch str {
	case "pending":
		// validation for pending requests does not work, skip them
		data.SkipValidation = true
		// pending should be treated as latest
		fallthrough
	case "latest":
		return recordedBlockNumber
	case "earliest":
		return 0

	default:
		// botched params are not recorded, so this will  never panic
		return int(hexutil.MustDecodeUint64(str))
	}
}

func (r rpcRequestProvider) Close() {
	r.iter.Close()
}
