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

const (
	VolumeUp       uint8 = 0
	VolumeDown     uint8 = 1
	VolumeMute     uint8 = 2
	MediaStop      uint8 = 3
	MediaPrev      uint8 = 4
	MediaPlayPause uint8 = 5
	MediaNext      uint8 = 6
	Number1        uint8 = 7
	Number2        uint8 = 8
	Number3        uint8 = 9
	Number4        uint8 = 10
	Number5        uint8 = 11
	Number6        uint8 = 12
	Number7        uint8 = 13
	Number8        uint8 = 14
	Number9        uint8 = 15
	Number10       uint8 = 16
	Number11       uint8 = 17
	Number12       uint8 = 18
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
	keyNumber0     uint16 = 0xB
	keyNumber1     uint16 = 0x2
	keyNumber2     uint16 = 0x3
	keyNumber3     uint16 = 0x4
	keyNumber4     uint16 = 0x5
	keyNumber5     uint16 = 0x6
	keyNumber6     uint16 = 0x7
	keyNumber7     uint16 = 0x8
	keyNumber8     uint16 = 0x9
	keyNumber9     uint16 = 0xA
	keyNumber10    uint16 = 0xB
	keyNumber11    uint16 = 0xC
	keyNumber12    uint16 = 0xD
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
	keyEquals      uint16 = 0xD
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
	devicePath     []string
)

type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

// Init will fetch an input device
func Init() {
	devicePath = findDevice()
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
	switch controlType {
	case 0:
		events = createInputEvent(keyVolumeUp)
		break
	case 1:
		events = createInputEvent(keyVolumeDown)
		break
	case 2:
		events = createInputEvent(keyVolumeMute)
		break
	case 3:
		events = createInputEvent(keyMediaStop)
		break
	case 4:
		events = createInputEvent(keyMediaPrev)
		break
	case 5:
		events = createInputEvent(keyMediaPlay)
		break
	case 6:
		events = createInputEvent(keyMediaNext)
		break
	case 7:
		events = createInputEvent(keyNumber1)
		break
	case 8:
		events = createInputEvent(keyNumber2)
		break
	case 9:
		events = createInputEvent(keyNumber3)
		break
	case 10:
		events = createInputEvent(keyNumber4)
		break
	case 11:
		events = createInputEvent(keyNumber5)
		break
	case 12:
		events = createInputEvent(keyNumber6)
		break
	case 13:
		events = createInputEvent(keyNumber7)
		break
	case 14:
		events = createInputEvent(keyNumber8)
		break
	case 15:
		events = createInputEvent(keyNumber9)
		break
	case 16:
		events = createInputEvent(keyNumber10)
		break
	case 17:
		events = createInputEvent(keyNumber11)
		break
	case 18:
		events = createInputEvent(keyNumber12)
		break
	}

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
