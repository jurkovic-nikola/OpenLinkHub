package devices

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/rgb"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// iterDev answers every method the package's device-map iterators call
type iterDev struct{ calls atomic.Int64 }

func (q *iterDev) StopDirty() uint8                                   { return 1 }
func (q *iterDev) GetRgbProfiles() interface{}                        { q.calls.Add(1); return 1 }
func (q *iterDev) UpdateRgbProfile(c int, p string) uint8             { q.calls.Add(1); return 1 }
func (q *iterDev) UpdateRgbProfileData(p string, r rgb.Profile) uint8 { q.calls.Add(1); return 1 }
func (q *iterDev) SchedulerBrightness(m uint8) uint8                  { q.calls.Add(1); return 1 }
func (q *iterDev) SchedulerLcdBrightness(m uint8) uint8               { q.calls.Add(1); return 1 }
func (q *iterDev) ControlDeviceRgb(m bool)                            { q.calls.Add(1) }
func (q *iterDev) GetDeviceLedData() interface{}                      { q.calls.Add(1); return 1 }
func (q *iterDev) GetTemperatureProbes() interface{}                  { q.calls.Add(1); return 1 }
func (q *iterDev) ResetSpeedProfiles(p string) uint8                  { q.calls.Add(1); return 1 }
func (q *iterDev) Stop()                                              { q.calls.Add(1) }

// Every reader of the device map, and shutdown's Stop, runs while hotplug adds and removes
// devices (including the stale-PSU path that drops a device while holding mutex). Run with -race:
// an unlocked iteration is a data race, and in production a fatal concurrent map write.
func TestAllIteratorsWhileHotplugging(t *testing.T) {
	inst := &iterDev{}
	stop := make(chan struct{})
	var writers sync.WaitGroup
	for w := 0; w < 2; w++ {
		writers.Add(1)
		go func(w int) {
			defer writers.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
				}
				s := fmt.Sprintf("it%d-%d", w, i%32)
				switch i % 4 {
				case 0:
					addDevice(&common.Device{Serial: s, ProductType: common.ProductTypeLinkHub, Instance: inst})
				case 1:
					addDevice(&common.Device{Serial: s, DeviceType: common.DeviceTypeMouse}) // no instance
				default:
					deleteDevice(s)
				}
				if i%7 == 0 { // forgetStale: drops the device under mutex, then devicesMu
					h := "/it/hid" + s
					reserveUSBDevPath("", h, "k1"+s, 7199)
					addDevice(&common.Device{Serial: s, ProductType: common.ProductTypeLinkHub, Instance: inst})
					trackUSBDevPath(h, &common.Device{Serial: s})
					reserveUSBDevPath("", h, "k2"+s, 7199)
					TakeSerialByDevPath(h, 7199)
				}
			}
		}(w)
	}

	readers := []func(){
		func() { _ = GetRgbProfiles() },
		func() { UpdateGlobalRgbProfile("static") },
		func() { UpdateAllDevicesStaticColor(rgb.Color{Red: 1}) },
		func() { ResetSpeedProfiles("Normal") },
		func() { ScheduleDeviceBrightness(1); ScheduleDeviceLcdBrightness(0) },
		func() { ControlDeviceRgb(true) },
		func() { _ = GetDevicesLedData() },
		func() { _ = GetTemperatureProbes() },
		func() { _ = GetDevicesEx(); _ = GetMouse(); _ = GetDevice("it0-1"); _ = GetDevices() },
		func() { CallDeviceMethod("it0-1", "Stop"); CallDeviceMethod("it1-1", "Stop") },
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		for _, f := range readers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 200; i++ {
					f()
				}
			}()
		}
		wg.Wait()
	}()
	select {
	case <-done:
	case <-time.After(60 * time.Second):
		t.Fatal("device map readers stuck (deadlock?)")
	}

	Stop() // shutdown while hotplug still runs
	close(stop)
	writers.Wait()
	initWG.Wait()
	Stop()
	if n := len(deviceSnapshot()); n != 0 {
		t.Fatalf("%d devices left after Stop", n)
	}
}
