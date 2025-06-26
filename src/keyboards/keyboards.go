package keyboards

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"encoding/json"
	"fmt"
	"os"
)

var (
	pwd       = ""
	location  = ""
	keyboards = map[string]Keyboard{}
)

type Keyboard struct {
	Version          int           `json:"version"`
	Key              string        `json:"key"`
	Device           string        `json:"device"`
	Layout           string        `json:"layout"`
	BufferSize       int           `json:"bufferSize"`
	Rows             int           `json:"rows"`
	Row              map[int]Row   `json:"row"`
	Zones            map[int]Zones `json:"zones"`
	Color            rgb.Color     `json:"color"`
	UppercaseClass   string        `json:"uppercaseClass"`
	FontSize         int           `json:"fontSize"`
	ModifierPosition uint8         `json:"modifierPosition"`
}

type Zones struct {
	Color rgb.Color `json:"color"`
}

type Row struct {
	Top  int         `json:"top"`
	Css  string      `json:"css"`
	Keys map[int]Key `json:"keys"`
}

type Key struct {
	KeyName                       string    `json:"keyName"`
	SubKeyName                    string    `json:"subKeyName"`
	KeyNameInternal               string    `json:"keyNameInternal"`
	Width                         int       `json:"width"`
	Height                        int       `json:"height"`
	Left                          int       `json:"left"`
	Top                           int       `json:"top"`
	PacketIndex                   []int     `json:"packetIndex"`
	Color                         rgb.Color `json:"color"`
	Zone                          int       `json:"zone"`
	Svg                           bool      `json:"svg"`
	Spacing                       []int     `json:"spacing"`
	Css                           string    `json:"css"`
	ExtraCss                      string    `json:"extraCss"`
	KeyEmpty                      []string  `json:"keyEmpty"`
	KeySpace                      string    `json:"keySpace"`
	KeyData                       []uint16  `json:"keyData"`
	CustomKeyData                 byte      `json:"customKeyData"`
	Default                       bool      `json:"default"`
	NoColor                       bool      `json:"noColor"`
	KeyHash                       []string  `json:"keyHash"`
	ActionType                    uint8     `json:"actionType"`
	ActionCommand                 uint16    `json:"actionCommand"`
	ActionHold                    bool      `json:"actionHold"`
	OnlyColor                     bool      `json:"onlyColor"`
	IsLock                        bool      `json:"isLock"`
	HalfKey                       bool      `json:"halfKey"`
	HalfKeyStart                  bool      `json:"halfKeyStart"`
	HalfKeyEnd                    bool      `json:"halfKeyEnd"`
	ColorOffOnFunctionKey         bool      `json:"colorOffOnFunctionKey"`
	ColorOffOnFunctionKeyInternal bool      `json:"colorOffOnFunctionKeyInternal"`
	Modifier                      bool      `json:"modifier"`
	FunctionKey                   bool      `json:"functionKey"`
	ModifierKey                   uint8     `json:"modifierKey"`
	BrightnessKey                 bool      `json:"brightnessKey"`
	ProfileKey                    bool      `json:"profileKey"`
	MacroRecordingKey             bool      `json:"macroRecordingKey"`
	MediaKey                      bool      `json:"mediaKey"`
	RgbKey                        bool      `json:"rgbKey"`
	FnActionType                  uint8     `json:"fnActionType"`
	FnActionCommand               uint16    `json:"fnActionCommand"`
	ModifierPacketValue           uint8     `json:"modifierPacketValue"`
	ModifierShift                 byte      `json:"modifierShift"`
	RetainOriginal                bool      `json:"retainOriginal"`
}

// Init will load and initialize keyboard data
func Init() {
	pwd = config.GetConfig().ConfigPath
	location = pwd + "/database/keyboard/"

	files, err := os.ReadDir(location)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Fatal("Unable to read content of a folder")
	}

	for _, fileInfo := range files {
		if fileInfo.IsDir() {
			continue // Exclude folders if any
		}

		// Define a full path of filename
		pullPath := location + fileInfo.Name()

		// Check if filename has .json extension
		if !common.IsValidExtension(pullPath, ".json") {
			continue
		}

		file, fe := os.Open(pullPath)
		if fe != nil {
			logger.Log(logger.Fields{"error": fe, "location": pullPath}).Error("Unable to open keyboard file")
			continue
		}

		// Decode and create profile
		var keyboard Keyboard

		reader := json.NewDecoder(file)
		if err = reader.Decode(&keyboard); err != nil {
			logger.Log(logger.Fields{"error": fe, "location": pullPath}).Error("Unable to decode keyboard file")
			continue
		}

		if len(keyboard.Layout) < 1 {
			logger.Log(logger.Fields{"error": fe, "location": pullPath}).Error("Keyboard has no layout field defined")
			continue
		}

		key := fmt.Sprintf("%s-%s", keyboard.Key, keyboard.Layout)
		keyboards[key] = keyboard
		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": fe, "location": pullPath}).Error("Unable to close keyboard file")
		}
	}
}

// GetKeyboard will return Keyboard struct for a given keyboard type
func GetKeyboard(key string) *Keyboard {
	if keyboard, ok := keyboards[key]; ok {
		return &keyboard
	}
	return nil
}

// GetLayouts will return a list of available layouts for given keyboard
func GetLayouts(key string) []string {
	var layouts []string
	for _, keyboard := range keyboards {
		if keyboard.Key == key {
			layouts = append(layouts, keyboard.Layout)
		}
	}
	return layouts
}
