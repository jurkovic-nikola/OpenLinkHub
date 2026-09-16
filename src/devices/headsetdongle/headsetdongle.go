package headsetdongle

// Package: Headset Dongle
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices/hs80rgbW"
	"OpenLinkHub/src/devices/ironclawW"
	"OpenLinkHub/src/devices/virtuosoSEW"
	"OpenLinkHub/src/devices/virtuosoW"
	"OpenLinkHub/src/devices/virtuosorgbXTW"
	"OpenLinkHub/src/logger"
	"encoding/binary"
	"fmt"
	"github.com/sstallion/go-hid"
	"slices"
	"strconv"
	"strings"
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
	slipstream     *common.Slipstream
	listener       *hid.Device
	Manufacturer   string `json:"manufacturer"`
	Product        string `json:"product"`
	Serial         string `json:"serial"`
	Firmware       string `json:"firmware"`
	ProductId      uint16
	VendorId       uint16
	Devices        map[int]*Devices `json:"devices"`
	SharedDevices  func(device *common.Device)
	DeviceList     map[string]*common.Device
	PairedDevices  map[uint16]any
	SingleDevice   bool
	Template       string
	Debug          bool
	Exit           bool
	timerKeepAlive *time.Ticker
	keepAliveChan  chan struct{}
	instance       *common.Device
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
	cmdProductId     = []byte{0x02, 0x12}
	cmdCommand       = byte(0x08)
	cmdEndpoint      = byte(0x09)
	transferTimeout  = 1000
	connectDelay     = 3000
)

func Init(vendorId, productId uint16, _, path string, callback func(device *common.Device)) *common.Device {
	// Open device, return if failure
	dev, err := hid.OpenPath(path)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId}).Error("Unable to open HID device")
		return nil
	}

	// Paired mice get their own handle: every handle receives a copy of each
	// report, so headset transfers draining or reading theirs cannot consume
	// responses meant for the mouse.
	pairedDev, err := hid.OpenPath(path)
	if err != nil {
		pairedDev = dev
	}

	// Init new struct with HID device
	d := &Device{
		dev: dev,
		slipstream: &common.Slipstream{
			Dev:       pairedDev,
			Connected: map[uint16]bool{},
			Prefix:    []byte{0x02},
		},
		VendorId:       vendorId,
		ProductId:      productId,
		PairedDevices:  make(map[uint16]any),
		DeviceList:     make(map[string]*common.Device),
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
	d.getDevices()           // Get devices
	d.addDevices()           // Add devices
	d.monitorDevice()        // Monitor device
	d.backendListener()      // Control listener
	d.initAvailableDevices() // Init devices
	d.createDevice()         // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// addDevices adda a mew device
func (d *Device) addDevices() {
	for _, value := range d.Devices {
		// Already added on a previous scan
		if _, ok := d.PairedDevices[value.ProductId]; ok {
			continue
		}

		switch value.ProductId {
		case 2658:
			{
				dev := virtuosorgbXTW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeVirtuosoXTW,
					Product:     "VIRTUOSO XT",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-headphone.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 2621:
			{
				dev := virtuosoSEW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeVirtuosoSEW,
					Product:     "VIRTUOSO SE",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-headphone.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 2623:
			{
				dev := virtuosoSEW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeVirtuosoSEW,
					Product:     "VIRTUOSO SE",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-headphone.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 2665:
			{
				dev := hs80rgbW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)
				object := &common.Device{
					ProductType: common.ProductTypeHS80RGBW,
					Product:     "HS80 RGB WIRELESS",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-headphone.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 2673:
			{
				dev := hs80rgbW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)
				object := &common.Device{
					ProductType: common.ProductTypeHS80RGBW,
					Product:     "HS80 RGB WIRELESS",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-headphone.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 2627:
			{
				dev := virtuosoW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeVirtuosoW,
					Product:     "VIRTUOSO",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-headphone.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 2625:
			{
				dev := virtuosoW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeVirtuosoW,
					Product:     "VIRTUOSO",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-headphone.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 6988: // IRONCLAW RGB WIRELESS
			{
				dev := ironclawW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.slipstream,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeIronClawRgbW,
					Product:     "IRONCLAW RGB",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
					DeviceType:  common.DeviceTypeMouse,
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

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeIronClawRgbW,
		Product:     "HEADSET DONGLE",
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
		if dev, found := value.(*virtuosoSEW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*hs80rgbW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*virtuosoW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*ironclawW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
	}

	d.setHardwareMode()
	if d.slipstream.Dev != nil && d.slipstream.Dev != d.dev {
		err := d.slipstream.Dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
		}
	}
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
		if dev, found := value.(*virtuosoSEW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*hs80rgbW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*virtuosoW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*ironclawW.Device); found {
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
	noProductType := false
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
			nullTerminator := false
			vendorId := uint16(data[position+1])<<8 | uint16(data[position])
			if data[position+2] == 0x00 && data[position+3] == 0x00 {
				productId := uint16(data[position+5])<<8 | uint16(data[position+4])
				deviceType := data[position+6]
				deviceIdLen := data[position+7]
				if position+8+int(deviceIdLen) > len(data) {
					logger.Log(logger.Fields{"serial": d.Serial, "length": len(buff), "position": position + 8 + int(deviceIdLen), "data": fmt.Sprintf("% 2x", buff)}).Warn("Requested position exceeds maximum length")
					continue
				}
				deviceId := data[position+8 : position+8+int(deviceIdLen)]
				if slices.Contains(deviceId, 0x00) && position+8+int(deviceIdLen)+1 <= len(data) {
					// Some device serials have random null terminator in data
					deviceId = data[position+8 : position+8+int(deviceIdLen)+1]
					nullTerminator = true
				}
				serial := strings.ReplaceAll(string(deviceId), "\x00", "")

				endpoint := base + deviceType
				if channels == 1 {
					endpoint = base + 1
				}

				device := &Devices{
					Type:      deviceType,
					Endpoint:  endpoint,
					Serial:    serial,
					VendorId:  vendorId,
					ProductId: productId,
				}
				logger.Log(logger.Fields{"serial": d.Serial, "device": device}).Info("Processing device")

				devices[i] = device
				if nullTerminator {
					position += 8 + int(deviceIdLen) + 1
				} else {
					position += 8 + int(deviceIdLen)
				}
			} else {
				noProductType = true
				productId := uint16(data[position+3])<<8 | uint16(data[position+2])
				deviceIdLen := data[position+4]
				if position+5+int(deviceIdLen) > len(data) {
					logger.Log(logger.Fields{"serial": d.Serial, "length": len(buff), "position": position + 5 + int(deviceIdLen), "data": fmt.Sprintf("% 2x", buff)}).Warn("Requested position exceeds maximum length")
					continue
				}

				deviceId := data[position+5 : position+5+int(deviceIdLen)]
				if slices.Contains(deviceId, 0x00) && position+5+int(deviceIdLen)+1 <= len(data) {
					// Some device serials have random null terminator in data
					deviceId = data[position+5 : position+5+int(deviceIdLen)+1]
					nullTerminator = true
				}
				serial := strings.ReplaceAll(string(deviceId), "\x00", "")

				endpoint := base + (byte(i) + 1)
				if channels == 1 {
					endpoint = base + 1
				}

				device := &Devices{
					Type:      byte(i) + 1,
					Endpoint:  endpoint,
					Serial:    serial,
					VendorId:  vendorId,
					ProductId: productId,
				}
				logger.Log(logger.Fields{"serial": d.Serial, "device": device}).Info("Processing device")

				devices[i] = device
				if nullTerminator {
					position += 5 + int(deviceIdLen) + 1
				} else {
					position += 5 + int(deviceIdLen)
				}
			}
		}
	} else {
		switch d.ProductId {
		case 2622, 2624:
			buff, _ = d.transferToDevice(cmdEndpoint, cmdProductId, nil, "")
			if d.Debug {
				logger.Log(logger.Fields{"serial": d.Serial, "length": len(buff), "data": fmt.Sprintf("% 2x", buff)}).Info("DEBUG")
			}
			productId := uint16(buff[position+5])<<8 | uint16(buff[position+4])
			device := &Devices{
				Type:      0x02,
				Endpoint:  0x09, // Single endpoint
				Serial:    strconv.Itoa(int(productId)),
				VendorId:  d.VendorId,
				ProductId: productId,
			}
			if d.Debug {
				logger.Log(logger.Fields{"serial": d.Serial, "device": device}).Info("Processing device")
			}
			devices[0] = device
		}
	}

	if len(devices) == 1 {
		d.SingleDevice = true
	}
	d.Devices = devices

	if noProductType {
		// Find productId and match with endpoint for devices that do not expose valid product type
		logger.Log(logger.Fields{"serial": d.Serial}).Info("No valid product type, probing devices...")
		position = 0
		for key, device := range d.Devices {
			logger.Log(logger.Fields{"serial": d.Serial, "device": device.ProductId, "endpoint": device.Endpoint}).Info("Probing...")

			buff, _ = d.transferToDevice(device.Endpoint, cmdProductId, nil, "")
			if d.Debug {
				logger.Log(logger.Fields{"serial": d.Serial, "length": len(buff), "data": fmt.Sprintf("% 2x", buff)}).Info("DEBUG")
			}
			productId := uint16(buff[position+5])<<8 | uint16(buff[position+4])
			for _, dev := range d.Devices {
				if dev.ProductId == productId {
					logger.Log(logger.Fields{"serial": d.Serial, "device": device.ProductId, "endpoint": dev.Endpoint, "new-endpoint": device.Endpoint}).Info("Device match found")
					dev.Endpoint = device.Endpoint
					devices[key] = dev
					break
				}
			}
		}
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

	for i := 1; i < int(buffer[6]); i++ {
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

// initAvailableDevices will run on initial start
func (d *Device) initAvailableDevices() {
	for _, value := range d.Devices {
		_, err := d.transferToDevice(value.Endpoint, cmdHeartbeat, nil, "initAvailableDevices")
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
		if device, found := dev.(*virtuosoSEW.Device); found {
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
		if device, found := dev.(*virtuosoW.Device); found {
			if !device.Connected {
				time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
				device.Connect()
			}
		}
		if device, found := dev.(*ironclawW.Device); found {
			if !device.Connected {
				time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
				device.Connect()
				d.slipstream.Connected[productId] = device.Connected
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
		if device, found := pairedDevice.(*virtuosoSEW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*hs80rgbW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*virtuosoW.Device); found {
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
				d.SharedDevices(d.DeviceList[device.Serial])
			}
		}
		if device, found := pairedDevice.(*virtuosoSEW.Device); found {
			if !device.Connected {
				time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
				device.Connect()
				d.SharedDevices(d.DeviceList[device.Serial])
			}
		}
		if device, found := pairedDevice.(*hs80rgbW.Device); found {
			if !device.Connected {
				time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
				device.Connect()
				d.SharedDevices(d.DeviceList[device.Serial])
			}
		}
		if device, found := pairedDevice.(*virtuosoW.Device); found {
			if !device.Connected {
				time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
				device.Connect()
				d.SharedDevices(d.DeviceList[device.Serial])
			}
		}
	}
}

// rescanDevices will re-read the paired device list when a device comes online
// that was not known yet. A device asleep while the dongle initializes is
// reported with product id 0 and would otherwise stay missing until restart.
func (d *Device) rescanDevices(status byte) {
	if d.SingleDevice {
		return
	}

	var known byte = 0
	for _, value := range d.Devices {
		if _, ok := d.PairedDevices[value.ProductId]; ok {
			known |= 1 << value.Type
		}
	}
	if status&^known == 0 {
		return
	}

	logger.Log(logger.Fields{"serial": d.Serial, "status": status}).Info("Unknown paired device online, rescanning...")
	d.getDevices()
	d.addDevices()
	for _, value := range d.Devices {
		if known&(1<<value.Type) == 0 {
			d.setDeviceOnlineByProductId(value.ProductId)
		}
	}
}

// getMouseStatusMask will return status bits used by paired mice. The status
// report is a bitmask with one bit per paired device type (1 << type).
func (d *Device) getMouseStatusMask() byte {
	var mask byte = 0
	for _, value := range d.Devices {
		if _, found := d.PairedDevices[value.ProductId].(*ironclawW.Device); found {
			mask |= 1 << value.Type
		}
	}
	return mask
}

// setMouseStatus will connect or disconnect paired mice based on status bits
func (d *Device) setMouseStatus(status byte) {
	for _, value := range d.Devices {
		device, found := d.PairedDevices[value.ProductId].(*ironclawW.Device)
		if !found {
			continue
		}

		online := status&(1<<value.Type) != 0
		if !online && device.Connected {
			device.SetConnected(false)
			d.slipstream.Connected[value.ProductId] = false
		}

		if online && !device.Connected {
			go func(device *ironclawW.Device, productId uint16) {
				// Mouse wakes up in hardware mode and needs a full re-init
				time.Sleep(time.Duration(connectDelay) * time.Millisecond)
				device.Connect()
				d.slipstream.Connected[productId] = device.Connected
				d.SharedDevices(d.DeviceList[device.Serial])
			}(device, value.ProductId)
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

// getMouseBySlot will return a paired mouse communicating on a given slot
func (d *Device) getMouseBySlot(slot byte) (*ironclawW.Device, bool) {
	for _, value := range d.Devices {
		if value.Endpoint-0x08 != slot {
			continue
		}
		if dev, found := d.PairedDevices[value.ProductId].(*ironclawW.Device); found {
			return dev, true
		}
	}
	return nil, false
}

// getListenerData will listen for keyboard events and return data on success or nil on failure.
// ReadWithTimeout is mandatory due to the nature of listening for events
func (d *Device) getListenerData() []byte {
	if d.listener == nil {
		return nil
	}
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
			if info.UsagePage == 65346 {
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

		if d.listener == nil {
			logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to open device listener")
			return
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
					time.Sleep(5 * time.Millisecond)
					continue
				}

				if d.Debug {
					logger.Log(logger.Fields{"data": fmt.Sprintf("% 2x", data)}).Info("Backend debug data")
				}

				// Events from a paired mouse, reported on its own slot. Handled
				// first, since a button mask can look like a headset mute event.
				if data[0] == 0x03 && data[1] != 0x00 {
					if dev, found := d.getMouseBySlot(data[1]); found {
						switch data[2] {
						case 0x01: // Battery: 03 02 01 0f 00 ee 02
							if data[3] == 0x0f {
								val := binary.LittleEndian.Uint16(data[5:7]) / 10
								if val > 0 {
									dev.ModifyBatteryLevel(val)
								}
							}
						case 0x02: // Buttons: 03 02 02 <mask>
							dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[3:7]))
						}
						continue
					}
				}

				// Battery
				// 03 01 01 0f 00 ac 03
				if (data[0] == 0x03 || data[1] == 0x01) && data[3] == 0x0f {
					val := binary.LittleEndian.Uint16(data[5:7]) / 10
					if val > 0 {
						for _, value := range d.PairedDevices {
							if dev, found := value.(*virtuosorgbXTW.Device); found {
								dev.ModifyBatteryLevel(val)
							}
							if dev, found := value.(*virtuosoSEW.Device); found {
								dev.ModifyBatteryLevel(val)
							}
							if dev, found := value.(*hs80rgbW.Device); found {
								dev.ModifyBatteryLevel(val)
							}
							if dev, found := value.(*virtuosoW.Device); found {
								dev.ModifyBatteryLevel(val)
							}
						}
					}
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
							if dev, found := value.(*virtuosoSEW.Device); found {
								dev.ModifyBatteryLevel(val)
							}
							if dev, found := value.(*hs80rgbW.Device); found {
								dev.ModifyBatteryLevel(val)
							}
							if dev, found := value.(*virtuosoW.Device); found {
								dev.ModifyBatteryLevel(val)
							}
						}
					}
				}

				if data[1] == 0x00 && data[3] == 0x36 {
					value := data[5]
					// 03 00 01 36 00 06: headset (0x02) and mouse (0x04) online
					d.rescanDevices(value)
					mouseMask := d.getMouseStatusMask()
					d.setMouseStatus(value & mouseMask)
					d.setDeviceStatus(value &^ mouseMask)
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
					} else if data[2] == 0x02 && data[3] == 0x01 {
						for _, value := range d.PairedDevices {
							if dev, found := value.(*virtuosoSEW.Device); found {
								dev.NotifyMuteChanged(data[3])
							}
							if dev, found := value.(*virtuosoW.Device); found {
								dev.NotifyMuteChanged(data[3])
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
	// Shared with paired devices that talk to the dongle directly
	d.slipstream.Mutex.Lock()
	defer d.slipstream.Mutex.Unlock()

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

	bufferR := make([]byte, bufferSize)

	if _, err := d.dev.Write(bufferW); err != nil {
		if d.Debug {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		}
		return bufferR, err
	}

	// Responses carry their slot; skip events and other paired devices
	if err := common.ReadResponse(d.dev, bufferR, command-0x08, endpoint, time.Duration(transferTimeout)*time.Millisecond); err != nil {
		if d.Debug {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		}
		return bufferR, err
	}
	return bufferR, nil
}

// transfer will send data to a device and retrieve device output
func (d *Device) transferToDevice(command byte, endpoint, buffer []byte, caller string) ([]byte, error) {
	// Shared with paired devices that talk to the dongle directly
	d.slipstream.Mutex.Lock()
	defer d.slipstream.Mutex.Unlock()

	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = 0x02
	bufferW[2] = command
	endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
	}

	bufferR := make([]byte, bufferSize)

	if _, err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return bufferR, err
	}

	// Responses carry their slot; skip events and other paired devices
	if err := common.ReadResponse(d.dev, bufferR, command-0x08, endpoint, time.Duration(transferTimeout)*time.Millisecond); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "caller": caller}).Error("Unable to read data from device")
		return bufferR, err
	}
	return bufferR, nil
}
