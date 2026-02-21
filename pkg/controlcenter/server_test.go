package controlcenter

import (
	"sync/atomic"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/daohu527/vlink/pkg/protocol"
)

// --- reuse the mockClient / mockMessage / mockToken from vehicle tests,
// duplicated here to keep packages independent. ---

type mockMessage struct {
	topic   string
	payload []byte
}

func (m *mockMessage) Duplicate() bool   { return false }
func (m *mockMessage) Qos() byte         { return 1 }
func (m *mockMessage) Retained() bool    { return false }
func (m *mockMessage) Topic() string     { return m.topic }
func (m *mockMessage) MessageID() uint16 { return 0 }
func (m *mockMessage) Payload() []byte   { return m.payload }
func (m *mockMessage) Ack()              {}

type mockToken struct{}

func (t *mockToken) Wait() bool                     { return true }
func (t *mockToken) WaitTimeout(time.Duration) bool { return true }
func (t *mockToken) Done() <-chan struct{}           { ch := make(chan struct{}); close(ch); return ch }
func (t *mockToken) Error() error                   { return nil }

type mockClient struct {
	published []struct{ topic string; payload []byte }
	handlers  map[string]mqtt.MessageHandler
}

func newMockClient() *mockClient {
	return &mockClient{handlers: make(map[string]mqtt.MessageHandler)}
}

func (c *mockClient) IsConnected() bool                                    { return true }
func (c *mockClient) IsConnectionOpen() bool                               { return true }
func (c *mockClient) Connect() mqtt.Token                                  { return &mockToken{} }
func (c *mockClient) Disconnect(uint)                                      {}
func (c *mockClient) Publish(topic string, _ byte, _ bool, payload interface{}) mqtt.Token {
	var p []byte
	switch v := payload.(type) {
	case []byte:
		p = v
	case string:
		p = []byte(v)
	}
	c.published = append(c.published, struct{ topic string; payload []byte }{topic, p})
	return &mockToken{}
}
func (c *mockClient) Subscribe(topic string, _ byte, h mqtt.MessageHandler) mqtt.Token {
	c.handlers[topic] = h
	return &mockToken{}
}
func (c *mockClient) SubscribeMultiple(map[string]byte, mqtt.MessageHandler) mqtt.Token {
	return &mockToken{}
}
func (c *mockClient) Unsubscribe(...string) mqtt.Token            { return &mockToken{} }
func (c *mockClient) AddRoute(string, mqtt.MessageHandler)        {}
func (c *mockClient) OptionsReader() mqtt.ClientOptionsReader     {
	return mqtt.NewClient(mqtt.NewClientOptions()).OptionsReader()
}

// ---

func TestServerUpdatesShadowOnStateMessage(t *testing.T) {
	srv := New(Config{ClientID: "cc"})
	mc := newMockClient()
	srv.ConnectWithClient(mc)

	state := &protocol.VehicleState{
		VehicleID: "car-001",
		Timestamp: time.Now().UnixMilli(),
		Mode:      "autonomous",
	}
	data, _ := protocol.Marshal(state)

	handler := mc.handlers[protocol.WildcardStateTopic()]
	if handler == nil {
		t.Fatal("no handler for wildcard state topic")
	}
	handler(mc, &mockMessage{topic: protocol.StateTopic("car-001"), payload: data})

	entry, ok := srv.Shadows().Get("car-001")
	if !ok {
		t.Fatal("shadow not updated")
	}
	if entry.State.VehicleID != "car-001" {
		t.Errorf("VehicleID = %q", entry.State.VehicleID)
	}
}

func TestServerForwardsAlerts(t *testing.T) {
	srv := New(Config{ClientID: "cc"})
	mc := newMockClient()
	srv.ConnectWithClient(mc)

	var called int32
	srv.Alerter().Register(func(a *protocol.TeleoperationAlert) {
		atomic.AddInt32(&called, 1)
	})

	alert := &protocol.TeleoperationAlert{
		VehicleID: "car-001",
		Reason:    "extreme_weather",
		Severity:  3,
	}
	data, _ := protocol.Marshal(alert)

	handler := mc.handlers[protocol.WildcardAlertTopic()]
	if handler == nil {
		t.Fatal("no handler for wildcard alert topic")
	}
	handler(mc, &mockMessage{topic: protocol.AlertTopic("car-001"), payload: data})

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("alert listener called %d times, want 1", called)
	}
}

func TestServerSendControl(t *testing.T) {
	srv := New(Config{ClientID: "cc"})
	mc := newMockClient()
	srv.ConnectWithClient(mc)

	cmd := &protocol.ControlCommand{
		CommandID: "cmd-1",
		VehicleID: "car-001",
		Action:    "stop",
	}
	if err := srv.SendControl(cmd); err != nil {
		t.Fatalf("SendControl: %v", err)
	}

	if len(mc.published) != 1 {
		t.Fatalf("published %d messages, want 1", len(mc.published))
	}
	got := mc.published[0].topic
	want := protocol.ControlTopic("car-001")
	if got != want {
		t.Errorf("topic = %q, want %q", got, want)
	}
}
