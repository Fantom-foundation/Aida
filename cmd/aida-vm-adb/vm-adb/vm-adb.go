package vm_adb

import (
	blockprocessor "github.com/Fantom-foundation/Aida/block_processor"
	"github.com/urfave/cli/v2"
)

// RunArchive performs block processing
func RunArchive(ctx *cli.Context) error {
	actions := blockprocessor.NewExtensionList([]blockprocessor.ProcessorExtensions{
		blockprocessor.NewProgressReportExtension(),
		blockprocessor.NewValidationExtension(),
		blockprocessor.NewProfileExtension(),
	})
	bp, err := blockprocessor.NewBlockProcessor(ctx, blockprocessor.VmAdbToolName)
	if err != nil {
		return err
	}
	return bp.Run(actions, blockprocessor.VmAdbIterate)
}
