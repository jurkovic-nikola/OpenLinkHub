package backup

import (
	"LumenForge/src/config"
	"archive/zip"
	"bytes"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestBackupIncludesOnlyConfigurationAndMutableData(t *testing.T) {
	root := t.TempDir()
	applicationRoot := filepath.Join(root, "app")
	configRoot := filepath.Join(root, "config")
	dataRoot := filepath.Join(root, "data")
	paths, err := config.ResolvePaths(config.PathOptions{
		Mode:            config.ServiceModeUser,
		ApplicationRoot: applicationRoot,
		ConfigRoot:      configRoot,
		DataRoot:        dataRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = config.EnsureRuntimeDirectories(paths); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(config.UsePathsForTest(paths))

	if err = os.MkdirAll(filepath.Join(applicationRoot, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(applicationRoot, "static", "immutable.js"), []byte("immutable"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(applicationRoot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(applicationRoot, 0o755) })
	if err = os.WriteFile(paths.ConfigurationFile, []byte(`{"listenPort":27003}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(paths.MutableProfilesRoot, "device.json"), []byte(`{"serial":"device"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(paths.DashboardFile, []byte(`{"theme":"default"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	PerformBackup(recorder, httptest.NewRequest("GET", "/api/backup", nil))
	if recorder.Code != 200 {
		t.Fatalf("PerformBackup() status = %d: %s", recorder.Code, recorder.Body.String())
	}

	reader, err := zip.NewReader(bytes.NewReader(recorder.Body.Bytes()), int64(recorder.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		names = append(names, file.Name)
	}
	for _, required := range []string{"config.json", "database/", "database/profiles/device.json", "dashboard.json", hashFileName} {
		if !slices.Contains(names, required) {
			t.Errorf("backup entries %v do not include %q", names, required)
		}
	}
	for _, name := range names {
		if name == "immutable.js" || name == "static/immutable.js" || name == "app/static/immutable.js" {
			t.Fatalf("backup included immutable application asset %q", name)
		}
	}

	archivePath := filepath.Join(root, "backup.zip")
	if err = os.WriteFile(archivePath, recorder.Body.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	restoreConfigRoot := filepath.Join(root, "restore-config")
	restoreDataRoot := filepath.Join(root, "restore-data")
	preexistingConfig := filepath.Join(restoreConfigRoot, "config.json")
	preexistingMutable := filepath.Join(restoreDataRoot, "database", "profiles", "device.json")
	for _, destination := range []string{preexistingConfig, preexistingMutable} {
		if err = os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(destination, []byte("old"), 0o666); err != nil {
			t.Fatal(err)
		}
		if err = os.Chmod(destination, 0o666); err != nil {
			t.Fatal(err)
		}
	}
	if err = unzipFile(archivePath, restoreConfigRoot, restoreDataRoot); err != nil {
		t.Fatal(err)
	}
	for _, restored := range []string{
		filepath.Join(restoreConfigRoot, "config.json"),
		filepath.Join(restoreDataRoot, "database", "profiles", "device.json"),
		filepath.Join(restoreDataRoot, "dashboard.json"),
	} {
		if _, err = os.Stat(restored); err != nil {
			t.Errorf("restore did not create %q: %v", restored, err)
		}
	}
	for _, restored := range []string{preexistingConfig, preexistingMutable} {
		info, statErr := os.Stat(restored)
		if statErr != nil {
			t.Fatal(statErr)
		}
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Errorf("restored mode for %q = %#o, want 0600", restored, mode)
		}
	}
	if _, err = os.Stat(filepath.Join(applicationRoot, "config.json")); !os.IsNotExist(err) {
		t.Fatalf("restore wrote into application root: %v", err)
	}
}
