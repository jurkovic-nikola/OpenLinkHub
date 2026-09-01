package backup

// Package: backup
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"archive/zip"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	maxUploadSize       = 5 * 1024 * 1024 // 5 MB
	maxRequestSize      = maxUploadSize + 1*1024*1024
	maxUncompressedSize = 100 * 1024 * 1024 // 100 MB
	maxArchiveFiles     = 10000
	maxHashFileSize     = 128
	hashFileName        = "_hash.txt"
)

// PerformBackup creates a ZIP with SHA-256 integrity hash.
func PerformBackup(w http.ResponseWriter, _ *http.Request) {
	cfg := config.GetConfig()
	srcFolder := filepath.Join(cfg.ConfigPath, "database")
	extraFile := filepath.Join(cfg.ConfigPath, "config.json")
	backupName := "backup_" + time.Now().Format("2006-01-02-15-04-05") + ".zip"

	tmpFile, err := os.CreateTemp("", "openlinkhub-backup-*.zip")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Warn("Unable to create temporary database backup")
		http.Error(w, "Unable to create backup", http.StatusInternalServerError)
		return
	}
	tmpName := tmpFile.Name()

	defer func() {
		if err := os.Remove(tmpName); err != nil && !os.IsNotExist(err) {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to remove temporary database backup")
		}
	}()

	defer func() {
		if err := tmpFile.Close(); err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to close temporary database backup")
		}
	}()

	archive := zip.NewWriter(tmpFile)
	archiveClosed := false

	defer func() {
		if !archiveClosed {
			if err := archive.Close(); err != nil {
				logger.Log(logger.Fields{"error": err}).Warn("Unable to close incomplete backup archive")
			}
		}
	}()

	hasher := sha256.New()

	// Add database folder
	if err := hashAndZipFolder(srcFolder, archive, hasher); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to add database folder to backup")
		http.Error(w, "Unable to create backup", http.StatusInternalServerError)
		return
	}

	// Add config.json
	if err := hashAndZipFile(extraFile, archive, hasher); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to add config.json to backup")
		http.Error(w, "Unable to create backup", http.StatusInternalServerError)
		return
	}

	// Write hash file
	sum := hex.EncodeToString(hasher.Sum(nil))
	hf, err := archive.Create(hashFileName)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to create hash file in backup archive")
		http.Error(w, "Unable to create hash file in archive", http.StatusInternalServerError)
		return
	}

	if _, err := io.WriteString(hf, sum); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write hash file in backup archive")
		http.Error(w, "Unable to write hash file", http.StatusInternalServerError)
		return
	}

	if err := archive.Close(); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to finalize backup archive")
		http.Error(w, "Unable to finalize backup", http.StatusInternalServerError)
		return
	}
	archiveClosed = true

	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to seek temporary database backup")
		http.Error(w, "Unable to prepare backup download", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, backupName))
	w.Header().Set("Content-Type", "application/zip")
	if _, err := io.Copy(w, tmpFile); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to send database backup")
	}
}

// PerformRestore validates and restores a ZIP backup.
func PerformRestore(w http.ResponseWriter, r *http.Request) {
	configPath := config.GetConfig().ConfigPath

	if r.Method != http.MethodPost {
		http.Error(w, "Use POST to upload backup file", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "File too large or invalid upload", http.StatusBadRequest)
		return
	}

	if r.MultipartForm != nil {
		defer func() {
			if err := r.MultipartForm.RemoveAll(); err != nil {
				logger.Log(logger.Fields{"error": err}).Warn("Unable to remove multipart temporary files")
			}
		}()
	}

	file, handler, err := r.FormFile("backupFile")
	if err != nil {
		http.Error(w,
			fmt.Sprintf("%s - %s", "Failed to read uploaded file", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	defer func(file multipart.File) {
		if err := file.Close(); err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to close uploaded backup")
		}
	}(file)

	if handler.Size > maxUploadSize {
		http.Error(w, "File too large or invalid upload", http.StatusBadRequest)
		return
	}

	out, err := os.CreateTemp("", "openlinkhub-restore-*.zip")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to create temporary restore archive")
		http.Error(w, "Unable to prepare restore", http.StatusInternalServerError)
		return
	}

	tmpZip := out.Name()

	defer func() {
		if err := os.Remove(tmpZip); err != nil && !os.IsNotExist(err) {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to remove temporary restore archive")
		}
	}()

	written, copyErr := io.Copy(
		out,
		io.LimitReader(file, maxUploadSize+1),
	)

	closeErr := out.Close()

	if copyErr != nil {
		logger.Log(logger.Fields{"error": copyErr}).Error("Unable to save uploaded backup")
		http.Error(w, "Unable to save uploaded backup", http.StatusInternalServerError)
		return
	}

	if closeErr != nil {
		logger.Log(logger.Fields{"error": closeErr}).Error("Unable to close temporary restore archive")
		http.Error(w, "Unable to save uploaded backup", http.StatusInternalServerError)
		return
	}

	if written > maxUploadSize {
		http.Error(w, "File too large or invalid upload", http.StatusBadRequest)
		return
	}

	if err := verifyZipIntegrity(tmpZip); err != nil {
		http.Error(w,
			fmt.Sprintf("%s - %s", "Backup verification failed",
				err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	stagingDir, err := os.MkdirTemp(configPath, ".openlinkhub-restore-*")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to create restore staging directory")
		http.Error(w, "Restore failed - unable to create staging directory", http.StatusInternalServerError)
		return
	}

	defer func() {
		if err := os.RemoveAll(stagingDir); err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to remove restore staging directory")
		}
	}()

	if err := unzipFile(tmpZip, stagingDir); err != nil {
		http.Error(w, fmt.Sprintf("%s - %s", "Restore failed", err.Error()), http.StatusBadRequest)
		return
	}

	if err := performRestore(stagingDir, configPath); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to apply restored database backup")
		http.Error(w, fmt.Sprintf("%s - %s", "Restore failed", err.Error()), http.StatusInternalServerError)
		return
	}

	if _, err := fmt.Fprintln(w, "Restore completed successfully"); err != nil {
		logger.Log(logger.Fields{"error": err}).Warn("Unable to write database restore response")
	}
}

// hashAndZipFolder zips folder and feeds canonical path, size, and data to hash.
func hashAndZipFolder(src string, archive *zip.Writer, hasher io.Writer) error {
	return filepath.Walk(
		src,
		func(filePath string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlinks are not supported in backup: %s", filePath)
			}

			if !info.IsDir() && !info.Mode().IsRegular() {
				return fmt.Errorf("unsupported file type in backup: %s", filePath)
			}

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}

			rel, err := filepath.Rel(filepath.Dir(src), filePath)
			if err != nil {
				return err
			}

			// ZIP paths always use '/'.
			header.Name = filepath.ToSlash(rel)

			if info.IsDir() {
				header.Name += "/"
			} else {
				header.Method = zip.Deflate
			}

			writer, err := archive.CreateHeader(header)
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if err := writeHashEntryPrefix(hasher,
				strings.TrimSuffix(header.Name, "/"),
				uint64(info.Size()),
			); err != nil {
				return err
			}

			f, err := os.Open(filePath)
			if err != nil {
				return err
			}

			_, copyErr := io.Copy(io.MultiWriter(writer, hasher), f)
			closeErr := f.Close()

			if copyErr != nil {
				return copyErr
			}

			if closeErr != nil {
				return closeErr
			}

			return nil
		},
	)
}

// hashAndZipFile adds single file to ZIP and hash.
func hashAndZipFile(filePath string, archive *zip.Writer, hasher io.Writer) error {
	info, err := os.Lstat(filePath)
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlinks are not supported in backup: %s", filePath)
	}

	if !info.Mode().IsRegular() {
		if info.IsDir() {
			return fmt.Errorf("%s is a directory", filePath)
		}
		return fmt.Errorf("unsupported file type in backup: %s", filePath)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Method = zip.Deflate
	header.Name = filepath.ToSlash(filepath.Base(filePath))

	writer, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}

	if err := writeHashEntryPrefix(hasher, header.Name, uint64(info.Size())); err != nil {
		return err
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(io.MultiWriter(writer, hasher), f)
	closeErr := f.Close()

	if copyErr != nil {
		return copyErr
	}

	return closeErr
}

// verifyZipIntegrity recalculates hash and compares it to stored _hash.txt.
func verifyZipIntegrity(zipPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}

	defer func() {
		if err := r.Close(); err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to close backup archive")
		}
	}()

	if len(r.File) == 0 {
		return fmt.Errorf("backup archive is empty")
	}

	if len(r.File) > maxArchiveFiles {
		return fmt.Errorf("backup contains too many entries")
	}

	hasher := sha256.New()

	seen := make(map[string]struct{}, len(r.File))

	var expectedHash string
	var totalUncompressed uint64
	var actualUncompressed uint64

	hasConfig := false
	hasDatabase := false

	for _, f := range r.File {
		if strings.Contains(f.Name, "..") {
			return fmt.Errorf("illegal path: %s", f.Name)
		}

		cleanName, err := validateArchiveEntry(f)
		if err != nil {
			return err
		}

		if _, exists := seen[cleanName]; exists {
			return fmt.Errorf("duplicate archive entry: %s", f.Name)
		}

		seen[cleanName] = struct{}{}

		if cleanName == hashFileName {
			if f.FileInfo().IsDir() {
				return fmt.Errorf("%s must be a file", hashFileName)
			}

			if f.UncompressedSize64 > maxHashFileSize {
				return fmt.Errorf("invalid %s", hashFileName)
			}

			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("unable to open %s: %w", hashFileName, err)
			}

			data, readErr := io.ReadAll(io.LimitReader(rc, maxHashFileSize+1))

			closeErr := rc.Close()

			if readErr != nil {
				return fmt.Errorf("unable to read %s: %w", hashFileName, readErr)
			}

			if closeErr != nil {
				return fmt.Errorf("unable to close %s: %w", hashFileName, closeErr)
			}

			if len(data) > maxHashFileSize {
				return fmt.Errorf("invalid %s", hashFileName)
			}

			expectedHash = strings.TrimSpace(string(data))
			continue
		}

		if cleanName == "config.json" {
			hasConfig = true
		}

		if cleanName == "database" ||
			strings.HasPrefix(cleanName, "database/") {
			hasDatabase = true
		}

		if f.FileInfo().IsDir() {
			continue
		}

		if f.UncompressedSize64 > maxUncompressedSize-totalUncompressed {
			return fmt.Errorf("backup exceeds maximum uncompressed size")
		}

		totalUncompressed += f.UncompressedSize64
		if err := writeHashEntryPrefix(
			hasher,
			cleanName,
			f.UncompressedSize64,
		); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		remaining := uint64(maxUncompressedSize) - actualUncompressed
		n, copyErr := io.Copy(hasher, io.LimitReader(rc, int64(remaining)+1))
		closeErr := rc.Close()

		if copyErr != nil {
			return copyErr
		}

		if closeErr != nil {
			return closeErr
		}

		if uint64(n) > remaining {
			return fmt.Errorf("backup exceeds maximum uncompressed size")
		}

		if uint64(n) != f.UncompressedSize64 {
			return fmt.Errorf("unexpected uncompressed size for %s", cleanName)
		}

		actualUncompressed += uint64(n)
	}

	if !hasConfig {
		return fmt.Errorf("missing config.json in archive")
	}

	if !hasDatabase {
		return fmt.Errorf("missing database directory in archive")
	}

	if expectedHash == "" {
		return fmt.Errorf("missing %s in archive", hashFileName)
	}

	expected, err := hex.DecodeString(expectedHash)
	if err != nil || len(expected) != sha256.Size {
		return fmt.Errorf("invalid %s", hashFileName)
	}

	actual := hasher.Sum(nil)
	if subtle.ConstantTimeCompare(expected, actual) != 1 {
		return fmt.Errorf("hash mismatch")
	}

	return nil
}

// unzipFile extracts all files
func unzipFile(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}

	defer func() {
		if err := r.Close(); err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to close backup archive")
		}
	}()

	destAbs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}

	var totalUncompressed uint64

	for _, f := range r.File {
		if strings.Contains(f.Name, "..") {
			return fmt.Errorf("illegal path: %s", f.Name)
		}
		
		cleanName, err := validateArchiveEntry(f)
		if err != nil {
			return err
		}

		if cleanName == hashFileName {
			continue
		}

		targetPath := filepath.Join(
			destAbs,
			filepath.FromSlash(cleanName),
		)

		targetAbs, err := filepath.Abs(targetPath)
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(destAbs, targetAbs)
		if err != nil {
			return err
		}

		if rel == ".." ||
			strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
			return fmt.Errorf("illegal path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			perm := f.Mode().Perm()
			if perm == 0 {
				perm = 0o755
			}

			if err := os.MkdirAll(targetAbs, perm); err != nil {
				return fmt.Errorf("unable to create directory %s: %w", cleanName, err)
			}

			continue
		}

		if f.UncompressedSize64 > maxUncompressedSize-totalUncompressed {
			return fmt.Errorf("backup exceeds maximum uncompressed size")
		}

		if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
			return fmt.Errorf("unable to create directory for %s: %w", cleanName, err)
		}

		perm := f.Mode().Perm()
		if perm == 0 {
			perm = 0o600
		}

		outFile, err := os.OpenFile(targetAbs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
		if err != nil {
			return fmt.Errorf("unable to create %s: %w", cleanName, err)
		}

		rc, err := f.Open()
		if err != nil {
			openErr := err

			if closeErr := outFile.Close(); closeErr != nil {
				logger.Log(logger.Fields{"error": closeErr}).Warn("Unable to close restored file")
			}

			return openErr
		}

		remaining := uint64(maxUncompressedSize) - totalUncompressed

		n, copyErr := io.Copy(outFile, io.LimitReader(rc, int64(remaining)+1))
		outCloseErr := outFile.Close()
		rcCloseErr := rc.Close()

		if copyErr != nil {
			return copyErr
		}

		if outCloseErr != nil {
			return outCloseErr
		}

		if rcCloseErr != nil {
			return rcCloseErr
		}

		if uint64(n) > remaining {
			return fmt.Errorf("backup exceeds maximum uncompressed size")
		}

		if uint64(n) != f.UncompressedSize64 {
			return fmt.Errorf("unexpected uncompressed size for %s", cleanName)
		}

		totalUncompressed += uint64(n)
	}

	return nil
}

// writeHashEntryPrefix writes the canonical file path and size into the hash.
func writeHashEntryPrefix(hasher io.Writer, name string, size uint64) error {
	canonicalName := pathpkg.Clean(
		filepath.ToSlash(name),
	)

	if canonicalName == "." || canonicalName == "" {
		return fmt.Errorf("invalid path for hash")
	}

	if _, err := io.WriteString(hasher, canonicalName); err != nil {
		return err
	}

	if _, err := io.WriteString(hasher, "\x00"); err != nil {
		return err
	}

	if _, err := io.WriteString(hasher, strconv.FormatUint(size, 10)); err != nil {
		return err
	}

	_, err := io.WriteString(hasher, "\x00")

	return err
}

// validateArchiveEntry validates the structure and type of ZIP entry.
func validateArchiveEntry(f *zip.File) (string, error) {
	if f == nil {
		return "", fmt.Errorf("invalid archive entry")
	}

	if f.Name == "" || strings.ContainsRune(f.Name, '\x00') || strings.Contains(f.Name, "\\") {
		return "", fmt.Errorf("invalid archive path: %q", f.Name)
	}

	rawName := strings.TrimSuffix(f.Name, "/")

	cleanName := pathpkg.Clean(rawName)
	if rawName != cleanName || cleanName == "." || pathpkg.IsAbs(cleanName) || cleanName == ".." ||
		strings.HasPrefix(cleanName, "../") {
		return "", fmt.Errorf("illegal path: %s", f.Name)
	}

	mode := f.Mode()

	if mode&os.ModeSymlink != 0 || (!mode.IsRegular() && !mode.IsDir()) {
		return "", fmt.Errorf("unsupported file type in archive: %s", f.Name)
	}

	allowed := cleanName == hashFileName || cleanName == "config.json" || cleanName == "database" ||
		strings.HasPrefix(cleanName, "database/")

	if !allowed {
		return "", fmt.Errorf("unexpected file in backup: %s", f.Name)
	}

	if cleanName == "config.json" && mode.IsDir() {
		return "", fmt.Errorf("config.json must be a file")
	}

	if cleanName == "database" && !mode.IsDir() {
		return "", fmt.Errorf("database must be a directory")
	}

	return cleanName, nil
}

// applyRestore replaces the current database and config.json
func performRestore(stagingDir, dest string) error {
	stagedDatabase := filepath.Join(stagingDir, "database")
	stagedConfig := filepath.Join(stagingDir, "config.json")
	destDatabase := filepath.Join(dest, "database")
	destConfig := filepath.Join(dest, "config.json")
	oldDatabase := filepath.Join(stagingDir, ".old-database")
	oldConfig := filepath.Join(stagingDir, ".old-config.json")

	// Verify staged database.
	dbInfo, err := os.Lstat(stagedDatabase)
	if err != nil {
		return fmt.Errorf("restored database is missing: %w", err)
	}

	if !dbInfo.IsDir() || dbInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("restored database is invalid")
	}

	// Verify staged config.json.
	configInfo, err := os.Lstat(stagedConfig)
	if err != nil {
		return fmt.Errorf("restored config.json is missing: %w", err)
	}

	if !configInfo.Mode().IsRegular() || configInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("restored config.json is invalid")
	}

	// Move existing database into the staging directory.
	hadDatabase, err := moveExisting(destDatabase, oldDatabase)
	if err != nil {
		return fmt.Errorf("unable to prepare current database for restore: %w", err)
	}

	// Move existing config into the staging directory.
	hadConfig, err := moveExisting(destConfig, oldConfig)
	if err != nil {
		if hadDatabase {
			_ = os.Rename(oldDatabase, destDatabase)
		}
		return fmt.Errorf("unable to prepare current config.json for restore: %w", err)
	}

	dbInstalled := false
	configInstalled := false

	rollback := func(originalErr error) error {
		var rollbackErrors []string

		// Remove newly installed config.json.
		if configInstalled {
			if err := os.Remove(destConfig); err != nil && !os.IsNotExist(err) {
				rollbackErrors = append(rollbackErrors, "remove restored config.json: "+err.Error())
			}
		}

		// Remove newly installed database.
		if dbInstalled {
			if err := os.RemoveAll(destDatabase); err != nil && !os.IsNotExist(err) {
				rollbackErrors = append(rollbackErrors, "remove restored database: "+err.Error())
			}
		}

		// Restore original config.
		if hadConfig {
			if err := os.Rename(oldConfig, destConfig); err != nil {
				rollbackErrors = append(rollbackErrors, "restore original config.json: "+err.Error())
			}
		}

		// Restore original database.
		if hadDatabase {
			if err := os.Rename(oldDatabase, destDatabase); err != nil {
				rollbackErrors = append(rollbackErrors, "restore original database: "+err.Error())
			}
		}

		if len(rollbackErrors) > 0 {
			return fmt.Errorf("%w; rollback failed: %s", originalErr, strings.Join(rollbackErrors, "; "))
		}

		return originalErr
	}

	if err := os.Rename(stagedDatabase, destDatabase); err != nil {
		return rollback(
			fmt.Errorf("unable to install restored database: %w", err),
		)
	}

	dbInstalled = true

	if err := os.Rename(stagedConfig, destConfig); err != nil {
		return rollback(fmt.Errorf("unable to install restored config.json: %w", err))
	}

	configInstalled = true

	if hadDatabase {
		if err := os.RemoveAll(oldDatabase); err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to remove previous database after restore")
		}
	}

	if hadConfig {
		if err := os.Remove(oldConfig); err != nil && !os.IsNotExist(err) {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to remove previous config.json after restore")
		}
	}

	return nil
}

// moveExisting moves an existing file/directory to a temporary location
func moveExisting(src, dst string) (bool, error) {
	_, err := os.Lstat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	if err := os.Rename(src, dst); err != nil {
		return false, err
	}

	return true, nil
}
