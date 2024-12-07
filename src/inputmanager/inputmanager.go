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
	VolumeUp   uint8 = 0
	VolumeDown uint8 = 1
	VolumeMute uint8 = 2
)

var (
	evKey         uint16 = 0x01
	evSyn         uint16 = 0x00
	keyVolumeUp   uint16 = 0x73
	keyVolumeDown uint16 = 0x72
	keyVolumeMute uint16 = 0x71
	devicePath           = ""
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

// VolumeControl will emulate volume control keys
func VolumeControl(controlType uint8) {
	// Open device
	device := openDevice()
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
func findDevice() string {
	vendor := "corsair"
	// Path to the directory we want to scan
	dir := "/dev/input/by-id/"

	// Read the directory contents
	files, err := os.ReadDir(dir)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "directory": dir}).Error("Error reading directory")
		return ""
	}

	// Loop through the files and filter the ones matching *-kbd
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if matched, _ := filepath.Match("*-kbd", file.Name()); matched {
			path := filepath.Join(dir, file.Name())
			if strings.Contains(strings.ToLower(path), strings.ToLower(vendor)) {
				return path
			}
		}
	}
	return ""
}

// openDevice will open input device
func openDevice() *os.File {
	file, err := os.OpenFile(devicePath, os.O_WRONLY, 0666)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "device": devicePath}).Error("Unable to open input device")
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
