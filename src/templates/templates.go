package templates

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/scheduler"
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
	Scheduler         *scheduler.Scheduler
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
		"web/cc.html",
		"web/ccxt.html",
		"web/elite.html",
		"web/lncore.html",
		"web/lnpro.html",
		"web/cpro.html",
		"web/xc7.html",
		"web/memory.html",
		"web/k65pm.html",
		"web/k65plus.html",
		"web/k65plusW.html",
		"web/k70core.html",
		"web/k70pro.html",
		"web/k55core.html",
		"web/k100air.html",
		"web/k100airW.html",
		"web/rgb.html",
		"web/temperature.html",
		"web/scheduler.html",
	)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Failed to load templates")
	}

	templates = tpl
}

func GetTemplate() *template.Template {
	return templates
}
