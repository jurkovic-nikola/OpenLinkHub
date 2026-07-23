package devices

import (
	"LumenForge/src/common"
	"testing"
)

type registryTestDevice struct{}

func (*registryTestDevice) SnapshotCount() int {
	return len(GetDevices())
}

func TestGetDevicesReturnsWrapperSnapshot(t *testing.T) {
	instance := &registryTestDevice{}
	mutex.Lock()
	previousDevices := devices
	devices = map[string]*common.Device{
		"test-device": {
			Product:     "Test Device",
			Serial:      "test-device",
			Instance:    instance,
			Unavailable: true,
		},
	}
	mutex.Unlock()
	t.Cleanup(func() {
		mutex.Lock()
		devices = previousDevices
		mutex.Unlock()
	})

	snapshot := GetDevices()
	snapshot["test-device"].Unavailable = false
	delete(snapshot, "test-device")

	secondSnapshot := GetDevices()
	if len(secondSnapshot) != 1 || !secondSnapshot["test-device"].Unavailable {
		t.Fatalf("mutating snapshot changed registry: %#v", secondSnapshot)
	}

	setDeviceAvailability("test-device", false)
	if GetDevices()["test-device"].Unavailable {
		t.Fatal("availability helper did not update registry wrapper")
	}
	setDevicePresentation("test-device", "Updated Product", "2.0", "updated.svg")
	presented := GetDevices()["test-device"]
	if presented.Product != "Updated Product" || presented.Firmware != "2.0" || presented.Image != "updated.svg" {
		t.Fatalf("presentation helper did not update registry wrapper: %#v", presented)
	}

	result := CallDeviceMethod("test-device", "SnapshotCount")
	if len(result) != 1 || result[0].Int() != 1 {
		t.Fatalf("reentrant registry method result = %#v", result)
	}
}
