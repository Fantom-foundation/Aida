package vm_sdb

import (
	"github.com/urfave/cli/v2"
)

// RunVM performs block processing
func RunVM(ctx *cli.Context) error {
	actions := []ProcessorActions{NewLoggingAction(), NewValidationAction(), NewProfileAction()}
	bp, err := NewBlockProcessor(ctx)
	if err != nil {
		return err
	}
	return bp.Run("Run-VM", actions)
}
