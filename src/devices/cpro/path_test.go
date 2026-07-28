package cpro

import (
	"LumenForge/src/config"
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
