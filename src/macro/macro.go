package macro

// Package: macro
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

type Macro struct {
	Id      int             `json:"id"`
	Name    string          `json:"name"`
	Actions map[int]Actions `json:"actions"`
}

type Actions struct {
	ActionType            uint8  `json:"actionType"`
	ActionCommand         uint16 `json:"actionCommand"`
	ActionDelay           uint16 `json:"actionDelay"`
	ActionHold            bool   `json:"actionHold"`
	ActionRepeat          uint8  `json:"actionRepeat"`
	ActionRepeatDelay     uint16 `json:"actionRepeatDelay"`
	ActionText            string `json:"actionText"`
	MousePositionX        int    `json:"mousePositionX"`
	MousePositionY        int    `json:"mousePositionY"`
	MousePositionAbsolute bool   `json:"mousePositionAbsolute"`
}

type Tracker struct {
	Value uint16
	Type  uint8
}

var (
	pwd      = ""
	location = ""
	macros   = map[int]Macro{}
	mutex    sync.Mutex
)

// Init will load all available macro profiles
func Init() {
	pwd = config.GetConfig().ConfigPath
	location = pwd + "/database/macros/"

	files, err := os.ReadDir(location)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to read content of a folder")
		return
	}

	for _, fi := range files {
		if fi.IsDir() {
			continue // Exclude folders if any
		}

		// Define a full path of filename
		profileLocation := location + fi.Name()

		// Check if filename has .json extension
		if !common.IsValidExtension(profileLocation, ".json") {
			continue
		}

		file, err := os.Open(profileLocation)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": profileLocation}).Error("Unable to read macro profile")
			continue
		}

		// Decode and create profile
		var profile Macro
		reader := json.NewDecoder(file)
		if err = reader.Decode(&profile); err != nil {
			logger.Log(logger.Fields{"error": err, "location": profileLocation}).Error("Unable to decode macro profile")
			continue
		}
		macros[profile.Id] = profile
	}
}

// GetProfile will return macro profile based on macro ID
func GetProfile(macroId int) *Macro {
	if val, ok := macros[macroId]; ok {
		return &val
	}
	return nil
}

// GetProfiles will return all macro profiles
func GetProfiles() map[int]Macro {
	return macros
}

// DeleteMacroValue will delete macro value
func DeleteMacroValue(macroId, macroIndex int) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	if val, ok := macros[macroId]; ok {
		profile := fmt.Sprintf("%s/database/macros/%s.json", config.GetConfig().ConfigPath, strings.ToLower(val.Name))
		if _, ok := val.Actions[macroIndex]; ok {
			delete(val.Actions, macroIndex)
			macros[macroId] = val
			SaveProfile(profile, val)
			return 1
		}
	}
	return 0
}

// validatePressAndHold will validate press and hold action
func validatePressAndHold(macroId int) bool {
	count := 0
	length := 0
	if val, ok := macros[macroId]; ok {
		for _, v := range val.Actions {
			if v.ActionType == 5 {
				continue
			}

			length++
			if v.ActionHold {
				count++
			}
		}

		if count+1 == length {
			return false
		}
	}
	return true
}

// UpdateMacroValue will update macro value
func UpdateMacroValue(macroId, macroIndex int, macroAction *Actions) uint8 {
	mutex.Lock()
	defer mutex.Unlock()
	if val, ok := macros[macroId]; ok {
		profile := fmt.Sprintf("%s/database/macros/%s.json", config.GetConfig().ConfigPath, strings.ToLower(val.Name))
		if action, ok := val.Actions[macroIndex]; ok {
			if macroAction.ActionHold {
				if !validatePressAndHold(macroId) {
					return 2
				}
			}
			action.ActionHold = macroAction.ActionHold
			action.ActionRepeat = macroAction.ActionRepeat
			action.ActionRepeatDelay = macroAction.ActionRepeatDelay
			action.MousePositionX = macroAction.MousePositionX
			action.MousePositionY = macroAction.MousePositionY
			action.MousePositionAbsolute = macroAction.MousePositionAbsolute
			val.Actions[macroIndex] = action
			macros[macroId] = val
			SaveProfile(profile, val)
			return 1
		}
	}
	return 0
}

// DeleteMacroProfile will delete macro profile
func DeleteMacroProfile(macroId int) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	if val, ok := macros[macroId]; ok {
		profile := fmt.Sprintf("%s/database/macros/%s.json", config.GetConfig().ConfigPath, strings.ToLower(val.Name))
		if common.FileExists(profile) {
			err := os.Remove(profile)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "profile": profile}).Fatal("Unable to delete macro")
				return 0
			}
			delete(macros, macroId)
			return 1
		}
	}
	return 0
}

// NewMacroProfile will create new macro profile
func NewMacroProfile(macroName string) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	profile := fmt.Sprintf("%s/database/macros/%s.json", config.GetConfig().ConfigPath, strings.ToLower(macroName))
	if common.FileExists(profile) {
		return 0
	}

	maxID := 0
	for id := range macros {
		if id > maxID {
			maxID = id
		}
	}
	macroId := maxID + 1
	if _, ok := macros[macroId]; ok {
		return 0
	}

	macro := Macro{
		Id:      macroId,
		Name:    macroName,
		Actions: map[int]Actions{},
	}
	macros[macroId] = macro
	SaveProfile(profile, macro)
	return 1
}

// NewMacroProfileValue will create new macro profile value
func NewMacroProfileValue(macroId int, macroAction *Actions) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	if val, ok := macros[macroId]; ok {
		profile := fmt.Sprintf("%s/database/macros/%s.json", config.GetConfig().ConfigPath, strings.ToLower(val.Name))
		if common.FileExists(profile) {
			length := len(val.Actions)
			val.Actions[length] = Actions{
				ActionType:            macroAction.ActionType,
				ActionCommand:         macroAction.ActionCommand,
				ActionDelay:           macroAction.ActionDelay,
				ActionText:            macroAction.ActionText,
				MousePositionX:        macroAction.MousePositionX,
				MousePositionY:        macroAction.MousePositionY,
				MousePositionAbsolute: macroAction.MousePositionAbsolute,
			}
			macros[macroId] = val
			SaveProfile(profile, macros[macroId])
			return 1
		}
	}
	return 0
}

// SaveProfile saves macro profile
func SaveProfile(path string, data Macro) {
	if err := common.SaveJsonData(path, data); err != nil {
		logger.Log(logger.Fields{"error": err, "location": path}).Error("Unable to save macro data")
		return
	}
}
