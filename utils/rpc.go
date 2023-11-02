package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common/math"
)

const (
	RPCMainnet = "https://rpcapi.fantom.network"
	RPCTestnet = "https://rpc.testnet.fantom.network/"
)

type JsonRPCRequest struct {
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      uint64        `json:"id"`
	JSONRPC string        `json:"jsonrpc"`
}

func SendRpcRequest(payload JsonRPCRequest, chainId ChainID) (map[string]interface{}, error) {
	url, err := GetProvider(chainId)
	if err != nil {
		return nil, err
	}

	jsonReq, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal req with first block; %v", err)
	}

	//resp, err := http.Post(RPCMainnet, "application/json", bytes.NewBuffer(jsonReq))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonReq))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	m := make(map[string]interface{})

	if err = json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}

	return m, nil
}

func GetProvider(chainId ChainID) (string, error) {
	if chainId == 250 {
		return RPCMainnet, nil
	} else if chainId == 4002 {
		return RPCTestnet, nil
	} else {
		return "", fmt.Errorf("unknown chain-id %v", chainId)
	}
}

// FindEpochNumber via RPC request GetBlockByNumber
func FindEpochNumber(blockNumber uint64, chainId ChainID) (uint64, error) {
	hex := strconv.FormatUint(blockNumber, 16)

	blockStr := "0x" + hex

	return getEpochByNumber(blockStr, chainId)
}

// FindHeadEpochNumber via RPC request GetBlockByNumber
func FindHeadEpochNumber(chainId ChainID) (uint64, error) {
	blockStr := "latest"

	return getEpochByNumber(blockStr, chainId)
}

func getEpochByNumber(blockStr string, chainId ChainID) (uint64, error) {
	payload := JsonRPCRequest{
		Method:  "ftm_getBlockByNumber",
		Params:  []interface{}{blockStr, false},
		ID:      1,
		JSONRPC: "2.0",
	}

	m, err := SendRpcRequest(payload, chainId)
	if err != nil {
		return 0, err
	}

	resultMap, ok := m["result"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpecetd answer: %v", m)
	}

	firstEpochHex, ok := resultMap["epoch"].(string)
	if !ok {
		return 0, fmt.Errorf("cannot find epoch in result: %v", resultMap)
	}

	epoch, ok := math.ParseUint64(firstEpochHex)
	if !ok {
		return 0, fmt.Errorf("cannot parse hex first epoch to uint: %v", firstEpochHex)
	}

	return epoch, nil
}
