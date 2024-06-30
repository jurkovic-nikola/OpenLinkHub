package controller

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/server"
	"OpenLinkHub/src/temperatures"
)

// Start will start new controller session
func Start() {
	config.Init()       // Configuration
	logger.Init()       // Logger
	rgb.Init()          // RGB
	temperatures.Init() // Temperatures
	devices.Init()      // Devices
	server.Init()       // REST & WebUI
}

// Stop will stop device control
func Stop() {
	devices.Stop()
}
