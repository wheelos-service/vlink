# vlink

Vehicle remote monitoring — communication framework between autonomous vehicles and the monitoring center.

## Overview

vlink implements a **cloud–network–edge** communication framework designed to solve four key challenges in autonomous-vehicle operations:

| Challenge | Solution |
|---|---|
| High concurrency | MQTT pub/sub with QoS 1, auto-reconnect |
| Low latency | State published at 10–50 Hz over a persistent connection |
| High reliability | Auto-reconnect, at-least-once delivery (QoS 1), out-of-order drop |
| Security | Mutual TLS 1.3 (`crypto/tls.VersionTLS13`) with per-component certificates |

## Architecture

```
Vehicle (端)  ──5G/TLS──►  MQTT Broker  ──TLS──►  Control Center (云)
   │                                                      │
   │  v1/vehicle/{id}/state  (10–50 Hz)                  │  shadow.Manager
   │  v1/vehicle/{id}/alert  (on-demand)                 │  teleoperation.Handler
   │◄──────────────────────────────────────────────────── │
      v1/vehicle/{id}/control (on-demand)
```

## Package Structure

```
vlink/
├── cmd/
│   ├── vehicle/          # Vehicle agent daemon
│   └── control-center/   # Monitoring center server
├── pkg/
│   ├── protocol/         # Message types (VehicleState, ControlCommand, TeleoperationAlert) + topic helpers
│   ├── security/         # TLS 1.3 / mTLS configuration
│   ├── vehicle/          # Vehicle agent (MQTT publisher / control subscriber)
│   ├── shadow/           # Digital twin — per-vehicle in-memory state replica
│   ├── controlcenter/    # Control center server (state subscriber, command publisher)
│   └── teleoperation/    # Teleoperation alert handler
└── proto/
    └── vehicle.proto     # Protobuf schema (reference)
```

## MQTT Topics

| Topic | Direction | Purpose |
|---|---|---|
| `v1/vehicle/{id}/state` | Vehicle → Center | Vehicle state at 10–50 Hz |
| `v1/vehicle/{id}/control` | Center → Vehicle | Control commands (stop/resume/teleoperation_start) |
| `v1/vehicle/{id}/alert` | Vehicle → Center | Teleoperation alert (extreme weather, construction, etc.) |

## Running

### Vehicle agent

```sh
go run ./cmd/vehicle \
  -id       car-001 \
  -broker   tls://broker:8883 \
  -cert     /etc/vlink/certs/vehicle.crt \
  -key      /etc/vlink/certs/vehicle.key \
  -ca       /etc/vlink/certs/ca.crt \
  -hz       20
```

### Control center server

```sh
go run ./cmd/control-center \
  -broker    tls://broker:8883 \
  -client-id control-center-01 \
  -cert      /etc/vlink/certs/cc.crt \
  -key       /etc/vlink/certs/cc.key \
  -ca        /etc/vlink/certs/ca.crt
```

## Tests

```sh
go test ./...
```
