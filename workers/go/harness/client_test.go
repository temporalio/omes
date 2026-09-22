package harness

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// cert-manager's tls-combined.pem layout: private key, newline, certificate.
func writeTestPEMs(t *testing.T) (combinedPath, certPath, keyPath string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "client.ns.tmprl-test.cloud"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	dir := t.TempDir()
	write := func(name string, data []byte) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	return write("tls-combined.pem", bytes.Join([][]byte{keyPEM, certPEM}, []byte("\n"))),
		write("tls.crt", certPEM),
		write("tls.key", keyPEM)
}

func TestBuildTLSConfig(t *testing.T) {
	combinedPath, certPath, keyPath := writeTestPEMs(t)

	t.Run("combined path carries both halves", func(t *testing.T) {
		cfg, err := buildTLSConfig(clientConfigOptions{TLSCombinedPath: combinedPath})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Certificates) != 1 {
			t.Fatalf("got %d certificates, want 1", len(cfg.Certificates))
		}
	})

	t.Run("cert and key paths load separately", func(t *testing.T) {
		cfg, err := buildTLSConfig(clientConfigOptions{TLSCertPath: certPath, TLSKeyPath: keyPath})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Certificates) != 1 {
			t.Fatalf("got %d certificates, want 1", len(cfg.Certificates))
		}
	})

	for name, tc := range map[string]struct {
		opts clientConfigOptions
		want string
	}{
		"cert with no key": {
			clientConfigOptions{TLSCertPath: certPath},
			"Client cert specified, but not client key!",
		},
		"key with no cert": {
			clientConfigOptions{TLSKeyPath: keyPath},
			"Client key specified, but not client cert!",
		},
		"combined with a cert path": {
			clientConfigOptions{TLSCombinedPath: combinedPath, TLSCertPath: certPath},
			"Combined TLS PEM specified together with a cert or key path!",
		},
		"combined with a key path": {
			clientConfigOptions{TLSCombinedPath: combinedPath, TLSKeyPath: keyPath},
			"Combined TLS PEM specified together with a cert or key path!",
		},
		"combined file missing its key": {
			clientConfigOptions{TLSCombinedPath: certPath},
			"failed to load certs",
		},
		"combined file missing": {
			clientConfigOptions{TLSCombinedPath: filepath.Join(t.TempDir(), "absent.pem")},
			"failed to read combined TLS PEM",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := buildTLSConfig(tc.opts)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}
