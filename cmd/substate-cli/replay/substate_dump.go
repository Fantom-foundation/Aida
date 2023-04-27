package replay

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// substate-cli dump command
var SubstateDumpCommand = cli.Command{
	Action:    substateDumpAction,
	Name:      "dump",
	Usage:     "returns content in substates in json format",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&substate.WorkersFlag,
		&substate.SubstateDirFlag,
	},
	Description: `
The substate-cli dump command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay transactions.`,
}

// replayTask replays a transaction substate
func substateDumpTask(block uint64, tx int, recording *substate.Substate, taskPool *substate.SubstateTaskPool) error {

	inputAlloc := recording.InputAlloc
	inputEnv := recording.Env
	inputMessage := recording.Message

	outputAlloc := recording.OutputAlloc
	outputResult := recording.Result

	out := fmt.Sprintf("block: %v Transaction: %v\n", block, tx)
	var jbytes []byte
	jbytes, _ = json.MarshalIndent(inputAlloc, "", " ")
	out += fmt.Sprintf("Recorded input substate:\n%s\n", jbytes)
	jbytes, _ = json.MarshalIndent(inputEnv, "", " ")
	out += fmt.Sprintf("Recorded input environmnet:\n%s\n", jbytes)
	jbytes, _ = json.MarshalIndent(inputMessage, "", " ")
	out += fmt.Sprintf("Recorded input message:\n%s\n", jbytes)
	jbytes, _ = json.MarshalIndent(outputAlloc, "", " ")
	out += fmt.Sprintf("Recorded output substate:\n%s\n", jbytes)
	jbytes, _ = json.MarshalIndent(outputResult, "", " ")
	out += fmt.Sprintf("Recorded output result:\n%s\n", jbytes)

	log.Println(out)

	return nil
}

// record-replay: func replayAction for replay command
func substateDumpAction(ctx *cli.Context) error {
	var err error

	if ctx.Args().Len() != 2 {
		return fmt.Errorf("substate-cli dump command requires exactly 2 arguments")
	}

	first, last, argErr := utils.SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1))
	if argErr != nil {
		return argErr
	}

	substate.SetSubstateDirectory(ctx.String(substate.SubstateDirFlag.Name))
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	taskPool := substate.NewSubstateTaskPool("substate-cli dump", substateDumpTask, first, last, ctx)
	err = taskPool.Execute()
	return err
}
