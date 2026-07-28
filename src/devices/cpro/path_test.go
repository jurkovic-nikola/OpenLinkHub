package cpro

import (
	"LumenForge/src/config"
	"LumenForge/src/logger"
	"os"
	"path/filepath"
	"testing"
)

func TestExternalDefinitionsLoadFromApplicationRoot(t *testing.T) {
	root := t.TempDir()
	applicationRoot := filepath.Join(root, "app")
	definitionRoot := filepath.Join(applicationRoot, "database", "external")
	if err := os.MkdirAll(definitionRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(definitionRoot, "cpro.json"), []byte(`[{"Index":42,"Name":"Test device","Total":1}]`), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(applicationRoot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(applicationRoot, 0o755)
	})

	paths, err := config.ResolvePaths(config.PathOptions{
		Mode:            config.ServiceModeUser,
		ApplicationRoot: applicationRoot,
		ConfigRoot:      filepath.Join(root, "config"),
		DataRoot:        filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(config.UsePathsForTest(paths))

	device := &Device{}
	device.loadExternalDevices()
	if len(device.ExternalLedDevice) != 1 || device.ExternalLedDevice[0].Index != 42 {
		t.Fatalf("external definitions = %#v", device.ExternalLedDevice)
	}
}

func TestMalformedExternalDefinitionsCloseFile(t *testing.T) {
	root := t.TempDir()
	applicationRoot := filepath.Join(root, "app")
	definitionRoot := filepath.Join(applicationRoot, "database", "external")
	if err := os.MkdirAll(definitionRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	definitionPath := filepath.Join(definitionRoot, "cpro.json")
	if err := os.WriteFile(definitionPath, []byte(`{"malformed"`), 0o444); err != nil {
		t.Fatal(err)
	}

	paths, err := config.ResolvePaths(config.PathOptions{
		Mode:            config.ServiceModeUser,
		ApplicationRoot: applicationRoot,
		ConfigRoot:      filepath.Join(root, "config"),
		DataRoot:        filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(config.UsePathsForTest(paths))
	logger.Init()

	if before := openFileDescriptorsForPath(t, definitionPath); before != 0 {
		t.Fatalf("metadata file unexpectedly open before load: %d descriptors", before)
	}
	(&Device{}).loadExternalDevices()
	if after := openFileDescriptorsForPath(t, definitionPath); after != 0 {
		t.Fatalf("malformed metadata file remains open after load: %d descriptors", after)
	}
}

func openFileDescriptorsForPath(t *testing.T, path string) int {
	t.Helper()

	entries, err := os.ReadDir("/proc/self/fd")
	if os.IsNotExist(err) {
		t.Skip("/proc/self/fd is unavailable")
	}
	if err != nil {
		t.Fatalf("read process file descriptors: %v", err)
	}

	count := 0
	for _, entry := range entries {
		target, readErr := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
		if readErr == nil && target == path {
			count++
		}
	}
	return count
}
