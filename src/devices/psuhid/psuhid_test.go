package psuhid

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/metrics"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/sstallion/go-hid"
)

// StopDirty must return 2 so devices.StopDirty drops the unplugged PSU from the device list,
// and must stop a running refresh loop without blocking.
func TestStopDirtyRemovesDevice(t *testing.T) {
	d := &Device{Serial: "x"}
	d.setAutoRefresh()

	done := make(chan uint8)
	go func() { done <- d.StopDirty() }()
	select {
	case r := <-done:
		if r != 2 {
			t.Fatalf("StopDirty returned %d, want 2", r)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("StopDirty blocked")
	}
	if d.timer != nil {
		t.Fatal("refresh timer not cleared")
	}
}

// A single PSU keeps its md5(product) serial; an identical second PSU, keyed with its USB port,
// gets its own serial (and profile).
func TestSerialFromKeyKeepsIdenticalPsusApart(t *testing.T) {
	single := serialFromKey("HX1500i", "7199")
	sum := md5.Sum([]byte("HX1500i"))
	if want := hex.EncodeToString(sum[:]); single != want {
		t.Fatalf("single PSU serial changed: %q, want %q", single, want)
	}
	second := serialFromKey("HX1500i", "7199-1-3")
	if second == single {
		t.Fatal("second identical PSU shares the first one's serial")
	}
	if third := serialFromKey("HX1500i", "7199-1-4"); third == second {
		t.Fatal("PSUs on different ports share a serial")
	}
}

// fakePSU answers every request like an HX1500i: it echoes the command and reports its product name.
// Replies queue up until read, like hidraw input reports.
type fakePSU struct {
	replies [][]byte
	closed  bool
	closes  int
	name    string // product string the PSU reports; "CORSAIR HX1500i PSU" when empty
}

func (f *fakePSU) GetMfrStr() (string, error) { return "CORSAIR", nil }
func (f *fakePSU) Close() error {
	f.closes++
	f.closed = true
	return nil
}

func (f *fakePSU) Write(p []byte) (int, error) {
	reply := make([]byte, readBufferSize)
	reply[0], reply[1] = p[1], p[2]
	if p[1] == cmdInit && p[2] == cmdRead {
		name := f.name
		if name == "" {
			name = "CORSAIR HX1500i PSU"
		}
		copy(reply[2:], name)
	}
	f.replies = append(f.replies, reply)
	return len(p), nil
}

func (f *fakePSU) ReadWithTimeout(p []byte, timeout time.Duration) (int, error) {
	if len(f.replies) == 0 {
		time.Sleep(timeout)
		return 0, hid.ErrTimeout
	}
	n := copy(p, f.replies[0])
	f.replies = f.replies[1:]
	return n, nil
}

// useFakePSUs makes Init open a fake PSU at any path and keep profiles in a temporary folder.
func useFakePSUs(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "database/profiles"), 0755); err != nil {
		t.Fatal(err)
	}
	pwdOnce.Do(func() {}) // Init must not overwrite pwd with the (unset) config path
	oldPwd, oldOpen := pwd, openPath
	pwd = dir
	openPath = func(string) (hidDevice, error) { return &fakePSU{}, nil }
	t.Cleanup(func() { pwd, openPath = oldPwd, oldOpen })
}

// Init registers a PSU under the serial derived from the key devices.go gave it, so an identical
// second PSU keyed with its USB port gets its own serial.
func TestInitUsesKeyForSerial(t *testing.T) {
	useFakePSUs(t)
	for _, key := range []string{"7199", "7199-1-3"} {
		dev := Init(0x1b1c, 7199, key, "/dev/hidraw0")
		if dev == nil {
			t.Fatalf("Init(%q) returned nil", key)
		}
		dev.Instance.(*Device).StopDirty()
		if want := serialFromKey("HX1500i", key); dev.Serial != want {
			t.Fatalf("Init(%q) serial = %q, want %q", key, dev.Serial, want)
		}
	}
}

// Two PSUs initialize at once at startup; run with -race.
func TestInitConcurrently(t *testing.T) {
	useFakePSUs(t)
	keys := []string{"7199", "7199-1-3"}
	devs := make([]*common.Device, len(keys))
	var wg sync.WaitGroup
	for i, key := range keys {
		wg.Add(1)
		go func() {
			defer wg.Done()
			devs[i] = Init(0x1b1c, 7199, key, "/dev/hidraw"+strconv.Itoa(i))
		}()
	}
	wg.Wait()
	for i, dev := range devs {
		if dev == nil {
			t.Fatalf("Init(%q) returned nil", keys[i])
		}
		dev.Instance.(*Device).StopDirty()
	}
	if devs[0].Serial == devs[1].Serial {
		t.Fatal("PSUs initialized at once share a serial")
	}
}

// A reply to an earlier request that arrives after the drain is skipped, not returned as an error.
func TestTransferSkipsLateReply(t *testing.T) {
	d := &Device{dev: &lateReplyPSU{&fakePSU{}}}
	output, err := d.transfer(d.createPacket(cmdInit, cmdRead, 0))
	if err != nil {
		t.Fatalf("transfer failed on a late reply to another request: %v", err)
	}
	if output[0] != cmdInit || output[1] != cmdRead {
		t.Fatalf("transfer returned the late reply %x", output[:2])
	}
}

// lateReplyPSU delivers a reply to an earlier request just after the request is written.
type lateReplyPSU struct{ *fakePSU }

func (l *lateReplyPSU) Write(p []byte) (int, error) {
	l.replies = append(l.replies, []byte{cmdRead, dataGetWatts})
	return l.fakePSU.Write(p)
}

// chattyPSU answers every read at once with a reply to some other request, like a PSU another
// program is polling hard; trickleReply answers one read late with such a reply, then goes quiet.
type chattyPSU struct{ fakePSU }
type trickleReply struct {
	fakePSU
	sent bool
}

func (c *chattyPSU) ReadWithTimeout(p []byte, _ time.Duration) (int, error) {
	return copy(p, []byte{cmdRead, dataGetWatts}), nil
}

func (r *trickleReply) ReadWithTimeout(p []byte, timeout time.Duration) (int, error) {
	if timeout > 0 && !r.sent {
		r.sent = true
		time.Sleep(transferTimeout * 6 / 10)
		return copy(p, []byte{cmdRead, dataGetWatts}), nil
	}
	time.Sleep(timeout)
	return 0, hid.ErrTimeout
}

// Neither a stream of other replies nor a slow one may stretch a transfer past its timeout.
func TestTransferTimeoutIsTotal(t *testing.T) {
	for name, dev := range map[string]hidDevice{"chatty": &chattyPSU{}, "trickle": &trickleReply{}} {
		d := &Device{dev: dev}
		done := make(chan error)
		start := time.Now()
		go func() {
			_, err := d.transfer(d.createPacket(cmdInit, cmdRead, 0))
			done <- err
		}()
		select {
		case err := <-done:
			if err == nil {
				t.Fatalf("%s: transfer accepted a reply to another request", name)
			}
			if elapsed := time.Since(start); elapsed > transferTimeout*13/10 {
				t.Fatalf("%s: transfer took %v, timeout is %v", name, elapsed, transferTimeout)
			}
		case <-time.After(3 * transferTimeout):
			t.Fatalf("%s: transfer never gave up", name)
		}
	}
}

// hungPSU never answers a read; unpluggedPSU fails every read, like a PSU pulled mid-init.
type hungPSU struct{ fakePSU }
type unpluggedPSU struct{ fakePSU }

func (h *hungPSU) ReadWithTimeout(p []byte, timeout time.Duration) (int, error) {
	if timeout < 0 {
		select {} // hidapi blocks forever on a negative timeout
	}
	time.Sleep(timeout)
	return 0, hid.ErrTimeout
}

func (u *unpluggedPSU) ReadWithTimeout([]byte, time.Duration) (int, error) {
	return 0, errors.New("device disconnected")
}

// A PSU that never answers must not block another PSU's init, and its own init must give up.
func TestHungPsuDoesNotBlockOthers(t *testing.T) {
	useFakePSUs(t)
	openPath = func(path string) (hidDevice, error) {
		if path == "/dev/hung" {
			return &hungPSU{}, nil
		}
		return &fakePSU{}, nil
	}
	hung := make(chan *common.Device)
	go func() { hung <- Init(0x1b1c, 7199, "7199", "/dev/hung") }()
	time.Sleep(100 * time.Millisecond) // the hung PSU is now waiting on its first read

	done := make(chan *common.Device)
	go func() { done <- Init(0x1b1c, 7199, "7199-1-3", "/dev/hidraw1") }()
	select {
	case dev := <-done:
		if dev == nil {
			t.Fatal("Init of the healthy PSU returned nil")
		}
		dev.Instance.(*Device).StopDirty()
	case <-time.After(transferTimeout / 2):
		t.Fatal("a hung PSU blocked another PSU's init")
	}

	select {
	case dev := <-hung:
		if dev != nil {
			t.Fatal("Init of a PSU that never answers returned a device")
		}
	case <-time.After(5 * transferTimeout):
		t.Fatal("Init of a hung PSU never returned")
	}
}

// A PSU unplugged during init must not be registered: its USB path is gone, so hotplug could
// never remove it again.
func TestInitUnpluggedPsuReturnsNil(t *testing.T) {
	useFakePSUs(t)
	psu := &unpluggedPSU{}
	openPath = func(string) (hidDevice, error) { return psu, nil }
	if dev := Init(0x1b1c, 7199, "7199", "/dev/hidraw0"); dev != nil {
		dev.Instance.(*Device).StopDirty()
		t.Fatal("Init of an unplugged PSU returned a device")
	}
	if !psu.closed {
		t.Fatal("Init left the unplugged PSU's HID handle open")
	}
}

// Replies left queued by a read that timed out (or by another program polling the PSU) must not
// be taken as the answer to the next request, even when they answer the same command.
func TestTransferSkipsStaleReply(t *testing.T) {
	useFakePSUs(t)
	stale := append([]byte{cmdInit, cmdRead}, "CORSAIR RM850i PSU"...)
	psu := &fakePSU{replies: [][]byte{stale, stale, {cmdRead, dataGetWatts}}}
	openPath = func(string) (hidDevice, error) { return psu, nil }
	dev := Init(0x1b1c, 7199, "7199", "/dev/hidraw0")
	if dev == nil {
		t.Fatal("a stale queued reply made Init fail")
	}
	dev.Instance.(*Device).StopDirty()
	if dev.Product != "HX1500i" {
		t.Fatalf("product = %q, want HX1500i", dev.Product)
	}
}

// replugPSU is a fakePSU that can be unplugged: once gone, every write and read fails like a
// dead hidraw node.
type replugPSU struct {
	fakePSU
	gone   bool
	writes int
	sent   [][3]byte // mode, command and data byte of every request written
}

func (r *replugPSU) Write(p []byte) (int, error) {
	r.writes++
	if r.gone {
		return 0, errors.New("No such device")
	}
	r.sent = append(r.sent, [3]byte{p[1], p[2], p[3]})
	return r.fakePSU.Write(p)
}

func (r *replugPSU) ReadWithTimeout(p []byte, timeout time.Duration) (int, error) {
	if r.gone {
		return 0, errors.New("No such device")
	}
	return r.fakePSU.ReadWithTimeout(p, timeout)
}

// psuSeries counts the metrics series published for one PSU
func psuSeries(serial string) int {
	n := 0
	for _, h := range metrics.GetDeviceMetrics() {
		if h.Serial == serial {
			n++
		}
	}
	return n
}

// initFakePSU initializes one PSU on a fake device at the given USB port and stops its
// refresh loop, so the test drives getDeviceData itself.
func initFakePSU(t *testing.T, dev hidDevice, port string) *Device {
	t.Helper()
	useFakePSUs(t)
	oldOpen, oldPort, oldEnum := openPath, usbPortOf, enumeratePaths
	t.Cleanup(func() { openPath, usbPortOf, enumeratePaths = oldOpen, oldPort, oldEnum })
	openPath = func(string) (hidDevice, error) { return dev, nil }
	usbPortOf = func(string) string { return port }

	c := Init(0x1b1c, 7199, "7199", "/dev/hidraw0")
	if c == nil {
		t.Fatal("Init returned nil")
	}
	d := c.Instance.(*Device)
	d.stopAutoRefresh()
	t.Cleanup(func() { metrics.Remove(d.Serial) })
	return d
}

// A PSU that drops off USB and re-enumerates on a new hidraw node, on the same port, is
// reopened by the next refresh. While it is gone its series are dropped, not published as 0.
func TestRefreshReopensReplugedPsu(t *testing.T) {
	old := &replugPSU{}
	d := initFakePSU(t, old, "/devices/usb7/7-10")
	d.getDeviceData()
	if psuSeries(d.Serial) == 0 {
		t.Fatal("no series published for a working PSU")
	}

	old.gone = true
	enumeratePaths = func(uint16, uint16) ([]string, error) { return nil, nil }
	d.getDeviceData()
	if !d.isLost() {
		t.Fatal("PSU not marked lost after its handle failed")
	}
	if n := psuSeries(d.Serial); n != 0 {
		t.Fatalf("%d series still published for an unplugged PSU", n)
	}
	d.getDeviceData() // still gone: no reopen, nothing published
	if psuSeries(d.Serial) != 0 || !d.isLost() {
		t.Fatal("unplugged PSU came back without being re-enumerated")
	}

	fresh := &replugPSU{}
	enumeratePaths = func(uint16, uint16) ([]string, error) { return []string{"/dev/hidraw5"}, nil }
	var opened string
	openPath = func(path string) (hidDevice, error) { opened = path; return fresh, nil }
	d.getDeviceData()
	if d.isLost() || opened != "/dev/hidraw5" {
		t.Fatalf("PSU not reopened (lost=%v, opened %q)", d.isLost(), opened)
	}
	if !old.closed {
		t.Fatal("dead handle not closed")
	}
	if psuSeries(d.Serial) == 0 {
		t.Fatal("no series published after the PSU was reopened")
	}
}

// A PSU on another USB port is never taken for the lost one.
func TestReopenOnlyMatchesSamePort(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	psu.gone = true
	d.getDeviceData()

	enumeratePaths = func(uint16, uint16) ([]string, error) { return []string{"/dev/hidraw6"}, nil }
	usbPortOf = func(string) string { return "/devices/usb7/7-11" }
	openPath = func(string) (hidDevice, error) { t.Fatal("opened the other PSU"); return nil, nil }
	if d.reopen() {
		t.Fatal("reopened a PSU on another port")
	}
}

// With no port recorded at init, a PSU is only reopened when exactly one candidate exists.
func TestReopenWithoutPortNeedsOneCandidate(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "")
	psu.gone = true
	d.getDeviceData()

	enumeratePaths = func(uint16, uint16) ([]string, error) { return []string{"/dev/hidraw5", "/dev/hidraw6"}, nil }
	openPath = func(string) (hidDevice, error) { t.Fatal("opened an ambiguous PSU"); return nil, nil }
	if d.reopen() {
		t.Fatal("reopened with two candidates and no recorded port")
	}
}

// A PSU that stops answering (still on USB: reads time out) keeps its last readings for a
// few refreshes, is not marked lost, and has its series dropped after maxFailedRefreshes.
func TestHungPsuSeriesDroppedAfterFailedRefreshes(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	d.getDeviceData()
	want := psuSeries(d.Serial)
	before := *d.Devices[0]

	oldTimeout := transferTimeout
	transferTimeout = 10 * time.Millisecond
	t.Cleanup(func() { transferTimeout = oldTimeout })
	hung := &hungPSU{}
	d.dev = hung

	for i := 1; i < maxFailedRefreshes; i++ {
		d.getDeviceData()
		if got := psuSeries(d.Serial); got != want {
			t.Fatalf("refresh %d: %d series, want the last %d kept", i, got, want)
		}
	}
	if *d.Devices[0] != before {
		t.Fatal("a failed refresh left zeroed readings in the device state")
	}
	d.getDeviceData()
	if n := psuSeries(d.Serial); n != 0 {
		t.Fatalf("%d series still published after %d failed refreshes", n, maxFailedRefreshes)
	}
	if d.isLost() {
		t.Fatal("a PSU that only timed out was marked lost")
	}
}

// A failed refresh must not short-circuit a later fan write from the API.
func TestFailedRefreshDoesNotDropFanWrite(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	d.refreshFailed = true
	if _, err := d.transfer(d.createPacket(cmdWrite, cmdSetFanMode, dataFanModeManual)); err != nil {
		t.Fatalf("fan write after a failed refresh: %v", err)
	}
}

// A lost handle is not written to again until the PSU is reopened.
func TestLostPsuIsNotWrittenTo(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	psu.gone = true
	enumeratePaths = func(uint16, uint16) ([]string, error) { return nil, nil }
	d.getDeviceData()
	writes := psu.writes
	d.getDeviceData()
	if _, err := d.transfer(d.createPacket(cmdRead, dataFanSpeed, 0)); err == nil {
		t.Fatal("transfer on a lost handle succeeded")
	}
	if psu.writes != writes {
		t.Fatalf("%d writes to a lost handle", psu.writes-writes)
	}
}

// After the first timed-out transfer, the rest of the refresh fails at once instead of each
// request waiting out its own timeout.
func TestHungRefreshFailsFast(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	oldTimeout := transferTimeout
	transferTimeout = 10 * time.Millisecond
	t.Cleanup(func() { transferTimeout = oldTimeout })
	hung := &hungPSU{}
	d.dev = hung

	d.getDeviceData()
	if n := len(hung.replies); n != 1 {
		t.Fatalf("hung refresh sent %d requests, want 1", n)
	}
}

// An HXi reporting "CORSAIR HX1500i Power Supply" is shown as "HX1500i" but keeps the serial
// 0.9.2 gave it (md5 of "HX1500i Power Supply"), so its saved profile still loads.
func TestPowerSupplyTrimKeepsSerial(t *testing.T) {
	d := initFakePSU(t, &replugPSU{fakePSU: fakePSU{name: "CORSAIR HX1500i Power Supply"}}, "")
	if d.Product != "HX1500i" {
		t.Fatalf("product = %q, want %q", d.Product, "HX1500i")
	}
	if want := "26aaedb20ebc87a53bef2b5cca293bf9"; d.Serial != want {
		t.Fatalf("serial = %q, want %q (0.9.2's)", d.Serial, want)
	}
}

// A reopened PSU gets its manual fan mode back: manual mode, then the profile's speed.
func TestReopenRestoresManualFanMode(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	d.DeviceProfile.FanMode = 7
	psu.gone = true
	d.getDeviceData()

	fresh := &replugPSU{}
	enumeratePaths = func(uint16, uint16) ([]string, error) { return []string{"/dev/hidraw5"}, nil }
	openPath = func(string) (hidDevice, error) { return fresh, nil }
	if !d.reopen() {
		t.Fatal("PSU not reopened")
	}
	manual := [3]byte{cmdWrite, cmdSetFanMode, dataFanModeManual}
	speed := [3]byte{cmdWrite, dataSetFanSpeed, 70}
	if len(fresh.sent) < 2 || fresh.sent[0] != manual || fresh.sent[1] != speed {
		t.Fatalf("reopened PSU got %x, want manual mode then 70%%", fresh.sent)
	}
}

// A fan change from the API while the refresh goroutine reopens the PSU; run with -race.
func TestFanChangeDuringReopen(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	psu.gone = true
	d.getDeviceData()
	enumeratePaths = func(uint16, uint16) ([]string, error) { return []string{"/dev/hidraw5"}, nil }
	var openMu sync.Mutex
	openPath = func(string) (hidDevice, error) {
		openMu.Lock()
		defer openMu.Unlock()
		return &replugPSU{}, nil
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			d.UpdatePsuFan(4 + i%7)
		}
	}()
	for i := 0; i < 20; i++ {
		d.reopen()
	}
	wg.Wait()
}

// A failed refresh restores the readings it overwrote, so the API never shows the zeroed decodes.
func TestFailedRefreshRestoresReadings(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	d.Devices[0].Watts, d.Devices[0].Rpm, d.Devices[0].Temperature = 240, 900, 41
	before := *d.Devices[0]

	oldTimeout := transferTimeout
	transferTimeout = 10 * time.Millisecond
	t.Cleanup(func() { transferTimeout = oldTimeout })
	d.dev = &hungPSU{}
	d.getDeviceData()
	if *d.Devices[0] != before {
		t.Fatalf("failed refresh left %+v, want %+v", *d.Devices[0], before)
	}
}

// Only failed refreshes IN A ROW drop the series: a good refresh resets the count.
func TestGoodRefreshResetsFailedRuns(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	oldTimeout := transferTimeout
	transferTimeout = 10 * time.Millisecond
	t.Cleanup(func() { transferTimeout = oldTimeout })
	hung := &hungPSU{}

	for round := 0; round < 3; round++ {
		d.dev = psu
		d.getDeviceData()
		d.dev = hung
		for i := 1; i < maxFailedRefreshes; i++ {
			d.getDeviceData()
		}
		if psuSeries(d.Serial) == 0 {
			t.Fatalf("round %d: series dropped although no %d refreshes failed in a row", round, maxFailedRefreshes)
		}
	}
}

// A handle that dies during an API fan write (not a refresh) still drops the PSU's series.
func TestLossDuringFanWriteDropsSeries(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	d.getDeviceData()
	psu.gone = true
	enumeratePaths = func(uint16, uint16) ([]string, error) { return nil, nil }
	d.UpdatePsuFan(5)
	d.getDeviceData()
	if n := psuSeries(d.Serial); n != 0 {
		t.Fatalf("%d series still published after the handle died in a fan write", n)
	}
}

// hidapi enumerate/open run under hidMutex during a reopen.
func TestReopenHoldsHidMutex(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	psu.gone = true
	d.getDeviceData()
	held := 0
	check := func() {
		if hidMutex.TryLock() {
			hidMutex.Unlock()
		} else {
			held++
		}
	}
	enumeratePaths = func(uint16, uint16) ([]string, error) { check(); return []string{"/dev/hidraw5"}, nil }
	openPath = func(string) (hidDevice, error) { check(); return &replugPSU{}, nil }
	if !d.reopen() || held != 2 {
		t.Fatalf("hidMutex held for %d of 2 hidapi calls", held)
	}
}

// StopDirty (hotplug removal) drops the PSU's series and closes its handle; a late API call
// then does not write to the closed device.
func TestStopDirtyDropsSeriesAndClosesHandle(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	d.getDeviceData()
	d.StopDirty()
	if n := psuSeries(d.Serial); n != 0 {
		t.Fatalf("%d series still published after StopDirty", n)
	}
	if !psu.closed {
		t.Fatal("handle not closed")
	}
	writes := psu.writes
	d.UpdatePsuFan(5)
	if psu.writes != writes {
		t.Fatal("wrote to a removed PSU")
	}
}

// The device API JSON-encodes the PSU while its refresh and fan changes run; run with -race.
func TestEncodeDuringRefresh(t *testing.T) {
	psu := &lockedPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 20; i++ {
			d.getDeviceData()
			d.UpdatePsuFan(4 + i%7)
		}
	}()
	for {
		select {
		case <-done:
			return
		default:
			if _, err := json.Marshal(d); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// lockedPSU is a fakePSU safe to use from the refresh and API goroutines at once
type lockedPSU struct {
	mu sync.Mutex
	fakePSU
}

func (l *lockedPSU) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.fakePSU.Write(p)
}

func (l *lockedPSU) ReadWithTimeout(p []byte, timeout time.Duration) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.replies) == 0 {
		return 0, hid.ErrTimeout
	}
	return l.fakePSU.ReadWithTimeout(p, timeout)
}

// A fan change the PSU never received (dead handle) is reported as failed, not as changed.
func TestFanChangeOnDeadPsuFails(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	if r := d.UpdatePsuFan(5); r != 1 {
		t.Fatalf("UpdatePsuFan on a working PSU = %d, want 1", r)
	}
	psu.gone = true
	if r := d.UpdatePsuFan(6); r != 0 {
		t.Fatalf("UpdatePsuFan on an unplugged PSU = %d, want 0", r)
	}
}

// The usual unplug order: the refresh finds the handle dead before hotplug's StopDirty runs.
// StopDirty still closes it, and only once however often it and Stop are called.
func TestStopDirtyClosesLostHandleOnce(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	psu.gone = true
	enumeratePaths = func(uint16, uint16) ([]string, error) { return nil, nil }
	d.getDeviceData()
	if !d.isLost() {
		t.Fatal("handle not marked lost")
	}
	d.StopDirty()
	if psu.closes != 1 {
		t.Fatalf("StopDirty closed the dead handle %d times, want 1", psu.closes)
	}
	d.StopDirty()
	d.Stop()
	if psu.closes != 1 {
		t.Fatalf("dead handle closed %d times, want 1", psu.closes)
	}
}

// reopen closes the dead handle once, and the fresh one is closed by a later StopDirty.
func TestReopenThenStopDirtyClosesEachHandleOnce(t *testing.T) {
	old := &replugPSU{}
	d := initFakePSU(t, old, "/devices/usb7/7-10")
	old.gone = true
	d.getDeviceData()
	fresh := &replugPSU{}
	enumeratePaths = func(uint16, uint16) ([]string, error) { return []string{"/dev/hidraw5"}, nil }
	openPath = func(string) (hidDevice, error) { return fresh, nil }
	if !d.reopen() {
		t.Fatal("not reopened")
	}
	d.StopDirty()
	d.StopDirty()
	if old.closes != 1 || fresh.closes != 1 {
		t.Fatalf("closes: old %d, fresh %d, want 1 each", old.closes, fresh.closes)
	}
}

// hidMutexProbePSU records whether hidMutex was free while it was written to
type hidMutexProbePSU struct {
	replugPSU
	heldDuringWrite bool
}

func (h *hidMutexProbePSU) Write(p []byte) (int, error) {
	if hidMutex.TryLock() {
		hidMutex.Unlock()
	} else {
		h.heldDuringWrite = true
	}
	return h.replugPSU.Write(p)
}

// The fan restore after a reopen runs without hidMutex, so a slow PSU can't block other PSUs'
// Init or reopen.
func TestReopenFanRestoreWithoutHidMutex(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	d.DeviceProfile.FanMode = 7
	psu.gone = true
	d.getDeviceData()
	fresh := &hidMutexProbePSU{}
	enumeratePaths = func(uint16, uint16) ([]string, error) { return []string{"/dev/hidraw5"}, nil }
	openPath = func(string) (hidDevice, error) { return fresh, nil }
	if !d.reopen() {
		t.Fatal("not reopened")
	}
	if fresh.writes == 0 || fresh.heldDuringWrite {
		t.Fatalf("fan restore: %d writes, hidMutex held: %v", fresh.writes, fresh.heldDuringWrite)
	}
}

// failingAfterPSU fails every write after the first n, like a PSU unplugged mid-command
type failingAfterPSU struct {
	replugPSU
	n int
}

func (f *failingAfterPSU) Write(p []byte) (int, error) {
	if f.writes >= f.n {
		f.writes++
		return 0, errors.New("No such device")
	}
	return f.replugPSU.Write(p)
}

// Every fan-mode write's failure reaches UpdatePsuFan: the default-mode write, and the speed
// write after a successful manual-mode write.
func TestFanChangeReportsEveryFailedWrite(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode int
		n    int // writes that succeed
	}{
		{"default mode", 0, 0},
		{"manual mode", 7, 0},
		{"speed after manual mode", 7, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			psu := &replugPSU{}
			d := initFakePSU(t, psu, "/devices/usb7/7-10")
			d.dev = &failingAfterPSU{n: tc.n}
			if r := d.UpdatePsuFan(tc.mode); r != 0 {
				t.Fatalf("UpdatePsuFan = %d with a failed write, want 0", r)
			}
		})
	}
}

// After Stop, a late API call doesn't write to the closed handle.
func TestNoWriteAfterStop(t *testing.T) {
	psu := &replugPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	d.Stop()
	writes := psu.writes
	d.UpdatePsuFan(5)
	if psu.writes != writes {
		t.Fatal("wrote to a stopped PSU")
	}
}

// The UI page renders a snapshot taken under the lock while refreshes and fan changes run;
// run with -race.
func TestTemplateSnapshotDuringRefresh(t *testing.T) {
	psu := &lockedPSU{}
	d := initFakePSU(t, psu, "/devices/usb7/7-10")
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 20; i++ {
			d.getDeviceData()
			d.UpdatePsuFan(4 + i%7)
		}
	}()
	for {
		select {
		case <-done:
			return
		default:
			c := d.TemplateSnapshot().(*Device)
			_ = c.DeviceProfile.FanMode
			for _, dev := range c.Devices {
				_, _, _ = dev.Name, dev.Watts, dev.TemperatureString
			}
		}
	}
}

// The UI page's copy carries everything web/psuhid.html shows: channels, fan modes, the profile.
func TestTemplateSnapshotFields(t *testing.T) {
	d := initFakePSU(t, &replugPSU{}, "/devices/usb7/7-10")
	d.DeviceProfile.FanMode = 7
	c := d.TemplateSnapshot().(*Device)
	if len(c.Devices) == 0 || len(c.Devices) != len(d.Devices) {
		t.Fatalf("snapshot has %d channels, device %d", len(c.Devices), len(d.Devices))
	}
	if len(c.FanModes) != len(d.FanModes) || c.DeviceProfile == nil || c.DeviceProfile.FanMode != 7 {
		t.Fatalf("snapshot fan modes %v, profile %+v", c.FanModes, c.DeviceProfile)
	}
	if c.Product != d.Product || c.Serial != d.Serial || c.InputVoltage != d.InputVoltage {
		t.Fatalf("snapshot %q/%q/%v, device %q/%q/%v", c.Product, c.Serial, c.InputVoltage, d.Product, d.Serial, d.InputVoltage)
	}
}

// silentManualPSU never answers the manual fan mode command (its read times out, which does not
// mark the handle lost) but answers everything else.
type silentManualPSU struct{ replugPSU }

func (s *silentManualPSU) Write(p []byte) (int, error) {
	if p[1] == cmdWrite && p[2] == cmdSetFanMode && p[3] == dataFanModeManual {
		return len(p), nil
	}
	return s.replugPSU.Write(p)
}

// A manual fan mode the PSU didn't acknowledge is reported as failed even though the speed
// write that follows would be answered.
func TestUnansweredManualModeReported(t *testing.T) {
	d := initFakePSU(t, &replugPSU{}, "/devices/usb7/7-10")
	oldTimeout := transferTimeout
	transferTimeout = 10 * time.Millisecond
	t.Cleanup(func() { transferTimeout = oldTimeout })
	d.dev = &silentManualPSU{}
	if r := d.UpdatePsuFan(7); r != 0 {
		t.Fatalf("UpdatePsuFan = %d with the manual mode unanswered, want 0", r)
	}
}
