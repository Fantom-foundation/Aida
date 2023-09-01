package utils

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

const invalidChainID ChainID = -1

// TestSendRPCRequest_Positive tests whether SendRPCRequest does not return error for a valid request and chainID
func TestSendRPCRequest_Positive(t *testing.T) {
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

// TestSendRPCRequest_InvalidChainID tests whether SendRPCRequest does return an error for a valid request and invalid chainID
func TestSendRPCRequest_InvalidChainID(t *testing.T) {
	req := JsonRPCRequest{
		Method:  "ftm_getBlockByNumber",
		Params:  []interface{}{"latest", false},
		ID:      1,
		JSONRPC: "2.0",
	}

	_, err := SendRPCRequest(req, invalidChainID)
	if err == nil {
		t.Fatal("SendRPCRequest must return an err")
	}

	if !strings.Contains(err.Error(), "unknown chain-id") {
		t.Fatalf("SendRPCRequest returned unexpected error: %v", err.Error())
	}

}

// TestSendRPCRequest_InvalidReqMethod tests whether SendRPCRequest does return an error for an invalid method inside request
func TestSendRPCRequest_InvalidReqMethod(t *testing.T) {
	req := JsonRPCRequest{
		Method:  "ftm_invalid",
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

			e, ok := res["error"]
			if !ok {
				t.Fatal("response did not have an error")
			}

			_, ok = e.(map[string]interface{})
			if !ok {
				t.Fatal("error cannot be retyped to map")
			}
		})
	}
}

// TestSendRPCRequest_InvalidReqMethod tests whether SendRPCRequest does return an error for an invalid block number inside request
func TestSendRPCRequest_InvalidBlockNumber(t *testing.T) {
	req := JsonRPCRequest{
		Method:  "ftm_getBlockByNumber",
		Params:  []interface{}{"0xinvalid", false},
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

			e, ok := res["error"]
			if !ok {
				t.Fatal("response did not have an error")
			}

			_, ok = e.(map[string]interface{})
			if !ok {
				t.Fatal("error cannot be retyped to map")
			}
		})
	}

}

// TestRPCFindEpochNumber_Positive tests whether FindEpochNumber does not return error for a valid block and chainID
func TestRPCFindEpochNumber_Positive(t *testing.T) {
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
