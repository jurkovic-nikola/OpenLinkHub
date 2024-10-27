package config

import (
	"OpenLinkHub/src/common"
	"encoding/json"
	"os"
)

type Configuration struct {
	Debug          bool   `json:"debug"`
	ListenPort     int    `json:"listenPort"`
	ListenAddress  string `json:"listenAddress"`
	CPUSensorChip  string `json:"cpuSensorChip"`
	Manual         bool   `json:"manual"`
	Frontend       bool   `json:"frontend"`
	RefreshOnStart bool   `json:"refreshOnStart"`
	Metrics        bool   `json:"metrics"`
	DbusMonitor    bool   `json:"dbusMonitor"`
	Memory         bool   `json:"memory"`
	MemorySmBus    string `json:"memorySmBus"`
	MemoryType     int    `json:"memoryType"`
	ConfigPath     string
}

var configuration Configuration

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
	cfg := pwd + "/config.json"
	f, err := os.Open(cfg)
	if err != nil {
		panic(err.Error())
	}
	if err = json.NewDecoder(f).Decode(&configuration); err != nil {
		panic(err.Error())
	}
	configuration.ConfigPath = configPath
}

// GetConfig will return structs.Configuration struct
func GetConfig() Configuration {
	return configuration
}
