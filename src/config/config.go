package config

import (
	"OpenLinkHub/src/common"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"os"
)

type Configuration struct {
	Debug           bool      `json:"debug"`
	ListenPort      int       `json:"listenPort"`
	ListenAddress   string    `json:"listenAddress"`
	CPUSensorChip   string    `json:"cpuSensorChip"`
	Manual          bool      `json:"manual"`
	Frontend        bool      `json:"frontend"`
	Metrics         bool      `json:"metrics"`
	Memory          bool      `json:"memory"`
	MemorySmBus     string    `json:"memorySmBus"`
	MemoryType      int       `json:"memoryType"`
	Exclude         []uint16  `json:"exclude"`
	DecodeMemorySku bool      `json:"decodeMemorySku"`
	MemorySku       string    `json:"memorySku"`
	ConfigPath      string    `json:",omitempty"`
	ResumeDelay     int       `json:"resumeDelay"`
	LogFile         string    `json:"logFile,omitempty"`
	LogLevel        log.Level `json:"logLevel"`
}

var (
	location      = ""
	configuration Configuration
	upgrade       = map[string]any{
		"decodeMemorySku": true,
		"memorySku":       "",
		"resumeDelay":     15000,
		"logLevel":        log.InfoLevel,
	}
)

// Init will initialize a new config object
func Init() {
	var configPath = ""

	pwd, _ := os.Getwd()
	isAtomic := common.FileExists(pwd + "/atomic")
	if isAtomic {
		pwd = "/etc/OpenLinkHub"
		configPath = "/etc/OpenLinkHub"
	} else {
		configPath = pwd
	}
	location = pwd + "/config.json"

	// Create or upgrade
	upgradeFile(location)

	f, err := os.Open(location)
	if err != nil {
		panic(err.Error())
	}
	if err = json.NewDecoder(f).Decode(&configuration); err != nil {
		panic(err.Error())
	}
	configuration.ConfigPath = configPath
}

// upgradeFile will create or upgrade config file
func upgradeFile(cfg string) {
	if !common.FileExists(cfg) {
		value := &Configuration{
			Debug:           false,
			ListenPort:      27003,
			ListenAddress:   "127.0.0.1",
			CPUSensorChip:   "",
			Manual:          false,
			Frontend:        true,
			Metrics:         false,
			Memory:          false,
			MemorySmBus:     "i2c-0",
			MemoryType:      4,
			Exclude:         make([]uint16, 0),
			DecodeMemorySku: true,
			MemorySku:       "",
			ResumeDelay:     15000,
		}
		saveConfigSettings(value)
	} else {
		save := false
		var data map[string]interface{}
		file, err := os.Open(location)
		defer func(file *os.File) {
			err = file.Close()
			if err != nil {
				panic(err.Error())
			}
		}(file)

		if err != nil {
			panic(err.Error())
		}
		if err = json.NewDecoder(file).Decode(&data); err != nil {
			panic(err.Error())
		}

		// Loop thru upgrade value
		for key, value := range upgrade {
			if _, ok := data[key]; !ok {
				data[key] = value
				save = true
			}
		}
		if save {
			saveConfigSettings(data)
		}
	}
}

// SaveConfigSettings will save dashboard settings
func saveConfigSettings(data any) {
	// Convert to JSON
	buffer, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		panic(err.Error())
	}

	// Create profile filename
	file, err := os.Create(location)
	if err != nil {
		panic(err.Error())
	}

	// Write JSON buffer to file
	_, err = file.Write(buffer)
	if err != nil {
		panic(err.Error())
	}

	// Close file
	err = file.Close()
	if err != nil {
		panic(err.Error())
	}
}

// GetConfig will return structs.Configuration struct
func GetConfig() Configuration {
	return configuration
}
