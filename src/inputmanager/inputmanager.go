package inputmanager

// Package: Input Manager
// This is the primary package for handling user inputs.
// All device input actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/logger"
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type KeyAssignment struct {
	Name          string `json:"name"`
	Default       bool   `json:"default"`
	ActionType    uint8  `json:"actionType"`
	ActionCommand uint8  `json:"actionCommand"`
	ActionHold    bool   `json:"actionHold"`
	ButtonIndex   int    `json:"buttonIndex"`
}

type InputAction struct {
	Name        string // Key name
	CommandCode uint16 // Key code
	Media       bool   // Key can control media playback
}

const (
	None           uint8 = 0
	VolumeUp       uint8 = 1
	VolumeDown     uint8 = 2
	VolumeMute     uint8 = 3
	MediaStop      uint8 = 4
	MediaPrev      uint8 = 5
	MediaPlayPause uint8 = 6
	MediaNext      uint8 = 7
	Number1        uint8 = 8
	Number2        uint8 = 9
	Number3        uint8 = 10
	Number4        uint8 = 11
	Number5        uint8 = 12
	Number6        uint8 = 13
	Number7        uint8 = 14
	Number8        uint8 = 15
	Number9        uint8 = 16
	Number0        uint8 = 17
	KeyMinus       uint8 = 18
	KeyEqual       uint8 = 19
	KeyQ           uint8 = 20
	KeyW           uint8 = 21
	KeyE           uint8 = 22
	KeyR           uint8 = 23
	KeyT           uint8 = 24
	KeyY           uint8 = 25
	KeyU           uint8 = 26
	KeyI           uint8 = 27
	KeyO           uint8 = 28
	KeyP           uint8 = 29
	KeyA           uint8 = 30
	KeyS           uint8 = 31
	KeyD           uint8 = 32
	KeyF           uint8 = 33
	KeyG           uint8 = 34
	KeyH           uint8 = 35
	KeyJ           uint8 = 36
	KeyK           uint8 = 37
	KeyL           uint8 = 38
	KeyZ           uint8 = 39
	KeyX           uint8 = 40
	KeyC           uint8 = 41
	KeyV           uint8 = 42
	KeyB           uint8 = 43
	KeyN           uint8 = 44
	KeyM           uint8 = 45
	KeyF1          uint8 = 46
	KeyF2          uint8 = 47
	KeyF3          uint8 = 48
	KeyF4          uint8 = 49
	KeyF5          uint8 = 50
	KeyF6          uint8 = 51
	KeyF7          uint8 = 52
	KeyF8          uint8 = 53
	KeyF9          uint8 = 54
	KeyF10         uint8 = 55
	KeyF11         uint8 = 56
	KeyF12         uint8 = 57
	KeyBack        uint8 = 58
	KeyTab         uint8 = 59
	KeyEsc         uint8 = 60
	KeyTilde       uint8 = 61
	KeyLeftSquare  uint8 = 62
	KeyRightSquare uint8 = 63
	KeyBackslash   uint8 = 64
	KeyCapslock    uint8 = 65
	KeySemicolon   uint8 = 66
	KeySingleQuote uint8 = 67
	KeyEnter       uint8 = 68
	KeyLeftShift   uint8 = 69
	KeyComma       uint8 = 70
	KeyDot         uint8 = 71
	KeySlash       uint8 = 72
	KeyRightShift  uint8 = 73
	KeyUp          uint8 = 74
	KeyLeftCtrl    uint8 = 75
	KeyWindowsKey  uint8 = 76
	KeyLeftAlt     uint8 = 77
	KeySpace       uint8 = 78
	KeyRightAlt    uint8 = 79
	KeyMenu        uint8 = 80
	KeyRightCtrl   uint8 = 81
	KeyLeft        uint8 = 82
	KeyDown        uint8 = 83
	KeyRight       uint8 = 84
	KeyIns         uint8 = 85
	KeyHome        uint8 = 86
	KeyPgUp        uint8 = 87
	KeyDel         uint8 = 88
	KeyEnd         uint8 = 89
	KeyPgDn        uint8 = 90
)

var (
	evKey          uint16 = 0x01
	evSyn          uint16 = 0x00
	keyVolumeUp    uint16 = 0x73
	keyVolumeDown  uint16 = 0x72
	keyVolumeMute  uint16 = 0x71
	keyMediaStop   uint16 = 0xA6
	keyMediaPrev   uint16 = 0xA5
	keyMediaPlay   uint16 = 0xA4
	keyMediaNext   uint16 = 0xA3
	keyNumber1     uint16 = 0x2
	keyNumber2     uint16 = 0x3
	keyNumber3     uint16 = 0x4
	keyNumber4     uint16 = 0x5
	keyNumber5     uint16 = 0x6
	keyNumber6     uint16 = 0x7
	keyNumber7     uint16 = 0x8
	keyNumber8     uint16 = 0x9
	keyNumber9     uint16 = 0xA
	keyNumber0     uint16 = 0xB
	keyEsc         uint16 = 0x1
	keyF1          uint16 = 0x3B
	keyF2          uint16 = 0x3C
	keyF3          uint16 = 0x3D
	keyF4          uint16 = 0x3E
	keyF5          uint16 = 0x3F
	keyF6          uint16 = 0x40
	keyF7          uint16 = 0x41
	keyF8          uint16 = 0x42
	keyF9          uint16 = 0x43
	keyF10         uint16 = 0x44
	keyF11         uint16 = 0x57
	keyF12         uint16 = 0x58
	keyTilde       uint16 = 0x29
	keyMinus       uint16 = 0xC
	keyEqual       uint16 = 0xD
	keyBack        uint16 = 0xE
	keyTab         uint16 = 0xF
	keyQ           uint16 = 0x10
	keyW           uint16 = 0x11
	keyE           uint16 = 0x12
	keyR           uint16 = 0x13
	keyT           uint16 = 0x14
	keyY           uint16 = 0x15
	keyU           uint16 = 0x16
	keyI           uint16 = 0x17
	keyO           uint16 = 0x18
	keyP           uint16 = 0x19
	keyLeftSquare  uint16 = 0x1A
	keyRightSquare uint16 = 0x1B
	keyBackslash   uint16 = 0x2B
	keyCapslock    uint16 = 0x3A
	keyA           uint16 = 0x1E
	keyS           uint16 = 0x1F
	keyD           uint16 = 0x20
	keyF           uint16 = 0x21
	keyG           uint16 = 0x22
	keyH           uint16 = 0x23
	keyJ           uint16 = 0x24
	keyK           uint16 = 0x25
	keyL           uint16 = 0x26
	keySemicolon   uint16 = 0x27
	keySingleQuote uint16 = 0x28
	keyEnter       uint16 = 0x1C
	keyLeftShift   uint16 = 0x2A
	keyZ           uint16 = 0x2C
	keyX           uint16 = 0x2D
	keyC           uint16 = 0x2E
	keyV           uint16 = 0x2F
	keyB           uint16 = 0x30
	keyN           uint16 = 0x31
	keyM           uint16 = 0x32
	keyComma       uint16 = 0x33
	keyDot         uint16 = 0x34
	keySlash       uint16 = 0x35
	keyRightShift  uint16 = 0x36
	keyUp          uint16 = 0x67
	keyLeftCtrl    uint16 = 0x1D
	keyWindowsKey  uint16 = 0x7D
	keyLeftAlt     uint16 = 0x38
	keySpace       uint16 = 0x39
	keyRightAlt    uint16 = 0x64
	keyMenu        uint16 = 0x7F
	keyRightCtrl   uint16 = 0x61
	keyLeft        uint16 = 0x69
	keyDown        uint16 = 0x6C
	keyRight       uint16 = 0x6A
	keyIns         uint16 = 0x6E
	keyHome        uint16 = 0x66
	keyPgUp        uint16 = 0x68
	keyDel         uint16 = 0x6F
	keyEnd         uint16 = 0x6B
	keyPgDn        uint16 = 0x6D

	devicePath   []string
	inputActions map[uint8]InputAction
)

type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

// buildInputActions will fill inputActions map with InputAction data
func buildInputActions() {
	inputActions = make(map[uint8]InputAction, 0)

	// Placeholder
	inputActions[None] = InputAction{Name: "None"}

	// Media
	inputActions[VolumeUp] = InputAction{Name: "Volume Up", CommandCode: keyVolumeUp, Media: true}
	inputActions[VolumeDown] = InputAction{Name: "Volume Down", CommandCode: keyVolumeDown, Media: true}
	inputActions[VolumeMute] = InputAction{Name: "Mute", CommandCode: keyVolumeMute, Media: true}
	inputActions[MediaStop] = InputAction{Name: "Stop", CommandCode: keyMediaStop, Media: true}
	inputActions[MediaPrev] = InputAction{Name: "Previous", CommandCode: keyMediaPrev, Media: true}
	inputActions[MediaPlayPause] = InputAction{Name: "Play", CommandCode: keyMediaPlay, Media: true}
	inputActions[MediaNext] = InputAction{Name: "Next", CommandCode: keyMediaNext, Media: true}

	// Numbers
	inputActions[Number0] = InputAction{Name: "Number 0", CommandCode: keyNumber0}
	inputActions[Number1] = InputAction{Name: "Number 1", CommandCode: keyNumber1}
	inputActions[Number2] = InputAction{Name: "Number 2", CommandCode: keyNumber2}
	inputActions[Number3] = InputAction{Name: "Number 3", CommandCode: keyNumber3}
	inputActions[Number4] = InputAction{Name: "Number 4", CommandCode: keyNumber4}
	inputActions[Number5] = InputAction{Name: "Number 5", CommandCode: keyNumber5}
	inputActions[Number6] = InputAction{Name: "Number 6", CommandCode: keyNumber6}
	inputActions[Number7] = InputAction{Name: "Number 7", CommandCode: keyNumber7}
	inputActions[Number8] = InputAction{Name: "Number 8", CommandCode: keyNumber8}
	inputActions[Number9] = InputAction{Name: "Number 9", CommandCode: keyNumber9}

	// Keys
	inputActions[KeyMinus] = InputAction{Name: "Minus (-)", CommandCode: keyMinus}
	inputActions[KeyEqual] = InputAction{Name: "Equals (=)", CommandCode: keyEqual}
	inputActions[KeyQ] = InputAction{Name: "Q", CommandCode: keyQ}
	inputActions[KeyW] = InputAction{Name: "W", CommandCode: keyW}
	inputActions[KeyE] = InputAction{Name: "E", CommandCode: keyE}
	inputActions[KeyR] = InputAction{Name: "R", CommandCode: keyR}
	inputActions[KeyT] = InputAction{Name: "T", CommandCode: keyT}
	inputActions[KeyY] = InputAction{Name: "Y", CommandCode: keyY}
	inputActions[KeyU] = InputAction{Name: "U", CommandCode: keyU}
	inputActions[KeyI] = InputAction{Name: "I", CommandCode: keyI}
	inputActions[KeyO] = InputAction{Name: "O", CommandCode: keyO}
	inputActions[KeyP] = InputAction{Name: "P", CommandCode: keyP}
	inputActions[KeyA] = InputAction{Name: "A", CommandCode: keyA}
	inputActions[KeyS] = InputAction{Name: "S", CommandCode: keyS}
	inputActions[KeyD] = InputAction{Name: "D", CommandCode: keyD}
	inputActions[KeyF] = InputAction{Name: "F", CommandCode: keyF}
	inputActions[KeyG] = InputAction{Name: "G", CommandCode: keyG}
	inputActions[KeyH] = InputAction{Name: "H", CommandCode: keyH}
	inputActions[KeyJ] = InputAction{Name: "J", CommandCode: keyJ}
	inputActions[KeyK] = InputAction{Name: "K", CommandCode: keyK}
	inputActions[KeyL] = InputAction{Name: "L", CommandCode: keyL}
	inputActions[KeyZ] = InputAction{Name: "Z", CommandCode: keyZ}
	inputActions[KeyX] = InputAction{Name: "X", CommandCode: keyX}
	inputActions[KeyC] = InputAction{Name: "C", CommandCode: keyC}
	inputActions[KeyV] = InputAction{Name: "V", CommandCode: keyV}
	inputActions[KeyB] = InputAction{Name: "B", CommandCode: keyB}
	inputActions[KeyN] = InputAction{Name: "N", CommandCode: keyN}
	inputActions[KeyM] = InputAction{Name: "M", CommandCode: keyM}
	inputActions[KeyF1] = InputAction{Name: "F1", CommandCode: keyF1}
	inputActions[KeyF2] = InputAction{Name: "F2", CommandCode: keyF2}
	inputActions[KeyF3] = InputAction{Name: "F3", CommandCode: keyF3}
	inputActions[KeyF4] = InputAction{Name: "F4", CommandCode: keyF4}
	inputActions[KeyF5] = InputAction{Name: "F5", CommandCode: keyF5}
	inputActions[KeyF6] = InputAction{Name: "F6", CommandCode: keyF6}
	inputActions[KeyF7] = InputAction{Name: "F7", CommandCode: keyF7}
	inputActions[KeyF8] = InputAction{Name: "F8", CommandCode: keyF8}
	inputActions[KeyF9] = InputAction{Name: "F9", CommandCode: keyF9}
	inputActions[KeyF10] = InputAction{Name: "F10", CommandCode: keyF10}
	inputActions[KeyF11] = InputAction{Name: "F11", CommandCode: keyF11}
	inputActions[KeyF12] = InputAction{Name: "F12", CommandCode: keyF12}
	inputActions[KeyBack] = InputAction{Name: "Back", CommandCode: keyBack}
	inputActions[KeyTab] = InputAction{Name: "Tab", CommandCode: keyTab}
	inputActions[KeyEsc] = InputAction{Name: "Esc", CommandCode: keyEsc}
	inputActions[KeyTilde] = InputAction{Name: "Tilde (`)", CommandCode: keyTilde}
	inputActions[KeyLeftSquare] = InputAction{Name: "Left Square [{", CommandCode: keyLeftSquare}
	inputActions[KeyRightSquare] = InputAction{Name: "Right Square }]", CommandCode: keyRightSquare}
	inputActions[KeyBackslash] = InputAction{Name: "Backslash (\\)", CommandCode: keyBackslash}
	inputActions[KeyCapslock] = InputAction{Name: "Capslock", CommandCode: keyCapslock}
	inputActions[KeySemicolon] = InputAction{Name: "Semicolon (;)", CommandCode: keySemicolon}
	inputActions[KeySingleQuote] = InputAction{Name: "Single Quote (')", CommandCode: keySingleQuote}
	inputActions[KeyEnter] = InputAction{Name: "Enter", CommandCode: keyEnter}
	inputActions[KeyLeftShift] = InputAction{Name: "Left Shift", CommandCode: keyLeftShift}
	inputActions[KeyComma] = InputAction{Name: "Comma (,)", CommandCode: keyComma}
	inputActions[KeyDot] = InputAction{Name: "Dot (.)", CommandCode: keyDot}
	inputActions[KeySlash] = InputAction{Name: "Slash (/)", CommandCode: keySlash}
	inputActions[KeyRightShift] = InputAction{Name: "Right Shift", CommandCode: keyRightShift}
	inputActions[KeyUp] = InputAction{Name: "Up", CommandCode: keyUp}
	inputActions[KeyLeftCtrl] = InputAction{Name: "Left Ctrl", CommandCode: keyLeftCtrl}
	inputActions[KeyWindowsKey] = InputAction{Name: "Windows Key", CommandCode: keyWindowsKey}
	inputActions[KeyLeftAlt] = InputAction{Name: "Left Alt", CommandCode: keyLeftAlt}
	inputActions[KeySpace] = InputAction{Name: "Space", CommandCode: keySpace}
	inputActions[KeyRightAlt] = InputAction{Name: "Right Alt", CommandCode: keyRightAlt}
	inputActions[KeyMenu] = InputAction{Name: "Menu", CommandCode: keyMenu}
	inputActions[KeyRightCtrl] = InputAction{Name: "Right Ctrl", CommandCode: keyRightCtrl}
	inputActions[KeyLeft] = InputAction{Name: "Left", CommandCode: keyLeft}
	inputActions[KeyDown] = InputAction{Name: "Down", CommandCode: keyDown}
	inputActions[KeyRight] = InputAction{Name: "Right", CommandCode: keyRight}
	inputActions[KeyIns] = InputAction{Name: "Insert", CommandCode: keyIns}
	inputActions[KeyHome] = InputAction{Name: "Home", CommandCode: keyHome}
	inputActions[KeyPgUp] = InputAction{Name: "Pg Up", CommandCode: keyPgUp}
	inputActions[KeyDel] = InputAction{Name: "Delete", CommandCode: keyDel}
	inputActions[KeyEnd] = InputAction{Name: "End", CommandCode: keyEnd}
	inputActions[KeyPgDn] = InputAction{Name: "Pg Dn", CommandCode: keyPgDn}
}

// Init will fetch an input device
func Init() {
	devicePath = findDevice()
	buildInputActions()
}

// GetMediaKeys will return a map of InputAction for Media keys
func GetMediaKeys() map[uint8]InputAction {
	keys := make(map[uint8]InputAction, 0)
	for key, value := range inputActions {
		if value.Media {
			keys[key] = value
		}
	}
	return keys
}

// GetInputKeys will return a map of InputAction for non-media keys
func GetInputKeys() map[uint8]InputAction {
	keys := make(map[uint8]InputAction, 0)
	for key, value := range inputActions {
		if value.Media {
			continue
		}
		keys[key] = value
	}
	return keys
}

// GetInputActions will return a map of InputAction
func GetInputActions() map[uint8]InputAction {
	return inputActions
}

// InputControl will emulate volume control keys
func InputControl(controlType uint8, serial string) {
	// Get a device path
	path := getDevicePathBySerial(serial)

	// Open device
	device := openDevice(path)
	if device == nil {
		return
	}

	var events []inputEvent

	// Get event key code
	actionType := getInputAction(controlType)
	if actionType == nil {
		return
	}

	// Create events
	events = createInputEvent(actionType.CommandCode)

	// Send events
	for _, event := range events {
		if err := emitEvent(device, event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit event")
			return
		}
	}

	// Close device
	closeDevice(device)
}

// getInputAction will return InputAction based on actionType
func getInputAction(actionType uint8) *InputAction {
	if action, ok := inputActions[actionType]; ok {
		return &action
	}
	return nil
}

// getDevicePathBySerial will return a device path by serial number
func getDevicePathBySerial(serial string) string {
	if devicePath != nil {
		for _, value := range devicePath {
			if strings.Contains(value, serial) {
				return value
			}
		}
	}
	return ""
}

// emitEvent will send an event toward the device
func emitEvent(file *os.File, event inputEvent) error {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, event); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Failed to serialize event")
		return err
	}

	if _, err := file.Write(buf.Bytes()); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Failed to write event")
		return err
	}
	return nil
}

// createInputEvent will create a list of input events
func createInputEvent(code uint16) []inputEvent {
	// Create an input event for key press
	keyPress := inputEvent{
		Type:  evKey,
		Code:  code,
		Value: 1, // Key press
	}

	// Create an input event for key release
	keyRelease := inputEvent{
		Type:  evKey,
		Code:  code,
		Value: 0, // Key release
	}

	// Synchronization event
	syncEvent := inputEvent{
		Type:  evSyn,
		Code:  0,
		Value: 0,
	}

	// Emit the events
	events := []inputEvent{keyPress, syncEvent, keyRelease, syncEvent}
	return events
}

// findDevice will find a Corsair keyboard input device
func findDevice() []string {
	var devices []string
	vendor := "corsair"
	// Path to the directory we want to scan
	dir := "/dev/input/by-id/"

	// Read the directory contents
	files, err := os.ReadDir(dir)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "directory": dir}).Error("Error reading directory")
		return nil
	}

	// Loop through the files and filter the ones matching *-kbd
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if matched, _ := filepath.Match("*-kbd", file.Name()); matched {
			path := filepath.Join(dir, file.Name())
			if strings.Contains(strings.ToLower(path), strings.ToLower(vendor)) {
				devices = append(devices, path)
			}
		}
	}
	return devices
}

// openDevice will open input device
func openDevice(path string) *os.File {
	file, err := os.OpenFile(path, os.O_WRONLY, 0666)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "device": path}).Error("Unable to open input device")
		return nil
	}
	return file
}

// closeDevice will close an input device
func closeDevice(f *os.File) {
	if f != nil {
		err := f.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close input device")
			return
		}
	}
}
