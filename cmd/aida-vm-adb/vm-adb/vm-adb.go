package vm_adb

import (
	bp "github.com/Fantom-foundation/Aida/block_processor"
	"github.com/urfave/cli/v2"
)

// RunArchive performs block processing
func RunArchive(ctx *cli.Context) error {
	actions := bp.ExtensionList{
		bp.NewVMAdbExtension(),
		bp.NewProgressReportExtension(),
		bp.NewValidationExtension(),
		bp.NewProfileExtension(),
	}
	bp, err := bp.NewBlockProcessor("vm-adb", ctx)
	if err != nil {
		return err
	}
	return bp.Run(actions)
}
