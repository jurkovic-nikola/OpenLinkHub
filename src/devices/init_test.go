package devices

import (
	"OpenLinkHub/src/common"
	"errors"
	"sync"
	"testing"
	"time"

	hid "github.com/sstallion/go-hid"
)

const usbRoot = "/devices/pci0000:00/0000:00:14.0/usb1/"

// fakePSUs stands in for HID enumeration, the sysfs USB port lookup and PSU initialization,
// so startup and InitManual can be driven with identical serial-less PSUs (HX1500i, product 7199).
type fakePSUs struct {
	mu      sync.Mutex
	plugged []string                 // hidraw names, in enumeration order
	ports   map[string]string        // hidraw name -> USB port
	hold    map[string]chan struct{} // HID path -> closed when its initialization may finish
	keys    map[string]string        // HID path -> key it was initialized with
}

func useFakePSUs(t *testing.T) *fakePSUs {
	t.Helper()
	f := &fakePSUs{ports: map[string]string{}, hold: map[string]chan struct{}{}, keys: map[string]string{}}
	usePlainKeyPorts(t)

	oldEnumerate, oldDevPath, oldNode, oldPSU := hidEnumerate, usbDevPathOf, hidNodeHolds, deviceRegisterMap[7199]
	oldDevices, oldList, oldPaths, oldInterface := devices, deviceList, usbDevPaths, interfaceId
	devices, deviceList, usbDevPaths = map[string]*common.Device{}, map[string]Device{}, map[string]usbDevice{}
	t.Cleanup(func() {
		initWG.Wait()
		hidEnumerate, usbDevPathOf, hidNodeHolds, deviceRegisterMap[7199] = oldEnumerate, oldDevPath, oldNode, oldPSU
		devices, deviceList, usbDevPaths, interfaceId = oldDevices, oldList, oldPaths, oldInterface
	})

	hidEnumerate = func(vid, pid uint16, fn hid.EnumFunc) error {
		f.mu.Lock()
		plugged := append([]string(nil), f.plugged...)
		f.mu.Unlock()
		if vid != vendorId || (pid != hid.ProductIDAny && pid != 7199) {
			return nil
		}
		for _, h := range plugged {
			if err := fn(&hid.DeviceInfo{Path: "/dev/" + h, VendorID: vid, ProductID: 7199}); err != nil {
				return err
			}
		}
		return nil
	}
	hidNodeHolds = func(path string, productId uint16) bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		if productId != 7199 {
			return false
		}
		for _, h := range f.plugged {
			if "/dev/"+h == path {
				return true
			}
		}
		return false
	}
	usbDevPathOf = func(hidraw string) (string, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if port, ok := f.ports[hidraw]; ok {
			return usbRoot + port, nil
		}
		return "", errors.New("usb device node not found")
	}
	deviceRegisterMap[7199] = Product{Name: "HX1500i", DeviceRegister: func(vid, pid uint16, key, path string) *common.Device {
		f.mu.Lock()
		f.keys[path] = key
		hold := f.hold[path]
		f.mu.Unlock()
		if hold != nil {
			<-hold
		}
		return &common.Device{ProductType: common.ProductTypePSUHid, Serial: "serial-" + key, Instance: &struct{}{}}
	}}
	return f
}

// plug adds a PSU on a USB port, or on an unresolvable one when port is empty; with hold, its initialization waits until hold is closed.
func (f *fakePSUs) plug(hidraw, port string, hold chan struct{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.plugged = append(f.plugged, hidraw)
	if len(port) > 0 {
		f.ports[hidraw] = port
	}
	if hold != nil {
		f.hold["/dev/"+hidraw] = hold
	}
}

// unplug removes a PSU the way the hotplug monitor does
func (f *fakePSUs) unplug(t *testing.T, hidraw string) {
	t.Helper()
	f.mu.Lock()
	port, ok := f.ports[hidraw]
	if !ok {
		port = "9-9" // udev still reports a USB path, but it was never resolved from the HID node
	}
	for i, h := range f.plugged {
		if h == hidraw {
			f.plugged = append(f.plugged[:i], f.plugged[i+1:]...)
			break
		}
	}
	delete(f.ports, hidraw)
	f.mu.Unlock()
	serial, ok := TakeSerialByDevPath(usbRoot+port, 7199)
	if !ok {
		t.Fatalf("no registered device on port %s", port)
	}
	devicesMu.Lock()
	delete(devices, serial)
	devicesMu.Unlock()
}

func (f *fakePSUs) key(hidraw string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.keys["/dev/"+hidraw]
}

func wantKey(t *testing.T, f *fakePSUs, hidraw, want string) {
	t.Helper()
	if got := f.key(hidraw); got != want {
		t.Fatalf("%s initialized with key %q, want %q", hidraw, got, want)
	}
	devicesMu.Lock()
	_, ok := devices["serial-"+want]
	devicesMu.Unlock()
	if !ok {
		t.Fatalf("%s (key %q) is not listed", hidraw, want)
	}
}

// restart forgets every device and pinned port, as if the service was restarted
func restart() {
	mutex.Lock()
	devicesMu.Lock()
	devices, deviceList, usbDevPaths = map[string]*common.Device{}, map[string]Device{}, map[string]usbDevice{}
	devicesMu.Unlock()
	mutex.Unlock()
	plainKeyMutex.Lock()
	plainKeyPorts = map[string]string{}
	plainKeyMutex.Unlock()
}

// startup does what Init does for HID devices, without touching real USB devices or the config folder
func startup() {
	initHIDDevices()
	initializeDeviceList()
	initWG.Wait()
}

// Two identical PSUs found at startup get distinct keys, and each is found by its USB port.
func TestInitKeepsIdenticalPsusApart(t *testing.T) {
	f := useFakePSUs(t)
	f.plug("hidraw0", "1-2", nil)
	f.plug("hidraw1", "1-3", nil)
	startup()

	wantKey(t, f, "hidraw0", "7199")
	wantKey(t, f, "hidraw1", "7199-1-3")
	if serial, ok := TakeSerialByDevPath(usbRoot+"1-3", 7199); !ok || serial != "serial-7199-1-3" {
		t.Fatalf("port 1-3 maps to (%q, %v), want (\"serial-7199-1-3\", true)", serial, ok)
	}
}

// A PSU hotplugged next to an identical one gets a key of its own, and a replugged PSU gets its
// old key back.
func TestInitManualKeepsIdenticalPsusApart(t *testing.T) {
	f := useFakePSUs(t)
	f.plug("hidraw0", "1-2", nil)
	startup()

	f.plug("hidraw1", "1-3", nil)
	InitManual(7199, "", usbRoot+"1-3")
	initWG.Wait()
	wantKey(t, f, "hidraw1", "7199-1-3")

	f.unplug(t, "hidraw0")
	f.plug("hidraw2", "1-2", nil)
	InitManual(7199, "", usbRoot+"1-2")
	initWG.Wait()
	wantKey(t, f, "hidraw2", "7199")
}

// A PSU plugged in while an identical one is still initializing does not take its key.
func TestInitManualWhileIdenticalPsuInitializes(t *testing.T) {
	f := useFakePSUs(t)
	startup()

	hold := make(chan struct{})
	f.plug("hidraw0", "1-2", hold)
	InitManual(7199, "", usbRoot+"1-2")
	f.plug("hidraw1", "1-3", nil)
	InitManual(7199, "", usbRoot+"1-3")
	close(hold)
	initWG.Wait()

	wantKey(t, f, "hidraw0", "7199")
	wantKey(t, f, "hidraw1", "7199-1-3")
}

// Once two identical PSUs were seen, each keeps its key across restarts and replugs, whichever
// is found or plugged in first.
func TestIdenticalPsusKeepKeysAcrossOrder(t *testing.T) {
	f := useFakePSUs(t)
	f.plug("hidraw0", "1-2", nil)
	f.plug("hidraw1", "1-3", nil)
	startup()
	wantKey(t, f, "hidraw0", "7199")
	wantKey(t, f, "hidraw1", "7199-1-3")

	// Restart with only the second PSU plugged in
	f.unplug(t, "hidraw0")
	f.unplug(t, "hidraw1")
	restart()
	f.plug("hidraw2", "1-3", nil)
	startup()
	wantKey(t, f, "hidraw2", "7199-1-3")

	// Restart with the second PSU found first
	f.unplug(t, "hidraw2")
	restart()
	f.plug("hidraw3", "1-3", nil)
	f.plug("hidraw4", "1-2", nil)
	startup()
	wantKey(t, f, "hidraw3", "7199-1-3")
	wantKey(t, f, "hidraw4", "7199")

	// Both unplugged, the second replugged first
	f.unplug(t, "hidraw3")
	f.unplug(t, "hidraw4")
	f.plug("hidraw5", "1-3", nil)
	InitManual(7199, "", usbRoot+"1-3")
	initWG.Wait()
	wantKey(t, f, "hidraw5", "7199-1-3")
	f.plug("hidraw6", "1-2", nil)
	InitManual(7199, "", usbRoot+"1-2")
	initWG.Wait()
	wantKey(t, f, "hidraw6", "7199")
}

// A PSU whose USB port can't be resolved still holds its key, so an identical PSU hotplugged
// next to it gets a key of its own instead of replacing it in the device list.
func TestInitManualNextToPortlessPsu(t *testing.T) {
	f := useFakePSUs(t)
	f.plug("hidraw0", "", nil)
	startup()
	wantKey(t, f, "hidraw0", "7199")

	f.plug("hidraw1", "1-3", nil)
	InitManual(7199, "", usbRoot+"1-3")
	initWG.Wait()
	wantKey(t, f, "hidraw1", "7199-1-3")
	wantKey(t, f, "hidraw0", "7199")
	if _, pinned := plainKeyPorts["7199"]; pinned {
		t.Fatal("plain key pinned to a PSU whose USB port is unknown")
	}
}

// A PSU whose USB port can't be resolved is found on unplug, so it leaves no ghost entry and
// frees its key.
func TestUnplugPortlessPsu(t *testing.T) {
	f := useFakePSUs(t)
	f.plug("hidraw0", "", nil)
	startup()
	wantKey(t, f, "hidraw0", "7199")

	f.unplug(t, "hidraw0")
	devicesMu.Lock()
	left := len(devices)
	devicesMu.Unlock()
	if left != 0 {
		t.Fatalf("%d devices listed after unplug, want 0", left)
	}

	f.plug("hidraw1", "1-3", nil)
	InitManual(7199, "", usbRoot+"1-3")
	initWG.Wait()
	wantKey(t, f, "hidraw1", "7199")
}

// A port-less PSU whose unplug was missed, replugged on its old HID node, keeps its key.
func TestReplugPortlessPsuAfterMissedUnplug(t *testing.T) {
	f := useFakePSUs(t)
	f.plug("hidraw0", "", nil)
	startup()
	wantKey(t, f, "hidraw0", "7199")

	// Unplugged without the monitor noticing, then plugged back on a port that now resolves
	f.mu.Lock()
	f.plugged = nil
	f.mu.Unlock()
	f.plug("hidraw0", "1-3", nil)
	InitManual(7199, "", usbRoot+"1-3")
	initWG.Wait()
	wantKey(t, f, "hidraw0", "7199")
}

// stopCounter stands in for a device instance and counts how often it was stopped dirty
type stopCounter struct{ stopped int }

func (s *stopCounter) StopDirty() uint8 {
	s.stopped++
	return 2
}

// A PSU that replaces one registered under the same serial stops the instance it replaces.
func TestReplugStopsReplacedInstance(t *testing.T) {
	f := useFakePSUs(t)
	register := deviceRegisterMap[7199]
	var instances []*stopCounter
	deviceRegisterMap[7199] = Product{Name: register.Name, DeviceRegister: func(vid, pid uint16, key, path string) *common.Device {
		dev := register.DeviceRegister(vid, pid, key, path)
		s := &stopCounter{}
		instances = append(instances, s) // initializations run one at a time here
		dev.Instance = s
		return dev
	}}
	f.plug("hidraw0", "", nil)
	startup()

	// Unplugged without the monitor noticing, then plugged back with the same key
	f.mu.Lock()
	f.plugged = nil
	f.mu.Unlock()
	f.plug("hidraw0", "1-3", nil)
	InitManual(7199, "", usbRoot+"1-3")
	initWG.Wait()
	wantKey(t, f, "hidraw0", "7199")

	if len(instances) != 2 {
		t.Fatalf("got %d initializations, want 2", len(instances))
	}
	if instances[0].stopped != 1 {
		t.Fatalf("replaced instance stopped %d times, want 1", instances[0].stopped)
	}
	if instances[1].stopped != 0 {
		t.Fatalf("new instance stopped %d times, want 0", instances[1].stopped)
	}
	devicesMu.Lock()
	got := devices["serial-7199"].Instance
	devicesMu.Unlock()
	if got != instances[1] {
		t.Fatal("serial-7199 is not the new instance")
	}
}

// Re-adding the same instance, or replacing a device that is not a PSU, stops nothing: other
// devices stay listed after an unplug already stopped them, and a second StopDirty panics.
func TestAddDeviceStopsOnlyReplacedPsus(t *testing.T) {
	useFakePSUs(t)
	psu := &stopCounter{}
	dev := &common.Device{ProductType: common.ProductTypePSUHid, Serial: "psu", Instance: psu}
	addDevice(dev)
	addDevice(&common.Device{ProductType: common.ProductTypePSUHid, Serial: "psu", Instance: psu})
	cc := &stopCounter{}
	addDevice(&common.Device{ProductType: common.ProductTypeCC, Serial: "cc", Instance: cc})
	addDevice(&common.Device{ProductType: common.ProductTypeCC, Serial: "cc", Instance: &stopCounter{}})
	initWG.Wait()

	if psu.stopped != 0 {
		t.Fatalf("re-added PSU instance stopped %d times, want 0", psu.stopped)
	}
	if cc.stopped != 0 {
		t.Fatalf("replaced non-PSU instance stopped %d times, want 0", cc.stopped)
	}
}

// blockedStop stands in for a PSU whose refresh is hung on its old node, so StopDirty waits
type blockedStop struct{ release chan struct{} }

func (b *blockedStop) StopDirty() uint8 {
	<-b.release
	return 2
}

// A replaced PSU stuck stopping does not hold up registering the PSU that replaced it.
func TestReplugNotHeldUpByReplacedInstance(t *testing.T) {
	f := useFakePSUs(t)
	register := deviceRegisterMap[7199]
	old := &blockedStop{release: make(chan struct{})}
	first := true
	deviceRegisterMap[7199] = Product{Name: register.Name, DeviceRegister: func(vid, pid uint16, key, path string) *common.Device {
		dev := register.DeviceRegister(vid, pid, key, path)
		if first { // initializations run one at a time here
			first = false
			dev.Instance = old
		}
		return dev
	}}
	f.plug("hidraw0", "", nil)
	startup()

	f.mu.Lock()
	f.plugged = nil
	f.mu.Unlock()
	f.plug("hidraw0", "1-3", nil)
	InitManual(7199, "", usbRoot+"1-3")
	defer initWG.Wait()
	defer close(old.release)

	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(time.Millisecond) {
		mutex.Lock()
		serial := usbDevPaths[usbRoot+"1-3"].serial
		mutex.Unlock()
		if serial == "serial-7199" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("new PSU not registered on its port while the replaced one is stopping")
		}
	}
}

// countStops makes each PSU initialization a stopCounter, returned by serial; read it once
// initializations are done
func countStops(t *testing.T) map[string]*stopCounter {
	t.Helper()
	register := deviceRegisterMap[7199]
	instances := map[string]*stopCounter{}
	var mu sync.Mutex // startup initializes PSUs in parallel
	deviceRegisterMap[7199] = Product{Name: register.Name, DeviceRegister: func(vid, pid uint16, key, path string) *common.Device {
		dev := register.DeviceRegister(vid, pid, key, path)
		s := &stopCounter{}
		mu.Lock()
		instances[dev.Serial] = s
		mu.Unlock()
		dev.Instance = s
		return dev
	}}
	return instances
}

func wantListed(t *testing.T, serial string, want bool) {
	t.Helper()
	devicesMu.Lock()
	_, ok := devices[serial]
	devicesMu.Unlock()
	if ok != want {
		t.Fatalf("%s listed = %v, want %v", serial, ok, want)
	}
}

// A port-less PSU whose unplug was missed and that came back on a known port leaves a stale
// entry; it is dropped and its device stopped on replug, so a later unplug of its twin takes the twin.
func TestStalePortlessPsuDoesNotSwallowTwinRemoval(t *testing.T) {
	f := useFakePSUs(t)
	instances := countStops(t)
	f.plug("hidraw0", "", nil)
	f.plug("hidraw1", "", nil)
	startup()
	wantKey(t, f, "hidraw0", "7199")
	wantKey(t, f, "hidraw1", "7199-hidraw1")

	// B is unplugged without the monitor noticing, then plugged back on a port that resolves
	f.mu.Lock()
	f.plugged = f.plugged[:1]
	f.mu.Unlock()
	f.plug("hidraw2", "7-10", nil)
	InitManual(7199, "", usbRoot+"7-10")
	initWG.Wait()
	wantKey(t, f, "hidraw2", "7199-7-10")
	wantListed(t, "serial-7199-hidraw1", false)
	if n := instances["serial-7199-hidraw1"].stopped; n != 1 {
		t.Fatalf("stale instance stopped %d times, want 1", n)
	}
	if n := instances["serial-7199"].stopped; n != 0 {
		t.Fatalf("live twin stopped %d times, want 0", n)
	}

	f.unplug(t, "hidraw0")
	initWG.Wait()
	wantListed(t, "serial-7199", false)
	wantListed(t, "serial-7199-7-10", true)
	for serial, s := range instances {
		if serial != "serial-7199-hidraw1" && s.stopped != 0 {
			t.Errorf("%s stopped %d times, want 0", serial, s.stopped) // the monitor stops the one unplugged
		}
	}
}

// Two port-less PSUs both unplugged, one unnoticed, can't be told apart on the noticed unplug;
// the one not taken is stopped, and neither is left listed.
func TestUnplugTakesEveryGonePortlessPsu(t *testing.T) {
	f := useFakePSUs(t)
	instances := countStops(t)
	f.plug("hidraw0", "", nil)
	f.plug("hidraw1", "", nil)
	startup()

	f.mu.Lock()
	f.plugged = f.plugged[:1] // B is unplugged without the monitor noticing
	f.mu.Unlock()
	f.unplug(t, "hidraw0")
	initWG.Wait()
	devicesMu.Lock()
	left := len(devices)
	devicesMu.Unlock()
	if left != 0 {
		t.Fatalf("%d devices listed after both PSUs are unplugged, want 0", left)
	}
	stops := 0
	for _, s := range instances {
		stops += s.stopped
	}
	if stops != 1 { // the monitor stops the one taken
		t.Fatalf("got %d stops, want 1", stops)
	}
}
