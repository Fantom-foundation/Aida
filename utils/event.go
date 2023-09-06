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
