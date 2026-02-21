// Package teleoperation implements the remote-takeover (远程接管) subsystem.
//
// When a vehicle encounters a scene it cannot handle autonomously (extreme
// weather, unmarked construction zones, etc.) it raises a TeleoperationAlert.
// The Handler at the control center receives the alert, logs it, and notifies
// registered listeners (e.g. human operators, video-stream starters).
package teleoperation

import (
	"log"
	"sync"

	"github.com/daohu527/vlink/pkg/protocol"
)

// AlertListener is called whenever a new TeleoperationAlert is received.
type AlertListener func(alert *protocol.TeleoperationAlert)

// Handler manages incoming teleoperation alerts.
type Handler struct {
	mu        sync.RWMutex
	listeners []AlertListener
}

// NewHandler creates a Handler with no listeners registered.
func NewHandler() *Handler {
	return &Handler{}
}

// Register adds a listener that will be called for every incoming alert.
func (h *Handler) Register(l AlertListener) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.listeners = append(h.listeners, l)
}

// Handle processes an incoming alert: logs it and notifies all listeners.
// Severity 3 (critical) is logged at a higher priority.
func (h *Handler) Handle(alert *protocol.TeleoperationAlert) {
	if alert.Severity >= 3 {
		log.Printf("[CRITICAL] teleoperation alert from vehicle %s: %s (lat=%.6f lon=%.6f)",
			alert.VehicleID, alert.Reason, alert.Latitude, alert.Longitude)
	} else {
		log.Printf("[WARN] teleoperation alert from vehicle %s: %s severity=%d",
			alert.VehicleID, alert.Reason, alert.Severity)
	}

	h.mu.RLock()
	ls := make([]AlertListener, len(h.listeners))
	copy(ls, h.listeners)
	h.mu.RUnlock()

	for _, l := range ls {
		l(alert)
	}
}

// NewAlert is a convenience constructor for vehicle code that needs to raise
// a teleoperation alert.
func NewAlert(vehicleID, reason string, lat, lon float64, severity int32) *protocol.TeleoperationAlert {
	return &protocol.TeleoperationAlert{
		VehicleID: vehicleID,
		Reason:    reason,
		Latitude:  lat,
		Longitude: lon,
		Severity:  severity,
	}
}
