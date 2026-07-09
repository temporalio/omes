package clioptions

import (
	"strings"
	"testing"

	"go.temporal.io/sdk/client"
)

func TestClientOptionsServerAddresses(t *testing.T) {
	tests := []struct {
		name string
		opts ClientOptions
		want []string
	}{
		{
			name: "default",
			want: []string{client.DefaultHostPort},
		},
		{
			name: "legacy address",
			opts: ClientOptions{Address: "127.0.0.1:7234"},
			want: []string{"127.0.0.1:7234"},
		},
		{
			name: "addresses",
			opts: ClientOptions{Addresses: []string{"127.0.0.1:7234", "127.0.0.1:8234"}},
			want: []string{"127.0.0.1:7234", "127.0.0.1:8234"},
		},
		{
			name: "trims empty",
			opts: ClientOptions{Addresses: []string{" 127.0.0.1:7234 ", ""}},
			want: []string{"127.0.0.1:7234"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.opts.ServerAddresses()
			if len(got) != len(test.want) {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
			for i := range got {
				if got[i] != test.want[i] {
					t.Fatalf("expected %v, got %v", test.want, got)
				}
			}
		})
	}
}

func TestClientOptionsRepeatedServerAddressFlag(t *testing.T) {
	var opts ClientOptions
	fs := opts.FlagSet()
	if err := fs.Set("server-address", "127.0.0.1:7234"); err != nil {
		t.Fatal(err)
	}
	if err := fs.Set("server-address", "127.0.0.1:8234"); err != nil {
		t.Fatal(err)
	}

	got := opts.ServerAddresses()
	want := []string{"127.0.0.1:7234", "127.0.0.1:8234"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestClientOptionsHostPort(t *testing.T) {
	single, err := (&ClientOptions{Addresses: []string{"127.0.0.1:7234"}}).HostPort()
	if err != nil {
		t.Fatal(err)
	}
	if single != "127.0.0.1:7234" {
		t.Fatalf("expected single address, got %q", single)
	}

	multi, err := (&ClientOptions{Addresses: []string{"127.0.0.1:7234", "127.0.0.1:8234"}}).HostPort()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(multi, "omes-manual-") || !strings.HasSuffix(multi, ":///temporal") {
		t.Fatalf("expected manual resolver target, got %q", multi)
	}

	_, err = (&ClientOptions{Addresses: []string{"", " "}}).HostPort()
	if err == nil {
		t.Fatal("expected empty address list to fail")
	}
}
