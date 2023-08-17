package vm_sdb

import (
	bp "github.com/Fantom-foundation/Aida/block_processor"
	"github.com/urfave/cli/v2"
)

// RunVM performs block processing
func RunVM(ctx *cli.Context) error {
	actions := bp.ExtensionList{
		bp.NewProgressReportExtension(),
		bp.NewValidationExtension(),
		bp.NewProfileExtension(),
		bp.NewDbManagerExtension(),
		bp.NewProxyLoggerExtension(),
		bp.NewProxyProfilerExtension(),
	}
	bp, err := bp.NewBlockProcessor("vm-sdb", ctx)
	if err != nil {
		return err
	}
	return bp.Run(actions)
}
