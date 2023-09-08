package extension

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

const testStateDbInfoFrequency = 1

func TestStateDbInfoLoggerExtension_NoLoggerIsCreatedIfDisabled(t *testing.T) {
	config := &utils.Config{}
	config.Quiet = true
	ext := MakeStateDbInfoLogger(config, testStateDbInfoFrequency)
	if _, ok := ext.(NilExtension); !ok {
		t.Errorf("Logger is enabled although not set in configuration")
	}

}

func TestStateDbInfoLoggerExtension_LoggingHappens(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	config := &utils.Config{}

	ext := makeStateDbInfoLogger(config, testStateDbInfoFrequency, log)

	gomock.InOrder(
		// scheduled logging
		db.EXPECT().GetMemoryUsage(),
		log.EXPECT().Infof(stateDbInfoLoggerReportFormat, 1, float64(0), gomock.Any()),
		// defer logging
		log.EXPECT().Noticef(finalSummaryStateDbInfoReportFormat, float64(0), float64(0), 1),
	)

	ext.PostBlock(executor.State{
		Block: 1,
		State: db,
	})

	ext.PostRun(executor.State{}, nil)
}
