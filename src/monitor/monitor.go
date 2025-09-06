package monitor

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/devices/lcd"
	"OpenLinkHub/src/logger"
	"slices"
	"syscall"
	"time"

	"OpenLinkHub/src/openrgb"
	"github.com/godbus/dbus/v5"
	"strconv"
)

const (
	NetlinkKernelObjectUEvent        = 15
	bufferSize                       = 2048
	sysRoot                          = "/sys"
	vendorId                  uint16 = 6940 // Corsair
)

// exclude list of device that are not supported via USB mode
var exclude = []uint16{10752, 2666, 2710, 2659}

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

						// Stop
						devices.Stop()
					} else {
						time.Sleep(time.Duration(config.GetConfig().ResumeDelay) * time.Millisecond)
						logger.Log(logger.Fields{}).Info("Resume detected. Sending Init() to all devices")

						// Init LCDs
						lcd.Reconnect()

						// Init devices
						devices.Init()
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
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Error opening sysfs socket")
			}
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
					if vid == vendorId {
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

						time.Sleep(5000 * time.Millisecond)
						if slices.Contains(longSleep, pid) {
							time.Sleep(2000 * time.Millisecond)
						}
						logger.Log(logger.Fields{"vendorId": vid, "productId": pid, "serial": serial}).Info("Init USB device...")
						devices.InitManual(pid, serial)
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

						if info.VendorID == vendorId {
							serial := info.Serial
							if len(serial) < 1 {
								serial = strconv.Itoa(int(info.ProductID))
							}
							logger.Log(logger.Fields{"vendorId": info.VendorID, "productId": info.ProductID, "serial": serial}).Info("Dirty USB removal...")

							devices.StopDirty(serial, info.ProductID)
							delete(cache, devPath)
							openrgb.NotifyControllerChange(serial)
						}
					}
				}
				break
			}
			time.Sleep(40 * time.Millisecond)
		}
	}()
}
