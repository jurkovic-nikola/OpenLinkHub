package server

import (
	"LumenForge/src/config"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestStaticAssetsResolveFromApplicationRoot(t *testing.T) {
	root := t.TempDir()
	applicationRoot := filepath.Join(root, "app")
	staticRoot := filepath.Join(applicationRoot, "static")
	if err := os.MkdirAll(staticRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticRoot, "path-test.txt"), []byte("from application root"), 0o444); err != nil {
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

	request := httptest.NewRequest(http.MethodGet, "/static/path-test.txt", nil)
	request.Host = "127.0.0.1"
	recorder := httptest.NewRecorder()
	setRoutes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "from application root" {
		t.Fatalf("static response = %d %q", recorder.Code, recorder.Body.String())
	}
}
