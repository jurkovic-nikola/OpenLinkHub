package templates

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/macro"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/scheduler"
	"OpenLinkHub/src/temperatures"
	"OpenLinkHub/src/version"
	"fmt"
	"html/template"
	"os"
)

var (
	templates *template.Template
)

type Web struct {
	Title             string
	Tpl               *template.Template
	Devices           map[string]*devices.Device
	Configuration     config.Configuration
	Device            interface{}
	Lcd               interface{}
	LCDImages         interface{}
	TemperatureProbes interface{}
	RGBProfiles       map[string]interface{}
	Temperatures      map[string]temperatures.TemperatureProfileData
	Macros            map[int]macro.Macro
	LCDProfiles       map[uint8]interface{}
	LCDSensors        map[uint8]string
	InputActions      map[uint8]inputmanager.InputAction
	Scheduler         *scheduler.Scheduler
	Rgb               map[string]rgb.Profile
	SystemInfo        interface{}
	Stats             interface{}
	CpuTemp           string
	GpuTemp           string
	Page              string
	StorageTemp       []temperatures.StorageTemperatures
	BuildInfo         *version.BuildInfo
	Dashboard         dashboard.Dashboard
}

// Init will parse all templates
func Init() {
	var templateList []string
	htmlDirectory := config.GetConfig().ConfigPath + "/web/"
	files, err := os.ReadDir(htmlDirectory)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "path": htmlDirectory}).Fatal("Unable to read content of a html directory")
	}

	for _, fi := range files {
		templateFile := fmt.Sprintf("%s%s", htmlDirectory, fi.Name())

		// Check if filename has .html extension
		if !common.IsValidExtension(templateFile, ".html") {
			continue
		}

		templateList = append(templateList, templateFile)
	}

	tpl, err := template.ParseFiles(templateList...)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Failed to load templates")
	}

	templates = tpl
}

// GetTemplate will return a list of all templates
func GetTemplate() *template.Template {
	return templates
}
