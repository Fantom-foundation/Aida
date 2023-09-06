package utils

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
	return event{make(chan struct{})}
}

type event struct {
	channel chan struct{}
}

func (e event) HasHappened() bool {
	// Tests whether the event has already happened by
	// attempting to read from the owned channel.
	select {
	case <-e.channel:
		return true
	default:
		return false
	}
}

func (e event) Wait() <-chan struct{} {
	return e.channel
}

func (e event) Signal() {
	// Atomically checks whether the channel is already closed and
	// closes the channel if this is not the case.
	select {
	case <-e.channel:
		return
	default:
		close(e.channel)
	}
}
