package scheduler

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/logger"
	"encoding/json"
	"os"
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
	timer       = &time.Ticker{}
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
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err, "file": location}).Error("Failed to close file.")
		}
	}(file)

	if err != nil {
		logger.Log(logger.Fields{"error": err, "file": location}).Error("Failed to decode json")
	}
	if err = json.NewDecoder(file).Decode(&scheduler); err != nil {
		logger.Log(logger.Fields{"error": err, "file": location}).Error("Failed to decode json")
	}

	if scheduler.RGBControl {
		startTasks()
	}
}

// SaveSchedulerSettings will save dashboard settings
func SaveSchedulerSettings(data any) uint8 {
	buffer, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
		return 0
	}

	// Create profile filename
	file, fileErr := os.Create(location)
	if fileErr != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to save device dashboard.")
		return 0
	}

	// Write JSON buffer to file
	_, err = file.Write(buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to save device dashboard.")
		return 0
	}

	// Close file
	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to save device dashboard.")
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

	scheduler.RGBOff = rgbOff.Format(layout)
	scheduler.RGBOn = rgbOn.Format(layout)
	scheduler.RGBControl = enabled
	SaveSchedulerSettings(scheduler)
	timer.Stop()
	if scheduler.RGBControl {
		startTasks()
	}
	return 1
}

// GetScheduler will return current scheduler settings
func GetScheduler() *Scheduler {
	return &scheduler
}

func startTasks() {
	scheduledTimeOff, _ := time.Parse("15:04", scheduler.RGBOff)
	scheduledTimeOn, _ := time.Parse("15:04", scheduler.RGBOn)

	// Define the times you want the task to run
	schedules := []Schedule{
		{
			Hour:   scheduledTimeOff.Hour(),
			Minute: scheduledTimeOff.Minute(),
			Action: func() {
				if !scheduler.LightsOut {
					scheduler.LightsOut = true
					devices.ScheduleDeviceBrightness(0)
					SaveSchedulerSettings(scheduler)
				}
			},
		},
		{
			Hour:   scheduledTimeOn.Hour(),
			Minute: scheduledTimeOn.Minute(),
			Action: func() {
				if scheduler.LightsOut {
					scheduler.LightsOut = false
					devices.ScheduleDeviceBrightness(50)
					SaveSchedulerSettings(scheduler)
				}
			},
		},
	}

	timer = time.NewTicker(time.Duration(refreshTime) * time.Millisecond)
	go func() {
		for {
			select {
			case now := <-timer.C:
				for _, schedule := range schedules {
					if now.Hour() == schedule.Hour && now.Minute() == schedule.Minute {
						go schedule.Action()
					}
				}
			}
		}
	}()
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
		defer func(file *os.File) {
			err = file.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err, "file": location}).Error("Failed to close file.")
			}
		}(file)

		if err != nil {
			logger.Log(logger.Fields{"error": err, "file": location}).Error("Unable to open file.")
			panic(err.Error())
		}
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
