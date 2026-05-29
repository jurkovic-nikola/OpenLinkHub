package scheduler

// Package: scheduler
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/logger"
	"encoding/json"
	"os"
	"sync"
	"time"
)

type Scheduler struct {
	LightsOut  bool
	RGBControl bool   `json:"rgbControl"`
	RGBOff     string `json:"rgbOff"`
	RGBOn      string `json:"rgbOn"`
}

var (
	location    = ""
	scheduler   Scheduler
	upgrade     = map[string]any{}
	layout      = "15:04"
	mu          sync.Mutex
	timer       *time.Ticker
	stopChan    chan struct{}
	refreshTime = 5000
)

// Schedule represents a specific time to execute a task
type Schedule struct {
	Hour   int
	Minute int
	Action func()
}

// Init will initialize a new config object
func Init() {
	location = config.GetConfig().ConfigPath + "/database/scheduler.json"
	upgradeFile()
	file, err := os.Open(location)

	if err != nil {
		logger.Log(logger.Fields{"error": err, "file": location}).Error("Failed to open scheduler file")
		return
	}

	defer func() {
		if err := file.Close(); err != nil {
			logger.Log(logger.Fields{"error": err, "file": location}).Error("Failed to close file")
		}
	}()

	var loaded Scheduler
	if err = json.NewDecoder(file).Decode(&loaded); err != nil {
		logger.Log(logger.Fields{"error": err, "file": location}).Error("Failed to decode json")
		return
	}

	mu.Lock()
	scheduler = loaded
	enabled := scheduler.RGBControl
	mu.Unlock()

	if enabled {
		startTasks()
	}
}

// SaveSchedulerSettings will save dashboard settings
func SaveSchedulerSettings(data any) uint8 {
	if err := common.SaveJsonData(location, data); err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to save scheduler data")
		return 0
	}
	return 1
}

// UpdateRgbSettings will update RGB scheduler settings
func UpdateRgbSettings(enabled bool, start, end string) uint8 {
	rgbOff, err := time.Parse(layout, start)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Failed to process rgb scheduler start time")
		return 0
	}

	rgbOn, err := time.Parse(layout, end)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Failed to process rgb scheduler end time")
		return 0
	}

	mu.Lock()
	scheduler.RGBOff = rgbOff.Format(layout)
	scheduler.RGBOn = rgbOn.Format(layout)
	scheduler.RGBControl = enabled
	current := scheduler
	mu.Unlock()

	SaveSchedulerSettings(current)
	if current.RGBControl {
		startTasks()
	} else {
		stopTasks()
	}

	return 1
}

// GetScheduler will return current scheduler settings
func GetScheduler() Scheduler {
	mu.Lock()
	defer mu.Unlock()
	return scheduler
}

// stopTasks will stop tasks
func stopTasks() {
	mu.Lock()
	defer mu.Unlock()

	if timer != nil {
		timer.Stop()
		timer = nil
	}
	if stopChan != nil {
		close(stopChan)
		stopChan = nil
	}
}

func startTasks() {
	stopTasks()

	mu.Lock()
	rgbOff := scheduler.RGBOff
	rgbOn := scheduler.RGBOn
	mu.Unlock()

	scheduledTimeOff, _ := time.Parse("15:04", rgbOff)
	scheduledTimeOn, _ := time.Parse("15:04", rgbOn)

	isInOffRange := func(now, off, on time.Time) bool {
		offToday := time.Date(now.Year(), now.Month(), now.Day(), off.Hour(), off.Minute(), 0, 0, now.Location())
		onToday := time.Date(now.Year(), now.Month(), now.Day(), on.Hour(), on.Minute(), 0, 0, now.Location())

		if onToday.Before(offToday) {
			return now.After(offToday) || now.Before(onToday)
		}
		return now.After(offToday) && now.Before(onToday)
	}

	// Define the times you want the task to run
	schedules := []Schedule{
		{
			Hour:   scheduledTimeOff.Hour(),
			Minute: scheduledTimeOff.Minute(),
			Action: func() {
				mu.Lock()
				if !scheduler.LightsOut {
					scheduler.LightsOut = true
					current := scheduler
					mu.Unlock()

					devices.ScheduleDeviceBrightness(0)
					SaveSchedulerSettings(current)
				} else {
					mu.Unlock()
				}
			},
		},
		{
			Hour:   scheduledTimeOn.Hour(),
			Minute: scheduledTimeOn.Minute(),
			Action: func() {
				mu.Lock()
				if scheduler.LightsOut {
					scheduler.LightsOut = false
					current := scheduler
					mu.Unlock()

					devices.ScheduleDeviceBrightness(1)
					SaveSchedulerSettings(current)
				} else {
					mu.Unlock()
				}
			},
		},
	}

	mu.Lock()
	localStop := make(chan struct{})
	localTimer := time.NewTicker(time.Duration(refreshTime) * time.Millisecond)
	stopChan = localStop
	timer = localTimer
	mu.Unlock()

	go func(t *time.Ticker, stop <-chan struct{}) {
		// Check if we started (or settings were updated) while already inside the off
		// range and apply the initial brightness state after a short settling delay.
		// This runs in the goroutine so that callers (Init, UpdateRgbSettings) are
		// not blocked for the duration of the sleep.
		time.Sleep(time.Duration(refreshTime) * time.Millisecond)
		timeNow := time.Now()
		if isInOffRange(timeNow, scheduledTimeOff, scheduledTimeOn) {
			mu.Lock()
			if !scheduler.LightsOut {
				scheduler.LightsOut = true
				current := scheduler
				mu.Unlock()

				devices.ScheduleDeviceBrightness(0)
				SaveSchedulerSettings(current)
			} else {
				mu.Unlock()
			}
		} else {
			mu.Lock()
			if scheduler.LightsOut {
				scheduler.LightsOut = false
				current := scheduler
				mu.Unlock()

				devices.ScheduleDeviceBrightness(1)
				SaveSchedulerSettings(current)
			} else {
				mu.Unlock()
			}
		}

		for {
			select {
			case now := <-t.C:
				for _, schedule := range schedules {
					if now.Hour() == schedule.Hour && now.Minute() == schedule.Minute {
						go schedule.Action()
					}
				}
			case <-stop:
				return
			}
		}
	}(localTimer, localStop)
}

// upgradeFile will perform json file upgrade or create initial file
func upgradeFile() {
	if !common.FileExists(location) {
		logger.Log(logger.Fields{"file": location}).Info("Scheduler file is missing, creating initial one.")

		// File isn't found, create initial one
		sche := &Scheduler{
			RGBControl: false,
			RGBOff:     time.Now().Format("15:04"),
			RGBOn:      time.Now().Format("15:04"),
		}
		if SaveSchedulerSettings(sche) == 1 {
			logger.Log(logger.Fields{"file": location}).Info("Scheduler file is created.")
		} else {
			logger.Log(logger.Fields{"file": location}).Warn("Unable to create scheduler file.")
		}
	} else {
		// File exists, check for possible upgrade
		logger.Log(logger.Fields{"file": location}).Info("Scheduler file is found, checking for any upgrade...")

		save := false
		var data map[string]interface{}
		file, err := os.Open(location)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "file": location}).Error("Unable to open file.")
			panic(err.Error())
		}
		defer func() {
			if err := file.Close(); err != nil {
				logger.Log(logger.Fields{"error": err, "file": location}).Error("Failed to close file.")
			}
		}()

		if err = json.NewDecoder(file).Decode(&data); err != nil {
			logger.Log(logger.Fields{"error": err, "file": location}).Error("Unable to decode file.")
			panic(err.Error())
		}

		// Loop thru upgrade value
		for key, value := range upgrade {
			if _, ok := data[key]; !ok {
				logger.Log(logger.Fields{"key": key, "value": value}).Info("Upgrading fields...")
				data[key] = value
				save = true
			}
		}

		// Save on change
		if save {
			SaveSchedulerSettings(data)
		} else {
			logger.Log(logger.Fields{"file": location}).Info("Nothing to upgrade.")
		}
	}
}
