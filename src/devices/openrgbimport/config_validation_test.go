package openrgbimport

import (
	"LumenForge/src/cluster"
	"LumenForge/src/config"
	"LumenForge/src/openrgb"
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func zonesWithCounts(counts ...int) []ZoneConfig {
	zones := make([]ZoneConfig, len(counts))
	for index, count := range counts {
		zones[index] = ZoneConfig{
			Name:     "Zone",
			LedCount: count,
		}
	}
	return zones
}

func TestValidateDeviceConfigReturnsNormalizedCopyWithoutMutatingInput(t *testing.T) {
	serial := "openrgb-validation-copy"
	input := DeviceConfig{
		Serial:         serial,
		Product:        "Validation Device",
		ExternalSerial: "external-validation",
		Location:       "location-validation",
		Vendor:         "Validation Vendor",
		Zones: []ZoneConfig{
			{Name: " \x00Main Zone\n ", LedCount: 2},
			{Name: "\t\x00", LedCount: 3},
		},
	}
	original := cloneDeviceConfig(&input)

	validated, err := validateDeviceConfig(serial, input, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(&input, original) {
		t.Fatalf("validator mutated input: got %#v, want %#v", input, *original)
	}
	if &validated.Zones[0] == &input.Zones[0] {
		t.Fatal("validator reused the caller's Zones backing array")
	}
	if validated.Zones[0].Name != "Main Zone" {
		t.Fatalf("first normalized zone name = %q, want %q", validated.Zones[0].Name, "Main Zone")
	}
	if validated.Zones[1].Name != "Zone 2" {
		t.Fatalf("empty normalized zone name = %q, want %q", validated.Zones[1].Name, "Zone 2")
	}

	validated.Zones[0].Name = "Changed"
	if input.Zones[0].Name != original.Zones[0].Name {
		t.Fatal("changing the validated copy changed the caller's Zones slice")
	}
}

func TestValidateDeviceConfigSemanticLimits(t *testing.T) {
	serial := "openrgb-validation-limits"
	tests := []struct {
		name        string
		target      string
		input       DeviceConfig
		allowLegacy bool
		wantSerial  string
		wantError   string
	}{
		{
			name:        "legacy empty serial",
			target:      serial,
			input:       DeviceConfig{Zones: zonesWithCounts(1)},
			allowLegacy: true,
			wantSerial:  serial,
		},
		{
			name:      "strict empty serial",
			target:    serial,
			input:     DeviceConfig{Zones: zonesWithCounts(1)},
			wantError: "empty internal serial",
		},
		{
			name:      "conflicting serial",
			target:    serial,
			input:     DeviceConfig{Serial: "openrgb-validation-other", Zones: zonesWithCounts(1)},
			wantError: "conflicting internal serial",
		},
		{
			name:      "unsafe target serial",
			target:    "../unsafe",
			input:     DeviceConfig{Serial: "../unsafe", Zones: zonesWithCounts(1)},
			wantError: "unusable internal serial",
		},
		{
			name:      "zero zones",
			target:    serial,
			input:     DeviceConfig{Serial: serial},
			wantError: "0 zones; expected 1 through 128",
		},
		{
			name:   "more than 128 zones",
			target: serial,
			input: DeviceConfig{
				Serial: serial,
				Zones:  zonesWithCounts(make([]int, 129)...),
			},
			wantError: "129 zones; expected 1 through 128",
		},
		{
			name:      "zero LEDs",
			target:    serial,
			input:     DeviceConfig{Serial: serial, Zones: zonesWithCounts(0)},
			wantError: "zone 1 has 0 LEDs; expected 1 through 1024",
		},
		{
			name:      "negative LEDs",
			target:    serial,
			input:     DeviceConfig{Serial: serial, Zones: zonesWithCounts(-1)},
			wantError: "zone 1 has -1 LEDs; expected 1 through 1024",
		},
		{
			name:      "more than 1024 LEDs in one zone",
			target:    serial,
			input:     DeviceConfig{Serial: serial, Zones: zonesWithCounts(1025)},
			wantError: "zone 1 has 1025 LEDs; expected 1 through 1024",
		},
		{
			name:       "exactly 4096 total LEDs",
			target:     serial,
			input:      DeviceConfig{Serial: serial, Zones: zonesWithCounts(1024, 1024, 1024, 1024)},
			wantSerial: serial,
		},
		{
			name:      "4097 total LEDs",
			target:    serial,
			input:     DeviceConfig{Serial: serial, Zones: zonesWithCounts(1024, 1024, 1024, 1024, 1)},
			wantError: "zone 5 has 1 LEDs and would exceed the permitted total range of 1 through 4096",
		},
		{
			name:      "very large LED count",
			target:    serial,
			input:     DeviceConfig{Serial: serial, Zones: zonesWithCounts(int(^uint(0) >> 1))},
			wantError: "zone 1 has",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			original := cloneDeviceConfig(&test.input)
			validated, err := validateDeviceConfig(test.target, test.input, test.allowLegacy)
			if test.wantError != "" {
				if err == nil {
					t.Fatal("validation succeeded, want error")
				}
				if !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("validation error = %q, want substring %q", err, test.wantError)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if validated.Serial != test.wantSerial {
					t.Fatalf("validated serial = %q, want %q", validated.Serial, test.wantSerial)
				}
			}
			if !reflect.DeepEqual(&test.input, original) {
				t.Fatalf("validator mutated input: got %#v, want %#v", test.input, *original)
			}
		})
	}
}

func TestIsConfigValidForControllerDoesNotMutateConfig(t *testing.T) {
	serial := "openrgb-validation-compatibility"
	cfg := DeviceConfig{
		Serial: serial,
		Zones: []ZoneConfig{
			{Name: " \x00Compatibility Zone\n", LedCount: 1},
		},
	}
	original := cloneDeviceConfig(&cfg)

	if !isConfigValidForController(&cfg, openrgb.DiscoveredController{}) {
		t.Fatal("valid compatibility config was rejected")
	}
	if !reflect.DeepEqual(&cfg, original) {
		t.Fatalf("compatibility validation mutated config: got %#v, want %#v", cfg, *original)
	}
}

func TestSaveDeviceConfigRejectsInvalidLayoutWithoutSideEffects(t *testing.T) {
	storePath := useStorePath(t)
	StopManager()
	setConfiguredDevices(nil)

	serial := "openrgb-validation-no-side-effects"
	saved := DeviceConfig{
		Serial:         serial,
		Product:        "Saved Product",
		ExternalSerial: "saved-external",
		Location:       "saved-location",
		Vendor:         "Saved Vendor",
		Zones:          []ZoneConfig{{Name: "Saved Zone", LedCount: 4}},
	}
	if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{serial: saved}}); err != nil {
		t.Fatal(err)
	}
	storeBefore, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatal(err)
	}

	profilePath := filepath.Join(t.TempDir(), "profile.json")
	profileBefore := []byte("profile-must-remain-unchanged")
	if err = os.WriteFile(profilePath, profileBefore, 0o644); err != nil {
		t.Fatal(err)
	}
	generatedProfilePath := filepath.Join(config.GetConfig().ConfigPath, "database", "profiles", serial+".json")
	if _, statErr := os.Stat(generatedProfilePath); !os.IsNotExist(statErr) {
		t.Fatalf("generated profile path unexpectedly exists before test: %v", statErr)
	}

	brightness := uint8(47)
	device := testDevice(saved)
	device.controllerId = 7
	device.effect = "rainbow"
	device.running = true
	device.stopChan = make(chan struct{})
	device.doneChan = make(chan struct{})
	close(device.doneChan)
	device.DeviceProfile = &DeviceProfile{
		Active:           true,
		Path:             profilePath,
		Product:          saved.Product,
		Serial:           serial,
		RGBProfile:       "rainbow",
		BrightnessSlider: &brightness,
		ZoneColors:       buildZoneColorsFromConfig(&saved, device.lastColor),
		RGBCluster:       true,
	}

	configBefore := cloneDeviceConfig(device.Config)
	configPointerBefore := device.Config
	profileStateBefore := cloneDeviceProfile(device.DeviceProfile)
	profilePointerBefore := device.DeviceProfile
	colorCountBefore := device.colorCount
	zoneAmountBefore := device.ZoneAmount
	effectBefore := device.effect
	runningBefore := device.running
	stopChanBefore := device.stopChan
	doneChanBefore := device.doneChan

	input := DeviceConfig{
		Serial: serial,
		Zones:  zonesWithCounts(1024, 1024, 1024, 1024, 1),
	}
	inputBefore := cloneDeviceConfig(&input)

	frameCalls := 0
	healthCalls := 0
	clusterCalls := 0
	storeRenameCalls := 0
	previousSendFrame := sendConfigFrame
	previousCheckHealth := checkConfigHealth
	previousGetCluster := getConfigCluster
	previousRename := renameConfigStore
	sendConfigFrame = func(uint32, []byte) error {
		frameCalls++
		return nil
	}
	checkConfigHealth = func() error {
		healthCalls++
		return nil
	}
	getConfigCluster = func() *cluster.Device {
		clusterCalls++
		return nil
	}
	renameConfigStore = func(oldPath, newPath string) error {
		storeRenameCalls++
		return previousRename(oldPath, newPath)
	}
	t.Cleanup(func() {
		sendConfigFrame = previousSendFrame
		checkConfigHealth = previousCheckHealth
		getConfigCluster = previousGetCluster
		renameConfigStore = previousRename
	})

	err = device.SaveDeviceConfig(&input)
	if err == nil {
		t.Fatal("SaveDeviceConfig succeeded for a 4097-LED layout")
	}
	if !strings.Contains(err.Error(), "permitted total range of 1 through 4096") {
		t.Fatalf("SaveDeviceConfig error = %q, want total-limit detail", err)
	}

	if !reflect.DeepEqual(device.Config, configBefore) {
		t.Fatalf("device config changed: got %#v, want %#v", device.Config, configBefore)
	}
	if device.Config != configPointerBefore {
		t.Fatal("device config pointer changed during validation rejection")
	}
	if device.colorCount != colorCountBefore || device.ZoneAmount != zoneAmountBefore {
		t.Fatalf("device dimensions changed: colorCount=%d zoneAmount=%d", device.colorCount, device.ZoneAmount)
	}
	if device.DeviceProfile != profilePointerBefore || !reflect.DeepEqual(device.DeviceProfile, profileStateBefore) {
		t.Fatalf("device profile changed: got %#v, want %#v", device.DeviceProfile, profileStateBefore)
	}
	if device.effect != effectBefore || device.running != runningBefore ||
		device.stopChan != stopChanBefore || device.doneChan != doneChanBefore {
		t.Fatal("effect state changed during validation rejection")
	}
	if !reflect.DeepEqual(&input, inputBefore) {
		t.Fatalf("SaveDeviceConfig mutated caller input: got %#v, want %#v", input, *inputBefore)
	}

	storeAfter, readErr := os.ReadFile(storePath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(storeAfter, storeBefore) {
		t.Fatal("invalid SaveDeviceConfig changed the importer store")
	}
	profileAfter, readErr := os.ReadFile(profilePath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(profileAfter, profileBefore) {
		t.Fatal("invalid SaveDeviceConfig changed the profile file")
	}
	if _, statErr := os.Stat(generatedProfilePath); !os.IsNotExist(statErr) {
		t.Fatalf("invalid SaveDeviceConfig created a profile: %v", statErr)
	}
	if frameCalls != 0 || healthCalls != 0 || clusterCalls != 0 || storeRenameCalls != 0 {
		t.Fatalf(
			"invalid SaveDeviceConfig side effects: frames=%d health=%d cluster=%d storeRenames=%d",
			frameCalls,
			healthCalls,
			clusterCalls,
			storeRenameCalls,
		)
	}
}

func TestSaveDeviceConfigPreservesCallerAndValidBehavior(t *testing.T) {
	storePath := useStorePath(t)
	StopManager()
	setConfiguredDevices(nil)

	serial := "openrgb-validation-valid-save"
	saved := testConfig(serial, "Compatible Product")
	if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{serial: saved}}); err != nil {
		t.Fatal(err)
	}

	device := testDevice(saved)
	device.DeviceProfile = &DeviceProfile{
		Active:           true,
		RGBProfile:       "static",
		BrightnessSlider: func() *uint8 { value := uint8(63); return &value }(),
		ZoneColors:       buildZoneColorsFromConfig(&saved, device.lastColor),
	}
	input := DeviceConfig{
		Serial:  serial,
		Product: "Caller Product Must Not Replace Stored Metadata",
		Zones: []ZoneConfig{
			{Name: " \x00Updated Zone\n ", LedCount: 2},
		},
	}
	inputBefore := cloneDeviceConfig(&input)

	if err := device.SaveDeviceConfig(&input); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(&input, inputBefore) {
		t.Fatalf("SaveDeviceConfig mutated caller input: got %#v, want %#v", input, *inputBefore)
	}
	if device.Config == nil || device.Config.Serial != serial {
		t.Fatalf("saved device serial = %#v, want %q", device.Config, serial)
	}
	if device.Config.Zones[0].Name != "Updated Zone" || device.Config.Zones[0].LedCount != 2 {
		t.Fatalf("saved device zones = %#v", device.Config.Zones)
	}
	if device.colorCount != 2 || device.ZoneAmount != 1 {
		t.Fatalf("saved dimensions = colorCount %d, zones %d", device.colorCount, device.ZoneAmount)
	}

	store, err := loadConfigStore()
	if err != nil {
		t.Fatal(err)
	}
	stored := store.Devices[serial]
	if stored.Product != saved.Product {
		t.Fatalf("stored product = %q, want preserved %q", stored.Product, saved.Product)
	}
	if stored.Zones[0].Name != "Updated Zone" || stored.Zones[0].LedCount != 2 {
		t.Fatalf("stored zones = %#v", stored.Zones)
	}
	if _, err = os.Stat(storePath); err != nil {
		t.Fatal(err)
	}
}

func TestSaveDeviceConfigDoesNotOverwriteMismatchedCallerSerial(t *testing.T) {
	device := testDevice(testConfig("openrgb-validation-target", "Target"))
	input := DeviceConfig{
		Serial: "openrgb-validation-other",
		Zones:  zonesWithCounts(1),
	}

	err := device.SaveDeviceConfig(&input)
	if err == nil || !strings.Contains(err.Error(), "conflicting internal serial") {
		t.Fatalf("SaveDeviceConfig error = %v, want conflicting serial", err)
	}
	if input.Serial != "openrgb-validation-other" {
		t.Fatalf("caller serial = %q, want unchanged", input.Serial)
	}
}

func TestSaveDeviceConfigNilInput(t *testing.T) {
	device := testDevice(testConfig("openrgb-validation-nil", "Nil"))
	if err := device.SaveDeviceConfig(nil); err == nil || err.Error() != "config is required" {
		t.Fatalf("SaveDeviceConfig(nil) error = %v, want config is required", err)
	}
}
