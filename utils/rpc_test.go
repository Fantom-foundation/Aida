package utils

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// TestSendRPCRequest sends a getEpochByNumber with "latest" argument and tries to unmarshal the result
func TestSendRPCRequest(t *testing.T) {
	req := JsonRPCRequest{
		Method:  "ftm_getBlockByNumber",
		Params:  []interface{}{"latest", false},
		ID:      1,
		JSONRPC: "2.0",
	}

	for _, id := range AvailableChainIDs {
		t.Run(fmt.Sprintf("ChainID %v", id), func(t *testing.T) {

			res, err := SendRPCRequest(req, id)
			if err != nil {
				t.Fatalf("SendRPCRequest returned err; %v", err)
			}

			if res == nil {
				t.Fatal("response was nil")
			}

			result, ok := res["result"]
			if !ok {
				t.Fatal("response did not have result")
			}

			resultMap, ok := result.(map[string]interface{})
			if !ok {
				t.Fatal("result cannot be retyped to map")
			}

			hexBlockNumber, ok := resultMap["number"]
			if !ok {
				t.Fatal("result did not contain block number")
			}

			str, ok := hexBlockNumber.(string)
			if !ok {
				t.Fatal("cannot retype hex block number to string")
			}

			blockNumber, err := strconv.ParseInt(strings.TrimPrefix(str, "0x"), 16, 64)
			if err != nil {
				t.Fatalf("cannot parse string hex into number")
			}

			if blockNumber == 0 {
				t.Fatalf("latest block number cannot be 0; block number: %v", blockNumber)
			}
		})
	}

}

func TestRPCFindEpochNumber(t *testing.T) {
	var (
		expectedMainnetEpoch uint64 = 5576
		testingMainnetBlock  uint64 = 4_564_025

		expectedTestnetEpoch uint64 = 2457
		testingTestnetBlock  uint64 = 479_326
	)

	for _, id := range AvailableChainIDs {
		t.Run(fmt.Sprintf("ChainID %v", id), func(t *testing.T) {
			var testingBlock, expectedEpoch uint64

			if id == 250 {
				testingBlock = testingMainnetBlock
				expectedEpoch = expectedMainnetEpoch
			} else if id == 4002 {
				testingBlock = testingTestnetBlock
				expectedEpoch = expectedTestnetEpoch
			} else {
				t.Fatalf("unknown chainID: %v", id)
			}

			returnedEpoch, err := FindEpochNumber(testingBlock, id)
			if err != nil {
				t.Fatalf("FindEpochNumber returned err; %v", err)
			}

			if expectedEpoch != returnedEpoch {
				t.Fatalf("block numbers are different; returned: %v, expected: %v", returnedEpoch, expectedEpoch)
			}
		})
	}

}
