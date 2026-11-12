package openrgb

import (
	"testing"

	"OpenLinkHub/src/common"
)

func TestRemoveDeviceControllerBySerial(t *testing.T) {
	previousEnabled := enabled
	previousControllers := controllers
	previousConn := conn
	t.Cleanup(func() {
		enabled = previousEnabled
		controllers = previousControllers
		conn = previousConn
	})

	enabled = true
	conn = nil
	// Three entries, with the same serial first and last, is the list that
	// panicked on hardware: slice bounds out of range [3:2].
	controllers = []*common.OpenRGBController{
		{Serial: "keyboard"},
		{Serial: "other"},
		{Serial: "keyboard"},
	}

	RemoveDeviceControllerBySerial("keyboard")

	if len(controllers) != 1 {
		t.Fatalf("len = %d, want 1", len(controllers))
	}
	if controllers[0].Serial != "other" {
		t.Fatalf("remaining serial = %q, want other", controllers[0].Serial)
	}

	RemoveDeviceControllerBySerial("missing")
	if len(controllers) != 1 || controllers[0].Serial != "other" {
		t.Fatalf("missing serial changed the list: %+v", controllers)
	}
}
