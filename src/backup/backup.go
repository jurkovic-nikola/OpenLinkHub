package backup

// Package: backup
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"LumenForge/src/config"
	"LumenForge/src/language"
	"LumenForge/src/logger"
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	maxUploadSize = 5 * 1024 * 1024 // 5 MiB
	hashFileName  = "_hash.txt"
)

var backupRestoreMutex sync.Mutex

// PerformBackup creates a ZIP with a SHA-256 corruption-detection hash.
func PerformBackup(w http.ResponseWriter, _ *http.Request) {
	backupRestoreMutex.Lock()
	defer backupRestoreMutex.Unlock()

	paths := config.GetPaths()
	srcFolder := paths.MutableDatabaseRoot
	extraFile := paths.BackupConfigurationFile
	backupName := "backup_" + time.Now().Format("2006-01-02-15-04-05") + ".zip"

	tmpFile, err := os.CreateTemp("", "lumenforge-backup-*.zip")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Warn("Unable to perform database backup")
		http.Error(w, "Unable to create backup", http.StatusInternalServerError)
		return
	}
	tmpPath := tmpFile.Name()
	defer removeTemporaryFile(tmpPath, "backup")

	archive := zip.NewWriter(tmpFile)
	hasher := sha256.New()

	if err = hashAndZipFolder(srcFolder, archive, hasher); err == nil {
		err = hashAndZipFile(extraFile, archive, hasher)
	}
	if err == nil {
		for _, runtimeFile := range []string{"dashboard.json", "display.json"} {
			filePath := filepath.Join(paths.BackupDataRoot, runtimeFile)
			if _, statErr := os.Stat(filePath); statErr == nil {
				if err = hashAndZipFile(filePath, archive, hasher); err != nil {
					break
				}
			} else if !os.IsNotExist(statErr) {
				err = statErr
				break
			}
		}
	}
	if err == nil {
		var hashWriter io.Writer
		hashWriter, err = archive.Create(hashFileName)
		if err == nil {
			_, err = hashWriter.Write([]byte(hex.EncodeToString(hasher.Sum(nil))))
		}
	}
	err = joinPrimaryError(err, "close backup archive", archive.Close())
	err = joinPrimaryError(err, "close backup temporary file", tmpFile.Close())
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Warn("Unable to perform database backup")
		http.Error(w, "Unable to create backup", http.StatusInternalServerError)
		return
	}

	tmpFile, err = os.Open(tmpPath)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to reopen database backup")
		http.Error(w, "Unable to read backup", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+backupName)
	w.Header().Set("Content-Type", "application/zip")
	_, copyErr := io.Copy(w, tmpFile)
	err = joinPrimaryError(copyErr, "close backup download file", tmpFile.Close())
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to send database backup")
	}
}

// PerformRestore validates, stages, and replaces mutable state from a ZIP backup.
func PerformRestore(w http.ResponseWriter, r *http.Request) {
	backupRestoreMutex.Lock()
	defer backupRestoreMutex.Unlock()

	if r.Method != http.MethodPost {
		http.Error(w, "Use POST to upload backup file", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "File too large or invalid upload", http.StatusBadRequest)
		return
	}
	if r.MultipartForm != nil {
		defer func() {
			if err := r.MultipartForm.RemoveAll(); err != nil {
				logger.Log(logger.Fields{"error": err}).Warn("Unable to remove multipart restore files")
			}
		}()
	}

	upload, _, err := r.FormFile("backupFile")
	if err != nil {
		http.Error(w, "Failed to read uploaded backup", http.StatusBadRequest)
		return
	}
	tmpFile, err := createRestoreUpload()
	if err != nil {
		err = errors.Join(err, wrapOptionalError("close uploaded backup", upload.Close()))
		logger.Log(logger.Fields{"error": err}).Warn("Unable to create restore upload file")
		http.Error(w, "Unable to stage uploaded backup", http.StatusInternalServerError)
		return
	}
	tmpPath := tmpFile.Name()
	defer removeTemporaryFile(tmpPath, "restore upload")

	copyErr := copyMultipartUpload(tmpFile, upload)
	if copyErr != nil {
		logger.Log(logger.Fields{"error": copyErr}).Warn("Unable to stage restore upload")
		http.Error(w, "Unable to stage uploaded backup", http.StatusInternalServerError)
		return
	}

	paths := config.GetPaths()
	if err = restoreBackup(tmpPath, paths.RestoreConfigurationRoot, paths.RestoreDataRoot, defaultRestoreFileOps()); err != nil {
		logRestoreError(err)
		http.Error(w, restorePublicMessage(err), http.StatusBadRequest)
		return
	}

	message := language.GetValue("txtRestoreSuccessRestartRequired")
	if message == "" {
		message = "Restore completed successfully. Restart LumenForge before making further changes."
	}
	if _, err = fmt.Fprintln(w, message); err != nil {
		logger.Log(logger.Fields{"error": err}).Warn("Unable to send database restore response")
	}
}

func createRestoreUpload() (*os.File, error) {
	file, err := os.CreateTemp("", "lumenforge-restore-upload-*.zip")
	if err != nil {
		return nil, err
	}
	if err = file.Chmod(0o600); err != nil {
		closeErr := file.Close()
		removeErr := os.Remove(file.Name())
		return nil, errors.Join(err, closeErr, removeErr)
	}
	return file, nil
}

func copyMultipartUpload(destination *os.File, upload multipart.File) error {
	_, copyErr := io.Copy(destination, upload)
	uploadCloseErr := upload.Close()
	destinationCloseErr := destination.Close()
	return errors.Join(
		copyErr,
		wrapOptionalError("close uploaded backup", uploadCloseErr),
		wrapOptionalError("close restore upload file", destinationCloseErr),
	)
}

func removeTemporaryFile(name, purpose string) {
	if err := os.Remove(name); err != nil && !os.IsNotExist(err) {
		logger.Log(logger.Fields{"error": err, "purpose": purpose}).Warn("Unable to remove temporary backup file")
	}
}

// hashAndZipFolder zips a folder and feeds regular-file data to the hash.
func hashAndZipFolder(src string, archive *zip.Writer, hasher io.Writer) error {
	return filepath.Walk(src, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name, err = filepath.Rel(filepath.Dir(src), filePath)
		if err != nil {
			return err
		}

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

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(io.MultiWriter(writer, hasher), file)
		return joinPrimaryError(copyErr, "close backup source file", file.Close())
	})
}

// hashAndZipFile adds one regular file to a ZIP and hash.
func hashAndZipFile(filePath string, archive *zip.Writer, hasher io.Writer) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", filePath)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Method = zip.Deflate
	header.Name = filepath.Base(filePath)
	writer, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(io.MultiWriter(writer, hasher), file)
	return joinPrimaryError(copyErr, "close backup source file", file.Close())
}

func joinPrimaryError(primary error, closeOperation string, closeErr error) error {
	return errors.Join(primary, wrapOptionalError(closeOperation, closeErr))
}

func wrapOptionalError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
