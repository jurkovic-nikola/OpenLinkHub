package monitor

// Package: monitor
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/audio"
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/openrgb"
	"github.com/godbus/dbus/v5"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	NetlinkKernelObjectUEvent = 15
	bufferSize                = 2048
	vendorId                  = uint16(6940)  // Corsair
	scufVendorId              = uint16(11925) // Scuf
)

// exclude list of device that are not supported via USB mode
var exclude = []uint16{10752, 2666, 2710, 2659}

// virtuosoXTFamily maps the wired-USB PID to its wireless-dongle PID.
// PID 2658 (0x0a62) = VIRTUOSO XT wired USB cable  → PipeWire "USB Gaming Headset"
// PID 2660 (0x0a64) = VIRTUOSO XT wireless dongle  → PipeWire "Wireless Gaming Receiver"
// Both can be present simultaneously; plugging/unplugging the cable fires hotplug for 2658.
var virtuosoXTFamily = map[uint16]uint16{
	2658: 2660, // wired → wireless fallback PID
}

// pipewireNodeNames are the PipeWire sink description fragments for each PID's sink.
var pipewireNodeNames = map[uint16]string{
	2658: "USB Gaming Headset",
	2660: "Wireless Gaming Receiver",
}

// logind is the service that sends PrepareForSleep; any other sender must be ignored
const (
	logindName            = "org.freedesktop.login1"
	logindPath            = dbus.ObjectPath("/org/freedesktop/login1")
	prepareForSleepMember = "org.freedesktop.login1.Manager.PrepareForSleep"
)

// The bus daemon reports a change of logind's owner with NameOwnerChanged; peers cannot send as it
const (
	busName                = "org.freedesktop.DBus"
	busPath                = dbus.ObjectPath("/org/freedesktop/DBus")
	nameOwnerChangedMember = "org.freedesktop.DBus.NameOwnerChanged"
)

// longSleep list of devices that require 5+ seconds to finish booting up
var longSleep = []uint16{7165, 7033}

type USBInfo struct {
	VendorID  uint16
	ProductID uint16
	Serial    string
}

// sleep is set by the D-Bus PrepareForSleep goroutine and read by the netlink uevent goroutine.
var sleep atomic.Bool

// sysRoot, pause and the device calls below are variables so tests can fake sysfs, skip the
// settle delays and observe what a uevent does to the devices without real hardware.
var (
	sysRoot             = "/sys"
	pause               = time.Sleep
	initManual          = devices.InitManual
	takeSerialByDevPath = devices.TakeSerialByDevPath
	stopDirty           = devices.StopDirty
	stopDevices         = devices.Stop
	stopInputs          = inputmanager.Stop
	exit                = os.Exit
	warn                = func(fields logger.Fields, msg string) { logger.Log(fields).Warn(msg) }
)

func Init() {
	go func() {
		// Connect to the system bus. The default signal handler stops keeping signals in order
		// once ch is full, and the cached logind owner relies on NameOwnerChanged arriving before
		// the new owner's PrepareForSleep
		conn, err := dbus.ConnectSystemBus(dbus.WithSignalHandler(dbus.NewSequentialSignalHandler()))
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to connect to system bus")
			return
		}
		defer func(conn *dbus.Conn) {
			err = conn.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Error closing dbus")
			}
		}(conn)

		listenForSleep(conn.Signal, func(rule string) error {
			return conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, rule).Store()
		}, func() (string, dbus.Sequence, error) {
			return lookupLogind(conn.BusObject())
		})
	}()

	go func() {
		cache := make(map[string]USBInfo)

		// Populate cache on initial start
		for _, product := range devices.GetProducts() {
			if slices.Contains(exclude, product.ProductId) {
				continue
			}
			cache[product.DevPath] = USBInfo{
				VendorID:  vendorId,
				ProductID: product.ProductId,
				Serial:    product.Serial,
			}
		}

		fd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_DGRAM, NetlinkKernelObjectUEvent)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Error opening sysfs socket")
			return
		}
		defer func(fd int) {
			err = syscall.Close(fd)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Error closing sysfs socket")
			}
		}(fd)

		sa := &syscall.SockaddrNetlink{
			Family: syscall.AF_NETLINK,
			Groups: 1,
		}

		if err = syscall.Bind(fd, sa); err != nil {
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to bind to sysfs socket")
			}
			return
		}

		logger.Log(logger.Fields{}).Info("Starting USB monitor...")

		for {
			buf := make([]byte, bufferSize)
			n, _, err := syscall.Recvfrom(fd, buf, 0)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Error receiving USB monitor data")
				continue
			}

			handleUEvent(common.ParseUEvent(buf[:n]), cache)
		}
	}()
}

// listenForSleep will register for signals through subscribe, add the sleep match rules through
// addMatch and handle PrepareForSleep until the signal channel closes. owner looks up logind's
// unique bus name, as watchSleepSignals describes. The bus calls are passed in so tests can run
// it without a bus.
func listenForSleep(subscribe func(chan<- *dbus.Signal), addMatch func(rule string) error, owner func() (string, dbus.Sequence, error)) {
	// Listen for the PrepareForSleep signal
	ch := make(chan *dbus.Signal, 10)
	subscribe(ch)

	watchSleepSignals(ch, owner, addSleepMatches(addMatch))
}

// lookupLogind will ask bus for logind's unique bus name, and return it with the sequence of
// the reply that carried it.
func lookupLogind(bus dbus.BusObject) (string, dbus.Sequence, error) {
	var owner string
	call := bus.Call("org.freedesktop.DBus.GetNameOwner", 0, logindName)
	err := call.Store(&owner)
	return owner, call.ResponseSequence, err
}

// addSleepMatches will ask the bus, through addMatch, for logind's PrepareForSleep and for
// changes of logind's owner. It returns whether logind's owner can be cached: only when owner
// changes are delivered, otherwise a restarted or replaced logind would never be recognised.
func addSleepMatches(addMatch func(rule string) error) bool {
	match := "type='signal',sender='" + logindName + "',path='" + string(logindPath) + "',interface='org.freedesktop.login1.Manager',member='PrepareForSleep'"
	if err := addMatch(match); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Failed to add D-Bus match")
	}

	// Follow logind's owner so it is looked up once, not once per PrepareForSleep
	match = "type='signal',sender='" + busName + "',path='" + string(busPath) + "',interface='" + busName + "',member='NameOwnerChanged',arg0='" + logindName + "'"
	if err := addMatch(match); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Failed to add D-Bus match. logind is looked up per PrepareForSleep")
		return false
	}
	return true
}

// watchSleepSignals will handle each PrepareForSleep on ch that logind sent. owner looks up
// logind's current unique bus name and the sequence of the reply that carried it. When
// cacheOwner is set, the name is cached until a NameOwnerChanged for logind arrives on ch.
// Otherwise it is trusted only for signals received before that reply: the bus routed them
// before answering, so each is judged by logind's owner at some point after it was sent, as a
// lookup per signal would. A queued flood then costs one lookup, not one per spoof.
func watchSleepSignals(ch <-chan *dbus.Signal, owner func() (string, dbus.Sequence, error), cacheOwner bool) {
	// conn.Signal delivers every signal on the connection, whatever the match rule says,
	// and any peer can name itself as the sender's interface. Only trust logind's unique
	// name, so a restarted logind is still recognised once its owner is looked up again.
	// Caching it keeps a flood of spoofs from costing a bus round trip each.
	// Spoofs are logged once, and each logind outage once, so a flood cannot fill the log.
	warnedSpoof, warnedLookup := false, false
	name, known := "", false
	var answered dbus.Sequence
	for signal := range ch {
		if logindOwnerChanged(signal) {
			known = false
			continue
		}
		if signal.Name != prepareForSleepMember {
			continue
		}
		if known && !cacheOwner && signal.Sequence >= answered {
			known = false
		}
		if !known {
			var err error
			name, answered, err = owner()
			if err != nil {
				if !warnedLookup {
					warnedLookup = true
					warn(logger.Fields{"error": err, "sender": signal.Sender}, "Unable to resolve logind on D-Bus. Ignoring PrepareForSleep")
				}
				continue
			}
			warnedLookup = false
			known = true
		}
		if !fromLogind(signal, name) {
			if !warnedSpoof {
				warnedSpoof = true
				warn(logger.Fields{"sender": signal.Sender, "path": signal.Path}, "Ignoring PrepareForSleep not sent by logind. Further ones are ignored silently")
			}
			continue
		}
		handleSleepSignal(signal)
	}
}

// logindOwnerChanged will return true when signal is the bus daemon reporting that logind's
// unique bus name changed.
func logindOwnerChanged(signal *dbus.Signal) bool {
	if signal.Sender != busName || signal.Path != busPath || signal.Name != nameOwnerChangedMember || len(signal.Body) == 0 {
		return false
	}
	name, ok := signal.Body[0].(string)
	return ok && name == logindName
}

// fromLogind will return true when signal is PrepareForSleep sent by logind, whose unique bus
// name is owner.
func fromLogind(signal *dbus.Signal, owner string) bool {
	return owner != "" && signal.Sender == owner && signal.Path == logindPath && signal.Name == prepareForSleepMember
}

// handleSleepSignal will stop all devices when a PrepareForSleep signal reports a suspend,
// and exit the process when it reports a resume.
func handleSleepSignal(signal *dbus.Signal) {
	if len(signal.Body) == 0 {
		return
	}
	isSleeping, ok := signal.Body[0].(bool)
	if !ok {
		sleep.Store(false)
		return
	}
	sleep.Store(isSleeping)
	if isSleeping {
		logger.Log(logger.Fields{}).Info("Suspend detected. Sending Stop() to all devices")

		// Cleanup
		if config.GetConfig().EnableOpenRGBTargetServer {
			openrgb.Close()
			openrgb.ClearDeviceControllers()
		}

		// Stop
		stopDevices()
		stopInputs()
	} else {
		pause(time.Duration(config.GetConfig().ResumeDelay) * time.Millisecond)
		logger.Log(logger.Fields{}).Info("Resume detected. Process is shutting down...")

		// Due to the issues encountered when sleeping and resuming, this is the best way to handle
		// the resume, as systemd will pick up non-zero exit codes and restart on failure.
		// If you're reading this and thinking a resume should work, good luck.
		// Enough time was spent on tweaking this and trying to do something that makes no sense;
		// just terminate the process and let systemd do the magic.
		exit(1)
	}
}

// handleUEvent will add or remove the Corsair or Scuf USB device a kernel uevent reports.
// cache maps USB device paths to the devices known to be plugged there.
func handleUEvent(msg map[string]string, cache map[string]USBInfo) {
	if msg["SUBSYSTEM"] != "usb" || msg["DEVTYPE"] != "usb_device" {
		return
	}

	action := msg["ACTION"]
	devPath := msg["DEVPATH"]

	if devPath == "" {
		return
	}

	switch action {
	case "add":
		{
			if sleep.Load() {
				break
			}
			basePath := sysRoot + devPath
			vendor := common.ReadFile(basePath + "/idVendor")
			vid := common.PidVidToUint16(vendor)
			if vid == vendorId || vid == scufVendorId {
				product := common.ReadFile(basePath + "/idProduct")
				pid := common.PidVidToUint16(product)

				if slices.Contains(exclude, pid) {
					return
				}
				serial := common.ReadFile(basePath + "/serial")
				cache[devPath] = USBInfo{
					VendorID:  vid,
					ProductID: pid,
					Serial:    serial,
				}

				pause(7000 * time.Millisecond)
				if slices.Contains(longSleep, pid) {
					pause(2000 * time.Millisecond)
				}
				logger.Log(logger.Fields{"vendorId": vid, "productId": pid, "serial": serial}).Info("Init USB device...")
				initManual(pid, serial, devPath)
				switchHeadsetAudioSink(pid)
			}
		}
		break
	case "remove":
		{
			if !sleep.Load() {
				pause(100 * time.Millisecond)
				info, cached := cache[devPath]
				registered, taken := "", false
				if !cached {
					info, registered, taken = portlessUSBInfo(msg, devPath)
				}
				if !cached && !taken {
					logger.Log(logger.Fields{"path": devPath}).Info("Trying to remove non-existing device")
					return
				}

				if info.VendorID == vendorId || info.VendorID == scufVendorId {
					serial := info.Serial
					if len(serial) < 1 {
						serial = strconv.Itoa(int(info.ProductID))
					}

					// Prefer the key the device registered under; it can differ from the USB serial
					if !taken {
						registered, taken = takeSerialByDevPath(devPath, info.ProductID)
					}
					if taken {
						serial = registered
					}
					logger.Log(logger.Fields{"vendorId": info.VendorID, "productId": info.ProductID, "serial": serial}).Info("Dirty USB removal...")

					stopDirty(serial, info.ProductID)
					delete(cache, devPath)
					openrgb.NotifyControllerChange(serial)
					fallbackHeadsetAudioSink(info.ProductID)
				}
			}
		}
		break
	}
	pause(40 * time.Millisecond)
}

// portlessUSBInfo will return the USB info and registered serial of a removed device that
// registered without a known USB port. The cache built at startup has no entry under its USB path.
func portlessUSBInfo(msg map[string]string, devPath string) (USBInfo, string, bool) {
	ids := strings.Split(msg["PRODUCT"], "/") // vendor/product/bcdDevice in hex
	if msg["DEVTYPE"] != "usb_device" || len(ids) < 2 {
		return USBInfo{}, "", false
	}
	vid, pid := common.PidVidToUint16(ids[0]), common.PidVidToUint16(ids[1])
	if (vid != vendorId && vid != scufVendorId) || slices.Contains(exclude, pid) {
		return USBInfo{}, "", false
	}
	serial, ok := takeSerialByDevPath(devPath, pid)
	return USBInfo{VendorID: vid, ProductID: pid}, serial, ok
}

// switchHeadsetAudioSink routes the virtual audio sink to the wired headset when it is plugged in.
func switchHeadsetAudioSink(pid uint16) {
	if _, ok := virtuosoXTFamily[pid]; !ok {
		return
	}
	fragment, ok := pipewireNodeNames[pid]
	if !ok {
		return
	}
	routeAudioSinkByFragment(fragment)
}

// fallbackHeadsetAudioSink falls back to the wireless sink when the wired headset is removed.
func fallbackHeadsetAudioSink(pid uint16) {
	wirelessPID, ok := virtuosoXTFamily[pid]
	if !ok {
		return
	}
	fragment, ok := pipewireNodeNames[wirelessPID]
	if !ok {
		return
	}
	routeAudioSinkByFragment(fragment)
}

// routeAudioSinkByFragment finds the PipeWire sink whose name or description contains
// the given fragment and sets it as the virtual audio target.
func routeAudioSinkByFragment(fragment string) {
	if !audio.GetAudio().Enabled {
		return
	}
	for _, sink := range audio.GetSinks() {
		if strings.Contains(sink.Name, fragment) || strings.Contains(sink.Desc, fragment) {
			// Value copy so we don't mutate the live audio struct outside the mutex
			current := *audio.GetAudio()
			current.SinkSerial = int(sink.Serial)
			current.SinkName = sink.Name
			current.SinkDesc = sink.Desc
			result := audio.UpdateTargetDevice(&current)
			if result == 0 {
				logger.Log(logger.Fields{"sink": sink.Name}).Warn("Unable to switch virtual audio sink")
			} else {
				logger.Log(logger.Fields{"sink": sink.Name}).Info("Virtual audio sink switched")
			}
			return
		}
	}
	logger.Log(logger.Fields{"fragment": fragment}).Warn("No matching PipeWire sink found for headset auto-switch")
}
