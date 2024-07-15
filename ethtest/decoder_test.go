package ethtest

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/logger"
)

func TestDecoder_DivideStateTests_DividesDataAccordingToIndexes(t *testing.T) {
	stJson := createTestStJson(t)
	d := Decoder{jsons: []*stJSON{stJson}, log: logger.NewLogger("info", "decoder-test")}
	for _, testCase := range d.DivideStateTests() {
		msg := testCase.GetMessage()
		if strings.Contains(fmt.Sprintf("%s", testCase), "Cancun") {
			// Cancun fork contains data 1 and data 2 but since map is not ordered we cannot guarantee
			gotData := hex.EncodeToString(msg.Data)
			if !(strings.Contains(gotData, data1) || strings.Contains(gotData, data2)) {
				t.Fatalf("unexpected data\ngot: %v\nwant: %v or %v", gotData, data1, data2)
			}

			gotValue := msg.Value
			want1, _ := new(big.Int).SetString(data1, 16)
			want2, _ := new(big.Int).SetString(data2, 16)
			if !(gotValue.Cmp(want1) == 0 || gotValue.Cmp(want2) == 0) {
				t.Fatalf("unexpected value\ngot: %v\nwant: %v or %v", gotValue, want1, want2)
			}
		} else {
			// London fork contains data 3 and data 4 but since map is not ordered we cannot guarantee
			got := hex.EncodeToString(msg.Data)
			if !(strings.Contains(got, data3) || strings.Contains(got, data4)) {
				t.Fatalf("unexpected data\ngot: %v\nwant: %v or %v", got, data1, data2)
			}

			gotValue := msg.Value
			want3, _ := new(big.Int).SetString(data3, 16)
			want4, _ := new(big.Int).SetString(data4, 16)
			if !(gotValue.Cmp(want3) == 0 || gotValue.Cmp(want4) == 0) {
				t.Fatalf("unexpected value\ngot: %v\nwant: %v or %v", got, data1, data2)
			}
		}
	}

}
