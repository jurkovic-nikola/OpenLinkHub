package inputmanager

import (
	"OpenLinkHub/src/logger"
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	VolumeUp       uint8 = 0
	VolumeDown     uint8 = 1
	VolumeMute     uint8 = 2
	MediaStop      uint8 = 3
	MediaPrev      uint8 = 4
	MediaPlayPause uint8 = 5
	MediaNext      uint8 = 6
)

var (
	evKey         uint16 = 0x01
	evSyn         uint16 = 0x00
	keyVolumeUp   uint16 = 0x73
	keyVolumeDown uint16 = 0x72
	keyVolumeMute uint16 = 0x71
	keyMediaStop  uint16 = 0xA6
	keyMediaPrev  uint16 = 0xA5
	keyMediaPlay  uint16 = 0xA4
	keyMediaNext  uint16 = 0xA3
	devicePath    []string
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
	}

	// Send events
	for _, event := range events {
		if err := emitEvent(device, event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit event")
			return
		}
		time.Sleep(10 * time.Millisecond) // Small delay for realism
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
	// Create an input event for F11 key press
	keyPress := inputEvent{
		Type:  evKey,
		Code:  code,
		Value: 1, // Key press
	}

	// Create an input event for F11 key release
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
