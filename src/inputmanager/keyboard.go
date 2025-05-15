package inputmanager

import (
	"OpenLinkHub/src/logger"
	"os"
	"syscall"
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
	for _, event := range events {
		if err := writeVirtualEvent(virtualKeyboardFile, &event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit event")
			return
		}
	}
}
