package appmonitor

// Package: appmonitor
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/applications"
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dispatcher"
	"OpenLinkHub/src/logger"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ApplicationProfile struct {
	ApplicationId   string            `json:"applicationId"`
	ApplicationName string            `json:"applicationName"`
	Enabled         bool              `json:"enabled"`
	Manual          bool              `json:"manual"`
	ActiveProfiles  map[string]string `json:"activeProfiles"`
	PassiveProfiles map[string]string `json:"passiveProfiles"`
	Executables     []string          `json:"-"`
}

var (
	pwd                 = ""
	location            = ""
	applicationProfiles = map[string]ApplicationProfile{}
	stop                chan struct{}
	done                chan struct{}
	mutex               sync.Mutex
	dispatch            dispatcher.DeviceDispatcher
	manual              = make(map[string]ApplicationProfile)
)

// Init starts application monitoring.
func Init() {
	pwd = config.GetConfig().ConfigPath
	location = pwd + "/database/apps/"

	LoadApplicationProfiles()

	if !config.GetConfig().AppMonitor || stop != nil {
		return
	}

	stop = make(chan struct{})
	done = make(chan struct{})

	go func() {
		defer close(done)

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		running := make(map[string]ApplicationProfile)
		for {
			current, err := scan()
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Warn("Unable to scan applications")
			} else {
				profiles := GetApplicationProfiles().(map[string]ApplicationProfile)
				for id, profile := range running {
					currentProfile, exists := profiles[id]
					if current[id] && exists && currentProfile.Enabled {
						continue
					}

					if profile.Manual {
						continue
					}

					if dispatch == nil {
						continue
					}

					for serial, profileName := range profile.PassiveProfiles {
						dispatch(serial, "ChangeDeviceProfile", profileName)
					}
					delete(running, id)
					logger.Log(logger.Fields{"applicationId": id}).Info("Application stopped")
				}

				for id := range current {
					profile, exists := profiles[id]
					if !exists || !profile.Enabled || profile.Manual {
						continue
					}

					if _, active := running[id]; active {
						running[id] = profile
						continue
					}

					if dispatch == nil {
						continue
					}
					
					for serial, profileName := range profile.ActiveProfiles {
						dispatch(serial, "ChangeDeviceProfile", profileName)
					}

					running[id] = profile
					logger.Log(logger.Fields{"applicationId": id}).Info("Application started")
				}
			}

			select {
			case <-stop:
				return
			case <-ticker.C:
			}
		}

	}()
}

// SetDispatcher will set device dispatcher
func SetDispatcher(ds dispatcher.DeviceDispatcher) {
	dispatch = ds
}

// Stop stops application monitoring and waits for the goroutine to exit.
func Stop() {
	if stop == nil {
		return
	}

	close(stop)
	<-done
	stop = nil
	done = nil
}

// LoadApplicationProfiles will load application profiles
func LoadApplicationProfiles() {
	mutex.Lock()
	defer mutex.Unlock()

	files, err := os.ReadDir(location)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to read content of a folder")
		return
	}

	for _, fileInfo := range files {
		if fileInfo.IsDir() {
			continue // Exclude folders if any
		}

		filePath := location + fileInfo.Name()

		// Check if filename has .json extension
		if !common.IsValidExtension(filePath, ".json") {
			continue
		}

		file, fe := os.Open(filePath)
		if fe != nil {
			logger.Log(logger.Fields{"error": fe, "location": filePath}).Error("Unable to open application file")
			continue
		}

		var applicationProfile ApplicationProfile

		reader := json.NewDecoder(file)
		if err = reader.Decode(&applicationProfile); err != nil {
			logger.Log(logger.Fields{"error": err, "location": filePath}).Error("Unable to decode application profile file")

			err = file.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err, "location": filePath}).Error("Unable to close application profile file")
			}
			continue
		}

		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": filePath}).Error("Unable to close application profile file")
		}

		if applicationProfile.ApplicationId == "" {
			logger.Log(logger.Fields{"location": filePath}).Warn("Application ID is profile empty")
			continue
		}

		appTemplate := applications.GetApplicationById(applicationProfile.ApplicationId)
		if appTemplate == nil {
			logger.Log(logger.Fields{"location": filePath}).Warn("Application Name in profile is empty")
			continue
		}

		applicationProfile.ApplicationName = appTemplate.Name
		applicationProfile.Executables = appTemplate.Executables
		applicationProfiles[applicationProfile.ApplicationId] = applicationProfile
	}
}

// ManualApplicationProfileChange will trigger manual application profile change
func ManualApplicationProfileChange(applicationId string) {
	mutex.Lock()
	defer mutex.Unlock()

	if val, ok := applicationProfiles[applicationId]; ok {
		if val.Enabled && val.Manual {
			if _, active := manual[val.ApplicationId]; active {
				for serial, profileName := range val.PassiveProfiles {
					if dispatch == nil {
						continue
					}
					dispatch(serial, "ChangeDeviceProfile", profileName)
				}
				delete(manual, val.ApplicationId)
			} else {
				manual[val.ApplicationId] = val
				for serial, profileName := range val.ActiveProfiles {
					if dispatch == nil {
						continue
					}
					dispatch(serial, "ChangeDeviceProfile", profileName)
				}
			}
		}
	}
}

// GetApplicationProfiles will return application profiles
func GetApplicationProfiles() interface{} {
	mutex.Lock()
	defer mutex.Unlock()

	profiles := make(map[string]ApplicationProfile, len(applicationProfiles))
	for id, profile := range applicationProfiles {
		active := make(map[string]string, len(profile.ActiveProfiles))
		for serial, name := range profile.ActiveProfiles {
			active[serial] = name
		}

		passive := make(map[string]string, len(profile.PassiveProfiles))
		for serial, name := range profile.PassiveProfiles {
			passive[serial] = name
		}

		profile.ActiveProfiles = active
		profile.PassiveProfiles = passive
		profile.Executables = append([]string(nil), profile.Executables...)
		profiles[id] = profile
	}

	return profiles
}

// GetManualApplicationProfiles will return application profiles with manual flag
func GetManualApplicationProfiles() interface{} {
	mutex.Lock()
	defer mutex.Unlock()

	profiles := make(map[string]ApplicationProfile, len(applicationProfiles))
	for id, profile := range applicationProfiles {
		if !profile.Manual {
			continue
		}

		active := make(map[string]string, len(profile.ActiveProfiles))
		for serial, name := range profile.ActiveProfiles {
			active[serial] = name
		}

		passive := make(map[string]string, len(profile.PassiveProfiles))
		for serial, name := range profile.PassiveProfiles {
			passive[serial] = name
		}

		profile.ActiveProfiles = active
		profile.PassiveProfiles = passive
		profile.Executables = append([]string(nil), profile.Executables...)
		profiles[id] = profile
	}

	return profiles
}

// AddApplicationProfile will add new application to the profiles
func AddApplicationProfile(profile ApplicationProfile) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := applicationProfiles[profile.ApplicationId]; ok {
		return 2
	}

	app := applications.GetApplicationById(profile.ApplicationId)
	if app == nil {
		return 0
	}

	profile.ApplicationName = app.Name
	profile.Executables = append([]string(nil), app.Executables...)
	profile.Manual = app.Manual

	if profile.ApplicationId == "" || strings.ContainsAny(profile.ApplicationId, "/\\") {
		return 0
	}

	filePath := filepath.Join(location, profile.ApplicationId+".json")

	// Save profile
	if err := common.SaveJsonData(filePath, profile); err != nil {
		logger.Log(logger.Fields{"error": err, "location": filePath}).Error("Unable to write application profile data")
		return 0
	}

	applicationProfiles[profile.ApplicationId] = profile
	return 1
}

// UpdateApplicationProfile will update application profile
func UpdateApplicationProfile(profile ApplicationProfile) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := applicationProfiles[profile.ApplicationId]; !ok {
		return 2
	}

	if profile.ApplicationId == "" || strings.ContainsAny(profile.ApplicationId, "/\\") {
		return 0
	}

	filePath := filepath.Join(location, profile.ApplicationId+".json")

	app := applications.GetApplicationById(profile.ApplicationId)
	if app == nil {
		return 0
	}

	profile.ApplicationName = app.Name
	profile.Executables = append([]string(nil), app.Executables...)

	// Save profile
	if err := common.SaveJsonData(filePath, profile); err != nil {
		logger.Log(logger.Fields{"error": err, "location": filePath}).Error("Unable to write application profile data")
		return 0
	}

	applicationProfiles[profile.ApplicationId] = profile
	return 1
}

// DeleteApplicationProfile will delete application profile
func DeleteApplicationProfile(applicationId string) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := applicationProfiles[applicationId]; !ok {
		return 0
	}

	if applicationId == "" || strings.ContainsAny(applicationId, "/\\") {
		return 0
	}

	filePath := filepath.Join(location, applicationId+".json")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		logger.Log(logger.Fields{
			"error":    err,
			"location": filePath,
		}).Error("Unable to delete application profile file")
		return 0
	}

	delete(applicationProfiles, applicationId)
	return 1
}

// scan returns application IDs matching running process executable names.
func scan() (map[string]bool, error) {
	processes, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	executables := make(map[string][]string)
	mutex.Lock()
	for _, profile := range applicationProfiles {
		if !profile.Enabled {
			continue
		}

		for _, executable := range profile.Executables {
			name := strings.ReplaceAll(executable, "\\", "/")
			name = strings.ToLower(filepath.Base(name))
			if name != "" && name != "." {
				executables[name] = append(executables[name], profile.ApplicationId)
			}
		}
	}
	mutex.Unlock()

	running := make(map[string]bool)
	for _, process := range processes {
		if !process.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(process.Name()); err != nil {
			continue
		}

		path := filepath.Join("/proc", process.Name())
		names := make([]string, 0, 3)

		if executable, err := os.Readlink(filepath.Join(path, "exe")); err == nil {
			names = append(names, strings.TrimSuffix(executable, " (deleted)"))
		}

		// Wine/Proton can expose the Windows executable as argv[0].
		if cmdline, err := os.ReadFile(filepath.Join(path, "cmdline")); err == nil {
			executable, _, _ := strings.Cut(string(cmdline), "\x00")
			if executable != "" {
				names = append(names, executable)
			}
		}

		if comm, err := os.ReadFile(filepath.Join(path, "comm")); err == nil {
			names = append(names, strings.TrimSpace(string(comm)))
		}

		for _, executable := range names {
			name := strings.ReplaceAll(executable, "\\", "/")
			name = strings.ToLower(filepath.Base(name))
			for _, id := range executables[name] {
				running[id] = true
			}
		}
	}

	return running, nil
}
