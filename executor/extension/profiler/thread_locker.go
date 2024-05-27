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

package profiler

import (
	"runtime"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
)

// MakeThreadLocker creates an executor.Extension locks the executor thread
// to a single OS level thread for the duration of the execution. This can
// have significant performance impact on EVM or StateDB executions.
func MakeThreadLocker[T any]() executor.Extension[T] {
	return threadLocker[T]{}
}

type threadLocker[T any] struct {
	extension.NilExtension[T]
}

func (threadLocker[T]) PreRun(executor.State[T], *executor.Context) error {
	runtime.LockOSThread()
	return nil
}

func (threadLocker[T]) PostRun(executor.State[T], *executor.Context, error) error {
	runtime.UnlockOSThread()
	return nil
}

func (threadLocker[T]) PreBlock(executor.State[T], *executor.Context) error {
	runtime.LockOSThread()
	return nil
}

func (threadLocker[T]) PostBlock(executor.State[T], *executor.Context) error {
	runtime.UnlockOSThread()
	return nil
}

func (threadLocker[T]) PreTransaction(executor.State[T], *executor.Context) error {
	runtime.LockOSThread()
	return nil
}

func (threadLocker[T]) PostTransaction(executor.State[T], *executor.Context) error {
	runtime.UnlockOSThread()
	return nil
}
