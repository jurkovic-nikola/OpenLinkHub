package backup

import (
	"LumenForge/src/logger"
	"archive/zip"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	maxArchiveEntries       = 4096
	maxArchivePathDepth     = 16
	maxArchiveEntrySize     = 32 * 1024 * 1024
	maxArchiveTotalSize     = 128 * 1024 * 1024
	maxArchiveHashEntrySize = 128
)

type restoreErrorKind string

const (
	restoreInvalidStructure  restoreErrorKind = "invalid archive structure"
	restoreUnsupportedEntry  restoreErrorKind = "unsupported archive entry"
	restoreLimitExceeded     restoreErrorKind = "archive limit exceeded"
	restoreCorruptBackup     restoreErrorKind = "corrupt or mismatched backup"
	restoreMalformedJSON     restoreErrorKind = "malformed JSON"
	restoreUnsafeDestination restoreErrorKind = "unsafe current destination"
	restoreStagingFailure    restoreErrorKind = "staging failure"
	restoreCommitFailure     restoreErrorKind = "commit failure"
	restoreRollbackFailure   restoreErrorKind = "rollback failure"
)

type restoreError struct {
	kind restoreErrorKind
	err  error
}

func (err *restoreError) Error() string {
	return fmt.Sprintf("%s: %v", err.kind, err.err)
}

func (err *restoreError) Unwrap() error {
	return err.err
}

func newRestoreError(kind restoreErrorKind, format string, args ...any) error {
	return &restoreError{kind: kind, err: fmt.Errorf(format, args...)}
}

func restorePublicMessage(err error) string {
	var restoreErr *restoreError
	if !errors.As(err, &restoreErr) {
		return "Restore failed"
	}
	switch restoreErr.kind {
	case restoreInvalidStructure:
		return "Restore failed: invalid backup archive structure"
	case restoreUnsupportedEntry:
		return "Restore failed: backup contains an unsupported archive entry"
	case restoreLimitExceeded:
		return "Restore failed: backup archive limit exceeded"
	case restoreCorruptBackup:
		return "Restore failed: backup is corrupt or does not match its hash"
	case restoreMalformedJSON:
		return "Restore failed: backup contains malformed JSON"
	case restoreUnsafeDestination:
		return "Restore failed: current restore destination is unsafe"
	case restoreStagingFailure:
		return "Restore failed: unable to stage backup"
	case restoreCommitFailure:
		return "Restore failed while replacing current data; original data was restored"
	case restoreRollbackFailure:
		return "Restore failed and the original data could not be fully restored; check local logs"
	default:
		return "Restore failed"
	}
}

func logRestoreError(err error) {
	var restoreErr *restoreError
	if errors.As(err, &restoreErr) {
		logger.Log(logger.Fields{"error": restoreErr.err, "category": string(restoreErr.kind)}).Warn("Unable to restore backup")
		return
	}
	logger.Log(logger.Fields{"error": err}).Warn("Unable to restore backup")
}

type archiveManifest struct {
	files            []*zip.File
	dashboardPresent bool
	displayPresent   bool
}

type restoreStage struct {
	configurationDirectory string
	dataDirectory          string
	configurationStage     string
	dataStage              string
	configFile             string
	databaseDirectory      string
	dashboardFile          string
	displayFile            string
	dashboardPresent       bool
	displayPresent         bool
}

type restoreFileOps struct {
	lstat     func(string) (os.FileInfo, error)
	rename    func(string, string) error
	remove    func(string) error
	removeAll func(string) error
}

func defaultRestoreFileOps() restoreFileOps {
	return restoreFileOps{
		lstat:     os.Lstat,
		rename:    os.Rename,
		remove:    os.Remove,
		removeAll: os.RemoveAll,
	}
}

func restoreBackup(zipPath, configurationDirectory, dataDirectory string, ops restoreFileOps) (resultErr error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return newRestoreError(restoreInvalidStructure, "open ZIP: %w", err)
	}
	readerOpen := true
	defer func() {
		if readerOpen {
			if closeErr := reader.Close(); closeErr != nil {
				resultErr = errors.Join(resultErr, newRestoreError(restoreCorruptBackup, "close ZIP: %w", closeErr))
			}
		}
	}()

	manifest, err := inspectArchive(reader.File)
	if err != nil {
		return err
	}
	if err = validateRestoreDestinations(configurationDirectory, dataDirectory, ops); err != nil {
		return err
	}
	stage, err := createRestoreStage(configurationDirectory, dataDirectory, manifest)
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := cleanupRestoreStage(stage); cleanupErr != nil {
			logger.Log(logger.Fields{"error": cleanupErr}).Warn("Unable to remove restore staging directory")
		}
	}()

	if err = extractAndValidateArchive(manifest, stage); err != nil {
		return err
	}
	if err = reader.Close(); err != nil {
		return newRestoreError(restoreCorruptBackup, "close ZIP before commit: %w", err)
	}
	readerOpen = false
	if err = commitRestoredState(stage, ops); err != nil {
		return err
	}
	return nil
}

func inspectArchive(files []*zip.File) (*archiveManifest, error) {
	if len(files) == 0 {
		return nil, newRestoreError(restoreInvalidStructure, "archive is empty")
	}
	if len(files) > maxArchiveEntries {
		return nil, newRestoreError(restoreLimitExceeded, "archive has %d entries; maximum is %d", len(files), maxArchiveEntries)
	}

	manifest := &archiveManifest{files: files}
	seen := make(map[string]struct{}, len(files))
	var configCount, hashCount int
	var metadataTotal uint64

	for _, file := range files {
		canonical, directory, err := validateArchivePath(file)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[canonical]; exists {
			return nil, newRestoreError(restoreInvalidStructure, "duplicate canonical archive path %q", canonical)
		}
		seen[canonical] = struct{}{}

		if file.Flags&0x1 != 0 {
			return nil, newRestoreError(restoreUnsupportedEntry, "encrypted entry %q", canonical)
		}
		if file.Method != zip.Store && file.Method != zip.Deflate {
			return nil, newRestoreError(restoreUnsupportedEntry, "compression method %d for %q", file.Method, canonical)
		}

		mode := file.Mode()
		if directory {
			if mode&os.ModeType != 0 && !mode.IsDir() {
				return nil, newRestoreError(restoreUnsupportedEntry, "non-directory metadata for %q", canonical)
			}
			if file.UncompressedSize64 != 0 || file.CompressedSize64 != 0 {
				return nil, newRestoreError(restoreUnsupportedEntry, "directory entry %q contains data", canonical)
			}
		} else if mode&os.ModeType != 0 {
			return nil, newRestoreError(restoreUnsupportedEntry, "special file metadata for %q", canonical)
		}

		switch {
		case directory:
			if canonical != "database" && !strings.HasPrefix(canonical, "database/") {
				return nil, newRestoreError(restoreInvalidStructure, "unexpected directory %q", canonical)
			}
		case canonical == "config.json":
			configCount++
		case canonical == hashFileName:
			hashCount++
			if file.UncompressedSize64 > maxArchiveHashEntrySize {
				return nil, newRestoreError(restoreLimitExceeded, "%s exceeds %d bytes", hashFileName, maxArchiveHashEntrySize)
			}
		case canonical == "dashboard.json":
			manifest.dashboardPresent = true
		case canonical == "display.json":
			manifest.displayPresent = true
		case strings.HasPrefix(canonical, "database/"):
		default:
			return nil, newRestoreError(restoreInvalidStructure, "unexpected archive path %q", canonical)
		}

		if !directory && canonical != hashFileName {
			if file.UncompressedSize64 > maxArchiveEntrySize {
				return nil, newRestoreError(restoreLimitExceeded, "entry %q exceeds %d bytes", canonical, maxArchiveEntrySize)
			}
			if file.UncompressedSize64 > maxArchiveTotalSize-metadataTotal {
				return nil, newRestoreError(restoreLimitExceeded, "archive exceeds %d uncompressed bytes", maxArchiveTotalSize)
			}
			metadataTotal += file.UncompressedSize64
		}
	}

	if configCount != 1 {
		return nil, newRestoreError(restoreInvalidStructure, "config.json must appear exactly once")
	}
	if hashCount != 1 {
		return nil, newRestoreError(restoreInvalidStructure, "%s must appear exactly once", hashFileName)
	}
	return manifest, nil
}

func validateArchivePath(file *zip.File) (string, bool, error) {
	name := file.Name
	if name == "" || !utf8.ValidString(name) || strings.ContainsRune(name, '\x00') {
		return "", false, newRestoreError(restoreInvalidStructure, "archive contains an empty or invalid path")
	}
	if strings.Contains(name, `\`) {
		return "", false, newRestoreError(restoreInvalidStructure, "archive path uses backslashes")
	}
	if strings.HasPrefix(name, "/") || path.IsAbs(name) {
		return "", false, newRestoreError(restoreInvalidStructure, "archive contains an absolute path")
	}

	directory := strings.HasSuffix(name, "/")
	trimmed := strings.TrimSuffix(name, "/")
	if trimmed == "" {
		return "", false, newRestoreError(restoreInvalidStructure, "archive contains an empty path")
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) > maxArchivePathDepth {
		return "", false, newRestoreError(restoreLimitExceeded, "archive path depth exceeds %d", maxArchivePathDepth)
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", false, newRestoreError(restoreInvalidStructure, "archive contains an ambiguous or traversing path")
		}
	}

	canonical := path.Clean(trimmed)
	expected := canonical
	if directory {
		expected += "/"
	}
	if canonical == "." || canonical == ".." || strings.HasPrefix(canonical, "../") || name != expected {
		return "", false, newRestoreError(restoreInvalidStructure, "archive path is not canonical")
	}
	if file.Mode().IsDir() != directory && file.Mode()&os.ModeType != 0 {
		return "", false, newRestoreError(restoreUnsupportedEntry, "entry type does not match path %q", canonical)
	}
	if canonical == "database" && !directory {
		return "", false, newRestoreError(restoreInvalidStructure, "database must be a directory")
	}
	return canonical, directory, nil
}

func validateRestoreDestinations(configurationDirectory, dataDirectory string, ops restoreFileOps) error {
	roots := []string{configurationDirectory}
	if filepath.Clean(dataDirectory) != filepath.Clean(configurationDirectory) {
		roots = append(roots, dataDirectory)
	}
	for _, root := range roots {
		if err := validateExistingRoot(root, ops); err != nil {
			return err
		}
	}

	targets := []struct {
		path      string
		directory bool
	}{
		{filepath.Join(configurationDirectory, "config.json"), false},
		{filepath.Join(dataDirectory, "database"), true},
		{filepath.Join(dataDirectory, "dashboard.json"), false},
		{filepath.Join(dataDirectory, "display.json"), false},
	}
	for _, target := range targets {
		info, err := ops.lstat(target.path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return newRestoreError(restoreUnsafeDestination, "inspect destination: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return newRestoreError(restoreUnsafeDestination, "destination is a symbolic link")
		}
		if target.directory && !info.IsDir() {
			return newRestoreError(restoreUnsafeDestination, "database destination is not a directory")
		}
		if !target.directory && !info.Mode().IsRegular() {
			return newRestoreError(restoreUnsafeDestination, "file destination is not regular")
		}
	}
	return nil
}

func validateExistingRoot(root string, ops restoreFileOps) error {
	cleaned := filepath.Clean(root)
	if !filepath.IsAbs(cleaned) {
		return newRestoreError(restoreUnsafeDestination, "restore root is not absolute")
	}
	current := string(os.PathSeparator)
	for _, component := range strings.Split(strings.TrimPrefix(cleaned, string(os.PathSeparator)), string(os.PathSeparator)) {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, err := ops.lstat(current)
		if err != nil {
			return newRestoreError(restoreUnsafeDestination, "inspect restore root component: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return newRestoreError(restoreUnsafeDestination, "restore root contains a symbolic-link component")
		}
		if !info.IsDir() {
			return newRestoreError(restoreUnsafeDestination, "restore root component is not a directory")
		}
	}
	return nil
}

func createRestoreStage(configurationDirectory, dataDirectory string, manifest *archiveManifest) (*restoreStage, error) {
	configurationStage, err := os.MkdirTemp(configurationDirectory, ".lumenforge-restore-*")
	if err != nil {
		return nil, newRestoreError(restoreStagingFailure, "create configuration staging directory: %w", err)
	}
	if err = os.Chmod(configurationStage, 0o700); err != nil {
		_ = os.RemoveAll(configurationStage)
		return nil, newRestoreError(restoreStagingFailure, "protect configuration staging directory: %w", err)
	}

	dataStage := configurationStage
	if filepath.Clean(dataDirectory) != filepath.Clean(configurationDirectory) {
		dataStage, err = os.MkdirTemp(dataDirectory, ".lumenforge-restore-*")
		if err != nil {
			_ = os.RemoveAll(configurationStage)
			return nil, newRestoreError(restoreStagingFailure, "create data staging directory: %w", err)
		}
		if err = os.Chmod(dataStage, 0o700); err != nil {
			_ = os.RemoveAll(configurationStage)
			_ = os.RemoveAll(dataStage)
			return nil, newRestoreError(restoreStagingFailure, "protect data staging directory: %w", err)
		}
	}

	stage := &restoreStage{
		configurationDirectory: configurationDirectory,
		dataDirectory:          dataDirectory,
		configurationStage:     configurationStage,
		dataStage:              dataStage,
		configFile:             filepath.Join(configurationStage, "config.json"),
		databaseDirectory:      filepath.Join(dataStage, "database"),
		dashboardFile:          filepath.Join(dataStage, "dashboard.json"),
		displayFile:            filepath.Join(dataStage, "display.json"),
		dashboardPresent:       manifest.dashboardPresent,
		displayPresent:         manifest.displayPresent,
	}
	if err = os.Mkdir(stage.databaseDirectory, 0o700); err != nil {
		_ = cleanupRestoreStage(stage)
		return nil, newRestoreError(restoreStagingFailure, "create staged database: %w", err)
	}
	return stage, nil
}

func cleanupRestoreStage(stage *restoreStage) error {
	if stage == nil {
		return nil
	}
	var cleanupErrors []error
	if err := os.RemoveAll(stage.configurationStage); err != nil {
		cleanupErrors = append(cleanupErrors, err)
	}
	if stage.dataStage != stage.configurationStage {
		if err := os.RemoveAll(stage.dataStage); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	return errors.Join(cleanupErrors...)
}

func extractAndValidateArchive(manifest *archiveManifest, stage *restoreStage) error {
	hasher := sha256.New()
	var expectedHash string
	var totalSize int64

	for _, file := range manifest.files {
		canonical := strings.TrimSuffix(file.Name, "/")
		if strings.HasSuffix(file.Name, "/") {
			if canonical == "database" {
				continue
			}
			destination := filepath.Join(stage.dataStage, filepath.FromSlash(canonical))
			if err := os.MkdirAll(destination, 0o700); err != nil {
				return newRestoreError(restoreStagingFailure, "create staged directory: %w", err)
			}
			if err := os.Chmod(destination, 0o700); err != nil {
				return newRestoreError(restoreStagingFailure, "protect staged directory: %w", err)
			}
			continue
		}
		if canonical == hashFileName {
			hash, err := readAndValidateHash(file)
			if err != nil {
				return err
			}
			expectedHash = hash
			continue
		}

		destination := stagedDestination(stage, canonical)
		if destination == "" {
			return newRestoreError(restoreInvalidStructure, "archive path has no restore destination")
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return newRestoreError(restoreStagingFailure, "create staged parent directory: %w", err)
		}
		output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return newRestoreError(restoreStagingFailure, "create staged file: %w", err)
		}
		if err = output.Chmod(0o600); err != nil {
			closeErr := output.Close()
			return newRestoreError(restoreStagingFailure, "protect staged file: %w", errors.Join(err, closeErr))
		}

		written, err := copyArchiveEntry(file, output, hasher, maxArchiveTotalSize-totalSize)
		totalSize += written
		if err != nil {
			return err
		}
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if expectedHash == "" || expectedHash != actualHash {
		return newRestoreError(restoreCorruptBackup, "backup hash mismatch")
	}
	if err := validateStagedJSON(stage); err != nil {
		return err
	}
	if err := preserveHostConfiguration(stage.configFile, filepath.Join(stage.configurationDirectory, "config.json")); err != nil {
		return err
	}
	return nil
}

func stagedDestination(stage *restoreStage, canonical string) string {
	switch canonical {
	case "config.json":
		return stage.configFile
	case "dashboard.json":
		return stage.dashboardFile
	case "display.json":
		return stage.displayFile
	default:
		if strings.HasPrefix(canonical, "database/") {
			return filepath.Join(stage.dataStage, filepath.FromSlash(canonical))
		}
		return ""
	}
}

func copyArchiveEntry(file *zip.File, output *os.File, hasher io.Writer, remainingTotal int64) (written int64, resultErr error) {
	reader, err := file.Open()
	if err != nil {
		closeErr := output.Close()
		openErr := newRestoreError(restoreCorruptBackup, "open compressed entry: %w", err)
		if closeErr != nil {
			return 0, errors.Join(openErr, newRestoreError(restoreStagingFailure, "close staged file after entry open failure: %w", closeErr))
		}
		return 0, openErr
	}
	defer func() {
		readerCloseErr := reader.Close()
		outputCloseErr := output.Close()
		if readerCloseErr != nil {
			resultErr = errors.Join(resultErr, newRestoreError(restoreCorruptBackup, "close compressed entry: %w", readerCloseErr))
		}
		if outputCloseErr != nil {
			resultErr = errors.Join(resultErr, newRestoreError(restoreStagingFailure, "close staged file: %w", outputCloseErr))
		}
	}()

	return copyBoundedArchiveData(reader, output, hasher, remainingTotal)
}

func copyBoundedArchiveData(reader io.Reader, output io.Writer, hasher io.Writer, remainingTotal int64) (written int64, resultErr error) {
	allowed := int64(maxArchiveEntrySize)
	limitKind := "individual entry"
	if remainingTotal < allowed {
		allowed = remainingTotal
		limitKind = "cumulative archive"
	}
	if allowed < 0 {
		return 0, newRestoreError(restoreLimitExceeded, "archive exceeds uncompressed size limit")
	}

	buffer := make([]byte, 32*1024)
	for {
		readSize := len(buffer)
		if remaining := allowed - written + 1; remaining < int64(readSize) {
			readSize = int(remaining)
		}
		if readSize <= 0 {
			return written, newRestoreError(restoreLimitExceeded, "%s exceeds uncompressed size limit", limitKind)
		}
		count, readErr := reader.Read(buffer[:readSize])
		if count > 0 {
			if written+int64(count) > allowed {
				return written, newRestoreError(restoreLimitExceeded, "%s exceeds uncompressed size limit", limitKind)
			}
			outputCount, writeErr := output.Write(buffer[:count])
			if writeErr != nil {
				return written, newRestoreError(restoreStagingFailure, "write staged entry: %w", writeErr)
			}
			if outputCount != count {
				return written, newRestoreError(restoreStagingFailure, "write staged entry: %w", io.ErrShortWrite)
			}
			hashCount, hashErr := hasher.Write(buffer[:count])
			if hashErr != nil {
				return written, newRestoreError(restoreStagingFailure, "hash staged entry: %w", hashErr)
			}
			if hashCount != count {
				return written, newRestoreError(restoreStagingFailure, "hash staged entry: %w", io.ErrShortWrite)
			}
			written += int64(count)
		}
		if readErr == io.EOF {
			return written, nil
		}
		if readErr != nil {
			return written, newRestoreError(restoreCorruptBackup, "read compressed entry: %w", readErr)
		}
		if count == 0 {
			return written, newRestoreError(restoreCorruptBackup, "compressed entry made no read progress")
		}
	}
}

func readAndValidateHash(file *zip.File) (hash string, resultErr error) {
	reader, err := file.Open()
	if err != nil {
		return "", newRestoreError(restoreCorruptBackup, "open hash entry: %w", err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			resultErr = errors.Join(resultErr, newRestoreError(restoreCorruptBackup, "close hash entry: %w", closeErr))
		}
	}()

	data, err := io.ReadAll(io.LimitReader(reader, maxArchiveHashEntrySize+1))
	if err != nil {
		return "", newRestoreError(restoreCorruptBackup, "read hash entry: %w", err)
	}
	if len(data) > maxArchiveHashEntrySize {
		return "", newRestoreError(restoreLimitExceeded, "%s exceeds %d bytes", hashFileName, maxArchiveHashEntrySize)
	}
	hash = strings.TrimSpace(string(data))
	if len(hash) != sha256.Size*2 {
		return "", newRestoreError(restoreCorruptBackup, "malformed backup hash")
	}
	if _, err = hex.DecodeString(hash); err != nil {
		return "", newRestoreError(restoreCorruptBackup, "malformed backup hash")
	}
	return strings.ToLower(hash), nil
}

func validateStagedJSON(stage *restoreStage) error {
	for _, file := range []struct {
		path    string
		present bool
	}{
		{stage.configFile, true},
		{stage.dashboardFile, stage.dashboardPresent},
		{stage.displayFile, stage.displayPresent},
	} {
		if file.present {
			if err := validateJSONFile(file.path); err != nil {
				return err
			}
		}
	}
	return filepath.Walk(stage.databaseDirectory, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return newRestoreError(restoreStagingFailure, "inspect staged database: %w", walkErr)
		}
		if !info.IsDir() && strings.EqualFold(filepath.Ext(filePath), ".json") {
			return validateJSONFile(filePath)
		}
		return nil
	})
}

func validateJSONFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return newRestoreError(restoreStagingFailure, "read staged JSON: %w", err)
	}
	if !json.Valid(data) {
		return newRestoreError(restoreMalformedJSON, "staged JSON is malformed")
	}
	return nil
}

func preserveHostConfiguration(stagedPath, livePath string) error {
	stagedData, err := os.ReadFile(stagedPath)
	if err != nil {
		return newRestoreError(restoreStagingFailure, "read staged configuration: %w", err)
	}
	var staged map[string]json.RawMessage
	if err = json.Unmarshal(stagedData, &staged); err != nil || staged == nil {
		return newRestoreError(restoreMalformedJSON, "config.json must contain a JSON object")
	}

	live := make(map[string]json.RawMessage)
	liveData, readErr := os.ReadFile(livePath)
	if readErr == nil {
		if err = json.Unmarshal(liveData, &live); err != nil || live == nil {
			return newRestoreError(restoreUnsafeDestination, "current config.json cannot be safely preserved")
		}
	} else if !os.IsNotExist(readErr) {
		return newRestoreError(restoreUnsafeDestination, "read current config.json: %w", readErr)
	}

	for _, key := range []string{"logFile", "amdsmiPath"} {
		if value, exists := live[key]; exists {
			staged[key] = append(json.RawMessage(nil), value...)
		} else {
			delete(staged, key)
		}
	}
	updated, err := json.MarshalIndent(staged, "", "  ")
	if err != nil {
		return newRestoreError(restoreStagingFailure, "encode staged configuration: %w", err)
	}
	updated = append(updated, '\n')
	if err = os.WriteFile(stagedPath, updated, 0o600); err != nil {
		return newRestoreError(restoreStagingFailure, "write staged configuration: %w", err)
	}
	if err = os.Chmod(stagedPath, 0o600); err != nil {
		return newRestoreError(restoreStagingFailure, "protect staged configuration: %w", err)
	}
	return nil
}

type commitTarget struct {
	destination   string
	replacement   string
	directory     bool
	original      string
	originalMoved bool
	installed     bool
}

func commitRestoredState(stage *restoreStage, ops restoreFileOps) error {
	targets := []*commitTarget{
		{destination: filepath.Join(stage.configurationDirectory, "config.json"), replacement: stage.configFile},
		{destination: filepath.Join(stage.dataDirectory, "database"), replacement: stage.databaseDirectory, directory: true},
		{destination: filepath.Join(stage.dataDirectory, "dashboard.json")},
		{destination: filepath.Join(stage.dataDirectory, "display.json")},
	}
	if stage.dashboardPresent {
		targets[2].replacement = stage.dashboardFile
	}
	if stage.displayPresent {
		targets[3].replacement = stage.displayFile
	}

	for _, target := range targets {
		info, err := ops.lstat(target.destination)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return newRestoreError(restoreUnsafeDestination, "destination became a symbolic link")
			}
			if target.directory && !info.IsDir() {
				return newRestoreError(restoreUnsafeDestination, "database destination changed type")
			}
			if !target.directory && !info.Mode().IsRegular() {
				return newRestoreError(restoreUnsafeDestination, "file destination changed type")
			}
			target.original, err = uniqueOriginalPath(target.destination, ops)
			if err != nil {
				return newRestoreError(restoreCommitFailure, "prepare original backup name: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return newRestoreError(restoreCommitFailure, "inspect destination before commit: %w", err)
		}
	}

	applied := make([]*commitTarget, 0, len(targets))
	for _, target := range targets {
		applied = append(applied, target)
		if target.original != "" {
			if err := ops.rename(target.destination, target.original); err != nil {
				return commitFailureWithRollback(fmt.Errorf("move current target aside: %w", err), applied, ops)
			}
			target.originalMoved = true
		}
		if target.replacement != "" {
			if err := ops.rename(target.replacement, target.destination); err != nil {
				return commitFailureWithRollback(fmt.Errorf("install staged target: %w", err), applied, ops)
			}
			target.installed = true
		}
	}

	for _, target := range targets {
		if !target.originalMoved {
			continue
		}
		var err error
		if target.directory {
			err = ops.removeAll(target.original)
		} else {
			err = ops.remove(target.original)
		}
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to remove temporary original after restore")
		}
	}
	return nil
}

func uniqueOriginalPath(destination string, ops restoreFileOps) (string, error) {
	for attempt := 0; attempt < 16; attempt++ {
		random := make([]byte, 12)
		if _, err := rand.Read(random); err != nil {
			return "", err
		}
		candidate := filepath.Join(
			filepath.Dir(destination),
			fmt.Sprintf(".lumenforge-restore-original-%s-%s", filepath.Base(destination), hex.EncodeToString(random)),
		)
		if _, err := ops.lstat(candidate); os.IsNotExist(err) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("unable to allocate a unique original path")
}

func commitFailureWithRollback(commitErr error, applied []*commitTarget, ops restoreFileOps) error {
	if rollbackErr := rollbackRestoredState(applied, ops); rollbackErr != nil {
		return newRestoreError(restoreRollbackFailure, "commit error: %v; rollback error: %w", commitErr, rollbackErr)
	}
	return newRestoreError(restoreCommitFailure, "%w", commitErr)
}

func rollbackRestoredState(applied []*commitTarget, ops restoreFileOps) error {
	var rollbackErrors []error
	for index := len(applied) - 1; index >= 0; index-- {
		target := applied[index]
		if target.installed {
			var err error
			if target.directory {
				err = ops.removeAll(target.destination)
			} else {
				err = ops.remove(target.destination)
			}
			if err != nil && !os.IsNotExist(err) {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("remove installed target: %w", err))
				continue
			}
		}
		if target.originalMoved {
			if err := ops.rename(target.original, target.destination); err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("restore original target: %w", err))
			}
		}
	}
	return errors.Join(rollbackErrors...)
}
