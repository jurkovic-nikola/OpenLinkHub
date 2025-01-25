package logger

import (
	"OpenLinkHub/src/config"
	log "github.com/sirupsen/logrus"
	"os"
)

type Fields = log.Fields

// Init will initialize new instance of logger
func Init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(config.GetConfig().LogLevel)

	logFilename := config.GetConfig().LogFile
	if logFilename == "" {
		logFilename = config.GetConfig().ConfigPath + "/stdout.log"
	}
	if logFilename != "-" {
		file, err := os.OpenFile(logFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(file)
		} else {
			log.WithFields(log.Fields{"error": err}).Warn("Failed to log to file, using default stderr")
		}
	}
}

// Log will save entries into a log file
func Log(m log.Fields) *log.Entry {
	return log.WithFields(m)
}
