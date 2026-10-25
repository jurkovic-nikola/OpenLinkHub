package monitor

import (
	"OpenLinkHub/src/logger"
	"errors"
	"github.com/godbus/dbus/v5"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const usbRoot = "/devices/pci0000:00/0000:00:14.0/usb1/"

// stopped is one devices.StopDirty call: the key the device is removed under and its product.
type stopped struct {
	serial    string
	productId uint16
}

// fakeDevices stands in for sysfs and the devices package, so uevents can be handled without
// real hardware. It registers devices the way devices does: under their USB path when the port
// is known, otherwise as port-less. A port-less key is taken for any product, so only the
// monitor's own filters keep an unrelated removal from taking it.
type fakeDevices struct {
	paths    map[string]string // USB device path -> registered key
	portless []string          // keys of PSUs registered without a known USB port
	inits    []string          // USB device paths passed to devices.InitManual
	stops    []stopped
	devStops int   // devices.Stop calls
	inStops  int   // inputmanager.Stop calls
	exits    []int // os.Exit codes
}

func useFakeDevices(t *testing.T) *fakeDevices {
	t.Helper()
	f := &fakeDevices{paths: map[string]string{}}
	oldRoot, oldPause, oldInit, oldTake, oldStop, oldSleep := sysRoot, pause, initManual, takeSerialByDevPath, stopDirty, sleep.Load()
	oldStopDevices, oldStopInputs, oldExit := stopDevices, stopInputs, exit
	t.Cleanup(func() {
		sysRoot, pause, initManual, takeSerialByDevPath, stopDirty = oldRoot, oldPause, oldInit, oldTake, oldStop
		stopDevices, stopInputs, exit = oldStopDevices, oldStopInputs, oldExit
		sleep.Store(oldSleep)
	})

	sysRoot = t.TempDir()
	sleep.Store(false)
	pause = func(time.Duration) {}
	initManual = func(productId uint16, key, devPath string) {
		f.inits = append(f.inits, devPath)
	}
	takeSerialByDevPath = func(devPath string, productId uint16) (string, bool) {
		if key, ok := f.paths[devPath]; ok {
			delete(f.paths, devPath)
			return key, true
		}
		if len(f.portless) > 0 {
			key := f.portless[0]
			f.portless = f.portless[1:]
			return key, true
		}
		return "", false
	}
	stopDirty = func(serial string, productId uint16) {
		f.stops = append(f.stops, stopped{serial, productId})
	}
	stopDevices = func() { f.devStops++ }
	stopInputs = func() { f.inStops++ }
	exit = func(code int) { f.exits = append(f.exits, code) }
	return f
}

// sysfs will create the sysfs attributes of a serial-less USB device plugged at port.
// vid and pid are in hex, as sysfs shows them.
func sysfs(t *testing.T, port, vid, pid string) string {
	t.Helper()
	devPath := usbRoot + port
	dir := filepath.Join(sysRoot, devPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{"idVendor": vid + "\n", "idProduct": pid + "\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return devPath
}

// uevent will return a kernel usb_device uevent for a PSU at devPath, as common.ParseUEvent parses it.
func uevent(action, devPath string) map[string]string {
	return map[string]string{
		"ACTION":    action,
		"DEVPATH":   devPath,
		"SUBSYSTEM": "usb",
		"DEVTYPE":   "usb_device",
		"PRODUCT":   "1b1c/1c1f/100", // vendor/product/bcdDevice in unpadded hex
	}
}

func wantStops(t *testing.T, f *fakeDevices, want ...stopped) {
	t.Helper()
	if !slices.Equal(f.stops, want) {
		t.Fatalf("StopDirty calls = %v, want %v", f.stops, want)
	}
}

func TestAddInitializesAndCachesPsu(t *testing.T) {
	f := useFakeDevices(t)
	cache := map[string]USBInfo{}
	devPath := sysfs(t, "1-4", "1b1c", "1c1f")

	handleUEvent(uevent("add", devPath), cache)
	if !slices.Equal(f.inits, []string{devPath}) {
		t.Fatalf("InitManual calls = %v, want [%s]", f.inits, devPath)
	}
	if got := cache[devPath]; got != (USBInfo{VendorID: vendorId, ProductID: 7199}) {
		t.Fatalf("cache[%s] = %+v, want the PSU", devPath, got)
	}
}

func TestAddIgnoresUnsupportedDevice(t *testing.T) {
	tests := map[string]func(t *testing.T, msg map[string]string){
		"during suspend": func(*testing.T, map[string]string) { sleep.Store(true) },
		"other vendor":   func(t *testing.T, _ map[string]string) { sysfs(t, "1-4", "046d", "1c1f") },
		"excluded":       func(t *testing.T, _ map[string]string) { sysfs(t, "1-4", "1b1c", "2a00") },
		"usb_interface":  func(_ *testing.T, msg map[string]string) { msg["DEVTYPE"] = "usb_interface" },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			f := useFakeDevices(t)
			cache := map[string]USBInfo{}
			msg := uevent("add", sysfs(t, "1-4", "1b1c", "1c1f"))
			change(t, msg)
			handleUEvent(msg, cache)
			if len(f.inits) != 0 || len(cache) != 0 {
				t.Fatalf("InitManual calls = %v, cache = %v, want none", f.inits, cache)
			}
		})
	}
}

func TestRemoveCachedPsu(t *testing.T) {
	f := useFakeDevices(t)
	devPath := usbRoot + "1-4"
	cache := map[string]USBInfo{devPath: {VendorID: vendorId, ProductID: 7199}}
	f.paths[devPath] = "7199-1-4"
	f.portless = []string{"7199"}

	handleUEvent(uevent("remove", devPath), cache)
	wantStops(t, f, stopped{"7199-1-4", 7199})
	if _, ok := cache[devPath]; ok {
		t.Fatal("removed PSU is still cached")
	}
	if len(f.portless) != 1 {
		t.Fatal("a cached removal took the port-less PSU's key")
	}
}

func TestRemoveCachedPsuWithoutRegisteredKey(t *testing.T) {
	f := useFakeDevices(t)
	devPath := usbRoot + "1-4"
	cache := map[string]USBInfo{devPath: {VendorID: vendorId, ProductID: 7199}}

	// No key registered under the path: fall back to the product ID devices keys serial-less PSUs by
	handleUEvent(uevent("remove", devPath), cache)
	wantStops(t, f, stopped{"7199", 7199})
}

func TestRemoveCachedDeviceUnderUsbSerial(t *testing.T) {
	f := useFakeDevices(t)
	devPath := usbRoot + "1-4"
	cache := map[string]USBInfo{devPath: {VendorID: vendorId, ProductID: 7199, Serial: "ABC123"}}

	handleUEvent(uevent("remove", devPath), cache)
	wantStops(t, f, stopped{"ABC123", 7199})
}

func TestRemovePortlessPsu(t *testing.T) {
	f := useFakeDevices(t)
	f.portless = []string{"7199"}

	// Registered without a known USB port, so the startup cache has no entry under its path
	handleUEvent(uevent("remove", usbRoot+"1-4"), map[string]USBInfo{})
	wantStops(t, f, stopped{"7199", 7199})
}

func TestRemoveTwoPortlessPsus(t *testing.T) {
	f := useFakeDevices(t)
	f.portless = []string{"7199", "7199-2"}
	cache := map[string]USBInfo{}

	handleUEvent(uevent("remove", usbRoot+"1-4"), cache)
	handleUEvent(uevent("remove", usbRoot+"1-5"), cache)
	wantStops(t, f, stopped{"7199", 7199}, stopped{"7199-2", 7199})
}

func TestRemoveIgnoresUsbInterface(t *testing.T) {
	f := useFakeDevices(t)
	f.portless = []string{"7199"}
	devPath := usbRoot + "1-4"
	cache := map[string]USBInfo{devPath: {VendorID: vendorId, ProductID: 7199}}

	msg := uevent("remove", devPath+"/1-4:1.0")
	msg["DEVTYPE"] = "usb_interface"
	handleUEvent(msg, cache)
	wantStops(t, f)
	if len(f.portless) != 1 {
		t.Fatal("an interface removal took the port-less PSU's key")
	}
}

func TestRemoveIgnoresUncachedUnknownDevice(t *testing.T) {
	tests := map[string]string{
		"other vendor":     "46d/1c1f/100",
		"excluded product": "1b1c/2a00/100",
		"no product":       "",
	}
	for name, product := range tests {
		t.Run(name, func(t *testing.T) {
			f := useFakeDevices(t)
			f.portless = []string{"7199"}
			msg := uevent("remove", usbRoot+"1-4")
			msg["PRODUCT"] = product
			handleUEvent(msg, map[string]USBInfo{})
			wantStops(t, f)
			if len(f.portless) != 1 {
				t.Fatal("an unknown device's removal took the port-less PSU's key")
			}
		})
	}
}

func TestRemoveWithNothingToTakeIsIgnored(t *testing.T) {
	f := useFakeDevices(t)
	handleUEvent(uevent("remove", usbRoot+"1-4"), map[string]USBInfo{})
	wantStops(t, f)
}

func TestRemoveWithoutDevPathIsIgnored(t *testing.T) {
	f := useFakeDevices(t)
	f.portless = []string{"7199"}
	handleUEvent(uevent("remove", ""), map[string]USBInfo{})
	wantStops(t, f)
}

func TestRemoveDuringSuspendIsIgnored(t *testing.T) {
	f := useFakeDevices(t)
	devPath := usbRoot + "1-4"
	cache := map[string]USBInfo{devPath: {VendorID: vendorId, ProductID: 7199}}
	sleep.Store(true)

	handleUEvent(uevent("remove", devPath), cache)
	wantStops(t, f)
}

// The suspend flag is written by the D-Bus goroutine while uevents are handled on another;
// run with -race to catch a regression to an unsynchronized flag.
func TestSuspendFlagSetDuringUEvent(t *testing.T) {
	useFakeDevices(t)
	done := make(chan struct{})
	go func() {
		sleep.Store(true)
		close(done)
	}()
	handleUEvent(uevent("remove", usbRoot+"1-4"), map[string]USBInfo{})
	<-done
}

// prepareForSleep will return a login1 PrepareForSleep signal carrying body.
func prepareForSleep(body ...interface{}) *dbus.Signal {
	return &dbus.Signal{Name: "org.freedesktop.login1.Manager.PrepareForSleep", Body: body}
}

func TestSuspendSignalSetsFlagAndStopsDevices(t *testing.T) {
	f := useFakeDevices(t)

	handleSleepSignal(prepareForSleep(true))
	if !sleep.Load() {
		t.Error("suspend flag not set")
	}
	if f.devStops != 1 || f.inStops != 1 || len(f.exits) != 0 {
		t.Errorf("devices.Stop = %d, inputmanager.Stop = %d, exits = %v; want one of each stop and no exit", f.devStops, f.inStops, f.exits)
	}

	// A device unplugged while suspended is then left alone
	devPath := usbRoot + "1-4"
	handleUEvent(uevent("remove", devPath), map[string]USBInfo{devPath: {VendorID: vendorId, ProductID: 7199}})
	wantStops(t, f)
}

func TestResumeSignalClearsFlagAndExits(t *testing.T) {
	f := useFakeDevices(t)
	sleep.Store(true)

	handleSleepSignal(prepareForSleep(false))
	if sleep.Load() {
		t.Error("suspend flag still set after resume")
	}
	if f.devStops+f.inStops != 0 || !slices.Equal(f.exits, []int{1}) {
		t.Errorf("stop calls = %d, exits = %v; want no stops and exit(1)", f.devStops+f.inStops, f.exits)
	}
}

func TestNonBoolSleepSignalClearsFlag(t *testing.T) {
	f := useFakeDevices(t)
	sleep.Store(true)

	handleSleepSignal(prepareForSleep("true"))
	if sleep.Load() {
		t.Error("suspend flag still set after a non-bool signal")
	}
	if f.devStops+f.inStops != 0 || len(f.exits) != 0 {
		t.Errorf("stop calls = %d, exits = %v; want neither", f.devStops+f.inStops, f.exits)
	}
}

func TestEmptySleepSignalKeepsFlag(t *testing.T) {
	useFakeDevices(t)
	sleep.Store(true)

	handleSleepSignal(prepareForSleep())
	if !sleep.Load() {
		t.Error("suspend flag cleared by a signal with no body")
	}
}

func TestOnlyLogindSleepSignalIsTrusted(t *testing.T) {
	const owner = ":1.4"
	logind := func() *dbus.Signal {
		signal := prepareForSleep(true)
		signal.Sender, signal.Path = owner, logindPath
		return signal
	}
	if !fromLogind(logind(), owner) {
		t.Error("PrepareForSleep from logind rejected")
	}

	for name, change := range map[string]func(*dbus.Signal){
		"another peer":      func(s *dbus.Signal) { s.Sender = ":1.99" },
		"well-known sender": func(s *dbus.Signal) { s.Sender = logindName },
		"no sender":         func(s *dbus.Signal) { s.Sender = "" },
		"another path":      func(s *dbus.Signal) { s.Path = "/org/freedesktop/login1/session/self" },
		"another member":    func(s *dbus.Signal) { s.Name = "org.freedesktop.login1.Manager.PrepareForShutdown" },
		"another interface": func(s *dbus.Signal) { s.Name = "org.example.Manager.PrepareForSleep" },
	} {
		t.Run(name, func(t *testing.T) {
			signal := logind()
			change(signal)
			if fromLogind(signal, owner) {
				t.Errorf("signal %+v trusted", signal)
			}
		})
	}

	// No logind on the bus: an empty owner must not match an empty sender
	signal := logind()
	signal.Sender = ""
	if fromLogind(signal, "") {
		t.Error("signal trusted with logind absent")
	}
}

// watch will run watchSleepSignals over signals, with logind owned by each name of owners in
// turn; an empty name, or none left, fails the lookup. It returns the number of lookups.
// Each lookup is answered after every signal, so an owner that is not cached is looked up per signal.
func watch(cacheOwner bool, owners []string, signals ...*dbus.Signal) int {
	return watchAnswered(dbus.NoSequence, cacheOwner, owners, signals...)
}

// watchAnswered will run watch with each lookup answered at sequence answered, signals being
// numbered in order from 1.
func watchAnswered(answered dbus.Sequence, cacheOwner bool, owners []string, signals ...*dbus.Signal) int {
	for i, signal := range signals {
		signal.Sequence = dbus.Sequence(i + 1)
	}
	ch := make(chan *dbus.Signal, len(signals))
	for _, signal := range signals {
		ch <- signal
	}
	close(ch)
	lookups := 0
	watchSleepSignals(ch, func() (string, dbus.Sequence, error) {
		lookups++
		if len(owners) == 0 || owners[0] == "" {
			if len(owners) > 0 {
				owners = owners[1:]
			}
			return "", answered, errors.New("org.freedesktop.login1 has no owner")
		}
		owner := owners[0]
		owners = owners[1:]
		return owner, answered, nil
	}, cacheOwner)
	return lookups
}

// sleepSignal will return a PrepareForSleep signal on logind's path from sender.
func sleepSignal(sender string, sleeping bool) *dbus.Signal {
	signal := prepareForSleep(sleeping)
	signal.Sender, signal.Path = sender, logindPath
	return signal
}

// ownerChanged will return the NameOwnerChanged the bus daemon sends when logind's owner changes.
func ownerChanged(oldOwner, newOwner string) *dbus.Signal {
	return &dbus.Signal{Sender: busName, Path: busPath, Name: nameOwnerChangedMember, Body: []interface{}{logindName, oldOwner, newOwner}}
}

// captureWarnings will return the messages of the warnings logged while the test runs.
func captureWarnings(t *testing.T) *[]string {
	t.Helper()
	var warnings []string
	old := warn
	t.Cleanup(func() { warn = old })
	warn = func(_ logger.Fields, msg string) { warnings = append(warnings, msg) }
	return &warnings
}

func TestWatchIgnoresSpoofedSleepSignals(t *testing.T) {
	for _, cacheOwner := range []bool{true, false} {
		f := useFakeDevices(t)

		watch(cacheOwner, []string{":1.4", ":1.4"}, sleepSignal(":1.99", true), sleepSignal(":1.99", false))
		watch(cacheOwner, nil, sleepSignal(":1.4", true), sleepSignal(":1.4", false))
		if sleep.Load() || f.devStops+f.inStops != 0 || len(f.exits) != 0 {
			t.Errorf("cacheOwner = %v: suspend = %v, stop calls = %d, exits = %v; want spoofs ignored", cacheOwner, sleep.Load(), f.devStops+f.inStops, f.exits)
		}
	}
}

func TestWatchCachesLogindOwner(t *testing.T) {
	f := useFakeDevices(t)

	// A flood of spoofs must not cost a lookup each before logind's own signal is handled
	signals := []*dbus.Signal{}
	for range 100 {
		signals = append(signals, sleepSignal(":1.99", false))
	}
	signals = append(signals, sleepSignal(":1.4", true))
	if lookups := watch(true, []string{":1.4"}, signals...); lookups != 1 {
		t.Errorf("lookups = %d, want 1", lookups)
	}
	if f.devStops != 1 || len(f.exits) != 0 {
		t.Errorf("devices.Stop = %d, exits = %v; want logind's suspend handled and spoofs ignored", f.devStops, f.exits)
	}
}

func TestWatchFollowsLogindOwner(t *testing.T) {
	// logind restarts after the first signal and comes back under a new unique name, so a
	// suspend from its old name must be ignored
	t.Run("cached", func(t *testing.T) {
		f := useFakeDevices(t)
		lookups := watch(true, []string{":1.4", ":1.50"}, sleepSignal(":1.4", true),
			ownerChanged(":1.4", ""), ownerChanged("", ":1.50"),
			sleepSignal(":1.4", true), sleepSignal(":1.50", false))
		if f.devStops != 1 || f.inStops != 1 || !slices.Equal(f.exits, []int{1}) || lookups != 2 {
			t.Errorf("devices.Stop = %d, inputmanager.Stop = %d, exits = %v, lookups = %d; want one stop, then exit(1) from the new owner only, after 2 lookups", f.devStops, f.inStops, f.exits, lookups)
		}
	})
	t.Run("looked up per signal", func(t *testing.T) {
		f := useFakeDevices(t)
		watch(false, []string{":1.4", ":1.50", ":1.50"}, sleepSignal(":1.4", true), sleepSignal(":1.4", true), sleepSignal(":1.50", false))
		if f.devStops != 1 || f.inStops != 1 || !slices.Equal(f.exits, []int{1}) {
			t.Errorf("devices.Stop = %d, inputmanager.Stop = %d, exits = %v; want one stop, then exit(1) from the new owner only", f.devStops, f.inStops, f.exits)
		}
	})
}

func TestWatchIgnoresSpoofedOwnerChange(t *testing.T) {
	f := useFakeDevices(t)

	// A peer claiming logind moved must neither drop the cached owner nor be trusted
	spoof := ownerChanged(":1.4", ":1.99")
	spoof.Sender = ":1.99"
	other := ownerChanged("", ":1.7")
	other.Body[0] = "org.example.Other"
	lookups := watch(true, []string{":1.4", ":1.99"}, sleepSignal(":1.4", true), spoof, other, sleepSignal(":1.99", false))
	if lookups != 1 || f.devStops != 1 || len(f.exits) != 0 {
		t.Errorf("lookups = %d, devices.Stop = %d, exits = %v; want 1 lookup and the spoofed resume ignored", lookups, f.devStops, f.exits)
	}
}

func TestWatchWarnsOncePerOutage(t *testing.T) {
	for _, cacheOwner := range []bool{true, false} {
		useFakeDevices(t)
		warnings := captureWarnings(t)

		// outage, recovery, outage: each outage warns once however many signals it drops
		watch(cacheOwner, []string{"", "", ":1.4", "", ""},
			sleepSignal(":1.4", true), sleepSignal(":1.4", true), ownerChanged("", ":1.4"),
			sleepSignal(":1.99", true), ownerChanged(":1.4", ""),
			sleepSignal(":1.4", true), sleepSignal(":1.4", true))
		unresolved := 0
		for _, msg := range *warnings {
			if strings.HasPrefix(msg, "Unable to resolve logind") {
				unresolved++
			}
		}
		if unresolved != 2 {
			t.Errorf("cacheOwner = %v: warnings = %q, want one per outage", cacheOwner, *warnings)
		}
	}
}

func TestWatchWarnsOnceForSpoofs(t *testing.T) {
	for _, cacheOwner := range []bool{true, false} {
		useFakeDevices(t)
		warnings := captureWarnings(t)

		watch(cacheOwner, []string{":1.4", ":1.4", ":1.4"}, sleepSignal(":1.99", true), sleepSignal(":1.98", true), sleepSignal(":1.99", false))
		if len(*warnings) != 1 {
			t.Errorf("cacheOwner = %v: warnings = %q, want one", cacheOwner, *warnings)
		}
	}
}

func TestOwnerCachedOnlyWhenOwnerChangesAreMatched(t *testing.T) {
	// The bus refuses the n-th AddMatch (0 for none), as one capping match rules per connection does
	for refused, want := range []bool{true, true, false} {
		var rules []string
		got := addSleepMatches(func(rule string) error {
			rules = append(rules, rule)
			if len(rules) == refused {
				return errors.New("org.freedesktop.DBus.Error.LimitsExceeded")
			}
			return nil
		})
		if got != want {
			t.Errorf("AddMatch %d refused: cacheOwner = %v, want %v", refused, got, want)
		}
		if len(rules) != 2 || !strings.Contains(rules[0], "member='PrepareForSleep'") || !strings.Contains(rules[1], "member='NameOwnerChanged',arg0='"+logindName+"'") {
			t.Errorf("AddMatch %d refused: rules = %q, want PrepareForSleep then logind's NameOwnerChanged", refused, rules)
		}
	}
}

func TestListenCachesOwnerOnlyWhenOwnerChangesAreMatched(t *testing.T) {
	// Two suspends from logind: cached, its owner is looked up once; otherwise once per signal
	for refused, want := range map[int]int{0: 1, 1: 1, 2: 2} {
		useFakeDevices(t)
		matches, lookups := 0, 0
		listenForSleep(func(ch chan<- *dbus.Signal) {
			// Subscribing after the matches could drop a signal the bus routes in between
			if matches != 0 {
				t.Errorf("AddMatch %d refused: subscribed after %d matches, want before any", refused, matches)
			}
			ch <- sleepSignal(":1.4", true)
			ch <- sleepSignal(":1.4", true)
			close(ch)
		}, func(string) error {
			if matches++; matches == refused {
				return errors.New("org.freedesktop.DBus.Error.LimitsExceeded")
			}
			return nil
		}, func() (string, dbus.Sequence, error) {
			lookups++
			return ":1.4", dbus.NoSequence, nil
		})
		if lookups != want {
			t.Errorf("AddMatch %d refused: lookups = %d, want %d", refused, lookups, want)
		}
	}
}

func TestWatchReusesLookupForQueuedSignals(t *testing.T) {
	f := useFakeDevices(t)

	// Without a cached owner, a flood queued before the lookup's reply costs one lookup
	signals := []*dbus.Signal{}
	for range 100 {
		signals = append(signals, sleepSignal(":1.99", false))
	}
	signals = append(signals, sleepSignal(":1.4", true))
	if lookups := watchAnswered(1000, false, []string{":1.4"}, signals...); lookups != 1 {
		t.Errorf("lookups = %d, want 1", lookups)
	}
	if f.devStops != 1 || len(f.exits) != 0 {
		t.Errorf("devices.Stop = %d, exits = %v; want logind's suspend handled and spoofs ignored", f.devStops, f.exits)
	}
}

func TestWatchLooksUpAgainForSignalsAfterReply(t *testing.T) {
	f := useFakeDevices(t)

	// logind is replaced right after the lookup's reply; its successor's resume, received
	// after that reply, must not be judged by it
	lookups := watchAnswered(2, false, []string{":1.4", ":1.50"}, sleepSignal(":1.4", true), sleepSignal(":1.50", false))
	if lookups != 2 || f.devStops != 1 || !slices.Equal(f.exits, []int{1}) {
		t.Errorf("lookups = %d, devices.Stop = %d, exits = %v; want the new owner's resume handled after a fresh lookup", lookups, f.devStops, f.exits)
	}
}

// replyingBus stands in for the bus daemon, answering every call with reply.
type replyingBus struct {
	dbus.BusObject
	reply *dbus.Call
}

func (b replyingBus) Call(string, dbus.Flags, ...interface{}) *dbus.Call {
	return b.reply
}

func TestLookupLogindReturnsReplySequence(t *testing.T) {
	// The reply's sequence is what lets queued signals share one lookup
	owner, answered, err := lookupLogind(replyingBus{reply: &dbus.Call{Body: []interface{}{":1.4"}, ResponseSequence: 7}})
	if owner != ":1.4" || answered != 7 || err != nil {
		t.Errorf("lookupLogind = %q, %d, %v; want :1.4, 7, nil", owner, answered, err)
	}
}
