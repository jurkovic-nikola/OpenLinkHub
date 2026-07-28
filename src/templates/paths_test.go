package templates

import (
	"LumenForge/src/config"
	"os"
	"path/filepath"
	"testing"
)

func TestInitLoadsTemplatesFromApplicationRoot(t *testing.T) {
	root := t.TempDir()
	applicationRoot := filepath.Join(root, "app")
	templateRoot := filepath.Join(applicationRoot, "web")
	if err := os.MkdirAll(templateRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateRoot, "test.html"), []byte(`{{define "test"}}shipped template{{end}}`), 0o444); err != nil {
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

	originalTemplates := templates
	t.Cleanup(func() { templates = originalTemplates })
	Init()
	if GetTemplate().Lookup("test") == nil {
		t.Fatalf("template was not loaded from %q", paths.TemplateRoot)
	}
}
