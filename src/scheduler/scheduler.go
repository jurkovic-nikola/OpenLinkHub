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
	location          = ""
	scheduler         Scheduler
	upgrade           = map[string]any{}
	schedulerInterval = 10000
	timer             = &time.Ticker{}
	schedulerChan     = make(chan bool)
)

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
		panic(err.Error())
	}
	if err = json.NewDecoder(file).Decode(&scheduler); err != nil {
		panic(err.Error())
	}
	runScheduler(false)
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
		if SaveSchedulerSettings(sche, false) == 1 {
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
			SaveSchedulerSettings(data, false)
		} else {
			logger.Log(logger.Fields{"file": location}).Info("Nothing to upgrade.")
		}
	}
}

// SaveSchedulerSettings will save dashboard settings
func SaveSchedulerSettings(data any, reload bool) uint8 {
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

	if reload {
		Init()
	}
	return 1
}

// UpdateRgbSettings will update RGB scheduler settings
func UpdateRgbSettings(enabled bool, start, end string) uint8 {
	layout := "15:04"
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
	SaveSchedulerSettings(scheduler, true)
	runScheduler(true)
	return 1
}

// GetScheduler will return current scheduler settings
func GetScheduler() *Scheduler {
	return &scheduler
}

func runScheduler(restart bool) {
	if restart {
		timer.Stop()
		schedulerChan <- true
	}

	timer = time.NewTicker(time.Duration(schedulerInterval) * time.Millisecond)
	schedulerChan = make(chan bool)
	go func() {
		for {
			select {
			case <-timer.C:
				rgbControl()
			case <-schedulerChan:
				timer.Stop()
				return
			}
		}
	}()
}

func rgbControl() {
	if scheduler.RGBControl {
		layout := "15:04"
		// Parse the string into a time.Time object
		ptTimeOff, err := time.Parse(layout, scheduler.RGBOff)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to process rgb scheduler start time")
			return
		}
		ptOffEnd, err := time.Parse(layout, scheduler.RGBOn)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to process rgb scheduler end time")
			return
		}

		cd := time.Now()
		combinedTimeOffStart := time.Date(cd.Year(), cd.Month(), cd.Day(), ptTimeOff.Hour(), ptTimeOff.Minute(), 0, 0, cd.Location())
		combinedTimeOffEnd := time.Date(cd.Year(), cd.Month(), cd.Day(), ptOffEnd.Hour(), ptOffEnd.Minute(), 0, 0, cd.Location())

		if cd.After(combinedTimeOffStart) && cd.Before(combinedTimeOffEnd) {
			if !scheduler.LightsOut {
				scheduler.LightsOut = true
				SaveSchedulerSettings(scheduler, false)
				devices.ScheduleDeviceBrightness(4)
			}
		} else {
			if scheduler.LightsOut {
				scheduler.LightsOut = false
				SaveSchedulerSettings(scheduler, false)
				devices.ScheduleDeviceBrightness(0)
			}
		}
	}
}
