package controller

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/devices/lcd"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/keyboards"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/monitor"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/scheduler"
	"OpenLinkHub/src/server"
	"OpenLinkHub/src/stats"
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
	scheduler.Init()    // Scheduler
	systeminfo.Init()   // Build system info
	metrics.Init()      // Metrics
	rgb.Init()          // RGB
	lcd.Init()          // LCD
	temperatures.Init() // Temperatures
	keyboards.Init()    // Keyboards
	inputmanager.Init() // Input Manager
	stats.Init()        // Statistics
	devices.Init()      // Devices
	monitor.Init()      // Monitor
	server.Init()       // REST & WebUI
}

// Stop will stop device control
func Stop() {
	devices.Stop() // Devices
}
