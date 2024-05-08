package logger

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

type Fields = log.Fields

// LogToConsole will log data into a console
func LogToConsole(message string) {
	msg := fmt.Sprintf(
		"[%s] %s",
		time.Now().Format("2006-01-02 15:04:05"),
		message,
	)
	fmt.Println(msg)
}

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
