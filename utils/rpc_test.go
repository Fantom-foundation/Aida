// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/exp/maps"
)

const invalidChainID ChainID = -1

// TestSendRPCRequest_Positive tests whether SendRpcRequest does not return error for a valid request and chainID
func TestSendRPCRequest_Positive(t *testing.T) {
	req := JsonRPCRequest{
		Method:  "ftm_getBlockByNumber",
		Params:  []interface{}{"latest", false},
		ID:      1,
		JSONRPC: "2.0",
	}

	for _, id := range maps.Keys(RealChainIDs) {
		t.Run(fmt.Sprintf("ChainID %v", id), func(t *testing.T) {

			res, err := SendRpcRequest(req, id)
			if errors.Is(err, RPCUnsupported) {
				t.Skip("RPC is not supported")
			}
			if err != nil {
				t.Fatalf("SendRpcRequest returned err; %v", err)
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

// TestSendRPCRequest_InvalidChainID tests whether SendRpcRequest does return an error for a valid request and invalid chainID
func TestSendRPCRequest_InvalidChainID(t *testing.T) {
	req := JsonRPCRequest{
		Method:  "ftm_getBlockByNumber",
		Params:  []interface{}{"latest", false},
		ID:      1,
		JSONRPC: "2.0",
	}

	_, err := SendRpcRequest(req, invalidChainID)
	if err == nil {
		t.Fatal("SendRpcRequest must return an err")
	}

	if !strings.Contains(err.Error(), "unknown chain-id") {
		t.Fatalf("SendRpcRequest returned unexpected error: %v", err.Error())
	}

}
