package voidV2dongle

// Package: Headset Dongle
// This is the primary package for Corsair Headset Dongle.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"OpenLinkHub/src/common"
	"OpenLinkHub/src/devices/voidV2W"
	"github.com/sstallion/go-hid"
)

type Devices struct {
	Type      byte   `json:"type"`
	Endpoint  byte   `json:"endpoint"`
	Serial    string `json:"serial"`
	VendorId  uint16 `json:"deviceId"`
	ProductId uint16 `json:"productId"`
}

type Device struct {
	dev            *hid.Device
	listener       *hid.Device
	Manufacturer   string `json:"manufacturer"`
	Product        string `json:"product"`
	Serial         string `json:"serial"`
	Firmware       string `json:"firmware"`
	ProductId      uint16
	VendorId       uint16
	Devices        *Devices `json:"devices"`
	SharedDevices  func(device *common.Device)
	PairedDevices  map[uint16]any
	SingleDevice   bool
	Template       string
	Debug          bool
	Exit           bool
	timerKeepAlive *time.Ticker
	keepAliveChan  chan struct{}
	mutex          sync.Mutex
	instance       *common.Device
}

var (
	bufferSize      = 64
	bufferSizeWrite = bufferSize + 1
	headerSize      = 3
	deviceKeepAlive = 10000
	cmdSoftwareMode = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode = []byte{0x01, 0x03, 0x00, 0x01}
	cmdHeartbeat    = []byte{0x12}
	cmdGetFirmware  = []byte{0x02, 0x13}
	cmdCommand      = byte(0x08)
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
		dev:            dev,
		VendorId:       vendorId,
		ProductId:      productId,
		PairedDevices:  make(map[uint16]any),
		Template:       "slipstream.html",
		keepAliveChan:  make(chan struct{}),
		timerKeepAlive: &time.Ticker{},
		SharedDevices:  callback,
	}

	d.getDebugMode()         // Debug
	d.getManufacturer()      // Manufacturer
	d.getProduct()           // Product
	d.getSerial()            // Serial
	d.getDeviceFirmware()    // Firmware
	d.setSoftwareMode()      // Switch to software mode
	d.getDevice()            // Get paired device
	d.addDevices()           // Add devices
	d.monitorDevice()        // Monitor device
	d.backendListener()      // Control listener
	d.createDevice()         // Device register
	d.initAvailableDevices() // Init devices
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeVoidV2W,
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
	case 10761:
		{
			dev := voidV2W.Init(
				d.Devices.VendorId,
				d.ProductId,
				d.Devices.ProductId,
				d.dev,
				d.Devices.Endpoint,
				d.Devices.Serial,
			)
			object := &common.Device{
				ProductType: common.ProductTypeVoidV2W,
				Product:     "VOID WIRELESS V2",
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

	d.timerKeepAlive.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.keepAliveChan != nil {
				close(d.keepAliveChan)
			}
		})
	}()

	for _, value := range d.PairedDevices {
		if dev, found := value.(*voidV2W.Device); found {
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

	d.timerKeepAlive.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.keepAliveChan != nil {
				close(d.keepAliveChan)
			}
		})
	}()

	for _, value := range d.PairedDevices {
		if dev, found := value.(*voidV2W.Device); found {
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
		Serial:    d.Serial + "W",
		VendorId:  d.VendorId,
		ProductId: 10761,
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

// getDeviceFirmware will return a firmware version out as string
func (d *Device) getDeviceFirmware() {
	fw, err := d.transfer(cmdCommand, cmdGetFirmware, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
	}

	v1, v2, v3 := int(fw[4]), int(fw[5]), int(fw[6])
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

// setHardwareMode will switch a device to hardware mode
func (d *Device) setHardwareMode() {
	_, err := d.transfer(cmdCommand, cmdHardwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	_, err := d.transfer(cmdCommand, cmdSoftwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
}

// initAvailableDevices will run on initial start
func (d *Device) initAvailableDevices() {
	_, err := d.transferToDevice(d.Devices.Endpoint, cmdHeartbeat, nil, "initAvailableDevices")
	if err != nil {
		logger.Log(logger.Fields{"error": err, "endpoint": d.Devices.Endpoint, "productId": d.Devices.ProductId}).Warn("Unable to read endpoint. Device is probably offline")
		return
	}
	d.setDeviceOnlineByProductId(d.Devices.ProductId)
}

// setDeviceOnlineByProductId will online device by given productId
func (d *Device) setDeviceOnlineByProductId(productId uint16) {
	if dev, ok := d.PairedDevices[productId]; ok {
		if device, found := dev.(*voidV2W.Device); found {
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
		if device, found := pairedDevice.(*voidV2W.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
	}
}

// setDeviceOffline will set device offline
func (d *Device) setDeviceOnline() {
	for _, pairedDevice := range d.PairedDevices {
		if device, found := pairedDevice.(*voidV2W.Device); found {
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
	case 0x00:
		d.setDevicesOffline()
		break
	case 0x02:
		d.setDeviceOnline()
		break
	}
}

// monitorDevice will refresh device data
func (d *Device) monitorDevice() {
	d.timerKeepAlive = time.NewTicker(time.Duration(deviceKeepAlive) * time.Millisecond)
	go func() {
		for {
			select {
			case <-d.timerKeepAlive.C:
				{
					if d.Exit {
						return
					}
					_, err := d.transfer(cmdCommand, cmdHeartbeat, nil)
					if err != nil {
						logger.Log(logger.Fields{"error": err}).Error("Unable to read slipstream endpoint")
					}

					_, e := d.transfer(d.Devices.Endpoint, cmdHeartbeat, nil)
					if e != nil {
						if d.Debug {
							logger.Log(logger.Fields{"error": err}).Error("Unable to read paired device endpoint")
						}
					}
				}
			case <-d.keepAliveChan:
				return
			}
		}
	}()
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
			if info.InterfaceNbr == 4 {
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

		// Listen loop
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

				// Battery
				if data[0] == 0x03 && data[1] == 0x01 && data[3] == 0x0f {
					val := binary.LittleEndian.Uint16(data[5:7]) / 10
					if val > 0 {
						for _, value := range d.PairedDevices {
							if dev, found := value.(*voidV2W.Device); found {
								dev.ModifyBatteryLevel(val)
							}
						}
					}
				}

				if data[1] == 0x00 && data[3] == 0x36 {
					value := data[5]
					d.setDeviceStatus(value)
				} else {
					//fmt.Println(fmt.Sprintf("% 2x", data))
					if data[0] == 0x03 && data[2] == 0x01 && data[3] == 0xa6 {
						for _, value := range d.PairedDevices {
							if dev, found := value.(*voidV2W.Device); found {
								dev.NotifyMuteChanged(data[5])
							}
						}
					}
				}
			}
		}
	}()
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(command byte, endpoint, buffer []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = 0x02
	bufferW[2] = command
	endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
	}

	reports := make([]byte, 1)
	err := d.dev.SetNonblock(true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to SetNonblock")
	}

	for {
		n, e := d.dev.Read(reports)
		if e != nil {
			if n < 0 {
				//
			}
			if e == hid.ErrTimeout || n == 0 {
				break
			}
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	err = d.dev.SetNonblock(false)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to SetNonblock")
	}

	// Create read buffer
	bufferR := make([]byte, bufferSize)

	// Send command to a device
	if _, err := d.dev.Write(bufferW); err != nil {
		if d.Debug {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		}
		return bufferR, err
	}

	// Get data from a device
	if _, err := d.dev.ReadWithTimeout(bufferR, time.Duration(transferTimeout)*time.Millisecond); err != nil {
		if d.Debug {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		}
		return bufferR, err
	}
	return bufferR, nil
}

// transfer will send data to a device and retrieve device output
func (d *Device) transferToDevice(command byte, endpoint, buffer []byte, caller string) ([]byte, error) {
	// Packet control, mandatory for this device
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = 0x02
	bufferW[2] = command
	endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
	}

	// Create read buffer
	bufferR := make([]byte, bufferSize)

	// Send command to a device
	if _, err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return bufferR, err
	}

	// Get data from a device
	if _, err := d.dev.ReadWithTimeout(bufferR, time.Duration(transferTimeout)*time.Millisecond); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "caller": caller}).Error("Unable to read data from device")
		return bufferR, err
	}
	return bufferR, nil
}
