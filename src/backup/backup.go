package backup

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxUploadSize = 5 * 1024 * 1024 // 5 MB
	hashFileName  = "_hash.txt"
)

// PerformBackup creates a ZIP with SHA-256 integrity hash
func PerformBackup(w http.ResponseWriter, _ *http.Request) {
	cfg := config.GetConfig()
	srcFolder := filepath.Join(cfg.ConfigPath, "database")
	extraFile := filepath.Join(cfg.ConfigPath, "config.json")
	backupName := "backup_" + time.Now().Format("2006-01-02-15-04-05") + ".zip"

	tmpFile, err := os.CreateTemp("", backupName)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Warn("Unable to perform database backup")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to remove temp database backup")
		}
	}(tmpFile.Name())
	defer func(tmpFile *os.File) {
		err := tmpFile.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to close temp database backup")
		}
	}(tmpFile)

	archive := zip.NewWriter(tmpFile)
	hasher := sha256.New()

	// Add database folder
	if err := hashAndZipFolder(srcFolder, archive, hasher); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add config.json
	if err := hashAndZipFile(extraFile, archive, hasher); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write hash file
	sum := hex.EncodeToString(hasher.Sum(nil))
	hf, err := archive.Create(hashFileName)
	if err != nil {
		http.Error(w, "Unable to create hash file in archive", http.StatusInternalServerError)
		return
	}
	if _, err := hf.Write([]byte(sum)); err != nil {
		http.Error(w, "Unable to write hash file", http.StatusInternalServerError)
		return
	}

	if err := archive.Close(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+backupName)
	w.Header().Set("Content-Type", "application/zip")

	_, err = tmpFile.Seek(0, 0)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to perform database backup")
		return
	}
	_, err = io.Copy(w, tmpFile)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to perform database backup")
		return
	}
}

// PerformRestore validates and restores a ZIP backup
func PerformRestore(w http.ResponseWriter, r *http.Request) {
	path := config.GetConfig().ConfigPath
	if r.Method != http.MethodPost {
		http.Error(w, "Use POST to upload backup file", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "File too large or invalid upload", http.StatusBadRequest)
		return
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
		err := file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to close temp database backup")
		}
	}(file)

	tmpZip := filepath.Join(os.TempDir(), handler.Filename)
	out, err := os.Create(tmpZip)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to remove temp database backup")
		}
	}(tmpZip)
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to close temp database backup")
		}
	}(out)

	if _, err := io.Copy(out, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := verifyZipIntegrity(tmpZip); err != nil {
		http.Error(w,
			fmt.Sprintf("%s - %s", "Backup verification failed", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	if err := unzipFile(tmpZip, path); err != nil {
		http.Error(w,
			fmt.Sprintf("%s - %s", "Restore failed", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	_, err = fmt.Fprintln(w, "Restore completed successfully")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Warn("Unable to perform database restore")
		return
	}
}

// hashAndZipFolder zips folder and feeds data to hash
func hashAndZipFolder(src string, archive *zip.Writer, hasher io.Writer) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name, err = filepath.Rel(filepath.Dir(src), path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		w, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func(f *os.File) {
				err := f.Close()
				if err != nil {
					logger.Log(logger.Fields{"error": err}).Warn("Unable to close temp database backup")
				}
			}(f)
			mw := io.MultiWriter(w, hasher)
			if _, err := io.Copy(mw, f); err != nil {
				return err
			}
		}
		return nil
	})
}

// hashAndZipFile adds single file to ZIP and hash
func hashAndZipFile(filePath string, archive *zip.Writer, hasher io.Writer) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", filePath)
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

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to close temp database backup")
		}
	}(f)

	mw := io.MultiWriter(writer, hasher)
	_, err = io.Copy(mw, f)
	return err
}

// verifyZipIntegrity recalculates hash and compares it to stored _hash.txt
func verifyZipIntegrity(zipPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer func(r *zip.ReadCloser) {
		err := r.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to close temp database backup")
		}
	}(r)

	hasher := sha256.New()
	var expectedHash string

	for _, f := range r.File {
		if f.Name == hashFileName {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			err := rc.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to close backup hash file")
				return err
			}
			expectedHash = strings.TrimSpace(string(data))
			continue
		}
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		if _, err := io.Copy(hasher, rc); err != nil {
			err := rc.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Warn("Unable to close backup file")
				return err
			}
			return err
		}
		err = rc.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to close backup file")
			return err
		}
	}

	if expectedHash == "" {
		return fmt.Errorf("missing %s in archive", hashFileName)
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if expectedHash != actualHash {
		return fmt.Errorf("hash mismatch")
	}
	return nil
}

// unzipFile extracts all files (skipping _hash.txt)
func unzipFile(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func(r *zip.ReadCloser) {
		err := r.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close backup file")
		}
	}(r)

	for _, f := range r.File {
		if f.Name == hashFileName {
			continue
		}
		path := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			err := os.MkdirAll(path, os.ModePerm)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to create directory")
				return err
			}
			continue
		}
		err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to create directory")
			return err
		}

		outFile, err := os.Create(path)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			err := outFile.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Warn("Unable to close backup file")
				return err
			}
			return err
		}
		if _, err := io.Copy(outFile, rc); err != nil {
			err := outFile.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Warn("Unable to close backup file")
				return err
			}
			err = rc.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Warn("Unable to close backup file")
				return err
			}
			return err
		}
		err = outFile.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to close backup file")
			return err
		}
		err = rc.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to close backup file")
			return err
		}
	}
	return nil
}
