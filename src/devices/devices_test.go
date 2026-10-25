package devices

import (
	"OpenLinkHub/src/common"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTakeSerialByDevPath(t *testing.T) {
	devPath := "/devices/pci0000:00/0000:00:14.0/usb1/1-2"
	usbDevPaths[devPath] = usbDevice{serial: "abc123", key: "7199"}
	t.Cleanup(func() { delete(usbDevPaths, devPath) })

	serial, ok := TakeSerialByDevPath(devPath, 7199)
	if !ok || serial != "abc123" {
		t.Fatalf("got (%q, %v), want (\"abc123\", true)", serial, ok)
	}
	// The mapping is consumed so a later replug on the same port registers fresh
	if _, ok = TakeSerialByDevPath(devPath, 7199); ok {
		t.Fatal("mapping should be removed after it is taken")
	}
}

// holder is keyHolder asked by a device with no HID path
func holder(key string) (string, bool) { return keyHolder(key, "") }

// Two identical PSUs (e.g. two HX1500i) have no USB serial. They must get distinct keys at startup
// and on hotplug, and a replugged PSU must get its old key back.
func TestSerialLessKeyKeepsIdenticalDevicesApart(t *testing.T) {
	const portA = "/devices/pci0000:00/0000:00:14.0/usb1/1-2"
	const portB = "/devices/pci0000:00/0000:00:14.0/usb1/1-3"

	usePlainKeyPorts(t)

	// Startup: the second PSU finds the first in the device list
	list := map[string]string{}
	inList := func(k string) (string, bool) { p, ok := list[k]; return p, ok }
	keyA := serialLessKey(7199, portA, "/dev/hidraw0", inList)
	list[keyA] = portA
	keyB := serialLessKey(7199, portB, "/dev/hidraw1", inList)
	if keyA != "7199" || keyB != "7199-1-3" {
		t.Fatalf("startup keys = %q, %q; want \"7199\", \"7199-1-3\"", keyA, keyB)
	}
	// The first PSU's port now holds the plain key
	if plainKeyPorts["7199"] != "1-2" {
		t.Fatalf("plain key port = %q, want \"1-2\"", plainKeyPorts["7199"])
	}
	// Port unknown: fall back to the HID path
	if k := serialLessKey(7199, "", "/dev/hidraw1", inList); k != "7199-hidraw1" {
		t.Fatalf("key without port = %q, want \"7199-hidraw1\"", k)
	}
	// Neither port known: the holder's unknown port is not this device's
	unknown := func(string) (string, bool) { return "", true }
	if k := serialLessKey(7203, "", "/dev/hidraw2", unknown); k != "7203-hidraw2" {
		t.Fatalf("key with both ports unknown = %q, want \"7203-hidraw2\"", k)
	}

	usbDevPaths[portA] = usbDevice{serial: "serialA", key: keyA}
	usbDevPaths[portB] = usbDevice{serial: "serialB", key: keyB}
	t.Cleanup(func() { delete(usbDevPaths, portA); delete(usbDevPaths, portB) })

	// Hotplug: B replugged while A is still plugged must not take A's key
	TakeSerialByDevPath(portB, 7199)
	if k := serialLessKey(7199, portB, "", holder); k != keyB {
		t.Fatalf("B replug key = %q, want %q", k, keyB)
	}
	usbDevPaths[portB] = usbDevice{serial: "serialB", key: keyB}

	// Hotplug: A replugged while B is still plugged gets its own key back
	TakeSerialByDevPath(portA, 7199)
	if k := serialLessKey(7199, portA, "", holder); k != keyA {
		t.Fatalf("A replug key = %q, want %q", k, keyA)
	}

	// Another product is not affected
	if k := serialLessKey(7203, portA, "", holder); k != "7203" {
		t.Fatalf("other product key = %q, want \"7203\"", k)
	}
}

// The key is reserved before a device initializes, so an identical PSU plugged in while the first
// one is still initializing does not take its key. The serial is filled in once it registers.
func TestReserveUSBDevPath(t *testing.T) {
	usePlainKeyPorts(t)
	const portA = "/devices/pci0000:00/0000:00:14.0/usb1/1-2"
	const portB = "/devices/pci0000:00/0000:00:14.0/usb1/1-3"
	t.Cleanup(func() { delete(usbDevPaths, portA); delete(usbDevPaths, portB) })

	reserveUSBDevPath(portA, "/dev/hidraw9", "7199", 7199)
	if k := serialLessKey(7199, portB, "", holder); k != "7199-1-3" {
		t.Fatalf("key while A initializes = %q, want \"7199-1-3\"", k)
	}
	// Unplugged while initializing: nothing to stop yet, and the late registration is not kept
	if _, ok := TakeSerialByDevPath(portA, 7199); ok {
		t.Fatal("device still initializing should report no serial")
	}
	trackUSBDevPath(portA, &common.Device{Serial: "serialA"})
	if _, ok := usbDevPaths[portA]; ok {
		t.Fatal("device unplugged while initializing should be forgotten")
	}

	// Failed to initialize: the key is released
	reserveUSBDevPath(portA, "/dev/hidraw9", "7199", 7199)
	trackUSBDevPath(portA, nil)
	if _, held := keyHolder("7199", ""); held {
		t.Fatal("key of a device that failed to initialize should be released")
	}

	// Registered: the serial is kept under the reserved key
	reserveUSBDevPath(portB, "/dev/hidraw9", "7199-1-3", 7199)
	trackUSBDevPath(portB, &common.Device{Serial: "serialB"})
	if serial, ok := TakeSerialByDevPath(portB, 7199); !ok || serial != "serialB" {
		t.Fatalf("got (%q, %v), want (\"serialB\", true)", serial, ok)
	}
}

// usePlainKeyPorts starts a test with no pinned ports, saved to a file of its own, and returns
// that file.
func usePlainKeyPorts(t *testing.T) string {
	t.Helper()
	location := filepath.Join(t.TempDir(), "serialless.json")
	oldPorts, oldLocation := plainKeyPorts, plainKeyPortsLocation
	plainKeyPorts = map[string]string{}
	plainKeyPortsLocation = func() string { return location }
	t.Cleanup(func() { plainKeyPorts, plainKeyPortsLocation = oldPorts, oldLocation })
	return location
}

// Once a port holds the plain key, only a PSU on that port gets it, in whatever order the PSUs
// are found. A single PSU that never met an identical one keeps the plain key on any port.
func TestSerialLessKeyPinsPlainKeyToPort(t *testing.T) {
	const portA = "/devices/pci0000:00/0000:00:14.0/usb1/1-2"
	const portB = "/devices/pci0000:00/0000:00:14.0/usb1/1-3"
	none := func(string) (string, bool) { return "", false }
	location := usePlainKeyPorts(t)

	if k := serialLessKey(7199, portB, "", none); k != "7199" {
		t.Fatalf("single PSU key = %q, want \"7199\"", k)
	}
	if common.FileExists(location) {
		t.Fatal("a single PSU should not pin its port")
	}

	plainKeyPorts["7199"] = "1-2"
	// B alone (A unplugged, or found first) does not take A's key
	if k := serialLessKey(7199, portB, "", none); k != "7199-1-3" {
		t.Fatalf("B alone key = %q, want \"7199-1-3\"", k)
	}
	if k := serialLessKey(7199, portA, "", none); k != "7199" {
		t.Fatalf("A key = %q, want \"7199\"", k)
	}
	// Port unknown: it can't be the pinned port
	if k := serialLessKey(7199, "", "/dev/hidraw1", none); k != "7199-hidraw1" {
		t.Fatalf("key without port = %q, want \"7199-hidraw1\"", k)
	}
	// A stale entry left on A's own port does not cost A its key
	stale := func(string) (string, bool) { return portA, true }
	if k := serialLessKey(7199, portA, "", stale); k != "7199" {
		t.Fatalf("A key with stale entry = %q, want \"7199\"", k)
	}
	// A PSU on the pinned port never shares the plain key
	held := func(string) (string, bool) { return "", true }
	if k := serialLessKey(7199, portA, "", held); k != "7199-1-2" {
		t.Fatalf("A key while plain key held = %q, want \"7199-1-2\"", k)
	}
	if common.FileExists(location) {
		t.Fatal("a pinned port should not be saved again")
	}
}

// Pinned ports survive a restart, and a damaged file is ignored rather than crashing.
func TestLoadPlainKeyPorts(t *testing.T) {
	const portA = "/devices/pci0000:00/0000:00:14.0/usb1/1-2"
	const portB = "/devices/pci0000:00/0000:00:14.0/usb1/1-3"
	location := usePlainKeyPorts(t)

	serialLessKey(7199, portB, "", func(string) (string, bool) { return portA, true })
	plainKeyPorts = map[string]string{}
	loadPlainKeyPorts()
	if plainKeyPorts["7199"] != "1-2" {
		t.Fatalf("loaded ports = %v, want 7199 on 1-2", plainKeyPorts)
	}

	for _, data := range []string{"null", "", "[]", `{"7199":5}`, `{"7199":""}`} {
		if err := os.WriteFile(location, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		plainKeyPorts = map[string]string{}
		loadPlainKeyPorts()
		if len(plainKeyPorts) != 0 {
			t.Fatalf("file %q loaded ports %v, want none", data, plainKeyPorts)
		}
		// Pinning still works
		if k := serialLessKey(7199, portB, "", func(string) (string, bool) { return portA, true }); k != "7199-1-3" {
			t.Fatalf("file %q: B key = %q, want \"7199-1-3\"", data, k)
		}
	}
}

// A removed device whose USB port is unknown is matched only by its product and a gone HID node,
// and only once it registered.
func TestTakeSerialByDevPathPortless(t *testing.T) {
	const usbPath = "/devices/pci0000:00/0000:00:14.0/usb1/1-9"
	present := map[string]bool{"/dev/hidraw0": true}
	oldNode, oldPaths := hidNodeHolds, usbDevPaths
	hidNodeHolds = func(path string, productId uint16) bool { return present[path] && productId == 7199 }
	usbDevPaths = map[string]usbDevice{
		"/dev/hidraw0": {serial: "serialA", key: "7199", productId: 7199, portless: true},
		"/dev/hidraw1": {key: "7199-hidraw1", productId: 7199, portless: true},
	}
	t.Cleanup(func() { hidNodeHolds, usbDevPaths = oldNode, oldPaths })

	if _, ok := TakeSerialByDevPath(usbPath, 7199); ok {
		t.Fatal("took a device that is still plugged in, or one still initializing")
	}
	delete(present, "/dev/hidraw0")
	if _, ok := TakeSerialByDevPath(usbPath, 7200); ok {
		t.Fatal("took a device of another product")
	}
	if serial, ok := TakeSerialByDevPath(usbPath, 7199); !ok || serial != "serialA" {
		t.Fatalf("got (%q, %v), want (\"serialA\", true)", serial, ok)
	}
	if _, ok := usbDevPaths["/dev/hidraw1"]; !ok {
		t.Fatal("forgot a device still initializing")
	}
}

// A device whose USB port is unknown holds its key only while its HID node is present.
func TestKeyHolderPortless(t *testing.T) {
	present := true
	oldNode, oldPaths := hidNodeHolds, usbDevPaths
	hidNodeHolds = func(string, uint16) bool { return present }
	usbDevPaths = map[string]usbDevice{"/dev/hidraw0": {serial: "serialA", key: "7199", productId: 7199, portless: true}}
	t.Cleanup(func() { hidNodeHolds, usbDevPaths = oldNode, oldPaths })

	if path, ok := keyHolder("7199", ""); !ok || path != "" {
		t.Fatalf("got (%q, %v), want (\"\", true)", path, ok)
	}
	// The device asking is on the holder's node, so the holder is gone
	if _, ok := keyHolder("7199", "/dev/hidraw0"); ok {
		t.Fatal("a device unplugged unnoticed holds the key of the device now on its node")
	}
	present = false
	if _, ok := keyHolder("7199", ""); ok {
		t.Fatal("an unplugged device still holds its key")
	}
}

// A device initialized on the HID node of a port-less device unplugged unnoticed takes over the
// node, so the port-less device no longer holds its key.
func TestReserveReplacesStalePortless(t *testing.T) {
	const portA = "/devices/pci0000:00/0000:00:14.0/usb1/1-2"
	oldNode, oldPaths := hidNodeHolds, usbDevPaths
	hidNodeHolds = func(string, uint16) bool { return true }
	usbDevPaths = map[string]usbDevice{"/dev/hidraw0": {serial: "serialA", key: "7199", productId: 7199, portless: true}}
	t.Cleanup(func() { hidNodeHolds, usbDevPaths = oldNode, oldPaths })

	reserveUSBDevPath(portA, "/dev/hidraw0", "abc", 2000)
	if _, ok := keyHolder("7199", ""); ok {
		t.Fatal("a port-less device unplugged unnoticed still holds its key")
	}
	if path, ok := keyHolder("abc", ""); !ok || path != portA {
		t.Fatalf("got (%q, %v), want (%q, true)", path, ok, portA)
	}
}

// A device that takes the key of a port-less device unplugged unnoticed shares its serial, so the
// stale entry is forgotten; a later removal of a twin must not take the live device.
func TestReserveForgetsStalePortlessKey(t *testing.T) {
	const portA, portB = "/devices/pci0000:00/0000:00:14.0/usb1/7-9", "/devices/pci0000:00/0000:00:14.0/usb1/7-10"
	oldNode, oldPaths := hidNodeHolds, usbDevPaths
	hidNodeHolds = func(string, uint16) bool { return false } // hidraw0 was taken by another vendor
	usbDevPaths = map[string]usbDevice{"/dev/hidraw0": {serial: "serial-7199", key: "7199", productId: 7199, portless: true}}
	t.Cleanup(func() { hidNodeHolds, usbDevPaths = oldNode, oldPaths })

	reserveUSBDevPath(portA, "/dev/hidraw2", "7199", 7199)
	trackUSBDevPath(portA, &common.Device{Serial: "serial-7199"})
	if _, ok := usbDevPaths["/dev/hidraw0"]; ok {
		t.Fatal("kept the entry of a port-less device whose key was taken")
	}
	if serial, ok := TakeSerialByDevPath(portB, 7199); ok {
		t.Fatalf("removal of an unregistered twin took %q", serial)
	}
	if serial, ok := TakeSerialByDevPath(portA, 7199); !ok || serial != "serial-7199" {
		t.Fatalf("got (%q, %v), want (\"serial-7199\", true)", serial, ok)
	}
}

// A twin reserved next to a port-less device still plugged in leaves that device's entry alone, so
// its removal is still found.
func TestReserveKeepsLivePortless(t *testing.T) {
	const portB = "/devices/pci0000:00/0000:00:14.0/usb1/7-10"
	present := map[string]bool{"/dev/hidraw0": true}
	oldNode, oldPaths := hidNodeHolds, usbDevPaths
	hidNodeHolds = func(path string, _ uint16) bool { return present[path] }
	usbDevPaths = map[string]usbDevice{"/dev/hidraw0": {serial: "serial-7199", key: "7199", productId: 7199, portless: true}}
	t.Cleanup(func() { hidNodeHolds, usbDevPaths = oldNode, oldPaths })

	reserveUSBDevPath(portB, "/dev/hidraw1", "7199-7-10", 7199)
	reserveUSBDevPath("", "/dev/hidraw2", "7199-hidraw2", 7199)
	if _, ok := usbDevPaths["/dev/hidraw0"]; !ok {
		t.Fatal("forgot a port-less device still plugged in")
	}
	delete(present, "/dev/hidraw0")
	if serial, ok := TakeSerialByDevPath("/devices/pci0000:00/0000:00:14.0/usb1/9-9", 7199); !ok || serial != "serial-7199" {
		t.Fatalf("got (%q, %v), want (\"serial-7199\", true)", serial, ok)
	}
}

// A HID node is held by a device only while it exists and sysfs reports that device's vendor and
// product on it.
func TestHidNodeHolds(t *testing.T) {
	dir := t.TempDir()
	oldClass := hidrawClassPath
	hidrawClassPath = filepath.Join(dir, "class")
	t.Cleanup(func() { hidrawClassPath = oldClass })
	node := func(name, uevent string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if len(uevent) > 0 {
			sys := filepath.Join(hidrawClassPath, name, "device")
			if err := os.MkdirAll(sys, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(sys, "uevent"), []byte(uevent), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return path
	}
	twoFields := node("hidraw5", "HID_ID=0003:00001B1C\n")
	garbage := node("hidraw6", "HID_ID=0003:zz:00001C1F\n")
	foreignSamePID := node("hidraw7", "HID_ID=0003:0000046D:00001C1F\n")
	psu := node("hidraw0", "DRIVER=hid-generic\nHID_ID=0003:00001B1C:00001C1F\nHID_NAME=Corsair HX1500i\n")
	foreign := node("hidraw1", "DRIVER=hid-generic\nHID_ID=0003:0000046D:0000C52B\nHID_NAME=Logitech USB Receiver\n")
	otherPSU := node("hidraw2", "HID_ID=0003:00001B1C:00001C20\n")
	unknown := node("hidraw3", "")

	for _, tc := range []struct {
		name string
		path string
		want bool
	}{
		{"same product", psu, true},
		{"another vendor", foreign, false},
		{"another product", otherPSU, false},
		{"owner unknown", unknown, true},
		{"HID_ID with two fields", twoFields, true},
		{"unparsable HID_ID", garbage, true},
		{"another vendor, same product ID", foreignSamePID, false},
		{"node gone", filepath.Join(dir, "hidraw4"), false},
	} {
		if got := hidNodeHolds(tc.path, 7199); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

// Pages range over GetDevices while hotplug adds and removes devices. Ranging over the live map
// during a write is a fatal runtime error, so GetDevices must hand out a copy.
func TestGetDevicesWhileHotplugging(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 2000; i++ {
			addDevice(&common.Device{Serial: "qa-hotplug"})
			deleteDevice("qa-hotplug")
		}
	}()
	for {
		select {
		case <-done:
			return
		default:
			for range GetDevices() {
			}
			_ = GetDevice("qa-hotplug")
			_ = GetDevicesEx()
			_ = GetMouse()
			for range deviceSnapshot() {
			}
		}
	}
}

// Readers of the device map don't wait for a device call in flight, which holds mutex across
// device I/O (a PSU fan write can take seconds).
func TestDeviceReadsDontWaitForDeviceCalls(t *testing.T) {
	mutex.Lock() // as CallDeviceMethod does while a slow device method runs
	defer mutex.Unlock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = GetDevices()
		_ = GetDevice("cluster")
		_ = GetDevicesEx()
		_ = GetMouse()
		deleteDevice("qa-slow")
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a device map read waited for a device call")
	}
}
