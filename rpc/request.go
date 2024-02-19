package rpc

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// RequestAndResults encapsulates request query and response for post-processing.
type RequestAndResults struct {
	Query                                    *Body
	Response                                 *Response
	Error                                    *ErrorResponse
	ParamsRaw                                []byte
	ResponseRaw                              []byte
	StateDB                                  *StateDBData
	SkipValidation                           bool
	RecordedBlock, RequestedBlock, Timestamp int
}

// DecodeInfo finds recorded and requested block numbers as well as timestamp of the recorded block.
func (r *RequestAndResults) DecodeInfo() {
	if r.Response != nil {
		r.RecordedBlock = int(r.Response.BlockID)
		r.Timestamp = int(r.Response.Timestamp)
	} else {
		r.RecordedBlock = int(r.Error.BlockID)
		r.Timestamp = int(r.Error.Timestamp)
	}
	r.findRequestedBlock()
}

func (r *RequestAndResults) findRequestedBlock() {
	l := len(r.Query.Params)
	if l < 2 {
		r.RequestedBlock = r.RecordedBlock
		return
	}

	str := r.Query.Params[l-1].(string)
	switch str {
	case "pending":
		// validation for pending requests does not work, skip them
		r.SkipValidation = true
		// pending should be treated as latest
		fallthrough
	case "latest":
		r.RequestedBlock = r.RecordedBlock
	case "earliest":
		r.RequestedBlock = 0

	default:
		// botched params are not recorded, so this will  never panic
		r.RequestedBlock = int(hexutil.MustDecodeUint64(str))
	}
}

// Body represents a decoded payload of a balancer.
// https://www.jsonrpc.org/specification#request_object
type Body struct {
	Version    string          `json:"jsonrpc,omitempty"`
	ID         json.RawMessage `json:"id,omitempty"`
	Method     string          `json:"method,omitempty"`
	Params     []interface{}   `json:"params,omitempty"`
	Namespace  string
	MethodBase string
}

// Response represents decoded request response body.
type Response struct {
	Version   string          `json:"jsonrpc,omitempty"`
	ID        json.RawMessage `json:"id,omitempty"`
	BlockID   uint64          `json:"blockid,omitempty"`
	Timestamp uint64          `json:"timestamp,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Payload   []byte          `json:"-"`
}

// ErrorResponse represents a structure of error response from the remote server
type ErrorResponse struct {
	Version   string          `json:"jsonrpc,omitempty"`
	Id        json.RawMessage `json:"id,omitempty"`
	BlockID   uint64          `json:"blockid,omitempty"`
	Timestamp uint64          `json:"timestamp,omitempty"`
	Error     ErrorMessage    `json:"error,omitempty"`
	Payload   []byte          `json:"-"`
}

// ErrorMessage represents the detailed error information inside an error response.
type ErrorMessage struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
