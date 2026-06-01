package slipstreamV2

// Package: Corsair Slipstream V2
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices/m65rgbultraW"
	"OpenLinkHub/src/devices/vanguard96W"
	"OpenLinkHub/src/devices/vanguard99airW"
	"OpenLinkHub/src/logger"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/sstallion/go-hid"
	"reflect"
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
	slipstream     *common.Slipstream
	Manufacturer   string `json:"manufacturer"`
	Product        string `json:"product"`
	Serial         string `json:"serial"`
	Firmware       string `json:"firmware"`
	ProductId      uint16
	VendorId       uint16
	Devices        map[int]*Devices `json:"devices"`
	PairedDevices  map[uint16]any
	SharedDevices  func(device *common.Device)
	DeviceList     map[string]*common.Device
	SingleDevice   bool
	Template       string
	Debug          bool
	Exit           bool
	timerKeepAlive *time.Ticker
	timerSleep     *time.Ticker
	keepAliveChan  chan struct{}
	sleepChan      chan struct{}
	instance       *common.Device
}

var (
	bufferSize       = 64
	bufferSizeWrite  = bufferSize + 1
	headerSize       = 4
	deviceKeepAlive  = 10000
	cmdSoftwareMode  = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode  = []byte{0x01, 0x03, 0x00, 0x01}
	cmdGetDevices    = []byte{0x24}
	cmdHeartbeat     = []byte{0x02, 0xe1}
	cmdInactivity    = []byte{0x02, 0x40}
	cmdOpenEndpoint  = []byte{0x0d, 0x00}
	cmdCloseEndpoint = []byte{0x05, 0x01}
	cmdGetFirmware   = []byte{0x02, 0x13}
	cmdRead          = []byte{0x08, 0x00}
	cmdWrite         = []byte{0x09, 0x00}
	cmdBatteryLevel  = []byte{0x02, 0x0f}
	cmdLogin         = []byte{0x1b, 0x01}
	cmdCommand       = byte(0x02)
	cmdBase          = byte(0x01)
	cmdDongle        = byte(0x00)
	transferTimeout  = 100
	connectDelay     = 10000
)

func Init(vendorId, productId uint16, _, path string, callback func(device *common.Device)) *common.Device {
	// Open device, return if failure
	dev, err := hid.OpenPath(path)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId, "path": path}).Error("Unable to open HID device")
		return nil
	}

	slipstream := &common.Slipstream{
		Dev:       dev,
		Mutex:     sync.Mutex{},
		Connected: map[uint16]bool{},
	}

	// Init new struct with HID device
	d := &Device{
		slipstream:     slipstream,
		VendorId:       vendorId,
		ProductId:      productId,
		PairedDevices:  make(map[uint16]any),
		DeviceList:     make(map[string]*common.Device),
		Template:       "slipstream.html",
		sleepChan:      make(chan struct{}),
		keepAliveChan:  make(chan struct{}),
		timerSleep:     &time.Ticker{},
		timerKeepAlive: &time.Ticker{},
		SharedDevices:  callback,
	}

	d.getDebugMode()         // Debug
	d.getManufacturer()      // Manufacturer
	d.getProduct()           // Product
	d.getSerial()            // Serial
	d.setDeviceLogin()       // Login
	d.getDeviceFirmware()    // Firmware
	d.setSoftwareMode()      // Switch to software mode
	d.getDevices()           // Get devices
	d.addDevices()           // Add devices
	d.monitorDevice()        // Monitor device
	d.sleepMonitor()         // Sleep
	d.backendListener()      // Control listener
	d.initAvailableDevices() // Init devices
	d.createDevice()         // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeSlipstream,
		Product:     "SLIPSTREAM V2",
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-dongle.svg",
		Instance:    d,
		Hidden:      true,
	}
}

// addDevices will add list of available devices
func (d *Device) addDevices() {
	for _, value := range d.Devices {
		switch value.ProductId {
		case 11041: // VANGUARD 99 AIR
			{
				dev := vanguard99airW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.slipstream,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeVanguard99AirW,
					Product:     "VANGUARD 99 AIR",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
					DeviceType:  common.DeviceTypeKeyboard,
					ProductId:   value.ProductId,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 11023: // VANGUARD 96 WIRELESS
			{
				dev := vanguard96W.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.slipstream,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeVanguard96W,
					Product:     "VANGUARD 96 WIRELESS",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
					DeviceType:  common.DeviceTypeKeyboard,
					ProductId:   value.ProductId,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		default:
			logger.Log(logger.Fields{"productId": value.ProductId}).Warn("Unsupported device detected")
		}
	}
}

// StopDirty will stop devices in a dirty way
func (d *Device) StopDirty() uint8 {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device (dirty)...")

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
		stopDirty := reflect.ValueOf(value).MethodByName("StopDirty")
		if !stopDirty.IsValid() {
			logger.Log(logger.Fields{"productId": d.ProductId}).Warn("Invalid or non-existing method called")
			continue
		}
		stopDirty.Call(nil)
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 2
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
		stopInternal := reflect.ValueOf(value).MethodByName("StopInternal")
		if !stopInternal.IsValid() {
			logger.Log(logger.Fields{"productId": d.ProductId}).Warn("Invalid or non-existing method called")
			continue
		}
		stopInternal.Call(nil)
	}

	time.Sleep(500 * time.Millisecond)
	d.setHardwareMode()
	if d.slipstream.Dev != nil {
		err := d.slipstream.Dev.Close()
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
func (d *Device) AddPairedDevice(productId uint16, device any, dev *common.Device) {
	d.PairedDevices[productId] = device
	d.DeviceList[dev.Serial] = dev
}

// GetDevice will return HID device
func (d *Device) GetDevice() *hid.Device {
	return d.slipstream.Dev
}

// getDevices will get a list of paired devices
func (d *Device) getDevices() {
	var devices = make(map[int]*Devices)
	buff := d.read(cmdGetDevices)
	if d.Debug {
		logger.Log(logger.Fields{"serial": d.Serial, "length": len(buff), "data": fmt.Sprintf("% 2x", buff)}).Info("DEBUG")
	}
	channels := buff[7]
	data := buff[8:]
	position := 0

	var base byte = 0x00
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
			logger.Log(logger.Fields{"serial": d.Serial, "device": device}).Info("Processing device")

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
	serial, err := d.slipstream.Dev.GetSerialNbr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device serial number")
	}
	d.Serial = serial
}

// getManufacturer will return device manufacturer
func (d *Device) getManufacturer() {
	manufacturer, err := d.slipstream.Dev.GetMfrStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get manufacturer")
	}
	d.Manufacturer = manufacturer
}

// getProduct will return device name
func (d *Device) getProduct() {
	product, err := d.slipstream.Dev.GetProductStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get product")
	}
	d.Product = product
}

// setDeviceLogin will set initial login packet
func (d *Device) setDeviceLogin() {
	var token [4]byte
	if _, err := rand.Read(token[:]); err != nil {
		return
	}
	buf := make([]byte, 4)
	copy(buf[0:4], token[:])

	_, err := d.transfer(cmdDongle, cmdBase, cmdLogin, buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to perform device login")
	}
}

// getDeviceFirmware will return a firmware version out as string
func (d *Device) getDeviceFirmware() {
	fw, err := d.transfer(cmdDongle, cmdCommand, cmdGetFirmware, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
	}

	v1, v2, v3 := int(fw[3]), int(fw[4]), int(binary.LittleEndian.Uint16(fw[5:7]))
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

// setHardwareMode will switch a device to hardware mode
func (d *Device) setHardwareMode() {
	_, err := d.transfer(cmdDongle, cmdCommand, cmdHardwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	_, err := d.transfer(cmdDongle, cmdCommand, cmdSoftwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
}

func (d *Device) readNext() []byte {
	buffer, err := d.transfer(cmdDongle, cmdCommand, cmdRead, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read endpoint")
	}
	return buffer
}

// read will read data from a device and return data as a byte array
func (d *Device) read(endpoint []byte) []byte {
	var buffer []byte

	_, err := d.transfer(cmdDongle, cmdCommand, cmdCloseEndpoint, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close endpoint")
	}

	_, err = d.transfer(cmdDongle, cmdCommand, cmdOpenEndpoint, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to open endpoint")
	}

	_, err = d.transfer(cmdDongle, cmdCommand, cmdWrite, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to open endpoint")
	}

	buffer, err = d.transfer(cmdDongle, cmdCommand, cmdRead, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read endpoint")
	}

	for i := 1; i < int(buffer[5]); i++ {
		next, e := d.transfer(cmdDongle, cmdCommand, cmdRead, endpoint)
		if e != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to read endpoint")
		}
		buffer = append(buffer, next[3:]...)
	}

	_, err = d.transfer(cmdDongle, cmdCommand, cmdCloseEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close endpoint")
	}
	return buffer
}

// initAvailableDevices will run on initial start
func (d *Device) initAvailableDevices() {
	for _, value := range d.Devices {
		_, err := d.transferToDevice(value.Endpoint, []byte{0x02, 0xe1}, nil, "initAvailableDevices")
		if err != nil {
			logger.Log(logger.Fields{"error": err, "endpoint": value.Endpoint, "productId": value.ProductId}).Warn("Unable to read endpoint. Device is probably offline")
			continue
		}

		if _, ok := d.PairedDevices[value.ProductId]; ok {
			connect := reflect.ValueOf(d.PairedDevices[value.ProductId]).MethodByName("Connect")
			if !connect.IsValid() {
				logger.Log(logger.Fields{"endpoint": value.Endpoint, "productId": value.ProductId}).Warn("Invalid or non-existing method called")
				return
			}
			connect.Call(nil)
			d.slipstream.Connected[value.ProductId] = true
		}
	}
}

// setDeviceOnlineByProductId will online device by given productId
func (d *Device) setDeviceBatteryLevelByProductId(productId, batteryLevel uint16) {
	if _, ok := d.PairedDevices[productId]; ok {
		battery := reflect.ValueOf(d.PairedDevices[productId]).MethodByName("ModifyBatteryLevel")
		if !battery.IsValid() {
			logger.Log(logger.Fields{"productId": productId}).Warn("Invalid or non-existing method called")
			return
		}

		reflectArgs := make([]reflect.Value, 1)
		reflectArgs[0] = reflect.ValueOf(batteryLevel)
		battery.Call(reflectArgs)
	}
}

// setDeviceStatus will set device status
func (d *Device) setDeviceStatus(status byte, productId uint16) {
	for _, v := range d.Devices {
		if productId == v.ProductId {
			if _, ok := d.PairedDevices[productId]; ok {
				setConnected := reflect.ValueOf(d.PairedDevices[productId]).MethodByName("SetConnected")
				if !setConnected.IsValid() {
					logger.Log(logger.Fields{"productId": v.ProductId, "method": "SetConnected"}).Warn("Invalid or non-existing method called")
					return
				}

				connect := reflect.ValueOf(d.PairedDevices[productId]).MethodByName("Connect")
				if !connect.IsValid() {
					logger.Log(logger.Fields{"productId": v.ProductId, "method": "Connect"}).Warn("Invalid or non-existing method called")
					return
				}

				switch status {
				case 0x00:
					reflectArgs := make([]reflect.Value, 1)
					reflectArgs[0] = reflect.ValueOf(false)
					setConnected.Call(reflectArgs)
					d.slipstream.Connected[v.ProductId] = false
				case 0x02:
					go func() {
						time.Sleep(time.Duration(connectDelay) * time.Millisecond)
						connect.Call(nil)
						d.slipstream.Connected[v.ProductId] = true
					}()
				}
			}
		} else {
			for _, val := range d.DeviceList {
				if _, ok := d.PairedDevices[val.ProductId]; ok {
					setConnected := reflect.ValueOf(d.PairedDevices[val.ProductId]).MethodByName("SetConnected")
					if !setConnected.IsValid() {
						logger.Log(logger.Fields{"productId": val.ProductId, "method": "SetConnected"}).Warn("Invalid or non-existing method called")
						return
					}

					connect := reflect.ValueOf(d.PairedDevices[val.ProductId]).MethodByName("Connect")
					if !connect.IsValid() {
						logger.Log(logger.Fields{"productId": val.ProductId, "method": "Connect"}).Warn("Invalid or non-existing method called")
						return
					}

					switch status {
					case 0x00:
						reflectArgs := make([]reflect.Value, 1)
						reflectArgs[0] = reflect.ValueOf(false)
						setConnected.Call(reflectArgs)
						d.slipstream.Connected[val.ProductId] = false
					case 0x02:
						go func() {
							time.Sleep(time.Duration(connectDelay) * time.Millisecond)
							connect.Call(nil)
							d.slipstream.Connected[v.ProductId] = true
						}()
					case 0x04:
						if val.DeviceType == common.DeviceTypeMouse {
							go func() {
								time.Sleep(time.Duration(connectDelay) * time.Millisecond)
								connect.Call(nil)
								d.slipstream.Connected[val.ProductId] = true
							}()
						} else {
							reflectArgs := make([]reflect.Value, 1)
							reflectArgs[0] = reflect.ValueOf(false)
							setConnected.Call(reflectArgs)
							d.slipstream.Connected[val.ProductId] = false
						}
					case 0x08:
						if val.DeviceType == common.DeviceTypeKeyboard {
							go func() {
								time.Sleep(time.Duration(connectDelay) * time.Millisecond)
								connect.Call(nil)
								d.slipstream.Connected[val.ProductId] = true
							}()
						} else {
							reflectArgs := make([]reflect.Value, 1)
							reflectArgs[0] = reflect.ValueOf(false)
							setConnected.Call(reflectArgs)
							d.slipstream.Connected[val.ProductId] = false
						}
					case 0x0c:
						go func() {
							time.Sleep(time.Duration(connectDelay) * time.Millisecond)
							connect.Call(nil)
							d.slipstream.Connected[val.ProductId] = true
						}()
					}
				}
			}
		}
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
					_, err := d.transfer(cmdDongle, cmdCommand, cmdHeartbeat, nil)
					if err != nil {
						logger.Log(logger.Fields{"error": err}).Error("Unable to read slipstream endpoint")
					}
					for _, value := range d.Devices {
						if d.slipstream.Connected[value.ProductId] {
							_, e := d.transfer(value.Endpoint, value.Endpoint, cmdHeartbeat, nil)
							if e != nil {
								if d.Debug {
									logger.Log(logger.Fields{"error": e}).Error("Unable to read paired device endpoint")
								}
								continue
							}

							batteryLevel, e := d.transfer(value.Endpoint, value.Endpoint, cmdBatteryLevel, nil)
							if e != nil {
								if d.Debug {
									logger.Log(logger.Fields{"error": e}).Error("Unable to read paired device endpoint for battery status")
								}
								continue
							}
							val := binary.LittleEndian.Uint16(batteryLevel[5:7])
							if val > 0 {
								d.setDeviceBatteryLevelByProductId(value.ProductId, val/10)
							}
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
		if d.Debug {
			logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device")
		}
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
						if d.slipstream.Connected[value.ProductId] {
							msg, err := d.transfer(value.Endpoint, value.Endpoint, cmdInactivity, nil)
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
												if d.Debug {
													logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
												}
												continue
											} else {
												if inactive >= sleepMode {
													d.slipstream.Connected[value.ProductId] = false
													method.Call(nil)
												}
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
	n, err := d.slipstream.Listener.ReadWithTimeout(data, 100*time.Millisecond)
	if err != nil || n == 0 {
		return nil
	}
	return data
}

func (d *Device) TriggerKeyboardKeyAssignment(data []byte) {
	for _, val := range d.DeviceList {
		if _, ok := d.PairedDevices[val.ProductId]; ok {
			if val.DeviceType == common.DeviceTypeKeyboard {
				keyAssignment := reflect.ValueOf(d.PairedDevices[val.ProductId]).MethodByName("TriggerKeyAssignment")
				if !keyAssignment.IsValid() {
					logger.Log(logger.Fields{"productId": val.ProductId, "method": "TriggerKeyAssignment"}).Warn("Invalid or non-existing method called")
					return
				}

				reflectArgs := make([]reflect.Value, 1)
				reflectArgs[0] = reflect.ValueOf(data)
				keyAssignment.Call(reflectArgs)
			}
		}
	}
}

func (d *Device) TriggerMousedKeyAssignment(data []byte) {
	for _, val := range d.DeviceList {
		if device, ok := d.PairedDevices[val.ProductId]; ok {
			if val.DeviceType == common.DeviceTypeMouse {
				keyAssignment := reflect.ValueOf(d.PairedDevices[val.ProductId]).MethodByName("TriggerKeyAssignment")
				if !keyAssignment.IsValid() {
					logger.Log(logger.Fields{"productId": val.ProductId, "method": "TriggerKeyAssignment"}).Warn("Invalid or non-existing method called")
					return
				}

				value := binary.LittleEndian.Uint32(data[2:6])
				reflectArgs := make([]reflect.Value, 1)
				reflectArgs[0] = reflect.ValueOf(value)
				keyAssignment.Call(reflectArgs)

				// Tilt
				if data[1] == 0x09 {
					var base uint32 = 32768

					// Override
					if _, found := device.(*m65rgbultraW.Device); found {
						base = 256
					}

					v := binary.LittleEndian.Uint32(data[3:7])
					switch v {
					case 1:
						v = base
						break
					case 257:
						v = base * 2
						break
					case 513:
						v = base * 4
						break
					case 769:
						v = base * 8
						break
					}

					tilt := reflect.ValueOf(d.PairedDevices[val.ProductId]).MethodByName("TriggerTiltAssignment")
					if !tilt.IsValid() {
						logger.Log(logger.Fields{"productId": val.ProductId, "method": "TriggerTiltAssignment"}).Warn("Invalid or non-existing method called")
						return
					}

					args := make([]reflect.Value, 1)
					args[0] = reflect.ValueOf(v)
					tilt.Call(args)
				}
			}
		}
	}
}

// backendListener will listen for events from the device
func (d *Device) backendListener() {
	go func() {
		enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
			if info.InterfaceNbr == 2 && info.SerialNbr == d.Serial {
				listener, err := hid.OpenPath(info.Path)
				if err != nil {
					return err
				}
				d.slipstream.Listener = listener
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
					err = d.slipstream.Listener.Close()
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

				if d.Debug {
					logger.Log(logger.Fields{"data": fmt.Sprintf("% 2x", data)}).Info("Backend debug data")
				}

				if data[3] == 0x01 && data[4] == 0x36 {
					value := data[6]
					productId := uint16(data[11])<<8 | uint16(data[10])
					d.setDeviceStatus(value, productId)
				} else {
					if data[3] == 0x02 || data[3] == 0x05 {
						switch data[0] {
						case 1:
							d.TriggerKeyboardKeyAssignment(data)
						}
					}
				}
			}
		}
	}()
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(target, command byte, endpoint, buffer []byte) ([]byte, error) {
	if d.slipstream == nil || d.slipstream.Dev == nil {
		return nil, errors.New("slipstream device is not initialized")
	}

	d.slipstream.Mutex.Lock()
	defer d.slipstream.Mutex.Unlock()

	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = target
	bufferW[2] = 0x01
	bufferW[3] = command
	endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
	}

	reports := make([]byte, 1)
	err := d.slipstream.Dev.SetNonblock(true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to SetNonblock")
	}

	for {
		n, e := d.slipstream.Dev.Read(reports)
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

	err = d.slipstream.Dev.SetNonblock(false)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to SetNonblock")
	}

	bufferR := make([]byte, bufferSize)

	if _, e := d.slipstream.Dev.Write(bufferW); e != nil {
		if d.Debug {
			logger.Log(logger.Fields{"error": e, "serial": d.Serial}).Error("Unable to write to a device")
		}
		return bufferR, e
	}

	if _, e := d.slipstream.Dev.ReadWithTimeout(bufferR, time.Duration(transferTimeout)*time.Millisecond); e != nil {
		if d.Debug {
			logger.Log(logger.Fields{"error": e, "serial": d.Serial}).Error("Unable to read data from device")
		}
		return bufferR, e
	}
	return bufferR, nil
}

// transfer will send data to a device and retrieve device output
func (d *Device) transferToDevice(command byte, endpoint, buffer []byte, caller string) ([]byte, error) {
	if d.slipstream == nil || d.slipstream.Dev == nil {
		return nil, errors.New("slipstream device is not initialized")
	}

	d.slipstream.Mutex.Lock()
	defer d.slipstream.Mutex.Unlock()

	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = 0x01
	bufferW[2] = 0x01
	bufferW[3] = command
	endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
	}

	bufferR := make([]byte, bufferSize)

	if _, err := d.slipstream.Dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return bufferR, err
	}

	if _, err := d.slipstream.Dev.ReadWithTimeout(bufferR, time.Duration(transferTimeout)*time.Millisecond); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "caller": caller}).Error("Unable to read data from device")
		return bufferR, err
	}
	return bufferR, nil
}
