package statetest

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type PostState interface {
	GetStateRoot() common.Hash
	GetLogs() common.Hash
	GetTxBytes() hexutil.Bytes
	GetExpectException() string
	GetIndexes() Index
}

type stPostState struct {
	RootHash        common.Hash   `json:"hash"`
	LogsHash        common.Hash   `json:"logs"`
	TxBytes         hexutil.Bytes `json:"txbytes"`
	ExpectException string        `json:"expectException"`
	indexes         Index
}
