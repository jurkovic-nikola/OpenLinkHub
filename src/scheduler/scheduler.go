package scheduler

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/logger"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// TaskManager manages the scheduling, execution, and lifecycle of tasks.
type TaskManager struct {
	mu       sync.Mutex
	wg       sync.WaitGroup
	exitCh   chan struct{}
	updateCh chan string
	runTime  string
	taskName string
	action   func()
}

type Scheduler struct {
	LightsOut  bool
	RGBControl bool   `json:"rgbControl"`
	RGBOff     string `json:"rgbOff"`
	RGBOn      string `json:"rgbOn"`
}

var (
	debug      = false
	location   = ""
	scheduler  Scheduler
	upgrade    = map[string]any{}
	layout     = "15:04"
	taskRgbOff = &TaskManager{}
	taskRgbOn  = &TaskManager{}
)

// Init will initialize a new config object
func Init() {
	debug = config.GetConfig().Debug
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
	rgbTasks(false)
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
	rgbTasks(true)
	return 1
}

// GetScheduler will return current scheduler settings
func GetScheduler() *Scheduler {
	return &scheduler
}

// rgbTasks controls tasks related to rgb
func rgbTasks(update bool) {
	if scheduler.RGBControl {
		if update {
			if taskRgbOff != nil {
				taskRgbOff.updateTime(scheduler.RGBOff)
			}
			if taskRgbOn != nil {
				taskRgbOn.updateTime(scheduler.RGBOn)
			}
		} else {
			taskRgbOff = newTaskManager(
				"Task_RGB_Off",
				func() { devices.ScheduleDeviceBrightness(0) },
				scheduler.RGBOff,
			)

			taskRgbOn = newTaskManager(
				"Task_RGB_On",
				func() { devices.ScheduleDeviceBrightness(50) },
				scheduler.RGBOn,
			)
			taskRgbOff.startTask()
			taskRgbOn.startTask()
		}
	} else {
		if taskRgbOff != nil {
			taskRgbOff.stopTask()
		}
		if taskRgbOn != nil {
			taskRgbOn.stopTask()
		}
	}
}

// NewTaskManager creates a new TaskManager.
func newTaskManager(taskName string, action func(), initialTime string) *TaskManager {
	return &TaskManager{
		exitCh:   make(chan struct{}),
		updateCh: make(chan string),
		runTime:  initialTime,
		taskName: taskName,
		action:   action,
	}
}

// Start begins the task manager's execution loop.
func (tm *TaskManager) startTask() {
	tm.wg.Add(1)
	go func() {
		defer tm.wg.Done()

		for {
			tm.mu.Lock()
			runTime := tm.runTime
			tm.mu.Unlock()

			// Calculate the next scheduled time
			now := time.Now()
			scheduledTime, _ := time.Parse("15:04", runTime)
			scheduledDateTime := time.Date(now.Year(), now.Month(), now.Day(), scheduledTime.Hour(), scheduledTime.Minute(), scheduledTime.Second(), 0, now.Location())
			if now.After(scheduledDateTime) {
				scheduledDateTime = scheduledDateTime.Add(24 * time.Hour)
			}

			duration := time.Until(scheduledDateTime)
			if debug {
				msg := fmt.Sprintf("%s scheduled to run at %s (in %s)", tm.taskName, scheduledDateTime.Format("15:04"), duration)
				logger.Log(logger.Fields{"message": msg}).Info("Scheduler")
			}

			// Wait for the scheduled time or an update/exit signal
			select {
			case <-time.After(duration):
				tm.action()
			case newTime := <-tm.updateCh:
				if debug {
					msg := fmt.Sprintf("%s time updated to %s", tm.taskName, newTime)
					logger.Log(logger.Fields{"message": msg}).Info("Scheduler")
				}
				tm.mu.Lock()
				tm.runTime = newTime
				tm.mu.Unlock()
			case <-tm.exitCh:
				if debug {
					msg := fmt.Sprintf("%s received exit signal, stopping...", tm.taskName)
					logger.Log(logger.Fields{"message": msg}).Info("Scheduler")
				}
				return
			}
		}
	}()
}

// UpdateTime updates the task's scheduled time.
func (tm *TaskManager) updateTime(newTime string) {
	tm.updateCh <- newTime
}

// Stop signals the task manager to stop.
func (tm *TaskManager) stopTask() {
	if tm.exitCh != nil {
		close(tm.exitCh)
	}
}

// Wait blocks until the task manager has stopped.
func (tm *TaskManager) wait() {
	tm.wg.Wait()
}

// Restart stops the task and starts it again with the same or updated parameters.
func (tm *TaskManager) Restart(newTime string) {
	if debug {
		msg := fmt.Sprintf("%s is restarting...", tm.taskName)
		logger.Log(logger.Fields{"message": msg}).Info("Scheduler")
	}

	tm.stopTask()
	tm.wait()
	tm.runTime = newTime
	tm.exitCh = make(chan struct{}) // Create a new channel for restart
	tm.updateCh = make(chan string)
	tm.startTask()
}
