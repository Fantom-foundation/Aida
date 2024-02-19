package rpc

import "testing"

func TestRequestAndResults_DecodeInfoPendingBlocksSkipValidation(t *testing.T) {
	r.DecodeInfo()
	if !r.SkipValidation {
		t.Fatal("skip validation must be true")
	}
}

var r = &RequestAndResults{
	Response: &Response{
		BlockID: 10,
	},
	Query: &Body{
		Params: []interface{}{
			"test", "pending",
		},
	},
	SkipValidation: false,
}
