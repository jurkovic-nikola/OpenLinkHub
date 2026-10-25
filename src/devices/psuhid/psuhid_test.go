package psuhid

import (
	"OpenLinkHub/src/common"
	"crypto/md5"
	"encoding/hex"
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
}

func (f *fakePSU) GetMfrStr() (string, error) { return "CORSAIR", nil }
func (f *fakePSU) Close() error               { f.closed = true; return nil }

func (f *fakePSU) Write(p []byte) (int, error) {
	reply := make([]byte, readBufferSize)
	reply[0], reply[1] = p[1], p[2]
	if p[1] == cmdInit && p[2] == cmdRead {
		copy(reply[2:], "CORSAIR HX1500i PSU")
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
