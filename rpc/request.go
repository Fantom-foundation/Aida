package rpc

import "encoding/json"

// RequestAndResults encapsulates request query and response for post-processing.
type RequestAndResults struct {
	Query       *Body
	Response    *Response
	Error       *ErrorResponse
	ParamsRaw   []byte
	ResponseRaw []byte
	ReturnState *ReturnState
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
