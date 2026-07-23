package devices

import (
	"LumenForge/src/common"
	"LumenForge/src/devices/openrgbimport"
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

func TestOpenRGBImportRegistryHelpersUseExactInstance(t *testing.T) {
	mutex.Lock()
	previousDevices := devices
	devices = make(map[string]*common.Device)
	mutex.Unlock()
	t.Cleanup(func() {
		mutex.Lock()
		devices = previousDevices
		mutex.Unlock()
	})

	serial := "openrgb-hash-registry-test"
	instance := &openrgbimport.Device{Serial: serial, Product: "Imported"}
	wrapper := &common.Device{
		Serial:      serial,
		Product:     "Imported",
		Firmware:    "1.2.3",
		Image:       "imported.svg",
		Instance:    instance,
		Unavailable: true,
	}
	if err := RegisterOpenRGBImport(wrapper, instance); err != nil {
		t.Fatal(err)
	}
	if !wrapper.Unavailable {
		t.Fatal("registered importer wrapper availability changed unexpectedly")
	}

	if err := RegisterOpenRGBImport(wrapper, instance); err != nil {
		t.Fatalf("same-wrapper idempotent register: %v", err)
	}
	differentWrapper := &common.Device{Serial: serial, Product: "Snapshot", Instance: instance}
	if err := RegisterOpenRGBImport(differentWrapper, instance); err == nil {
		t.Fatal("different wrapper for the same importer instance was accepted")
	}
	lookedUp, lookedUpInstance, ok := LookupOpenRGBImport(serial)
	if !ok || lookedUpInstance != instance || lookedUp == wrapper {
		t.Fatalf("lookup = %#v, %p, %v", lookedUp, lookedUpInstance, ok)
	}
	lookedUp.Product = "Mutated Snapshot"
	if GetDevices()[serial].Product != "Imported" {
		t.Fatal("mutating lookup snapshot changed the registry")
	}

	replacement := &openrgbimport.Device{Serial: serial, Product: "Replacement"}
	if err := RegisterOpenRGBImport(&common.Device{Serial: serial, Instance: replacement}, replacement); err == nil {
		t.Fatal("different importer instance collision was accepted")
	}
	if removed, ok := RemoveOpenRGBImport(serial, replacement); ok || removed != nil {
		t.Fatal("pointer-mismatch removal succeeded")
	}
	removed, ok := RemoveOpenRGBImport(serial, instance)
	if !ok || removed != wrapper {
		t.Fatal("exact importer instance was not removed")
	}
	if removed.Product != "Imported" || removed.Firmware != "1.2.3" ||
		removed.Image != "imported.svg" || !removed.Unavailable ||
		removed.Instance != instance {
		t.Fatalf("removed wrapper fields changed: %#v", removed)
	}
	if _, _, ok := LookupOpenRGBImport(serial); ok {
		t.Fatal("removed importer still exists")
	}
}
