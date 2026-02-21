// Package shadow implements the Digital Twin (影子系统) for the control center.
// It maintains an in-memory replica of every connected vehicle's last known
// state, enabling the control center to answer queries without waiting for a
// new state message from the vehicle.
package shadow

import (
	"sync"
	"time"

	"github.com/daohu527/vlink/pkg/protocol"
)

// Entry is the shadow record for a single vehicle.
type Entry struct {
	State     *protocol.VehicleState
	UpdatedAt time.Time
}

// Manager stores and queries vehicle shadow state.
type Manager struct {
	mu      sync.RWMutex
	shadows map[string]*Entry
}

// NewManager creates an empty shadow Manager.
func NewManager() *Manager {
	return &Manager{
		shadows: make(map[string]*Entry),
	}
}

// Update stores (or replaces) the shadow for the vehicle identified by state.VehicleID.
// Out-of-order updates (older timestamp than the stored one) are silently dropped.
func (m *Manager) Update(state *protocol.VehicleState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.shadows[state.VehicleID]
	if ok && existing.State.Timestamp > state.Timestamp {
		// Drop stale update.
		return
	}

	m.shadows[state.VehicleID] = &Entry{
		State:     state,
		UpdatedAt: time.Now(),
	}
}

// Get returns the shadow entry for vehicleID, or (nil, false) if not found.
func (m *Manager) Get(vehicleID string) (*Entry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	e, ok := m.shadows[vehicleID]
	return e, ok
}

// All returns a snapshot of all current shadow entries keyed by vehicle ID.
func (m *Manager) All() map[string]*Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*Entry, len(m.shadows))
	for id, e := range m.shadows {
		result[id] = e
	}
	return result
}

// ActiveVehicles returns IDs of vehicles whose last update is within maxAge.
func (m *Manager) ActiveVehicles(maxAge time.Duration) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cutoff := time.Now().Add(-maxAge)
	ids := make([]string, 0)
	for id, e := range m.shadows {
		if e.UpdatedAt.After(cutoff) {
			ids = append(ids, id)
		}
	}
	return ids
}

// Remove deletes the shadow entry for vehicleID.
func (m *Manager) Remove(vehicleID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.shadows, vehicleID)
}
