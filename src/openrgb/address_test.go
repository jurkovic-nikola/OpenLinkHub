package openrgb

import (
	"LumenForge/src/config"
	"testing"
)

func TestTargetListenAddressAlwaysUsesIPv4Loopback(t *testing.T) {
	tests := []struct {
		name          string
		listenAddress string
		openRGBPort   int
		want          string
	}{
		{name: "legacy wildcard ignored", listenAddress: "0.0.0.0", openRGBPort: 6742, want: "127.0.0.1:6742"},
		{name: "legacy LAN address ignored", listenAddress: "192.168.1.50", openRGBPort: 6743, want: "127.0.0.1:6743"},
		{name: "legacy Tailscale address ignored", listenAddress: "100.64.0.10", openRGBPort: 16666, want: "127.0.0.1:16666"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := config.Configuration{
				ListenAddress: test.listenAddress,
				OpenRGBPort:   test.openRGBPort,
			}
			if got := targetListenAddress(cfg); got != test.want {
				t.Fatalf("targetListenAddress(%+v) = %q, want %q", cfg, got, test.want)
			}
		})
	}
}
