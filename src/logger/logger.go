package logger

// Package: logger
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Fields map[string]interface{}
type Entry struct {
	fields Fields
}

var (
	logger   *log.Logger
	logLevel common.LogLevel
)

// Init initializes the logger and sets the log level
func Init() {
	logLevel = levelFromString(config.GetConfig().LogLevel)

	logFilename := config.GetConfig().LogFile
	if logFilename == "" {
		logFilename = config.GetConfig().ConfigPath + "/stdout.log"
	}

	if _, err := os.Stat(logFilename); err == nil {
		if err = archiveLog(logFilename); err != nil {
			fmt.Println("failed to archive log file", err)
		}
	}

	var output *os.File
	var err error

	if logFilename == "-" {
		output = os.Stderr
	} else {
		output, err = os.OpenFile(logFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			if _, e := fmt.Fprintf(os.Stderr, "Failed to log to file: %v, using stderr\n", err); e != nil {
				panic("Unable to write to stderr: " + e.Error())
			}
			output = os.Stderr
		}
	}

	logger = log.New(output, "", 0)
}

// Log is new log entry with fields
func Log(fields Fields) *Entry {
	return &Entry{fields: fields}
}

// archiveLog will compress an existing log file into .tar.gz and remove the original
func archiveLog(logFilename string) error {
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	archiveName := fmt.Sprintf("%s.%s.tar.gz", logFilename, timestamp)

	logFile, err := os.Open(logFilename)
	if err != nil {
		return err
	}
	defer func(logFile *os.File) {
		err = logFile.Close()
		if err != nil {
			fmt.Println("Unable to close log file", err)
		}
	}(logFile)

	outFile, err := os.Create(archiveName)
	if err != nil {
		return err
	}
	defer func(outFile *os.File) {
		err = outFile.Close()
		if err != nil {
			fmt.Println("Unable to close log file", err)
		}
	}(outFile)

	gw := gzip.NewWriter(outFile)
	defer func(gw *gzip.Writer) {
		err = gw.Close()
		if err != nil {
			fmt.Println("Unable to close log file", err)
		}
	}(gw)

	tw := tar.NewWriter(gw)
	defer func(tw *tar.Writer) {
		err = tw.Close()
		if err != nil {
			fmt.Println("Unable to close log file", err)
		}
	}(tw)

	info, err := logFile.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = filepath.Base(logFilename)

	if err = tw.WriteHeader(header); err != nil {
		return err
	}

	if _, err = io.Copy(tw, logFile); err != nil {
		return err
	}

	return os.Remove(logFilename)
}

// levelFromString will convert string level to integer
func levelFromString(level string) common.LogLevel {
	switch strings.ToLower(level) {
	case "info":
		return common.LogInfo
	case "warn":
		return common.LogWarn
	case "error":
		return common.LogError
	case "fatal":
		return common.LogFatal
	case "silent":
		return common.LogSilent
	default:
		return common.LogInfo
	}
}

// normalizeFields will normalize JSON fields
func normalizeFields(fields Fields) Fields {
	out := make(Fields, len(fields))

	for k, v := range fields {
		switch val := v.(type) {
		case error:
			if val != nil {
				out[k] = val.Error()
			} else {
				out[k] = nil
			}
		default:
			out[k] = v
		}
	}

	return out
}

// logWithLevel writes the log entry if level >= configured logLevel
func (e *Entry) logWithLevel(level, msg string) {
	if levelFromString(level) < logLevel {
		return
	}

	e.fields["timestamp"] = time.Now().Format(time.RFC3339)
	e.fields["level"] = level
	e.fields["message"] = msg

	normalized := normalizeFields(e.fields)
	data, err := json.Marshal(normalized)
	if err != nil {
		if _, e := fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err); e != nil {
			panic("Unable to write to stderr: " + e.Error())
		}
		return
	}

	logger.Println(string(data))
}

// Error will log error
func (e *Entry) Error(msg string) {
	e.logWithLevel("error", msg)
}

// Fatal will log fatal and panic
func (e *Entry) Fatal(msg string) {
	e.logWithLevel("fatal", msg)
	panic(msg)
}

// Info will log info
func (e *Entry) Info(msg string) {
	e.logWithLevel("info", msg)
}

// Warn will log warning
func (e *Entry) Warn(msg string) {
	e.logWithLevel("warn", msg)
}

// Errorf will log formatted error
func (e *Entry) Errorf(format string, args ...interface{}) {
	e.logWithLevel("error", fmt.Sprintf(format, args...))
}

// Warnf will log formatted warning
func (e *Entry) Warnf(format string, args ...interface{}) {
	e.logWithLevel("warn", fmt.Sprintf(format, args...))
}

// Infof will log formatted info
func (e *Entry) Infof(format string, args ...interface{}) {
	e.logWithLevel("info", fmt.Sprintf(format, args...))
}
