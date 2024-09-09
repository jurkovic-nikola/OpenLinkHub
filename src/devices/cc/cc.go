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
	"OpenLinkHub/src/devices/lcd"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/rgb"
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
	pwd, _                     = os.Getwd()
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
)

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
}

// DeviceProfile struct contains all device profile
type DeviceProfile struct {
	Active        bool
	Path          string
	Product       string
	Serial        string
	LCDMode       uint8
	Brightness    uint8
	RGBProfiles   map[int]string
	SpeedProfiles map[int]string
	Labels        map[int]string
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
	dev           *hid.Device
	lcd           *hid.Device
	Manufacturer  string                    `json:"manufacturer"`
	Product       string                    `json:"product"`
	Serial        string                    `json:"serial"`
	Firmware      string                    `json:"firmware"`
	AIOType       string                    `json:"-"`
	Devices       map[int]*Devices          `json:"devices"`
	UserProfiles  map[string]*DeviceProfile `json:"userProfiles"`
	DeviceProfile *DeviceProfile
	deviceMonitor *DeviceMonitor
	activeRgb     *rgb.ActiveRGB
	Template      string
	HasLCD        bool
	VendorId      uint16
	LCDModes      map[int]string
	Brightness    map[int]string
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
	// Open device, return if failure
	dev, err := hid.Open(vendorId, productId, serial)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId}).Error("Unable to open HID device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:      dev,
		Template: "cc.html",
		VendorId: vendorId,
		LCDModes: map[int]string{
			0: "Liquid Temperature",
			1: "Pump Speed",
			2: "CPU Temperature",
			3: "GPU Temperature",
			4: "Combined",
		},
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
	}

	// There are 2 CCs. One has a packet size of 64 and the other has 96.
	// This matters only for RGB operations due to packet chunking.
	if productId == 3100 { // 0c1c
		bufferSize = 96
		bufferSizeWrite = bufferSize + 1
		maxBufferSizePerRequest = 93
	}

	// Bootstrap
	d.getManufacturer()    // Manufacturer
	d.getProduct()         // Product
	d.getSerial()          // Serial
	d.loadDeviceProfiles() // Load all device profiles
	d.getDeviceLcd()       // Check if LCD pump cover is installed
	d.getDeviceProfile()   // Get device profile if any
	d.getDeviceFirmware()  // Firmware
	d.setSoftwareMode()    // Activate software mode
	d.initLedPorts()       // Init LED ports
	d.getDeviceType()      // Find an AIO device type
	d.getLedDevices()      // Get LED devices
	d.getDevices()         // Get devices connected to a hub
	d.setColorEndpoint()   // Set device color endpoint
	d.setDefaults()        // Set default speed and color values for fans and pumps
	d.setAutoRefresh()     // Set auto device refresh
	d.saveDeviceProfile()  // Save
	d.resetLEDPorts()      // Reset device LED
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

		// Initialize LED channel data
		leds := &LedChannel{
			Total:   0,
			Command: 00,
		}

		// Check if device status is 2, aka connected
		connected := ld[m] == 2

		if connected {
			// Get number of LEDs
			numLEDs = binary.LittleEndian.Uint16(ld[m+2 : m+2+2])

			// Each LED device has different command code
			switch numLEDs {
			case 4:
				{
					command = 03
				}
			case 8:
				{
					command = 05
				}
			case 10:
				{
					command = 01
				}
			case 12:
				{
					command = 04
				}
			case 16:
				{
					command = 02
				}
			case 24:
				{
					// Pump, no command codes here
				}
			case 29:
				{
					// Pump, no command codes here
				}
			case 34:
				{
					command = 06
				}
			}

			// Set values
			leds.Total = uint8(numLEDs)
			leds.Command = command
		}

		// Add to a device map
		internalLedDevices[i] = leds
		m += 4
	}
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

	for _, device := range d.Devices {
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
	for _, device := range d.Devices {
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
	for _, device := range d.Devices {
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
		colorwarpGeneratedReverse := false
		d.activeRgb = rgb.Exit()

		// Generate random colors
		d.activeRgb.RGBStartColor = rgb.GenerateRandomColor(1)
		d.activeRgb.RGBEndColor = rgb.GenerateRandomColor(1)

		keys := make([]int, 0)

		for k := range d.Devices {
			keys = append(keys, k)
		}
		sort.Ints(keys)

		for {
			select {
			case <-d.activeRgb.Exit:
				return
			default:
				buff := make([]byte, 0)

				for _, k := range keys {
					if d.Devices[k].IsTemperatureProbe {
						continue
					}

					rgbCustomColor := true
					profile := rgb.GetRgbProfile(d.Devices[k].RGB)
					if profile == nil {
						for i := 0; i < int(d.Devices[k].LedChannels); i++ {
							buff = append(buff, []byte{0, 0, 0}...)
						}
						logger.Log(logger.Fields{"profile": d.Devices[k].RGB, "serial": d.Serial}).Warn("No such RGB profile found")
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

					switch d.Devices[k].RGB {
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
							if counterCircleshift[k] >= int(d.Devices[k].LedChannels) {
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
							if counterCircle[k] >= int(d.Devices[k].LedChannels) {
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
							if counterSpinner[k] >= int(d.Devices[k].LedChannels) {
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
			}
		}
	}(lightChannels)
}

// getDevices will fetch all devices connected to a hub
func (d *Device) getDevices() int {
	var devices = make(map[int]*Devices, 0)
	var m = 0

	// Fans
	response := d.read(modeGetFans, dataTypeGetFans)
	amount := d.getChannelAmount(response)
	for i := 0; i < amount; i++ {
		status := response[6:][i]
		if status == 0x07 {
			// Get a persistent speed profile. Fallback to Normal is anything fails
			speedProfile := "Normal"
			label := "Not Set"
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
					if rgb.GetRgbProfile(rp) != nil { // Speed profile exists in configuration
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

			// Get LED data
			var LedChannels uint8 = 0
			if internalLedDevice, ok := internalLedDevices[m]; ok {
				LedChannels = internalLedDevice.Total
			}

			// Build device object
			device := &Devices{
				ChannelId:   i,
				DeviceId:    fmt.Sprintf("%s-%v", "Fan", i),
				Name:        fmt.Sprintf("Fan %d", i),
				Rpm:         0,
				Temperature: 0,
				Description: "Fan",
				LedChannels: LedChannels,
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

			device.RGB = rgbProfile
			devices[m] = device
			m++
		}
	}

	// Temperature probe
	response = d.read(modeGetTemperatures, dataTypeGetTemperatures)
	sensorData := response[9:]
	for i, s := 0, 0; i < 1; i, s = i+1, s+3 {
		label := "Not Set"
		status := sensorData[s : s+3][0]
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
			m++
		}
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
		if aioType.RadiatorSize == radiatorSize && aioType.PumpVersion == pumpVersion {
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
	for i, s := 0, 0; i < amount; i, s = i+1, s+2 {
		currentSensor := sensorData[s : s+2]
		status := channels[6:][i]
		if status == 0x07 {
			if _, ok := d.Devices[m]; ok {
				rpm := int16(binary.LittleEndian.Uint16(currentSensor))
				if rpm > 1 {
					d.Devices[m].Rpm = rpm
				}
				m++
			}
		}
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
					}
				} else {
					if _, ok := d.Devices[m]; ok {
						d.Devices[m].Temperature = float32(int16(binary.LittleEndian.Uint16(currentSensor[1:3]))) / 10.0
					}
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

// setAutoRefresh will refresh device data
func (d *Device) setAutoRefresh() {
	timer = time.NewTicker(time.Duration(deviceRefreshInterval) * time.Millisecond)
	authRefreshChan = make(chan bool)
	go func() {
		for {
			select {
			case <-timer.C:
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

// UpdateRgbProfile will update device RGB profile
func (d *Device) UpdateRgbProfile(channelId int, profile string) uint8 {
	if rgb.GetRgbProfile(profile) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}

	if _, ok := d.Devices[channelId]; ok {
		// Update channel with new profile
		d.Devices[channelId].RGB = profile
	} else {
		return 0
	}

	d.DeviceProfile.RGBProfiles[channelId] = profile // Set profile
	d.saveDeviceProfile()                            // Save profile
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
	return 1
}

// GetAIOData will return AIO pump speed and liquid temperature
func (d *Device) GetAIOData() (int16, float32) {
	for _, device := range d.Devices {
		if device.ContainsPump {
			return device.Rpm, device.Temperature
		}
	}
	return 0, 0
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
	for i := 1; i <= 6; i++ {
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
	buf = append(buf, 0x01)
	buf = append(buf, 0x08)

	// Start at 1, since 0 is the pump, and iterate through all 6 physical connectors
	for i := 1; i <= 6; i++ {
		if z, ok := internalLedDevices[i]; ok {
			// Channel activation
			buf = append(buf, 0x01)

			// Fan LED command code, each LED device has different command code
			buf = append(buf, z.Command)
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

	for _, device := range d.Devices {
		if device.IsTemperatureProbe {
			continue
		}
		speedProfiles[device.ChannelId] = device.Profile
		rgbProfiles[device.ChannelId] = device.RGB
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
		Path:          profilePath,
	}

	// First save, assign saved profile to a device
	if d.DeviceProfile == nil {
		// RGB
		for _, device := range d.Devices {
			if device.IsTemperatureProbe {
				continue
			}
			rgbProfiles[device.ChannelId] = "static"
		}

		// Labels
		for _, device := range d.Devices {
			labels[device.ChannelId] = "Not Set"
		}

		// LCD
		if d.HasLCD {
			deviceProfile.LCDMode = 0
		}

		deviceProfile.Active = true
		d.DeviceProfile = deviceProfile
	} else {
		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.Brightness = d.DeviceProfile.Brightness
		if len(d.DeviceProfile.Path) < 1 {
			deviceProfile.Path = profilePath
			d.DeviceProfile.Path = profilePath
		} else {
			deviceProfile.Path = d.DeviceProfile.Path
		}
		deviceProfile.LCDMode = d.DeviceProfile.LCDMode
	}

	// Convert to JSON
	buffer, err := json.Marshal(deviceProfile)
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
						buffer := lcd.GenerateScreenImage(lcd.DisplayCPU, int(temperatures.GetCpuTemperature()), 0, 0)
						d.transferToLcd(buffer)
					}
				case lcd.DisplayGPU:
					{
						buffer := lcd.GenerateScreenImage(lcd.DisplayGPU, int(temperatures.GetGpuTemperature()), 0, 0)
						d.transferToLcd(buffer)
					}
				case lcd.DisplayLiquid:
					{
						for _, device := range d.Devices {
							if device.ContainsPump {
								buffer := lcd.GenerateScreenImage(lcd.DisplayLiquid, int(device.Temperature), 0, 0)
								d.transferToLcd(buffer)
							}
						}
					}
				case lcd.DisplayPump:
					{
						for _, device := range d.Devices {
							if device.ContainsPump {
								buffer := lcd.GenerateScreenImage(lcd.DisplayPump, int(device.Rpm), 0, 0)
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
						buffer := lcd.GenerateScreenImage(lcd.DisplayAllInOne, liquidTemp, cpuTemp, pumpSpeed)
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
