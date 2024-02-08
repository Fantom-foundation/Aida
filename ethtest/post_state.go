package ethtest

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

type Index struct {
	Data  int `json:"data"`
	Gas   int `json:"gas"`
	Value int `json:"value"`
}

func (s *StJSON) GetStateRoot() common.Hash {
	return s.getPostState().RootHash
}

func (s *StJSON) GetLogs() common.Hash {
	return s.getPostState().LogsHash
}

func (s *StJSON) GetTxBytes() hexutil.Bytes {
	return s.getPostState().TxBytes
}

func (s *StJSON) GetExpectException() string {
	return s.getPostState().ExpectException
}

func (s *StJSON) GetIndexes() Index {
	return s.getPostState().indexes
}
