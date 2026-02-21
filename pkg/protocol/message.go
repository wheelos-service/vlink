// Package protocol defines the wire messages and MQTT topic helpers used
// across the vlink communication framework.
package protocol

import (
	"encoding/json"
	"fmt"
	"time"
)

// Gear represents the vehicle's transmission gear.
type Gear int32

const (
	GearUnknown Gear = 0
	GearPark    Gear = 1
	GearDrive   Gear = 2
	GearReverse Gear = 3
	GearNeutral Gear = 4
)

// VehicleState is published by the vehicle at 10–50 Hz to v1/vehicle/{id}/state.
type VehicleState struct {
	VehicleID   string  `json:"vehicle_id"`
	Timestamp   int64   `json:"timestamp"` // Unix milliseconds
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Altitude    float64 `json:"altitude"`
	Speed       float32 `json:"speed"`       // m/s
	Heading     float32 `json:"heading"`     // degrees 0-360
	Gear        Gear    `json:"gear"`
	BatteryPct  float32 `json:"battery_pct"` // 0-100
	Mode        string  `json:"mode"`        // autonomous / manual / teleoperation
	Emergency   bool    `json:"emergency"`
}

// ControlCommand is published by the control center to v1/vehicle/{id}/control.
type ControlCommand struct {
	CommandID     string  `json:"command_id"`
	VehicleID     string  `json:"vehicle_id"`
	Timestamp     int64   `json:"timestamp"` // Unix milliseconds
	Action        string  `json:"action"`    // stop / resume / teleoperation_start
	TargetSpeed   float32 `json:"target_speed"`
	TargetHeading float32 `json:"target_heading"`
	Payload       string  `json:"payload"` // JSON-encoded extra parameters
}

// TeleoperationAlert is sent by the vehicle when human intervention is needed.
type TeleoperationAlert struct {
	VehicleID string  `json:"vehicle_id"`
	Timestamp int64   `json:"timestamp"` // Unix milliseconds
	Reason    string  `json:"reason"`    // extreme_weather / unmarked_construction
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Severity  int32   `json:"severity"` // 1 (low) – 3 (critical)
}

// NewVehicleState creates a VehicleState stamped with the current time.
func NewVehicleState(id string) *VehicleState {
	return &VehicleState{
		VehicleID: id,
		Timestamp: time.Now().UnixMilli(),
		Mode:      "autonomous",
	}
}

// Marshal serialises a message to JSON bytes.
func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal deserialises JSON bytes into the target struct.
func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// --- MQTT topic helpers ---

const topicPrefix = "v1/vehicle"

// StateTopic returns the state publish topic for a vehicle.
//
//	v1/vehicle/{id}/state
func StateTopic(vehicleID string) string {
	return fmt.Sprintf("%s/%s/state", topicPrefix, vehicleID)
}

// ControlTopic returns the control subscribe topic for a vehicle.
//
//	v1/vehicle/{id}/control
func ControlTopic(vehicleID string) string {
	return fmt.Sprintf("%s/%s/control", topicPrefix, vehicleID)
}

// AlertTopic returns the teleoperation alert topic for a vehicle.
//
//	v1/vehicle/{id}/alert
func AlertTopic(vehicleID string) string {
	return fmt.Sprintf("%s/%s/alert", topicPrefix, vehicleID)
}

// WildcardStateTopic returns a broker-side wildcard for all vehicle state topics.
func WildcardStateTopic() string {
	return fmt.Sprintf("%s/+/state", topicPrefix)
}

// WildcardAlertTopic returns a broker-side wildcard for all vehicle alert topics.
func WildcardAlertTopic() string {
	return fmt.Sprintf("%s/+/alert", topicPrefix)
}
