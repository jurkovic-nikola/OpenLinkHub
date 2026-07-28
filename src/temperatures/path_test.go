package temperatures

import (
	"LumenForge/src/config"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemperatureProfileWritesToMutableDatabaseRoot(t *testing.T) {
	root := t.TempDir()
	paths, err := config.ResolvePaths(config.PathOptions{
		Mode:            config.ServiceModeUser,
		ApplicationRoot: filepath.Join(root, "app"),
		ConfigRoot:      filepath.Join(root, "config"),
		DataRoot:        filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = config.EnsureRuntimeDirectories(paths); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(config.UsePathsForTest(paths))

	originalPWD := pwd
	originalLocation := location
	originalProfiles := profiles
	originalTemperatures := temperatures
	t.Cleanup(func() {
		pwd = originalPWD
		location = originalLocation
		profiles = originalProfiles
		temperatures = originalTemperatures
	})
	profiles = map[string]TemperatureProfileData{}

	Init()
	if status := UpdateTemperatureProfileGraph("Custom", TemperatureProfileData{}); status != 1 {
		t.Fatalf("UpdateTemperatureProfileGraph() status = %d, want 1", status)
	}
	expected := filepath.Join(paths.MutableTemperaturesRoot, "Custom.json")
	if _, err = os.Stat(expected); err != nil {
		t.Fatalf("temperature profile was not written at %q: %v", expected, err)
	}
}

func TestExternalSourceProfileStoresOnlyRegistryID(t *testing.T) {
	root := t.TempDir()
	paths, err := config.ResolvePaths(config.PathOptions{
		Mode:             config.ServiceModeDevelopment,
		WorkingDirectory: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = config.EnsureRuntimeDirectories(paths); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(config.UsePathsForTest(paths))

	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	registry := map[string]any{
		"sources": []map[string]any{{
			"id":         "test-source",
			"name":       "Test Source",
			"executable": executable,
			"args":       []string{"fixed"},
		}},
	}
	registryData, err := json.Marshal(registry)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(paths.ExternalSourcesFile, registryData, 0o600); err != nil {
		t.Fatal(err)
	}

	originalPWD := pwd
	originalLocation := location
	originalProfiles := profiles
	originalTemperatures := temperatures
	t.Cleanup(func() {
		pwd = originalPWD
		location = originalLocation
		profiles = originalProfiles
		temperatures = originalTemperatures
	})
	profiles = map[string]TemperatureProfileData{}
	Init()

	if ok := AddTemperatureProfile(&NewTemperatureProfile{
		Profile:          "RegistryProfile",
		Sensor:           SensorTypeExternalExecutable,
		ExternalSourceID: "test-source",
		DeviceId:         executable,
	}); !ok {
		t.Fatal("AddTemperatureProfile() rejected a registered source")
	}

	profilePath := filepath.Join(paths.MutableTemperaturesRoot, "RegistryProfile.json")
	saved, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(saved, &fields); err != nil {
		t.Fatal(err)
	}
	var externalSourceID string
	if err = json.Unmarshal(fields["externalSourceId"], &externalSourceID); err != nil {
		t.Fatal(err)
	}
	if externalSourceID != "test-source" {
		t.Fatalf("externalSourceId = %q", externalSourceID)
	}
	for _, forbidden := range []string{"externalExecutable", "args", "executable"} {
		if _, exists := fields[forbidden]; exists {
			t.Fatalf("saved profile contains forbidden field %q: %s", forbidden, saved)
		}
	}
	if strings.Contains(string(saved), executable) || strings.Contains(string(saved), `"fixed"`) {
		t.Fatalf("saved profile contains registry execution details: %s", saved)
	}
}

func TestLegacyTypeSevenDevicePathIsCleared(t *testing.T) {
	profile := sanitizeExternalSourceProfile(TemperatureProfileData{
		Sensor: SensorTypeExternalExecutable,
		Device: "/tmp/untrusted-profile-program",
	})
	if profile.Device != "" {
		t.Fatalf("legacy executable path survived sanitization: %#v", profile)
	}
	if profile.ExternalSourceID != "" {
		t.Fatalf("legacy profile unexpectedly acquired an external source id: %#v", profile)
	}
}

func TestNonExternalProfileDropsExternalSourceID(t *testing.T) {
	profile := sanitizeExternalSourceProfile(TemperatureProfileData{
		Sensor:           SensorTypeCPU,
		Device:           "ordinary-device-value",
		ExternalSourceID: "must-not-persist",
	})
	if profile.Device != "ordinary-device-value" {
		t.Fatalf("non-type-7 device changed: %#v", profile)
	}
	if profile.ExternalSourceID != "" {
		t.Fatalf("non-type-7 external source id survived: %#v", profile)
	}
}
