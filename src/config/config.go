package config

import (
	"OpenLinkHub/src/common"
	"encoding/json"
	"os"
	"slices"
)

type Configuration struct {
	Debug                     bool     `json:"debug"`
	ListenPort                int      `json:"listenPort"`
	ListenAddress             string   `json:"listenAddress"`
	CPUSensorChip             string   `json:"cpuSensorChip"`
	Manual                    bool     `json:"manual"`
	Frontend                  bool     `json:"frontend"`
	Metrics                   bool     `json:"metrics"`
	Memory                    bool     `json:"memory"`
	MemorySmBus               string   `json:"memorySmBus"`
	MemoryType                int      `json:"memoryType"`
	Exclude                   []uint16 `json:"exclude"`
	DecodeMemorySku           bool     `json:"decodeMemorySku"`
	MemorySku                 string   `json:"memorySku"`
	ConfigPath                string   `json:",omitempty"`
	ResumeDelay               int      `json:"resumeDelay"`
	LogFile                   string   `json:"logFile"`
	LogLevel                  string   `json:"logLevel"`
	EnhancementKits           []byte   `json:"enhancementKits"`
	TemperatureOffset         int      `json:"temperatureOffset"`
	AMDGpuIndex               int      `json:"amdGpuIndex"`
	AMDSmiPath                string   `json:"amdsmiPath"`
	CheckDevicePermission     bool     `json:"checkDevicePermission"`
	GraphProfiles             bool     `json:"graphProfiles"`
	CpuTempFile               string   `json:"cpuTempFile"`
	RamTempViaHwmon           bool     `json:"ramTempViaHwmon"`
	NvidiaGpuIndex            []int    `json:"nvidiaGpuIndex"`
	DefaultNvidiaGPU          int      `json:"defaultNvidiaGPU"`
	OpenRGBPort               int      `json:"openRGBPort"`
	EnableOpenRGBTargetServer bool     `json:"enableOpenRGBTargetServer"`
}

var (
	location      = ""
	configuration Configuration
	upgrade       = map[string]any{
		"decodeMemorySku":           true,
		"memorySku":                 "",
		"resumeDelay":               15000,
		"logLevel":                  "info",
		"logFile":                   "",
		"enhancementKits":           make([]byte, 0),
		"temperatureOffset":         0,
		"amdGpuIndex":               0,
		"amdsmiPath":                "",
		"checkDevicePermission":     true,
		"cpuTempFile":               "",
		"graphProfiles":             false,
		"ramTempViaHwmon":           false,
		"nvidiaGpuIndex":            []int{0},
		"defaultNvidiaGPU":          0,
		"openRGBPort":               6743,
		"enableOpenRGBTargetServer": false,
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
			Debug:                     false,
			ListenPort:                27003,
			ListenAddress:             "127.0.0.1",
			CPUSensorChip:             "",
			Manual:                    false,
			Frontend:                  true,
			Metrics:                   false,
			Memory:                    false,
			MemorySmBus:               "i2c-0",
			MemoryType:                5,
			Exclude:                   make([]uint16, 0),
			DecodeMemorySku:           true,
			MemorySku:                 "",
			ResumeDelay:               15000,
			LogLevel:                  "info",
			LogFile:                   "",
			EnhancementKits:           make([]byte, 0),
			TemperatureOffset:         0,
			AMDGpuIndex:               0,
			AMDSmiPath:                "",
			CheckDevicePermission:     true,
			CpuTempFile:               "",
			GraphProfiles:             true,
			RamTempViaHwmon:           false,
			NvidiaGpuIndex:            []int{0},
			DefaultNvidiaGPU:          0,
			OpenRGBPort:               6743,
			EnableOpenRGBTargetServer: false,
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
	buffer, err := json.MarshalIndent(data, "", "  ")
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

// UpdateSupportedDevices will update the Exclude slice based on the enabled flag for each product ID
func UpdateSupportedDevices(productIds map[uint16]bool) uint8 {
	for productId, enabled := range productIds {
		if enabled {
			if i := slices.Index(configuration.Exclude, productId); i != -1 {
				configuration.Exclude = append(configuration.Exclude[:i], configuration.Exclude[i+1:]...)
			}
		} else {
			if !slices.Contains(configuration.Exclude, productId) {
				configuration.Exclude = append(configuration.Exclude, productId)
			}
		}
	}
	saveConfigSettings(configuration)
	return 1
}
