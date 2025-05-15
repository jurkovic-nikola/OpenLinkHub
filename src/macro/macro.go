package macro

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
	ActionType    uint8  `json:"actionType"`
	ActionCommand uint16 `json:"actionCommand"`
	ActionDelay   uint16 `json:"actionDelay"`
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
		logger.Log(logger.Fields{"error": err, "location": location}).Fatal("Unable to read content of a folder")
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
			logger.Log(logger.Fields{"error": err, "location": profileLocation}).Fatal("Unable to read macro profile")
			continue
		}

		// Decode and create profile
		var profile Macro
		reader := json.NewDecoder(file)
		if err = reader.Decode(&profile); err != nil {
			logger.Log(logger.Fields{"error": err, "location": profileLocation}).Fatal("Unable to decode temperature profile")
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

	macroId := len(macros) + 1
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
func NewMacroProfileValue(macroId int, actionType uint8, actionCommand uint16, actionDelay uint16) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	if val, ok := macros[macroId]; ok {
		profile := fmt.Sprintf("%s/database/macros/%s.json", config.GetConfig().ConfigPath, strings.ToLower(val.Name))
		if common.FileExists(profile) {
			length := len(val.Actions)
			val.Actions[length] = Actions{
				ActionType:    actionType,
				ActionCommand: actionCommand,
				ActionDelay:   actionDelay,
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
	// Convert to JSON
	buffer, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
		return
	}

	file, fileErr := os.Create(path)
	if fileErr != nil {
		logger.Log(logger.Fields{"error": err, "location": path}).Error("Unable to save nacro profile")
		return
	}

	_, err = file.Write(buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": path}).Error("Unable to write data")
		return
	}

	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": path}).Error("Unable to close file handle")
	}
}
