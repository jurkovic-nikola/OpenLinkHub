package rgb

import (
	"LumenForge/src/config"
	"os"
	"path/filepath"
	"testing"
)

func TestInitReadsShippedRGBDefinitionsFromApplicationRoot(t *testing.T) {
	root := t.TempDir()
	applicationRoot := filepath.Join(root, "app")
	dataRoot := filepath.Join(root, "data")
	if err := os.MkdirAll(filepath.Join(applicationRoot, "database"), 0o755); err != nil {
		t.Fatal(err)
	}
	definition := `{"device":"shipped","defaultColor":{},"profiles":{"static":{"profileName":"static"}}}`
	if err := os.WriteFile(filepath.Join(applicationRoot, "database", "rgb.json"), []byte(definition), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(applicationRoot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(applicationRoot, 0o755) })

	paths, err := config.ResolvePaths(config.PathOptions{
		Mode:            config.ServiceModeUser,
		ApplicationRoot: applicationRoot,
		ConfigRoot:      filepath.Join(root, "config"),
		DataRoot:        dataRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = config.EnsureRuntimeDirectories(paths); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(config.UsePathsForTest(paths))

	original := rgb
	t.Cleanup(func() { rgb = original })
	Init()
	if GetRGB().Device != "shipped" {
		t.Fatalf("RGB definitions were not loaded from %q", paths.ShippedDatabaseRoot)
	}
	if err = os.WriteFile(filepath.Join(dataRoot, "write-check"), []byte("ok"), 0o600); err != nil {
		t.Fatalf("mutable data write failed with read-only application root: %v", err)
	}
}
