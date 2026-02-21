package shadow

import (
	"testing"
	"time"

	"github.com/daohu527/vlink/pkg/protocol"
)

func makeState(id string, ts int64) *protocol.VehicleState {
	return &protocol.VehicleState{
		VehicleID: id,
		Timestamp: ts,
		Mode:      "autonomous",
	}
}

func TestUpdateAndGet(t *testing.T) {
	m := NewManager()

	s := makeState("car-001", time.Now().UnixMilli())
	m.Update(s)

	entry, ok := m.Get("car-001")
	if !ok {
		t.Fatal("expected entry to exist")
	}
	if entry.State.VehicleID != "car-001" {
		t.Errorf("VehicleID = %q", entry.State.VehicleID)
	}
}

func TestGetMissing(t *testing.T) {
	m := NewManager()
	if _, ok := m.Get("unknown"); ok {
		t.Error("expected no entry for unknown vehicle")
	}
}

func TestUpdateDropsStaleMessages(t *testing.T) {
	m := NewManager()
	now := time.Now().UnixMilli()

	m.Update(makeState("car-001", now))
	m.Update(makeState("car-001", now-1000)) // older — should be dropped

	entry, _ := m.Get("car-001")
	if entry.State.Timestamp != now {
		t.Errorf("Timestamp = %d, want %d (stale update should be dropped)", entry.State.Timestamp, now)
	}
}

func TestUpdateOverwritesWithNewer(t *testing.T) {
	m := NewManager()
	now := time.Now().UnixMilli()

	m.Update(makeState("car-001", now))
	m.Update(makeState("car-001", now+1000)) // newer

	entry, _ := m.Get("car-001")
	if entry.State.Timestamp != now+1000 {
		t.Errorf("Timestamp = %d, want %d", entry.State.Timestamp, now+1000)
	}
}

func TestAll(t *testing.T) {
	m := NewManager()
	now := time.Now().UnixMilli()
	m.Update(makeState("car-001", now))
	m.Update(makeState("car-002", now))

	all := m.All()
	if len(all) != 2 {
		t.Errorf("len(All) = %d, want 2", len(all))
	}
}

func TestActiveVehicles(t *testing.T) {
	m := NewManager()

	m.Update(makeState("car-001", time.Now().UnixMilli()))

	// Inject an old entry manually.
	m.mu.Lock()
	m.shadows["car-old"] = &Entry{
		State:     makeState("car-old", time.Now().UnixMilli()-10000),
		UpdatedAt: time.Now().Add(-10 * time.Minute),
	}
	m.mu.Unlock()

	active := m.ActiveVehicles(time.Minute)
	if len(active) != 1 || active[0] != "car-001" {
		t.Errorf("ActiveVehicles = %v, want [car-001]", active)
	}
}

func TestRemove(t *testing.T) {
	m := NewManager()
	m.Update(makeState("car-001", time.Now().UnixMilli()))
	m.Remove("car-001")

	if _, ok := m.Get("car-001"); ok {
		t.Error("entry should have been removed")
	}
}
