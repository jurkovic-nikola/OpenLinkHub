package voidelitedongle

// Package: Headset Dongle
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices/voideliteW"
	"OpenLinkHub/src/logger"
	"github.com/sstallion/go-hid"
	"strconv"
	"sync"
	"time"
)

type Devices struct {
	Type      byte   `json:"type"`
	Endpoint  byte   `json:"endpoint"`
	Serial    string `json:"serial"`
	VendorId  uint16 `json:"deviceId"`
	ProductId uint16 `json:"productId"`
}

type Device struct {
	dev           *hid.Device
	listener      *hid.Device
	Manufacturer  string `json:"manufacturer"`
	Product       string `json:"product"`
	Serial        string `json:"serial"`
	Firmware      string `json:"firmware"`
	ProductId     uint16
	VendorId      uint16
	Devices       *Devices `json:"devices"`
	SharedDevices func(device *common.Device)
	PairedDevices map[uint16]any
	SingleDevice  bool
	Template      string
	Debug         bool
	Exit          bool
	mutex         sync.Mutex
	instance      *common.Device
	MicPosition   byte
	MuteStatus    byte
}

var (
	bufferSize      = 32
	cmdSoftwareMode = []byte{0x01, 0x00}
	cmdHardwareMode = []byte{0x00, 0x00}
	cmdDeviceMode   = byte(0xc8)
	transferTimeout = 1000
)

func Init(vendorId, productId uint16, _, path string, callback func(device *common.Device)) *common.Device {
	// Open device, return if failure
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
		Template:      "slipstream.html",
		SharedDevices: callback,
	}

	d.getDebugMode()         // Debug
	d.getManufacturer()      // Manufacturer
	d.getProduct()           // Product
	d.getSerial()            // Serial
	d.setSoftwareMode()      // Switch to software mode
	d.getDevice()            // Get paired device
	d.addDevices()           // Add devices
	d.backendListener()      // Control listener
	d.createDevice()         // Device register
	d.initAvailableDevices() // Init devices
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeVoidEliteW,
		Product:     "HEADSET DONGLE",
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-dongle.svg",
		Instance:    d,
		Hidden:      true,
	}
}

// addDevices adda a mew device
func (d *Device) addDevices() {
	switch d.Devices.ProductId {
	case 2641:
		{
			dev := voideliteW.Init(
				d.Devices.VendorId,
				d.ProductId,
				d.Devices.ProductId,
				d.dev,
				d.Devices.Endpoint,
				d.Devices.Serial,
			)
			object := &common.Device{
				ProductType: common.ProductTypeVoidEliteW,
				Product:     "VOID ELITE WIRELESS",
				Serial:      dev.Serial,
				Firmware:    dev.Firmware,
				Image:       "icon-headphone.svg",
				Instance:    dev,
			}
			d.SharedDevices(object)
			d.AddPairedDevice(d.Devices.ProductId, dev)
		}
	default:
		logger.Log(logger.Fields{"productId": d.Devices.ProductId}).Warn("Unsupported device detected")
	}
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")

	for _, value := range d.PairedDevices {
		if dev, found := value.(*voideliteW.Device); found {
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

// StopDirty will stop devices in a dirty way
func (d *Device) StopDirty() uint8 {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device (dirty)...")

	for _, value := range d.PairedDevices {
		if dev, found := value.(*voideliteW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 2
}

// getManufacturer will return device manufacturer
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
}

// GetDeviceTemplate will return device template name
func (d *Device) GetDeviceTemplate() string {
	return d.Template
}

// AddPairedDevice will add a paired device
func (d *Device) AddPairedDevice(productId uint16, device any) {
	d.PairedDevices[productId] = device
}

// GetDevice will return HID device
func (d *Device) GetDevice() *hid.Device {
	return d.dev
}

// getDevice will get paired devices
func (d *Device) getDevice() {
	d.Devices = &Devices{
		Endpoint:  0x09,
		Serial:    strconv.Itoa(int(d.ProductId)),
		VendorId:  d.VendorId,
		ProductId: d.ProductId,
	}
}

// getSerial will return device serial number
func (d *Device) getSerial() {
	serial, err := d.dev.GetSerialNbr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device serial number")
	}
	d.Serial = serial
}

// getManufacturer will return device manufacturer
func (d *Device) getManufacturer() {
	manufacturer, err := d.dev.GetMfrStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get manufacturer")
	}
	d.Manufacturer = manufacturer
}

// getProduct will return device name
func (d *Device) getProduct() {
	product, err := d.dev.GetProductStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get product")
	}
	d.Product = product
}

// setHardwareMode will switch a device to hardware mode
func (d *Device) setHardwareMode() {
	_, err := d.transfer(cmdDeviceMode, cmdHardwareMode, false)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	_, err := d.transfer(cmdDeviceMode, cmdSoftwareMode, false)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
}

// initAvailableDevices will run on initial start
func (d *Device) initAvailableDevices() {
	output, err := d.transfer(byte(0xc9), []byte{0x64}, true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device firmware feature report")
		return
	}

	if output[2] != 0x00 {
		d.setDeviceOnlineByProductId(d.Devices.ProductId)
	}
}

// setDeviceOnlineByProductId will online device by given productId
func (d *Device) setDeviceOnlineByProductId(productId uint16) {
	if dev, ok := d.PairedDevices[productId]; ok {
		if device, found := dev.(*voideliteW.Device); found {
			if !device.Connected {
				time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
				device.Connect()
			}
		}
	}
}

// setDevicesOffline will set all device offline
func (d *Device) setDevicesOffline() {
	for _, pairedDevice := range d.PairedDevices {
		if device, found := pairedDevice.(*voideliteW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
	}
}

// setDeviceOffline will set device offline
func (d *Device) setDeviceOnline() {
	for _, pairedDevice := range d.PairedDevices {
		if device, found := pairedDevice.(*voideliteW.Device); found {
			if !device.Connected {
				time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
				device.Connect()
			}
		}
	}
}

// setDeviceOffline will set device offline
func (d *Device) setDeviceStatus(status byte) {
	switch status {
	case 0:
		d.setDevicesOffline()
		break
	case 1:
		d.setDeviceOnline()
		break
	}
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
			if info.InterfaceNbr == 3 {
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

				if data[2] == 0x00 && data[3] == 0x33 {
					d.setDeviceStatus(0)
				}

				if data[2] == 0x00 && data[3] == 0xb1 {
					d.setDeviceStatus(1)
				}

				if data[0] == 0x66 {
					for _, value := range d.PairedDevices {
						if dev, found := value.(*voideliteW.Device); found {
							dev.SetDeviceFirmware(data)
						}
					}
				}

				if data[4] == 0x01 || data[4] == 0x05 || data[4] == 0x02 {
					for _, value := range d.PairedDevices {
						if dev, found := value.(*voideliteW.Device); found {
							val := data[2]
							if val > 100 {
								val = data[2] - 0x80
							}
							dev.ModifyBatteryLevel(val, data[4])
						}
					}
				}

				if data[0] == 0x64 && data[2] > 100 {
					d.MicPosition = 1
				} else if data[0] == 0x64 && data[2] <= 100 {
					d.MicPosition = 0
				}

				if data[0] == 0x01 && data[1] == 0x00 {
					for _, value := range d.PairedDevices {
						if dev, found := value.(*voideliteW.Device); found {
							dev.NotifyMuteChanged(d.MicPosition)
						}
					}
				}

				if data[0] == 0x64 && data[1] == 0x02 {
					for _, value := range d.PairedDevices {
						if dev, found := value.(*voideliteW.Device); found {
							if d.MicPosition == 0 {
								if d.MuteStatus == 0 {
									d.MuteStatus = 1
								} else {
									d.MuteStatus = 0
								}
								dev.NotifyMuteChanged(d.MuteStatus)
							}
						}
					}
				}

				time.Sleep(5 * time.Millisecond)
			}
		}
	}()
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(command byte, data []byte, read bool) ([]byte, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	bufferW := make([]byte, bufferSize)

	bufferW[0] = command
	if len(data) > 0 {
		copy(bufferW[1:], data)
	}

	bufferR := make([]byte, bufferSize)

	if _, err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return bufferR, err
	}

	if read {
		if _, err := d.dev.ReadWithTimeout(bufferR, time.Duration(transferTimeout)*time.Millisecond); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
			return bufferR, err
		}
	}

	return bufferR, nil
}
