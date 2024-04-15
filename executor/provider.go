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

package executor

//go:generate mockgen -source provider.go -destination provider_mocks.go -package executor

type Provider[T any] interface {
	// Run iterates through transaction in the block range [from,to) in order
	// and forwards payload information for each transaction in the range to
	// the provided consumer. Execution aborts if the consumer returns an error
	// or an error during the payload retrieval process occurred.
	Run(from int, to int, consumer Consumer[T]) error
	// Close releases resources held by the provider implementation. After this
	// no more operations are allowed on the same instance.
	Close()
}

// Consumer is a type alias for the type of function to which payload information
// can be forwarded by a Provider.
type Consumer[T any] func(TransactionInfo[T]) error

// TransactionInfo summarizes the per-transaction information provided by a
// Provider.
type TransactionInfo[T any] struct {
	Block       int
	Transaction int
	Data        T
}
