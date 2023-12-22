package register

import (
	"os"
	"testing"
	"path/filepath"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/state"

	//db
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"go.uber.org/mock/gomock"
)

const (
	sqlite3SelectFromStats string = `
		select start, end, memory, disk, tx_rate, gas_rate, overall_tx_rate, overall_gas_rate
		from stats
		where start>=:start and end<=:end;
	`
)

type query struct {
	Start  int `db:"start"`
	End    int `db:"end"`
}

type statsResponse struct {
	Start          int     `db:"start"`
	End            int     `db:"end"`
	Memory         int     `db:"memory"`
	Disk           int     `db:"disk"`
	TxRate         float64 `db:"tx_rate"`
	GasRate        float64 `db:"gas_rate"`
	OverallTxRate  float64 `db:"overall_tx_rate"`
	OverallGasRate float64 `db:"overall_gas_rate"`
}

func TestRegisterProgress_DoNothingIfDisabled(t *testing.T) {
	cfg := &utils.Config{}
	cfg.RegisterRun = ""
	ext := MakeRegisterProgress(cfg, 0)
	if _, ok := ext.(extension.NilExtension[*substate.Substate]); !ok {
		t.Errorf("RegisterProgress is enabled even though not disabled in configuration.")
	}
}

func TestRegisterProgress_InsertToDbIfEnabled(t *testing.T) {
	var (
		dummyStateDbPath string = filepath.Join(t.TempDir(), "dummy.txt")
		connection string = filepath.Join(t.TempDir(), "tmp.db")
	)

	// Check if path to state db is writable
	if err := os.WriteFile(dummyStateDbPath, []byte("hello world"), 0x600); err != nil {
		t.Fatalf("failed to prepare disk content for %s.", dummyStateDbPath)
	}

	// Check if path to stats db is writable
	sDb, err := sqlx.Open("sqlite3", connection)
	if err != nil {
		t.Fatalf("Failed to connect to database at %s.", connection)
	}

	_, err = sDb.Exec(RegisterProgressCreateTableIfNotExist)
	if err != nil {
		t.Fatalf("Unable to create stats table at database %s.\n%s", connection, err)
	}

	stmt, err := sDb.PrepareNamed(sqlite3SelectFromStats)
	if err != nil {
		t.Fatalf("Failed to prepare statement using db at %s. \n%s", connection, err)
	}
	
	ctrl := gomock.NewController(t)
	stateDb := state.NewMockStateDB(ctrl)

	cfg := &utils.Config{}
	cfg.RegisterRun = connection // enabled here
	cfg.First = 5
	cfg.Last = 25
	interval := 10
	// expects [5-9]P[10-19]P[20-24]P, where P is print
	
	itv := utils.NewInterval(cfg.First, cfg.Last, uint64(interval))
	ext := MakeRegisterProgress(cfg, interval)
	if _, err := ext.(extension.NilExtension[*substate.Substate]); err {
		t.Errorf("RegisterProgress is disabled even though enabled in configuration.")
	}

	ctx := &executor.Context{State: stateDb, StateDbPath: dummyStateDbPath}

	s := &substate.Substate {
		Result: &substate.SubstateResult{
			Status: 0,
			GasUsed: 100,
		},
	}

	expectedRowCount := 0

	// prints 3 times
	gomock.InOrder(
		stateDb.EXPECT().GetMemoryUsage().Return(&state.MemoryUsage{UsedBytes: 1234}),
		stateDb.EXPECT().GetMemoryUsage().Return(&state.MemoryUsage{UsedBytes: 4321}),
		stateDb.EXPECT().GetMemoryUsage().Return(&state.MemoryUsage{UsedBytes: 5555}),
	)

	ext.PreRun(executor.State[*substate.Substate]{}, ctx)

	for b := int(cfg.First) ; b < int(cfg.Last); b++ {
		ext.PreBlock(executor.State[*substate.Substate]{Block: b, Data: s}, ctx)

		// check if a print happens here
		if b > int(itv.End()) {
			itv.Next()
			expectedRowCount++
		}
		stats := []statsResponse{}
		stmt.Select(&stats, query{int(cfg.First), int(cfg.Last)})
		if len(stats) != expectedRowCount {
			t.Errorf("Expected #Row: %d, Actual #Row: %d", expectedRowCount, len(stats))
		}

		ext.PreTransaction(executor.State[*substate.Substate]{Data: s}, ctx)
		ext.PostTransaction(executor.State[*substate.Substate]{Data: s}, ctx)
		ext.PostBlock(executor.State[*substate.Substate]{Block: b, Data: s}, ctx)
	}

	ext.PostRun(executor.State[*substate.Substate]{}, ctx, nil)
	
	// check if a print happens here
	expectedRowCount++
	stats := []statsResponse{}
	stmt.Select(&stats, query{int(cfg.First), int(cfg.Last)})
	if len(stats) != expectedRowCount {
		t.Errorf("Expected #Row: %d, Actual #Row: %d", expectedRowCount, len(stats))
	}

	stmt.Close()
	sDb.Close()
}
