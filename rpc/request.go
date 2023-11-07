package rpc

import "encoding/json"

// RequestAndResults encapsulates request query and response for post-processing.
type RequestAndResults struct {
	Query       *Body
	Response    json.RawMessage
	Error       *Error
	ParamsRaw   []byte
	ResponseRaw []byte
	ReturnState *ReturnState // ReturnState is attached after execution is completed
	Timestamp   uint64
	Block       int
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

// Error represents the detailed error information inside an error response.
type Error struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
