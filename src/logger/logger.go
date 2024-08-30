package logger

import (
	log "github.com/sirupsen/logrus"
	"os"
)

type Fields = log.Fields

// Init will initialize new instance of logger
func Init() {
	log.SetFormatter(&log.JSONFormatter{})
	file, err := os.OpenFile("stdout.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to log to file, using default stderr")
	}
}

// Log will save entries into a log file
func Log(m log.Fields) *log.Entry {
	return log.WithFields(m)
}
