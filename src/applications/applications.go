package applications

// Package: applications
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"encoding/json"
	"os"
	"sort"
)

var (
	pwd          = ""
	location     = ""
	applications = map[string]Application{}
)

type Application struct {
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	Layout      string   `json:"layout"`
	Executables []string `json:"executables"`
	SteamAppId  int      `json:"steamAppId"`
	Manual      bool     `json:"manual"`
}

// Init will load and initialize application metadata
func Init() {
	pwd = config.GetConfig().ConfigPath
	location = pwd + "/database/applications/"

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

		var application Application

		reader := json.NewDecoder(file)
		if err = reader.Decode(&application); err != nil {
			logger.Log(logger.Fields{"error": err, "location": filePath}).Error("Unable to decode application file")

			err = file.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err, "location": filePath}).Error("Unable to close application file")
			}
			continue
		}

		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": filePath}).Error("Unable to close application file")
		}

		if application.Id == "" {
			logger.Log(logger.Fields{"location": filePath}).Warn("Application ID is empty")
			continue
		}

		if application.Name == "" {
			logger.Log(logger.Fields{"location": filePath}).Warn("Application name is empty")
			continue
		}

		if !application.Manual {
			if len(application.Executables) == 0 {
				logger.Log(logger.Fields{"location": filePath}).Warn("Application executables are empty")
				continue
			}

			valid := true
			for _, executable := range application.Executables {
				if executable == "" {
					valid = false
					break
				}
			}

			if !valid {
				logger.Log(logger.Fields{"location": filePath}).Warn("Application contains an empty executable")
				continue
			}
		}

		applications[application.Id] = application
	}
}

// GetApplicationById will return application data by id
func GetApplicationById(id string) *Application {
	if application, ok := applications[id]; ok {
		return &application
	}
	return nil
}

// GetApplicationByName will return application data by name
func GetApplicationByName(name string) *Application {
	for _, application := range applications {
		if application.Name == name {
			return &application
		}
	}
	return nil
}

// GetAllApplications will return all supported applications
func GetAllApplications() []Application {
	var apps []Application
	for _, application := range applications {
		apps = append(apps, application)
	}

	sort.Slice(apps, func(i, j int) bool {
		return apps[i].Name < apps[j].Name
	})

	return apps
}

// GetApplicationBySteamId will return application by steamAppId
func GetApplicationBySteamId(id int) *Application {
	if id <= 0 {
		return nil
	}

	for _, application := range applications {
		if application.SteamAppId == id {
			return &application
		}
	}
	return nil
}
