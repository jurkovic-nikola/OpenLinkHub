package keyboards

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"encoding/json"
	"os"
)

var (
	pwd       = ""
	location  = ""
	keyboards = map[string]Keyboard{}
)

type Keyboard struct {
	Key    string        `json:"key"`
	Device string        `json:"device"`
	Layout string        `json:"layout"`
	Rows   int           `json:"rows"`
	Row    map[int]Row   `json:"row"`
	Zones  map[int]Zones `json:"zones"`
}

type Zones struct {
	Color rgb.Color `json:"color"`
}

type Row struct {
	Keys map[int]Key `json:"keys"`
}

type Key struct {
	KeyName     string    `json:"keyName"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Left        int       `json:"left"`
	Top         int       `json:"top"`
	PacketIndex []int     `json:"packetIndex"`
	Color       rgb.Color `json:"color"`
	Zone        int       `json:"zone"`
}

// Init will load and initialize keyboard data
func Init() {
	pwd = config.GetConfig().ConfigPath
	location = pwd + "/database/keyboard/"

	files, err := os.ReadDir(location)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Fatal("Unable to read content of a folder")
	}

	for _, fileInfo := range files {
		if fileInfo.IsDir() {
			continue // Exclude folders if any
		}

		// Define a full path of filename
		pullPath := location + fileInfo.Name()

		// Check if filename has .json extension
		if !common.IsValidExtension(pullPath, ".json") {
			continue
		}

		file, fe := os.Open(pullPath)
		if fe != nil {
			logger.Log(logger.Fields{"error": fe, "location": pullPath}).Error("Unable to open keyboard file")
			continue
		}

		// Decode and create profile
		var keyboard Keyboard

		reader := json.NewDecoder(file)
		if err = reader.Decode(&keyboard); fe != nil {
			logger.Log(logger.Fields{"error": fe, "location": pullPath}).Error("Unable to decode keyboard file")
			continue
		}

		keyboards[keyboard.Key] = keyboard
		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": fe, "location": pullPath}).Error("Unable to close keyboard file")
		}
	}
}

// GetKeyboard will return Keyboard struct for a given keyboard type
func GetKeyboard(key string) *Keyboard {
	if keyboard, ok := keyboards[key]; ok {
		return &keyboard
	}
	return nil
}
