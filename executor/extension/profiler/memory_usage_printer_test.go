package profiler

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/gogo/protobuf/plugin/stringer"
	"go.uber.org/mock/gomock"
)

func TestMemoryUsagePrinter_MemoryBreakdownIsNotPrintedWhenBreakdownIsNil(t *testing.T) {
	config := &utils.Config{}
	config.MemoryBreakdown = true

	ctrl := gomock.NewController(t)

	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)
	ext := makeMemoryUsagePrinter(config, log)

	usage := &state.MemoryUsage{
		Breakdown: nil,
	}

	gomock.InOrder(
		// Prerun
		db.EXPECT().GetMemoryUsage().Return(usage),
		log.EXPECT().Notice(gomock.Any()),

		// Postrun
		db.EXPECT().GetMemoryUsage().Return(usage),
		log.EXPECT().Notice(gomock.Any()),
	)

	ext.PreRun(executor.State{}, &executor.Context{State: db})
	ext.PostRun(executor.State{}, &executor.Context{State: db}, nil)

}

func TestMemoryUsagePrinter_MemoryBreakdownIsPrintedWhenEnabled(t *testing.T) {
	config := &utils.Config{}
	config.MemoryBreakdown = true

	ctrl := gomock.NewController(t)

	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)
	ext := makeMemoryUsagePrinter(config, log)

	usage := &state.MemoryUsage{
		UsedBytes: 1,
		Breakdown: stringer.NewStringer(),
	}

	gomock.InOrder(
		// Prerun
		db.EXPECT().GetMemoryUsage().Return(usage),
		log.EXPECT().Noticef(gomock.Any(), uint64(1), gomock.Any()),

		// Postrun
		db.EXPECT().GetMemoryUsage().Return(usage),
		log.EXPECT().Noticef(gomock.Any(), uint64(1), gomock.Any()),
	)

	ext.PreRun(executor.State{}, &executor.Context{State: db})
	ext.PostRun(executor.State{}, &executor.Context{State: db}, nil)

}

func TestMemoryUsagePrinter_NoPrinterIsCreatedIfNotEnabled(t *testing.T) {
	config := &utils.Config{}
	ext := MakeMemoryUsagePrinter(config)

	if _, ok := ext.(extension.NilExtension); !ok {
		t.Errorf("profiler is enabled although not set in configuration")
	}
}
