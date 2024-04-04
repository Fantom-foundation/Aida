package utildb

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Fantom-foundation/Substate/db"
	"github.com/Fantom-foundation/Substate/substate"
)

// SubstateDumpTask dumps substate data
func SubstateDumpTask(block uint64, tx int, recording *substate.Substate, taskPool *db.SubstateTaskPool) error {

	InputSubstate := recording.InputSubstate
	inputEnv := recording.Env
	inputMessage := recording.Message

	outputAlloc := recording.OutputSubstate
	outputResult := recording.Result

	out := fmt.Sprintf("block: %v Transaction: %v\n", block, tx)
	var jbytes []byte
	jbytes, _ = json.MarshalIndent(InputSubstate, "", " ")
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
