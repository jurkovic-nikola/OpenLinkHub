package config

// Package: config
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"LumenForge/src/common"
	"encoding/json"
	"os"
	"os/user"
	"slices"
	"strings"
)

type Configuration struct {
	Debug      bool   `json:"debug"`
	Manual     bool   `json:"manual"`
	Frontend   bool   `json:"frontend"`
	Metrics    bool   `json:"metrics"`
	ConfigPath string `json:",omitempty"`

	ListenAddress string `json:"listenAddress"`
	ListenPort    int    `json:"listenPort"`

	LogFile  string `json:"logFile"`
	LogLevel string `json:"logLevel"`

	EnableSystemTray          bool `json:"enableSystemTray"`
	EnableGamepad             bool `json:"enableGamepad"`
	EnableMotherboard         bool `json:"enableMotherboard"`
	EnableOpenRGBTargetServer bool `json:"enableOpenRGBTargetServer,omitempty"` // Deprecated: legacy target listener compatibility.
	MotherboardBiosOnExit     bool `json:"motherboardBiosOnExit"`

	CheckDevicePermission bool `json:"checkDevicePermission"`
	GraphProfiles         bool `json:"graphProfiles"`
	ResumeDelay           int  `json:"resumeDelay"`
	TemperatureOffset     int  `json:"temperatureOffset"`

	CPUSensorChip string `json:"cpuSensorChip"`
	CpuTempFile   string `json:"cpuTempFile"`

	Memory                 bool   `json:"memory"`
	MemoryType             int    `json:"memoryType"`
	MemorySmBus            string `json:"memorySmBus"`
	MemorySku              string `json:"memorySku"`
	MemoryRegisterOverride []byte `json:"memoryRegisterOverride"`
	RamTempViaHwmon        bool   `json:"ramTempViaHwmon"`
	EnhancementKits        []byte `json:"enhancementKits"`

	AMDGpuIndex      int    `json:"amdGpuIndex"`
	AMDSmiPath       string `json:"amdsmiPath"`
	NvidiaGpuIndex   []int  `json:"nvidiaGpuIndex"`
	DefaultNvidiaGPU int    `json:"defaultNvidiaGPU"`

	OpenRGBPort int `json:"openRGBPort"`

	Exclude []uint16 `json:"exclude"`
}

var (
	location      = ""
	configuration Configuration
	upgrade       = map[string]any{
		"memorySku":              "",
		"resumeDelay":            15000,
		"logLevel":               "info",
		"logFile":                "",
		"enhancementKits":        make([]byte, 0),
		"temperatureOffset":      0,
		"amdGpuIndex":            0,
		"amdsmiPath":             "",
		"checkDevicePermission":  false,
		"cpuTempFile":            "",
		"graphProfiles":          false,
		"ramTempViaHwmon":        false,
		"nvidiaGpuIndex":         []int{0},
		"defaultNvidiaGPU":       0,
		"openRGBPort":            6742,
		"enableGamepad":          true,
		"enableMotherboard":      false,
		"motherboardBiosOnExit":  false,
		"memoryRegisterOverride": make([]byte, 0),
		"enableSystemTray":       false,
	}
	systemService = true
)

// Init will initialize a new config object
func Init() {
	setSystemService()

	var configPath = ""

	pwd, _ := os.Getwd()
	isAtomic := common.FileExists(pwd + "/atomic")
	if isAtomic {
		pwd = "/etc/LumenForge"
		configPath = "/etc/LumenForge"
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
	defer f.Close()
	if err = json.NewDecoder(f).Decode(&configuration); err != nil {
		panic(err.Error())
	}
	configuration.ConfigPath = configPath
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

// IsSystemService will return true if service runs under system context
func IsSystemService() bool {
	return systemService
}

// upgradeFile will create or upgrade config file
func upgradeFile(cfg string) {
	if !common.FileExists(cfg) {
		value := &Configuration{
			Debug:    false,
			Manual:   false,
			Frontend: true,
			Metrics:  false,

			ListenAddress: "127.0.0.1",
			ListenPort:    27003,

			LogFile:  "",
			LogLevel: "info",

			EnableSystemTray:      false,
			EnableGamepad:         true,
			EnableMotherboard:     false,
			MotherboardBiosOnExit: false,

			CheckDevicePermission: false,
			GraphProfiles:         true,
			ResumeDelay:           15000,
			TemperatureOffset:     0,

			CPUSensorChip: "",
			CpuTempFile:   "",

			Memory:                 false,
			MemoryType:             5,
			MemorySmBus:            "i2c-0",
			MemorySku:              "",
			MemoryRegisterOverride: make([]byte, 0),
			RamTempViaHwmon:        true,
			EnhancementKits:        make([]byte, 0),

			AMDGpuIndex:      0,
			AMDSmiPath:       "",
			NvidiaGpuIndex:   []int{0},
			DefaultNvidiaGPU: 0,

			OpenRGBPort: 6742,

			Exclude: make([]uint16, 0),
		}
		saveConfigSettings(value)
	} else {
		save := false
		var data map[string]interface{}
		file, err := os.Open(location)
		if err != nil {
			panic(err.Error())
		}
		defer func(f *os.File) {
			if closeErr := f.Close(); closeErr != nil {
				panic(closeErr.Error())
			}
		}(file)
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

// setSystemService will check and set systemService state
func setSystemService() {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LUMENFORGE_SERVICE_MODE"))) {
	case "system":
		systemService = true
		return
	case "user", "desktop":
		systemService = false
		return
	}

	uid := os.Getuid()
	if uid < 1000 {
		systemService = true
		return
	}

	if os.Getenv("DISPLAY") != "" ||
		os.Getenv("WAYLAND_DISPLAY") != "" ||
		os.Getenv("XDG_SESSION_TYPE") != "" {
		systemService = false
	}

	u, err := user.Current()
	if err != nil {
		systemService = true
		return
	}

	systemService = !strings.HasPrefix(u.HomeDir, "/home/")
}
