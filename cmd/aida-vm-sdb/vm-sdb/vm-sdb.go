package vm_sdb

import (
	blockprocessor "github.com/Fantom-foundation/Aida/block_processor"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// RunVM performs block processing
func RunVM(ctx *cli.Context) error {
	actions := blockprocessor.NewExtensionList([]blockprocessor.ProcessorExtensions{
		blockprocessor.NewProgressReportExtension(),
		blockprocessor.NewValidationExtension(),
		blockprocessor.NewProfileExtension(),
		blockprocessor.NewDbManagerExtension(),
		blockprocessor.NewProxyLoggerExtension(),
		blockprocessor.NewProxyProfilerExtension(),
	})

	bp, err := blockprocessor.NewBlockProcessor(ctx, blockprocessor.VmSdbToolName)
	if err != nil {
		return err
	}
	defer utils.PrintEvmStatistics(bp.GetConfig())

	return bp.Run(actions, blockprocessor.BasicIterator)
}
