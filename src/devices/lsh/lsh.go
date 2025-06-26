package lsh

// Package: iCUE Link System Hub
// This is the primary package for iCUE Link System Hub.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices/lcd"
	"OpenLinkHub/src/led"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/stats"
	"OpenLinkHub/src/systeminfo"
	"OpenLinkHub/src/temperatures"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"math"
	"math/rand"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DeviceProfile struct {
	Active             bool
	Path               string
	Product            string
	Serial             string
	LCDMode            uint8
	LCDRotation        uint8
	Brightness         uint8
	BrightnessSlider   *uint8
	OriginalBrightness uint8
	SpeedProfiles      map[int]string
	RGBProfiles        map[int]string
	Labels             map[int]string
	DevicePosition     map[int]string
	ExternalAdapter    map[int]int
	LCDModes           map[int]uint8
	LCDImages          map[int]string
	LCDRotations       map[int]uint8
	LCDDevices         map[int]string
	MultiRGB           string
	MultiProfile       string
}

type LCD struct {
	Lcd       *hid.Device
	ProductId uint16
}

type TemperatureProbe struct {
	ChannelId int
	Name      string
	Label     string
	Serial    string
	Product   string
}

// SupportedDevice contains definition of supported devices
type SupportedDevice struct {
	DeviceId         byte   `json:"deviceId"`
	Model            byte   `json:"deviceModel"`
	Name             string `json:"deviceName"`
	LedChannels      uint8  `json:"ledChannels"`
	ContainsPump     bool   `json:"containsPump"`
	Desc             string `json:"desc"`
	AIO              bool   `json:"aio"`
	TemperatureProbe bool   `json:"temperatureProbe"`
	LinkAdapter      bool   `json:"linkAdapter"`
	CpuBlock         bool   `json:"cpuBlock"`
	HasSpeed         bool   `json:"hasSpeed"`
}

// Devices contain information about devices connected to an iCUE Link
type Devices struct {
	ChannelId          int             `json:"channelId"`
	Type               byte            `json:"type"`
	Model              byte            `json:"-"`
	DeviceId           string          `json:"deviceId"`
	Name               string          `json:"name"`
	DefaultValue       byte            `json:"-"`
	Rpm                int16           `json:"rpm"`
	Temperature        float32         `json:"temperature"`
	TemperatureString  string          `json:"temperatureString"`
	LedChannels        uint8           `json:"-"`
	ContainsPump       bool            `json:"-"`
	Description        string          `json:"description"`
	HubId              string          `json:"-"`
	PumpModes          map[byte]string `json:"-"`
	Profile            string          `json:"profile"`
	RGB                string          `json:"rgb"`
	Label              string          `json:"label"`
	PortId             uint8           `json:"portId"`
	IsTemperatureProbe bool
	IsLinkAdapter      bool
	IsCpuBlock         bool
	HasSpeed           bool
	HasTemps           bool
	AIO                bool
	Position           int
	ExternalAdapter    int
	LCDSerial          string
}

type Device struct {
	Debug             bool
	dev               *hid.Device
	Manufacturer      string                    `json:"manufacturer"`
	Product           string                    `json:"product"`
	Serial            string                    `json:"serial"`
	Firmware          string                    `json:"firmware"`
	AIO               bool                      `json:"aio"`
	Devices           map[int]*Devices          `json:"devices"`
	UserProfiles      map[string]*DeviceProfile `json:"userProfiles"`
	DeviceProfile     *DeviceProfile
	OriginalProfile   *DeviceProfile
	TemperatureProbes *[]TemperatureProbe
	activeRgb         *rgb.ActiveRGB
	ledProfile        *led.Device
	Template          string
	HasLCD            bool
	VendorId          uint16
	ProductId         uint16
	LCDModes          map[int]string
	LCDRotations      map[int]string
	Brightness        map[int]string
	RGBStrips         map[int]string
	PortProtection    map[uint8]int
	GlobalBrightness  float64
	IsCritical        bool
	FirmwareInternal  []int
	CpuTemp           float32
	GpuTemp           float32
	XD5LCDs           int
	Rgb               *rgb.RGB
	Exit              bool
	LCDImage          map[int]*lcd.ImageData
	lcdRefreshChan    chan struct{}
	lcdImageChan      chan struct{}
	autoRefreshChan   chan struct{}
	speedRefreshChan  chan struct{}
	timer             *time.Ticker
	timerSpeed        *time.Ticker
	lcdTimer          *time.Ticker
	mutex             sync.Mutex
	mutexLcd          sync.Mutex
	lcdDevices        map[string]*LCD
}

var (
	pwd                         = ""
	cmdOpenEndpoint             = []byte{0x0d, 0x01}
	cmdOpenColorEndpoint        = []byte{0x0d, 0x00}
	cmdCloseEndpoint            = []byte{0x05, 0x01, 0x01}
	cmdGetFirmware              = []byte{0x02, 0x13}
	cmdSoftwareMode             = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode             = []byte{0x01, 0x03, 0x00, 0x01}
	cmdWrite                    = []byte{0x06, 0x01}
	cmdWriteColor               = []byte{0x06, 0x00}
	cmdRead                     = []byte{0x08, 0x01}
	modeGetDevices              = []byte{0x36}
	modeGetTemperatures         = []byte{0x21}
	modeGetSpeeds               = []byte{0x17}
	modeSetSpeed                = []byte{0x18}
	modeSetColor                = []byte{0x22}
	dataTypeGetDevices          = []byte{0x21, 0x00}
	dataTypeGetTemperatures     = []byte{0x10, 0x00}
	dataTypeGetSpeeds           = []byte{0x25, 0x00}
	dataTypeSetSpeed            = []byte{0x07, 0x00}
	dataTypeSetColor            = []byte{0x12, 0x00}
	dataTypeSubColor            = []byte{0x07, 0x00}
	bufferSize                  = 512
	headerSize                  = 3
	headerWriteSize             = 4
	bufferSizeWrite             = bufferSize + 1
	transferTimeout             = 500
	maxBufferSizePerRequest     = 508
	defaultSpeedValue           = 70
	temperaturePullingInterval  = 3000
	lcdRefreshInterval          = 1000
	deviceRefreshInterval       = 1000
	lcdLedChannels              = 24
	lcdHeaderSize               = 8
	lcdBufferSize               = 1024
	maxLCDBufferSizePerRequest  = lcdBufferSize - lcdHeaderSize
	portProtectionMaximumStage1 = 238
	portProtectionMaximumStage2 = 340
	portProtectionMaximumStage3 = 442
	criticalAioCoolantTemp      = 57.0
	zeroRpmLimit                = 40
	i2cPrefix                   = "i2c"
	rgbProfileUpgrade           = []string{"custom"}
	supportedDevices            = []SupportedDevice{
		{DeviceId: 1, Model: 0, Name: "iCUE LINK QX RGB", LedChannels: 34, ContainsPump: false, Desc: "Fan", TemperatureProbe: true, HasSpeed: true},
		{DeviceId: 2, Model: 0, Name: "iCUE LINK LX RGB", LedChannels: 18, ContainsPump: false, Desc: "Fan", HasSpeed: true},
		{DeviceId: 3, Model: 0, Name: "iCUE LINK RX RGB MAX", LedChannels: 8, ContainsPump: false, Desc: "Fan", HasSpeed: true},
		{DeviceId: 19, Model: 0, Name: "iCUE LINK RX", LedChannels: 0, ContainsPump: false, Desc: "Fan", HasSpeed: true},
		{DeviceId: 15, Model: 0, Name: "iCUE LINK RX RGB", LedChannels: 8, ContainsPump: false, Desc: "Fan", HasSpeed: true},
		{DeviceId: 4, Model: 0, Name: "iCUE LINK RX MAX", LedChannels: 8, ContainsPump: false, Desc: "Fan", HasSpeed: true},
		{DeviceId: 7, Model: 2, Name: "iCUE LINK H150i", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true, HasSpeed: true},
		{DeviceId: 7, Model: 5, Name: "iCUE LINK H150i", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true, HasSpeed: true},
		{DeviceId: 7, Model: 1, Name: "iCUE LINK H115i", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true, HasSpeed: true},
		{DeviceId: 7, Model: 3, Name: "iCUE LINK H170i", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true, HasSpeed: true},
		{DeviceId: 7, Model: 0, Name: "iCUE LINK H100i", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true, HasSpeed: true},
		{DeviceId: 7, Model: 4, Name: "iCUE LINK H100i", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true, HasSpeed: true},
		{DeviceId: 9, Model: 0, Name: "iCUE LINK XC7 ELITE", LedChannels: 24, ContainsPump: false, Desc: "CPU Block", TemperatureProbe: true},
		{DeviceId: 9, Model: 1, Name: "iCUE LINK XC7 Elite", LedChannels: 24, ContainsPump: false, Desc: "CPU Block", TemperatureProbe: true},
		{DeviceId: 10, Model: 0, Name: "iCUE LINK XG3 HYBRID", LedChannels: 22, ContainsPump: false, Desc: "GPU Block", HasSpeed: true},
		{DeviceId: 13, Model: 0, Name: "iCUE LINK XG7 RGB", LedChannels: 16, ContainsPump: false, Desc: "GPU Hybrid Block"},
		{DeviceId: 12, Model: 0, Name: "iCUE LINK XD5 ELITE", LedChannels: 22, ContainsPump: true, Desc: "Pump/Res", HasSpeed: true},
		{DeviceId: 25, Model: 0, Name: "iCUE LINK XD6 ELITE", LedChannels: 22, ContainsPump: true, Desc: "Pump/Res", HasSpeed: true},
		{DeviceId: 16, Model: 0, Name: "VRM Cooler Module", LedChannels: 0, ContainsPump: false, Desc: "Fan", HasSpeed: true},
		{DeviceId: 17, Model: 0, Name: "iCUE LINK TITAN 240", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true, HasSpeed: true},
		{DeviceId: 17, Model: 4, Name: "iCUE LINK TITAN 240", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true, HasSpeed: true},
		{DeviceId: 17, Model: 2, Name: "iCUE LINK TITAN 360", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true, HasSpeed: true},
		{DeviceId: 17, Model: 5, Name: "iCUE LINK TITAN 360", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true, HasSpeed: true},
		{DeviceId: 17, Model: 1, Name: "iCUE LINK TITAN 280", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true, HasSpeed: true},
		{DeviceId: 17, Model: 3, Name: "iCUE LINK TITAN 420", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true, HasSpeed: true},
		{DeviceId: 5, Model: 0, Name: "iCUE LINK ADAPTER", LedChannels: 0, ContainsPump: false, Desc: "Adapter", LinkAdapter: true},
	}
)

// Init will initialize a new device
func Init(vendorId, productId uint16, serial string) *Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	// Open device, return if failure
	dev, err := hid.Open(vendorId, productId, serial)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId, "serial": serial}).Error("Unable to open HID device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:       dev,
		Template:  "lsh.html",
		VendorId:  vendorId,
		ProductId: productId,
		LCDModes: map[int]string{
			0:   "Liquid Temperature",
			1:   "Pump Speed",
			2:   "CPU Temperature",
			3:   "GPU Temperature",
			4:   "Combined",
			6:   "CPU / GPU Temp",
			7:   "CPU / GPU Load",
			8:   "CPU / GPU Load/Temp",
			9:   "Time",
			10:  "Image / GIF",
			100: "Arc",
			101: "Double Arc",
			102: "Animation",
		},
		LCDRotations: map[int]string{
			0: "default",
			1: "90 degrees",
			2: "180 degrees",
			3: "270 degrees",
		},
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
		RGBStrips: map[int]string{
			0: "None",
			1: "1x LS350 Aurora",
			2: "2x LS350 Aurora",
			3: "1x LS430 Aurora",
			4: "2x LS430 Aurora",
		},
		PortProtection:   make(map[uint8]int, 2),
		lcdRefreshChan:   make(chan struct{}),
		lcdImageChan:     make(chan struct{}),
		autoRefreshChan:  make(chan struct{}),
		speedRefreshChan: make(chan struct{}),
		timer:            &time.Ticker{},
		timerSpeed:       &time.Ticker{},
		lcdTimer:         &time.Ticker{},
		lcdDevices:       make(map[string]*LCD, lcd.GetLcdAmount()),
	}

	// Bootstrap
	d.getDebugMode()        // Debug mode
	d.getManufacturer()     // Manufacturer
	d.getProduct()          // Product
	d.getSerial()           // Serial
	d.loadRgb()             // Load RGB
	d.loadDeviceProfiles()  // Load all device profiles
	d.getDeviceLcd()        // Check for LCDs
	d.getDeviceFirmware()   // Firmware
	d.setSoftwareMode()     // Activate software mode
	d.getDevices()          // Get devices connected to a hub
	d.setColorEndpoint()    // Set device color endpoint
	d.setDeviceProtection() // Protect device
	d.setDefaults()         // Set default speed and color values for fans and pumps
	d.setAutoRefresh()      // Set auto device refresh
	d.saveDeviceProfile()   // Save profile
	d.setupLedProfile()     // LED profile
	d.getTemperatureProbe() // Devices with temperature probes
	if config.GetConfig().Manual {
		fmt.Println(
			fmt.Sprintf("[%s [%s]] Manual flag enabled. Process will not monitor temperature or adjust fan speed.", d.Serial, d.Product),
		)
	} else {
		d.updateDeviceSpeed() // Update device speed
	}
	d.setTemperatures() // Get initial temps
	d.setDeviceColor()  // Device color
	if d.HasLCD {
		d.getLcdImages()
		d.setLcdRotation() // LCD rotation
		d.setupLCD()       // LCD
		d.setupLCDImage()  // LCD images
	}

	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")
	return d
}

// GetDeviceLedData will return led profiles as interface
func (d *Device) GetDeviceLedData() interface{} {
	return d.ledProfile
}

// getLedProfileColor will get RGB color based on channelId and ledId
func (d *Device) getLedProfileColor(channelId int, ledId int) *rgb.Color {
	if channels, ok := d.ledProfile.Devices[channelId]; ok {
		if color, found := channels.Channels[ledId]; found {
			return &color
		}
	}
	return nil
}

// setupLedProfile will init and load LED profile
func (d *Device) setupLedProfile() {
	d.ledProfile = led.LoadProfile(d.Serial)
	if d.ledProfile == nil {
		d.saveLedProfile()                       // Save profile
		d.ledProfile = led.LoadProfile(d.Serial) // Reload
	}

	profileLength := len(d.ledProfile.Devices)
	actualLength := len(d.Devices)
	if profileLength != actualLength {
		logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product, "profile": profileLength, "actual": actualLength}).Info("Device amount changed. LED profile will be re-created.")
		d.saveLedProfile()                       // Save profile
		d.ledProfile = led.LoadProfile(d.Serial) // Reload
	}
}

// saveLedProfile will save new LED profile
func (d *Device) saveLedProfile() {
	// Default profile
	profile := d.GetRgbProfile("static")
	if profile == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Error("Unable to load static rgb profile")
		return
	}

	// Init
	lightChannels := 0
	keys := make([]int, 0)
	for k := range d.Devices {
		lightChannels += int(d.Devices[k].LedChannels)
		keys = append(keys, k)
	}
	sort.Ints(keys)

	device := led.Device{
		Serial:     d.Serial,
		DeviceName: d.Product,
	}

	devices := map[int]led.DeviceData{}

	for _, k := range keys {
		channels := map[int]rgb.Color{}
		deviceData := led.DeviceData{}
		deviceData.LedChannels = d.Devices[k].LedChannels
		deviceData.Pump = d.Devices[k].ContainsPump
		deviceData.AIO = d.Devices[k].AIO
		deviceData.Fan = d.Devices[k].HasSpeed && d.Devices[k].ContainsPump == false

		if d.HasLCD && d.Devices[k].AIO {
			for i := 0; i < int(d.Devices[k].LedChannels); i++ {
				channels[i] = rgb.Color{
					Red:   0,
					Green: 255,
					Blue:  255,
					Hex:   fmt.Sprintf("#%02x%02x%02x", 0, 255, 255),
				}
				if i > 15 && i < 20 {
					channels[i] = rgb.Color{
						Red:   0,
						Green: 0,
						Blue:  0,
						Hex:   fmt.Sprintf("#%02x%02x%02x", 0, 255, 255),
					}
				}
			}
		} else {
			for i := 0; i < int(d.Devices[k].LedChannels); i++ {
				channels[i] = rgb.Color{
					Red:   0,
					Green: 255,
					Blue:  255,
					Hex:   fmt.Sprintf("#%02x%02x%02x", 0, 255, 255),
				}
			}
		}
		deviceData.Channels = channels
		devices[k] = deviceData
	}
	device.Devices = devices
	led.SaveProfile(d.Serial, device)
}

// setDeviceProtection will reduce LED brightness if you are running too many devices per hub physical port.
// Reduction is applied globally, not per physical port
func (d *Device) setDeviceProtection() {
	for _, portLedChannels := range d.PortProtection {

		// > 7 QX fans
		if portLedChannels > portProtectionMaximumStage1 {
			d.GlobalBrightness = 0.66
		}

		// > 10 QX fans
		if portLedChannels > portProtectionMaximumStage2 {
			d.GlobalBrightness = 0.33
		}

		// > 13 QX fans
		if portLedChannels > portProtectionMaximumStage3 {
			d.GlobalBrightness = 0.1
		}
	}

	if d.GlobalBrightness != 0 {
		logger.Log(logger.Fields{"serial": d.Serial}).Warnf("Your device brightness has been reduced by %2.f percent due to port power draw", (1-d.GlobalBrightness)*100)
	}
}

// GetRgbProfiles will return RGB profiles for a target device
func (d *Device) GetRgbProfiles() interface{} {
	return d.Rgb
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}
	d.timer.Stop()
	close(d.autoRefreshChan)

	if d.HasLCD {
		close(d.lcdRefreshChan)
		close(d.lcdImageChan)
		d.lcdTimer.Stop()
	}

	if !config.GetConfig().Manual {
		d.timerSpeed.Stop()
		close(d.speedRefreshChan)
	}

	for _, lcdHidDevice := range d.lcdDevices {
		if lcdHidDevice.Lcd != nil {
			lcdReports := map[int][]byte{0: {0x03, 0x1e, 0x01, 0x01}, 1: {0x03, 0x1d, 0x00, 0x01}}
			for i := 0; i <= 1; i++ {
				_, e := lcdHidDevice.Lcd.SendFeatureReport(lcdReports[i])
				if e != nil {
					logger.Log(logger.Fields{"error": e}).Error("Unable to send report to LCD HID device")
				}
			}
			err := lcdHidDevice.Lcd.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to close LCD HID device")
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

// StopDirty will stop device in a dirty way
func (d *Device) StopDirty() uint8 {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device (dirty)...")
	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}
	d.timer.Stop()
	close(d.autoRefreshChan)

	if d.HasLCD {
		close(d.lcdRefreshChan)
		close(d.lcdImageChan)
		d.lcdTimer.Stop()
	}

	if !config.GetConfig().Manual {
		d.timerSpeed.Stop()
		close(d.speedRefreshChan)
	}

	for _, lcdHidDevice := range d.lcdDevices {
		if lcdHidDevice.Lcd != nil {
			lcdReports := map[int][]byte{0: {0x03, 0x1e, 0x01, 0x01}, 1: {0x03, 0x1d, 0x00, 0x01}}
			for i := 0; i <= 1; i++ {
				_, e := lcdHidDevice.Lcd.SendFeatureReport(lcdReports[i])
				if e != nil {
					logger.Log(logger.Fields{"error": e}).Error("Unable to send report to LCD HID device")
				}
			}
			err := lcdHidDevice.Lcd.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to close LCD HID device")
			}
		}
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 1
}

// getLcdImages will preload lcd images for LCD devices
func (d *Device) getLcdImages() {
	for key, device := range d.Devices {
		if len(device.LCDSerial) > 0 && (device.AIO || device.ContainsPump) {
			if lcdMode, ok := d.DeviceProfile.LCDModes[device.ChannelId]; ok {
				if lcdMode == lcd.DisplayImage {
					if image, ok := d.DeviceProfile.LCDImages[device.ChannelId]; ok {
						lcdImage := lcd.GetLcdImage(image)
						if lcdImage == nil {
							if len(lcd.GetLcdImages()) > 0 {
								lcdImage = lcd.GetLcdImage(lcd.GetLcdImages()[0].Name)
							}
						}
						if lcdImage == nil {
							// Complete failure
							d.DeviceProfile.LCDModes[device.ChannelId] = 0
							d.DeviceProfile.LCDImages[device.ChannelId] = ""
							d.saveDeviceProfile()
						} else {
							d.DeviceProfile.LCDImages[device.ChannelId] = lcdImage.Name
							d.LCDImage[key] = lcdImage
							d.saveDeviceProfile()
						}
					}
				}
			}
		}
	}
}

// loadRgb will load RGB file if found, or create the default.
func (d *Device) loadRgb() {
	rgbDirectory := pwd + "/database/rgb/"
	rgbFilename := rgbDirectory + d.Serial + ".json"

	// Check if filename has .json extension
	if !common.IsValidExtension(rgbFilename, ".json") {
		return
	}

	if !common.FileExists(rgbFilename) {
		profile := rgb.GetRGB()
		profile.Device = d.Product

		// Convert to JSON
		buffer, err := json.MarshalIndent(profile, "", "    ")
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": rgbFilename}).Warn("Unable to encode RGB json")
			return
		}

		// Create profile filename
		file, err := os.Create(rgbFilename)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": rgbFilename}).Warn("Unable to create RGB json file")
			return
		}

		// Write JSON buffer to file
		_, err = file.Write(buffer)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": rgbFilename}).Warn("Unable to write to RGB json file")
			return
		}

		// Close file
		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": rgbFilename}).Warn("Unable to close RGB json file")
			return
		}
	}

	file, err := os.Open(rgbFilename)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": rgbFilename}).Warn("Unable to load RGB")
		return
	}
	if err = json.NewDecoder(file).Decode(&d.Rgb); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": rgbFilename}).Warn("Unable to decode profile")
		return
	}

	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"location": rgbFilename, "serial": d.Serial}).Warn("Failed to close file handle")
	}

	d.upgradeRgbProfile(rgbFilename, rgbProfileUpgrade)
}

// upgradeRgbProfile will upgrade current rgb profile list
func (d *Device) upgradeRgbProfile(path string, profiles []string) {
	save := false
	for _, profile := range profiles {
		pf := d.GetRgbProfile(profile)
		if pf == nil {
			save = true
			logger.Log(logger.Fields{"profile": profile}).Info("Upgrading RGB profile")
			d.Rgb.Profiles[profile] = rgb.Profile{}
		}
	}

	if save {
		buffer, err := json.MarshalIndent(d.Rgb, "", "    ")
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
			return
		}

		f, err := os.Create(path)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": path}).Error("Unable to save rgb profile")
			return
		}

		_, err = f.Write(buffer)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": path}).Error("Unable to write data")
		}
	}
}

// GetRgbProfile will return rgb.Profile struct
func (d *Device) GetRgbProfile(profile string) *rgb.Profile {
	if d.Rgb == nil {
		return nil
	}

	if val, ok := d.Rgb.Profiles[profile]; ok {
		return &val
	}
	return nil
}

// GetDeviceTemplate will return device template name
func (d *Device) GetDeviceTemplate() string {
	return d.Template
}

// loadDeviceProfiles will load custom user profiles
func (d *Device) loadDeviceProfiles() {
	profileList := make(map[string]*DeviceProfile)
	userProfileDirectory := pwd + "/database/profiles/"

	files, err := os.ReadDir(userProfileDirectory)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": userProfileDirectory, "serial": d.Serial}).Fatal("Unable to read content of a folder")
	}

	for _, fi := range files {
		pf := &DeviceProfile{}
		if fi.IsDir() {
			continue // Exclude folders if any
		}

		// Define a full path of filename
		profileLocation := userProfileDirectory + fi.Name()

		// Check if filename has .json extension
		if !common.IsValidExtension(profileLocation, ".json") {
			continue
		}

		fileName := strings.Split(fi.Name(), ".")[0]
		if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", fileName); !m {
			continue
		}

		fileSerial := ""
		if strings.Contains(fileName, "-") {
			fileSerial = strings.Split(fileName, "-")[0]
		} else {
			fileSerial = fileName
		}

		if fileSerial != d.Serial {
			continue
		}

		file, err := os.Open(profileLocation)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": profileLocation}).Warn("Unable to load profile")
			continue
		}
		if err = json.NewDecoder(file).Decode(pf); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": profileLocation}).Warn("Unable to decode profile")
			continue
		}
		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"location": profileLocation, "serial": d.Serial}).Warn("Failed to close file handle")
		}

		if pf.Serial == d.Serial {
			if fileName == d.Serial {
				profileList["default"] = pf
			} else {
				name := strings.Split(fileName, "-")[1]
				profileList[name] = pf
			}
			logger.Log(logger.Fields{"location": profileLocation, "serial": d.Serial}).Info("Loaded custom user profile")
		}
	}
	d.UserProfiles = profileList
	d.getDeviceProfile()
}

// getDeviceLcd will check if AIO has LCD pump cover
func (d *Device) getDeviceLcd() {
	if lcd.GetLcdAmount() > 0 {
		d.HasLCD = true
		d.LCDImage = make(map[int]*lcd.ImageData, lcd.GetLcdAmount())
	}
	d.XD5LCDs = len(lcd.GetNonAIOLCDSerials())
}

// getLedStripData will return number of LEDs for given strip ID
func (d *Device) getLedStripData(stripId int) uint8 {
	switch stripId {
	case 0:
		return 0 // None
	case 1:
		return 40 // 1x iCUE LINK LS350 Aurora RGB Light Strip
	case 2:
		return 80 // 2x iCUE LINK LS350 Aurora RGB Light Strip
	case 3:
		return 49 // 1x iCUE LINK LS430 Aurora RGB Light Strip
	case 4:
		return 98 // 2x iCUE LINK LS430 Aurora RGB Light Strip
	}
	return 0
}

// getDeviceProfile will load persistent device configuration
func (d *Device) getDeviceProfile() {
	if len(d.UserProfiles) == 0 {
		logger.Log(logger.Fields{"serial": d.Serial}).Warn("No profile found for device. Probably initial start")
	} else {
		for _, pf := range d.UserProfiles {
			if pf.Active {
				if d.OriginalProfile == nil {
					d.OriginalProfile = pf
				}
				d.DeviceProfile = pf
			}
		}
	}
}

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	var defaultBrightness = uint8(100)
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"

	speedProfiles := make(map[int]string, len(d.Devices))
	rgbProfiles := make(map[int]string, len(d.Devices))
	labels := make(map[int]string, len(d.Devices))
	devicePositions := make(map[int]string, len(d.Devices))
	external := make(map[int]int, len(d.Devices))
	lcdModes := make(map[int]uint8, len(d.Devices))
	lcdImages := make(map[int]string, len(d.Devices))
	lcdRotations := make(map[int]uint8, len(d.Devices))
	lcdDevices := make(map[int]string, len(d.Devices))

	for _, device := range d.Devices {
		speedProfiles[device.ChannelId] = device.Profile
		rgbProfiles[device.ChannelId] = device.RGB
		labels[device.ChannelId] = device.Label

		if device.ContainsPump || device.AIO {
			lcdDevices[device.ChannelId] = device.LCDSerial
		}
	}

	deviceProfile := &DeviceProfile{
		Product:            d.Product,
		Serial:             d.Serial,
		SpeedProfiles:      speedProfiles,
		RGBProfiles:        rgbProfiles,
		Labels:             labels,
		Path:               profilePath,
		BrightnessSlider:   &defaultBrightness,
		OriginalBrightness: 100,
	}

	// First save, assign saved profile to a device
	if d.DeviceProfile == nil {
		m := 1
		for _, device := range d.Devices {
			if device.IsLinkAdapter {
				external[device.ChannelId] = 0
			}

			if device.ContainsPump || device.AIO {
				lcdModes[device.ChannelId] = 0
				lcdRotations[device.ChannelId] = 0
				lcdImages[device.ChannelId] = ""
			}

			rgbProfiles[device.ChannelId] = "static"
			labels[device.ChannelId] = "Set Label"
			devicePositions[m] = device.DeviceId
			m++
		}
		deviceProfile.Active = true
		deviceProfile.DevicePosition = devicePositions
		deviceProfile.ExternalAdapter = external
		deviceProfile.LCDModes = lcdModes
		deviceProfile.LCDImages = lcdImages
		deviceProfile.LCDRotations = lcdRotations
		deviceProfile.LCDDevices = lcdDevices
		d.DeviceProfile = deviceProfile
	} else {
		if d.DeviceProfile.BrightnessSlider == nil {
			deviceProfile.BrightnessSlider = &defaultBrightness
			d.DeviceProfile.BrightnessSlider = &defaultBrightness
		} else {
			deviceProfile.BrightnessSlider = d.DeviceProfile.BrightnessSlider
		}

		if d.DeviceProfile.LCDImages == nil {
			for _, device := range d.Devices {
				if device.ContainsPump || device.AIO {
					lcdImages[device.ChannelId] = ""
				}
			}
			deviceProfile.LCDImages = lcdImages
		} else {
			deviceProfile.LCDImages = d.DeviceProfile.LCDImages
		}

		if d.DeviceProfile.LCDModes == nil {
			for _, device := range d.Devices {
				if device.ContainsPump || device.AIO {
					lcdModes[device.ChannelId] = d.DeviceProfile.LCDMode
				}
			}
			deviceProfile.LCDModes = lcdModes
		} else {
			deviceProfile.LCDModes = d.DeviceProfile.LCDModes
		}

		if d.DeviceProfile.LCDRotations == nil {
			for _, device := range d.Devices {
				if device.ContainsPump || device.AIO {
					lcdRotations[device.ChannelId] = d.DeviceProfile.LCDRotation
				}
			}
			deviceProfile.LCDRotations = lcdRotations
		} else {
			deviceProfile.LCDRotations = d.DeviceProfile.LCDRotations
		}

		if d.DeviceProfile.LCDDevices == nil {
			for _, device := range d.Devices {
				if device.ContainsPump || device.AIO {
					lcdDevices[device.ChannelId] = device.LCDSerial
				}
			}
			deviceProfile.LCDDevices = lcdDevices
		} else {
			deviceProfile.LCDDevices = d.DeviceProfile.LCDDevices
		}

		if d.DeviceProfile.ExternalAdapter == nil {
			for _, device := range d.Devices {
				if device.IsLinkAdapter {
					external[device.ChannelId] = 0
				}
			}
			deviceProfile.ExternalAdapter = external
		} else {
			deviceProfile.ExternalAdapter = d.DeviceProfile.ExternalAdapter
		}

		if d.DeviceProfile.DevicePosition == nil {
			m := 1
			for _, device := range d.Devices {
				devicePositions[m] = device.DeviceId
				m++
			}
			deviceProfile.DevicePosition = devicePositions
		} else {
			posLen := len(d.DeviceProfile.DevicePosition)
			devLen := len(d.Devices)
			if posLen != devLen {
				// New devices are connected, override positions with new data
				logger.Log(logger.Fields{"positions": posLen, "devices": devLen}).Info("Device amount changed compared to positions.")
				m := 1
				for _, device := range d.Devices {
					devicePositions[m] = device.DeviceId
					m++
				}
				deviceProfile.DevicePosition = devicePositions
			} else {
				logger.Log(logger.Fields{"positions": posLen, "devices": devLen}).Info("Device amount matches position amount.")
				deviceProfile.DevicePosition = d.DeviceProfile.DevicePosition
			}
		}

		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.Brightness = d.DeviceProfile.Brightness
		deviceProfile.OriginalBrightness = d.DeviceProfile.OriginalBrightness
		if len(d.DeviceProfile.Path) < 1 {
			deviceProfile.Path = profilePath
			d.DeviceProfile.Path = profilePath
		} else {
			deviceProfile.Path = d.DeviceProfile.Path
		}
		deviceProfile.LCDMode = d.DeviceProfile.LCDMode
		deviceProfile.LCDRotation = d.DeviceProfile.LCDRotation
		deviceProfile.MultiProfile = d.DeviceProfile.MultiProfile
		deviceProfile.MultiRGB = d.DeviceProfile.MultiRGB
	}

	keys := make([]int, 0, len(deviceProfile.DevicePosition))
	for k := range deviceProfile.DevicePosition {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	pos := make(map[int]string, len(d.Devices))
	for _, k := range keys {
		pos[k] = deviceProfile.DevicePosition[k]
	}
	deviceProfile.DevicePosition = pos

	// Convert to JSON
	buffer, err := json.MarshalIndent(deviceProfile, "", "    ")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
		return
	}

	// Create profile filename
	file, fileErr := os.Create(deviceProfile.Path)
	if fileErr != nil {
		logger.Log(logger.Fields{"error": err, "location": deviceProfile.Path}).Error("Unable to create new device profile")
		return
	}

	// Write JSON buffer to file
	_, err = file.Write(buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": deviceProfile.Path}).Error("Unable to write data")
		return
	}

	// Close file
	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": deviceProfile.Path}).Error("Unable to close file handle")
	}
	d.loadDeviceProfiles()
}

// ResetSpeedProfiles will reset channel speed profile if it matches with the current speed profile
// This is used when speed profile is deleted from the UI
func (d *Device) ResetSpeedProfiles(profile string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	i := 0
	for _, device := range d.Devices {
		if device.Profile == profile {
			d.Devices[device.ChannelId].Profile = "Normal"
			i++
		}
	}

	if i > 0 {
		// Save only if something was changed
		d.saveDeviceProfile()
	}
}

// GetTemperatureProbes will return a list of temperature probes
func (d *Device) GetTemperatureProbes() *[]TemperatureProbe {
	return d.TemperatureProbes
}

// UpdateDevicePosition will update device position on WebUI
func (d *Device) UpdateDevicePosition(position, direction int) uint8 {
	newChannelId := ""
	newPosition := 0
	if _, ok := d.DeviceProfile.DevicePosition[position]; ok {
		if direction == 0 {
			if position == 1 {
				return 2
			}
			newChannelId = d.DeviceProfile.DevicePosition[position-1]
			newPosition = position - 1
		} else {
			if position >= len(d.DeviceProfile.DevicePosition) {
				return 2
			}
			newChannelId = d.DeviceProfile.DevicePosition[position+1]
			newPosition = position + 1
		}

		for ck, ch := range d.DeviceProfile.DevicePosition {
			if ch == newChannelId {
				newPosition = ck
				break
			}
		}

		// Current channel id
		currentChannelId := d.DeviceProfile.DevicePosition[position]

		// Swap positions
		d.DeviceProfile.DevicePosition[position] = newChannelId
		d.DeviceProfile.DevicePosition[newPosition] = currentChannelId

		// Save it
		d.saveDeviceProfile()
		return 1
	} else {
		return 0
	}
}

// UpdateDeviceSpeed will update device channel speed.
func (d *Device) UpdateDeviceSpeed(channelId int, value uint16) uint8 {
	// Check if actual channelId exists in the device list
	if device, ok := d.Devices[channelId]; ok {
		channelSpeeds := map[int]byte{}

		if value < 20 {
			value = 20
		}
		// Minimal pump speed should be 50%
		if device.ContainsPump {
			if value < 50 {
				value = 50
			}
		}
		channelSpeeds[device.ChannelId] = byte(value)
		d.setSpeed(channelSpeeds, 0)
		return 1
	}
	return 0
}

// UpdateDeviceLabel will set / update device label
func (d *Device) UpdateDeviceLabel(channelId int, label string) uint8 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if _, ok := d.Devices[channelId]; !ok {
		return 0
	}

	d.Devices[channelId].Label = label
	d.saveDeviceProfile()
	return 1
}

// UpdateDeviceLcd will update device LCD
func (d *Device) UpdateDeviceLcd(channelId int, mode uint8) uint8 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.HasLCD {
		if mode == lcd.DisplayImage {
			if len(lcd.GetLcdImages()) == 0 {
				return 0
			}
		}

		if _, ok := d.DeviceProfile.LCDModes[channelId]; ok {
			if mode == lcd.DisplayImage {
				if lcdImage, ok := d.DeviceProfile.LCDImages[channelId]; ok {
					if len(lcdImage) == 0 {
						image := lcd.GetLcdImages()[0]
						d.DeviceProfile.LCDImages[channelId] = image.Name
						d.LCDImage[channelId] = lcd.GetLcdImage(image.Name)
					} else {
						d.LCDImage[channelId] = lcd.GetLcdImage(lcdImage)
					}
				}
			}
			d.DeviceProfile.LCDModes[channelId] = mode
		}
		d.saveDeviceProfile()
		return 1
	} else {
		return 2
	}
}

// ChangeDeviceLcd will change device LCD
func (d *Device) ChangeDeviceLcd(channelId int, lcdSerial string) uint8 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.HasLCD {
		if _, ok := d.DeviceProfile.LCDDevices[channelId]; ok {
			if device, found := d.Devices[channelId]; found {
				if device.ContainsPump {
					d.DeviceProfile.LCDDevices[channelId] = lcdSerial
					d.Devices[channelId].LCDSerial = lcdSerial
					d.lcdDevices[lcdSerial] = &LCD{
						Lcd:       lcd.GetLcdBySerial(lcdSerial),
						ProductId: 0,
					}
				} else {
					return 2
				}
			} else {
				return 0
			}
		} else {
			return 0
		}
		d.saveDeviceProfile()
		return 1
	} else {
		return 2
	}
}

// UpdateDeviceLedData will update device LED data
func (d *Device) UpdateDeviceLedData(ledProfile led.Device) uint8 {
	if d.ledProfile == nil {
		return 0
	}

	// Go through all devices
	for key, value := range d.ledProfile.Devices {
		// Go through all channels
		for i := range value.Channels {
			// Check if channel we sent exists
			if _, ok := ledProfile.Devices[key].Channels[i]; ok {
				// Update specified channel
				d.ledProfile.Devices[key].Channels[i] = ledProfile.Devices[key].Channels[i]
			}
		}
	}
	return 1
}

// UpdateDeviceLcdRotation will update device LCD rotation
func (d *Device) UpdateDeviceLcdRotation(channelId int, rotation uint8) uint8 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.HasLCD {
		if _, ok := d.DeviceProfile.LCDRotations[channelId]; ok {
			d.DeviceProfile.LCDRotations[channelId] = rotation
		}
		d.saveDeviceProfile()
		d.setLcdRotation()
		return 1
	} else {
		return 2
	}
}

// UpdateDeviceLcdImage will update device LCD image
func (d *Device) UpdateDeviceLcdImage(channelId int, image string) uint8 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.HasLCD {
		if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", image); !m {
			return 0
		}
		if len(lcd.GetLcdImages()) == 0 {
			return 0
		}

		lcdImage := lcd.GetLcdImage(image)
		if lcdImage == nil {
			return 0
		}

		if _, ok := d.DeviceProfile.LCDModes[channelId]; !ok {
			return 0
		}

		d.DeviceProfile.LCDImages[channelId] = image
		d.LCDImage[channelId] = lcdImage
		d.saveDeviceProfile()
		return 1
	} else {
		return 0
	}
}

// setLcdRotation will change LCD rotation
func (d *Device) setLcdRotation() {
	for _, device := range d.Devices {
		if len(device.LCDSerial) > 0 && (device.AIO || device.ContainsPump) {
			if lcdDevice, ok := d.lcdDevices[device.LCDSerial]; ok {
				if lcdDevice.Lcd != nil {
					if rotation, ok := d.DeviceProfile.LCDRotations[device.ChannelId]; ok {
						lcdReport := []byte{0x03, 0x0c, rotation, 0x01}
						_, err := lcdDevice.Lcd.SendFeatureReport(lcdReport)
						if err != nil {
							logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId, "productId": d.ProductId, "serial": d.Serial}).Error("Unable to change LCD rotation")
						}
					}
				}
			}
		}
	}
}

// ChangeDeviceBrightness will change device brightness
func (d *Device) ChangeDeviceBrightness(mode uint8) uint8 {
	if d.GlobalBrightness != 0 {
		return 2
	}
	d.DeviceProfile.Brightness = mode
	d.saveDeviceProfile()
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
	return 1
}

// ChangeDeviceBrightnessValue will change device brightness via slider
func (d *Device) ChangeDeviceBrightnessValue(value uint8) uint8 {
	if d.GlobalBrightness != 0 {
		return 2
	}

	if value < 0 || value > 100 {
		return 0
	}

	d.DeviceProfile.BrightnessSlider = &value
	d.saveDeviceProfile()

	if d.isRgbStatic() {
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true // Exit current RGB mode
			d.activeRgb = nil
		}
		d.setDeviceColor() // Restart RGB
	}
	return 1
}

// SchedulerBrightness will change device brightness via scheduler
func (d *Device) SchedulerBrightness(value uint8) uint8 {
	if value == 0 {
		d.DeviceProfile.OriginalBrightness = *d.DeviceProfile.BrightnessSlider
		d.DeviceProfile.BrightnessSlider = &value
	} else {
		d.DeviceProfile.BrightnessSlider = &d.DeviceProfile.OriginalBrightness
	}

	d.saveDeviceProfile()
	if d.isRgbStatic() {
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true // Exit current RGB mode
			d.activeRgb = nil
		}
		d.setDeviceColor() // Restart RGB
	}
	return 1
}

// ChangeDeviceProfile will change device profile
func (d *Device) ChangeDeviceProfile(profileName string) uint8 {
	if profile, ok := d.UserProfiles[profileName]; ok {
		currentProfile := d.DeviceProfile
		currentProfile.Active = false
		d.DeviceProfile = currentProfile
		d.saveDeviceProfile()

		// RGB reset
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true // Exit current RGB mode
			d.activeRgb = nil
		}

		for _, device := range d.Devices {
			if device.LedChannels > 0 {
				d.Devices[device.ChannelId].RGB = profile.RGBProfiles[device.ChannelId]
			}
			if device.HasSpeed {
				d.Devices[device.ChannelId].Profile = profile.SpeedProfiles[device.ChannelId]
			}
			d.Devices[device.ChannelId].Label = profile.Labels[device.ChannelId]
		}

		newProfile := profile
		newProfile.Active = true
		d.DeviceProfile = newProfile
		d.saveDeviceProfile()
		d.setDeviceColor()
		// Speed reset
		if !config.GetConfig().Manual {
			d.timerSpeed.Stop()
			d.updateDeviceSpeed() // Update device speed
		}
		return 1
	}
	return 0
}

// SaveUserProfile will generate a new user profile configuration and save it to a file
func (d *Device) SaveUserProfile(profileName string) uint8 {
	if d.DeviceProfile != nil {
		profilePath := pwd + "/database/profiles/" + d.Serial + "-" + profileName + ".json"

		newProfile := d.DeviceProfile
		newProfile.Path = profilePath
		newProfile.Active = false

		buffer, err := json.Marshal(newProfile)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
			return 0
		}

		// Create profile filename
		file, err := os.Create(profilePath)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": newProfile.Path}).Error("Unable to create new device profile")
			return 0
		}

		_, err = file.Write(buffer)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": newProfile.Path}).Error("Unable to write data")
			return 0
		}

		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": newProfile.Path}).Error("Unable to close file handle")
			return 0
		}
		d.loadDeviceProfiles()
		return 1
	}
	return 0
}

// UpdateSpeedProfile will update device channel speed.
// If channelId is 0, all device channels will be updated
func (d *Device) UpdateSpeedProfile(channelId int, profile string) uint8 {
	valid := false
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Check if the profile exists
	profiles := temperatures.GetTemperatureProfile(profile)
	if profiles == nil {
		return 0
	}

	// If the profile is liquid temperature, check for the presence of AIOs
	if profiles.Sensor == temperatures.SensorTypeLiquidTemperature {
		for _, device := range d.Devices {
			if device.AIO || device.ContainsPump {
				valid = true
				break
			}
		}

		if !valid {
			return 2
		}
	}

	if profiles.ZeroRpm && !valid {
		return 2
	}

	if profiles.Sensor == temperatures.SensorTypeTemperatureProbe {
		if strings.HasPrefix(profiles.Device, i2cPrefix) {
			if temperatures.GetMemoryTemperature(profiles.ChannelId) == 0 {
				return 5
			}
		} else {
			if profiles.Device != d.Serial {
				return 3
			}

			if _, ok := d.Devices[profiles.ChannelId]; !ok {
				return 4
			}
		}
	}

	if channelId < 0 {
		d.DeviceProfile.MultiProfile = profile
		// All devices
		for _, device := range d.Devices {
			d.Devices[device.ChannelId].Profile = profile
		}
	} else {
		// Check if actual channelId exists in the device list
		if _, ok := d.Devices[channelId]; ok {
			// Update channel with new profile
			d.Devices[channelId].Profile = profile
		}
	}

	// Save to profile
	d.saveDeviceProfile()
	return 1
}

// UpdateSpeedProfileBulk will update device channel speed.
func (d *Device) UpdateSpeedProfileBulk(channelIds []int, profile string) uint8 {
	valid := false
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Check if the profile exists
	profiles := temperatures.GetTemperatureProfile(profile)
	if profiles == nil {
		return 0
	}

	// If the profile is liquid temperature, check for the presence of AIOs
	if profiles.Sensor == temperatures.SensorTypeLiquidTemperature {
		for _, device := range d.Devices {
			if device.AIO || device.ContainsPump {
				valid = true
				break
			}
		}

		if !valid {
			return 2
		}
	}

	if profiles.ZeroRpm && !valid {
		return 2
	}

	if profiles.Sensor == temperatures.SensorTypeTemperatureProbe {
		if strings.HasPrefix(profiles.Device, i2cPrefix) {
			if temperatures.GetMemoryTemperature(profiles.ChannelId) == 0 {
				return 5
			}
		} else {
			if profiles.Device != d.Serial {
				return 3
			}

			if _, ok := d.Devices[profiles.ChannelId]; !ok {
				return 4
			}
		}
	}

	if len(channelIds) > 0 {
		d.DeviceProfile.MultiProfile = profile
		for _, channelId := range channelIds {
			if _, ok := d.Devices[channelId]; ok {
				// Update channel with new profile
				d.Devices[channelId].Profile = profile
			} else {
				return 0
			}
		}
	} else {
		return 0
	}

	// Save to profile
	d.saveDeviceProfile()
	return 1
}

// UpdateDeviceMetrics will update device metrics
func (d *Device) UpdateDeviceMetrics() {
	for _, device := range d.Devices {
		header := &metrics.Header{
			Product:          d.Product,
			Serial:           d.Serial,
			Firmware:         d.Firmware,
			ChannelId:        strconv.Itoa(device.ChannelId),
			Name:             device.Name,
			Description:      device.Description,
			Profile:          device.Profile,
			Label:            device.Label,
			RGB:              device.RGB,
			AIO:              strconv.FormatBool(device.AIO),
			ContainsPump:     strconv.FormatBool(device.ContainsPump),
			Temperature:      float64(device.Temperature),
			LedChannels:      strconv.Itoa(int(device.LedChannels)),
			Rpm:              device.Rpm,
			TemperatureProbe: strconv.FormatBool(device.IsTemperatureProbe),
		}
		metrics.Populate(header)
	}
}

// getTemperatureProbe will return all devices with a temperature probe
func (d *Device) getTemperatureProbe() {
	var probes []TemperatureProbe

	keys := make([]int, 0)
	for k := range d.Devices {
		keys = append(keys, k)
	}

	for _, k := range keys {
		if d.Devices[k].IsTemperatureProbe {
			probe := TemperatureProbe{
				ChannelId: d.Devices[k].ChannelId,
				Name:      d.Devices[k].Name,
				Label:     d.Devices[k].Label,
				Serial:    d.Serial,
				Product:   d.Product,
			}
			probes = append(probes, probe)
		}
	}
	d.TemperatureProbes = &probes
}

// saveRgbProfile will save rgb profile data
func (d *Device) saveRgbProfile() {
	rgbDirectory := pwd + "/database/rgb/"
	rgbFilename := rgbDirectory + d.Serial + ".json"
	if common.FileExists(rgbFilename) {
		buffer, err := json.MarshalIndent(d.Rgb, "", "    ")
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": rgbFilename}).Warn("Unable to encode RGB json")
			return
		}

		// Create profile filename
		file, err := os.Create(rgbFilename)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": rgbFilename}).Warn("Unable to create RGB json file")
			return
		}

		// Write JSON buffer to file
		_, err = file.Write(buffer)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": rgbFilename}).Warn("Unable to write to RGB json file")
			return
		}

		// Close file
		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": rgbFilename}).Warn("Unable to close RGB json file")
			return
		}
	}
}

// UpdateRgbProfileData will update RGB profile data
func (d *Device) UpdateRgbProfileData(profileName string, profile rgb.Profile) uint8 {
	if d.GetRgbProfile(profileName) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}

	pf := d.GetRgbProfile(profileName)
	profile.StartColor.Brightness = pf.StartColor.Brightness
	profile.EndColor.Brightness = pf.EndColor.Brightness
	pf.StartColor = profile.StartColor
	pf.EndColor = profile.EndColor
	pf.Speed = profile.Speed

	d.Rgb.Profiles[profileName] = *pf
	d.saveRgbProfile()
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
	return 1
}

// UpdateRgbProfile will update device RGB profile
func (d *Device) UpdateRgbProfile(channelId int, profile string) uint8 {
	if d.GetRgbProfile(profile) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}
	hasPump := false

	for _, device := range d.Devices {
		if device.ContainsPump {
			hasPump = true
			break
		}
	}

	if profile == "liquid-temperature" {
		if !hasPump {
			logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Unable to apply liquid-temperature profile without a pump of AIO")
			return 2
		}
	}

	if channelId < 0 {
		d.DeviceProfile.MultiRGB = profile
		for _, device := range d.Devices {
			if device.LedChannels > 0 {
				d.DeviceProfile.RGBProfiles[device.ChannelId] = profile
				d.Devices[device.ChannelId].RGB = profile
			}
		}
	} else {
		if _, ok := d.Devices[channelId]; ok {
			d.DeviceProfile.RGBProfiles[channelId] = profile // Set profile
			d.Devices[channelId].RGB = profile
		} else {
			return 0
		}
	}

	d.saveDeviceProfile() // Save profile
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
	return 1
}

// UpdateRgbProfileBulk will update device RGB profile on bulk selected devices
func (d *Device) UpdateRgbProfileBulk(channelIds []int, profile string) uint8 {
	if d.GetRgbProfile(profile) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}
	hasPump := false

	for _, device := range d.Devices {
		if device.ContainsPump {
			hasPump = true
			break
		}
	}

	if profile == "liquid-temperature" {
		if !hasPump {
			logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Unable to apply liquid-temperature profile without a pump of AIO")
			return 2
		}
	}

	if len(channelIds) > 0 {
		d.DeviceProfile.MultiRGB = profile
		for _, channelId := range channelIds {
			if _, ok := d.Devices[channelId]; ok {
				d.DeviceProfile.RGBProfiles[channelId] = profile // Set profile
				d.Devices[channelId].RGB = profile
			} else {
				return 0
			}
		}
	} else {
		return 0
	}

	d.saveDeviceProfile() // Save profile
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
	return 1
}

// UpdateExternalAdapter will update external RGB adapter
func (d *Device) UpdateExternalAdapter(channelId int, stripId int) uint8 {
	if device, ok := d.Devices[channelId]; ok {
		if device.IsLinkAdapter {
			d.DeviceProfile.ExternalAdapter[channelId] = stripId // Set profile
			d.Devices[channelId].ExternalAdapter = stripId
			d.Devices[channelId].LedChannels = d.getLedStripData(stripId)
			d.saveDeviceProfile() // Save profile
			if d.activeRgb != nil {
				d.activeRgb.Exit <- true // Exit current RGB mode
				d.activeRgb = nil
			}
			d.setDeviceColor() // Restart RGB
			return 1
		} else {
			return 2
		}
	} else {
		return 0
	}
}

// getLiquidTemperature will fetch temperature from AIO device
func (d *Device) getLiquidTemperature() float32 {
	for _, device := range d.Devices {
		if device.AIO || device.ContainsPump {
			return device.Temperature
		}
	}
	return 0
}

// updateDeviceSpeed will update device speed based on a temperature reading
func (d *Device) updateDeviceSpeed() {
	d.timerSpeed = time.NewTicker(time.Duration(temperaturePullingInterval) * time.Millisecond)
	go func() {
		tmp := make(map[int]string)
		channelSpeeds := map[int]byte{}

		keys := make([]int, 0)
		for k := range d.Devices {
			keys = append(keys, k)
		}

		for _, k := range keys {
			channelSpeeds[d.Devices[k].ChannelId] = byte(defaultSpeedValue)
		}

		zeroRpmFraction := 1.0
		for {
			select {
			case <-d.timerSpeed.C:
				for _, k := range keys {
					var temp float32 = 0
					profiles := temperatures.GetTemperatureProfile(d.Devices[k].Profile)
					if profiles == nil {
						// No such profile, default to Normal
						profiles = temperatures.GetTemperatureProfile("Normal")
					}

					switch profiles.Sensor {
					case temperatures.SensorTypeGPU:
						{
							temp = temperatures.GetNVIDIAGpuTemperature()
							if temp == 0 {
								temp = temperatures.GetAMDGpuTemperature()
								if temp == 0 {
									logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial}).Warn("Unable to get GPU sensor temperature. Going to fallback to CPU")
									temp = temperatures.GetCpuTemperature()
									if temp == 0 {
										temp = 70
									}
								}
							}
						}
					case temperatures.SensorTypeCPU:
						{
							temp = temperatures.GetCpuTemperature()
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial}).Warn("Unable to get CPU sensor temperature.")
							}
						}
					case temperatures.SensorTypeLiquidTemperature:
						{
							temp = d.getLiquidTemperature()
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial}).Warn("Unable to get liquid temperature.")
							}
						}
					case temperatures.SensorTypeStorage:
						{
							temp = temperatures.GetStorageTemperature(profiles.Device)
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "hwmonDeviceId": profiles.Device}).Warn("Unable to get storage temperature.")
							}
						}
					case temperatures.SensorTypeTemperatureProbe:
						{
							if strings.HasPrefix(profiles.Device, i2cPrefix) {
								temp = temperatures.GetMemoryTemperature(profiles.ChannelId)
							} else {
								if d.Devices[profiles.ChannelId].IsTemperatureProbe {
									temp = d.Devices[profiles.ChannelId].Temperature
								}
							}

							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "channelId": profiles.ChannelId}).Warn("Unable to get probe temperature.")
							}
						}
					case temperatures.SensorTypeCpuGpu:
						{
							cpuTemp := temperatures.GetCpuTemperature()
							gpuTemp := temperatures.GetNVIDIAGpuTemperature()
							if gpuTemp == 0 {
								gpuTemp = temperatures.GetAMDGpuTemperature()
							}

							if gpuTemp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "channelId": profiles.ChannelId, "cpu": cpuTemp, "gpu": gpuTemp}).Warn("Unable to get GPU temperature. Setting to 50")
								gpuTemp = 50
							}

							temp = float32(math.Max(float64(cpuTemp), float64(gpuTemp)))
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "channelId": profiles.ChannelId, "cpu": cpuTemp, "gpu": gpuTemp}).Warn("Unable to get maximum temperature value out of 2 numbers.")
							}
						}
					}

					// All temps failed, default to 50
					if temp == 0 {
						temp = 50
					}

					if config.GetConfig().GraphProfiles {
						var speed byte = 0x00
						pumpValue := temperatures.Interpolate(profiles.Points[0], temp)
						fansValue := temperatures.Interpolate(profiles.Points[1], temp)

						pump := int(math.Round(float64(pumpValue)))
						fans := int(math.Round(float64(fansValue)))

						// Failsafe
						if fans < 20 {
							fans = 20
						}
						if d.Devices[k].AIO {
							if pump < 50 {
								pump = 70
							}
						} else {
							if pump < 20 {
								pump = 30
							}
						}
						if pump > 100 {
							pump = 100
						}
						if fans > 100 {
							fans = 100
						}

						cp := fmt.Sprintf("%s-%d-%f", d.Devices[k].Profile, d.Devices[k].ChannelId, temp)
						if ok := tmp[d.Devices[k].ChannelId]; ok != cp {
							tmp[d.Devices[k].ChannelId] = cp
							if d.Devices[k].ContainsPump {
								speed = byte(pump)
							} else {
								speed = byte(fans)
							}
							if channelSpeeds[d.Devices[k].ChannelId] != speed {
								channelSpeeds[d.Devices[k].ChannelId] = speed
								d.setSpeed(channelSpeeds, 0)
								if d.Debug {
									logger.Log(logger.Fields{"serial": d.Serial, "pump": pump, "fans": fans, "temp": temp, "device": d.Devices[k].Name, "zeroRpm": profiles.ZeroRpm}).Info("updateDeviceSpeed()")
								}
							}
						}
					} else {
						for i := 0; i < len(profiles.Profiles); i++ {
							profile := profiles.Profiles[i]
							minimum := profile.Min + 0.1
							if common.InBetween(temp, minimum, profile.Max) {
								cp := fmt.Sprintf("%s-%d-%d-%d", d.Devices[k].Profile, d.Devices[k].ChannelId, profile.Fans, profile.Pump)
								if ok := tmp[d.Devices[k].ChannelId]; ok != cp {
									tmp[d.Devices[k].ChannelId] = cp
									// Validation
									if profile.Mode < 0 || profile.Mode > 1 {
										profile.Mode = 0
									}

									if profile.Pump > 100 {
										profile.Pump = 70
									}

									if d.Devices[k].AIO {
										if profile.Pump < 50 {
											profile.Pump = 70
										}
									} else {
										if profile.Pump < 20 {
											profile.Pump = 30
										}
									}

									var speed byte = 0x00
									if profiles.ZeroRpm {
										if d.getLiquidTemperature() < 10 {
											if d.Devices[k].ContainsPump {
												speed = byte(profile.Pump)
											} else {
												speed = byte(profile.Fans)
											}
											if channelSpeeds[d.Devices[k].ChannelId] != speed {
												channelSpeeds[d.Devices[k].ChannelId] = speed
												d.setSpeed(channelSpeeds, 0)
											}
										} else {
											if d.Devices[k].ContainsPump {
												speed = byte(profile.Pump)
											} else {
												if d.getLiquidTemperature()+float32(zeroRpmFraction) <= float32(zeroRpmLimit) {
													speed = 0x00
												} else {
													speed = byte(profile.Fans)
												}
											}
											if channelSpeeds[d.Devices[k].ChannelId] != speed {
												channelSpeeds[d.Devices[k].ChannelId] = speed
												d.setSpeed(channelSpeeds, 0)
											}
										}
									} else {
										if d.Devices[k].ContainsPump {
											speed = byte(profile.Pump)
										} else {
											speed = byte(profile.Fans)
										}
										if channelSpeeds[d.Devices[k].ChannelId] != speed {
											channelSpeeds[d.Devices[k].ChannelId] = speed
											d.setSpeed(channelSpeeds, 0)
										}
									}
								}
							}
						}
					}
				}
			case <-d.speedRefreshChan:
				d.timerSpeed.Stop()
				return
			}
		}
	}()
}

// setDefaults will set default mode for all devices
func (d *Device) setDefaults() {
	channelDefaults := map[int]byte{}
	for device := range d.Devices {
		channelDefaults[device] = byte(defaultSpeedValue)
	}
	d.setSpeed(channelDefaults, 0)
}

// setSpeed will generate a speed buffer and send it to a device
func (d *Device) setSpeed(data map[int]byte, mode uint8) {
	if d.Exit {
		return
	}

	buffer := make([]byte, len(data)*4+1)
	buffer[0] = byte(len(data))
	i := 1

	keys := make([]int, 0)

	for k := range data {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		buffer[i] = byte(k)
		buffer[i+1] = mode
		buffer[i+2] = data[k]
		i += 4
	}

	// Validate device response. In case of error, repeat packet
	response := d.write(modeSetSpeed, dataTypeSetSpeed, buffer)
	if len(response) >= 4 {
		if response[3] != 0x00 {
			m := 0
			for {
				m++
				response = d.write(modeSetSpeed, dataTypeSetSpeed, buffer)
				if response[3] == 0x00 || m > 20 {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// setCpuTemperature will store current CPU temperature
func (d *Device) setTemperatures() {
	d.CpuTemp = temperatures.GetCpuTemperature()
	d.GpuTemp = temperatures.GetGpuTemperature()
}

// setAutoRefresh will refresh device data
func (d *Device) setAutoRefresh() {
	d.timer = time.NewTicker(time.Duration(deviceRefreshInterval) * time.Millisecond)
	go func() {
		for {
			select {
			case <-d.timer.C:
				if d.Exit {
					return
				}
				d.setTemperatures()
				d.getDeviceData()
			case <-d.autoRefreshChan:
				d.timer.Stop()
				return
			}
		}
	}()
}

// setDefaults will set default mode for all devices
func (d *Device) getDeviceData() {
	if d.Exit {
		return
	}
	// Speed
	response := d.read(modeGetSpeeds, dataTypeGetSpeeds)
	if response == nil {
		return
	}

	if d.Debug {
		logger.Log(logger.Fields{"serial": d.Serial, "data": fmt.Sprintf("% 2x", response), "type": "speed"}).Info("getDeviceData()")
	}
	amount := response[6]
	sensorData := response[7:]
	valid := response[7]
	if valid == 0x01 {
		for i := 0; i < int(amount); i++ {
			currentSensor := sensorData[i*3 : (i+1)*3]
			status := currentSensor[0]
			if status == 0x00 {
				if _, ok := d.Devices[i]; ok {
					rpm := int16(binary.LittleEndian.Uint16(currentSensor[1:3]))
					if rpm > 1 {
						d.Devices[i].Rpm = rpm
					}
				}
			}
		}
	}

	// Temperature
	if d.Exit {
		return
	}
	response = d.read(modeGetTemperatures, dataTypeGetTemperatures)
	if response[3] == 0x00 {
		amount = response[6]
		sensorData = response[7:]
		valid = response[7]
		if d.Debug {
			logger.Log(logger.Fields{"serial": d.Serial, "data": fmt.Sprintf("% 2x", response), "type": "temperature"}).Info("getDeviceData()")
		}
		if valid == 0x01 {
			for i, s := 0, 0; i < int(amount); i, s = i+1, s+3 {
				currentSensor := sensorData[s : s+3]
				status := currentSensor[0]
				if status == 0x00 {
					if _, ok := d.Devices[i]; ok {
						temp := float32(int16(binary.LittleEndian.Uint16(currentSensor[1:3]))) / 10.0
						if temp > 1 {
							d.Devices[i].Temperature = temp
							d.Devices[i].TemperatureString = dashboard.GetDashboard().TemperatureToString(temp)
						}
					}
				}
			}
		}
	}

	// Update stats
	for key, value := range d.Devices {
		if value.Rpm > 0 || value.Temperature > 0 {
			rpmString := fmt.Sprintf("%v RPM", value.Rpm)
			temperatureString := dashboard.GetDashboard().TemperatureToString(value.Temperature)
			stats.UpdateAIOStats(d.Serial, value.Name, temperatureString, rpmString, value.Label, key)
		}
	}
	d.protectLiquidCooler()
}

// protectLiquidCooler will try to protect your liquid cooler when the temperature reaches critical point
func (d *Device) protectLiquidCooler() {
	for _, device := range d.Devices {
		if device.AIO {
			if device.Temperature > float32(criticalAioCoolantTemp) {
				if d.IsCritical == false {
					d.IsCritical = true
					d.DeviceProfile.LCDMode = 0
					for _, device := range d.Devices {
						d.Devices[device.ChannelId].Profile = "aioCriticalTemperature"
						if device.LedChannels > 0 {
							d.Devices[device.ChannelId].RGB = "colorpulse"
						}
					}
					if d.activeRgb != nil {
						d.activeRgb.Exit <- true // Exit current RGB mode
						d.activeRgb = nil
					}
					d.setDeviceColor() // Restart RGB
				}
			} else {
				if d.IsCritical == true {
					if device.Temperature < float32(criticalAioCoolantTemp-5) {
						d.IsCritical = false
						d.DeviceProfile.LCDMode = d.OriginalProfile.LCDMode
						for _, device := range d.Devices {
							d.Devices[device.ChannelId].Profile = d.OriginalProfile.SpeedProfiles[device.ChannelId]
							if device.LedChannels > 0 {
								d.Devices[device.ChannelId].RGB = d.OriginalProfile.RGBProfiles[device.ChannelId]
							}
						}
						if d.activeRgb != nil {
							d.activeRgb.Exit <- true // Exit current RGB mode
							d.activeRgb = nil
						}
						d.setDeviceColor() // Restart RGB
					}
				}
			}
		}
	}
}

// getSupportedDevice will return supported device or nil pointer
func (d *Device) getSupportedDevice(deviceId byte, deviceModel byte) *SupportedDevice {
	for _, device := range supportedDevices {
		if device.DeviceId == deviceId && device.Model == deviceModel {
			return &device
		}
	}
	return nil
}

// getDevices will fetch all devices connected to a hub
func (d *Device) getDevices() int {
	lcdAvailable := false
	var devices = make(map[int]*Devices)
	var nonAIOLcdData = lcd.GetNonAioLCDData()

	response := d.read(modeGetDevices, dataTypeGetDevices)
	if d.Debug {
		logger.Log(logger.Fields{"serial": d.Serial, "data": fmt.Sprintf("% 2x", response)}).Info("getDevices()")
	}

	channels := response[6]
	data := response[7:]
	position := 0
	for i := 1; i <= int(channels); i++ {
		deviceIdLen := data[position+7]
		if deviceIdLen == 0 {
			position += 8
			continue
		}
		deviceTypeModel := data[position : position+8]
		if deviceTypeModel[2] == 6 || deviceTypeModel[2] == 14 {
			// iCUE LINK COOLER PUMP LCD
			// iCUE LINK XD5 Elite LCD
			lcdAvailable = true
		}

		deviceId := data[position+8 : position+8+int(deviceIdLen)]

		// Get device definition
		deviceMeta := d.getSupportedDevice(deviceTypeModel[2], deviceTypeModel[3])
		if deviceMeta == nil {
			if deviceIdLen > 0 {
				position += 8 + int(deviceIdLen)
			} else {
				position += 8
			}
			continue
		}

		// Get a persistent speed profile. Fallback to Normal is anything fails
		speedProfile := "Normal"
		label := "Set Label"
		if d.DeviceProfile != nil {
			// Profile is set
			if sp, ok := d.DeviceProfile.SpeedProfiles[i]; ok {
				// Profile device channel exists
				if temperatures.GetTemperatureProfile(sp) != nil { // Speed profile exists in configuration
					// Speed profile exists in configuration
					speedProfile = sp
				} else {
					logger.Log(logger.Fields{"serial": d.Serial, "profile": sp}).Warn("Tried to apply non-existing profile")
				}
			} else {
				logger.Log(logger.Fields{"serial": d.Serial, "profile": sp}).Warn("Tried to apply non-existing channel")
			}

			// Device label
			if lb, ok := d.DeviceProfile.Labels[i]; ok {
				if len(lb) > 0 {
					label = lb
				}
			}
		} else {
			logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
		}

		// Get a persistent speed profile. Fallback to Normal is anything fails
		rgbProfile := "static"
		if d.DeviceProfile != nil {
			// Profile is set
			if rp, ok := d.DeviceProfile.RGBProfiles[i]; ok {
				// Profile device channel exists
				if d.GetRgbProfile(rp) != nil { // Speed profile exists in configuration
					// Speed profile exists in configuration
					rgbProfile = rp
				} else {
					logger.Log(logger.Fields{"serial": d.Serial, "profile": rp}).Warn("Tried to apply non-existing rgb profile")
				}
			} else {
				logger.Log(logger.Fields{"serial": d.Serial, "profile": rp}).Warn("Tried to apply rgb profile to the non-existing channel")
			}
		} else {
			logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
		}
		/*
			lcdSerial := ""
			if d.DeviceProfile != nil {
				// Profile is set
				if ls, ok := d.DeviceProfile.LCDDevices[i]; ok {
					if len(ls) > 0 {
						lcdSerial = ls
					}
				} else {
					logger.Log(logger.Fields{"serial": d.Serial, "lcdSerial": ls}).Warn("Tried to apply rgb profile to the non-existing channel")
				}
			} else {
				logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
			}
		*/
		var ledChannels uint8 = 0
		var adapterLedData uint8 = 0
		var adapterData = 0
		if d.DeviceProfile != nil {
			if ea, ok := d.DeviceProfile.ExternalAdapter[i]; ok {
				adapterLedData = d.getLedStripData(ea)
				adapterData = ea
			}
		} else {
			logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
		}

		if deviceTypeModel[2] == 5 {
			ledChannels = adapterLedData
		} else {
			ledChannels = deviceMeta.LedChannels
		}

		// Build device object
		device := &Devices{
			ChannelId:          i,
			Type:               deviceTypeModel[2],
			Model:              deviceTypeModel[3],
			DeviceId:           string(deviceId),
			Name:               deviceMeta.Name,
			DefaultValue:       0,
			Rpm:                0,
			Temperature:        0,
			LedChannels:        ledChannels,
			ContainsPump:       deviceMeta.ContainsPump,
			Description:        deviceMeta.Desc,
			HubId:              d.Serial,
			Profile:            speedProfile,
			RGB:                rgbProfile,
			Label:              label,
			HasSpeed:           deviceMeta.HasSpeed,
			HasTemps:           true,
			AIO:                deviceMeta.AIO,
			PortId:             0,
			IsTemperatureProbe: deviceMeta.TemperatureProbe,
			IsLinkAdapter:      deviceMeta.LinkAdapter,
			ExternalAdapter:    adapterData,
		}

		if device.ContainsPump && device.AIO && len(lcd.GetAioLCDSerial()) > 0 {
			lcdData := lcd.GetAioLCDData()
			if lcdData.Lcd != nil {
				device.LCDSerial = lcdData.Serial
				lcdHidData := &LCD{
					ProductId: 0,
					Lcd:       lcdData.Lcd,
				}
				d.lcdDevices[lcdData.Serial] = lcdHidData
			}
		} else if device.ContainsPump && !device.AIO && len(nonAIOLcdData) == 1 {
			if nonAIOLcdData != nil {
				lcdData := nonAIOLcdData[0]
				if lcdData.Lcd != nil {
					device.LCDSerial = lcdData.Serial
					lcdHidData := &LCD{
						ProductId: 0,
						Lcd:       lcdData.Lcd,
					}
					d.lcdDevices[lcdData.Serial] = lcdHidData
				}
			}
		}

		if i >= 13 {
			device.PortId = 1
		}

		if d.FirmwareInternal[0] < 2 {
			if i >= 7 {
				device.PortId = 1
			}
		}

		if d.Debug {
			logger.Log(logger.Fields{"serial": d.Serial, "device": device}).Info("getDevices()")
		}

		devices[i] = device
		position += 8 + int(deviceIdLen)
	}

	if !lcdAvailable {
		d.HasLCD = lcdAvailable
	}

	// Check if we have LCD Pump Cap and add additional LED channels
	for key, device := range devices {
		// LCD
		if lcdAvailable {
			if device.ContainsPump {
				if device.AIO {
					// AIO LCD cover with additional LEDs
					devices[key].LedChannels = devices[key].LedChannels + uint8(lcdLedChannels)
					// AIO have single LCD pump cover, default to single one
					devices[key].LCDSerial = lcd.GetAioLCDSerial()
				}
				devices[key].Name = devices[key].Name + " LCD"
			}
		}
		// Port overload protection
		d.PortProtection[device.PortId] += int(device.LedChannels)
	}
	d.Devices = devices
	return len(devices)
}

// isRgbStatic will return true or false if all devices are set to static RGB mode
func (d *Device) isRgbStatic() bool {
	s, l := 0, 0

	keys := make([]int, 0)
	for k := range d.Devices {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		if d.Devices[k].LedChannels > 0 {
			l++ // device has LED
			if d.Devices[k].RGB == "static" {
				s++ // led profile is set to static
			}
		}
	}

	if s > 0 || l > 0 { // We have some values
		if s == l {
			return true
		}
	}
	return false
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	// Reset
	reset := map[int][]byte{}
	var buffer []byte
	lightChannels := 0

	keys := make([]int, 0)
	for k := range d.Devices {
		lightChannels += int(d.Devices[k].LedChannels)
		keys = append(keys, k)
	}
	sort.Ints(keys)

	// Do we have any RGB component in the system?
	if lightChannels == 0 {
		logger.Log(logger.Fields{}).Info("No RGB compatible devices found")
		return
	}

	// Reset color
	color := &rgb.Color{Red: 0, Green: 0, Blue: 0, Brightness: 0}
	for _, k := range keys {
		for i := 0; i < int(d.Devices[k].LedChannels); i++ {
			reset[i] = []byte{
				byte(color.Red),
				byte(color.Green),
				byte(color.Blue),
			}
		}
	}
	buffer = rgb.SetColor(reset)
	d.writeColor(buffer)

	// When do we have a combination of QX and RX fans in the chain, QX fan lighting randomly won't turn on.
	// I'm not able to figure out why this is happening, could be related to fans being daisy-chained and how data is
	// flowing through connections.
	// In short, once the initial reset color is sent, we need to wait for 40 ms
	// before sending any new color packets to devices.
	time.Sleep(40 * time.Millisecond)

	if d.isRgbStatic() {
		static := map[int][]byte{}
		profile := d.GetRgbProfile("static")
		if profile == nil {
			return
		}
		profile.StartColor.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)

		// Global override
		if d.GlobalBrightness != 0 {
			profile.StartColor.Brightness = d.GlobalBrightness
		}

		profileColor := rgb.ModifyBrightness(profile.StartColor)
		m := 0

		for _, k := range keys {
			if d.HasLCD && d.Devices[k].AIO {
				for i := 0; i < int(d.Devices[k].LedChannels); i++ {
					static[m] = []byte{
						byte(profileColor.Red),
						byte(profileColor.Green),
						byte(profileColor.Blue),
					}
					if i > 15 && i < 20 {
						static[m] = []byte{byte(color.Red), byte(color.Green), byte(color.Blue)}
					}
					m++
				}
			} else {
				for i := 0; i < int(d.Devices[k].LedChannels); i++ {
					static[m] = []byte{
						byte(profileColor.Red),
						byte(profileColor.Green),
						byte(profileColor.Blue),
					}
					m++
				}
			}
		}
		buffer = rgb.SetColor(static)
		d.writeColor(buffer) // Write color once
		return
	}

	go func(lightChannels int) {
		startTime := time.Now()
		d.activeRgb = rgb.Exit()

		// Generate random colors
		d.activeRgb.RGBStartColor = rgb.GenerateRandomColor(1)
		d.activeRgb.RGBEndColor = rgb.GenerateRandomColor(1)
		rand.New(rand.NewSource(time.Now().UnixNano()))
		for {
			select {
			case <-d.activeRgb.Exit:
				return
			default:
				buff := make([]byte, 0)
				for _, k := range keys {
					rgbCustomColor := true
					profile := d.GetRgbProfile(d.Devices[k].RGB)
					if profile == nil {
						for i := 0; i < int(d.Devices[k].LedChannels); i++ {
							buff = append(buff, []byte{0, 0, 0}...)
						}
						continue
					}

					rgbModeSpeed := common.FClamp(profile.Speed, 0.1, 10)
					// Check if we have custom colors
					if (rgb.Color{}) == profile.StartColor || (rgb.Color{}) == profile.EndColor {
						rgbCustomColor = false
					}

					r := rgb.New(
						int(d.Devices[k].LedChannels),
						rgbModeSpeed,
						nil,
						nil,
						profile.Brightness,
						common.Clamp(profile.Smoothness, 1, 100),
						time.Duration(rgbModeSpeed)*time.Second,
						rgbCustomColor,
					)
					r.HasLCD = d.HasLCD
					r.IsAIO = d.Devices[k].AIO

					if rgbCustomColor {
						r.RGBStartColor = &profile.StartColor
						r.RGBEndColor = &profile.EndColor
					} else {
						r.RGBStartColor = d.activeRgb.RGBStartColor
						r.RGBEndColor = d.activeRgb.RGBEndColor
					}

					// Brightness
					r.RGBBrightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
					r.RGBStartColor.Brightness = r.RGBBrightness
					r.RGBEndColor.Brightness = r.RGBBrightness

					// Global override
					if d.GlobalBrightness != 0 {
						r.RGBBrightness = d.GlobalBrightness
						r.RGBStartColor.Brightness = r.RGBBrightness
						r.RGBEndColor.Brightness = r.RGBBrightness
					}

					r.MinTemp = profile.MinTemp
					r.MaxTemp = profile.MaxTemp

					switch d.Devices[k].RGB {
					case "custom":
						{
							for n := 0; n < int(d.Devices[k].LedChannels); n++ {
								value := d.getLedProfileColor(k, n)
								value.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
								if d.GlobalBrightness != 0 {
									value.Brightness = d.GlobalBrightness
								}
								val := rgb.ModifyBrightness(*value)
								buff = append(buff, []byte{byte(val.Red), byte(val.Green), byte(val.Blue)}...)
							}
						}
					case "off":
						{
							for n := 0; n < int(d.Devices[k].LedChannels); n++ {
								buff = append(buff, []byte{0, 0, 0}...)
							}
						}
					case "rainbow":
						{
							r.Rainbow(startTime)
							buff = append(buff, r.Output...)
						}
					case "watercolor":
						{
							r.Watercolor(startTime)
							buff = append(buff, r.Output...)
						}
					case "liquid-temperature":
						{
							r.Temperature(float64(d.getLiquidTemperature()))
							buff = append(buff, r.Output...)
						}
					case "cpu-temperature":
						{
							r.Temperature(float64(d.CpuTemp))
							buff = append(buff, r.Output...)
						}
					case "gpu-temperature":
						{
							r.Temperature(float64(d.GpuTemp))
							buff = append(buff, r.Output...)
						}
					case "colorpulse":
						{
							r.Colorpulse(&startTime)
							buff = append(buff, r.Output...)
						}
					case "static":
						{
							r.Static()
							buff = append(buff, r.Output...)
						}
					case "rotator":
						{
							r.Rotator(&startTime)
							buff = append(buff, r.Output...)
						}
					case "wave":
						{
							r.Wave(&startTime)
							buff = append(buff, r.Output...)
						}
					case "storm":
						{
							r.Storm()
							buff = append(buff, r.Output...)
						}
					case "flickering":
						{
							r.Flickering(&startTime)
							buff = append(buff, r.Output...)
						}
					case "colorshift":
						{
							r.Colorshift(&startTime, d.activeRgb)
							buff = append(buff, r.Output...)
						}
					case "circleshift":
						{
							r.CircleShift(&startTime)
							buff = append(buff, r.Output...)
						}
					case "circle":
						{
							r.Circle(&startTime)
							buff = append(buff, r.Output...)
						}
					case "spinner":
						{
							r.Spinner(&startTime)
							buff = append(buff, r.Output...)
						}
					case "colorwarp":
						{
							r.Colorwarp(&startTime, d.activeRgb)
							buff = append(buff, r.Output...)
						}
					}
				}
				// Send it
				d.writeColor(buff)
				time.Sleep(30 * time.Millisecond)
			}
		}
	}(lightChannels)
}

// setColorEndpoint will activate hub color endpoint for further usage
func (d *Device) setColorEndpoint() {
	// Close any RGB endpoint
	_, err := d.transfer(cmdCloseEndpoint, modeSetColor)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close endpoint")
	}

	// Open RGB endpoint
	_, err = d.transfer(cmdOpenColorEndpoint, modeSetColor)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to open endpoint")
	}
}

// setHardwareMode will switch a device to hardware mode
func (d *Device) setHardwareMode() {
	_, err := d.transfer(cmdHardwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	_, err := d.transfer(cmdSoftwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
	time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
}

// getManufacturer will return device manufacturer
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
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
	product = strings.Replace(product, "CORSAIR ", "", -1)
	d.Product = product
}

// getSerial will return device serial number
func (d *Device) getSerial() {
	serial, err := d.dev.GetSerialNbr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device serial number")
	}
	d.Serial = serial
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	fw, err := d.transfer(cmdGetFirmware, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
	}

	v1, v2, v3 := int(fw[4]), int(fw[5]), int(binary.LittleEndian.Uint16(fw[6:8]))
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
	if v1 < 2 {
		// 2.3.427 firmware implemented 24 devices and 2.4.438 was released as stable firmware
		logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product, "firmware": d.Firmware}).Info("This firmware can support only 14 devices.")
	}
	d.FirmwareInternal = []int{v1, v2, v3}
}

// read will read data from a device and return data as a byte array
func (d *Device) read(endpoint, bufferType []byte) []byte {
	// Endpoint data
	var buffer []byte

	// Close specified endpoint
	_, err := d.transfer(cmdCloseEndpoint, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close endpoint")
	}

	// Open endpoint
	_, err = d.transfer(cmdOpenEndpoint, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to open endpoint")
	}

	// Read data from endpoint
	buffer, err = d.transfer(cmdRead, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read endpoint")
	}

	if responseMatch(buffer, bufferType) {
		// More data than it can fit into single 512 byte buffer
		next, e := d.transfer(cmdRead, endpoint)
		if e != nil {
			logger.Log(logger.Fields{"error": e}).Error("Unable to read endpoint")
		}
		buffer = append(buffer, next[4:]...)
	}

	// Close specified endpoint
	_, err = d.transfer(cmdCloseEndpoint, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close endpoint")
	}
	return buffer
}

// setupLCDImage will set up lcd image. This function runs continuously through the program lifetime.
func (d *Device) setupLCDImage() {
	go func() {
		for {
			select {
			default:
				{
					if d.Exit {
						return
					}
					for _, device := range d.Devices {
						if len(device.LCDSerial) > 0 && (device.AIO || device.ContainsPump) {
							if lcdMode, ok := d.DeviceProfile.LCDModes[device.ChannelId]; ok {
								if lcdMode != lcd.DisplayImage {
									continue // Don't process images here
								}
								if lcdDevice, ok := d.lcdDevices[device.LCDSerial]; ok {
									if lcdDevice.Lcd == nil {
										d.DeviceProfile.LCDModes[device.ChannelId] = 0
										d.saveDeviceProfile()
										continue
									}

									if image, ok := d.DeviceProfile.LCDImages[device.ChannelId]; ok {
										if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", image); !m {
											d.DeviceProfile.LCDModes[device.ChannelId] = 0
											d.saveDeviceProfile()
											continue
										}

										lcdImage := d.LCDImage[device.ChannelId]
										if lcdImage == nil {
											if len(lcd.GetLcdImages()) > 0 {
												lcdImage = lcd.GetLcdImage(lcd.GetLcdImages()[0].Name)
											}
											if lcdImage == nil {
												d.DeviceProfile.LCDModes[device.ChannelId] = 0
												d.saveDeviceProfile()
												continue
											}
										}

										if lcdImage.Frames > 1 {
											for i := 0; i < d.LCDImage[device.ChannelId].Frames; i++ {
												if d.DeviceProfile.LCDModes[device.ChannelId] != lcd.DisplayImage {
													break
												}
												data := d.LCDImage[device.ChannelId].Buffer[i]
												buffer := data.Buffer
												delay := data.Delay
												d.transferToLcd(buffer, lcdDevice.Lcd)
												if delay > 0 {
													time.Sleep(time.Duration(delay) * time.Millisecond)
												} else {
													// Single frame, static image, generate 100ms of delay
													time.Sleep(10 * time.Millisecond)
												}
											}
										} else {
											data := lcdImage.Buffer[0]
											buffer := data.Buffer
											delay := data.Delay
											d.transferToLcd(buffer, lcdDevice.Lcd)
											if delay > 0 {
												time.Sleep(time.Duration(delay) * time.Millisecond)
											} else {
												// Single frame, static image, generate 100ms of delay
												time.Sleep(10 * time.Millisecond)
											}
										}
									} else {
										continue
									}
								}
							}
						}
					}
					time.Sleep(10 * time.Millisecond)
				}
			case <-d.lcdImageChan:
				return
			}
		}
	}()
}

// setupLCD will activate and configure LCD
func (d *Device) setupLCD() {
	lcdTimer := time.NewTicker(time.Duration(lcdRefreshInterval) * time.Millisecond)
	d.lcdRefreshChan = make(chan struct{})
	go func() {
		for {
			select {
			case <-lcdTimer.C:
				for _, device := range d.Devices {
					if len(device.LCDSerial) > 0 && (device.AIO || device.ContainsPump) {
						if lcdDevice, ok := d.lcdDevices[device.LCDSerial]; ok {
							if lcdDevice.Lcd == nil {
								continue
							}

							if lcdMode, ok := d.DeviceProfile.LCDModes[device.ChannelId]; ok {
								if lcdMode == lcd.DisplayImage {
									continue // Don't process images here
								}

								switch lcdMode {
								case lcd.DisplayCPU:
									{
										buffer := lcd.GenerateScreenImage(
											lcd.DisplayCPU,
											int(temperatures.GetCpuTemperature()),
											0,
											0,
											0,
										)
										d.transferToLcd(buffer, lcdDevice.Lcd)
									}
								case lcd.DisplayGPU:
									{
										buffer := lcd.GenerateScreenImage(
											lcd.DisplayGPU,
											int(temperatures.GetGpuTemperature()),
											0,
											0,
											0,
										)
										d.transferToLcd(buffer, lcdDevice.Lcd)
									}
								case lcd.DisplayLiquid:
									{
										buffer := lcd.GenerateScreenImage(
											lcd.DisplayLiquid,
											int(device.Temperature),
											0,
											0,
											0,
										)
										d.transferToLcd(buffer, lcdDevice.Lcd)
									}
								case lcd.DisplayPump:
									{
										buffer := lcd.GenerateScreenImage(
											lcd.DisplayPump,
											int(device.Rpm),
											0,
											0,
											0,
										)
										d.transferToLcd(buffer, lcdDevice.Lcd)
									}
								case lcd.DisplayAllInOne:
									{
										liquidTemp := 0
										cpuTemp := 0
										pumpSpeed := 0
										liquidTemp = int(device.Temperature)
										pumpSpeed = int(device.Rpm)

										cpuTemp = int(temperatures.GetCpuTemperature())
										buffer := lcd.GenerateScreenImage(
											lcd.DisplayAllInOne,
											liquidTemp,
											cpuTemp,
											pumpSpeed,
											0,
										)
										d.transferToLcd(buffer, lcdDevice.Lcd)
									}
								case lcd.DisplayCpuGpuTemp:
									{
										cpuTemp := int(temperatures.GetCpuTemperature())
										gpuTemp := int(temperatures.GetGpuTemperature())
										buffer := lcd.GenerateScreenImage(
											lcd.DisplayCpuGpuTemp,
											cpuTemp,
											gpuTemp,
											0,
											0,
										)
										d.transferToLcd(buffer, lcdDevice.Lcd)
									}
								case lcd.DisplayCpuGpuLoad:
									{
										cpuUtil := int(systeminfo.GetCpuUtilization())
										gpuUtil := systeminfo.GetGPUUtilization()
										buffer := lcd.GenerateScreenImage(
											lcd.DisplayCpuGpuLoad,
											cpuUtil,
											gpuUtil,
											0,
											0,
										)
										d.transferToLcd(buffer, lcdDevice.Lcd)
									}
								case lcd.DisplayCpuGpuLoadTemp:
									{
										cpuTemp := int(temperatures.GetCpuTemperature())
										gpuTemp := int(temperatures.GetGpuTemperature())
										cpuUtil := int(systeminfo.GetCpuUtilization())
										gpuUtil := systeminfo.GetGPUUtilization()
										buffer := lcd.GenerateScreenImage(
											lcd.DisplayCpuGpuLoadTemp,
											cpuTemp,
											gpuTemp,
											cpuUtil,
											gpuUtil,
										)
										d.transferToLcd(buffer, lcdDevice.Lcd)
									}
								case lcd.DisplayTime:
									{
										buffer := lcd.GenerateScreenImage(
											lcd.DisplayTime,
											0,
											0,
											0,
											0,
										)
										d.transferToLcd(buffer, lcdDevice.Lcd)
									}
								case lcd.DisplayArc:
									{
										val := 0
										arcType := 0
										sensor := 0
										switch lcd.GetArc().Sensor {
										case 0:
											val = int(temperatures.GetCpuTemperature())
											break
										case 1:
											val = int(temperatures.GetGpuTemperature())
											arcType = 1
											break
										case 2:
											val = int(d.getLiquidTemperature())
											arcType = 2
											sensor = 2
											break
										case 3:
											val = int(systeminfo.GetCpuUtilization())
											sensor = 3
											break
										case 4:
											val = systeminfo.GetGPUUtilization()
											sensor = 4
										}
										image := lcd.GenerateArcScreenImage(arcType, sensor, val)
										if image != nil {
											d.transferToLcd(image, lcdDevice.Lcd)
										}
									}
								case lcd.DisplayDoubleArc:
									{
										values := []float32{
											temperatures.GetCpuTemperature(),
											temperatures.GetGpuTemperature(),
											d.getLiquidTemperature(),
											float32(systeminfo.GetCpuUtilization()),
											float32(systeminfo.GetGPUUtilization()),
										}
										image := lcd.GenerateDoubleArcScreenImage(values)
										if image != nil {
											d.transferToLcd(image, lcdDevice.Lcd)
										}
									}
								case lcd.DisplayAnimation:
									{
										values := []float32{
											temperatures.GetCpuTemperature(),
											temperatures.GetGpuTemperature(),
											d.getLiquidTemperature(),
											float32(systeminfo.GetCpuUtilization()),
											float32(systeminfo.GetGPUUtilization()),
										}
										image := lcd.GenerateAnimationScreenImage(values)
										if image != nil {
											imageLen := len(image)
											for i := 0; i < imageLen; i++ {
												d.transferToLcd(image[i].Buffer, lcdDevice.Lcd)
												if i != imageLen-1 {
													if image[i].Delay > 0 {
														time.Sleep(time.Duration(image[i].Delay) * time.Millisecond)
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			case <-d.lcdRefreshChan:
				lcdTimer.Stop()
				return
			}
		}
	}()
}

// write will write data to the device with specific endpoint
func (d *Device) write(endpoint, bufferType, data []byte) []byte {
	// Buffer
	buffer := make([]byte, len(bufferType)+len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)+2))
	copy(buffer[headerWriteSize:headerWriteSize+len(bufferType)], bufferType)
	copy(buffer[headerWriteSize+len(bufferType):], data)

	// Create read buffer
	bufferR := make([]byte, bufferSize)

	if d.Exit {
		return bufferR
	}

	// Close endpoint
	_, err := d.transfer(cmdCloseEndpoint, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close endpoint")
		return bufferR
	}

	// Open endpoint
	_, err = d.transfer(cmdOpenEndpoint, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to open endpoint")
		return bufferR
	}

	// Send it
	bufferR, err = d.transfer(cmdWrite, buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to endpoint")
		return bufferR
	}

	// Close endpoint
	_, err = d.transfer(cmdCloseEndpoint, endpoint)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close endpoint")
		return bufferR
	}
	return bufferR
}

// writeColor will write data to the device with a specific endpoint.
// writeColor does not require endpoint closing and opening like normal Write requires.
// Endpoint is open only once. Once the endpoint is open, color can be sent continuously.
func (d *Device) writeColor(data []byte) {
	if d.Exit {
		return
	}

	// Buffer
	buffer := make([]byte, len(dataTypeSetColor)+len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)+2))
	copy(buffer[headerWriteSize:headerWriteSize+len(dataTypeSetColor)], dataTypeSetColor)
	copy(buffer[headerWriteSize+len(dataTypeSetColor):], data)

	// Process buffer and create a chunked array if needed
	writeColorEp := cmdWriteColor
	colorEp := make([]byte, len(writeColorEp))
	copy(colorEp, writeColorEp)

	chunks := common.ProcessMultiChunkPacket(buffer, maxBufferSizePerRequest)
	for i, chunk := range chunks {
		if d.Exit {
			break
		}
		if i == 0 {
			// Initial packet is using cmdWriteColor
			_, err := d.transfer(cmdWriteColor, chunk)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
			}
		} else {
			// Chunks don't use cmdWriteColor, they use static dataTypeSubColor
			_, err := d.transfer(dataTypeSubColor, chunk)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to endpoint")
			}
		}
	}
}

// transferToLcd will transfer data to LCD panel
func (d *Device) transferToLcd(buffer []byte, lcdDevice *hid.Device) {
	d.mutexLcd.Lock()
	defer d.mutexLcd.Unlock()

	chunks := common.ProcessMultiChunkPacket(buffer, maxLCDBufferSizePerRequest)
	for i, chunk := range chunks {
		if d.Exit {
			break
		}

		bufferW := make([]byte, lcdBufferSize)
		bufferW[0] = 0x02
		bufferW[1] = 0x05
		bufferW[2] = 0x01

		// The last packet needs to end with 0x01 in order for display to render data
		if len(chunk) < maxLCDBufferSizePerRequest {
			bufferW[3] = 0x01
		}

		bufferW[4] = byte(i)
		binary.LittleEndian.PutUint16(bufferW[6:8], uint16(len(chunk)))
		copy(bufferW[lcdHeaderSize:], chunk)

		if d.Debug {
			logger.Log(logger.Fields{
				"lcdData": fmt.Sprintf("% 2x", bufferW),
				"length":  len(chunk),
				"chunk":   i,
			}).Info("LCD DEBUG DATA")
		}
		if lcdDevice != nil {
			if _, err := lcdDevice.Write(bufferW); err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
				break
			}
		}
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint, buffer []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Create read buffer
	bufferR := make([]byte, bufferSize)
	if d.Exit {
		// Create write buffer
		bufferW := make([]byte, bufferSizeWrite)
		bufferW[2] = 0x01
		endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
		copy(endpointHeaderPosition, endpoint)
		if len(buffer) > 0 {
			copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
		}
		if _, err := d.dev.Write(bufferW); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		}
	} else {
		// Create write buffer
		bufferW := make([]byte, bufferSizeWrite)
		bufferW[2] = 0x01
		endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
		copy(endpointHeaderPosition, endpoint)
		if len(buffer) > 0 {
			copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
		}

		// Send command to a device
		if _, err := d.dev.Write(bufferW); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		}

		// Get data from a device
		if _, err := d.dev.Read(bufferR); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		}
	}
	return bufferR, nil
}

// responseMatch will check if two byte arrays match
func responseMatch(response, expected []byte) bool {
	responseBuffer := response[4:6]
	return bytes.Equal(responseBuffer, expected)
}
