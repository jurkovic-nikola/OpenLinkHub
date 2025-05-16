package logger

import (
	"OpenLinkHub/src/common"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"OpenLinkHub/src/config"
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
	default:
		return common.LogInfo
	}
}

// logWithLevel writes the log entry if level >= configured logLevel
func (e *Entry) logWithLevel(level, msg string) {
	if levelFromString(level) < logLevel {
		return
	}

	e.fields["timestamp"] = time.Now().Format(time.RFC3339)
	e.fields["level"] = level
	e.fields["message"] = msg

	data, err := json.Marshal(e.fields)
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
