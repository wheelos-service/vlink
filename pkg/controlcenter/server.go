// Package controlcenter implements the monitoring / control-center server.
// It connects to the MQTT broker, subscribes to all vehicle state and alert
// topics, keeps the shadow system up-to-date, and forwards teleoperation
// alerts to registered operators.
package controlcenter

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/daohu527/vlink/pkg/protocol"
	"github.com/daohu527/vlink/pkg/security"
	"github.com/daohu527/vlink/pkg/shadow"
	"github.com/daohu527/vlink/pkg/teleoperation"
)

// Config holds the control-center configuration.
type Config struct {
	// BrokerURL is the MQTT broker address (e.g. "tls://broker:8883").
	BrokerURL string
	// ClientID is the MQTT client ID for the control center.
	ClientID string
	// CertFile, KeyFile, CAFile are paths for mTLS authentication.
	CertFile string
	KeyFile  string
	CAFile   string
}

// Server is the control-center MQTT server.
type Server struct {
	cfg     Config
	client  mqtt.Client
	shadows *shadow.Manager
	alerter *teleoperation.Handler
}

// New creates a Server with a fresh shadow manager and teleoperation handler.
func New(cfg Config) *Server {
	return &Server{
		cfg:     cfg,
		shadows: shadow.NewManager(),
		alerter: teleoperation.NewHandler(),
	}
}

// Shadows returns the digital-twin manager (read-only access for callers).
func (s *Server) Shadows() *shadow.Manager { return s.shadows }

// Alerter returns the teleoperation handler so callers can register listeners.
func (s *Server) Alerter() *teleoperation.Handler { return s.alerter }

// Connect establishes the MQTT connection. When CertFile, KeyFile and CAFile
// are set in Config, mutual TLS 1.3 authentication is used.
func (s *Server) Connect() error {
	opts := mqtt.NewClientOptions().
		AddBroker(s.cfg.BrokerURL).
		SetClientID(s.cfg.ClientID).
		SetCleanSession(false).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetOnConnectHandler(s.onConnect).
		SetConnectionLostHandler(s.onConnectionLost)

	if s.cfg.CertFile != "" && s.cfg.KeyFile != "" && s.cfg.CAFile != "" {
		tlsCfg, err := security.ServerTLSConfig(s.cfg.CertFile, s.cfg.KeyFile, s.cfg.CAFile)
		if err != nil {
			return fmt.Errorf("control-center tls config: %w", err)
		}
		opts.SetTLSConfig(tlsCfg)
	}

	s.client = mqtt.NewClient(opts)

	token := s.client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("control-center connect: %w", token.Error())
	}
	return nil
}

// ConnectWithClient injects a pre-configured client (used in tests).
func (s *Server) ConnectWithClient(c mqtt.Client) {
	s.client = c
	s.subscribeTopics(c)
}

// SendControl publishes a ControlCommand to the given vehicle.
func (s *Server) SendControl(cmd *protocol.ControlCommand) error {
	cmd.Timestamp = time.Now().UnixMilli()

	data, err := protocol.Marshal(cmd)
	if err != nil {
		return err
	}

	topic := protocol.ControlTopic(cmd.VehicleID)
	token := s.client.Publish(topic, 1, false, data)
	token.Wait()
	return token.Error()
}

// Disconnect gracefully closes the MQTT connection.
func (s *Server) Disconnect() {
	if s.client != nil {
		s.client.Disconnect(250)
	}
}

// --- private ---

func (s *Server) onConnect(c mqtt.Client) {
	log.Printf("control-center %s: connected to broker", s.cfg.ClientID)
	s.subscribeTopics(c)
}

func (s *Server) onConnectionLost(_ mqtt.Client, err error) {
	log.Printf("control-center %s: connection lost: %v", s.cfg.ClientID, err)
}

func (s *Server) subscribeTopics(c mqtt.Client) {
	topics := map[string]mqtt.MessageHandler{
		protocol.WildcardStateTopic(): s.handleState,
		protocol.WildcardAlertTopic(): s.handleAlert,
	}
	for topic, handler := range topics {
		token := c.Subscribe(topic, 1, handler)
		token.Wait()
		if err := token.Error(); err != nil {
			log.Printf("control-center: subscribe %s error: %v", topic, err)
		}
	}
}

func (s *Server) handleState(_ mqtt.Client, msg mqtt.Message) {
	state := &protocol.VehicleState{}
	if err := protocol.Unmarshal(msg.Payload(), state); err != nil {
		log.Printf("control-center: bad state message on %s: %v", msg.Topic(), err)
		return
	}
	s.shadows.Update(state)
}

func (s *Server) handleAlert(_ mqtt.Client, msg mqtt.Message) {
	alert := &protocol.TeleoperationAlert{}
	if err := protocol.Unmarshal(msg.Payload(), alert); err != nil {
		log.Printf("control-center: bad alert message on %s: %v", msg.Topic(), err)
		return
	}
	s.alerter.Handle(alert)
}
