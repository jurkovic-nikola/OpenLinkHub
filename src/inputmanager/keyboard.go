package inputmanager

import (
	"OpenLinkHub/src/logger"
	"os"
	"syscall"
	"time"
	"unsafe"
)

// destroyVirtualKeyboard will destroy virtual keyboard and close uinput device
func destroyVirtualKeyboard() {
	if virtualKeyboardPointer != 0 {
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualKeyboardPointer, UiDevDestroy, 0); errno != 0 {
			logger.Log(logger.Fields{"error": errno}).Error("Failed to destroy virtual keyboard")
		}

		if err := virtualKeyboardFile.Close(); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to close /dev/uinput")
			return
		}
	}
}

// createVirtualKeyboard will create new virtual keyboard input device
func createVirtualKeyboard(vendorId, productId uint16) error {
	// Open device
	f, err := os.OpenFile("/dev/uinput", os.O_WRONLY, 0660)
	if err != nil {
		virtualKeyboardFile = nil
		logger.Log(logger.Fields{"error": err}).Error("Failed to open /dev/uinput")
		return err
	}
	virtualKeyboardFile = f

	// Set non-blocking mode
	virtualKeyboardPointer = virtualKeyboardFile.Fd()
	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, virtualKeyboardPointer, syscall.F_SETFL, syscall.O_NONBLOCK)
	if errno != 0 {
		logger.Log(logger.Fields{"error": err}).Error("Unable to set non-blocking mode")
	}

	// Define virtual device
	uInputDevice := uInputUserDev{
		ID: inputID{
			BusType: 0x06, // BUS_VIRTUAL
			Vendor:  vendorId,
			Product: productId,
			Version: 1,
		},
		FFEffects: 0,
	}

	// Set keyboard name
	copy(uInputDevice.Name[:], "OpenLinkHub Virtual Keyboard")

	// Ensure all required key event properties are enabled
	if _, _, errno = syscall.Syscall(syscall.SYS_IOCTL, virtualKeyboardPointer, UiSetEvbit, uintptr(EvKey)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable key events")
		return err
	}

	// Enable sync events
	if _, _, errno = syscall.Syscall(syscall.SYS_IOCTL, virtualKeyboardPointer, UiSetEvbit, uintptr(EvSyn)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable sync events")
		return errno
	}

	// Enable standard keyboard keys (letters, numbers, function keys)
	for i := 0; i < 255; i++ {
		if _, _, errno = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), UiSetKeybit, uintptr(i)); errno != 0 {
			logger.Log(logger.Fields{"error": errno, "key": i}).Error("Failed to enable key")
			continue
		}
	}

	// Ensure we correctly write the struct before creating the device
	if _, e := f.Write((*(*[unsafe.Sizeof(uInputDevice)]byte)(unsafe.Pointer(&uInputDevice)))[:]); e != nil {
		logger.Log(logger.Fields{"error": e}).Error("Failed to write virtual keyboard data struct")
		return e
	}

	// Ensure device is created
	if _, _, errno = syscall.Syscall(syscall.SYS_IOCTL, virtualKeyboardPointer, UiDevCreate, 0); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to create new virtual keyboard")
		return errno
	}

	return nil
}

// InputControlKeyboard will emulate input events based on virtual keyboard
func InputControlKeyboard(controlType uint16, hold bool) {
	if virtualKeyboardFile == nil {
		logger.Log(logger.Fields{}).Error("Virtual keyboard is not present")
		return
	}

	var events []inputEvent

	// Get event key code
	actionType := getInputAction(controlType)
	if actionType == nil {
		return
	}

	// Create events
	events = createInputEvent(actionType.CommandCode, hold)

	// Send events
	for i, event := range events {
		if err := writeVirtualEvent(virtualKeyboardFile, &event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit event")
			return
		}
		if i == 1 && !hold && len(events) > 2 {
			// Delay rapid events
			time.Sleep(20 * time.Millisecond)
		}
	}
}

// InputControlKeyboardHold will emulate input events based on virtual keyboard
func InputControlKeyboardHold(controlType uint16, press bool) {
	if virtualKeyboardFile == nil {
		logger.Log(logger.Fields{}).Error("Virtual keyboard is not present")
		return
	}

	var events []inputEvent

	// Get event key code
	actionType := getInputAction(controlType)
	if actionType == nil {
		return
	}

	// Create events
	events = createInputEventHold(actionType.CommandCode, press)

	// Send events
	for _, event := range events {
		if err := writeVirtualEvent(virtualKeyboardFile, &event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit event")
			return
		}
	}
}

// InputControlCtrlEnd emulates Ctrl + End
func InputControlCtrlEnd() {
	if virtualKeyboardFile == nil {
		logger.Log(logger.Fields{}).Error("Virtual keyboard is not present")
		return
	}

	var events []inputEvent

	events = append(events, createInputEventHold(keyLeftCtrl, true)...)
	events = append(events, createInputEvent(keyEnd, false)...)
	events = append(events, createInputEventHold(keyLeftCtrl, false)...)

	for _, event := range events {
		if err := writeVirtualEvent(virtualKeyboardFile, &event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit Ctrl+End")
			return
		}
	}
}

// InputControlZoomReset will reset zoom to default (Ctrl + 0)
func InputControlZoomReset() {
	if virtualKeyboardFile == nil {
		logger.Log(nil).Error("Virtual keyboard is not present")
		return
	}

	pressCtrl := inputEvent{
		Type:  evKey,
		Code:  keyLeftCtrl,
		Value: 1,
	}
	pressZero := inputEvent{
		Type:  evKey,
		Code:  keyNumber0,
		Value: 1,
	}
	releaseZero := inputEvent{
		Type:  evKey,
		Code:  keyNumber0,
		Value: 0,
	}
	releaseCtrl := inputEvent{
		Type:  evKey,
		Code:  keyLeftCtrl,
		Value: 0,
	}
	sync := inputEvent{
		Type:  evSyn,
		Code:  0,
		Value: 0,
	}

	events := []inputEvent{
		pressCtrl, sync,
		pressZero, sync,
		releaseZero, sync,
		releaseCtrl, sync,
	}

	for _, e := range events {
		if err := writeVirtualEvent(virtualKeyboardFile, &e); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit zoom reset")
			return
		}
	}
}

// InputControlScrollHorizontalReset moves view to the far left (Home)
func InputControlScrollHorizontalReset() {
	if virtualKeyboardFile == nil {
		logger.Log(nil).Error("Virtual keyboard is not present")
		return
	}

	pressHome := inputEvent{
		Type:  evKey,
		Code:  keyHome,
		Value: 1,
	}
	releaseHome := inputEvent{
		Type:  evKey,
		Code:  keyHome,
		Value: 0,
	}
	sync := inputEvent{
		Type:  evSyn,
		Code:  0,
		Value: 0,
	}

	events := []inputEvent{
		pressHome, sync,
		releaseHome, sync,
	}

	for _, e := range events {
		if err := writeVirtualEvent(virtualKeyboardFile, &e); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to reset horizontal scroll")
			return
		}
	}
}

// InputControlKeyboardText will send given text string to keyboard
func InputControlKeyboardText(text string) {
	if virtualKeyboardFile == nil {
		logger.Log(logger.Fields{}).Error("Virtual keyboard is not present")
		return
	}

	for _, char := range text {
		useShift := false
		baseChar := char

		// Check if the char needs Shift
		if shifted, exists := shiftedToBase[char]; exists {
			baseChar = shifted
			useShift = true
		}

		// Lookup key code using baseChar
		keyCode, ok := charToKey[baseChar]
		if !ok {
			continue // skip unknown characters
		}

		// Press Shift if needed
		if useShift {
			err := writeVirtualEvent(virtualKeyboardFile, &inputEvent{Type: evKey, Code: keyLeftShift, Value: 1})
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to write virtual keyboard data struct for Shift")
				return
			}
			err = writeVirtualEvent(virtualKeyboardFile, &inputEvent{Type: evSyn, Code: 0, Value: 0})
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to write virtual keyboard data struct for Shift")
				return
			}
		}

		// Key press
		err := writeVirtualEvent(virtualKeyboardFile, &inputEvent{Type: evKey, Code: keyCode, Value: 1})
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to write virtual keyboard data struct for key press")
			return
		}
		err = writeVirtualEvent(virtualKeyboardFile, &inputEvent{Type: evSyn, Code: 0, Value: 0})
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to write virtual keyboard data struct for key press")
			return
		}
		time.Sleep(20 * time.Millisecond)

		// Key release
		err = writeVirtualEvent(virtualKeyboardFile, &inputEvent{Type: evKey, Code: keyCode, Value: 0})
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to write virtual keyboard data struct for key release")
			return
		}
		err = writeVirtualEvent(virtualKeyboardFile, &inputEvent{Type: evSyn, Code: 0, Value: 0})
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to write virtual keyboard data struct for key release")
			return
		}

		// Release Shift
		if useShift {
			err = writeVirtualEvent(virtualKeyboardFile, &inputEvent{Type: evKey, Code: keyLeftShift, Value: 0})
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to write virtual keyboard data struct for Shift release")
				return
			}
			err = writeVirtualEvent(virtualKeyboardFile, &inputEvent{Type: evSyn, Code: 0, Value: 0})
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to write virtual keyboard data struct for Shift release")
				return
			}
		}

		time.Sleep(40 * time.Millisecond) // normal typing rhythm
	}
}
