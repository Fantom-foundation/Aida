package vm_sdb

import (
	bp "github.com/Fantom-foundation/Aida/block_processor"
	"github.com/urfave/cli/v2"
)

// RunVM performs block processing
func RunVM(ctx *cli.Context) error {
	actions := []bp.ProcessorActions{bp.NewLoggingAction(), bp.NewValidationAction(), bp.NewProfileAction()}
	bp, err := bp.NewBlockProcessor(ctx)
	if err != nil {
		return err
	}
	return bp.Run("vm-sdb", actions)
}
