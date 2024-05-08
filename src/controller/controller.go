package controller

import (
	"OpenICUELinkHub/src/config"
	"OpenICUELinkHub/src/device"
	"OpenICUELinkHub/src/device/temperaturemonitor"
	"OpenICUELinkHub/src/logger"
	"OpenICUELinkHub/src/server"
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
