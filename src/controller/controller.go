package controller

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/device"
	"OpenLinkHub/src/device/temperaturemonitor"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/server"
)

// Start will start new controller session
func Start() {
	config.Init()
	logger.Init()
	device.Init()
	temperaturemonitor.Init()
	server.Init()
}

// Stop will stop device control
func Stop() {
	temperaturemonitor.Stop()
	device.Stop()
}
