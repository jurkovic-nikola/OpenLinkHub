package templates

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/temperatures"
	"OpenLinkHub/src/version"
	"html/template"
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
	TemperatureProbes interface{}
	Temperatures      map[string]temperatures.TemperatureProfileData
	Rgb               map[string]rgb.Profile
	SystemInfo        interface{}
	CpuTemp           string
	GpuTemp           string
	StorageTemp       []temperatures.StorageTemperatures
	BuildInfo         *version.BuildInfo
	Dashboard         dashboard.Dashboard
}

func Init() {
	tpl, err := template.ParseFiles(
		"web/devices.html",
		"web/docs.html",
		"web/index.html",
		"web/lsh.html",
		"web/lsh-vertical.html",
		"web/cc.html",
		"web/cc-vertical.html",
		"web/ccxt.html",
		"web/ccxt-vertical.html",
		"web/elite.html",
		"web/lncore.html",
		"web/lnpro.html",
		"web/cpro.html",
		"web/xc7.html",
		"web/memory.html",
		"web/rgb.html",
		"web/temperature.html",
	)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Failed to load templates")
	}

	templates = tpl
}

func GetTemplate() *template.Template {
	return templates
}
