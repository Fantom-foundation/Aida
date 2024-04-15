// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package utildb

import (
	"encoding/json"
	"fmt"
	"log"

	substate "github.com/Fantom-foundation/Substate"
)

// SubstateDumpTask dumps substate data
func SubstateDumpTask(block uint64, tx int, recording *substate.Substate, taskPool *substate.SubstateTaskPool) error {

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
