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
