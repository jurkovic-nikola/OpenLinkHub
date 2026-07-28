package keyboards

// Package: keyboards
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"LumenForge/src/common"
	"LumenForge/src/config"
	"LumenForge/src/logger"
	"LumenForge/src/rgb"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	pwd       = ""
	location  = ""
	keyboards = map[string]Keyboard{}
)

type FlashTapKey struct {
	Name    string
	KeyData int
}
type FlashTap struct {
	Active int                 `json:"active"`
	Mode   int                 `json:"mode"`
	Modes  map[int]string      `json:"modes"`
	Keys   map[int]FlashTapKey `json:"keys"`
	Color  rgb.Color           `json:"color"`
}

type KeyActuation struct {
	ActuationAllKeys              bool
	ActuationPoint                byte
	EnableActuationPointReset     bool
	ActuationResetPoint           byte
	EnableSecondaryActuationPoint bool
	SecondaryActuationPoint       byte
	SecondaryActuationResetPoint  byte
}
type Keyboard struct {
	Version             int           `json:"version"`
	Key                 string        `json:"key"`
	Device              string        `json:"device"`
	Layout              string        `json:"layout"`
	BufferSize          int           `json:"bufferSize"`
	KeyAssignmentLength int           `json:"keyAssignmentLength"`
	Rows                int           `json:"rows"`
	Row                 map[int]Row   `json:"row"`
	Zones               map[int]Zones `json:"zones"`
	Color               rgb.Color     `json:"color"`
	UppercaseClass      string        `json:"uppercaseClass"`
	FontSize            int           `json:"fontSize"`
	ModifierPosition    uint8         `json:"modifierPosition"`
}

type Zones struct {
	Color rgb.Color `json:"color"`
}

type Row struct {
	Top         int         `json:"top"`
	Css         string      `json:"css"`
	OverrideCss string      `json:"overrideCss"`
	Keys        map[int]Key `json:"keys"`
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
	ToggleDelay                   uint16    `json:"toggleDelay"`
	OnlyColor                     bool      `json:"onlyColor"`
	IsLock                        bool      `json:"isLock"`
	IsDialChange                  bool      `json:"isDialChange"`
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
	HasSubAction                  bool      `json:"hasSubAction"`
	RgbKey                        bool      `json:"rgbKey"`
	FnActionType                  uint8     `json:"fnActionType"`
	FnActionCommand               uint16    `json:"fnActionCommand"`
	ModifierPacketValue           uint8     `json:"modifierPacketValue"`
	ModifierShift                 byte      `json:"modifierShift"`
	RetainOriginal                bool      `json:"retainOriginal"`
	ProfileSwitch                 bool      `json:"profileSwitch"`
	OverrideBackground            bool      `json:"overrideBackground"`
	BluetoothProfile1             bool      `json:"bluetoothProfile1"`
	BluetoothProfile2             bool      `json:"bluetoothProfile2"`
	BluetoothProfile3             bool      `json:"bluetoothProfile3"`
	SlipstreamProfile             bool      `json:"slipstreamProfile"`
	ActuationPoint                byte      `json:"actuationPoint"`
	ActuationResetPoint           byte      `json:"actuationResetPoint"`
	EnableActuationPointReset     bool      `json:"enableActuationPointReset"`
	EnableSecondaryActuationPoint bool      `json:"enableSecondaryActuationPoint"`
	SecondaryActuationPoint       byte      `json:"secondaryActuationPoint"`
	SecondaryActuationResetPoint  byte      `json:"secondaryActuationResetPoint"`
	NoActuation                   bool      `json:"noActuation"`
	FlashTap                      bool      `json:"flashTap"`
	DeviceId                      string    `json:"deviceId"`
}

// Init will load and initialize keyboard data
func Init() {
	pwd = config.GetPaths().ApplicationRoot
	location = filepath.Join(config.GetPaths().ShippedDatabaseRoot, "keyboard")

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

// BuildKeyXMap parses the keyboard layout, sorts columns physically,
// and maps each key's packetIndex to a normalized X coordinate (0.0 to 1.0) using key centers.
func BuildKeyXMap(k *Keyboard) map[int]float64 {
	if k == nil {
		return nil
	}
	xMap := make(map[int]float64)
	absXMap := make(map[int]float64)
	maxX := 0.0

	// Determine global maxX across all rows first
	for _, row := range k.Row {
		var keyIds []int
		for kid := range row.Keys {
			keyIds = append(keyIds, kid)
		}
		sort.Ints(keyIds)

		currentX := 0.0
		for _, kid := range keyIds {
			key := row.Keys[kid]
			width := float64(key.Width)
			if key.KeyName == "----------" {
				width = 410
			}
			keyLeftEdge := currentX + float64(key.Left)
			rightEdge := keyLeftEdge + width
			if rightEdge > maxX {
				maxX = rightEdge
			}
			currentX = rightEdge
		}
	}

	isK95Platinum := strings.HasPrefix(k.Key, "k95platinum")

	// Calculate and assign keyCenters, stretching the lightbar row to full width if applicable
	for rowID, row := range k.Row {
		var keyIds []int
		for kid := range row.Keys {
			keyIds = append(keyIds, kid)
		}
		sort.Ints(keyIds)

		currentX := 0.0
		numKeys := len(keyIds)

		for i, kid := range keyIds {
			key := row.Keys[kid]
			width := float64(key.Width)
			if key.KeyName == "----------" {
				width = 410
			}
			keyLeftEdge := currentX + float64(key.Left)
			keyCenter := keyLeftEdge + width/2.0

			// Stretch Row 0 (lightbar) on K95 Platinum models to cover the entire width (10.0 to maxX - 60.0)
			// This shifts the lightbar coordinates slightly to the left, aligning its animation with the keys.
			if isK95Platinum && rowID == 0 && numKeys > 1 {
				t := float64(i) / float64(numKeys-1)
				keyCenter = 10.0 + t*(maxX-70.0)
			}

			for _, idx := range key.PacketIndex {
				absXMap[idx] = keyCenter
			}
			rightEdge := keyLeftEdge + width
			currentX = rightEdge
		}
	}

	if maxX > 0 {
		for idx, absX := range absXMap {
			xMap[idx] = absX / maxX
		}
	}
	return xMap
}
