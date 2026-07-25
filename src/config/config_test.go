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
	"ConfigPath",
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
		if key != "ConfigPath" && key != "enableOpenRGBTargetServer" {
			expected = append(expected, key)
		}
	}
	if actual := jsonObjectKeysInOrder(t, data); !reflect.DeepEqual(actual, expected) {
		t.Fatalf("generated config field order:\n got: %v\nwant: %v", actual, expected)
	}
}

func TestConfigurationOptionalFieldsUseGroupedOrder(t *testing.T) {
	data, err := json.Marshal(Configuration{
		ConfigPath:                "/tmp/lumenforge-config",
		EnableOpenRGBTargetServer: true,
	})
	if err != nil {
		t.Fatalf("marshal configuration: %v", err)
	}

	if actual := jsonObjectKeysInOrder(t, data); !reflect.DeepEqual(actual, groupedConfigurationKeys) {
		t.Fatalf("configuration field order:\n got: %v\nwant: %v", actual, groupedConfigurationKeys)
	}
}

func TestSetSystemServiceExplicitMode(t *testing.T) {
	originalSystemService := systemService
	t.Cleanup(func() {
		systemService = originalSystemService
	})

	tests := []struct {
		name       string
		mode       string
		wantSystem bool
	}{
		{name: "system", mode: "system", wantSystem: true},
		{name: "user", mode: "user", wantSystem: false},
		{name: "desktop alias", mode: "desktop", wantSystem: false},
		{name: "normalized system", mode: " SYSTEM ", wantSystem: true},
		{name: "normalized user", mode: " USER ", wantSystem: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LUMENFORGE_SERVICE_MODE", test.mode)
			systemService = !test.wantSystem

			setSystemService()

			if systemService != test.wantSystem {
				t.Fatalf("setSystemService() with LUMENFORGE_SERVICE_MODE=%q set systemService to %t; want %t", test.mode, systemService, test.wantSystem)
			}
		})
	}
}

func TestSetSystemServiceUnrecognizedModeUsesFallback(t *testing.T) {
	originalSystemService := systemService
	t.Cleanup(func() {
		systemService = originalSystemService
	})

	t.Setenv("LUMENFORGE_SERVICE_MODE", "")
	setSystemService()
	wantSystem := systemService

	t.Setenv("LUMENFORGE_SERVICE_MODE", "unsupported")
	systemService = !wantSystem
	setSystemService()

	if systemService != wantSystem {
		t.Fatalf("unrecognized LUMENFORGE_SERVICE_MODE did not use fallback: got %t; want %t", systemService, wantSystem)
	}
}
