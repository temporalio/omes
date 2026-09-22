package lambda

import (
	"strings"
	"testing"
)

// The AWS fetches are unreachable for these cases: selection is decided before
// any secret is read.
func TestLoadClientCertSelection(t *testing.T) {
	for name, tc := range map[string]struct {
		combinedID, certID, keyID string
		want                      string
	}{
		"combined and cert": {"combined", "cert", "", "not both"},
		"combined and key":  {"combined", "", "key", "not both"},
		"cert without key":  {"", "cert", "", "must be set together"},
		"key without cert":  {"", "", "key", "must be set together"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := loadClientCert(t.Context(), nil, tc.combinedID, tc.certID, tc.keyID)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not mention %q", err, tc.want)
			}
		})
	}

	t.Run("none set", func(t *testing.T) {
		cert, err := loadClientCert(t.Context(), nil, "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cert != nil {
			t.Fatal("expected no certificate")
		}
	})
}
