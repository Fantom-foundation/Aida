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

package extension

import "github.com/Fantom-foundation/Aida/executor"

// NilExtension is an implementation of the executor.Extension interface
// ignoring all incoming events. It is mainly intended as a fall-back
// implementation when a no-op implementation is required, as well as an
// implementation that may be embedded in other extensions to avoid the
// need to implement all possible events.
type NilExtension[T any] struct{}

func (NilExtension[T]) PreRun(executor.State[T], *executor.Context) error          { return nil }
func (NilExtension[T]) PostRun(executor.State[T], *executor.Context, error) error  { return nil }
func (NilExtension[T]) PreBlock(executor.State[T], *executor.Context) error        { return nil }
func (NilExtension[T]) PostBlock(executor.State[T], *executor.Context) error       { return nil }
func (NilExtension[T]) PreTransaction(executor.State[T], *executor.Context) error  { return nil }
func (NilExtension[T]) PostTransaction(executor.State[T], *executor.Context) error { return nil }
