package language

// Package: language
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"LumenForge/src/common"
	"LumenForge/src/config"
	"LumenForge/src/dashboard"
	"LumenForge/src/logger"
	"encoding/json"
	"os"
	"path/filepath"
)

var (
	pwd       = ""
	location  = ""
	languages = map[string]Language{}
)

type Language struct {
	Name   string            `json:"name"`
	Code   string            `json:"code"`
	Values map[string]string `json:"values"`
}

// Init will initialize a new language object
func Init() {
	pwd = config.GetPaths().ApplicationRoot
	location = filepath.Join(config.GetPaths().ShippedDatabaseRoot, "language")

	files, err := os.ReadDir(location)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Fatal("Unable to read content of a folder")
	}

	for _, fileInfo := range files {
		if fileInfo.IsDir() {
			continue // Exclude folders if any
		}

		// Define a full path of filename
		pullPath := filepath.Join(location, fileInfo.Name())

		// Check if filename has .json extension
		if !common.IsValidExtension(pullPath, ".json") {
			continue
		}

		file, fe := os.Open(pullPath)
		if fe != nil {
			logger.Log(logger.Fields{"error": fe, "location": pullPath}).Error("Unable to open language file")
			continue
		}

		// Decode and create profile
		var language Language

		reader := json.NewDecoder(file)
		if err = reader.Decode(&language); err != nil {
			logger.Log(logger.Fields{"error": err, "location": pullPath}).Error("Unable to decode language file")
			continue
		}

		languages[language.Code] = language
		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": fe, "location": pullPath}).Error("Unable to close language file")
		}
	}
}

// GetLanguages will return list of languages
func GetLanguages() map[string]Language {
	return languages
}

// GetLanguage will return Language with given key
func GetLanguage(key string) *Language {
	if len(key) == 0 {
		key = dashboard.GetDashboard().LanguageCode
	}
	if lang, ok := languages[key]; ok {
		return &lang
	}
	return nil
}

// GetValue will return value based on given key and current language code
func GetValue(key string) string {
	languageCode := dashboard.GetDashboard().LanguageCode
	if len(languageCode) == 0 {
		languageCode = "en_US"
	}

	if lang, ok := languages[languageCode]; ok {
		if value, found := lang.Values[key]; found {
			return value
		}
	}
	return ""
}
