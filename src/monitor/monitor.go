package monitor

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/devices/lcd"
	"OpenLinkHub/src/logger"
	"github.com/godbus/dbus/v5"
	"time"
)

func Init() {
	go func() {
		// Connect to the session bus
		conn, err := dbus.ConnectSystemBus()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to connect to system bus")
			return
		}
		defer func(conn *dbus.Conn) {
			err = conn.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Error closing dbus")
			}
		}(conn)

		// Listen for the PrepareForSleep signal
		_ = conn.Object("org.freedesktop.login1", "/org/freedesktop/login1")
		ch := make(chan *dbus.Signal, 10)
		conn.Signal(ch)

		match := "type='signal',interface='org.freedesktop.login1.Manager',member='PrepareForSleep'"
		err = conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, match).Store()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to add D-Bus match")
		}
		for signal := range ch {
			if len(signal.Body) > 0 {
				if isSleeping, ok := signal.Body[0].(bool); ok {
					if isSleeping {
						logger.Log(logger.Fields{}).Info("Suspend detected. Sending Stop() to all devices")

						// Stop
						devices.Stop()
					} else {
						time.Sleep(time.Duration(config.GetConfig().ResumeDelay) * time.Millisecond)
						logger.Log(logger.Fields{}).Info("Resume detected. Sending Init() to all devices")

						// Init LCDs
						lcd.Reconnect()

						// Init devices
						devices.Init()
					}
				}
			}
		}
	}()
}
