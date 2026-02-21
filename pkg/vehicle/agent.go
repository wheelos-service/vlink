// Package vehicle provides the Vehicle Agent that runs on each autonomous
// vehicle. It connects to the MQTT broker, publishes state at a configured
// frequency (10–50 Hz), subscribes to control commands, and raises
// teleoperation alerts when needed.
package vehicle

import (
	"context"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/daohu527/vlink/pkg/protocol"
	"github.com/daohu527/vlink/pkg/security"
	"github.com/daohu527/vlink/pkg/teleoperation"
)

// Config holds the agent's runtime configuration.
type Config struct {
	// VehicleID is the unique identifier for this vehicle (e.g. "car-001").
	VehicleID string
	// BrokerURL is the MQTT broker address (e.g. "tls://broker:8883").
	BrokerURL string
	// PublishHz is the state publication frequency (10–50).
	PublishHz float64
	// CertFile, KeyFile, CAFile are paths for mTLS authentication.
	CertFile string
	KeyFile  string
	CAFile   string
}

// StateProvider is a function that the agent calls each tick to obtain the
// latest vehicle state. Implementations should return a fresh snapshot.
type StateProvider func() *protocol.VehicleState

// Agent manages the MQTT connection and state publishing loop.
type Agent struct {
	cfg       Config
	client    mqtt.Client
	alerter   *teleoperation.Handler
	stateFn   StateProvider
}

// New creates a new Agent. stateProvider is called each publish interval
// to obtain the current vehicle state.
func New(cfg Config, stateProvider StateProvider) *Agent {
	return &Agent{
		cfg:     cfg,
		alerter: teleoperation.NewHandler(),
		stateFn: stateProvider,
	}
}

// Connect establishes the MQTT connection. When CertFile, KeyFile and CAFile
// are set in Config, mutual TLS 1.3 authentication is used.
func (a *Agent) Connect() error {
	opts := mqtt.NewClientOptions().
		AddBroker(a.cfg.BrokerURL).
		SetClientID(a.cfg.VehicleID).
		SetCleanSession(false).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetOnConnectHandler(a.onConnect).
		SetConnectionLostHandler(a.onConnectionLost)

	if a.cfg.CertFile != "" && a.cfg.KeyFile != "" && a.cfg.CAFile != "" {
		tlsCfg, err := security.ClientTLSConfig(a.cfg.CertFile, a.cfg.KeyFile, a.cfg.CAFile)
		if err != nil {
			return fmt.Errorf("vehicle agent tls config: %w", err)
		}
		opts.SetTLSConfig(tlsCfg)
	}

	a.client = mqtt.NewClient(opts)

	token := a.client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("vehicle agent connect: %w", token.Error())
	}
	return nil
}

// ConnectWithClient is used in tests to inject a pre-configured mqtt.Client.
func (a *Agent) ConnectWithClient(c mqtt.Client) {
	a.client = c
}

// Run starts the state-publishing loop. It blocks until ctx is cancelled.
func (a *Agent) Run(ctx context.Context) error {
	if a.cfg.PublishHz <= 0 {
		a.cfg.PublishHz = 10
	}
	interval := time.Duration(float64(time.Second) / a.cfg.PublishHz)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := a.publishState(); err != nil {
				log.Printf("vehicle %s: publish error: %v", a.cfg.VehicleID, err)
			}
		}
	}
}

// RaiseAlert publishes a TeleoperationAlert and switches the vehicle mode to
// "teleoperation", increasing its heartbeat rate.
func (a *Agent) RaiseAlert(reason string, lat, lon float64, severity int32) error {
	alert := teleoperation.NewAlert(a.cfg.VehicleID, reason, lat, lon, severity)
	alert.Timestamp = time.Now().UnixMilli()

	data, err := protocol.Marshal(alert)
	if err != nil {
		return err
	}

	topic := protocol.AlertTopic(a.cfg.VehicleID)
	token := a.client.Publish(topic, 1, false, data)
	token.Wait()
	return token.Error()
}

// Disconnect gracefully closes the MQTT connection.
func (a *Agent) Disconnect() {
	if a.client != nil {
		a.client.Disconnect(250)
	}
}

// --- private ---

func (a *Agent) onConnect(c mqtt.Client) {
	log.Printf("vehicle %s: connected to broker", a.cfg.VehicleID)
	a.subscribeControl(c)
}

func (a *Agent) onConnectionLost(_ mqtt.Client, err error) {
	log.Printf("vehicle %s: connection lost: %v", a.cfg.VehicleID, err)
}

func (a *Agent) subscribeControl(c mqtt.Client) {
	topic := protocol.ControlTopic(a.cfg.VehicleID)
	token := c.Subscribe(topic, 1, a.handleControl)
	token.Wait()
	if err := token.Error(); err != nil {
		log.Printf("vehicle %s: subscribe %s error: %v", a.cfg.VehicleID, topic, err)
	}
}

func (a *Agent) handleControl(_ mqtt.Client, msg mqtt.Message) {
	cmd := &protocol.ControlCommand{}
	if err := protocol.Unmarshal(msg.Payload(), cmd); err != nil {
		log.Printf("vehicle %s: bad control message: %v", a.cfg.VehicleID, err)
		return
	}
	log.Printf("vehicle %s: received command action=%s speed=%.1f heading=%.1f",
		a.cfg.VehicleID, cmd.Action, cmd.TargetSpeed, cmd.TargetHeading)
}

func (a *Agent) publishState() error {
	state := a.stateFn()
	state.Timestamp = time.Now().UnixMilli()

	data, err := protocol.Marshal(state)
	if err != nil {
		return err
	}

	topic := protocol.StateTopic(a.cfg.VehicleID)
	token := a.client.Publish(topic, 0, false, data)
	token.Wait()
	return token.Error()
}
