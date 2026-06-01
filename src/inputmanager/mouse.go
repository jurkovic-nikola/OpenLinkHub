package inputmanager

// Package: inputmanager
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/display"
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
			logger.Log(logger.Fields{"error": errno}).Error("Failed to destroy virtual mouse")
		}

		if err := virtualMouseFile.Close(); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to close /dev/uinput")
			return
		}
	}

	if virtualMouseAbsPointer != 0 {
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiDevDestroy, 0); errno != 0 {
			logger.Log(logger.Fields{"error": errno}).Error("Failed to destroy virtual absolute mouse")
		}

		if err := virtualMouseAbsFile.Close(); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to close /dev/uinput")
			return
		}
	}
}

// createVirtualMouseAbs will create a separate absolute-position virtual mouse device.
func createVirtualMouseAbs(vendorId, productId uint16) error {
	f, err := os.OpenFile("/dev/uinput", os.O_WRONLY, 0660)
	if err != nil {
		return err
	}

	virtualMouseAbsFile = f
	virtualMouseAbsPointer = f.Fd()

	uInputDevice := uInputUserDev{
		ID: inputID{
			BusType: 0x03, // BUS_USB
			Vendor:  vendorId,
			Product: productId,
			Version: 1,
		},
	}

	copy(uInputDevice.Name[:], "OpenLinkHub Virtual Absolute Mouse")

	uInputDevice.AbsMin[AbsX] = AbsMin
	uInputDevice.AbsMax[AbsX] = int32(display.GetScreenResolution().Width)
	uInputDevice.AbsMin[AbsY] = AbsMin
	uInputDevice.AbsMax[AbsY] = int32(display.GetScreenResolution().Height)

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiSetEvbit, uintptr(EvKey)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable absolute mouse key events")
		return errno
	}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiSetEvbit, uintptr(EvAbs)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable absolute mouse absolute events")
		return errno
	}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiSetEvbit, uintptr(EvSyn)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable absolute mouse sync events")
		return errno
	}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiSetAbsbit, uintptr(AbsX)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable absolute mouse ABS_X")
		return errno
	}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiSetAbsbit, uintptr(AbsY)); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to enable absolute mouse ABS_Y")
		return errno
	}

	for _, code := range []uint16{btnLeft, btnRight, btnMiddle, btnBack, btnForward} {
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiSetKeybit, uintptr(code)); errno != 0 {
			logger.Log(logger.Fields{"error": errno, "code": code}).Error("Failed to enable absolute mouse button event")
			return errno
		}
	}

	if _, err := f.Write((*(*[unsafe.Sizeof(uInputDevice)]byte)(unsafe.Pointer(&uInputDevice)))[:]); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Failed to write absolute mouse uinput struct")
		return err
	}

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMouseAbsPointer, UiDevCreate, 0); errno != 0 {
		logger.Log(logger.Fields{"error": errno}).Error("Failed to create absolute virtual mouse")
		return errno
	}

	return nil
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

	// Enable RelHWheel events
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, virtualMousePointer, UiSetRelbit, uintptr(RelHWheel)); errno != 0 {
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

// InputControlScroll will trigger vertical scroll (up / down)
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

// InputControlScrollHorizontal will trigger horizontal scroll (left / right)
func InputControlScrollHorizontal(left bool) {
	if virtualMouseFile == nil {
		logger.Log(logger.Fields{}).Error("Virtual mouse is not present")
		return
	}

	value := int32(-1)
	if left {
		value = 1
	}

	scrollEvent := inputEvent{
		Type:  evRel,
		Code:  relHWheel,
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

// InputControlMove will move the virtual mouse by the given relative offsets (x, y)
func InputControlMove(x, y int32) {
	if virtualMouseFile == nil {
		logger.Log(logger.Fields{}).Error("Virtual mouse is not present")
		return
	}

	var events []inputEvent

	if x != 0 {
		events = append(events, inputEvent{
			Type:  evRel,
			Code:  RelX,
			Value: x,
		})
	}
	if y != 0 {
		events = append(events, inputEvent{
			Type:  evRel,
			Code:  RelY,
			Value: y,
		})
	}

	events = append(events, inputEvent{
		Type:  evSyn,
		Code:  0,
		Value: 0,
	})

	for _, event := range events {
		if err := writeVirtualEvent(virtualMouseFile, &event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit move event")
			return
		}
	}
}

// InputControlMoveAbsolute will move the virtual mouse to absolute coordinates.
func InputControlMoveAbsolute(x, y int32) {
	if virtualMouseAbsFile == nil {
		logger.Log(logger.Fields{}).Error("Virtual mouse is not present")
		return
	}

	if screenWidth <= 1 || screenHeight <= 1 {
		logger.Log(logger.Fields{"width": screenWidth, "height": screenHeight}).Error("Invalid screen size")
		return
	}

	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x >= screenWidth {
		x = screenWidth - 1
	}
	if y >= screenHeight {
		y = screenHeight - 1
	}

	tmpX := x
	tmpY := y

	if tmpX < screenWidth {
		tmpX++
	} else {
		tmpX--
	}

	if tmpY < screenHeight {
		tmpY++
	} else {
		tmpY--
	}

	events := []inputEvent{
		{Type: evAbs, Code: AbsX, Value: tmpX},
		{Type: evAbs, Code: AbsY, Value: tmpY},
		{Type: evSyn, Code: 0, Value: 0},
		{Type: evAbs, Code: AbsX, Value: x},
		{Type: evAbs, Code: AbsY, Value: y},
		{Type: evSyn, Code: 0, Value: 0},
	}

	for _, event := range events {
		if err := writeVirtualEvent(virtualMouseAbsFile, &event); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to emit absolute move event")
			return
		}
	}
}

// GetVirtualMouse will return virtual mouse pointer
func GetVirtualMouse() *os.File {
	return virtualMouseFile
}
