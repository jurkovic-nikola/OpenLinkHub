package controller

// Package: controller
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"LumenForge/src/audio"
	"LumenForge/src/cleanup"
	"LumenForge/src/config"
	"LumenForge/src/dashboard"
	"LumenForge/src/devices"
	"LumenForge/src/devices/lcd"
	"LumenForge/src/display"
	"LumenForge/src/inputmanager"
	"LumenForge/src/keyboards"
	"LumenForge/src/language"
	"LumenForge/src/logger"
	"LumenForge/src/macro"
	"LumenForge/src/media"
	"LumenForge/src/metrics"
	"LumenForge/src/monitor"
	"LumenForge/src/motherboards"
	"LumenForge/src/rgb"
	"LumenForge/src/scheduler"
	"LumenForge/src/server"
	"LumenForge/src/stats"
	"LumenForge/src/systeminfo"
	"LumenForge/src/systray"
	"LumenForge/src/temperatures"
	"LumenForge/src/version"
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

const serverShutdownTimeout = 7 * time.Second

var stopOnce sync.Once

var cleanupState struct {
	sync.Mutex
	media bool
	audio bool
}

// Start will start new controller session
func Start() {
	version.Init() // Build info
	config.Init()  // Configuration
	if legacyAddress, ignored := config.IgnoredListenAddress(); ignored {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"WARNING: deprecated non-loopback listenAddress %q is ignored; LumenForge listeners are restricted to 127.0.0.1\n",
			legacyAddress,
		)
	}
	logger.Init()  // Logger
	display.Init() // Displays
	media.Init()   // Media client
	cleanupState.Lock()
	cleanupState.media = true
	cleanupState.Unlock()
	audio.Init() // Audio
	cleanupState.Lock()
	cleanupState.audio = true
	cleanupState.Unlock()
	dashboard.Init()    // Dashboard
	systeminfo.Init()   // Build system info
	metrics.Init()      // Metrics
	rgb.Init()          // RGB
	lcd.Init()          // LCD
	temperatures.Init() // Temperatures
	keyboards.Init()    // Keyboards
	inputmanager.Init() // Input Manager
	cleanup.RegisterInput(inputmanager.Stop)
	stats.Init()        // Statistics
	macro.Init()        // Macro
	motherboards.Init() // Motherboards
	devices.Init()      // Devices
	cleanup.RegisterDevices(devices.Stop)
	monitor.Init()     // Monitor
	language.Init()    // Language
	scheduler.Init()   // Scheduler
	systray.InitTray() // System Tray
	server.Init()      // REST & WebUI
}

// Stop will stop device control
func Stop() {
	stopOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to gracefully stop REST server")
		}

		systray.Stop()
		cleanupState.Lock()
		audioStarted := cleanupState.audio
		mediaStarted := cleanupState.media
		cleanupState.Unlock()

		cleanup.StopDevices()
		cleanup.StopInput()
		if audioStarted {
			audio.StopAudio() // Virtual Audio
		}
		if mediaStarted {
			media.Stop() // Media client
		}
	})
}
