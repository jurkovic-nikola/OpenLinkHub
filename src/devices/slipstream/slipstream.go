package slipstream

import (
	"OpenLinkHub/src/devices/ironclawW"
	"OpenLinkHub/src/devices/k100airW"
	"OpenLinkHub/src/devices/nightsabreW"
	"OpenLinkHub/src/logger"
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/sstallion/go-hid"
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
	dev           *hid.Device
	listener      *hid.Device
	Manufacturer  string `json:"manufacturer"`
	Product       string `json:"product"`
	Serial        string `json:"serial"`
	Firmware      string `json:"firmware"`
	ProductId     uint16
	VendorId      uint16
	Devices       map[int]*Devices `json:"devices"`
	PairedDevices map[uint16]any
	SingleDevice  bool
	Template      string
}

var (
	mutex              sync.Mutex
	bufferSize         = 64
	bufferSizeWrite    = bufferSize + 1
	headerSize         = 2
	deviceKeepAlive    = 5000
	timerKeepAlive     = &time.Ticker{}
	timerSleep         = &time.Ticker{}
	keepAliveChan      = make(chan bool)
	sleepChan          = make(chan bool)
	cmdSoftwareMode    = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode    = []byte{0x01, 0x03, 0x00, 0x01}
	cmdGetDevices      = []byte{0x24}
	cmdHeartbeat       = []byte{0x12}
	cmdInactivity      = []byte{0x02, 0x40}
	cmdOpenEndpoint    = []byte{0x0d, 0x00}
	cmdCloseEndpoint   = []byte{0x05, 0x01}
	cmdGetFirmware     = []byte{0x02, 0x13}
	cmdRead            = []byte{0x08, 0x00}
	cmdWrite           = []byte{0x09, 0x00}
	cmdCommand         = byte(0x08)
	dataTypeGetDevices = []byte{0x21, 0x00}
	transferTimeout    = 50
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
		dev:           dev,
		VendorId:      vendorId,
		ProductId:     productId,
		PairedDevices: make(map[uint16]any, 0),
		Template:      "slipstream.html",
	}

	d.getManufacturer()   // Manufacturer
	d.getProduct()        // Product
	d.getSerial()         // Serial
	d.getDeviceFirmware() // Firmware
	d.setSoftwareMode()   // Switch to software mode
	d.getDevices()        // Get devices
	d.monitorDevice()     // Monitor device
	d.sleepMonitor()      // Sleep
	d.controlListener()   // Control listener
	return d
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	logger.Log(logger.Fields{"serial": d.Serial}).Info("Stopping device...")
	keepAliveChan <- true
	timerKeepAlive.Stop()

	sleepChan <- true
	timerSleep.Stop()

	for key, value := range d.PairedDevices {
		switch key {
		case 7096:
			if dev, found := value.(*nightsabreW.Device); found {
				dev.StopInternal()
			}
			break
		case 7083:
			if dev, found := value.(*k100airW.Device); found {
				dev.StopInternal()
			}
			break
		case 6988:
			if dev, found := value.(*ironclawW.Device); found {
				dev.StopInternal()
			}
			break
		}
	}

	d.setHardwareMode()
	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
		}
	}

	if d.listener != nil {
		err := d.listener.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
		}
	}
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
	channels := buff[5]
	data := buff[6:]
	position := 0
	var base byte = 8
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
	// Endpoint data
	var buffer []byte

	// Close specified endpoint
	_, err := d.transfer(cmdCommand, cmdCloseEndpoint, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close endpoint")
	}

	// Open endpoint
	_, err = d.transfer(cmdCommand, cmdOpenEndpoint, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to open endpoint")
	}

	// Open endpoint
	_, err = d.transfer(cmdCommand, cmdWrite, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to open endpoint")
	}

	// Read data from endpoint
	buffer, err = d.transfer(cmdCommand, cmdRead, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read endpoint")
	}

	if responseMatch(buffer, dataTypeGetDevices) {
		next, err := d.transfer(cmdCommand, cmdRead, endpoint)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to read endpoint")
		}
		buffer = append(buffer, next[3:]...)
	}

	// Close specified endpoint
	_, err = d.transfer(cmdCommand, cmdCloseEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close endpoint")
	}
	return buffer
}

// processDevice will process device status
func (d *Device) processDevice(productId uint16, packet []byte) {
	if dev, ok := d.PairedDevices[productId]; ok {
		switch productId {
		case 7096:
			{
				if value, found := dev.(*nightsabreW.Device); found {
					switch packet[1] {
					case 0x01:
						{
							if d.SingleDevice {
								if packet[0] == 0x01 {
									value.SetConnected(false)
								}
							} else {
								if packet[0] == 0x02 {
									value.SetConnected(false)
								}
							}
						}
					case 0x00:
						{
							value.SetConnected(false)
						}
						break
					case 0x12:
						{
							if d.SingleDevice {
								if packet[0] == 0x01 {
									value.Connect()
								}
							} else {
								if packet[0] == 0x02 {
									value.Connect()
								}
							}
						}
						break
					}
				}
			}
		case 7083:
			{
				if value, found := dev.(*k100airW.Device); found {
					switch packet[1] {
					case 0x00:
						{
							value.SetConnected(false)
						}
						break
					case 0x12:
						{
							if d.SingleDevice {
								if packet[0] == 0x01 {
									value.Connect()
								}
							} else {
								if packet[0] == 0x03 {
									value.Connect()
								}
							}
						}
						break
					}
				}
			}
		case 6988:
			{
				if value, found := dev.(*ironclawW.Device); found {
					switch packet[1] {
					case 0x01:
						{
							if d.SingleDevice {
								if packet[0] == 0x01 {
									value.SetConnected(false)
								}
							} else {
								if packet[0] == 0x02 {
									value.SetConnected(false)
								}
							}
						}
					case 0x00:
						{
							value.SetConnected(false)
						}
						break
					case 0x12:
						{
							if d.SingleDevice {
								if packet[0] == 0x01 {
									value.Connect()
								}
							} else {
								if packet[0] == 0x02 {
									value.Connect()
								}
							}
						}
						break
					}
				}
			}
			break
		}
	}
}

// monitorDevice will refresh device data
func (d *Device) monitorDevice() {
	timerKeepAlive = time.NewTicker(time.Duration(deviceKeepAlive) * time.Millisecond)
	go func() {
		for {
			select {
			case <-timerKeepAlive.C:
				{
					_, err := d.transfer(cmdCommand, cmdHeartbeat, nil)
					if err != nil {
						logger.Log(logger.Fields{"error": err}).Error("Unable to read endpoint")
					}
					for _, value := range d.Devices {
						msg, err := d.transfer(value.Endpoint, cmdHeartbeat, nil)
						if err != nil {
							logger.Log(logger.Fields{"error": err}).Error("Unable to read endpoint")
						}
						d.processDevice(value.ProductId, msg)
					}
				}
			case <-keepAliveChan:
				return
			}
		}
	}()
}

// sleepMonitor will monitor for device inactivity
func (d *Device) sleepMonitor() {
	timerSleep = time.NewTicker(time.Duration(deviceKeepAlive) * time.Millisecond)
	go func() {
		for {
			select {
			case <-timerSleep.C:
				{
					for _, value := range d.Devices {
						msg, err := d.transfer(value.Endpoint, cmdInactivity, nil)
						if err != nil {
							logger.Log(logger.Fields{"error": err}).Error("Unable to read endpoint")
						}

						if (msg[0] == 0x02 || msg[0] == 0x01) && msg[1] == 0x02 { // Mouse // Connected
							inactive := int(binary.LittleEndian.Uint16(msg[3:5]))
							if inactive > 0 {
								if dev, ok := d.PairedDevices[value.ProductId]; ok {
									switch value.ProductId {
									case 7096:
										{
											if device, found := dev.(*nightsabreW.Device); found {
												sleepMode := device.GetSleepMode() * 60
												if inactive >= sleepMode {
													device.SetSleepMode()
												}
											}
										}
									case 6988:
										{
											if device, found := dev.(*ironclawW.Device); found {
												sleepMode := device.GetSleepMode() * 60
												if inactive >= sleepMode {
													device.SetSleepMode()
												}
											}
										}
										break
									}
								}
							}
						}
					}
				}
			case <-sleepChan:
				return
			}
		}
	}()
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
		data := make([]byte, bufferSize)
		for {
			if d.listener == nil {
				break
			}

			_, err = d.listener.Read(data)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Error reading data")
				break
			}

			switch data[0] {
			case 1, 2: // Mouse
				{
					for key, value := range d.PairedDevices {
						switch key {
						case 7096:
							if dev, found := value.(*nightsabreW.Device); found {
								if data[2] == 0x80 {
									dev.ModifyDpi(true)
								} else if data[2] == 0x00 && data[3] == 0x01 {
									dev.ModifyDpi(false)
								}
							}
							break

						case 6988:
							if dev, found := value.(*ironclawW.Device); found {
								switch data[2] {
								case 32: // DPI Button Up
									dev.ModifyDpi(true)
									break
								case 64: // DPI Button Down
									dev.ModifyDpi(false)
									break
								case 8: // Forward button
									// TO-DO
									break
								case 16: // Back button
									// TO-DO
									break
								}
							}
							break
						}
					}
				}
			case 3: // Keyboard
				{
					for key, value := range d.PairedDevices {
						switch key {
						case 7083:
							if dev, found := value.(*k100airW.Device); found {
								if data[16] == 2 {
									dev.ModifyBrightness()
								}
							}
							break
						}
					}
				}
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(command byte, endpoint, buffer []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	mutex.Lock()
	defer mutex.Unlock()

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
		return nil, err
	}

	// Get data from a device
	if _, err := d.dev.ReadWithTimeout(bufferR, time.Duration(transferTimeout)*time.Millisecond); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		return bufferR, err
	}
	return bufferR, nil
}

// responseMatch will check if two byte arrays match
func responseMatch(response, expected []byte) bool {
	responseBuffer := response[3:5]
	return bytes.Equal(responseBuffer, expected)
}
