package slipstream

// Package: Corsair Slipstream
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices/darkcorergbproW"
	"OpenLinkHub/src/devices/darkcorergbproseW"
	"OpenLinkHub/src/devices/darkstarW"
	"OpenLinkHub/src/devices/harpoonW"
	"OpenLinkHub/src/devices/ironclawSEW"
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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sstallion/go-hid"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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
	DeviceProfile  *DeviceProfile
	SingleDevice   bool
	Template       string
	Debug          bool
	Exit           bool
	timerKeepAlive *time.Ticker
	timerSleep     *time.Ticker
	keepAliveChan  chan struct{}
	sleepChan      chan struct{}
	instance       *common.Device
	PollingRates   map[int]string
}

type DeviceProfile struct {
	Active      bool
	Path        string
	Product     string
	Serial      string
	PollingRate int
	RgbOff      bool
}

var (
	pwd                = ""
	bufferSize         = 64
	bufferSizeWrite    = bufferSize + 1
	headerSize         = 2
	deviceKeepAlive    = 10000
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
	cmdBatteryLevel    = []byte{0x02, 0x0f}
	cmdSetPollingRate  = []byte{0x01, 0x01, 0x00}
	cmdCommand         = byte(0x08)
	transferTimeout    = 100
	connectDelay       = 5000
	pollingRateDevices = []uint16{7132, 11008}
)

func Init(vendorId, productId uint16, _, path string, callback func(device *common.Device)) *common.Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

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
		PollingRates: map[int]string{
			0: "Not Set",
			1: "125 Hz / 8 msec",
			2: "250 Hz / 4 msec",
			3: "500 Hz / 2 msec",
			4: "1000 Hz / 1 msec",
		},
	}

	d.getDebugMode()         // Debug
	d.getManufacturer()      // Manufacturer
	d.getProduct()           // Product
	d.getSerial()            // Serial
	d.loadDeviceProfile()    // Load device profile
	d.saveDeviceProfile()    // Save profile
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
					d.slipstream,
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
					DeviceType:  common.DeviceTypeMouse,
					ProductId:   value.ProductId,
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
					d.slipstream,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeScimitarRgbEliteW,
					Product:     "SCIMITAR ELITE",
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
		case 11042: // CORSAIR SCIMITAR ELITE WIRELESS SE
			{
				dev := scimitarSEW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.slipstream,
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
					DeviceType:  common.DeviceTypeMouse,
					ProductId:   value.ProductId,
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
					d.slipstream,
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
					DeviceType:  common.DeviceTypeMouse,
					ProductId:   value.ProductId,
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
					d.slipstream,
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
					DeviceType:  common.DeviceTypeKeyboard,
					ProductId:   value.ProductId,
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
		case 11058: // IRONCLAW SE WIRELESS
			{
				dev := ironclawSEW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.slipstream,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeIronClawSEW,
					Product:     "IRONCLAW SE",
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
		case 7038: // DARK CORE RGB PRO SE WIRELESS
			{
				dev := darkcorergbproseW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.slipstream,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeDarkCoreRgbProSEW,
					Product:     "DARK CORE PRO SE",
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
		case 7040: // DARK CORE RGB PRO WIRELESS
			{
				dev := darkcorergbproW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.slipstream,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeDarkCoreRgbProW,
					Product:     "DARK CORE PRO",
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
		case 7153: // M75 WIRELESS
			{
				dev := m75W.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.slipstream,
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
					DeviceType:  common.DeviceTypeMouse,
					ProductId:   value.ProductId,
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
					d.slipstream,
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
					DeviceType:  common.DeviceTypeMouse,
					ProductId:   value.ProductId,
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
					d.slipstream,
					value.Endpoint,
					value.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeM75AirW,
					Product:     "M75 AIR",
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
		case 7006: // HARPOON RGB WIRELESS
			{
				dev := harpoonW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.slipstream,
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
					DeviceType:  common.DeviceTypeMouse,
					ProductId:   value.ProductId,
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
					d.slipstream,
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
					DeviceType:  common.DeviceTypeMouse,
					ProductId:   value.ProductId,
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
					d.slipstream,
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
					DeviceType:  common.DeviceTypeMouse,
					ProductId:   value.ProductId,
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
					d.slipstream,
					value.Endpoint,
					value.Serial,
					d.Serial,
				)

				object := &common.Device{
					ProductType: common.ProductTypeK70CoreTklW,
					Product:     "K70 CORE RGB TKL",
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
		case 7064: // CORSAIR SABRE RGB PRO WIRELESS Gaming Mouse
			{
				dev := sabrergbproW.Init(
					value.VendorId,
					d.ProductId,
					value.ProductId,
					d.slipstream,
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
					DeviceType:  common.DeviceTypeMouse,
					ProductId:   value.ProductId,
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
					d.slipstream,
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
					DeviceType:  common.DeviceTypeKeyboard,
					ProductId:   value.ProductId,
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
					d.slipstream,
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
					DeviceType:  common.DeviceTypeKeyboard,
					ProductId:   value.ProductId,
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
					d.slipstream,
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

// loadDeviceProfiles will load custom user profiles
func (d *Device) loadDeviceProfile() {
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"
	pf := &DeviceProfile{}

	// Check if filename has .json extension
	if !common.IsValidExtension(profilePath, ".json") {
		return
	}

	file, err := os.Open(profilePath)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": profilePath}).Warn("Unable to load profile")
		return
	}
	if err = json.NewDecoder(file).Decode(pf); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": profilePath}).Warn("Unable to decode profile")
		return
	}
	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"location": profilePath, "serial": d.Serial}).Warn("Failed to close file handle")
	}
	d.DeviceProfile = pf
}

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"
	deviceProfile := &DeviceProfile{
		Product: d.Product,
		Serial:  d.Serial,
		Path:    profilePath,
	}

	if d.DeviceProfile == nil {
		deviceProfile.Active = true
		deviceProfile.PollingRate = 4
	} else {
		if d.DeviceProfile.PollingRate == 0 {
			deviceProfile.PollingRate = 4
		} else {
			deviceProfile.PollingRate = d.DeviceProfile.PollingRate
		}
		deviceProfile.Active = d.DeviceProfile.Active
		if len(d.DeviceProfile.Path) < 1 {
			deviceProfile.Path = profilePath
			d.DeviceProfile.Path = profilePath
		} else {
			deviceProfile.Path = d.DeviceProfile.Path
		}
		deviceProfile.RgbOff = d.DeviceProfile.RgbOff
	}

	// Fix profile paths if folder database/ folder is moved
	filename := filepath.Base(deviceProfile.Path)
	path := fmt.Sprintf("%s/database/profiles/%s", pwd, filename)
	if deviceProfile.Path != path {
		logger.Log(logger.Fields{"original": deviceProfile.Path, "new": path}).Warn("Detected mismatching device profile path. Fixing paths...")
		deviceProfile.Path = path
	}

	// Save profile
	if err := common.SaveJsonData(deviceProfile.Path, deviceProfile); err != nil {
		logger.Log(logger.Fields{"error": err, "location": deviceProfile.Path}).Error("Unable to write device profile data")
		return
	}
	d.loadDeviceProfile()
}

// UpdatePollingRate will set device polling rate
func (d *Device) UpdatePollingRate(pullingRate int) uint8 {
	if !slices.Contains(pollingRateDevices, d.ProductId) {
		logger.Log(logger.Fields{}).Error("Unsupported device for polling rate change")
		return 0
	}

	if _, ok := d.PollingRates[pullingRate]; ok {
		if d.DeviceProfile == nil {
			return 0
		}
		d.Exit = true
		time.Sleep(40 * time.Millisecond)

		d.DeviceProfile.PollingRate = pullingRate
		d.saveDeviceProfile()
		buf := make([]byte, 1)
		buf[0] = byte(pullingRate)
		_, err := d.transfer(cmdCommand, cmdSetPollingRate, buf, false)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to set mouse polling rate")
			return 0
		}
		return 1
	}
	return 0
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

// getDeviceFirmware will return a firmware version out as string
func (d *Device) getDeviceFirmware() {
	fw, err := d.transfer(cmdCommand, cmdGetFirmware, nil, true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
	}

	v1, v2, v3 := int(fw[3]), int(fw[4]), int(binary.LittleEndian.Uint16(fw[5:7]))
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

// setHardwareMode will switch a device to hardware mode
func (d *Device) setHardwareMode() {
	_, err := d.transfer(cmdCommand, cmdHardwareMode, nil, true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	_, err := d.transfer(cmdCommand, cmdSoftwareMode, nil, true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
}

func (d *Device) readNext() []byte {
	buffer, err := d.transfer(cmdCommand, cmdRead, nil, true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read endpoint")
	}
	return buffer
}

// read will read data from a device and return data as a byte array
func (d *Device) read(endpoint []byte) []byte {
	var buffer []byte

	_, err := d.transfer(cmdCommand, cmdCloseEndpoint, endpoint, true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close endpoint")
	}

	_, err = d.transfer(cmdCommand, cmdOpenEndpoint, endpoint, true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to open endpoint")
	}

	_, err = d.transfer(cmdCommand, cmdWrite, endpoint, true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to open endpoint")
	}

	buffer, err = d.transfer(cmdCommand, cmdRead, endpoint, true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read endpoint")
	}

	for i := 1; i < int(buffer[5]); i++ {
		next, e := d.transfer(cmdCommand, cmdRead, endpoint, true)
		if e != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to read endpoint")
		}
		buffer = append(buffer, next[3:]...)
	}

	_, err = d.transfer(cmdCommand, cmdCloseEndpoint, nil, true)
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
						//
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
					_, err := d.transfer(cmdCommand, cmdHeartbeat, nil, true)
					if err != nil {
						logger.Log(logger.Fields{"error": err}).Error("Unable to read slipstream endpoint")
					}
					for _, value := range d.Devices {
						if d.slipstream.Connected[value.ProductId] {
							_, e := d.transfer(value.Endpoint, cmdHeartbeat, nil, true)
							if e != nil {
								if d.Debug {
									logger.Log(logger.Fields{"error": e}).Error("Unable to read paired device endpoint")
								}
								continue
							}

							batteryLevel, e := d.transfer(value.Endpoint, cmdBatteryLevel, nil, true)
							if e != nil {
								if d.Debug {
									logger.Log(logger.Fields{"error": e}).Error("Unable to read paired device endpoint for battery status")
								}
								continue
							}
							val := binary.LittleEndian.Uint16(batteryLevel[3:5])
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
							msg, err := d.transfer(value.Endpoint, cmdInactivity, nil, true)
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

				if data[1] == 0x01 && data[2] == 0x36 {
					value := data[4]
					productId := uint16(data[11])<<8 | uint16(data[10])
					d.setDeviceStatus(value, productId)
				} else {
					if data[1] == 0x02 || data[1] == 0x09 || data[1] == 0x05 {
						switch data[0] {
						case 1:
							d.TriggerMousedKeyAssignment(data)
							d.TriggerKeyboardKeyAssignment(data)
						case 2:
							d.TriggerMousedKeyAssignment(data)
						case 3:
							d.TriggerKeyboardKeyAssignment(data)
						}
					}
				}
			}
		}
	}()
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(command byte, endpoint, buffer []byte, read bool) ([]byte, error) {
	if d.slipstream == nil || d.slipstream.Dev == nil {
		return nil, errors.New("slipstream device is not initialized")
	}

	d.slipstream.Mutex.Lock()
	defer d.slipstream.Mutex.Unlock()

	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = command
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

	if read {
		if _, e := d.slipstream.Dev.ReadWithTimeout(bufferR, time.Duration(transferTimeout)*time.Millisecond); e != nil {
			if d.Debug {
				logger.Log(logger.Fields{"error": e, "serial": d.Serial}).Error("Unable to read data from device")
			}
			return bufferR, e
		}
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
	bufferW[1] = command
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
