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
	"syscall"
	"time"
)

const (
	NetlinkKernelObjectUEvent = 15
	bufferSize                = 2048
	sysRoot                   = "/sys"
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

// longSleep list of devices that require 5+ seconds to finish booting up
var longSleep = []uint16{7165, 7033}

type USBInfo struct {
	VendorID  uint16
	ProductID uint16
	Serial    string
}

var sleep bool

func Init() {
	go func() {
		// Connect to the session bus
		conn, err := dbus.ConnectSystemBus()
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

		// Listen for the PrepareForSleep signal
		_ = conn.Object("org.freedesktop.login1", "/org/freedesktop/login1")
		ch := make(chan *dbus.Signal, 10)
		conn.Signal(ch)

		match := "type='signal',interface='org.freedesktop.login1.Manager',member='PrepareForSleep'"
		err = conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, match).Store()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Failed to add D-Bus match")
		}
		for signal := range ch {
			if len(signal.Body) > 0 {
				if isSleeping, ok := signal.Body[0].(bool); ok {
					sleep = isSleeping
					if isSleeping {
						logger.Log(logger.Fields{}).Info("Suspend detected. Sending Stop() to all devices")

						// Cleanup
						if config.GetConfig().EnableOpenRGBTargetServer {
							openrgb.Close()
							openrgb.ClearDeviceControllers()
						}

						// Stop
						devices.Stop()
						inputmanager.Stop()
					} else {
						time.Sleep(time.Duration(config.GetConfig().ResumeDelay) * time.Millisecond)
						logger.Log(logger.Fields{}).Info("Resume detected. Process is shutting down...")

						// Due to the issues encountered when sleeping and resuming, this is the best way to handle
						// the resume, as systemd will pick up non-zero exit codes and restart on failure.
						// If you're reading this and thinking a resume should work, good luck.
						// Enough time was spent on tweaking this and trying to do something that makes no sense;
						// just terminate the process and let systemd do the magic.
						os.Exit(1)
					}
				} else {
					sleep = false
				}
			}
		}
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

			msg := common.ParseUEvent(buf[:n])
			if msg["SUBSYSTEM"] != "usb" || msg["DEVTYPE"] != "usb_device" {
				continue
			}

			action := msg["ACTION"]
			devPath := msg["DEVPATH"]

			if devPath == "" {
				continue
			}

			switch action {
			case "add":
				{
					if sleep {
						break
					}
					basePath := sysRoot + devPath
					vendor := common.ReadFile(basePath + "/idVendor")
					vid := common.PidVidToUint16(vendor)
					if vid == vendorId || vid == scufVendorId {
						product := common.ReadFile(basePath + "/idProduct")
						pid := common.PidVidToUint16(product)

						if slices.Contains(exclude, pid) {
							continue
						}
						serial := common.ReadFile(basePath + "/serial")
						cache[devPath] = USBInfo{
							VendorID:  vid,
							ProductID: pid,
							Serial:    serial,
						}

						time.Sleep(7000 * time.Millisecond)
						if slices.Contains(longSleep, pid) {
							time.Sleep(2000 * time.Millisecond)
						}
						logger.Log(logger.Fields{"vendorId": vid, "productId": pid, "serial": serial}).Info("Init USB device...")
						devices.InitManual(pid, serial)
						switchHeadsetAudioSink(pid)
					}
				}
				break
			case "remove":
				{
					if !sleep {
						time.Sleep(100 * time.Millisecond)
						info, ok := cache[devPath]
						if !ok {
							logger.Log(logger.Fields{"path": devPath}).Info("Trying to remove non-existing device")
							continue
						}

						if info.VendorID == vendorId || info.VendorID == scufVendorId {
							serial := info.Serial
							if len(serial) < 1 {
								serial = strconv.Itoa(int(info.ProductID))
							}
							logger.Log(logger.Fields{"vendorId": info.VendorID, "productId": info.ProductID, "serial": serial}).Info("Dirty USB removal...")

							devices.StopDirty(serial, info.ProductID)
							delete(cache, devPath)
							openrgb.NotifyControllerChange(serial)
							fallbackHeadsetAudioSink(info.ProductID)
						}
					}
				}
				break
			}
			time.Sleep(40 * time.Millisecond)
		}
	}()
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
