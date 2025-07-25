package headsetdongle

// Package: Headset Dongle
// This is the primary package for Corsair Headset Dongle.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices/hs80rgbW"
	"OpenLinkHub/src/devices/virtuosorgbXTW"
	"OpenLinkHub/src/logger"
	"encoding/binary"
	"fmt"
	"github.com/sstallion/go-hid"
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
	dev            *hid.Device
	listener       *hid.Device
	Manufacturer   string `json:"manufacturer"`
	Product        string `json:"product"`
	Serial         string `json:"serial"`
	Firmware       string `json:"firmware"`
	ProductId      uint16
	VendorId       uint16
	Devices        map[int]*Devices `json:"devices"`
	SharedDevices  map[string]*common.Device
	DeviceList     map[string]*common.Device
	PairedDevices  map[uint16]any
	SingleDevice   bool
	Template       string
	Debug          bool
	Exit           bool
	timerKeepAlive *time.Ticker
	keepAliveChan  chan struct{}
	mutex          sync.Mutex
}

var (
	bufferSize       = 64
	bufferSizeWrite  = bufferSize + 1
	headerSize       = 3
	deviceKeepAlive  = 10000
	cmdSoftwareMode  = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode  = []byte{0x01, 0x03, 0x00, 0x01}
	cmdGetDevices    = []byte{0x24}
	cmdHeartbeat     = []byte{0x12}
	cmdOpenEndpoint  = []byte{0x0d, 0x01}
	cmdCloseEndpoint = []byte{0x05, 0x01, 0x01}
	cmdGetFirmware   = []byte{0x02, 0x13}
	cmdRead          = []byte{0x08, 0x01}
	cmdWrite         = []byte{0x09, 0x01}
	cmdCommand       = byte(0x08)
	transferTimeout  = 1000
	connectDelay     = 3000
)

func Init(vendorId, productId uint16, key string, devices map[string]*common.Device) *Device {
	// Open device, return if failure
	dev, err := hid.OpenPath(key)
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
		DeviceList:     make(map[string]*common.Device),
		Template:       "slipstream.html",
		keepAliveChan:  make(chan struct{}),
		timerKeepAlive: &time.Ticker{},
		SharedDevices:  devices,
	}

	d.getDebugMode()      // Debug
	d.getManufacturer()   // Manufacturer
	d.getProduct()        // Product
	d.getSerial()         // Serial
	d.getDeviceFirmware() // Firmware
	d.setSoftwareMode()   // Switch to software mode
	d.getDevices()        // Get devices
	d.monitorDevice()     // Monitor device
	d.backendListener()   // Control listener
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")
	return d
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
		if dev, found := value.(*virtuosorgbXTW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*hs80rgbW.Device); found {
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
		if dev, found := value.(*virtuosorgbXTW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*hs80rgbW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
	}

	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 1
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
func (d *Device) AddPairedDevice(productId uint16, device any, dev *common.Device) {
	d.PairedDevices[productId] = device
	d.DeviceList[dev.Serial] = dev
}

// GetDevice will return HID device
func (d *Device) GetDevice() *hid.Device {
	return d.dev
}

// getDevices will get a list of paired devices
func (d *Device) getDevices() {
	var devices = make(map[int]*Devices)
	buff := d.read(cmdGetDevices)
	if d.Debug {
		logger.Log(logger.Fields{"serial": d.Serial, "length": len(buff), "data": fmt.Sprintf("% 2x", buff)}).Info("DEBUG")
	}

	channels := buff[6]
	data := buff[7:]
	position := 0

	var base byte = 0x08
	if channels > 0 {
		for i := 0; i < int(channels); i++ {
			vendorId := uint16(data[position+1])<<8 | uint16(data[position])
			productId := uint16(data[position+5])<<8 | uint16(data[position+4])
			deviceType := data[position+6]
			deviceIdLen := data[position+7]
			if position+8+int(deviceIdLen) > len(data) {
				logger.Log(logger.Fields{"serial": d.Serial, "length": len(buff), "position": position + 8 + int(deviceIdLen), "data": fmt.Sprintf("% 2x", buff)}).Warn("Requested position exceeds maximum length")
				continue
			}
			deviceId := data[position+8 : position+8+int(deviceIdLen)]
			endpoint := base + deviceType
			if channels == 1 {
				endpoint = base + 1
			}
			device := &Devices{
				Type:      deviceType,
				Endpoint:  endpoint,
				Serial:    string(deviceId),
				VendorId:  vendorId,
				ProductId: productId,
			}

			if d.Debug {
				logger.Log(logger.Fields{"serial": d.Serial, "device": device}).Info("Processing device")
			}

			devices[i] = device
			position += 8 + int(deviceIdLen)
		}
	}

	if len(devices) == 1 {
		d.SingleDevice = true
	}
	d.Devices = devices
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

	v1, v2, v3 := int(fw[3]), int(fw[4]), int(binary.LittleEndian.Uint16(fw[5:7]))
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

func (d *Device) readNext() []byte {
	buffer, err := d.transfer(cmdCommand, cmdRead, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read endpoint")
	}
	return buffer
}

// read will read data from a device and return data as a byte array
func (d *Device) read(endpoint []byte) []byte {
	var buffer []byte

	_, err := d.transfer(cmdCommand, cmdCloseEndpoint, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close endpoint")
	}

	_, err = d.transfer(cmdCommand, cmdOpenEndpoint, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to open endpoint")
	}

	_, err = d.transfer(cmdCommand, cmdWrite, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to open endpoint")
	}

	buffer, err = d.transfer(cmdCommand, cmdRead, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read endpoint")
	}

	for i := 1; i < int(buffer[5]); i++ {
		next, e := d.transfer(cmdCommand, cmdRead, endpoint)
		if e != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to read endpoint")
		}
		buffer = append(buffer, next[3:]...)
	}

	_, err = d.transfer(cmdCommand, cmdCloseEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close endpoint")
	}
	return buffer
}

// InitAvailableDevices will run on initial start
func (d *Device) InitAvailableDevices() {
	for _, value := range d.Devices {
		_, err := d.transferToDevice(value.Endpoint, cmdHeartbeat, nil, "InitAvailableDevices")
		if err != nil {
			logger.Log(logger.Fields{"error": err, "endpoint": value.Endpoint, "productId": value.ProductId}).Warn("Unable to read endpoint. Device is probably offline")
			continue
		}
		d.setDeviceOnlineByProductId(value.ProductId)
	}
}

// setDeviceOnlineByProductId will online device by given productId
func (d *Device) setDeviceOnlineByProductId(productId uint16) {
	if dev, ok := d.PairedDevices[productId]; ok {
		if device, found := dev.(*virtuosorgbXTW.Device); found {
			if !device.Connected {
				time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
				device.Connect()
			}
		}
		if device, found := dev.(*hs80rgbW.Device); found {
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
		if device, found := pairedDevice.(*virtuosorgbXTW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*hs80rgbW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
	}
}

// setDeviceOffline will set device offline
func (d *Device) setDeviceOnline() {
	time.Sleep(time.Duration(connectDelay) * time.Millisecond)
	for _, pairedDevice := range d.PairedDevices {
		if device, found := pairedDevice.(*virtuosorgbXTW.Device); found {
			if !device.Connected {
				time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
				device.Connect()
				d.SharedDevices[device.Serial] = d.DeviceList[device.Serial]
			}
		}
		if device, found := pairedDevice.(*hs80rgbW.Device); found {
			if !device.Connected {
				time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
				device.Connect()
				d.SharedDevices[device.Serial] = d.DeviceList[device.Serial]
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
					for _, value := range d.Devices {
						_, e := d.transfer(value.Endpoint, cmdHeartbeat, nil)
						if e != nil {
							if d.Debug {
								logger.Log(logger.Fields{"error": err}).Error("Unable to read paired device endpoint")
							}
							continue
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
				if data[1] == 0x01 && data[2] == 0x12 {
					var val uint16 = 0
					if data[7] > 0 { // Unclear why it switches 1 position next
						val = binary.LittleEndian.Uint16(data[6:8]) / 10
					} else {
						val = binary.LittleEndian.Uint16(data[5:7]) / 10
					}

					if val > 0 {
						for _, value := range d.PairedDevices {
							if dev, found := value.(*virtuosorgbXTW.Device); found {
								dev.ModifyBatteryLevel(val)
							}
							if dev, found := value.(*hs80rgbW.Device); found {
								dev.ModifyBatteryLevel(val)
							}
						}
					}
				}

				if data[1] == 0x00 && data[3] == 0x36 {
					value := data[5]
					d.setDeviceStatus(value)
				} else {
					if data[2] == 0x01 && (data[3] == 0x8e || data[3] == 0xa6) {
						for _, value := range d.PairedDevices {
							if dev, found := value.(*virtuosorgbXTW.Device); found {
								dev.NotifyMuteChanged(data[5])
							}
							if dev, found := value.(*hs80rgbW.Device); found {
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
