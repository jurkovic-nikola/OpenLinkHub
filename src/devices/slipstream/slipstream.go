package slipstream

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices/darkcorergbproW"
	"OpenLinkHub/src/devices/darkcorergbproseW"
	"OpenLinkHub/src/devices/darkstarW"
	"OpenLinkHub/src/devices/harpoonW"
	"OpenLinkHub/src/devices/ironclawW"
	"OpenLinkHub/src/devices/k100airW"
	"OpenLinkHub/src/devices/k70coretklW"
	"OpenLinkHub/src/devices/m55W"
	"OpenLinkHub/src/devices/m75W"
	"OpenLinkHub/src/devices/nightsabreW"
	"OpenLinkHub/src/devices/scimitarW"
	"OpenLinkHub/src/logger"
	"encoding/binary"
	"fmt"
	"github.com/sstallion/go-hid"
	"reflect"
	"sync"
	"time"
)

// Package: Corsair Slipstream
// This is the primary package for Corsair Slipstream.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

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
	PairedDevices  map[uint16]any
	SingleDevice   bool
	Template       string
	Debug          bool
	Exit           bool
	timerKeepAlive *time.Ticker
	timerSleep     *time.Ticker
	keepAliveChan  chan struct{}
	sleepChan      chan struct{}
	mutex          sync.Mutex
}

var (
	bufferSize       = 64
	bufferSizeWrite  = bufferSize + 1
	headerSize       = 2
	deviceKeepAlive  = 5000
	cmdSoftwareMode  = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode  = []byte{0x01, 0x03, 0x00, 0x01}
	cmdGetDevices    = []byte{0x24}
	cmdHeartbeat     = []byte{0x12}
	cmdInactivity    = []byte{0x02, 0x40}
	cmdOpenEndpoint  = []byte{0x0d, 0x00}
	cmdCloseEndpoint = []byte{0x05, 0x01}
	cmdGetFirmware   = []byte{0x02, 0x13}
	cmdRead          = []byte{0x08, 0x00}
	cmdWrite         = []byte{0x09, 0x00}
	cmdCommand       = byte(0x08)
	transferTimeout  = 100
)

func Init(vendorId, productId uint16, key string) *Device {
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
		PairedDevices:  make(map[uint16]any, 0),
		Template:       "slipstream.html",
		sleepChan:      make(chan struct{}),
		keepAliveChan:  make(chan struct{}),
		timerSleep:     &time.Ticker{},
		timerKeepAlive: &time.Ticker{},
	}

	d.getDebugMode()      // Debug
	d.getManufacturer()   // Manufacturer
	d.getProduct()        // Product
	d.getSerial()         // Serial
	d.getDeviceFirmware() // Firmware
	d.setSoftwareMode()   // Switch to software mode
	d.getDevices()        // Get devices
	d.monitorDevice()     // Monitor device
	d.sleepMonitor()      // Sleep
	d.controlListener()   // Control listener
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")
	return d
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")

	d.timerSleep.Stop()
	d.timerKeepAlive.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.keepAliveChan != nil {
				close(d.keepAliveChan)
			}
			if d.sleepChan != nil {
				close(d.sleepChan)
			}
		})
	}()

	for _, value := range d.PairedDevices {
		if dev, found := value.(*m55W.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*nightsabreW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*k100airW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*ironclawW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*scimitarW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*darkcorergbproseW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*darkcorergbproW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*m75W.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*harpoonW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*darkstarW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*k70coretklW.Device); found {
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

// getDevices will get a list of paired devices
func (d *Device) getDevices() {
	var devices = make(map[int]*Devices, 0)
	buff := d.read(cmdGetDevices)
	if d.Debug {
		logger.Log(logger.Fields{"serial": d.Serial, "length": len(buff), "data": fmt.Sprintf("% 2x", buff)}).Info("DEBUG")
	}

	channels := buff[5]
	data := buff[6:]
	position := 0

	var base byte = 0x08
	if channels > 0 {
		for i := 0; i < int(channels); i++ {
			vendorId := uint16(data[position+1])<<8 | uint16(data[position])
			productId := uint16(data[position+5])<<8 | uint16(data[position+4])
			deviceType := data[position+6]
			deviceIdLen := data[position+7]
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
		if device, found := dev.(*m55W.Device); found {
			if !device.Connected {
				device.Connect()
			}
		}
		if device, found := dev.(*scimitarW.Device); found {
			if !device.Connected {
				device.Connect()
			}
		}
		if device, found := dev.(*nightsabreW.Device); found {
			if !device.Connected {
				device.Connect()
			}
		}
		if device, found := dev.(*k100airW.Device); found {
			if !device.Connected {
				device.Connect()
			}
		}
		if device, found := dev.(*ironclawW.Device); found {
			if !device.Connected {
				device.Connect()
			}
		}
		if device, found := dev.(*darkcorergbproseW.Device); found {
			if !device.Connected {
				device.Connect()
			}
		}
		if device, found := dev.(*darkcorergbproW.Device); found {
			if !device.Connected {
				device.Connect()
			}
		}
		if device, found := dev.(*m75W.Device); found {
			if !device.Connected {
				device.Connect()
			}
		}
		if device, found := dev.(*harpoonW.Device); found {
			if !device.Connected {
				device.Connect()
			}
		}
		if device, found := dev.(*darkstarW.Device); found {
			if !device.Connected {
				device.Connect()
			}
		}
		if device, found := dev.(*k70coretklW.Device); found {
			if !device.Connected {
				device.Connect()
			}
		}
	}
}

// setDevicesOffline will set all device offline
func (d *Device) setDevicesOffline() {
	for _, pairedDevice := range d.PairedDevices {
		if device, found := pairedDevice.(*m55W.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*scimitarW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*nightsabreW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*k100airW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*ironclawW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*darkcorergbproseW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*darkcorergbproW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*m75W.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*harpoonW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*darkstarW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*k70coretklW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
	}
}

// setDeviceTypeOffline will set specific device type offline
func (d *Device) setDeviceTypeOffline(deviceType int) {
	for _, pairedDevice := range d.PairedDevices {
		switch deviceType {
		case 0:
			{
				// Keyboards
				if device, found := pairedDevice.(*k100airW.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
				if device, found := pairedDevice.(*k70coretklW.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
			}
			break
		case 1:
			{
				// Mouses
				if device, found := pairedDevice.(*m55W.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
				if device, found := pairedDevice.(*scimitarW.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
				if device, found := pairedDevice.(*nightsabreW.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
				if device, found := pairedDevice.(*ironclawW.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
				if device, found := pairedDevice.(*darkcorergbproseW.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
				if device, found := pairedDevice.(*darkcorergbproW.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
				if device, found := pairedDevice.(*m75W.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
				if device, found := pairedDevice.(*harpoonW.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
				if device, found := pairedDevice.(*darkstarW.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
			}
			break
		}
	}
}

// setDeviceOffline will set device offline
func (d *Device) setDeviceOnline(deviceType int) {
	for _, pairedDevice := range d.PairedDevices {
		switch deviceType {
		case 0:
			{
				// Keyboards
				if device, found := pairedDevice.(*k100airW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*k70coretklW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
			}
			break
		case 1:
			{
				// Mouses
				if device, found := pairedDevice.(*m55W.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*scimitarW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*nightsabreW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*ironclawW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*darkcorergbproseW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*darkcorergbproW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*m75W.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*harpoonW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*darkstarW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
			}
			break
		case 2:
			{
				// All
				if device, found := pairedDevice.(*k100airW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*m55W.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*scimitarW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*nightsabreW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*ironclawW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*darkcorergbproseW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*darkcorergbproW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*m75W.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*harpoonW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*darkstarW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
				if device, found := pairedDevice.(*k70coretklW.Device); found {
					if !device.Connected {
						device.Connect()
					}
				}
			}
			break
		}
	}
}

// setDeviceOffline will set device offline
func (d *Device) setDeviceStatus(status byte) {
	switch status {
	case 0x00: // ALl offline
		d.setDevicesOffline()
		break
	case 0x02: // Single device, online
		d.setDeviceOnline(2)
		break
	case 0x04: // Multiple paired devices, mouse
		d.setDeviceTypeOffline(0)
		d.setDeviceOnline(1)
		break
	case 0x08: // Multiple paired devices, keyboard
		d.setDeviceTypeOffline(1)
		d.setDeviceOnline(0)
		break
	case 0x0c:
		d.setDeviceOnline(2)
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

// getSleepMode will return device sleep mode
func (d *Device) getSleepMode(dev any) int {
	sleepMode := 0
	methodName := "GetSleepMode"
	method := reflect.ValueOf(dev).MethodByName(methodName)
	if !method.IsValid() {
		logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
		return 0
	} else {
		results := method.Call(nil)
		if len(results) > 0 {
			val := results[0]
			sleepMode = int(val.Int())
		}
	}
	return sleepMode
}

// sleepMonitor will monitor for device inactivity
func (d *Device) sleepMonitor() {
	d.timerSleep = time.NewTicker(time.Duration(deviceKeepAlive) * time.Millisecond)
	go func() {
		for {
			select {
			case <-d.timerSleep.C:
				{
					if d.Exit {
						return
					}
					for _, value := range d.Devices {
						msg, err := d.transfer(value.Endpoint, cmdInactivity, nil)
						if err != nil {
							if d.Debug {
								logger.Log(logger.Fields{"error": err}).Error("Unable to read device endpoint")
							}
							continue
						}
						if (msg[0] == 0x02 || msg[0] == 0x01) && msg[1] == 0x02 { // Mouse // Connected
							inactive := int(binary.LittleEndian.Uint16(msg[3:5]))
							if inactive > 0 {
								if dev, ok := d.PairedDevices[value.ProductId]; ok {
									sleepMode := d.getSleepMode(dev) * 60
									if sleepMode > 0 {
										methodName := "SetSleepMode"
										method := reflect.ValueOf(dev).MethodByName(methodName)
										if !method.IsValid() {
											logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
											continue
										} else {
											if inactive >= sleepMode {
												method.Call(nil)
											}
										}
									}
								}
							}
						}
					}
				}
			case <-d.sleepChan:
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

// controlListener will listen for events from the control buttons
func (d *Device) controlListener() {
	go func() {
		enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
			if info.InterfaceNbr == 2 {
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

				if data[1] == 0x01 && data[2] == 0x36 {
					value := data[4]
					d.setDeviceStatus(value)
				} else {
					switch data[0] {
					case 1, 2, 3:
						{
							for _, value := range d.PairedDevices {
								if dev, found := value.(*k100airW.Device); found {
									if data[16] == 2 {
										dev.ModifyBrightness()
									}
								}
								if dev, found := value.(*m55W.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(data[2], d.Serial)
									}
								}
								if dev, found := value.(*m75W.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(data[2], d.Serial)
									}
								}
								if dev, found := value.(*scimitarW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]), d.Serial)
									}
								}
								if dev, found := value.(*nightsabreW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]), d.Serial)
									}
								}
								if dev, found := value.(*darkcorergbproseW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]), d.Serial)
									}
								}
								if dev, found := value.(*darkcorergbproW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]), d.Serial)
									}
								}
								if dev, found := value.(*harpoonW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(data[2], d.Serial)
									}
								}
								if dev, found := value.(*ironclawW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]), d.Serial)
									}
								}
								if dev, found := value.(*darkstarW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]), d.Serial)
									}
								}

								if dev, found := value.(*k70coretklW.Device); found {
									if data[1] == 0x02 && data[2] == 0x04 {
										dev.ControlDial(data)
									} else if data[1] == 0x05 && (data[4] == 0x01 || data[4] == 0xff) {
										dev.ControlDial(data)
									}
								}
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
	bufferW[1] = command
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
		n, err := d.dev.Read(reports)
		if err != nil {
			if n < 0 {
				//
			}
			if err == hid.ErrTimeout || n == 0 {
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
	bufferW[1] = command
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
