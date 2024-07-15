package ethtest

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/logger"
	"go.uber.org/mock/gomock"
	"golang.org/x/exp/maps"
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

func TestDecoder_NewDecoder_SortsForks(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	log.EXPECT().Warningf("Unknown name fork name %v, removing", strings.ToLower("toBeRemoved"))

	got := sortForks(log, []string{"CaNcuN", "london", "toBeRemoved"})
	want := []string{"cancun", "london"}
	if !slices.Equal(got, want) {
		t.Fatalf("unexpected forks, got: %v\nwant: %v", got, want)
	}
}

func TestDecoder_NewDecoder_AllAddsAllForks(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	got := sortForks(log, []string{"all"})
	want := maps.Keys(usableForks)
	// Maps are unordered...
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("unexpected forks, got: %v\nwant: %v", got, want)
	}
}
