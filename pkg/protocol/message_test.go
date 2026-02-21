package protocol

import (
	"testing"
	"time"
)

func TestStateTopic(t *testing.T) {
	got := StateTopic("car-001")
	want := "v1/vehicle/car-001/state"
	if got != want {
		t.Errorf("StateTopic = %q, want %q", got, want)
	}
}

func TestControlTopic(t *testing.T) {
	got := ControlTopic("car-001")
	want := "v1/vehicle/car-001/control"
	if got != want {
		t.Errorf("ControlTopic = %q, want %q", got, want)
	}
}

func TestAlertTopic(t *testing.T) {
	got := AlertTopic("car-001")
	want := "v1/vehicle/car-001/alert"
	if got != want {
		t.Errorf("AlertTopic = %q, want %q", got, want)
	}
}

func TestWildcardTopics(t *testing.T) {
	if got := WildcardStateTopic(); got != "v1/vehicle/+/state" {
		t.Errorf("WildcardStateTopic = %q", got)
	}
	if got := WildcardAlertTopic(); got != "v1/vehicle/+/alert" {
		t.Errorf("WildcardAlertTopic = %q", got)
	}
}

func TestMarshalUnmarshalVehicleState(t *testing.T) {
	original := &VehicleState{
		VehicleID:  "car-001",
		Timestamp:  time.Now().UnixMilli(),
		Latitude:   39.9042,
		Longitude:  116.4074,
		Altitude:   50.0,
		Speed:      12.5,
		Heading:    90.0,
		Gear:       GearDrive,
		BatteryPct: 78.3,
		Mode:       "autonomous",
		Emergency:  false,
	}

	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	decoded := &VehicleState{}
	if err := Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.VehicleID != original.VehicleID {
		t.Errorf("VehicleID mismatch: got %q, want %q", decoded.VehicleID, original.VehicleID)
	}
	if decoded.Latitude != original.Latitude {
		t.Errorf("Latitude mismatch: got %v, want %v", decoded.Latitude, original.Latitude)
	}
	if decoded.Gear != original.Gear {
		t.Errorf("Gear mismatch: got %v, want %v", decoded.Gear, original.Gear)
	}
}

func TestMarshalUnmarshalControlCommand(t *testing.T) {
	cmd := &ControlCommand{
		CommandID: "cmd-xyz",
		VehicleID: "car-001",
		Timestamp: time.Now().UnixMilli(),
		Action:    "stop",
	}
	data, err := Marshal(cmd)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	decoded := &ControlCommand{}
	if err := Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.Action != cmd.Action {
		t.Errorf("Action mismatch: got %q, want %q", decoded.Action, cmd.Action)
	}
}

func TestNewVehicleState(t *testing.T) {
	before := time.Now().UnixMilli()
	s := NewVehicleState("car-001")
	after := time.Now().UnixMilli()

	if s.VehicleID != "car-001" {
		t.Errorf("VehicleID = %q", s.VehicleID)
	}
	if s.Timestamp < before || s.Timestamp > after {
		t.Errorf("Timestamp %d not in range [%d, %d]", s.Timestamp, before, after)
	}
	if s.Mode != "autonomous" {
		t.Errorf("Mode = %q, want autonomous", s.Mode)
	}
}
