package inputmanager

import (
	"OpenLinkHub/src/logger"
	"os"
	"syscall"
	"time"
	"unsafe"
)

// destroyVirtualMouse will destroy virtual mouse and close uinput device
func destroyVirtualMouse() {
	if virtualMousePointer != 0 {
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiDevDestroy, 0); errno != 0 {
			logger.Log(logger.Fields{"error": errno}).Error("Failed to destroy virtual keyboard")
		}

		if err := virtualMouseFile.Close(); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to close /dev/uinput")
			return
		}
	}
}

// createVirtualMouse will create new virtual mouse input device
func createVirtualMouse(vendorId, productId uint16) error {
	f, err := os.OpenFile("/dev/uinput", os.O_WRONLY, 0660)
	if err != nil {
		return err
	}
	virtualMouseFile = f
	virtualMousePointer = f.Fd()

	// Define device info
	uInputDevice := uInputUserDev{
		ID: inputID{
			BusType: 0x03, // BUS_USB
			Vendor:  vendorId,
			Product: productId,
			Version: 1,
		},
	}

	// Set mouse name
	copy(uInputDevice.Name[:], "OpenLinkHub Virtual Mouse")

	// Ensure all required key event properties are enabled
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiSetEvbit, uintptr(EvKey)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable key events")
		return errno
	}

	// Enable rel events
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiSetEvbit, uintptr(EvRel)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable key events")
		return errno
	}

	// Enable sync events
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiSetEvbit, uintptr(EvSyn)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable key events")
		return errno
	}

	// Enable RelX events
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiSetRelbit, uintptr(RelX)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable key events")
		return errno
	}

	// Enable RelY events
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiSetRelbit, uintptr(RelY)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable key events")
		return errno
	}

	// Enable RelWheel events
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiSetRelbit, uintptr(RelWheel)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable key events")
		return errno
	}

	// Enable button events
	for _, code := range []uint16{btnLeft, btnRight, btnMiddle, btnBack, btnForward} {
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiSetKeybit, uintptr(code)); errno != 0 {
			logger.Log(logger.Fields{"error": errno}).Error("Failed to enable key events")
			return errno
		}
	}

	// Enable keyboard keys used for zoom (Ctrl)
	for _, code := range []uint16{keyLeftCtrl, keyRightCtrl, keyLeftAlt, keyLeftShift} {
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiSetKeybit, uintptr(code)); errno != 0 {
			logger.Log(logger.Fields{"error": errno, "code": code}).Error("Failed to enable keyboard key event")
			return errno
		}
	}
	
	// Ensure we correctly write the struct before creating the device
	if _, e := f.Write((*(*[unsafe.Sizeof(uInputDevice)]byte)(unsafe.Pointer(&uInputDevice)))[:]); e != nil {
		logger.Log(logger.Fields{"error": e}).Error("Failed to write virtual mouse data struct")
		return e
	}

	// Ensure device is created
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiDevCreate, 0); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to create new mouse keyboard")
		return errno
	}
	return nil
}

// InputControlMouse will emulate input events based on virtual mouse
func InputControlMouse(controlType uint16) {
	if virtualMouseFile == nil {
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
	events = createInputEvent(actionType.CommandCode, false)

	// Send events
	for _, event := range events {
		if err := writeVirtualEvent(virtualMouseFile, &event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit event")
			return
		}
	}
}

// InputControlScroll will trigger scroll up / down
func InputControlScroll(up bool) {
	if virtualMouseFile == nil {
		logger.Log(logger.Fields{}).Error("Virtual mouse is not present")
		return
	}

	value := int32(-1)
	if up {
		value = 1
	}

	scrollEvent := inputEvent{
		Type:  evRel,
		Code:  relWheel,
		Value: value,
	}

	syncEvent := inputEvent{
		Type:  evSyn,
		Code:  0,
		Value: 0,
	}

	events := []inputEvent{scrollEvent, syncEvent}

	for _, event := range events {
		if err := writeVirtualEvent(virtualMouseFile, &event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit scroll event")
			return
		}
	}
}

// InputControlZoom will trigger zoom
func InputControlZoom(in bool) {
	if virtualMouseFile == nil {
		logger.Log(nil).Error("Virtual mouse is not present")
		return
	}

	scrollValue := int32(1)
	if !in {
		scrollValue = -1
	}

	pressCtrl := inputEvent{Type: evKey, Code: keyLeftCtrl, Value: 1}
	sync := inputEvent{Type: evSyn, Code: 0, Value: 0}
	scroll := inputEvent{Type: evRel, Code: relWheel, Value: scrollValue}
	releaseCtrl := inputEvent{Type: evKey, Code: keyLeftCtrl, Value: 0}

	events := []inputEvent{
		pressCtrl, sync,
		scroll, sync,
		releaseCtrl, sync,
	}

	for _, e := range events {
		if err := writeVirtualEvent(virtualMouseFile, &e); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit zoom event")
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// InputControlMouseHold will emulate input events based on virtual mouse and button hold.
// Experimental for now
func InputControlMouseHold(controlType uint16, press bool) {
	if virtualMouseFile == nil {
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
		if err := writeVirtualEvent(virtualMouseFile, &event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit event")
			return
		}
	}
}
