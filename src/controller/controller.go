package controller

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/devices/lcd"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/server"
	"OpenLinkHub/src/systeminfo"
	"OpenLinkHub/src/temperatures"
	"OpenLinkHub/src/version"
)

// Start will start new controller session
func Start() {
	version.Init()      // Build info
	config.Init()       // Configuration
	logger.Init()       // Logger
	dashboard.Init()    // Dashboard
	systeminfo.Init()   // Build system info
	metrics.Init()      // Metrics
	rgb.Init()          // RGB
	lcd.Init()          // LCD
	temperatures.Init() // Temperatures
	devices.Init()      // Devices
	server.Init()       // REST & WebUI
}

// Stop will stop device control
func Stop() {
	devices.Stop()
}
