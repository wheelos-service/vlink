// Command control-center is the monitoring-center server daemon.
//
// It connects to the MQTT broker, subscribes to all vehicle state and alert
// topics, maintains a digital-twin shadow for each vehicle, and is ready to
// send control commands (e.g., stop, resume, teleoperation_start).
//
// Usage:
//
//	control-center -broker tls://broker:8883 \
//	               -cert /etc/vlink/certs/cc.crt \
//	               -key  /etc/vlink/certs/cc.key  \
//	               -ca   /etc/vlink/certs/ca.crt
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/daohu527/vlink/pkg/controlcenter"
	"github.com/daohu527/vlink/pkg/protocol"
)

func main() {
	broker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	clientID := flag.String("client-id", "control-center-01", "MQTT client ID")
	certFile := flag.String("cert", "", "path to TLS certificate")
	keyFile := flag.String("key", "", "path to TLS private key")
	caFile := flag.String("ca", "", "path to CA certificate")
	flag.Parse()

	cfg := controlcenter.Config{
		BrokerURL: *broker,
		ClientID:  *clientID,
		CertFile:  *certFile,
		KeyFile:   *keyFile,
		CAFile:    *caFile,
	}

	srv := controlcenter.New(cfg)

	// Register a simple teleoperation listener that logs the alert.
	srv.Alerter().Register(func(alert *protocol.TeleoperationAlert) {
		log.Printf("[OPERATOR] vehicle %s needs takeover: %s (severity %d)",
			alert.VehicleID, alert.Reason, alert.Severity)
		// In production: trigger video stream, notify operator dashboard, etc.
	})

	if err := srv.Connect(); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer srv.Disconnect()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("control-center %s started", *clientID)

	// Periodically print a summary of known vehicles.
	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				all := srv.Shadows().All()
				log.Printf("shadow summary: %d vehicle(s) tracked", len(all))
			}
		}
	}()

	<-ctx.Done()
	log.Printf("control-center %s stopped", *clientID)
}
