package server

import (
	"LumenForge/src/config"
	"testing"
)

func TestHTTPListenAddressAlwaysUsesIPv4Loopback(t *testing.T) {
	tests := []struct {
		name          string
		listenAddress string
		listenPort    int
		want          string
	}{
		{name: "legacy wildcard ignored", listenAddress: "0.0.0.0", listenPort: 27003, want: "127.0.0.1:27003"},
		{name: "legacy LAN address ignored", listenAddress: "192.168.1.50", listenPort: 28080, want: "127.0.0.1:28080"},
		{name: "legacy Tailscale address ignored", listenAddress: "100.64.0.10", listenPort: 8080, want: "127.0.0.1:8080"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := config.Configuration{
				ListenAddress: test.listenAddress,
				ListenPort:    test.listenPort,
			}
			if got := httpListenAddress(cfg); got != test.want {
				t.Fatalf("httpListenAddress(%+v) = %q, want %q", cfg, got, test.want)
			}
		})
	}
}
