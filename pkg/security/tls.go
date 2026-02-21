// Package security provides TLS 1.3 configuration helpers for mutual
// authentication between vehicles and the control-center gateway.
package security

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
)

// TLSConfig builds a crypto/tls.Config that enforces TLS 1.3 with
// mutual authentication (mTLS).
//
// Parameters:
//   - certFile: path to the PEM-encoded certificate of this endpoint.
//   - keyFile:  path to the PEM-encoded private key of this endpoint.
//   - caFile:   path to the PEM-encoded CA certificate used to verify the peer.
//
// Both the vehicle agent and the control-center gateway must call this
// function with their respective key-pairs and the shared CA certificate.
func TLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	caPEM, err := os.ReadFile(caFile) // #nosec G304 – caller-controlled path
	if err != nil {
		return nil, err
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("security: failed to parse CA certificate")
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}, nil
}

// ServerTLSConfig creates a TLS config for the server side (control center gateway).
// It requires the connecting client to present a valid certificate signed by caFile.
func ServerTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	cfg, err := TLSConfig(certFile, keyFile, caFile)
	if err != nil {
		return nil, err
	}
	cfg.ClientAuth = tls.RequireAndVerifyClientCert
	return cfg, nil
}

// ClientTLSConfig creates a TLS config for the client side (vehicle agent).
// It presents its own certificate and verifies the server certificate against caFile.
func ClientTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	cfg, err := TLSConfig(certFile, keyFile, caFile)
	if err != nil {
		return nil, err
	}
	// Client does not set ClientAuth – that field is server-side only.
	cfg.ClientAuth = tls.NoClientCert
	return cfg, nil
}
