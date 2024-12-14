package dongle

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices/katarproW"
	"OpenLinkHub/src/logger"
	"encoding/binary"
	"fmt"
	"github.com/sstallion/go-hid"
	"sync"
	"time"
)

// Package: Corsair Dongle
// This is the primary package for Corsair Dongle.
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
	Debug         bool
}

var (
	mutex            sync.Mutex
	bufferSize       = 64
	bufferSizeWrite  = bufferSize + 1
	headerSize       = 2
	deviceKeepAlive  = 5000
	timerKeepAlive   = &time.Ticker{}
	timerSleep       = &time.Ticker{}
	keepAliveChan    = make(chan bool)
	sleepChan        = make(chan bool)
	cmdSoftwareMode  = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode  = []byte{0x01, 0x03, 0x00, 0x01}
	cmdHeartbeat     = []byte{0x12}
	cmdInactivity    = []byte{0x02, 0x40}
	cmdOpenEndpoint  = []byte{0x0d, 0x00}
	cmdCloseEndpoint = []byte{0x05, 0x01}
	cmdGetFirmware   = []byte{0x02, 0x13}
	cmdRead          = []byte{0x08, 0x00}
	cmdWrite         = []byte{0x09, 0x00}
	cmdCommand       = byte(0x08)
	transferTimeout  = 50
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
		case 7195:
			if dev, found := value.(*katarproW.Device); found {
				if dev.Connected {
					dev.StopInternal()
				}
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

	if d.listener != nil {
		err := d.listener.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
		}
	}
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

	data, err := d.transfer(cmdCommand, []byte{0x02, 0x011}, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device")
	}
	productId := uint16(data[3])<<8 | uint16(data[4])

	device := &Devices{
		Type:      3,
		Endpoint:  byte(0x09),
		Serial:    d.Serial,
		VendorId:  d.VendorId,
		ProductId: productId,
	}
	devices[0] = device
	d.SingleDevice = true
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

	for i := 0; i < int(buffer[5]); i++ {
		next, err := d.transfer(cmdCommand, cmdRead, endpoint)
		if err != nil {
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

// processDevice will process device status
func (d *Device) processDevice(productId uint16, packet []byte) {
	if dev, ok := d.PairedDevices[productId]; ok {
		switch productId {
		case 7195:
			{
				if value, found := dev.(*katarproW.Device); found {
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
									case 7195:
										{
											if device, found := dev.(*katarproW.Device); found {
												sleepMode := device.GetSleepMode() * 60
												if inactive >= sleepMode {
													device.SetSleepMode()
												}
											}
										}
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
						case 7195:
							{
								if dev, found := value.(*katarproW.Device); found {
									if data[1] == 0x02 {
										if data[2] == 0x20 {
											dev.ModifyDpi()
										} else if data[2] == 0x08 {
											// Upper side button
										} else if data[2] == 0x10 {
											// Bottom side button
										}
									}
								}
							}
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
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return bufferR, err
	}

	// Get data from a device
	if _, err := d.dev.ReadWithTimeout(bufferR, time.Duration(transferTimeout)*time.Millisecond); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		return bufferR, err
	}
	return bufferR, nil
}
