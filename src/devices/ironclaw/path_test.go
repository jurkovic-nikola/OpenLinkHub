package ironclaw

import (
	"LumenForge/src/config"
	"LumenForge/src/inputmanager"
	"os"
	"path/filepath"
	"testing"
)

func TestKeyAssignmentsWriteToMutableDatabaseRoot(t *testing.T) {
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

	originalPWD := pwd
	pwd = paths.MutableDataRoot
	t.Cleanup(func() {
		pwd = originalPWD
	})
	device := &Device{
		DeviceProfile:     &DeviceProfile{},
		KeyAssignment:     map[int]inputmanager.KeyAssignment{},
		keyAssignmentFile: "/database/key-assignments/ironclaw.json",
	}
	device.saveKeyAssignments()

	expected := filepath.Join(paths.MutableKeyAssignmentsRoot, "ironclaw.json")
	if _, err = os.Stat(expected); err != nil {
		t.Fatalf("key assignments were not written at %q: %v", expected, err)
	}
}
