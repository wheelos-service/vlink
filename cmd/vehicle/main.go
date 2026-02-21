// Command vehicle is the vehicle-agent daemon.
//
// It connects to the MQTT broker and continuously publishes vehicle state
// at the configured frequency, subscribing to control commands from the
// monitoring center.
//
// Usage:
//
//	vehicle -id car-001 -broker tls://broker:8883 \
//	        -cert /etc/vlink/certs/vehicle.crt \
//	        -key  /etc/vlink/certs/vehicle.key  \
//	        -ca   /etc/vlink/certs/ca.crt       \
//	        -hz 20
package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"

	"github.com/daohu527/vlink/pkg/protocol"
	"github.com/daohu527/vlink/pkg/vehicle"
)

func main() {
	id := flag.String("id", "car-001", "unique vehicle ID")
	broker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	certFile := flag.String("cert", "", "path to vehicle TLS certificate")
	keyFile := flag.String("key", "", "path to vehicle TLS private key")
	caFile := flag.String("ca", "", "path to CA certificate")
	hz := flag.Float64("hz", 10, "state publish frequency (10-50 Hz)")
	flag.Parse()

	if *id == "" {
		log.Fatal("vehicle id must not be empty")
	}

	cfg := vehicle.Config{
		VehicleID: *id,
		BrokerURL: *broker,
		CertFile:  *certFile,
		KeyFile:   *keyFile,
		CAFile:    *caFile,
		PublishHz: *hz,
	}

	agent := vehicle.New(cfg, func() *protocol.VehicleState {
		// In production this would read from real sensors.
		return &protocol.VehicleState{
			VehicleID:  *id,
			Latitude:   39.9042 + (rand.Float64()-0.5)*0.01,
			Longitude:  116.4074 + (rand.Float64()-0.5)*0.01,
			Speed:      float32(10 + rand.Float64()*5),
			Heading:    float32(rand.Float64() * 360),
			Gear:       protocol.GearDrive,
			BatteryPct: 80,
			Mode:       "autonomous",
		}
	})

	if err := agent.Connect(); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer agent.Disconnect()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("vehicle agent %s started at %.0f Hz", *id, *hz)
	if err := agent.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("run: %v", err)
	}
	log.Printf("vehicle agent %s stopped", *id)
}
