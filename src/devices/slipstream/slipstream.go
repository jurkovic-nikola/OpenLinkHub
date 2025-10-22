package slipstream

// Package: Corsair Slipstream
// This is the primary package for Corsair Slipstream.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices/darkcorergbproW"
	"OpenLinkHub/src/devices/darkcorergbproseW"
	"OpenLinkHub/src/devices/darkstarW"
	"OpenLinkHub/src/devices/harpoonW"
	"OpenLinkHub/src/devices/ironclawW"
	"OpenLinkHub/src/devices/k100airW"
	"OpenLinkHub/src/devices/k57rgbW"
	"OpenLinkHub/src/devices/k70coretklW"
	"OpenLinkHub/src/devices/k70pmW"
	"OpenLinkHub/src/devices/m55W"
	"OpenLinkHub/src/devices/m65rgbultraW"
	"OpenLinkHub/src/devices/m75AirW"
	"OpenLinkHub/src/devices/m75W"
	"OpenLinkHub/src/devices/makr75W"
	"OpenLinkHub/src/devices/nightsabreW"
	"OpenLinkHub/src/devices/sabrergbproW"
	"OpenLinkHub/src/devices/scimitarSEW"
	"OpenLinkHub/src/devices/scimitarW"
	"OpenLinkHub/src/logger"
	"encoding/binary"
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
	mutex          sync.Mutex
	instance       *common.Device
}

var (
	bufferSize       = 64
	bufferSizeWrite  = bufferSize + 1
	headerSize       = 2
	deviceKeepAlive  = 10000
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
	cmdBatteryLevel  = []byte{0x02, 0x0f}
	cmdCommand       = byte(0x08)
	transferTimeout  = 100
	connectDelay     = 5000
)

func Init(vendorId, productId uint16, _, path string, callback func(device *common.Device)) *common.Device {
	// Open device, return if failure
	dev, err := hid.OpenPath(path)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId, "path": path}).Error("Unable to open HID device")
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
		Product:     "SLIPSTREAM",
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
		case 7163: // M55
			{
				dev := m55W.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeM55W,
					Product:     "M55 WIRELESS",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 7131: // SCIMITAR
			{
				dev := scimitarW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeScimitarRgbEliteW,
					Product:     "SCIMITAR RGB ELITE",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 11042: // CORSAIR SCIMITAR ELITE WIRELESS SE
			{
				dev := scimitarSEW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeScimitarRgbEliteSEW,
					Product:     "SCIMITAR ELITE SE",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 7096: // NIGHTSABRE
			{
				dev := nightsabreW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeNightsabreW,
					Product:     "NIGHTSABRE",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}

				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 7083: // K100 AIR WIRELESS
			{
				dev := k100airW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeK100AirW,
					Product:     "K100 AIR",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
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
					d.dev,
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
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 7038: // DARK CORE RGB PRO SE WIRELESS
			{
				dev := darkcorergbproseW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeDarkCoreRgbProSEW,
					Product:     "DARK CORE RGB PRO SE",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 7040: // DARK CORE RGB PRO WIRELESS
			{
				dev := darkcorergbproW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeDarkCoreRgbProW,
					Product:     "DARK CORE RGB PRO",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 11016: // M75 WIRELESS
			{
				dev := m75W.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeM75W,
					Product:     "M75 WIRELESS",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 7154: // M75 AIR WIRELESS
			{
				dev := m75AirW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeM75AirW,
					Product:     "M75 AIR WIRELESS",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 7006: // HARPOON RGB WIRELESS
			{
				dev := harpoonW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeHarpoonRgbW,
					Product:     "HARPOON RGB",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 7090: // CORSAIR DARKSTAR RGB WIRELESS Gaming Mouse
			{
				dev := darkstarW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeDarkstarW,
					Product:     "DARKSTAR",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 7093, 7126: // M65 RGB ULTRA WIRELESS Gaming Mouse
			{
				dev := m65rgbultraW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeM65RgbUltraW,
					Product:     "M65 RGB ULTRA",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 11010: // K70 CORE TKL WIRELESS
			{
				dev := k70coretklW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
					d.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeK70CoreTklW,
					Product:     "K70 CORE TKL",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 7064: // CORSAIR SABRE RGB PRO WIRELESS Gaming Mouse
			{
				dev := sabrergbproW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeSabreRgbProW,
					Product:     "SABRE RGB PRO",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-mouse.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 7094: // K70 PRO MINI
			{
				dev := k70pmW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeK70PMW,
					Product:     "K70 PRO MINI",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 11012: // MAKR 75
			{
				dev := makr75W.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeMakr75W,
					Product:     "MAKR 75",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
				}
				d.SharedDevices(object)
				d.AddPairedDevice(value.ProductId, dev, object)
			}
		case 7022: // K57 RGB WIRELESS
			{
				dev := k57rgbW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.dev,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeK57RgbW,
					Product:     "K57 RGB",
					Serial:      dev.Serial,
					Firmware:    dev.Firmware,
					Image:       "icon-keyboard.svg",
					Instance:    dev,
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
		if dev, found := value.(*m55W.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*nightsabreW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*k100airW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*k70pmW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*ironclawW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*scimitarW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*scimitarSEW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*darkcorergbproseW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*darkcorergbproW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*m75W.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*m75AirW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*harpoonW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*darkstarW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*k70coretklW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*m65rgbultraW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*sabrergbproW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*makr75W.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
		if dev, found := value.(*k57rgbW.Device); found {
			if dev.Connected {
				dev.StopDirty()
			}
		}
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
		if dev, found := value.(*makr75W.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*k57rgbW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*k70pmW.Device); found {
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
		if dev, found := value.(*scimitarSEW.Device); found {
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
		if dev, found := value.(*m75AirW.Device); found {
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
		if dev, found := value.(*m65rgbultraW.Device); found {
			if dev.Connected {
				dev.StopInternal()
			}
		}
		if dev, found := value.(*sabrergbproW.Device); found {
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
func (d *Device) setDeviceBatteryLevelByProductId(productId, batteryLevel uint16) {
	if dev, ok := d.PairedDevices[productId]; ok {
		if device, found := dev.(*k100airW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*makr75W.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*k57rgbW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*k70pmW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*darkcorergbproW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*darkcorergbproseW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*darkstarW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*nightsabreW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*scimitarW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*scimitarSEW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*m55W.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*m75W.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*m75AirW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*ironclawW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*harpoonW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*k70coretklW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*m65rgbultraW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
		if device, found := dev.(*sabrergbproW.Device); found {
			if device.Connected {
				device.ModifyBatteryLevel(batteryLevel)
			}
		}
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
		if device, found := dev.(*scimitarSEW.Device); found {
			if !device.Connected {
				time.Sleep(2000 * time.Millisecond)
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
		if device, found := dev.(*makr75W.Device); found {
			if !device.Connected {
				device.Connect()
			}
		}
		if device, found := dev.(*k57rgbW.Device); found {
			if !device.Connected {
				device.Connect()
			}
		}
		if device, found := dev.(*k70pmW.Device); found {
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
		if device, found := dev.(*m75AirW.Device); found {
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
		if device, found := dev.(*m65rgbultraW.Device); found {
			if !device.Connected {
				device.Connect()
			}
		}
		if device, found := dev.(*sabrergbproW.Device); found {
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
		if device, found := pairedDevice.(*scimitarSEW.Device); found {
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
		if device, found := pairedDevice.(*makr75W.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*k57rgbW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*k70pmW.Device); found {
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
		if device, found := pairedDevice.(*m75AirW.Device); found {
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
		if device, found := pairedDevice.(*m65rgbultraW.Device); found {
			if device.Connected {
				device.SetConnected(false)
			}
		}
		if device, found := pairedDevice.(*sabrergbproW.Device); found {
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
				if device, found := pairedDevice.(*makr75W.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
				if device, found := pairedDevice.(*k70pmW.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
				if device, found := pairedDevice.(*k70coretklW.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
				if device, found := pairedDevice.(*k57rgbW.Device); found {
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
				if device, found := pairedDevice.(*scimitarSEW.Device); found {
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
				if device, found := pairedDevice.(*m75AirW.Device); found {
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
				if device, found := pairedDevice.(*m65rgbultraW.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
				if device, found := pairedDevice.(*sabrergbproW.Device); found {
					if device.Connected {
						device.SetConnected(false)
					}
				}
			}
			break
		}
	}
}

// setDeviceOnline will set device online
func (d *Device) setDeviceOnline(deviceType int) {
	time.Sleep(time.Duration(connectDelay) * time.Millisecond)
	for _, pairedDevice := range d.PairedDevices {
		switch deviceType {
		case 0:
			{
				// Keyboards
				if device, found := pairedDevice.(*k100airW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*makr75W.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*k70pmW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*k70coretklW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*k57rgbW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
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
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*scimitarW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*scimitarSEW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*nightsabreW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*ironclawW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*darkcorergbproseW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*darkcorergbproW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*m75W.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*m75AirW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*harpoonW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*darkstarW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*m65rgbultraW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*sabrergbproW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
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
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*makr75W.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*k70pmW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*k57rgbW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*m55W.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*scimitarW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*scimitarSEW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*nightsabreW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*ironclawW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*darkcorergbproseW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*darkcorergbproW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*m75W.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*m75AirW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*harpoonW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*darkstarW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*k70coretklW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*m65rgbultraW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
				if device, found := pairedDevice.(*sabrergbproW.Device); found {
					if !device.Connected {
						device.Connect()
						d.SharedDevices(d.DeviceList[device.Serial])
					}
				}
			}
			break
		}
	}
}

// setDeviceStatus will set device status
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
								logger.Log(logger.Fields{"error": e}).Error("Unable to read paired device endpoint")
							}
							continue
						}

						batteryLevel, e := d.transfer(value.Endpoint, cmdBatteryLevel, nil)
						if e != nil {
							if d.Debug {
								logger.Log(logger.Fields{"error": e}).Error("Unable to read paired device endpoint")
							}
							continue
						}
						val := binary.LittleEndian.Uint16(batteryLevel[3:5])
						if val > 0 {
							d.setDeviceBatteryLevelByProductId(value.ProductId, val/10)
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
											if d.Debug {
												logger.Log(logger.Fields{"method": methodName}).Warn("Method not found or method is not supported for this device type")
											}
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

// backendListener will listen for events from the device
func (d *Device) backendListener() {
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
									dev.TriggerKeyAssignment(data)
								}
								if dev, found := value.(*makr75W.Device); found {
									dev.TriggerKeyAssignment(data)
								}
								if dev, found := value.(*k57rgbW.Device); found {
									dev.TriggerKeyAssignment(data)
								}
								if dev, found := value.(*k70pmW.Device); found {
									dev.TriggerKeyAssignment(data)
								}
								if dev, found := value.(*m55W.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(data[2])
									}
								}
								if dev, found := value.(*m75W.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(data[2])
									}
								}
								if dev, found := value.(*m75AirW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(data[2])
									}
								}
								if dev, found := value.(*scimitarW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]))
									}
								}
								if dev, found := value.(*scimitarSEW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]))
									}
								}
								if dev, found := value.(*nightsabreW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]))
									}
								}
								if dev, found := value.(*darkcorergbproseW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]))
									}
								}
								if dev, found := value.(*darkcorergbproW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]))
									}
								}
								if dev, found := value.(*harpoonW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(data[2])
									}
								}
								if dev, found := value.(*ironclawW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]))
									}
								}
								if dev, found := value.(*darkstarW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]))
									}
								}
								if dev, found := value.(*m65rgbultraW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]))
									}
								}
								if dev, found := value.(*sabrergbproW.Device); found {
									if data[1] == 0x02 {
										dev.TriggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]))
									}
								}
								if dev, found := value.(*k70coretklW.Device); found {
									dev.TriggerKeyAssignment(data)
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
