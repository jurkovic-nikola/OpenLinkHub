package dashboard

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"encoding/json"
	"fmt"
	"math"
	"os"
)

type Dashboard struct {
	ShowCpu     bool `json:"showCpu"`
	ShowDisk    bool `json:"showDisk"`
	ShowGpu     bool `json:"showGpu"`
	ShowDevices bool `json:"showDevices"`
	VerticalUi  bool `json:"verticalUi"`
	Celsius     bool `json:"celsius"`
	ShowLabels  bool `json:"showLabels"`
}

var (
	location  = ""
	dashboard Dashboard
	upgrade   = map[string]any{
		"celsius":    true,
		"showLabels": true,
	}
)

// Init will initialize a new config object
func Init() {
	location = config.GetConfig().ConfigPath + "/dashboard.json"
	upgradeFile()
	file, err := os.Open(location)
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err, "file": location}).Error("Failed to close file.")
		}
	}(file)

	if err != nil {
		panic(err.Error())
	}
	if err = json.NewDecoder(file).Decode(&dashboard); err != nil {
		panic(err.Error())
	}
}

// upgradeFile will perform json file upgrade or create initial file
func upgradeFile() {
	if !common.FileExists(location) {
		logger.Log(logger.Fields{"file": location}).Info("Dashboard file is missing, creating initial one.")

		// File isn't found, create initial one
		dash := &Dashboard{
			ShowCpu:     true,
			ShowDisk:    true,
			ShowGpu:     true,
			ShowDevices: false,
			VerticalUi:  false,
			Celsius:     true,
			ShowLabels:  true,
		}
		if SaveDashboardSettings(dash, false) == 1 {
			logger.Log(logger.Fields{"file": location}).Info("Dashboard file is created.")
		} else {
			logger.Log(logger.Fields{"file": location}).Warn("Unable to create dashboard file.")
		}
	} else {
		// File exists, check for possible upgrade
		logger.Log(logger.Fields{"file": location}).Info("Dashboard file is found, checking for any upgrade...")

		save := false
		var data map[string]interface{}
		file, err := os.Open(location)
		defer func(file *os.File) {
			err = file.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err, "file": location}).Error("Failed to close file.")
			}
		}(file)

		if err != nil {
			logger.Log(logger.Fields{"error": err, "file": location}).Error("Unable to open file.")
			panic(err.Error())
		}
		if err = json.NewDecoder(file).Decode(&data); err != nil {
			logger.Log(logger.Fields{"error": err, "file": location}).Error("Unable to decode file.")
			panic(err.Error())
		}

		// Loop thru upgrade value
		for key, value := range upgrade {
			if _, ok := data[key]; !ok {
				logger.Log(logger.Fields{"key": key, "value": value}).Info("Upgrading fields...")
				data[key] = value
				save = true
			}
		}

		// Save on change
		if save {
			SaveDashboardSettings(data, false)
		} else {
			logger.Log(logger.Fields{"file": location}).Info("Nothing to upgrade.")
		}
	}
}

// SaveDashboardSettings will save dashboard settings
func SaveDashboardSettings(data any, reload bool) uint8 {
	// Convert to JSON
	buffer, err := json.Marshal(data)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format.")
		return 0
	}

	// Create profile filename
	file, fileErr := os.Create(location)
	if fileErr != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to save device dashboard.")
		return 0
	}

	// Write JSON buffer to file
	_, err = file.Write(buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to save device dashboard.")
		return 0
	}

	// Close file
	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to save device dashboard.")
	}

	if reload {
		Init()
	}
	return 1
}

// GetDashboard will return Dashboard struct
func GetDashboard() Dashboard {
	return dashboard
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

// TemperatureToString will return string temperature with unit value
func (d Dashboard) TemperatureToString(celsius float32) string {
	val := ""
	if d.Celsius {
		val = fmt.Sprintf("%.1f %s", roundFloat(float64(celsius), 2), "°C")
	} else {
		val = fmt.Sprintf("%.1f %s", roundFloat(float64((celsius*9/5)+32), 2), "°F")
	}
	return val
}
