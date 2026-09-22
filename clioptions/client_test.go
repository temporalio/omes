package clioptions

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
func combinedPEM(t *testing.T) (combined, certOnly, keyOnly []byte) {
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
	certOnly = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyOnly = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return bytes.Join([][]byte{keyOnly, certOnly}, []byte("\n")), certOnly, keyOnly
}

func TestX509KeyPairFromCombinedPEM(t *testing.T) {
	combined, certOnly, keyOnly := combinedPEM(t)

	// Block order is not load-bearing.
	for name, blob := range map[string][]byte{
		"key first":  combined,
		"cert first": bytes.Join([][]byte{certOnly, keyOnly}, []byte("\n")),
	} {
		t.Run(name, func(t *testing.T) {
			cert, err := X509KeyPairFromCombinedPEM(blob)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cert.Certificate) != 1 {
				t.Fatalf("got %d certificates, want 1", len(cert.Certificate))
			}
		})
	}

	t.Run("cert without key", func(t *testing.T) {
		if _, err := X509KeyPairFromCombinedPEM(certOnly); err == nil {
			t.Fatal("expected an error for a blob with no private key")
		}
	})
}

func TestLoadTLSConfigCombinedPEM(t *testing.T) {
	combined, certOnly, keyOnly := combinedPEM(t)
	dir := t.TempDir()

	write := func(name string, data []byte) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	combinedPath := write("tls-combined.pem", combined)
	certPath := write("tls.crt", certOnly)
	keyPath := write("tls.key", keyOnly)

	t.Run("combined path carries both halves", func(t *testing.T) {
		c := &ClientOptions{ClientCombinedPath: combinedPath}
		cfg, err := c.loadTLSConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Certificates) != 1 {
			t.Fatalf("got %d certificates, want 1", len(cfg.Certificates))
		}
	})

	t.Run("cert and key paths load separately", func(t *testing.T) {
		c := &ClientOptions{ClientCertPath: certPath, ClientKeyPath: keyPath}
		cfg, err := c.loadTLSConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Certificates) != 1 {
			t.Fatalf("got %d certificates, want 1", len(cfg.Certificates))
		}
	})

	for name, tc := range map[string]struct {
		opts *ClientOptions
		want string
	}{
		"cert with no key": {
			&ClientOptions{ClientCertPath: certPath},
			"got TLS cert with no key",
		},
		"key with no cert": {
			&ClientOptions{ClientKeyPath: keyPath},
			"got TLS key with no cert",
		},
		"combined with a cert path": {
			&ClientOptions{ClientCombinedPath: combinedPath, ClientCertPath: certPath},
			"use one or the other",
		},
		"combined with a key path": {
			&ClientOptions{ClientCombinedPath: combinedPath, ClientKeyPath: keyPath},
			"use one or the other",
		},
		"combined file missing its key": {
			&ClientOptions{ClientCombinedPath: certPath},
			"failed to load certs",
		},
		"combined file missing": {
			&ClientOptions{ClientCombinedPath: filepath.Join(dir, "absent.pem")},
			"failed to read combined TLS PEM",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := tc.opts.loadTLSConfig()
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}
