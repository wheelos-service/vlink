package vehicle

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/daohu527/vlink/pkg/protocol"
)

// --- mock MQTT client ---

type mockMessage struct {
	topic   string
	payload []byte
}

func (m *mockMessage) Duplicate() bool       { return false }
func (m *mockMessage) Qos() byte             { return 1 }
func (m *mockMessage) Retained() bool        { return false }
func (m *mockMessage) Topic() string         { return m.topic }
func (m *mockMessage) MessageID() uint16     { return 0 }
func (m *mockMessage) Payload() []byte       { return m.payload }
func (m *mockMessage) Ack()                  {}

type mockToken struct{}

func (t *mockToken) Wait() bool                        { return true }
func (t *mockToken) WaitTimeout(time.Duration) bool    { return true }
func (t *mockToken) Done() <-chan struct{}              { ch := make(chan struct{}); close(ch); return ch }
func (t *mockToken) Error() error                      { return nil }

type mockClient struct {
	mu        sync.Mutex
	published []mockMessage
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
	c.mu.Lock()
	defer c.mu.Unlock()
	var p []byte
	switch v := payload.(type) {
	case []byte:
		p = v
	case string:
		p = []byte(v)
	}
	c.published = append(c.published, mockMessage{topic: topic, payload: p})
	return &mockToken{}
}
func (c *mockClient) Subscribe(topic string, _ byte, h mqtt.MessageHandler) mqtt.Token {
	c.mu.Lock()
	c.handlers[topic] = h
	c.mu.Unlock()
	return &mockToken{}
}
func (c *mockClient) SubscribeMultiple(map[string]byte, mqtt.MessageHandler) mqtt.Token {
	return &mockToken{}
}
func (c *mockClient) Unsubscribe(...string) mqtt.Token  { return &mockToken{} }
func (c *mockClient) AddRoute(string, mqtt.MessageHandler) {}
func (c *mockClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.NewClient(mqtt.NewClientOptions()).OptionsReader()
}

// --- tests ---

func stateProvider(id string) StateProvider {
	return func() *protocol.VehicleState {
		return &protocol.VehicleState{
			VehicleID:  id,
			Timestamp:  time.Now().UnixMilli(),
			Latitude:   39.9042,
			Longitude:  116.4074,
			Speed:      10.0,
			Mode:       "autonomous",
		}
	}
}

func TestAgentPublishesState(t *testing.T) {
	cfg := Config{VehicleID: "car-001", PublishHz: 20}
	agent := New(cfg, stateProvider("car-001"))
	mc := newMockClient()
	agent.ConnectWithClient(mc)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = agent.Run(ctx) // returns ctx.Err() = context.DeadlineExceeded

	mc.mu.Lock()
	count := len(mc.published)
	mc.mu.Unlock()

	// At 20 Hz over 200 ms we expect roughly 4 publishes (allow some slack).
	if count < 2 {
		t.Errorf("expected at least 2 published messages, got %d", count)
	}
}

func TestAgentStateTopicFormat(t *testing.T) {
	cfg := Config{VehicleID: "car-001", PublishHz: 10}
	agent := New(cfg, stateProvider("car-001"))
	mc := newMockClient()
	agent.ConnectWithClient(mc)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	_ = agent.Run(ctx)

	mc.mu.Lock()
	defer mc.mu.Unlock()
	if len(mc.published) == 0 {
		t.Fatal("no messages published")
	}
	want := protocol.StateTopic("car-001")
	got := mc.published[0].topic
	if got != want {
		t.Errorf("topic = %q, want %q", got, want)
	}
}

func TestAgentPublishedPayloadDecodable(t *testing.T) {
	cfg := Config{VehicleID: "car-001", PublishHz: 10}
	agent := New(cfg, stateProvider("car-001"))
	mc := newMockClient()
	agent.ConnectWithClient(mc)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	_ = agent.Run(ctx)

	mc.mu.Lock()
	defer mc.mu.Unlock()
	if len(mc.published) == 0 {
		t.Fatal("no messages published")
	}
	var state protocol.VehicleState
	if err := json.Unmarshal(mc.published[0].payload, &state); err != nil {
		t.Fatalf("could not unmarshal payload: %v", err)
	}
	if state.VehicleID != "car-001" {
		t.Errorf("VehicleID = %q", state.VehicleID)
	}
}

func TestAgentHandlesControlCommand(t *testing.T) {
	cfg := Config{VehicleID: "car-001", PublishHz: 10}
	agent := New(cfg, stateProvider("car-001"))
	mc := newMockClient()
	agent.ConnectWithClient(mc)

	// Simulate control message delivery by calling the handler directly.
	agent.subscribeControl(mc)
	handler := mc.handlers[protocol.ControlTopic("car-001")]
	if handler == nil {
		t.Fatal("no handler registered for control topic")
	}

	cmd := &protocol.ControlCommand{
		CommandID: "cmd-1",
		VehicleID: "car-001",
		Action:    "stop",
	}
	data, _ := protocol.Marshal(cmd)
	handler(mc, &mockMessage{topic: protocol.ControlTopic("car-001"), payload: data})
	// Verify no panic; command is logged.
}
