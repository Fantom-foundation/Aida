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

package utils

import "sync/atomic"

// Event is a synchronization primitive for signaling the occurrence of
// a one-time event. The life cycle of each event is divided in two
// phases: the time before and the time after the event.
//
//	event := MakeEvent()
//	// The event has not yet occurred
//	event.Signal()
//	// The event has occurred, any further signaling has no effect
//
// Signaling and checking whether the event has occurred is thread safe.
// Furthermore, goroutines may wait for the occurrence of the event through
// a provided channel.
type Event interface {
	// HasHappened returns whether the event has already occurred or not.
	HasHappened() bool
	// Wait provides a channel which will be closed once the event occurred.
	Wait() <-chan struct{}
	// Signal triggers the event.
	Signal()
}

func MakeEvent() Event {
	return &event{channel: make(chan struct{})}
}

type event struct {
	channel  chan struct{}
	occurred atomic.Bool
}

func (e *event) HasHappened() bool {
	return e.occurred.Load()
}

func (e *event) Wait() <-chan struct{} {
	return e.channel
}

func (e *event) Signal() {
	// Atomically checks whether the channel is already closed and
	// closes the channel if this is not the case.
	if hasOccurred := e.occurred.Swap(true); !hasOccurred {
		close(e.channel)
	}
}
