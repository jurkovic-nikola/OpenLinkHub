package dashboard

import (
	"OpenLinkHub/src/logger"
	"encoding/json"
	"os"
)

type Dashboard struct {
	ShowCpu     bool `json:"showCpu"`
	ShowDisk    bool `json:"showDisk"`
	ShowGpu     bool `json:"showGpu"`
	ShowDevices bool `json:"showDevices"`
	VerticalUi  bool `json:"verticalUi"`
}

var (
	pwd, _    = os.Getwd()
	dashboard Dashboard
)

// Init will initialize a new config object
func Init() {
	cfg := pwd + "/dashboard.json"
	f, err := os.Open(cfg)
	if err != nil {
		panic(err.Error())
	}
	if err = json.NewDecoder(f).Decode(&dashboard); err != nil {
		panic(err.Error())
	}
}

// SaveDashboardSettings will save dashboard settings
func SaveDashboardSettings(data *Dashboard) uint8 {
	cfg := pwd + "/dashboard.json"

	// Convert to JSON
	buffer, err := json.Marshal(data)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
		return 0
	}

	// Create profile filename
	file, fileErr := os.Create(cfg)
	if fileErr != nil {
		logger.Log(logger.Fields{"error": err, "location": cfg}).Error("Unable to save device dashboard")
		return 0
	}

	// Write JSON buffer to file
	_, err = file.Write(buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": cfg}).Error("Unable to save device dashboard")
		return 0
	}

	// Close file
	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": cfg}).Error("Unable to save device dashboard")
	}

	Init()
	return 1
}

// GetDashboard will return Dashboard struct
func GetDashboard() Dashboard {
	return dashboard
}
