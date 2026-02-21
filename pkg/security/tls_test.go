package security

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

// generateTestCerts writes a self-signed CA + leaf cert+key into dir and
// returns (certFile, keyFile, caFile).
func generateTestCerts(t *testing.T) (certFile, keyFile, caFile string) {
	t.Helper()

	dir := t.TempDir()

	// Generate CA key pair (ECDSA P-256)
	caKey, err := newECDSAKey()
	if err != nil {
		t.Fatalf("CA key: %v", err)
	}
	caCert, err := selfSignedCA(caKey)
	if err != nil {
		t.Fatalf("CA cert: %v", err)
	}

	// Generate leaf key pair signed by CA
	leafKey, err := newECDSAKey()
	if err != nil {
		t.Fatalf("leaf key: %v", err)
	}
	leafCert, err := signedLeaf(leafKey, caCert, caKey)
	if err != nil {
		t.Fatalf("leaf cert: %v", err)
	}

	caFile = filepath.Join(dir, "ca.pem")
	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")

	writePEM(t, caFile, "CERTIFICATE", caCert.Raw)
	writePEM(t, certFile, "CERTIFICATE", leafCert.Raw)
	writeKeyPEM(t, keyFile, leafKey)

	return certFile, keyFile, caFile
}

func TestTLSConfigMinVersion(t *testing.T) {
	certFile, keyFile, caFile := generateTestCerts(t)

	cfg, err := TLSConfig(certFile, keyFile, caFile)
	if err != nil {
		t.Fatalf("TLSConfig: %v", err)
	}
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Errorf("MinVersion = %d, want TLS 1.3 (%d)", cfg.MinVersion, tls.VersionTLS13)
	}
}

func TestTLSConfigCertLoaded(t *testing.T) {
	certFile, keyFile, caFile := generateTestCerts(t)

	cfg, err := TLSConfig(certFile, keyFile, caFile)
	if err != nil {
		t.Fatalf("TLSConfig: %v", err)
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("len(Certificates) = %d, want 1", len(cfg.Certificates))
	}
}

func TestTLSConfigMissingFiles(t *testing.T) {
	if _, err := TLSConfig("/no/such/cert.pem", "/no/such/key.pem", "/no/such/ca.pem"); err == nil {
		t.Error("expected error for missing files, got nil")
	}
}

func TestServerTLSConfigRequiresClientCert(t *testing.T) {
	certFile, keyFile, caFile := generateTestCerts(t)

	cfg, err := ServerTLSConfig(certFile, keyFile, caFile)
	if err != nil {
		t.Fatalf("ServerTLSConfig: %v", err)
	}
	if cfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("ClientAuth = %v, want RequireAndVerifyClientCert", cfg.ClientAuth)
	}
}

func TestClientTLSConfigNoClientAuth(t *testing.T) {
	certFile, keyFile, caFile := generateTestCerts(t)

	cfg, err := ClientTLSConfig(certFile, keyFile, caFile)
	if err != nil {
		t.Fatalf("ClientTLSConfig: %v", err)
	}
	if cfg.ClientAuth != tls.NoClientCert {
		t.Errorf("ClientAuth = %v, want NoClientCert", cfg.ClientAuth)
	}
}

// --- helpers ---

func writePEM(t *testing.T, path, blockType string, data []byte) {
	t.Helper()
	f, err := os.Create(path) // #nosec G304
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: blockType, Bytes: data}); err != nil {
		t.Fatalf("pem encode %s: %v", path, err)
	}
}

func writeKeyPEM(t *testing.T, path string, key any) {
	t.Helper()
	der, err := marshalKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	writePEM(t, path, "EC PRIVATE KEY", der)
}

// helpers in a separate file (cert_helpers_test.go) so the test file is not too long.
var _ = x509.NewCertPool // import used
