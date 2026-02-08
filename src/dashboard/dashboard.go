package dashboard

// Package: dashboard
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"slices"
	"strings"
)

type Dashboard struct {
	ShowCpu          bool     `json:"showCpu"`
	ShowDisk         bool     `json:"showDisk"`
	ShowGpu          bool     `json:"showGpu"`
	ShowDevices      bool     `json:"showDevices"`
	VerticalUi       bool     `json:"verticalUi"`
	Celsius          bool     `json:"celsius"`
	ShowLabels       bool     `json:"showLabels"`
	ShowBattery      bool     `json:"showBattery"`
	TemperatureBar   bool     `json:"temperatureBar"`
	SidebarCollapsed bool     `json:"sidebarCollapsed"`
	LanguageCode     string   `json:"languageCode"`
	PageTitle        string   `json:"pageTitle"`
	Devices          []string `json:"devices"`
	Theme            string   `json:"theme"`
	Themes           []string `json:"themes"`
}

var (
	location  = ""
	dashboard Dashboard
	upgrade   = map[string]any{
		"celsius":          true,
		"showLabels":       true,
		"showBattery":      false,
		"languageCode":     "en_US",
		"temperatureBar":   true,
		"pageTitle":        "OPENLINKHUB WebUI",
		"sidebarCollapsed": false,
		"devices":          []string{},
		"theme":            "default",
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
	loadThemes()

	if len(dashboard.Theme) == 0 {
		dashboard.Theme = "default"
	}
}

// upgradeFile will perform json file upgrade or create initial file
func upgradeFile() {
	if !common.FileExists(location) {
		logger.Log(logger.Fields{"file": location}).Info("Dashboard file is missing, creating initial one.")

		// File isn't found, create initial one
		dash := &Dashboard{
			ShowCpu:          true,
			ShowDisk:         true,
			ShowGpu:          true,
			ShowDevices:      false,
			VerticalUi:       false,
			Celsius:          true,
			ShowLabels:       true,
			ShowBattery:      false,
			TemperatureBar:   true,
			SidebarCollapsed: false,
			LanguageCode:     "en_US",
			PageTitle:        "OPENLINKHUB WebUI",
			Devices:          []string{},
			Theme:            "default",
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

// loadThemes will load CSS themes
func loadThemes() {
	dashboard.Themes = nil

	themesPath := config.GetConfig().ConfigPath + "/static/css/themes/"
	files, err := os.ReadDir(themesPath)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": themesPath}).Fatal("Unable to read content of a folder")
	}

	for _, fi := range files {
		if fi.IsDir() {
			continue // Exclude folders if any
		}

		// Define a full path of filename
		themeLocation := themesPath + fi.Name()

		// Check if filename has .json extension
		if !common.IsValidExtension(themeLocation, ".css") {
			continue
		}

		fileName := strings.Split(fi.Name(), ".")[0]
		if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", fileName); !m {
			continue
		}
		dashboard.Themes = append(dashboard.Themes, fileName)
	}
}

// SaveDashboardSettings will save dashboard settings
func SaveDashboardSettings(data any, reload bool) uint8 {
	if err := common.SaveJsonData(location, data); err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to save dashboard data")
		return 0
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

// Temperature will return string temperature with unit value
func (d Dashboard) Temperature(celsius float32) []string {
	val := make([]string, 2)
	if d.Celsius {
		val[0] = fmt.Sprintf("%.0f", roundFloat(float64(celsius), 2))
		val[1] = "°C"
	} else {
		val[0] = fmt.Sprintf("%.0f", roundFloat(float64((celsius*9/5)+32), 2))
		val[1] = "°F"
	}
	return val
}

func GetDevices() []string {
	return dashboard.Devices
}

func AddDevice(serial string) uint8 {
	if slices.Contains(dashboard.Devices, serial) {
		return 0
	}
	dashboard.Devices = append(dashboard.Devices, serial)

	SaveDashboardSettings(dashboard, true)
	return 1
}

func RemoveDevice(serial string) uint8 {
	if !slices.Contains(dashboard.Devices, serial) {
		return 0
	}

	filtered := dashboard.Devices[:0]
	for _, d := range dashboard.Devices {
		if d != serial {
			filtered = append(filtered, d)
		}
	}
	dashboard.Devices = filtered

	SaveDashboardSettings(dashboard, true)
	return 1
}
