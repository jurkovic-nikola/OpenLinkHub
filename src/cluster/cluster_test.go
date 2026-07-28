package cluster

import (
	"LumenForge/src/config"
	"LumenForge/src/rgb"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFreshClusterRgbUsesCanonicalDefaults(t *testing.T) {
	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	repositoryRoot := filepath.Clean(filepath.Join(originalWorkingDirectory, "..", ".."))
	temporaryConfig := t.TempDir()
	paths, err := config.ResolvePaths(config.PathOptions{
		Mode:             config.ServiceModeDevelopment,
		ApplicationRoot:  repositoryRoot,
		ConfigRoot:       temporaryConfig,
		DataRoot:         temporaryConfig,
		WorkingDirectory: repositoryRoot,
	})
	if err != nil {
		t.Fatalf("resolve temporary paths: %v", err)
	}
	t.Cleanup(config.UsePathsForTest(paths))

	rgb.Init()
	canonical := rgb.GetRGB()
	canonical.Device = "Cluster"

	for _, directory := range []string{"rgb", "profiles"} {
		if err = os.MkdirAll(filepath.Join(temporaryConfig, "database", directory), 0o755); err != nil {
			t.Fatalf("create isolated %s directory: %v", directory, err)
		}
	}

	originalConfigPath := pwd
	pwd = temporaryConfig
	t.Cleanup(func() {
		pwd = originalConfigPath
	})

	rgbModes := make([]string, 0, len(canonical.Profiles))
	for profile := range canonical.Profiles {
		rgbModes = append(rgbModes, profile)
	}

	device := &Device{
		Product:  "Cluster",
		Serial:   "cluster",
		RGBModes: rgbModes,
	}
	device.saveDeviceProfile()
	if _, err = os.Stat(filepath.Join(temporaryConfig, "database", "profiles", "cluster.json")); err != nil {
		t.Fatalf("cluster profile was not written beneath mutable database root: %v", err)
	}
	device.loadRgb()

	generatedPath := filepath.Join(temporaryConfig, "database", "rgb", "cluster.json")
	generatedData, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("read generated cluster RGB defaults: %v", err)
	}

	var generated rgb.RGB
	if err = json.Unmarshal(generatedData, &generated); err != nil {
		t.Fatalf("decode generated cluster RGB defaults: %v", err)
	}
	if !reflect.DeepEqual(generated, canonical) {
		t.Fatal("fresh cluster RGB defaults drifted from database/rgb.json")
	}

	var topLevel map[string]json.RawMessage
	if err = json.Unmarshal(generatedData, &topLevel); err != nil {
		t.Fatalf("decode generated top-level fields: %v", err)
	}
	for _, key := range []string{
		"DeviceOrder",
		"RGBProfile",
		"LastNonOffProfile",
		"RgbOff",
		"deviceOrder",
		"rgbProfile",
		"lastNonOffProfile",
		"rgbOff",
	} {
		if _, exists := topLevel[key]; exists {
			t.Errorf("generated RGB defaults contain machine-specific field %q", key)
		}
	}
	for key := range topLevel {
		if key != "device" && key != "defaultColor" && key != "profiles" {
			t.Errorf("generated RGB defaults contain unexpected top-level field %q", key)
		}
	}

	expectedProfiles := map[string]struct {
		speed      float64
		smoothness int
	}{
		"comet":           {speed: 5, smoothness: 20},
		"datastream":      {speed: 9.8, smoothness: 20},
		"flame":           {speed: 0.8, smoothness: 20},
		"cyberpunkglitch": {speed: 0.55, smoothness: 20},
		"nebula":          {speed: 4, smoothness: 0},
		"tokyonight":      {speed: 2.8, smoothness: 20},
	}
	for name, expected := range expectedProfiles {
		profile, exists := generated.Profiles[name]
		if !exists {
			t.Errorf("generated RGB defaults are missing %q", name)
			continue
		}
		if profile.Speed != expected.speed {
			t.Errorf("%s speed = %v, want %v", name, profile.Speed, expected.speed)
		}
		if profile.Smoothness != expected.smoothness {
			t.Errorf("%s smoothness = %d, want %d", name, profile.Smoothness, expected.smoothness)
		}
	}
}
