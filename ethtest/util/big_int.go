package util

import (
	"encoding/json"
	"math/big"
	"strings"
)

type BigInt struct {
	big.Int
}

func (i *BigInt) Convert() *big.Int {
	if i == nil {
		return new(big.Int)
	}
	return &i.Int
}

func (i *BigInt) UnmarshalJSON(b []byte) error {
	var val string
	err := json.Unmarshal(b, &val)
	if err != nil {
		return err
	}

	i.SetString(strings.TrimPrefix(val, "0x"), 16)

	return nil
}

func (i *BigInt) MarshalJSON() ([]byte, error) {
	str := "0x" + i.Text(16)
	return json.Marshal(str)
}
