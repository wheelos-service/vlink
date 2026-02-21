package teleoperation

import (
	"sync/atomic"
	"testing"

	"github.com/daohu527/vlink/pkg/protocol"
)

func TestHandleNotifiesListeners(t *testing.T) {
	h := NewHandler()

	var callCount int32
	h.Register(func(a *protocol.TeleoperationAlert) {
		atomic.AddInt32(&callCount, 1)
	})

	alert := NewAlert("car-001", "extreme_weather", 39.9042, 116.4074, 2)
	h.Handle(alert)

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("listener called %d times, want 1", callCount)
	}
}

func TestHandleMultipleListeners(t *testing.T) {
	h := NewHandler()

	var count int32
	for i := 0; i < 3; i++ {
		h.Register(func(a *protocol.TeleoperationAlert) {
			atomic.AddInt32(&count, 1)
		})
	}

	h.Handle(NewAlert("car-002", "unmarked_construction", 39.0, 116.0, 1))

	if atomic.LoadInt32(&count) != 3 {
		t.Errorf("listeners called %d times, want 3", count)
	}
}

func TestHandleCriticalSeverity(t *testing.T) {
	h := NewHandler()

	received := make(chan *protocol.TeleoperationAlert, 1)
	h.Register(func(a *protocol.TeleoperationAlert) {
		received <- a
	})

	alert := NewAlert("car-003", "sensor_failure", 0, 0, 3)
	h.Handle(alert)

	got := <-received
	if got.Severity != 3 {
		t.Errorf("Severity = %d, want 3", got.Severity)
	}
}

func TestNewAlert(t *testing.T) {
	a := NewAlert("car-001", "extreme_weather", 39.9042, 116.4074, 2)
	if a.VehicleID != "car-001" {
		t.Errorf("VehicleID = %q", a.VehicleID)
	}
	if a.Reason != "extreme_weather" {
		t.Errorf("Reason = %q", a.Reason)
	}
	if a.Severity != 2 {
		t.Errorf("Severity = %d", a.Severity)
	}
}
