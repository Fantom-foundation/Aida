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
	cfg := &utils.Config{}
	cfg.MemoryBreakdown = true

	ctrl := gomock.NewController(t)

	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)
	ext := makeMemoryUsagePrinter[any](cfg, log)

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

	ext.PreRun(executor.State[any]{}, &executor.Context{State: db})
	ext.PostRun(executor.State[any]{}, &executor.Context{State: db}, nil)

}

func TestMemoryUsagePrinter_MemoryBreakdownIsPrintedWhenEnabled(t *testing.T) {
	cfg := &utils.Config{}
	cfg.MemoryBreakdown = true

	ctrl := gomock.NewController(t)

	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)
	ext := makeMemoryUsagePrinter[any](cfg, log)

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

	ext.PreRun(executor.State[any]{}, &executor.Context{State: db})
	ext.PostRun(executor.State[any]{}, &executor.Context{State: db}, nil)

}

func TestMemoryUsagePrinter_NoPrinterIsCreatedIfNotEnabled(t *testing.T) {
	config := &utils.Config{}
	ext := MakeMemoryUsagePrinter[any](config)

	if _, ok := ext.(extension.NilExtension[any]); !ok {
		t.Errorf("profiler is enabled although not set in configuration")
	}
}
