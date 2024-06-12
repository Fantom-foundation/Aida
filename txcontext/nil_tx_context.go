// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package txcontext

import (
	"github.com/ethereum/go-ethereum/core"
)

// NilTxContext is an implementation of the TxContext interface
// ignoring all incoming events. It is mainly intended as a fall-back
// implementation when a no-op implementation is required, as well as
// an implementation that may be embedded in other extensions to avoid
// the need to implement all possible events.
type NilTxContext struct{}

func (NilTxContext) GetInputState() WorldState { return nil }

func (NilTxContext) GetBlockEnvironment() BlockEnvironment { return nil }

func (NilTxContext) GetMessage() core.Message { return core.Message{} }

func (NilTxContext) GetOutputState() WorldState { return nil }

func (NilTxContext) GetResult() Result { return nil }
