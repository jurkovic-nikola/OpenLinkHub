package led

import (
	"LumenForge/src/config"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveProfileWritesToMutableDatabaseRoot(t *testing.T) {
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

	SaveProfile("device", Device{Serial: "device"})
	expected := filepath.Join(paths.MutableLEDRoot, "device.json")
	if _, err = os.Stat(expected); err != nil {
		t.Fatalf("LED profile was not written at %q: %v", expected, err)
	}
}
