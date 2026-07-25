package memory

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadSupportedDevicesAndLookup(t *testing.T) {
	devices, err := loadSupportedDevices(filepath.Join("..", "..", "..", "database", "external", "memory.json"))
	if err != nil {
		t.Fatalf("load memory metadata: %v", err)
	}
	if len(devices) != len(defaultSupportedDevices) {
		t.Fatalf("loaded %d memory families, want %d", len(devices), len(defaultSupportedDevices))
	}
	if !reflect.DeepEqual(devices, defaultSupportedDevices) {
		t.Fatalf("external memory metadata differs from built-in fallback:\n got: %#v\nwant: %#v", devices, defaultSupportedDevices)
	}

	device := &Device{supportedDevices: devices}
	ddr4 := device.getDeviceMetadata(4, "W")
	if ddr4 == nil || ddr4.Name != "VENGEANCE RGB PRO" || ddr4.LedChannels != 10 || ddr4.Register != 0x31 {
		t.Fatalf("DDR4 W metadata = %#v", ddr4)
	}
	ddr5 := device.getDeviceMetadata(5, "P")
	if ddr5 == nil || ddr5.Name != "DOMINATOR TITANIUM RGB" || ddr5.LedChannels != 11 || ddr5.Register != 0x31 {
		t.Fatalf("DDR5 P metadata = %#v", ddr5)
	}
	if unknown := device.getDeviceMetadata(5, "X"); unknown != nil {
		t.Fatalf("unknown metadata = %#v, want nil", unknown)
	}
}

func TestLoadSupportedDevicesRejectsInvalidOrEmptyMetadata(t *testing.T) {
	for name, contents := range map[string]string{
		"invalid": "{",
		"empty":   "[]",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "memory.json")
			if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
				t.Fatalf("write metadata fixture: %v", err)
			}
			if _, err := loadSupportedDevices(path); err == nil {
				t.Fatal("loadSupportedDevices() returned nil error")
			}
		})
	}
}
