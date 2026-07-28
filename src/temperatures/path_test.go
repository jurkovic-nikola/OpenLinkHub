package temperatures

import (
	"LumenForge/src/config"
	"os"
	"path/filepath"
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
