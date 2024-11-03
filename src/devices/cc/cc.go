package cc

// Package: CORSAIR iCUE COMMANDER CORE
// This is the primary package for CORSAIR iCUE COMMANDER CORE.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later
// Supported devices:
// - iCUE COMMANDER CORE - 0c32
// - iCUE COMMANDER CORE - 0c1c
// - iCUE H100i ELITE CAPELLIX
// - iCUE H115i ELITE CAPELLIX
// - iCUE H150i ELITE CAPELLIX
// - iCUE H170i ELITE CAPELLIX
// - H100i ELITE LCD
// - H150i ELITE LCD
// - H170i ELITE LCD
// - H100i ELITE CAPELLIX
// - H150i ELITE CAPELLIX
// - H170i ELITE CAPELLIX
// - iCUE H100i ELITE LCD XT
// - iCUE H115i ELITE LCD XT
// - iCUE H150i ELITE LCD XT
// - iCUE H170i ELITE LCD XT
// - iCUE H100i ELITE CAPELLIX XT
// - iCUE H115i ELITE CAPELLIX XT
// - iCUE H150i ELITE CAPELLIX XT
// - iCUE H170i ELITE CAPELLIX XT
// - 1x Temperature Probe

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices/lcd"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/systeminfo"
	"OpenLinkHub/src/temperatures"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	pwd                        = ""
	cmdOpenEndpoint            = []byte{0x0d, 0x01}
	cmdOpenColorEndpoint       = []byte{0x0d, 0x00}
	cmdCloseEndpoint           = []byte{0x05, 0x01, 0x01}
	cmdGetFirmware             = []byte{0x02, 0x13}
	cmdGetPumpVersion          = []byte{0x02, 0x57}
	cmdGetRadiatorType         = []byte{0x02, 0x58}
	cmdSoftwareMode            = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode            = []byte{0x01, 0x03, 0x00, 0x01}
	cmdWrite                   = []byte{0x06, 0x01}
	cmdWriteColor              = []byte{0x06, 0x00}
	cmdRead                    = []byte{0x08, 0x01}
	cmdGetDeviceMode           = []byte{0x02, 0x03, 0x00}
	cmdSetLedPorts             = []byte{0x1e}
	modeGetLeds                = []byte{0x20}
	modeGetSpeeds              = []byte{0x17}
	modeSetSpeed               = []byte{0x18}
	modeGetTemperatures        = []byte{0x21}
	modeGetFans                = []byte{0x1a}
	modeSetColor               = []byte{0x22}
	dataTypeGetTemperatures    = []byte{0x10, 0x00}
	dataTypeGetSpeeds          = []byte{0x06, 0x00}
	dataTypeSetSpeed           = []byte{0x07, 0x00}
	dataTypeGetFans            = []byte{0x09, 0x00}
	dataTypeGetLeds            = []byte{0x0f, 0x00}
	dataTypeSetColor           = []byte{0x12, 0x00}
	dataTypeSubColor           = []byte{0x07, 0x00}
	mutex                      sync.Mutex
	mutexLcd                   sync.Mutex
	bufferSize                 = 64
	bufferSizeWrite            = bufferSize + 1
	transferTimeout            = 500
	headerSize                 = 2
	headerWriteSize            = 4
	authRefreshChan            = make(chan bool)
	speedRefreshChan           = make(chan bool)
	lcdRefreshChan             = make(chan bool)
	deviceRefreshInterval      = 1000
	lcdRefreshInterval         = 1000
	defaultSpeedValue          = 50
	temperaturePullingInterval = 3000
	ledStartIndex              = 6
	maxBufferSizePerRequest    = 61
	timer                      = &time.Ticker{}
	timerSpeed                 = &time.Ticker{}
	lcdTimer                   = &time.Ticker{}
	internalLedDevices         = make(map[int]*LedChannel, 7)
	lcdHeaderSize              = 8
	lcdBufferSize              = 1024
	maxLCDBufferSizePerRequest = lcdBufferSize - lcdHeaderSize
	aioList                    = []AIOList{
		{Name: "H100i ELITE CAPELLIX", PumpVersion: 1, RadiatorSize: 240},
		{Name: "H100i ELITE CAPELLIX", PumpVersion: 2, RadiatorSize: 240},
		{Name: "H115i ELITE CAPELLIX", PumpVersion: 1, RadiatorSize: 280},
		{Name: "H150i ELITE CAPELLIX", PumpVersion: 1, RadiatorSize: 360},
		{Name: "H150i ELITE CAPELLIX", PumpVersion: 2, RadiatorSize: 360},
		{Name: "H170i ELITE CAPELLIX", PumpVersion: 1, RadiatorSize: 420},
		{Name: "H100i ELITE LCD", PumpVersion: 3, RadiatorSize: 240, LCD: true},
		{Name: "H150i ELITE LCD", PumpVersion: 3, RadiatorSize: 360, LCD: true},
		{Name: "H170i ELITE LCD", PumpVersion: 3, RadiatorSize: 420, LCD: true},
		{Name: "H100i ELITE CAPELLIX", PumpVersion: 3, RadiatorSize: 240, LCD: false},
		{Name: "H150i ELITE CAPELLIX", PumpVersion: 3, RadiatorSize: 360, LCD: false},
		{Name: "H170i ELITE CAPELLIX", PumpVersion: 3, RadiatorSize: 420, LCD: false},
		{Name: "H100i ELITE LCD XT", PumpVersion: 5, RadiatorSize: 240, LCD: true},
		{Name: "H100i ELITE LCD XT", PumpVersion: 6, RadiatorSize: 240, LCD: true},
		{Name: "H115i ELITE LCD XT", PumpVersion: 5, RadiatorSize: 280, LCD: true},
		{Name: "H150i ELITE LCD XT", PumpVersion: 5, RadiatorSize: 360, LCD: true},
		{Name: "H150i ELITE LCD XT", PumpVersion: 6, RadiatorSize: 360, LCD: true},
		{Name: "H170i ELITE LCD XT", PumpVersion: 5, RadiatorSize: 420, LCD: true},
		{Name: "H100i ELITE CAPELLIX XT", PumpVersion: 5, RadiatorSize: 240, LCD: false},
		{Name: "H100i ELITE CAPELLIX XT", PumpVersion: 6, RadiatorSize: 240, LCD: false},
		{Name: "H115i ELITE CAPELLIX XT", PumpVersion: 5, RadiatorSize: 280, LCD: false},
		{Name: "H150i ELITE CAPELLIX XT", PumpVersion: 5, RadiatorSize: 360, LCD: false},
		{Name: "H150i ELITE CAPELLIX XT", PumpVersion: 6, RadiatorSize: 360, LCD: false},
		{Name: "H170i ELITE CAPELLIX XT", PumpVersion: 5, RadiatorSize: 420, LCD: false},
	}
	externalLedDevices = []ExternalLedDevice{
		{
			Index:   0,
			Name:    "No Device",
			Total:   0,
			Command: 00,
		},
		{
			Index:   1,
			Name:    "RGB Led Strip",
			Total:   10,
			Command: 01,
		},
		{
			Index:   2,
			Name:    "HD RGB Series Fan",
			Total:   12,
			Command: 04,
		},
		{
			Index:   3,
			Name:    "LL RGB Series Fan",
			Total:   16,
			Command: 02,
		},
		{
			Index:   4,
			Name:    "ML PRO RGB Series Fan",
			Total:   4,
			Command: 03,
		},
		{
			Index:   5,
			Name:    "QL RGB Series Fan",
			Total:   34,
			Command: 06,
		},
		{
			Index:   6,
			Name:    "8-LED Series Fan",
			Total:   8,
			Command: 05,
		},
	}
)

type ExternalLedDevice struct {
	Index   int
	Name    string
	Total   int
	Command byte
}

// AIOList struct for supported AIO devices
type AIOList struct {
	Name         string
	PumpVersion  int16
	RadiatorSize int16
	LCD          bool
}

// DeviceMonitor struct contains the shared variable and synchronization primitives
type DeviceMonitor struct {
	Status byte
	Lock   sync.Mutex
	Cond   *sync.Cond
}

// LedChannel struct for LED pump and fan data
type LedChannel struct {
	Total   uint8
	Command byte
	Name    string
}

// DeviceProfile struct contains all device profile
type DeviceProfile struct {
	Active        bool
	Path          string
	Product       string
	Serial        string
	LCDMode       uint8
	LCDRotation   uint8
	Brightness    uint8
	RGBProfiles   map[int]string
	SpeedProfiles map[int]string
	Labels        map[int]string
	RGBLabels     map[int]string
	CustomLEDs    map[int]int
}

type TemperatureProbe struct {
	ChannelId int
	Name      string
	Label     string
	Serial    string
	Product   string
}

// Devices struct contain information about connected devices
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
	HasSpeed           bool
	HasTemps           bool
	IsTemperatureProbe bool
}

// Device struct contains primary device data
type Device struct {
	Debug             bool
	dev               *hid.Device
	lcd               *hid.Device
	Manufacturer      string                    `json:"manufacturer"`
	Product           string                    `json:"product"`
	Serial            string                    `json:"serial"`
	Firmware          string                    `json:"firmware"`
	AIOType           string                    `json:"-"`
	Devices           map[int]*Devices          `json:"devices"`
	RgbDevices        map[int]*Devices          `json:"rgbDevices"`
	UserProfiles      map[string]*DeviceProfile `json:"userProfiles"`
	ExternalLedDevice []ExternalLedDevice
	DeviceProfile     *DeviceProfile
	deviceMonitor     *DeviceMonitor
	TemperatureProbes *[]TemperatureProbe
	activeRgb         *rgb.ActiveRGB
	Template          string
	HasLCD            bool
	VendorId          uint16
	LCDModes          map[int]string
	LCDRotations      map[int]string
	Brightness        map[int]string
	CpuTemp           float32
	GpuTemp           float32
	FreeLedPorts      map[int]string
	FreeLedPortLEDs   map[int]string
}

/*
// Hard reset of all device LED ports
// Uses ONLY if you brick your device regarding LED (e.g., stuck on red color permanently)
func (d *Device) hardLedReset() {
	buf := make([]byte, 16)
	m := 4
	buf[0] = 0x0d
	buf[1] = 0x00
	buf[2] = 0x07
	buf[3] = 0x00
	for i := 0; i < 6; i++ {
		buf[m] = 0x01
		m++
		buf[m] = 0x06
		m++
	}
	d.write(cmdSetLedPorts, nil, buf, false)
}
*/

// Init will initialize a new device
func Init(vendorId, productId uint16, serial string) *Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	// Open device, return if failure
	dev, err := hid.Open(vendorId, productId, serial)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId}).Error("Unable to open HID device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:               dev,
		Template:          "cc.html",
		VendorId:          vendorId,
		ExternalLedDevice: externalLedDevices,
		LCDModes: map[int]string{
			0: "Liquid Temperature",
			1: "Pump Speed",
			2: "CPU Temperature",
			3: "GPU Temperature",
			4: "Combined",
			6: "CPU / GPU Temp",
			7: "CPU / GPU Load",
			8: "CPU / GPU Load/Temp",
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
		FreeLedPorts:    make(map[int]string, 6),
		FreeLedPortLEDs: make(map[int]string, 34),
	}

	// Generate maximum amount of LEDs port can hold
	for i := 1; i < 35; i++ {
		if i > 1 {
			d.FreeLedPortLEDs[i] = fmt.Sprintf("%d LEDs", i)
		} else {
			d.FreeLedPortLEDs[i] = fmt.Sprintf("%d LED", i)
		}
	}

	// There are 2 CCs. One has a packet size of 64 and the other has 96.
	// This matters only for RGB operations due to packet chunking.
	if productId == 3100 { // 0c1c
		bufferSize = 96
		bufferSizeWrite = bufferSize + 1
		maxBufferSizePerRequest = 93
	}

	// Bootstrap
	d.getDebugMode()        // Debug mode
	d.getManufacturer()     // Manufacturer
	d.getProduct()          // Product
	d.getSerial()           // Serial
	d.loadDeviceProfiles()  // Load all device profiles
	d.getDeviceLcd()        // Check if LCD pump cover is installed
	d.getDeviceProfile()    // Get device profile if any
	d.getDeviceFirmware()   // Firmware
	d.setSoftwareMode()     // Activate software mode
	d.initLedPorts()        // Init LED ports
	d.getDeviceType()       // Find an AIO device type
	d.getLedDevices()       // Get LED devices
	d.getDevices()          // Get devices connected to a hub
	d.getRgbDevices()       // Get RGB devices connected to a hub
	d.setColorEndpoint()    // Set device color endpoint
	d.setDefaults()         // Set default speed and color values for fans and pumps
	d.setAutoRefresh()      // Set auto device refresh
	d.saveDeviceProfile()   // Save
	d.getTemperatureProbe() // Devices with temperature probes
	d.resetLEDPorts()       // Reset device LED
	if config.GetConfig().Manual {
		fmt.Println(
			fmt.Sprintf("[%s] Manual flag enabled. Process will not monitor temperature or adjust fan speed.", d.Serial),
		)
	} else {
		d.updateDeviceSpeed() // Update device speed
	}
	d.setDeviceColor()   // Device color
	d.newDeviceMonitor() // Device monitor
	if d.HasLCD {
		d.setupLCD() // LCD
	}
	logger.Log(logger.Fields{"device": d}).Info("Device successfully initialized")
	return d
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	logger.Log(logger.Fields{"serial": d.Serial}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}
	timer.Stop()
	if !config.GetConfig().Manual {
		timerSpeed.Stop()
		speedRefreshChan <- true
	}
	authRefreshChan <- true

	if d.lcd != nil {
		lcdRefreshChan <- true
		lcdTimer.Stop()

		// Switch LCD back to hardware mode
		lcdReports := map[int][]byte{0: {0x03, 0x1e, 0x01, 0x01}, 1: {0x03, 0x1d, 0x00, 0x01}}
		for i := 0; i <= 1; i++ {
			_, e := d.lcd.SendFeatureReport(lcdReports[i])
			if e != nil {
				logger.Log(logger.Fields{"error": e}).Error("Unable to send report to LCD HID device")
			}
		}
		// Close it
		err := d.lcd.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close LCD HID device")
		}
	}

	d.setHardwareMode()
	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
		}
	}
}

// loadDeviceProfiles will load custom user profiles
func (d *Device) loadDeviceProfiles() {
	profileList := make(map[string]*DeviceProfile, 0)
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
	lcdSerialNumber := ""
	var lcdProductId uint16 = 3129

	enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
		if info.InterfaceNbr == 0 {
			d.HasLCD = true
			lcdSerialNumber = info.SerialNbr
		}
		return nil
	})

	// Enumerate all Corsair devices
	err := hid.Enumerate(d.VendorId, lcdProductId, enum)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId, "serial": d.Serial}).Fatal("Unable to enumerate LCD devices")
		return
	}

	if d.HasLCD {
		logger.Log(logger.Fields{"vendorId": d.VendorId, "productId": lcdProductId, "serial": d.Serial, "lcdSerial": lcdSerialNumber}).Info("LCD pump cover detected")
		lcdPanel, e := hid.Open(d.VendorId, lcdProductId, lcdSerialNumber)
		if e != nil {
			d.HasLCD = false // We failed
			logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId, "productId": lcdProductId, "serial": d.Serial}).Error("Unable to open LCD HID device")
			return
		}
		d.lcd = lcdPanel
	}
}

// getManufacturer will return device manufacturer
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
}

// getManufacturer will return device manufacturer
func (d *Device) getManufacturer() {
	manufacturer, err := d.dev.GetMfrStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to get manufacturer")
	}
	d.Manufacturer = manufacturer
}

// getProduct will return device name
func (d *Device) getProduct() {
	product, err := d.dev.GetProductStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to get product")
	}
	product = strings.Replace(product, "CORSAIR ", "", -1)
	d.Product = product
}

// getSerial will return device serial number
func (d *Device) getSerial() {
	serial, err := d.dev.GetSerialNbr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to get device serial number")
	}
	d.Serial = serial
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	fw, err := d.transfer(
		cmdGetFirmware,
		nil,
		nil,
	)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to a device")
	}

	v1, v2, v3 := int(fw[3]), int(fw[4]), int(binary.LittleEndian.Uint16(fw[5:7]))
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

// getLedDevices will get all connected LED data
func (d *Device) getLedDevices() {
	m := 0
	// LED channels
	lc := d.read(modeGetLeds, dataTypeGetLeds)
	ld := lc[ledStartIndex:] // Channel data starts from position 6 and 4x increments per channel
	amount := 7
	for i := 0; i < amount; i++ {
		var numLEDs uint16 = 0
		var command byte = 00
		var name = ""
		// Initialize LED channel data
		leds := &LedChannel{
			Total:   0,
			Command: 00,
			Name:    "",
		}

		// Check if device status is 2, aka connected
		connected := ld[m] == 2

		if connected {
			// Get number of LEDs
			numLEDs = binary.LittleEndian.Uint16(ld[m+2 : m+4])

			// Each LED device has different command code
			switch numLEDs {
			case 4:
				{
					command = 03
					name = "ML PRO RGB Series Fan"
				}
			case 8:
				{
					command = 05
					name = "8-LED Series Fan"
				}
			case 10:
				{
					command = 01
					name = "RGB Led Strip"
				}
			case 12:
				{
					command = 04
					name = "HD RGB Series Fan"
				}
			case 16:
				{
					command = 02
					name = "LL RGB Series Fan"
				}
			case 21:
				{
					command = 0x08
					name = "Pump"
				}
			case 24:
				{
					command = 0x08
					name = "Pump"
				}
			case 29:
				{
					command = 0x08
					name = "Pump"
				}
			case 34:
				{
					command = 06
					name = "QL RGB Series Fan"
				}
			}

			// Set values
			leds.Total = uint8(numLEDs)
			leds.Command = command
			leds.Name = name
			// Add to a device map
			internalLedDevices[i] = leds
		} else {
			// Add to a device map
			internalLedDevices[i] = leds
			d.FreeLedPorts[i] = fmt.Sprintf("RGB Port %d", i)
		}
		m += 4
	}
}

// getExternalLedDevice will return ExternalLedDevice based on given device index
func (d *Device) getExternalLedDevice(index int) *ExternalLedDevice {
	for _, externalLedDevice := range externalLedDevices {
		if externalLedDevice.Index == index {
			return &externalLedDevice
		}
	}
	return nil
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	// Reset
	reset := map[int][]byte{}
	var buffer []byte

	// Reset all channels
	color := &rgb.Color{
		Red:        0,
		Green:      0,
		Blue:       0,
		Brightness: 0,
	}

	for _, device := range d.RgbDevices {
		LedChannels := device.LedChannels
		if LedChannels > 0 {
			for i := 0; i < int(LedChannels); i++ {
				reset[i] = []byte{
					byte(color.Red),
					byte(color.Green),
					byte(color.Blue),
				}
			}
		}
	}
	buffer = rgb.SetColor(reset)
	d.writeColor(buffer)

	// Get the number of LED channels we have
	lightChannels := 0
	for _, device := range d.RgbDevices {
		lightChannels += int(device.LedChannels)
	}

	// Do we have any RGB component in the system?
	if lightChannels == 0 {
		logger.Log(logger.Fields{}).Info("No RGB compatible devices found")
		return
	}

	// Are all devices under static mode?
	// In static mode, we only need to send color once;
	// there is no need for continuous packet sending.
	s, l := 0, 0
	for _, device := range d.RgbDevices {
		if device.LedChannels > 0 {
			l++ // device has LED
			if device.RGB == "static" {
				s++ // led profile is set to static
			}
		}
	}
	if s > 0 || l > 0 { // We have some values
		if s == l { // number of devices matches number of devices with static profile
			profile := rgb.GetRgbProfile("static")
			if d.DeviceProfile.Brightness != 0 {
				profile.StartColor.Brightness = rgb.GetBrightnessValue(d.DeviceProfile.Brightness)
			}

			profileColor := rgb.ModifyBrightness(profile.StartColor)
			for i := 0; i < lightChannels; i++ {
				reset[i] = []byte{
					byte(profileColor.Red),
					byte(profileColor.Green),
					byte(profileColor.Blue),
				}
			}
			buffer = rgb.SetColor(reset)
			d.writeColor(buffer) // Write color once
			return
		}
	}

	go func(lightChannels int) {
		lock := sync.Mutex{}
		startTime := time.Now()
		reverse := map[int]bool{}
		counterColorpulse := map[int]int{}
		counterFlickering := map[int]int{}
		counterColorshift := map[int]int{}
		counterCircleshift := map[int]int{}
		counterCircle := map[int]int{}
		counterColorwarp := map[int]int{}
		counterSpinner := map[int]int{}
		counterCpuTemp := map[int]int{}
		counterGpuTemp := map[int]int{}
		counterLiquidTemp := map[int]int{}
		temperatureKeys := map[int]*rgb.Color{}

		colorwarpGeneratedReverse := false
		d.activeRgb = rgb.Exit()

		// Generate random colors
		d.activeRgb.RGBStartColor = rgb.GenerateRandomColor(1)
		d.activeRgb.RGBEndColor = rgb.GenerateRandomColor(1)

		keys := make([]int, 0)
		for k := range d.RgbDevices {
			keys = append(keys, k)
		}
		sort.Ints(keys)

		hue := 1
		wavePosition := 0.0

		for {
			select {
			case <-d.activeRgb.Exit:
				return
			default:
				buff := make([]byte, 0)

				for _, k := range keys {
					if d.RgbDevices[k].IsTemperatureProbe {
						continue
					}

					rgbCustomColor := true
					profile := rgb.GetRgbProfile(d.RgbDevices[k].RGB)
					if profile == nil {
						for i := 0; i < int(d.RgbDevices[k].LedChannels); i++ {
							buff = append(buff, []byte{0, 0, 0}...)
						}
						logger.Log(logger.Fields{"profile": d.RgbDevices[k].RGB, "serial": d.Serial}).Warn("No such RGB profile found")
						continue
					}
					rgbModeSpeed := common.FClamp(profile.Speed, 0.1, 10)
					// Check if we have custom colors
					if (rgb.Color{}) == profile.StartColor || (rgb.Color{}) == profile.EndColor {
						rgbCustomColor = false
					}

					r := rgb.New(
						int(d.RgbDevices[k].LedChannels),
						rgbModeSpeed,
						nil,
						nil,
						profile.Brightness,
						common.Clamp(profile.Smoothness, 1, 100),
						time.Duration(rgbModeSpeed)*time.Second,
						rgbCustomColor,
					)

					if rgbCustomColor {
						r.RGBStartColor = &profile.StartColor
						r.RGBEndColor = &profile.EndColor
					} else {
						r.RGBStartColor = d.activeRgb.RGBStartColor
						r.RGBEndColor = d.activeRgb.RGBEndColor
					}

					// Brightness
					if d.DeviceProfile.Brightness > 0 {
						r.RGBBrightness = rgb.GetBrightnessValue(d.DeviceProfile.Brightness)
						r.RGBStartColor.Brightness = r.RGBBrightness
						r.RGBEndColor.Brightness = r.RGBBrightness
					}

					switch d.RgbDevices[k].RGB {
					case "off":
						{
							for n := 0; n < int(d.RgbDevices[k].LedChannels); n++ {
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
							lock.Lock()
							counterLiquidTemp[k]++
							if counterLiquidTemp[k] >= r.Smoothness {
								counterLiquidTemp[k] = 0
							}

							if _, ok := temperatureKeys[k]; !ok {
								temperatureKeys[k] = r.RGBStartColor
							}

							r.MinTemp = profile.MinTemp
							r.MaxTemp = profile.MaxTemp
							res := r.Temperature(float64(d.getLiquidTemperature()), counterLiquidTemp[k], temperatureKeys[k])
							temperatureKeys[k] = res
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "cpu-temperature":
						{
							lock.Lock()
							counterCpuTemp[k]++
							if counterCpuTemp[k] >= r.Smoothness {
								counterCpuTemp[k] = 0
							}

							if _, ok := temperatureKeys[k]; !ok {
								temperatureKeys[k] = r.RGBStartColor
							}

							r.MinTemp = profile.MinTemp
							r.MaxTemp = profile.MaxTemp
							res := r.Temperature(float64(d.CpuTemp), counterCpuTemp[k], temperatureKeys[k])
							temperatureKeys[k] = res
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "gpu-temperature":
						{
							lock.Lock()
							counterGpuTemp[k]++
							if counterGpuTemp[k] >= r.Smoothness {
								counterGpuTemp[k] = 0
							}

							if _, ok := temperatureKeys[k]; !ok {
								temperatureKeys[k] = r.RGBStartColor
							}

							r.MinTemp = profile.MinTemp
							r.MaxTemp = profile.MaxTemp
							res := r.Temperature(float64(d.GpuTemp), counterGpuTemp[k], temperatureKeys[k])
							temperatureKeys[k] = res
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "colorpulse":
						{
							lock.Lock()
							counterColorpulse[k]++
							if counterColorpulse[k] >= r.Smoothness {
								counterColorpulse[k] = 0
							}

							r.Colorpulse(counterColorpulse[k])
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "static":
						{
							r.Static()
							buff = append(buff, r.Output...)
						}
					case "rotator":
						{
							r.Rotator(hue)
							buff = append(buff, r.Output...)
						}
					case "wave":
						{
							r.Wave(wavePosition)
							buff = append(buff, r.Output...)
						}
					case "storm":
						{
							r.Storm()
							buff = append(buff, r.Output...)
						}
					case "flickering":
						{
							lock.Lock()
							if counterFlickering[k] >= r.Smoothness {
								counterFlickering[k] = 0
							} else {
								counterFlickering[k]++
							}

							r.Flickering(counterFlickering[k])
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "colorshift":
						{
							lock.Lock()
							if counterColorshift[k] >= r.Smoothness && !reverse[k] {
								counterColorshift[k] = 0
								reverse[k] = true
							} else if counterColorshift[k] >= r.Smoothness && reverse[k] {
								counterColorshift[k] = 0
								reverse[k] = false
							}

							r.Colorshift(counterColorshift[k], reverse[k])
							counterColorshift[k]++
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "circleshift":
						{
							lock.Lock()
							if counterCircleshift[k] >= int(d.RgbDevices[k].LedChannels) {
								counterCircleshift[k] = 0
							} else {
								counterCircleshift[k]++
							}

							r.Circle(counterCircleshift[k])
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "circle":
						{
							lock.Lock()
							if counterCircle[k] >= int(d.RgbDevices[k].LedChannels) {
								counterCircle[k] = 0
							} else {
								counterCircle[k]++
							}

							r.Circle(counterCircle[k])
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "spinner":
						{
							lock.Lock()
							if counterSpinner[k] >= int(d.RgbDevices[k].LedChannels) {
								counterSpinner[k] = 0
							} else {
								counterSpinner[k]++
							}
							r.Spinner(counterSpinner[k])
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "colorwarp":
						{
							lock.Lock()
							if counterColorwarp[k] >= r.Smoothness {
								if !colorwarpGeneratedReverse {
									colorwarpGeneratedReverse = true
									d.activeRgb.RGBStartColor = d.activeRgb.RGBEndColor
									d.activeRgb.RGBEndColor = rgb.GenerateRandomColor(r.RGBBrightness)
								}
								counterColorwarp[k] = 0
							} else if counterColorwarp[k] == 0 && colorwarpGeneratedReverse == true {
								colorwarpGeneratedReverse = false
							} else {
								counterColorwarp[k]++
							}

							r.Colorwarp(counterColorwarp[k], d.activeRgb.RGBStartColor, d.activeRgb.RGBEndColor)
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					}
				}

				// Send it
				d.writeColor(buff)
				time.Sleep(20 * time.Millisecond)
				hue++
				wavePosition += 0.2
			}
		}
	}(lightChannels)
}

// getRgbDevices will get all RGB devices
func (d *Device) getRgbDevices() {
	var devices = make(map[int]*Devices, 0)
	var m = 0
	amount := 7

	for i := 0; i < amount; i++ {
		if internalLedDevice, ok := internalLedDevices[i]; ok {
			if internalLedDevice.Total > 0 {
				rgbProfile := "static"
				label := "Set Label"
				if d.DeviceProfile != nil {
					// Profile is set
					if rp, ok := d.DeviceProfile.RGBProfiles[i]; ok {
						// Profile device channel exists
						if rgb.GetRgbProfile(rp) != nil { // Speed profile exists in configuration
							// Speed profile exists in configuration
							rgbProfile = rp
						} else {
							logger.Log(logger.Fields{"serial": d.Serial, "profile": rp}).Warn("Tried to apply non-existing rgb profile")
						}
					} else {
						logger.Log(logger.Fields{"serial": d.Serial, "profile": rp}).Warn("Tried to apply rgb profile to the non-existing channel")
					}

					// Device label
					if lb, ok := d.DeviceProfile.RGBLabels[i]; ok {
						if len(lb) > 0 {
							label = lb
						}
					}
				} else {
					logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
				}

				name := internalLedDevice.Name
				containsPump := false
				if i == 0 {
					name = d.Devices[i].Name
					containsPump = true
				}
				// Build device object
				device := &Devices{
					ChannelId:    i,
					DeviceId:     fmt.Sprintf("%s-%v", "Fan", i),
					Name:         name,
					Rpm:          0,
					Temperature:  0,
					Description:  "LED",
					LedChannels:  internalLedDevice.Total,
					HubId:        d.Serial,
					Profile:      "",
					Label:        label,
					RGB:          rgbProfile,
					HasSpeed:     false,
					HasTemps:     false,
					ContainsPump: containsPump,
				}
				devices[m] = device
			} else {
				if d.DeviceProfile != nil {
					if deviceType, ok := d.DeviceProfile.CustomLEDs[i]; ok {
						if deviceType > 0 {
							rgbProfile := "static"
							label := "Set Label"
							if d.DeviceProfile != nil {
								// Profile is set
								if rp, ok := d.DeviceProfile.RGBProfiles[i]; ok {
									// Profile device channel exists
									if rgb.GetRgbProfile(rp) != nil { // Speed profile exists in configuration
										// Speed profile exists in configuration
										rgbProfile = rp
									} else {
										logger.Log(logger.Fields{"serial": d.Serial, "profile": rp}).Warn("Tried to apply non-existing rgb profile")
									}
								} else {
									logger.Log(logger.Fields{"serial": d.Serial, "profile": rp}).Warn("Tried to apply rgb profile to the non-existing channel")
								}

								// Device label
								if lb, ok := d.DeviceProfile.RGBLabels[i]; ok {
									if len(lb) > 0 {
										label = lb
									}
								}
							} else {
								logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
							}

							ledChannels := 0
							name := ""
							externalDeviceType := d.getExternalLedDevice(d.DeviceProfile.CustomLEDs[i])
							if externalDeviceType != nil {
								ledChannels = externalDeviceType.Total
								name = externalDeviceType.Name
							}

							// Build device object
							device := &Devices{
								ChannelId:   i,
								DeviceId:    fmt.Sprintf("%s-%v", "Fan", i),
								Name:        name,
								Rpm:         0,
								Temperature: 0,
								Description: "LED",
								LedChannels: uint8(ledChannels),
								HubId:       d.Serial,
								Profile:     "",
								Label:       label,
								RGB:         rgbProfile,
								HasSpeed:    false,
								HasTemps:    false,
							}
							devices[m] = device
						}
					}
				}
			}
		}
		m++
	}
	d.RgbDevices = devices
}

// getDevices will fetch all devices connected to a hub
func (d *Device) getDevices() int {
	var devices = make(map[int]*Devices, 0)
	var m = 0

	// Fans
	response := d.read(modeGetFans, dataTypeGetFans)
	amount := d.getChannelAmount(response)
	if d.Debug {
		logger.Log(logger.Fields{"serial": d.Serial, "data": fmt.Sprintf("%2x", response), "amount": amount, "device": d.Product}).Info("getDevices() - Speed")
	}

	for i := 0; i < amount; i++ {
		status := response[6:][i]
		if d.Debug {
			logger.Log(logger.Fields{"serial": d.Serial, "status": status, "device": d.Product, "channel": i}).Info("getDevices() - Speed")
		}
		if status == 0x07 {
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

			// Build device object
			device := &Devices{
				ChannelId:   i,
				DeviceId:    fmt.Sprintf("%s-%v", "Fan", i),
				Name:        fmt.Sprintf("Fan %d", i),
				Rpm:         0,
				Temperature: 0,
				Description: "Fan",
				HubId:       d.Serial,
				Profile:     speedProfile,
				Label:       label,
				HasSpeed:    true,
				HasTemps:    false,
			}

			if i == 0 { // Pump is on channel 0
				device.HasSpeed = true
				device.HasTemps = true
				device.Description = "AIO"
				device.Name = d.AIOType
				device.DeviceId = fmt.Sprintf("%s-%v", "AIO", i)
				device.ContainsPump = true
			}
			devices[m] = device
		}
		m++
	}

	// Temperature probe
	response = d.read(modeGetTemperatures, dataTypeGetTemperatures)
	sensorData := response[9:]
	if d.Debug {
		logger.Log(logger.Fields{"serial": d.Serial, "data": fmt.Sprintf("%2x", response)}).Info("getDevices() - Temperature")
	}
	for i, s := 0, 0; i < 1; i, s = i+1, s+3 {
		label := "Set Label"
		status := sensorData[s : s+3][0]
		if d.Debug {
			logger.Log(logger.Fields{"serial": d.Serial, "status": status, "device": d.Product, "channel": i}).Info("getDevices() - Temperature")
		}
		if status == 0x00 {
			if d.DeviceProfile != nil {
				// Device label
				if lb, ok := d.DeviceProfile.Labels[m]; ok {
					if len(lb) > 0 {
						label = lb
					}
				}
			}

			// Build device object
			device := &Devices{
				ChannelId:          m,
				DeviceId:           fmt.Sprintf("%s-%v", "Probe", i),
				Name:               fmt.Sprintf("Temperature Probe %d", i),
				Rpm:                0,
				Temperature:        0,
				Description:        "Probe",
				HubId:              d.Serial,
				HasSpeed:           false,
				HasTemps:           true,
				IsTemperatureProbe: true,
				Label:              label,
			}
			devices[m] = device
		}
		m++
	}

	d.Devices = devices
	return len(devices)
}

// setSpeed will generate a speed buffer and send it to a device
func (d *Device) setSpeed(data map[int]byte, mode uint8) {
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

	response := d.write(modeSetSpeed, dataTypeSetSpeed, buffer, true)
	if len(response) >= 4 {
		if response[5] != 0x07 {
			m := 0
			for {
				m++
				response = d.write(modeSetSpeed, dataTypeSetSpeed, buffer, true)
				if response[5] == 0x07 || m > 20 {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// ResetSpeedProfiles will reset channel speed profile if it matches with the current speed profile
// This is used when speed profile is deleted from the UI
func (d *Device) ResetSpeedProfiles(profile string) {
	mutex.Lock()
	defer mutex.Unlock()

	i := 0
	for _, device := range d.Devices {
		if device.HasSpeed {
			if device.Profile == profile {
				d.Devices[device.ChannelId].Profile = "Normal"
				i++
			}
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

// getLiquidTemperature will fetch temperature from AIO device
func (d *Device) getLiquidTemperature() float32 {
	for _, device := range d.Devices {
		if device.ContainsPump {
			return device.Temperature
		}
	}
	return 0
}

// updateDeviceSpeed will update device speed based on a temperature reading
func (d *Device) updateDeviceSpeed() {
	timerSpeed = time.NewTicker(time.Duration(temperaturePullingInterval) * time.Millisecond)
	go func() {
		tmp := make(map[int]string, 0)
		channelSpeeds := map[int]byte{}

		// Init speed channels
		for _, device := range d.Devices {
			if device.IsTemperatureProbe {
				continue
			}
			channelSpeeds[device.ChannelId] = byte(defaultSpeedValue)
		}
		for {
			select {
			case <-timerSpeed.C:
				var temp float32 = 0
				for _, device := range d.Devices {
					if device.IsTemperatureProbe {
						continue
					}

					profiles := temperatures.GetTemperatureProfile(device.Profile)
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
									logger.Log(logger.Fields{"temperature": temp}).Warn("Unable to get sensor temperature. Going to fallback to CPU")
									temp = temperatures.GetCpuTemperature()
								}
							}
						}
					case temperatures.SensorTypeCPU:
						{
							temp = temperatures.GetCpuTemperature()
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
							if d.Devices[profiles.ChannelId].IsTemperatureProbe {
								temp = d.Devices[profiles.ChannelId].Temperature
							}
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "channelId": profiles.ChannelId}).Warn("Unable to get probe temperature.")
							}
						}
					}

					// All temps failed, default to 50
					if temp == 0 {
						temp = 50
					}
					for i := 0; i < len(profiles.Profiles); i++ {
						profile := profiles.Profiles[i]
						minimum := profile.Min + 0.1
						if common.InBetween(temp, minimum, profile.Max) {
							cp := fmt.Sprintf("%s-%d-%d-%d-%d", device.Profile, device.ChannelId, profile.Id, profile.Fans, profile.Pump)
							if ok := tmp[device.ChannelId]; ok != cp {
								tmp[device.ChannelId] = cp

								// Validation
								if profile.Mode < 0 || profile.Mode > 1 {
									profile.Mode = 0
								}

								if profile.Pump < 50 {
									profile.Pump = 50
								}

								if profile.Pump > 100 {
									profile.Pump = 100
								}

								if device.ContainsPump {
									channelSpeeds[device.ChannelId] = byte(profile.Pump)
								} else {
									channelSpeeds[device.ChannelId] = byte(profile.Fans)
								}
								d.setSpeed(channelSpeeds, 0)
							}
						}
					}
				}
			case <-speedRefreshChan:
				timerSpeed.Stop()
				return
			}
		}
	}()
}

// setHardwareMode will switch a device to hardware mode
func (d *Device) setHardwareMode() {
	_, err := d.transfer(cmdHardwareMode, nil, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	_, err := d.transfer(cmdSoftwareMode, nil, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// getDeviceType will set a type of AIO
func (d *Device) getDeviceType() {
	deviceType, err := d.transfer(cmdGetPumpVersion, nil, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
	}
	pumpVersion := int16(deviceType[3])

	deviceType, err = d.transfer(cmdGetRadiatorType, nil, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
	}
	radiatorSize := int16(binary.LittleEndian.Uint16(deviceType[3:5]))

	// We match a device with radiator size and pump version
	for _, aioType := range aioList {
		if aioType.RadiatorSize == radiatorSize && aioType.PumpVersion == pumpVersion && aioType.LCD == d.HasLCD {
			d.AIOType = aioType.Name
			break
		}
	}

	if len(d.AIOType) == 0 {
		d.AIOType = "Unknown AIO"
	}
}

// setColorEndpoint will activate hub color endpoint for further usage
func (d *Device) setColorEndpoint() {
	// Close any RGB endpoint
	_, err := d.transfer(cmdCloseEndpoint, modeSetColor, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
	}

	// Open RGB endpoint
	_, err = d.transfer(cmdOpenColorEndpoint, modeSetColor, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to open endpoint")
	}
}

// getDeviceProfile will load persistent device configuration
func (d *Device) getDeviceProfile() {
	if len(d.UserProfiles) == 0 {
		logger.Log(logger.Fields{"serial": d.Serial}).Warn("No profile found for device. Probably initial start")
	} else {
		for _, pf := range d.UserProfiles {
			if pf.Active {
				d.DeviceProfile = pf
			}
		}
	}
}

// getChannelAmount will return a number of available channels
func (d *Device) getChannelAmount(data []byte) int {
	return int(data[5])
}

// getDeviceData will fetch device data
func (d *Device) getDeviceData() {
	// Channels
	channels := d.read(modeGetFans, dataTypeGetFans)
	if channels == nil {
		return
	}
	var m = 0

	// Speed
	response := d.read(modeGetSpeeds, dataTypeGetSpeeds)
	amount := d.getChannelAmount(channels)
	sensorData := response[6:]

	if d.Debug {
		logger.Log(logger.Fields{"serial": d.Serial, "data": fmt.Sprintf("%2x", response), "amount": amount, "device": d.Product}).Info("getDeviceData() - Speed")
	}

	for i, s := 0, 0; i < amount; i, s = i+1, s+2 {
		currentSensor := sensorData[s : s+2]
		status := channels[6:][i]
		if status == 0x07 {
			if _, ok := d.Devices[m]; ok {
				rpm := int16(binary.LittleEndian.Uint16(currentSensor))
				if rpm > 20 {
					d.Devices[m].Rpm = rpm
				}
			}
		}
		m++
	}

	// Temperature
	response = d.read(modeGetTemperatures, dataTypeGetTemperatures)
	amount = d.getChannelAmount(response)
	sensorData = response[6:]
	for i, s := 0, 0; i < amount; i, s = i+1, s+3 {
		currentSensor := sensorData[s : s+3]
		status := currentSensor[0]
		if status == 0x00 {
			temp := float32(int16(binary.LittleEndian.Uint16(currentSensor[1:3]))) / 10.0
			if temp > 10 && temp < 100 {
				if i == 0 {
					if _, ok := d.Devices[i]; ok {
						d.Devices[i].Temperature = float32(int16(binary.LittleEndian.Uint16(currentSensor[1:3]))) / 10.0
						d.Devices[i].TemperatureString = dashboard.GetDashboard().TemperatureToString(temp)
					}
				} else {
					if _, ok := d.Devices[m]; ok {
						d.Devices[m].Temperature = float32(int16(binary.LittleEndian.Uint16(currentSensor[1:3]))) / 10.0
						d.Devices[m].TemperatureString = dashboard.GetDashboard().TemperatureToString(temp)
					}
					m++
				}
			}
		}
	}
}

// setDefaults will set default mode for all devices
func (d *Device) setDefaults() {
	channelDefaults := map[int]byte{}
	for device := range d.Devices {
		if d.Devices[device].HasSpeed {
			channelDefaults[device] = byte(defaultSpeedValue)
		}
	}
	d.setSpeed(channelDefaults, 0)
}

// setCpuTemperature will store current CPU temperature
func (d *Device) setTemperatures() {
	d.CpuTemp = temperatures.GetCpuTemperature()
	d.GpuTemp = temperatures.GetGpuTemperature()
}

// setAutoRefresh will refresh device data
func (d *Device) setAutoRefresh() {
	timer = time.NewTicker(time.Duration(deviceRefreshInterval) * time.Millisecond)
	authRefreshChan = make(chan bool)
	go func() {
		for {
			select {
			case <-timer.C:
				d.setTemperatures()
				d.setDeviceStatus()
				d.getDeviceData()
			case <-authRefreshChan:
				timer.Stop()
				return
			}
		}
	}()
}

// UpdateDeviceSpeed will update device channel speed.
func (d *Device) UpdateDeviceSpeed(channelId int, value uint16) uint8 {
	// Check if actual channelId exists in the device list
	if device, ok := d.Devices[channelId]; ok {
		if device.IsTemperatureProbe {
			return 0
		}
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
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := d.Devices[channelId]; !ok {
		return 0
	}

	d.Devices[channelId].Label = label
	d.saveDeviceProfile()
	return 1
}

// UpdateRGBDeviceLabel will set / update device label
func (d *Device) UpdateRGBDeviceLabel(channelId int, label string) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := d.RgbDevices[channelId]; !ok {
		return 0
	}

	d.RgbDevices[channelId].Label = label
	d.saveDeviceProfile()
	return 1
}

// UpdateDeviceLcd will update device LCD
func (d *Device) UpdateDeviceLcd(mode uint8) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	if d.HasLCD {
		d.DeviceProfile.LCDMode = mode
		d.saveDeviceProfile()
		return 1
	} else {
		return 2
	}
}

// UpdateDeviceLcdRotation will update device LCD rotation
func (d *Device) UpdateDeviceLcdRotation(rotation uint8) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	if d.HasLCD {
		d.DeviceProfile.LCDRotation = rotation
		d.saveDeviceProfile()
		return 1
	} else {
		return 2
	}
}

// ChangeDeviceBrightness will change device brightness
func (d *Device) ChangeDeviceBrightness(mode uint8) uint8 {
	d.DeviceProfile.Brightness = mode
	d.saveDeviceProfile()
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
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

		for _, device := range d.RgbDevices {
			if device.LedChannels > 0 {
				d.RgbDevices[device.ChannelId].RGB = profile.RGBProfiles[device.ChannelId]
			}
			d.RgbDevices[device.ChannelId].Label = profile.RGBLabels[device.ChannelId]
		}

		for _, device := range d.Devices {
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
			timerSpeed.Stop()
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
	mutex.Lock()
	defer mutex.Unlock()

	// Check if the profile exists
	profiles := temperatures.GetTemperatureProfile(profile)
	if profiles == nil {
		return 0
	}

	// If the profile is liquid temperature, check for the presence of AIOs
	if profiles.Sensor == temperatures.SensorTypeLiquidTemperature {
		valid := false
		for _, device := range d.Devices {
			if device.ContainsPump {
				valid = true
				break
			}
		}

		if !valid {
			return 2
		}
	}

	if profiles.Sensor == temperatures.SensorTypeTemperatureProbe {
		if profiles.Device != d.Serial {
			return 3
		}

		if _, ok := d.Devices[profiles.ChannelId]; !ok {
			return 4
		}
	}

	if channelId < 0 {
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

// UpdateRgbProfile will update device RGB profile
func (d *Device) UpdateRgbProfile(channelId int, profile string) uint8 {
	if rgb.GetRgbProfile(profile) == nil {
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
		for _, device := range d.RgbDevices {
			if device.LedChannels > 0 {
				d.DeviceProfile.RGBProfiles[device.ChannelId] = profile
				d.RgbDevices[device.ChannelId].RGB = profile
			}
		}
	} else {
		if _, ok := d.RgbDevices[channelId]; ok {
			d.DeviceProfile.RGBProfiles[channelId] = profile // Set profile
			d.RgbDevices[channelId].RGB = profile
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

// UpdateARGBDevice will update or create a new device with ARGB 3-pin support
func (d *Device) UpdateARGBDevice(portId, deviceType int) uint8 {
	if portId < 1 || portId > 6 {
		return 0
	}

	if _, ok := d.FreeLedPorts[portId]; ok {
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true // Exit current RGB mode
			d.activeRgb = nil
		}
		d.DeviceProfile.CustomLEDs[portId] = deviceType
		if deviceType == 0 {
			delete(d.RgbDevices, portId)
		} else {
			externalLedDevice := d.getExternalLedDevice(deviceType)
			if externalLedDevice != nil {
				ledChannels := externalLedDevice.Total
				name := externalLedDevice.Name

				// Build device object
				device := &Devices{
					ChannelId:   portId,
					DeviceId:    fmt.Sprintf("%s-%v", "Fan", portId),
					Name:        name,
					Rpm:         0,
					Temperature: 0,
					Description: "RGB Device",
					LedChannels: uint8(ledChannels),
					HubId:       d.Serial,
					Profile:     "",
					Label:       "Set Label",
					RGB:         "static",
					HasSpeed:    false,
					HasTemps:    false,
				}
				d.RgbDevices[portId] = device
			}
		}

		d.resetLEDPorts()     // Reset LED ports
		d.saveDeviceProfile() // Save profile
		d.setDeviceColor()    // Restart RGB
		return 1
	} else {
		return 2 // No such free port
	}
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
			AIO:              strconv.FormatBool(device.ContainsPump),
			ContainsPump:     strconv.FormatBool(device.ContainsPump),
			Temperature:      float64(device.Temperature),
			LedChannels:      strconv.Itoa(int(device.LedChannels)),
			Rpm:              device.Rpm,
			TemperatureProbe: strconv.FormatBool(device.IsTemperatureProbe),
		}
		metrics.Populate(header)
	}
}

// initLedPorts will prep LED physical ports for reading
func (d *Device) initLedPorts() {
	for i := 0; i <= 6; i++ {
		var command = []byte{0x14, byte(i), 0x01}
		_, err := d.transfer(command, nil, nil)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "port": i}).Error("Failed to initialize LED ports")
		}
	}

	// We need to wait around 500 ms for physical ports to re-initialize
	// After that we can grab any new connected / disconnected device values
	time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
}

// resetLEDPorts will reset hub LED ports and configure currently connected LED device
func (d *Device) resetLEDPorts() {
	var buf []byte

	buf = append(buf, 0x0d)
	buf = append(buf, 0x00)
	buf = append(buf, 0x07)

	// Start at 1, since 0 is the pump, and iterate through all 6 physical connectors
	for i := 0; i <= 6; i++ {
		if z, ok := internalLedDevices[i]; ok {
			if z.Total > 0 {
				// Channel activation
				buf = append(buf, 0x01)
				// Fan LED command code, each LED device has different command code
				buf = append(buf, z.Command)
			} else {
				if deviceType, valid := d.DeviceProfile.CustomLEDs[i]; valid {
					if deviceType > 0 {
						externalDeviceType := d.getExternalLedDevice(d.DeviceProfile.CustomLEDs[i])
						// Channel activation
						buf = append(buf, 0x01)
						buf = append(buf, externalDeviceType.Command)
					} else {
						// Port is not configured for ARGB
						buf = append(buf, 0x00)
					}
				} else {
					// Empty, disable port
					buf = append(buf, 0x00)
				}
			}
		} else {
			// Channel is not active
			buf = append(buf, 0x00)
		}
	}
	d.write(cmdSetLedPorts, nil, buf, false)
}

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"

	speedProfiles := make(map[int]string, len(d.Devices))
	rgbProfiles := make(map[int]string, len(d.Devices))
	labels := make(map[int]string, len(d.Devices))
	rgbLabels := make(map[int]string, len(d.Devices))
	customLEDs := make(map[int]int, len(d.Devices))

	for _, device := range d.Devices {
		if device.IsTemperatureProbe {
			continue
		}
		speedProfiles[device.ChannelId] = device.Profile
	}

	for _, device := range d.RgbDevices {
		if device.LedChannels > 0 {
			rgbProfiles[device.ChannelId] = device.RGB
		}
		rgbLabels[device.ChannelId] = device.Label
	}

	for _, device := range d.Devices {
		labels[device.ChannelId] = device.Label
	}

	deviceProfile := &DeviceProfile{
		Product:       d.Product,
		Serial:        d.Serial,
		SpeedProfiles: speedProfiles,
		RGBProfiles:   rgbProfiles,
		Labels:        labels,
		RGBLabels:     rgbLabels,
		Path:          profilePath,
	}

	// First save, assign saved profile to a device
	if d.DeviceProfile == nil {
		// RGB
		for _, device := range d.RgbDevices {
			if device.LedChannels > 0 {
				rgbProfiles[device.ChannelId] = "static"
			}
			rgbLabels[device.ChannelId] = "Set Label"
		}

		for i := 1; i < 7; i++ {
			customLEDs[i] = 0
		}
		deviceProfile.CustomLEDs = customLEDs

		// Labels
		for _, device := range d.Devices {
			labels[device.ChannelId] = "Set Label"
		}

		// LCD
		if d.HasLCD {
			deviceProfile.LCDMode = 0
			deviceProfile.LCDRotation = 0
		}

		deviceProfile.CustomLEDs = customLEDs
		deviceProfile.Active = true
		d.DeviceProfile = deviceProfile
	} else {
		if d.DeviceProfile.CustomLEDs == nil {
			for i := 1; i < 7; i++ {
				customLEDs[i] = 0
			}
			deviceProfile.CustomLEDs = customLEDs
		} else {
			deviceProfile.CustomLEDs = d.DeviceProfile.CustomLEDs
		}

		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.Brightness = d.DeviceProfile.Brightness
		if len(d.DeviceProfile.Path) < 1 {
			deviceProfile.Path = profilePath
			d.DeviceProfile.Path = profilePath
		} else {
			deviceProfile.Path = d.DeviceProfile.Path
		}
		deviceProfile.LCDMode = d.DeviceProfile.LCDMode
		deviceProfile.LCDRotation = d.DeviceProfile.LCDRotation
	}

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
		logger.Log(logger.Fields{"error": err, "location": deviceProfile.Path}).Fatal("Unable to close file handle")
	}

	d.loadDeviceProfiles() // Reload
}

// newDeviceMonitor initializes and returns a new Monitor
func (d *Device) newDeviceMonitor() {
	m := &DeviceMonitor{}
	m.Cond = sync.NewCond(&m.Lock)
	go d.waitForDevice(func() {
		// Device woke up after machine was sleeping
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true
			d.activeRgb = nil
		}
		d.setSoftwareMode()  // Activate software mode
		d.setColorEndpoint() // Set device color endpoint
		d.setDeviceColor()   // Set RGB
		if !config.GetConfig().Manual {
			timerSpeed.Stop()
			d.updateDeviceSpeed() // Update device speed
		}
		d.newDeviceMonitor() // Device monitor
	})
	d.deviceMonitor = m
}

// setDeviceStatus sets the status and notifies a waiting goroutine if necessary
func (d *Device) setDeviceStatus() {
	mode, err := d.transfer(cmdGetDeviceMode, nil, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
	}
	if len(mode) == 0 {
		return
	}

	d.deviceMonitor.Lock.Lock()
	defer d.deviceMonitor.Lock.Unlock()
	d.deviceMonitor.Status = mode[3]
	d.deviceMonitor.Cond.Broadcast()
}

// waitForDevice waits for the status to change from zero to one and back to zero before running the action
func (d *Device) waitForDevice(action func()) {
	d.deviceMonitor.Lock.Lock()
	for d.deviceMonitor.Status != 2 {
		d.deviceMonitor.Cond.Wait()
	}
	d.deviceMonitor.Lock.Unlock()

	d.deviceMonitor.Lock.Lock()
	for d.deviceMonitor.Status != 1 {
		d.deviceMonitor.Cond.Wait()
	}
	d.deviceMonitor.Lock.Unlock()
	action()
}

// read will read data from a device and return data as a byte array
func (d *Device) read(endpoint, bufferType []byte) []byte {
	// Endpoint data
	var buffer []byte

	// Close specified endpoint
	_, err := d.transfer(cmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to close endpoint")
	}

	// Open endpoint
	_, err = d.transfer(cmdOpenEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to open endpoint")
	}

	// Read data from endpoint
	buffer, err = d.transfer(cmdRead, endpoint, bufferType)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read endpoint")
	}

	// Close specified endpoint
	_, err = d.transfer(cmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to close endpoint")
	}
	return buffer
}

// writeColor will write data to the device with a specific endpoint.
// writeColor does not require endpoint closing and opening like normal Write requires.
// Endpoint is open only once. Once the endpoint is open, color can be sent continuously.
func (d *Device) writeColor(data []byte) {
	buffer := make([]byte, len(dataTypeSetColor)+len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)+2))
	copy(buffer[headerWriteSize:headerWriteSize+len(dataTypeSetColor)], dataTypeSetColor)
	copy(buffer[headerWriteSize+len(dataTypeSetColor):], data)

	// Split packet into chunks
	chunks := common.ProcessMultiChunkPacket(buffer, maxBufferSizePerRequest)
	for i, chunk := range chunks {
		if i == 0 {
			// Initial packet is using cmdWriteColor
			_, err := d.transfer(cmdWriteColor, chunk, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
			}
		} else {
			// Chunks don't use cmdWriteColor, they use static dataTypeSubColor
			_, err := d.transfer(dataTypeSubColor, chunk, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to endpoint")
			}
		}
	}
}

// write will write data to the device with specific endpoint
func (d *Device) write(endpoint, bufferType, data []byte, extra bool) []byte {
	// Buffer
	buffer := make([]byte, len(bufferType)+len(data)+headerWriteSize)
	if extra {
		binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)+2))
	} else {
		binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	}
	copy(buffer[headerWriteSize:headerWriteSize+len(bufferType)], bufferType)
	copy(buffer[headerWriteSize+len(bufferType):], data)

	// Create read buffer
	bufferR := make([]byte, bufferSize)

	// Close endpoint
	_, err := d.transfer(cmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to close endpoint")
	}

	// Open endpoint
	_, err = d.transfer(cmdOpenEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to open endpoint")
	}

	// Send it
	bufferR, err = d.transfer(cmdWrite, buffer, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to endpoint")
	}

	// Close endpoint
	_, err = d.transfer(cmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to close endpoint")
	}

	return bufferR
}

// getLCDRotation will return rotation value based on rotation mode
func (d *Device) getLCDRotation() int {
	switch d.DeviceProfile.LCDRotation {
	case 0:
		return 0
	case 1:
		return 90
	case 2:
		return 180
	case 3:
		return 270
	}
	return 0
}

// setupLCD will activate and configure LCD
func (d *Device) setupLCD() {
	lcdTimer = time.NewTicker(time.Duration(lcdRefreshInterval) * time.Millisecond)
	lcdRefreshChan = make(chan bool)
	go func() {
		for {
			select {
			case <-lcdTimer.C:
				switch d.DeviceProfile.LCDMode {
				case lcd.DisplayCPU:
					{
						buffer := lcd.GenerateScreenImage(
							lcd.DisplayCPU,
							int(temperatures.GetCpuTemperature()),
							0,
							0,
							0,
							d.getLCDRotation(),
						)
						d.transferToLcd(buffer)
					}
				case lcd.DisplayGPU:
					{
						buffer := lcd.GenerateScreenImage(
							lcd.DisplayGPU,
							int(temperatures.GetGpuTemperature()),
							0,
							0,
							0,
							d.getLCDRotation(),
						)
						d.transferToLcd(buffer)
					}
				case lcd.DisplayLiquid:
					{
						for _, device := range d.Devices {
							if device.ContainsPump {
								buffer := lcd.GenerateScreenImage(
									lcd.DisplayLiquid,
									int(device.Temperature),
									0,
									0,
									0,
									d.getLCDRotation(),
								)
								d.transferToLcd(buffer)
							}
						}
					}
				case lcd.DisplayPump:
					{
						for _, device := range d.Devices {
							if device.ContainsPump {
								buffer := lcd.GenerateScreenImage(
									lcd.DisplayPump,
									int(device.Rpm),
									0,
									0,
									0,
									d.getLCDRotation(),
								)
								d.transferToLcd(buffer)
							}
						}
					}
				case lcd.DisplayAllInOne:
					{
						liquidTemp := 0
						cpuTemp := 0
						pumpSpeed := 0
						for _, device := range d.Devices {
							if device.ContainsPump {
								liquidTemp = int(device.Temperature)
								pumpSpeed = int(device.Rpm)
							}
						}

						cpuTemp = int(temperatures.GetCpuTemperature())
						buffer := lcd.GenerateScreenImage(
							lcd.DisplayAllInOne,
							liquidTemp,
							cpuTemp,
							pumpSpeed,
							0,
							d.getLCDRotation(),
						)
						d.transferToLcd(buffer)
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
							d.getLCDRotation(),
						)
						d.transferToLcd(buffer)
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
							d.getLCDRotation(),
						)
						d.transferToLcd(buffer)
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
							d.getLCDRotation(),
						)
						d.transferToLcd(buffer)
					}
				}
			case <-lcdRefreshChan:
				lcdTimer.Stop()
				return
			}
		}
	}()
}

// transferToLcd will transfer data to LCD panel
func (d *Device) transferToLcd(buffer []byte) {
	mutexLcd.Lock()
	defer mutexLcd.Unlock()
	chunks := common.ProcessMultiChunkPacket(buffer, maxLCDBufferSizePerRequest)
	for i, chunk := range chunks {
		bufferW := make([]byte, lcdBufferSize)
		bufferW[0] = 0x02
		bufferW[1] = 0x05

		// The last packet needs to end with 0x01 in order for display to render data
		if len(chunk) < maxLCDBufferSizePerRequest {
			bufferW[3] = 0x01
		}

		bufferW[4] = byte(i)
		binary.LittleEndian.PutUint16(bufferW[6:8], uint16(len(chunk)))
		copy(bufferW[8:], chunk)

		if _, err := d.lcd.Write(bufferW); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
			break
		}
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint, buffer, bufferType []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	mutex.Lock()
	defer mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = 0x08
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
	if _, err := d.dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		return nil, err
	}

	// Read remaining data from a device
	if len(bufferType) == 2 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(transferTimeout)*time.Millisecond)
		defer cancel()

		for ctx.Err() != nil && !responseMatch(bufferR, bufferType) {
			if _, err := d.dev.Read(bufferR); err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
				return nil, err
			}
		}
		if ctx.Err() != nil {
			logger.Log(logger.Fields{"error": ctx.Err(), "serial": d.Serial}).Error("Unable to read data from device")
			return nil, ctx.Err()
		}
	}

	return bufferR, nil
}

// responseMatch will check if two byte arrays match
func responseMatch(response, expected []byte) bool {
	responseBuffer := response[4:6]
	return bytes.Equal(responseBuffer, expected)
}
