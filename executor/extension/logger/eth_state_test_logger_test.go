package logger

import (
	"testing"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"go.uber.org/mock/gomock"
)

func TestEthStateTestLogger_PreTransactionLogsProgress(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	ext := makeEthStateTestLogger(log, 2)
	s := executor.State[txcontext.TxContext]{Data: ethtest.CreateTestData(t)}

	gomock.InOrder(
		log.EXPECT().Infof("Currently running:\n%s", s.Data),
		log.EXPECT().Infof("Currently running:\n%s", s.Data),
		log.EXPECT().Noticef("%v tests has been processed so far...", 2),
	)

	err := ext.PreTransaction(s, &executor.Context{})
	if err != nil {
		t.Fatalf("pre-tx failed: %v", err)
	}

	err = ext.PreTransaction(s, &executor.Context{})
	if err != nil {
		t.Fatalf("pre-tx failed: %v", err)
	}
}

func TestEthStateTestLogger_PostRunLogsOverall(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	ext := makeEthStateTestLogger(log, 0)
	ext.overall = 2
	s := executor.State[txcontext.TxContext]{Data: ethtest.CreateTestData(t)}

	gomock.InOrder(
		log.EXPECT().Noticef("Total %v tests processed.", 2),
	)

	err := ext.PostRun(s, &executor.Context{}, nil)
	if err != nil {
		t.Fatalf("post-run failed: %v", err)
	}

}
