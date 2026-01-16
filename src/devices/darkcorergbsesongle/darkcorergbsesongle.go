package darkcorergbsesongle

// Package: CORSAIR DARK CORE RGB SE Wireless USB Receiver
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices/darkcorergbseW"
	"OpenLinkHub/src/logger"
	"encoding/binary"
	"fmt"
	"github.com/sstallion/go-hid"
	"strconv"
	"sync"
	"time"
)

type Device struct {
	Debug         bool
	dev           *hid.Device
	listener      *hid.Device
	Manufacturer  string `json:"manufacturer"`
	Product       string `json:"product"`
	Serial        string `json:"serial"`
	Firmware      string `json:"firmware"`
	VendorId      uint16
	ProductId     uint16
	mutex         sync.Mutex
	PairedDevices map[uint16]any
	SharedDevices func(device *common.Device)
	DeviceList    map[string]*common.Device
	instance      *common.Device
	Exit          bool
}

var (
	cmdSoftwareMode = []byte{0x04, 0x02}
	cmdHardwareMode = []byte{0x04, 0x01}
	cmdGetFirmware  = byte(0x01)
	cmdRead         = byte(0x0e)
	cmdDeviceStatus = byte(0x51)
	bufferSize      = 64
	bufferSizeWrite = bufferSize + 1
	headerSize      = 2
	firmwareIndex   = 9
)

func Init(vendorId, productId uint16, _, path string, callback func(device *common.Device)) *common.Device {
	dev, err := hid.OpenPath(path)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId}).Error("Unable to open HID device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:           dev,
		VendorId:      vendorId,
		ProductId:     productId,
		PairedDevices: make(map[uint16]any),
		DeviceList:    make(map[string]*common.Device),
		SharedDevices: callback,
	}

	d.getDebugMode()         // Debug mode
	d.getManufacturer()      // Manufacturer
	d.getSerial()            // Serial
	d.setSoftwareMode()      // Software mode
	d.addDevices()           // Add devices
	d.backendListener()      // Control listener
	d.createDevice()         // Device register
	d.initAvailableDevices() // Init devices
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// addDevices adda a mew device
func (d *Device) addDevices() {
	var pid uint16 = 6987
	dev := darkcorergbseW.Init(
		d.VendorId,
		pid,
		d.dev,
		strconv.Itoa(int(pid)),
	)

	object := &common.Device{
		ProductType: common.ProductTypeDarkCoreRgbSEW,
		Product:     "DARK CORE SE",
		Serial:      dev.Serial,
		Firmware:    dev.Firmware,
		Image:       "icon-mouse.svg",
		Instance:    dev,
	}

	d.SharedDevices(object)
	d.AddPairedDevice(pid, dev, object)
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeDongle,
		Product:     "DONGLE",
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-dongle.svg",
		Instance:    d,
		Hidden:      true,
	}
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")

	for _, value := range d.PairedDevices {
		if dev, found := value.(*darkcorergbseW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
	}

	d.setHardwareMode()
	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
		}
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
}

// StopDirty will stop device in a dirty way
func (d *Device) StopDirty() uint8 {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device (dirty)...")

	for _, val := range d.PairedDevices {
		if device, found := val.(*darkcorergbseW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
	}

	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 2
}

// GetDevice will return HID device
func (d *Device) GetDevice() *hid.Device {
	return d.dev
}

// AddPairedDevice will add a paired device
func (d *Device) AddPairedDevice(productId uint16, device any, dev *common.Device) {
	d.PairedDevices[productId] = device
	d.DeviceList[dev.Serial] = dev
}

// getManufacturer will return device manufacturer
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
}

// getManufacturer will return device manufacturer
func (d *Device) getManufacturer() {
	manufacturer, err := d.dev.GetMfrStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to get manufacturer")
	}
	d.Manufacturer = manufacturer
}

// getSerial will return device serial number
func (d *Device) getSerial() {
	serial, err := d.dev.GetSerialNbr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to get device serial number")
	}
	d.Serial = serial
}

// setHardwareMode will switch a device to hardware mode
func (d *Device) setHardwareMode() {
	err := d.transfer(cmdHardwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	buf := make([]byte, bufferSizeWrite)
	buf[1] = cmdRead
	buf[2] = cmdGetFirmware
	n, err := d.dev.SendFeatureReport(buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device firmware feature report")
		return
	}

	n, err = d.dev.GetFeatureReport(buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device firmware feature report")
		return
	}
	output := buf[:n]
	v1, v2 := fmt.Sprintf("%x", output[firmwareIndex+1]), fmt.Sprintf("%x", output[firmwareIndex])
	d.Firmware = v1 + "." + v2
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	err := d.transfer(cmdSoftwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// getDeviceStatus will fetch device status
func (d *Device) getDeviceStatus() []byte {
	buf := make([]byte, bufferSizeWrite)
	buf[1] = cmdRead
	buf[2] = cmdDeviceStatus

	n, err := d.dev.SendFeatureReport(buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device firmware feature report")
		return nil
	}

	n, err = d.dev.GetFeatureReport(buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device firmware feature report")
		return nil
	}
	output := buf[:n]
	return output
}

// initAvailableDevices will run on initial start
func (d *Device) initAvailableDevices() {
	status := d.getDeviceStatus()
	if status == nil || len(status) == 0 {
		return
	}
	if status[5] == 0x02 {
		for _, val := range d.PairedDevices {
			if device, found := val.(*darkcorergbseW.Device); found {
				if !device.Connected {
					device.Connect()
				}
			}
		}
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint, buffer []byte) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = 0x07
	endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
	}

	if _, err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return err
	}
	return nil
}

// getListenerData will listen for keyboard events and return data on success or nil on failure.
// ReadWithTimeout is mandatory due to the nature of listening for events
func (d *Device) getListenerData() []byte {
	data := make([]byte, bufferSize)
	n, err := d.listener.ReadWithTimeout(data, 100*time.Millisecond)
	if err != nil || n == 0 {
		return nil
	}
	return data
}

// backendListener will listen for events from the device
func (d *Device) backendListener() {
	go func() {
		enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
			if info.InterfaceNbr == 0 {
				listener, err := hid.OpenPath(info.Path)
				if err != nil {
					return err
				}
				d.listener = listener
			}
			return nil
		})

		err := hid.Enumerate(d.VendorId, d.ProductId, enum)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to enumerate devices")
		}

		for {
			select {
			default:
				if d.Exit {
					err = d.listener.Close()
					if err != nil {
						logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Failed to close listener")
						return
					}
					return
				}

				data := d.getListenerData()
				if len(data) == 0 || data == nil {
					continue
				}

				// Reduce packet spam
				if data[5] > 0 || data[7] > 0 || data[1] == 1 {
					continue
				}

				if data[0] == 0x04 {
					status := d.getDeviceStatus()
					if len(status) == 0 || status == nil {
						continue
					}

					if status[5] == 0x02 {
						for _, val := range d.PairedDevices {
							if device, found := val.(*darkcorergbseW.Device); found {
								if !device.Connected {
									time.Sleep(2000 * time.Millisecond)
									device.Connect()
									d.SharedDevices(d.DeviceList[device.Serial])
								}
							}
						}
					} else {
						for _, val := range d.PairedDevices {
							if device, found := val.(*darkcorergbseW.Device); found {
								if device.Connected {
									device.SetConnected(false)
								}
							}
						}
					}

					for _, val := range d.PairedDevices {
						if device, found := val.(*darkcorergbseW.Device); found {
							if device.Connected {
								device.GetBatteryLevelData()
							}
						}
					}
				}

				if data[0] == 0x03 {
					buf := make([]byte, 2)
					buf[0] = data[1]
					buf[1] = data[4]
					val := binary.LittleEndian.Uint16(buf)

					for _, value := range d.PairedDevices {
						if dev, found := value.(*darkcorergbseW.Device); found {
							dev.TriggerKeyAssignment(val)
						}
					}
				}
			}
		}
	}()
}
