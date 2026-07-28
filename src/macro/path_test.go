package macro

import (
	"LumenForge/src/config"
	"os"
	"path/filepath"
	"testing"
)

func TestNewMacroProfileWritesToMutableDatabaseRoot(t *testing.T) {
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
	originalMacros := macros
	t.Cleanup(func() {
		pwd = originalPWD
		location = originalLocation
		macros = originalMacros
	})
	macros = map[int]Macro{}

	Init()
	if status := NewMacroProfile("TestProfile"); status != 1 {
		t.Fatalf("NewMacroProfile() status = %d, want 1", status)
	}
	expected := filepath.Join(paths.MutableMacrosRoot, "testprofile.json")
	if _, err = os.Stat(expected); err != nil {
		t.Fatalf("macro profile was not written at %q: %v", expected, err)
	}
}
