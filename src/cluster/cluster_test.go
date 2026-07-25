package cluster

import (
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
	if err = os.Chdir(repositoryRoot); err != nil {
		t.Fatalf("change to repository root: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWorkingDirectory); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	rgb.Init()
	canonical := rgb.GetRGB()
	canonical.Device = "Cluster"

	temporaryConfig := t.TempDir()
	if err = os.MkdirAll(filepath.Join(temporaryConfig, "database", "rgb"), 0o755); err != nil {
		t.Fatalf("create isolated RGB directory: %v", err)
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
