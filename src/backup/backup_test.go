package backup

import (
	"LumenForge/src/config"
	"LumenForge/src/logger"
	"archive/zip"
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	logger.Init()

	if err = os.MkdirAll(filepath.Join(applicationRoot, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(applicationRoot, "static", "immutable.js"), []byte("immutable"), 0o644); err != nil {
		t.Fatal(err)
	}
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
		if strings.Contains(name, "immutable.js") {
			t.Fatalf("backup included immutable application asset %q", name)
		}
	}

	archivePath := filepath.Join(root, "backup.zip")
	if err = os.WriteFile(archivePath, recorder.Body.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	restoreConfigRoot, restoreDataRoot := prepareLiveRoots(t)
	if err = restoreBackup(archivePath, restoreConfigRoot, restoreDataRoot, defaultRestoreFileOps()); err != nil {
		t.Fatal(err)
	}
	for _, restored := range []string{
		filepath.Join(restoreConfigRoot, "config.json"),
		filepath.Join(restoreDataRoot, "database", "profiles", "device.json"),
		filepath.Join(restoreDataRoot, "dashboard.json"),
	} {
		info, statErr := os.Stat(restored)
		if statErr != nil {
			t.Errorf("restore did not create %q: %v", restored, statErr)
			continue
		}
		if !info.IsDir() && info.Mode().Perm() != 0o600 {
			t.Errorf("restored mode for %q = %#o, want 0600", restored, info.Mode().Perm())
		}
	}
	if _, err = os.Stat(filepath.Join(applicationRoot, "config.json")); !os.IsNotExist(err) {
		t.Fatalf("restore wrote into application root: %v", err)
	}
}

func TestRestoreUploadFilesAreUniquePrivateAndRemoved(t *testing.T) {
	first, err := createRestoreUpload()
	if err != nil {
		t.Fatal(err)
	}
	second, err := createRestoreUpload()
	if err != nil {
		_ = first.Close()
		_ = os.Remove(first.Name())
		t.Fatal(err)
	}
	firstName, secondName := first.Name(), second.Name()
	t.Cleanup(func() {
		_ = first.Close()
		_ = second.Close()
		_ = os.Remove(firstName)
		_ = os.Remove(secondName)
	})
	if firstName == secondName {
		t.Fatal("restore upload names are not unique")
	}
	for _, file := range []*os.File{first, second} {
		info, statErr := file.Stat()
		if statErr != nil {
			t.Fatal(statErr)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("restore upload mode = %#o, want 0600", info.Mode().Perm())
		}
	}
}

func TestPerformRestoreRequiresRestartAndCleansUpload(t *testing.T) {
	configRoot, dataRoot := prepareLiveRoots(t)
	paths, err := config.ResolvePaths(config.PathOptions{
		Mode:            config.ServiceModeUser,
		ApplicationRoot: filepath.Join(t.TempDir(), "app"),
		ConfigRoot:      configRoot,
		DataRoot:        dataRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(config.UsePathsForTest(paths))
	logger.Init()

	archive := testArchiveBytes(t, validRestoreEntries(), archiveOptions{})
	uploadDirectory := t.TempDir()
	t.Setenv("TMPDIR", uploadDirectory)
	request := multipartRestoreRequest(t, archive)
	recorder := httptest.NewRecorder()
	PerformRestore(recorder, request)
	if recorder.Code != 200 {
		t.Fatalf("PerformRestore() status = %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(strings.ToLower(recorder.Body.String()), "restart") {
		t.Fatalf("success response does not require restart: %q", recorder.Body.String())
	}
	files, err := os.ReadDir(uploadDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("restore upload directory not cleaned: %v", files)
	}

	badRequest := multipartRestoreRequest(t, []byte("not a zip"))
	badRecorder := httptest.NewRecorder()
	PerformRestore(badRecorder, badRequest)
	if badRecorder.Code != 400 {
		t.Fatalf("malformed restore status = %d, want 400", badRecorder.Code)
	}
	files, err = os.ReadDir(uploadDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("failed restore upload directory not cleaned: %v", files)
	}
}

func multipartRestoreRequest(t *testing.T, archive []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("backupFile", "../../predictable.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(archive); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/api/restore", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}
