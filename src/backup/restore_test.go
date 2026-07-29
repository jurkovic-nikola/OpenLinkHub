package backup

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testArchiveEntry struct {
	name   string
	data   []byte
	mode   os.FileMode
	method uint16
	stored bool
}

type archiveOptions struct {
	hashCount int
	hashValue string
}

func validRestoreEntries() []testArchiveEntry {
	return []testArchiveEntry{
		{name: "database/", mode: os.ModeDir | 0o700},
		{name: "database/profiles/", mode: os.ModeDir | 0o700},
		{name: "database/profiles/device.json", data: []byte(`{"profile":"restored"}`)},
		{name: "database/lcd/images/image.bin", data: []byte("non-json mutable data")},
		{name: "config.json", data: []byte(`{"listenPort":28000,"logFile":"/archive/log","amdsmiPath":"/archive/amd-smi","archiveUnknown":{"kept":true}}`)},
		{name: "dashboard.json", data: []byte(`{"theme":"restored"}`)},
		{name: "display.json", data: []byte(`{"order":[1,2]}`)},
	}
}

func testArchiveBytes(t *testing.T, entries []testArchiveEntry, options archiveOptions) []byte {
	t.Helper()
	if options.hashCount == 0 {
		options.hashCount = 1
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	hasher := sha256.New()
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: entry.method}
		if strings.HasSuffix(entry.name, "/") || entry.stored {
			header.Method = zip.Store
		} else if header.Method == 0 {
			header.Method = zip.Deflate
		}
		mode := entry.mode
		if mode == 0 {
			if strings.HasSuffix(entry.name, "/") {
				mode = os.ModeDir | 0o700
			} else {
				mode = 0o600
			}
		}
		header.SetMode(mode)
		entryWriter, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = entryWriter.Write(entry.data); err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(entry.name, "/") && entry.name != hashFileName {
			_, _ = hasher.Write(entry.data)
		}
	}
	hashValue := options.hashValue
	if hashValue == "" {
		hashValue = hex.EncodeToString(hasher.Sum(nil))
	}
	for index := 0; index < options.hashCount; index++ {
		hashWriter, err := writer.Create(hashFileName)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = hashWriter.Write([]byte(hashValue)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func writeTestArchive(t *testing.T, entries []testArchiveEntry, options archiveOptions) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "backup.zip")
	if err := os.WriteFile(archivePath, testArchiveBytes(t, entries, options), 0o600); err != nil {
		t.Fatal(err)
	}
	return archivePath
}

func prepareLiveRoots(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	configRoot := filepath.Join(root, "config")
	dataRoot := filepath.Join(root, "data")
	if err := os.MkdirAll(configRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dataRoot, "database"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(configRoot, "config.json"),
		[]byte(`{"listenPort":27003,"logFile":"/live/log","amdsmiPath":"/live/amd-smi","liveOnly":"not-preserved"}`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "database", "old.json"), []byte(`{"old":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "dashboard.json"), []byte(`{"theme":"old"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "display.json"), []byte(`{"old":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return configRoot, dataRoot
}

func requireRestoreKind(t *testing.T, err error, kind restoreErrorKind) {
	t.Helper()
	if err == nil {
		t.Fatalf("restore unexpectedly succeeded; want %s", kind)
	}
	var restoreErr *restoreError
	if !errors.As(err, &restoreErr) {
		t.Fatalf("error %T %v is not a restoreError", err, err)
	}
	if restoreErr.kind != kind {
		t.Fatalf("restore error kind = %q, want %q: %v", restoreErr.kind, kind, err)
	}
}

func TestRestoreValidArchivePreservesHostConfigAndUsesSnapshotSemantics(t *testing.T) {
	configRoot, dataRoot := prepareLiveRoots(t)
	archive := writeTestArchive(t, validRestoreEntries(), archiveOptions{})
	if err := restoreBackup(archive, configRoot, dataRoot, defaultRestoreFileOps()); err != nil {
		t.Fatal(err)
	}

	configData, err := os.ReadFile(filepath.Join(configRoot, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var restored map[string]json.RawMessage
	if err = json.Unmarshal(configData, &restored); err != nil {
		t.Fatal(err)
	}
	for key, expected := range map[string]string{
		"logFile":    `"/live/log"`,
		"amdsmiPath": `"/live/amd-smi"`,
		"listenPort": `28000`,
	} {
		if string(restored[key]) != expected {
			t.Errorf("%s = %s, want %s", key, restored[key], expected)
		}
	}
	if _, exists := restored["archiveUnknown"]; !exists {
		t.Error("unknown archive config field was discarded")
	}
	if _, exists := restored["liveOnly"]; exists {
		t.Error("unrelated live config field incorrectly overlaid the archive")
	}
	if _, err = os.Stat(filepath.Join(dataRoot, "database", "old.json")); !os.IsNotExist(err) {
		t.Fatalf("old database file survived snapshot restore: %v", err)
	}
	for _, expected := range []string{
		filepath.Join(dataRoot, "database", "profiles", "device.json"),
		filepath.Join(dataRoot, "database", "lcd", "images", "image.bin"),
		filepath.Join(dataRoot, "dashboard.json"),
		filepath.Join(dataRoot, "display.json"),
	} {
		info, statErr := os.Stat(expected)
		if statErr != nil {
			t.Errorf("missing restored path %q: %v", expected, statErr)
			continue
		}
		if !info.IsDir() && info.Mode().Perm() != 0o600 {
			t.Errorf("mode for %q = %#o, want 0600", expected, info.Mode().Perm())
		}
	}
}

func TestRestoreAbsentOptionalFilesRemovesLiveCopies(t *testing.T) {
	configRoot, dataRoot := prepareLiveRoots(t)
	entries := []testArchiveEntry{
		{name: "database/", mode: os.ModeDir | 0o700},
		{name: "database/current.json", data: []byte(`{"current":true}`)},
		{name: "config.json", data: []byte(`{"listenPort":28000}`)},
	}
	if err := restoreBackup(writeTestArchive(t, entries, archiveOptions{}), configRoot, dataRoot, defaultRestoreFileOps()); err != nil {
		t.Fatal(err)
	}
	for _, optional := range []string{"dashboard.json", "display.json"} {
		if _, err := os.Stat(filepath.Join(dataRoot, optional)); !os.IsNotExist(err) {
			t.Errorf("absent optional file %q was retained: %v", optional, err)
		}
	}
}

func TestRestoreRejectsInvalidArchiveStructures(t *testing.T) {
	validConfig := testArchiveEntry{name: "config.json", data: []byte(`{"listenPort":27003}`)}
	tests := []struct {
		name     string
		entries  []testArchiveEntry
		options  archiveOptions
		wantKind restoreErrorKind
	}{
		{name: "parent traversal", entries: []testArchiveEntry{validConfig, {name: "../escape.json", data: []byte(`{}`)}}, wantKind: restoreInvalidStructure},
		{name: "nested traversal", entries: []testArchiveEntry{validConfig, {name: "database/a/../../escape", data: []byte("x")}}, wantKind: restoreInvalidStructure},
		{name: "absolute", entries: []testArchiveEntry{validConfig, {name: "/etc/passwd", data: []byte("x")}}, wantKind: restoreInvalidStructure},
		{name: "backslash", entries: []testArchiveEntry{validConfig, {name: `database\escape`, data: []byte("x")}}, wantKind: restoreInvalidStructure},
		{name: "unexpected top level", entries: []testArchiveEntry{validConfig, {name: "install.sh", data: []byte("x")}}, wantKind: restoreInvalidStructure},
		{name: "duplicate path", entries: []testArchiveEntry{validConfig, {name: "database/a.json", data: []byte(`{}`)}, {name: "database/a.json", data: []byte(`{}`)}}, wantKind: restoreInvalidStructure},
		{name: "canonical alias duplicate", entries: []testArchiveEntry{validConfig, {name: "database/b.json", data: []byte(`{}`)}, {name: "database/a/../b.json", data: []byte(`{}`)}}, wantKind: restoreInvalidStructure},
		{name: "duplicate config", entries: []testArchiveEntry{validConfig, validConfig}, wantKind: restoreInvalidStructure},
		{name: "duplicate hash", entries: []testArchiveEntry{validConfig}, options: archiveOptions{hashCount: 2}, wantKind: restoreInvalidStructure},
		{name: "missing config", entries: []testArchiveEntry{{name: "database/a.json", data: []byte(`{}`)}}, wantKind: restoreInvalidStructure},
		{name: "missing hash", entries: []testArchiveEntry{validConfig}, options: archiveOptions{hashCount: -1}, wantKind: restoreInvalidStructure},
		{name: "symlink", entries: []testArchiveEntry{validConfig, {name: "database/link", data: []byte("target"), mode: os.ModeSymlink | 0o777}}, wantKind: restoreUnsupportedEntry},
		{name: "named pipe", entries: []testArchiveEntry{validConfig, {name: "database/pipe", mode: os.ModeNamedPipe | 0o600}}, wantKind: restoreUnsupportedEntry},
		{name: "device", entries: []testArchiveEntry{validConfig, {name: "database/device", mode: os.ModeDevice | 0o600}}, wantKind: restoreUnsupportedEntry},
		{name: "malformed hash", entries: []testArchiveEntry{validConfig}, options: archiveOptions{hashValue: "not-a-sha256"}, wantKind: restoreCorruptBackup},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configRoot, dataRoot := prepareLiveRoots(t)
			options := test.options
			archiveBytes := testArchiveBytesWithOptionalHash(t, test.entries, options)
			archivePath := filepath.Join(t.TempDir(), "invalid.zip")
			if err := os.WriteFile(archivePath, archiveBytes, 0o600); err != nil {
				t.Fatal(err)
			}
			err := restoreBackup(archivePath, configRoot, dataRoot, defaultRestoreFileOps())
			requireRestoreKind(t, err, test.wantKind)
		})
	}
}

func testArchiveBytesWithOptionalHash(t *testing.T, entries []testArchiveEntry, options archiveOptions) []byte {
	t.Helper()
	if options.hashCount >= 0 {
		return testArchiveBytes(t, entries, options)
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		header.SetMode(0o600)
		entryWriter, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = entryWriter.Write(entry.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func TestRestoreRejectsArchiveLimits(t *testing.T) {
	regular := func(name string, size uint64) *zip.File {
		return &zip.File{FileHeader: zip.FileHeader{
			Name:               name,
			Method:             zip.Store,
			UncompressedSize64: size,
			CompressedSize64:   size,
		}}
	}
	required := []*zip.File{regular("config.json", 2), regular(hashFileName, 64)}

	t.Run("entry count", func(t *testing.T) {
		files := make([]*zip.File, maxArchiveEntries+1)
		for index := range files {
			files[index] = regular(fmt.Sprintf("database/%d.json", index), 2)
		}
		_, err := inspectArchive(files)
		requireRestoreKind(t, err, restoreLimitExceeded)
	})
	t.Run("path depth", func(t *testing.T) {
		deep := "database/" + strings.Repeat("a/", maxArchivePathDepth-1) + "value.json"
		_, err := inspectArchive(append(required, regular(deep, 2)))
		requireRestoreKind(t, err, restoreLimitExceeded)
	})
	t.Run("individual size", func(t *testing.T) {
		_, err := inspectArchive(append(required, regular("database/large.bin", maxArchiveEntrySize+1)))
		requireRestoreKind(t, err, restoreLimitExceeded)
	})
	t.Run("cumulative size", func(t *testing.T) {
		files := append([]*zip.File{}, required...)
		for index := 0; index < 5; index++ {
			files = append(files, regular(fmt.Sprintf("database/%d.bin", index), 30*1024*1024))
		}
		_, err := inspectArchive(files)
		requireRestoreKind(t, err, restoreLimitExceeded)
	})
	t.Run("hash size", func(t *testing.T) {
		_, err := inspectArchive([]*zip.File{regular("config.json", 2), regular(hashFileName, maxArchiveHashEntrySize+1)})
		requireRestoreKind(t, err, restoreLimitExceeded)
	})
	t.Run("bounded reader overrun", func(t *testing.T) {
		_, err := copyBoundedArchiveData(strings.NewReader("four"), io.Discard, sha256.New(), 3)
		requireRestoreKind(t, err, restoreLimitExceeded)
	})
}

func TestRestorePropagatesReadAndWriteFailures(t *testing.T) {
	t.Run("read failure", func(t *testing.T) {
		_, err := copyBoundedArchiveData(&failingReader{}, io.Discard, sha256.New(), maxArchiveTotalSize)
		requireRestoreKind(t, err, restoreCorruptBackup)
	})
	t.Run("write failure", func(t *testing.T) {
		_, err := copyBoundedArchiveData(strings.NewReader("data"), failingWriter{}, sha256.New(), maxArchiveTotalSize)
		requireRestoreKind(t, err, restoreStagingFailure)
	})
}

type failingReader struct{}

func (*failingReader) Read([]byte) (int, error) {
	return 0, errors.New("injected read failure")
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("injected write failure")
}

func TestRestoreRejectsCorruptAndTruncatedZIPData(t *testing.T) {
	entries := validRestoreEntries()
	for index := range entries {
		if !strings.HasSuffix(entries[index].name, "/") {
			entries[index].stored = true
		}
	}
	valid := testArchiveBytes(t, entries, archiveOptions{})

	t.Run("CRC mismatch", func(t *testing.T) {
		corrupt := append([]byte(nil), valid...)
		reader, err := zip.NewReader(bytes.NewReader(corrupt), int64(len(corrupt)))
		if err != nil {
			t.Fatal(err)
		}
		var target *zip.File
		for _, file := range reader.File {
			if file.Name == "database/profiles/device.json" {
				target = file
				break
			}
		}
		if target == nil {
			t.Fatal("fixture target not found")
		}
		offset, err := target.DataOffset()
		if err != nil {
			t.Fatal(err)
		}
		corrupt[offset] ^= 0xff
		archive := filepath.Join(t.TempDir(), "corrupt.zip")
		if err = os.WriteFile(archive, corrupt, 0o600); err != nil {
			t.Fatal(err)
		}
		configRoot, dataRoot := prepareLiveRoots(t)
		requireRestoreKind(t, restoreBackup(archive, configRoot, dataRoot, defaultRestoreFileOps()), restoreCorruptBackup)
	})

	t.Run("truncated stream", func(t *testing.T) {
		archive := filepath.Join(t.TempDir(), "truncated.zip")
		if err := os.WriteFile(archive, valid[:len(valid)-20], 0o600); err != nil {
			t.Fatal(err)
		}
		configRoot, dataRoot := prepareLiveRoots(t)
		requireRestoreKind(t, restoreBackup(archive, configRoot, dataRoot, defaultRestoreFileOps()), restoreInvalidStructure)
	})
}

func TestRestoreValidationFailureLeavesLiveStateAndCleansStages(t *testing.T) {
	tests := []struct {
		name    string
		entries []testArchiveEntry
	}{
		{
			name: "malformed config",
			entries: []testArchiveEntry{
				{name: "database/valid.json", data: []byte(`{"valid":true}`)},
				{name: "config.json", data: []byte(`{"broken":`)},
			},
		},
		{
			name: "malformed database JSON",
			entries: []testArchiveEntry{
				{name: "database/broken.json", data: []byte(`{"broken":`)},
				{name: "database/valid.bin", data: []byte("supported non-json")},
				{name: "config.json", data: []byte(`{"listenPort":28000}`)},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configRoot, dataRoot := prepareLiveRoots(t)
			configBefore, _ := os.ReadFile(filepath.Join(configRoot, "config.json"))
			databaseBefore, _ := os.ReadFile(filepath.Join(dataRoot, "database", "old.json"))
			err := restoreBackup(writeTestArchive(t, test.entries, archiveOptions{}), configRoot, dataRoot, defaultRestoreFileOps())
			requireRestoreKind(t, err, restoreMalformedJSON)
			configAfter, _ := os.ReadFile(filepath.Join(configRoot, "config.json"))
			databaseAfter, _ := os.ReadFile(filepath.Join(dataRoot, "database", "old.json"))
			if !bytes.Equal(configBefore, configAfter) || !bytes.Equal(databaseBefore, databaseAfter) {
				t.Fatal("validation failure changed live state")
			}
			requireNoRestoreArtifacts(t, configRoot, dataRoot)
		})
	}
}

func TestRestoreRejectsSymlinkedDestinationOrRoot(t *testing.T) {
	archive := writeTestArchive(t, validRestoreEntries(), archiveOptions{})
	t.Run("destination", func(t *testing.T) {
		configRoot, dataRoot := prepareLiveRoots(t)
		dashboard := filepath.Join(dataRoot, "dashboard.json")
		if err := os.Remove(dashboard); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(dataRoot, "display.json"), dashboard); err != nil {
			t.Fatal(err)
		}
		requireRestoreKind(t, restoreBackup(archive, configRoot, dataRoot, defaultRestoreFileOps()), restoreUnsafeDestination)
	})
	t.Run("root component", func(t *testing.T) {
		root := t.TempDir()
		realConfig, dataRoot := prepareLiveRoots(t)
		configLink := filepath.Join(root, "config-link")
		if err := os.Symlink(realConfig, configLink); err != nil {
			t.Fatal(err)
		}
		requireRestoreKind(t, restoreBackup(archive, configLink, dataRoot, defaultRestoreFileOps()), restoreUnsafeDestination)
	})
}

func TestRestoreCommitFailureRollsBackOriginalTargets(t *testing.T) {
	configRoot, dataRoot := prepareLiveRoots(t)
	archive := writeTestArchive(t, validRestoreEntries(), archiveOptions{})
	configBefore, _ := os.ReadFile(filepath.Join(configRoot, "config.json"))
	databaseBefore, _ := os.ReadFile(filepath.Join(dataRoot, "database", "old.json"))

	ops := defaultRestoreFileOps()
	renameCount := 0
	realRename := ops.rename
	ops.rename = func(oldPath, newPath string) error {
		renameCount++
		if renameCount == 4 {
			return errors.New("injected database install failure")
		}
		return realRename(oldPath, newPath)
	}
	err := restoreBackup(archive, configRoot, dataRoot, ops)
	requireRestoreKind(t, err, restoreCommitFailure)

	configAfter, _ := os.ReadFile(filepath.Join(configRoot, "config.json"))
	databaseAfter, _ := os.ReadFile(filepath.Join(dataRoot, "database", "old.json"))
	if !bytes.Equal(configBefore, configAfter) || !bytes.Equal(databaseBefore, databaseAfter) {
		t.Fatal("commit failure did not restore original targets")
	}
	requireNoRestoreArtifacts(t, configRoot, dataRoot)
}

func TestRestoreFailureBeforeFirstCommitLeavesOriginals(t *testing.T) {
	configRoot, dataRoot := prepareLiveRoots(t)
	archive := writeTestArchive(t, validRestoreEntries(), archiveOptions{})
	configBefore, _ := os.ReadFile(filepath.Join(configRoot, "config.json"))

	ops := defaultRestoreFileOps()
	ops.rename = func(string, string) error {
		return errors.New("injected first commit failure")
	}
	err := restoreBackup(archive, configRoot, dataRoot, ops)
	requireRestoreKind(t, err, restoreCommitFailure)
	configAfter, _ := os.ReadFile(filepath.Join(configRoot, "config.json"))
	if !bytes.Equal(configBefore, configAfter) {
		t.Fatal("failure before first rename changed config")
	}
}

func TestRestoreRollbackFailureIsSurfaced(t *testing.T) {
	configRoot, dataRoot := prepareLiveRoots(t)
	archive := writeTestArchive(t, validRestoreEntries(), archiveOptions{})
	ops := defaultRestoreFileOps()
	realRename := ops.rename
	renameCount := 0
	ops.rename = func(oldPath, newPath string) error {
		renameCount++
		if renameCount == 4 {
			return errors.New("injected commit failure")
		}
		if strings.Contains(filepath.Base(oldPath), ".lumenforge-restore-original-database-") &&
			newPath == filepath.Join(dataRoot, "database") {
			return errors.New("injected rollback failure")
		}
		return realRename(oldPath, newPath)
	}
	requireRestoreKind(t, restoreBackup(archive, configRoot, dataRoot, ops), restoreRollbackFailure)
}

func TestRestoreSuccessfulCommitRemovesTemporaryArtifacts(t *testing.T) {
	configRoot, dataRoot := prepareLiveRoots(t)
	if err := restoreBackup(writeTestArchive(t, validRestoreEntries(), archiveOptions{}), configRoot, dataRoot, defaultRestoreFileOps()); err != nil {
		t.Fatal(err)
	}
	requireNoRestoreArtifacts(t, configRoot, dataRoot)
}

func TestRestoreStageUsesRestrictivePermissionsAndCleansUp(t *testing.T) {
	configRoot, dataRoot := prepareLiveRoots(t)
	archivePath := writeTestArchive(t, validRestoreEntries(), archiveOptions{})
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	manifest, err := inspectArchive(reader.File)
	if err != nil {
		t.Fatal(err)
	}
	stage, err := createRestoreStage(configRoot, dataRoot, manifest)
	if err != nil {
		t.Fatal(err)
	}
	stageRoots := []string{stage.configurationStage, stage.dataStage}
	for _, stageRoot := range stageRoots {
		info, statErr := os.Stat(stageRoot)
		if statErr != nil {
			t.Fatal(statErr)
		}
		if info.Mode().Perm() != 0o700 {
			t.Errorf("staging directory mode = %#o, want 0700", info.Mode().Perm())
		}
	}
	if err = extractAndValidateArchive(manifest, stage); err != nil {
		t.Fatal(err)
	}
	for _, stagedFile := range []string{
		stage.configFile,
		filepath.Join(stage.databaseDirectory, "profiles", "device.json"),
		stage.dashboardFile,
		stage.displayFile,
	} {
		info, statErr := os.Stat(stagedFile)
		if statErr != nil {
			t.Fatal(statErr)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("staged file mode = %#o, want 0600", info.Mode().Perm())
		}
	}
	if err = cleanupRestoreStage(stage); err != nil {
		t.Fatal(err)
	}
	for _, stageRoot := range stageRoots {
		if _, statErr := os.Stat(stageRoot); !os.IsNotExist(statErr) {
			t.Errorf("staging directory remains after cleanup: %v", statErr)
		}
	}
}

func requireNoRestoreArtifacts(t *testing.T, roots ...string) {
	t.Helper()
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".lumenforge-restore-") {
				t.Errorf("restore artifact remains in %q: %s", root, entry.Name())
			}
		}
	}
}

func TestPreserveHostConfigurationDeletesAbsentHostLocalFields(t *testing.T) {
	root := t.TempDir()
	staged := filepath.Join(root, "staged.json")
	live := filepath.Join(root, "live.json")
	if err := os.WriteFile(staged, []byte(`{"logFile":"/archive","amdsmiPath":"/archive/amd","other":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(live, []byte(`{"listenPort":27003}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := preserveHostConfiguration(staged, live); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(staged)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]json.RawMessage
	if err = json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	if _, exists := value["logFile"]; exists {
		t.Error("archived logFile survived when live config omitted it")
	}
	if _, exists := value["amdsmiPath"]; exists {
		t.Error("archived amdsmiPath survived when live config omitted it")
	}
	if _, exists := value["other"]; !exists {
		t.Error("unrelated archived field was discarded")
	}
}

func TestRestoreDevelopmentRootsCannotReachApplicationFiles(t *testing.T) {
	root := t.TempDir()
	configRoot := root
	dataRoot := root
	for _, directory := range []string{"database", "src", "static", "web"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "config.json"), []byte(`{"listenPort":27003}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "sentinel.go"), []byte("package sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "install.sh"), []byte("sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	entries := []testArchiveEntry{
		{name: "database/profile.json", data: []byte(`{"restored":true}`)},
		{name: "config.json", data: []byte(`{"listenPort":28000}`)},
	}
	if err := restoreBackup(writeTestArchive(t, entries, archiveOptions{}), configRoot, dataRoot, defaultRestoreFileOps()); err != nil {
		t.Fatal(err)
	}
	source, _ := os.ReadFile(filepath.Join(root, "src", "sentinel.go"))
	installer, _ := os.ReadFile(filepath.Join(root, "install.sh"))
	if string(source) != "package sentinel" || string(installer) != "sentinel" {
		t.Fatal("development restore changed application/repository files")
	}
}

func TestHashWriterFailureIsPropagated(t *testing.T) {
	_, err := copyBoundedArchiveData(strings.NewReader("data"), io.Discard, failingHash{}, maxArchiveTotalSize)
	requireRestoreKind(t, err, restoreStagingFailure)
}

type failingHash struct{}

func (failingHash) Write([]byte) (int, error) {
	return 0, errors.New("injected hash failure")
}
