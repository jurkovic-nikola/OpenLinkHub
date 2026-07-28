package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var groupedConfigurationKeys = []string{
	"debug",
	"manual",
	"frontend",
	"metrics",
	"listenAddress",
	"listenPort",
	"logFile",
	"logLevel",
	"enableSystemTray",
	"enableGamepad",
	"enableMotherboard",
	"enableOpenRGBTargetServer",
	"motherboardBiosOnExit",
	"checkDevicePermission",
	"graphProfiles",
	"resumeDelay",
	"temperatureOffset",
	"cpuSensorChip",
	"cpuTempFile",
	"memory",
	"memoryType",
	"memorySmBus",
	"memorySku",
	"memoryRegisterOverride",
	"ramTempViaHwmon",
	"enhancementKits",
	"amdGpuIndex",
	"amdsmiPath",
	"nvidiaGpuIndex",
	"defaultNvidiaGPU",
	"openRGBPort",
	"exclude",
}

func jsonObjectKeysInOrder(t *testing.T, data []byte) []string {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		t.Fatalf("read opening JSON token: %v", err)
	}
	if token != json.Delim('{') {
		t.Fatalf("opening JSON token = %v, want {", token)
	}

	keys := make([]string, 0)
	for decoder.More() {
		token, err = decoder.Token()
		if err != nil {
			t.Fatalf("read JSON object key: %v", err)
		}
		key, ok := token.(string)
		if !ok {
			t.Fatalf("JSON object key token has type %T, want string", token)
		}
		keys = append(keys, key)

		var value json.RawMessage
		if err = decoder.Decode(&value); err != nil {
			t.Fatalf("skip value for JSON key %q: %v", key, err)
		}
	}

	token, err = decoder.Token()
	if err != nil {
		t.Fatalf("read closing JSON token: %v", err)
	}
	if token != json.Delim('}') {
		t.Fatalf("closing JSON token = %v, want }", token)
	}
	return keys
}

func TestFreshConfigUsesGroupedFieldOrder(t *testing.T) {
	originalLocation := location
	t.Cleanup(func() {
		location = originalLocation
	})

	location = filepath.Join(t.TempDir(), "config.json")
	upgradeFile(location)

	data, err := os.ReadFile(location)
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}

	expected := make([]string, 0, len(groupedConfigurationKeys)-2)
	for _, key := range groupedConfigurationKeys {
		if key != "listenAddress" && key != "enableOpenRGBTargetServer" {
			expected = append(expected, key)
		}
	}
	if actual := jsonObjectKeysInOrder(t, data); !reflect.DeepEqual(actual, expected) {
		t.Fatalf("generated config field order:\n got: %v\nwant: %v", actual, expected)
	}
}

func TestConfigurationOptionalFieldsUseGroupedOrder(t *testing.T) {
	data, err := json.Marshal(Configuration{
		ListenAddress:             "192.168.1.50",
		EnableOpenRGBTargetServer: true,
	})
	if err != nil {
		t.Fatalf("marshal configuration: %v", err)
	}

	if actual := jsonObjectKeysInOrder(t, data); !reflect.DeepEqual(actual, groupedConfigurationKeys) {
		t.Fatalf("configuration field order:\n got: %v\nwant: %v", actual, groupedConfigurationKeys)
	}
}

func TestDecodeConfigurationAcceptsLegacyListenAddress(t *testing.T) {
	cfg, hasLegacyListenAddress, err := decodeConfiguration(bytes.NewBufferString(`{
		"listenAddress": "192.168.1.50",
		"listenPort": 28080,
		"openRGBPort": 6743
	}`))
	if err != nil {
		t.Fatalf("decodeConfiguration() returned error: %v", err)
	}
	if !hasLegacyListenAddress {
		t.Fatal("decodeConfiguration() did not report the legacy listenAddress field")
	}
	if cfg.ListenAddress != "192.168.1.50" {
		t.Fatalf("decoded ListenAddress = %q, want %q", cfg.ListenAddress, "192.168.1.50")
	}
	if cfg.ListenPort != 28080 {
		t.Fatalf("decoded ListenPort = %d, want 28080", cfg.ListenPort)
	}
	if cfg.OpenRGBPort != 6743 {
		t.Fatalf("decoded OpenRGBPort = %d, want 6743", cfg.OpenRGBPort)
	}
}

func TestUpgradePreservesLegacyListenAddress(t *testing.T) {
	originalLocation := location
	t.Cleanup(func() {
		location = originalLocation
	})

	location = filepath.Join(t.TempDir(), "config.json")
	legacyConfig := []byte(`{
		"listenAddress": "192.168.1.50",
		"listenPort": 28080
	}`)
	if err := os.WriteFile(location, legacyConfig, 0600); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	upgradeFile(location)

	data, err := os.ReadFile(location)
	if err != nil {
		t.Fatalf("read upgraded config: %v", err)
	}
	var upgraded map[string]json.RawMessage
	if err = json.Unmarshal(data, &upgraded); err != nil {
		t.Fatalf("decode upgraded config: %v", err)
	}
	var listenAddress string
	if err = json.Unmarshal(upgraded["listenAddress"], &listenAddress); err != nil {
		t.Fatalf("decode preserved listenAddress: %v", err)
	}
	if listenAddress != "192.168.1.50" {
		t.Fatalf("preserved listenAddress = %q, want %q", listenAddress, "192.168.1.50")
	}
}

func TestIgnoredListenAddressReportsOnlyConfiguredNonLoopbackValues(t *testing.T) {
	originalConfiguration := configuration
	originalConfigured := legacyListenAddressConfigured
	t.Cleanup(func() {
		configuration = originalConfiguration
		legacyListenAddressConfigured = originalConfigured
	})

	tests := []struct {
		name       string
		configured bool
		address    string
		wantWarn   bool
	}{
		{name: "field absent", configured: false, address: "", wantWarn: false},
		{name: "IPv4 loopback", configured: true, address: "127.0.0.1", wantWarn: false},
		{name: "other loopback", configured: true, address: "127.0.0.2", wantWarn: false},
		{name: "localhost", configured: true, address: "localhost", wantWarn: false},
		{name: "wildcard", configured: true, address: "0.0.0.0", wantWarn: true},
		{name: "empty legacy value", configured: true, address: "", wantWarn: true},
		{name: "LAN address", configured: true, address: "192.168.1.50", wantWarn: true},
		{name: "Tailscale address", configured: true, address: "100.64.0.10", wantWarn: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration.ListenAddress = test.address
			legacyListenAddressConfigured = test.configured

			gotAddress, gotWarn := IgnoredListenAddress()
			if gotWarn != test.wantWarn {
				t.Fatalf("IgnoredListenAddress() warning = %t, want %t", gotWarn, test.wantWarn)
			}
			if gotWarn && gotAddress != test.address {
				t.Fatalf("IgnoredListenAddress() address = %q, want %q", gotAddress, test.address)
			}
		})
	}
}
