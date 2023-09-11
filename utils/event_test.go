package utils

import "testing"

func TestEvent_IsNotTriggeredAtCreationTime(t *testing.T) {
	event := MakeEvent()
	if event.HasHappened() {
		t.Errorf("fresh event should not be marked as happened")
	}
}

func TestEvent_SignalingTheEventMarksItAsHappened(t *testing.T) {
	event := MakeEvent()
	if event.HasHappened() {
		t.Errorf("fresh event should not be marked as happened")
	}
	event.Signal()
	if !event.HasHappened() {
		t.Errorf("a signaled event should be reported as happened")
	}
}

func TestEvent_EventsCanBeSignaledMultipleTimes(t *testing.T) {
	event := MakeEvent()
	for i := 0; i < 10; i++ {
		event.Signal()
		if !event.HasHappened() {
			t.Errorf("a signaled event should be reported as happened")
		}
	}
}

func TestEvent_SignalingAnEventReleasesWaitingGoRoutines(t *testing.T) {
	event := MakeEvent()

	i := 0
	running := make(chan bool)
	done := make(chan bool)
	go func() {
		i = 1
		close(running)
		<-event.Wait()
		i = 2
		close(done)
	}()

	<-running
	if i != 1 {
		t.Fatalf("test runner not properly started")
	}
	event.Signal()
	<-done
	if i != 2 {
		t.Fatalf("waiting goroutine not released")
	}
}
