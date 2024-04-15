// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// RequestAndResults encapsulates request query and response for post-processing.
type RequestAndResults struct {
	Query                         *Body
	Response                      *Response
	Error                         *ErrorResponse
	ParamsRaw                     []byte
	ResponseRaw                   []byte
	SkipValidation                bool
	IsRecovered                   bool
	RecordedBlock, RequestedBlock int
	Timestamp                     uint64
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

// DecodeInfo finds recorded and requested block numbers as well as timestamp of the recorded block.
func (r *RequestAndResults) DecodeInfo() {
	if r.Response != nil {
		r.RecordedBlock = int(r.Response.BlockID)
		r.Timestamp = uint64(time.Unix(0, int64(r.Response.Timestamp)).Unix())
	} else {
		r.RecordedBlock = int(r.Error.BlockID)
		r.Timestamp = uint64(time.Unix(0, int64(r.Error.Timestamp)).Unix())
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
