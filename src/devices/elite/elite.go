package elite

// Package: Elite
// This is the primary package for Corsair Elite AIOs.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/stats"
	"OpenLinkHub/src/temperatures"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Shutdown struct {
	command byte
	data    []byte
}

type SpeedMode struct {
	Value   byte
	ZeroRpm bool
	Pump    bool
}

type DeviceProfile struct {
	Active             bool
	Path               string
	Product            string
	Serial             string
	Brightness         uint8
	BrightnessSlider   *uint8
	OriginalBrightness uint8
	RGBProfiles        map[int]string
	SpeedProfiles      map[int]string
	Labels             map[int]string
}

type DeviceList struct {
	Name      string
	Channel   byte
	Index     int
	Type      byte
	Pump      bool
	Desc      string
	PumpModes map[byte]string
	HasSpeed  bool
	HasTemps  bool
}

type SupportedDevice struct {
	ProductId uint16 `json:"productId"`
	Product   string `json:"product"`
	Fans      uint8  `json:"fans"`
	FanLeds   uint8  `json:"fanLeds"`
	PumpLeds  uint8  `json:"pumpLeds"`
}

type Devices struct {
	ChannelId          int     `json:"channelId"`
	DeviceId           string  `json:"deviceId"`
	Type               byte    `json:"type"`
	Mode               byte    `json:"-"`
	Name               string  `json:"name"`
	Rpm                uint16  `json:"rpm"`
	Temperature        float64 `json:"temperature"`
	TemperatureString  string  `json:"temperatureString"`
	LedChannels        uint8   `json:"-"`
	ContainsPump       bool    `json:"-"`
	Description        string  `json:"description"`
	Profile            string  `json:"profile"`
	RGB                string  `json:"rgb"`
	Label              string  `json:"label"`
	PumpModes          map[byte]string
	HasSpeed           bool
	HasTemps           bool
	IsTemperatureProbe bool
}

type Device struct {
	dev               *hid.Device
	ProductId         uint16
	Manufacturer      string                    `json:"manufacturer"`
	Product           string                    `json:"product"`
	Serial            string                    `json:"serial"`
	Firmware          string                    `json:"firmware"`
	RGB               string                    `json:"rgb"`
	Fans              int                       `json:"fans"`
	RequireActivation bool                      `json:"requireActivation"`
	AIO               bool                      `json:"aio"`
	Devices           map[int]*Devices          `json:"devices"`
	UserProfiles      map[string]*DeviceProfile `json:"userProfiles"`
	ActiveDevice      SupportedDevice
	activeRgb         *rgb.ActiveRGB
	sequence          byte
	DeviceProfile     *DeviceProfile
	ExternalHub       bool
	RGBDeviceOnly     bool
	Template          string
	Brightness        map[int]string
	HasLCD            bool
	CpuTemp           float32
	GpuTemp           float32
	Rgb               *rgb.RGB
	rgbMutex          sync.RWMutex
	InvertRgb         bool
	mutex             sync.Mutex
	sequenceMutex     sync.Mutex
	autoRefreshChan   chan struct{}
	speedRefreshChan  chan struct{}
	timer             *time.Ticker
	timerSpeed        *time.Ticker
	Exit              bool
}

// https://www.3dbrew.org/wiki/CRC-8-CCITT
var crcTable = [256]uint8{
	0x00, 0x07, 0x0E, 0x09, 0x1C, 0x1B, 0x12, 0x15, 0x38, 0x3F, 0x36, 0x31, 0x24, 0x23, 0x2A, 0x2D,
	0x70, 0x77, 0x7E, 0x79, 0x6C, 0x6B, 0x62, 0x65, 0x48, 0x4F, 0x46, 0x41, 0x54, 0x53, 0x5A, 0x5D,
	0xE0, 0xE7, 0xEE, 0xE9, 0xFC, 0xFB, 0xF2, 0xF5, 0xD8, 0xDF, 0xD6, 0xD1, 0xC4, 0xC3, 0xCA, 0xCD,
	0x90, 0x97, 0x9E, 0x99, 0x8C, 0x8B, 0x82, 0x85, 0xA8, 0xAF, 0xA6, 0xA1, 0xB4, 0xB3, 0xBA, 0xBD,
	0xC7, 0xC0, 0xC9, 0xCE, 0xDB, 0xDC, 0xD5, 0xD2, 0xFF, 0xF8, 0xF1, 0xF6, 0xE3, 0xE4, 0xED, 0xEA,
	0xB7, 0xB0, 0xB9, 0xBE, 0xAB, 0xAC, 0xA5, 0xA2, 0x8F, 0x88, 0x81, 0x86, 0x93, 0x94, 0x9D, 0x9A,
	0x27, 0x20, 0x29, 0x2E, 0x3B, 0x3C, 0x35, 0x32, 0x1F, 0x18, 0x11, 0x16, 0x03, 0x04, 0x0D, 0x0A,
	0x57, 0x50, 0x59, 0x5E, 0x4B, 0x4C, 0x45, 0x42, 0x6F, 0x68, 0x61, 0x66, 0x73, 0x74, 0x7D, 0x7A,
	0x89, 0x8E, 0x87, 0x80, 0x95, 0x92, 0x9B, 0x9C, 0xB1, 0xB6, 0xBF, 0xB8, 0xAD, 0xAA, 0xA3, 0xA4,
	0xF9, 0xFE, 0xF7, 0xF0, 0xE5, 0xE2, 0xEB, 0xEC, 0xC1, 0xC6, 0xCF, 0xC8, 0xDD, 0xDA, 0xD3, 0xD4,
	0x69, 0x6E, 0x67, 0x60, 0x75, 0x72, 0x7B, 0x7C, 0x51, 0x56, 0x5F, 0x58, 0x4D, 0x4A, 0x43, 0x44,
	0x19, 0x1E, 0x17, 0x10, 0x05, 0x02, 0x0B, 0x0C, 0x21, 0x26, 0x2F, 0x28, 0x3D, 0x3A, 0x33, 0x34,
	0x4E, 0x49, 0x40, 0x47, 0x52, 0x55, 0x5C, 0x5B, 0x76, 0x71, 0x78, 0x7F, 0x6A, 0x6D, 0x64, 0x63,
	0x3E, 0x39, 0x30, 0x37, 0x22, 0x25, 0x2C, 0x2B, 0x06, 0x01, 0x08, 0x0F, 0x1A, 0x1D, 0x14, 0x13,
	0xAE, 0xA9, 0xA0, 0xA7, 0xB2, 0xB5, 0xBC, 0xBB, 0x96, 0x91, 0x98, 0x9F, 0x8A, 0x8D, 0x84, 0x83,
	0xDE, 0xD9, 0xD0, 0xD7, 0xC2, 0xC5, 0xCC, 0xCB, 0xE6, 0xE1, 0xE8, 0xEF, 0xFA, 0xFD, 0xF4, 0xF3,
}

var controlLighting = []byte{
	0x01, 0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f,
	0x7f, 0x7f, 0x7f, 0xff, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0xff,
	0xff, 0xff, 0xff, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
}

var shutdown = map[int]Shutdown{
	0: {
		command: cmdActivateChannels,
		data: []byte{
			0x0a, 0x01, 0x04, 0x07, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x00, 0x01, 0x02, 0x03, 0x04,
			0x01, 0x0a, 0x07, 0x04, 0x0b, 0x0a, 0x09, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x00,
			0x01, 0x0a, 0x07, 0x04, 0x01, 0x0a, 0x09, 0x08, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		},
	},
	1: {
		command: cmdActivateChannels + 1,
		data: []byte{
			0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x00, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c,
			0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c,
			0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		},
	},
	2: {
		command: cmdWriteColor,
		data: []byte{
			0x00, 0x00, 0xff, 0x00, 0x4a, 0xff, 0x00, 0x94, 0xff, 0x00, 0xdf, 0xff, 0x00, 0xff, 0xaa, 0x00,
			0xff, 0x15, 0x7f, 0x7f, 0x00, 0xfa, 0x00, 0x06, 0xdb, 0x00, 0x32, 0xd7, 0x00, 0x58, 0xf6, 0x00,
			0x76, 0x94, 0x00, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		},
	},
	3: {
		command: cmdControlLighting,
		data: []byte{
			0x00, 0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f,
			0x7f, 0x7f, 0x7f, 0x7f, 0x09, 0x20, 0x07, 0x00, 0x0b, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0xff,
			0xff, 0xff, 0xff, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		},
	},
}

var (
	pwd                        = ""
	cmdGetState                = []byte{0xff, 0x00}
	modeSetSpeed               = []byte{0x00, 0x03}
	cmdState                   = byte(0x00)
	cmdWriteColor              = byte(0x04)
	cmdControlLighting         = byte(0x01)
	cmdActivateChannels        = byte(0x02)
	BufferSize                 = 64
	HidBufferSize              = BufferSize + 1
	BufferLength               = BufferSize - 1
	deviceRefreshInterval      = 1000
	temperaturePullingInterval = 3000
	manualSpeedModes           = map[int]*SpeedMode{}
	rgbProfileUpgrade          = []string{"custom"}
	supportedDevices           = []SupportedDevice{
		{ProductId: 3095, Product: "H115i RGB PLATINUM", Fans: 2, FanLeds: 4, PumpLeds: 16},
		{ProductId: 3096, Product: "H100i RGB PLATINUM", Fans: 2, FanLeds: 4, PumpLeds: 16},
		{ProductId: 3097, Product: "H100i RGB PLATINUM SE", Fans: 2, FanLeds: 16, PumpLeds: 16},
		{ProductId: 3104, Product: "iCUE H100i RGB PRO XT", Fans: 2, FanLeds: 0, PumpLeds: 16},
		{ProductId: 3105, Product: "iCUE H115i RGB PRO XT", Fans: 2, FanLeds: 0, PumpLeds: 16},
		{ProductId: 3106, Product: "iCUE H150i RGB PRO XT", Fans: 3, FanLeds: 0, PumpLeds: 16},
		{ProductId: 3125, Product: "iCUE H100i RGB ELITE", Fans: 2, FanLeds: 0, PumpLeds: 16}, // Black
		{ProductId: 3126, Product: "iCUE H115i RGB ELITE", Fans: 2, FanLeds: 0, PumpLeds: 16}, // Black
		{ProductId: 3127, Product: "iCUE H150i RGB ELITE", Fans: 3, FanLeds: 0, PumpLeds: 16}, // Black
		{ProductId: 3136, Product: "iCUE H100i RGB ELITE", Fans: 2, FanLeds: 0, PumpLeds: 16}, // White
		{ProductId: 3137, Product: "iCUE H150i RGB ELITE", Fans: 3, FanLeds: 0, PumpLeds: 16}, // White
	}
	deviceList = []DeviceList{
		{
			Name:    "Pump",
			Channel: 28,
			Index:   0,
			Type:    0,
			Pump:    true,
			Desc:    "Pump",
			PumpModes: map[byte]string{
				0: "Quiet",
				1: "Normal",
				2: "Performance",
			},
			HasSpeed: true,
			HasTemps: true,
		},
		{
			Name:      "Fan 1",
			Channel:   14,
			Index:     1,
			Type:      1,
			Pump:      false,
			Desc:      "Fan",
			PumpModes: map[byte]string{},
			HasSpeed:  true,
		},
		{
			Name:      "Fan 2",
			Channel:   21,
			Index:     2,
			Type:      1,
			Pump:      false,
			Desc:      "Fan",
			PumpModes: map[byte]string{},
			HasSpeed:  true,
		},
		{
			Name:      "Fan 3",
			Channel:   42,
			Index:     3,
			Type:      1,
			Pump:      false,
			Desc:      "Fan",
			PumpModes: map[byte]string{},
			HasSpeed:  true,
		},
	}
)

func Init(vendorId, productId uint16) *Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	// Open device, return if failure
	dev, err := hid.OpenFirst(vendorId, productId)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId}).Error("Unable to open HID device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:      dev,
		AIO:      true,
		Template: "elite.html",
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
		InvertRgb:        true,
		autoRefreshChan:  make(chan struct{}),
		speedRefreshChan: make(chan struct{}),
		timer:            &time.Ticker{},
		timerSpeed:       &time.Ticker{},
	}

	d.ProductId = productId

	// Bootstrap
	d.getManufacturer()    // Manufacturer
	d.getProduct()         // Product
	d.getSerial()          // Serial
	d.loadRgb()            // Load RGB
	d.setFans()            // Number of fans
	d.loadDeviceProfiles() // Load all device profiles
	d.getDeviceFirmware()  // Firmware
	d.getDevices()         // Get devices
	d.setAutoRefresh()     // Set auto device refresh
	d.saveDeviceProfile()  // Save profile
	d.initLeds()           // Device lighting mode
	d.setDeviceColor()     // Device color
	if config.GetConfig().Manual {
		fmt.Println(
			fmt.Sprintf("[%s [%s]] Manual flag enabled. Process will not monitor temperature or adjust fan speed.", d.Serial, d.Product),
		)
	} else {
		d.updateDeviceSpeed() // Update device speed
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")
	return d
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
	var once sync.Once
	go func() {
		once.Do(func() {
			if !config.GetConfig().Manual {
				d.timerSpeed.Stop()
				if d.speedRefreshChan != nil {
					close(d.speedRefreshChan)
				}
			}

			if d.autoRefreshChan != nil {
				close(d.autoRefreshChan)
			}
		})
	}()

	d.setHardwareMode() // Hardware mode
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
	var once sync.Once
	go func() {
		once.Do(func() {
			if !config.GetConfig().Manual {
				d.timerSpeed.Stop()
				if d.speedRefreshChan != nil {
					close(d.speedRefreshChan)
				}
			}

			if d.autoRefreshChan != nil {
				close(d.autoRefreshChan)
			}
		})
	}()
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 1
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

// lightingControl will create an empty byte slice with length of 80.
// After that, we fill an array with byte value from 0 to 79
func (d *Device) lightingControl() []byte {
	buf := make([]byte, 80)
	for i := 0; i < len(buf); i++ {
		buf[i] = byte(i)
	}
	return buf
}

// setHardwareMode will put a device back to hardware mode
func (d *Device) setHardwareMode() {
	indexes := d.lightingControl()
	chunks := common.ProcessMultiChunkPacket(indexes, 40)
	for i, chunk := range chunks {
		buf := make([]byte, 61)
		copy(buf[0:], chunk)
		for m := len(chunk); m < 61; m++ {
			buf[m] = byte(0xff)
		}
		command := cmdActivateChannels + byte(i)
		d.transfer(command, buf)
		time.Sleep(100 * time.Millisecond)
	}

	for i := 0; i < len(shutdown); i++ {
		value := shutdown[i]
		d.transfer(value.command, value.data)
	}
}

// initLeds will initialize LED channels
func (d *Device) initLeds() {
	indexes := d.lightingControl()
	chunks := common.ProcessMultiChunkPacket(indexes, 40)
	for i, chunk := range chunks {
		buf := make([]byte, 61)
		copy(buf[0:], chunk)
		for m := len(chunk); m < 61; m++ {
			buf[m] = byte(0xff)
		}
		command := cmdActivateChannels + byte(i)
		d.transfer(command, buf)
		time.Sleep(100 * time.Millisecond)
	}
	d.transfer(cmdControlLighting, controlLighting)
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

	// Get the number of LED channels we have
	lightChannels := 0
	m := 0
	for _, device := range d.Devices {
		lightChannels += int(device.LedChannels)
	}

	// Do we have any RGB component in the system?
	if lightChannels == 0 {
		logger.Log(logger.Fields{}).Info("No RGB compatible devices found")
		return
	}

	if lightChannels > 0 {
		for i := 0; i < lightChannels; i++ {
			reset[i] = []byte{
				byte(color.Red),
				byte(color.Green),
				byte(color.Blue),
			}
		}
		m++
	}

	buffer = rgb.SetColorInverted(reset)
	d.writeColor(buffer)

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
			profile := d.GetRgbProfile("static")
			profile.StartColor.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
			profileColor := rgb.ModifyBrightness(profile.StartColor)
			for i := 0; i < lightChannels; i++ {
				reset[i] = []byte{
					byte(profileColor.Blue),
					byte(profileColor.Green),
					byte(profileColor.Red),
				}
			}
			buffer = rgb.SetColor(reset)
			d.writeColor(buffer)
			return
		}
	}

	go func(lightChannels int) {
		startTime := time.Now()
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

					r.Inverted = d.InvertRgb
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
					case "liquid-temperature":
						{
							r.MinTemp = profile.MinTemp
							r.MaxTemp = profile.MaxTemp
							r.Temperature(float64(d.getLiquidTemperature()))
							buff = append(buff, r.Output...)
						}
					case "cpu-temperature":
						{
							r.MinTemp = profile.MinTemp
							r.MaxTemp = profile.MaxTemp
							r.Temperature(float64(d.CpuTemp))
							buff = append(buff, r.Output...)
						}
					case "gpu-temperature":
						{
							r.MinTemp = profile.MinTemp
							r.MaxTemp = profile.MaxTemp
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
				time.Sleep(20 * time.Millisecond)
			}
		}
	}(lightChannels)
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

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	var defaultBrightness = uint8(100)
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"

	speedProfiles := make(map[int]string, len(d.Devices))
	rgbProfiles := make(map[int]string, len(d.Devices))
	labels := make(map[int]string, len(d.Devices))

	for _, device := range d.Devices {
		speedProfiles[device.ChannelId] = device.Profile
		labels[device.ChannelId] = device.Label
		if device.LedChannels > 0 {
			rgbProfiles[device.ChannelId] = device.RGB
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
		for _, device := range d.Devices {
			if device.LedChannels > 0 {
				rgbProfiles[device.ChannelId] = "static"
			}
			labels[device.ChannelId] = "Set Label"
		}
		deviceProfile.Active = true
		d.DeviceProfile = deviceProfile
	} else {
		if d.DeviceProfile.BrightnessSlider == nil {
			deviceProfile.BrightnessSlider = &defaultBrightness
			d.DeviceProfile.BrightnessSlider = &defaultBrightness
		} else {
			deviceProfile.BrightnessSlider = d.DeviceProfile.BrightnessSlider
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

// getPumpMode will return byte pump mode based on a profile name
func (d *Device) getPumpMode(index int, profile string) byte {
	for device := range deviceList {
		if deviceList[device].Index == index {
			for pumpMode, modeName := range deviceList[device].PumpModes {
				if modeName == profile {
					return pumpMode
				}
			}
		}
	}
	return 0
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

// ChangeDeviceBrightnessValue will change device brightness via slider
func (d *Device) ChangeDeviceBrightnessValue(value uint8) uint8 {
	if value < 0 || value > 100 {
		return 0
	}

	d.DeviceProfile.BrightnessSlider = &value
	d.saveDeviceProfile()

	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB

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
func (d *Device) UpdateSpeedProfile(channelId int, profile string) uint8 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Check if the profile exists
	profiles := temperatures.GetTemperatureProfile(profile)
	if profiles == nil {
		return 0
	}

	// If the profile is liquid temperature, check for the presence of AIOs
	if profiles.Sensor == temperatures.SensorTypeLiquidTemperature {
		valid := false
		for _, device := range d.Devices {
			if device.ChannelId == 0 { // Pump
				valid = true
				break
			}
		}

		if !valid {
			return 2
		}
	}

	// Check if actual channelId exists in the device list
	if _, ok := d.Devices[channelId]; ok {
		d.Devices[channelId].Profile = profile
	}

	d.saveDeviceProfile()
	return 1
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
	d.rgbMutex.Lock()
	defer d.rgbMutex.Unlock()

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

	d.DeviceProfile.RGBProfiles[channelId] = profile // Set profile
	d.saveDeviceProfile()                            // Save profile
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
	return 1
}

// UpdateDeviceMetrics will update device metrics
func (d *Device) UpdateDeviceMetrics() {
	if d.Exit {
		return
	}

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
			Temperature:      device.Temperature,
			LedChannels:      strconv.Itoa(int(device.LedChannels)),
			Rpm:              int16(device.Rpm),
			TemperatureProbe: strconv.FormatBool(device.IsTemperatureProbe),
		}
		metrics.Populate(header)
	}
}

// getDevices will fetch all devices connected to a hub
func (d *Device) getDevices() int {
	var devices = make(map[int]*Devices)
	response := d.read(cmdState, cmdGetState)

	for device := range deviceList {
		if deviceList[device].Index > d.Fans {
			// Depending on AIO type, skip last fan in an array
			continue
		}

		// Get a persistent speed profile. Fallback to Normal is anything fails
		speedProfile := "Normal"
		label := "Set Label"
		speedMode := &SpeedMode{
			ZeroRpm: false,
			Pump:    deviceList[device].Pump,
			Value:   70,
		}

		if d.DeviceProfile != nil {
			// Profile is set
			if sp, ok := d.DeviceProfile.SpeedProfiles[deviceList[device].Index]; ok {
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
			if lb, ok := d.DeviceProfile.Labels[deviceList[device].Index]; ok {
				if len(lb) > 0 {
					label = lb
				}
			}
		} else {
			logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
		}

		rgbProfile := "static"
		var ledChannels uint8 = 0
		// LED channels
		supportedDevice := d.getSupportedDevice(d.ProductId)
		if deviceList[device].Pump {
			speedMode.Value = 1
			ledChannels = supportedDevice.PumpLeds
		} else {
			ledChannels = supportedDevice.FanLeds
		}

		if ledChannels > 0 {
			// Get a persistent speed profile. Fallback to Normal is anything fails
			if d.DeviceProfile != nil {
				// Profile is set
				if rp, ok := d.DeviceProfile.RGBProfiles[deviceList[device].Index]; ok {
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
		}

		// Temps
		var temperature float64 = 0
		if deviceList[device].Type == 0 {
			temperature = float64(response[8]) + float64(response[7])/255
		}

		// RPM
		rpm := binary.LittleEndian.Uint16(response[deviceList[device].Channel+1:])

		// Device object
		dev := &Devices{
			ChannelId:    deviceList[device].Index,
			Type:         deviceList[device].Type,
			DeviceId:     fmt.Sprintf("%s-%v", deviceList[device].Desc, deviceList[device].Index),
			Mode:         response[24],
			Name:         deviceList[device].Name,
			Rpm:          rpm,
			Temperature:  math.Floor(temperature*100) / 100,
			LedChannels:  ledChannels,
			ContainsPump: deviceList[device].Pump,
			Description:  deviceList[device].Desc,
			Profile:      speedProfile,
			PumpModes:    deviceList[device].PumpModes,
			HasSpeed:     deviceList[device].HasSpeed,
			HasTemps:     deviceList[device].HasTemps,
			RGB:          rgbProfile,
			Label:        label,
		}

		// Default speed modes
		manualSpeedModes[deviceList[device].Index] = speedMode

		// Add to array
		devices[deviceList[device].Index] = dev
	}
	d.Devices = devices
	return len(devices)
}

// setDefaults will set default mode for all devices
func (d *Device) getDeviceData() {
	if d.Exit {
		return
	}

	response := d.read(cmdState, cmdGetState)
	if response == nil {
		return
	}

	for device := range deviceList {
		// Temp
		var temperature float64 = 0
		if deviceList[device].Type == 0 {
			temperature = float64(response[8]) + float64(response[7])/255
		}

		// RPM
		rpm := binary.LittleEndian.Uint16(response[deviceList[device].Channel+1:])

		// Update
		if _, ok := d.Devices[deviceList[device].Index]; ok {
			if rpm > 0 {
				d.Devices[deviceList[device].Index].Rpm = rpm
			}

			if temperature > 0 {
				temp := math.Floor(temperature*100) / 100
				d.Devices[deviceList[device].Index].Temperature = temp
				d.Devices[deviceList[device].Index].TemperatureString = dashboard.GetDashboard().TemperatureToString(float32(temp))
			}

			rpmString := fmt.Sprintf("%v RPM", d.Devices[deviceList[device].Index].Rpm)

			stats.UpdateAIOStats(
				d.Serial,
				d.Devices[deviceList[device].Index].Name,
				d.Devices[deviceList[device].Index].TemperatureString,
				rpmString,
				d.Devices[deviceList[device].Index].Label,
				deviceList[device].Index,
			)
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

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	response := d.read(cmdState, cmdGetState)
	if response == nil {
		logger.Log(logger.Fields{}).Error("Unable to get device firmware")
	}

	v1, v2, v3 := int(response[2]>>4), int(response[2]>>4), int(response[3])
	d.Firmware = fmt.Sprintf("%d.%.2d.%d", v1, v2, v3)
}

// getManufacturer will return device manufacturer
func (d *Device) getManufacturer() {
	manufacturer, err := d.dev.GetMfrStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to get manufacturer")
	}
	d.Manufacturer = manufacturer
}

// setFans will set number of fans
func (d *Device) setFans() {
	product := d.getSupportedDevice(d.ProductId)
	if product == nil {
		d.Fans = 0
	} else {
		d.Fans = int(product.Fans)
	}
}

// getProduct will set product name
func (d *Device) getProduct() {
	product := d.getSupportedDevice(d.ProductId)
	if product == nil {
		pd, err := d.dev.GetProductStr()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Fatal("Unable to get product")
		}
		d.Product = pd
	} else {
		d.Product = product.Product
	}
}

// getSerial will set the device serial number.
// In case of no serial, productId will be placed as serial number
func (d *Device) getSerial() {
	serial, err := d.dev.GetSerialNbr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to get device serial number")
	}
	d.Serial = serial

	if len(serial) == 0 {
		d.Serial = strconv.Itoa(int(d.ProductId))
	}
}

// getSupportedDevice will return supported device or nil pointer
func (d *Device) getSupportedDevice(productId uint16) *SupportedDevice {
	for _, device := range supportedDevices {
		if device.ProductId == productId {
			return &device
		}
	}
	return nil
}

// nextSequence will increment next sequence for packet
func (d *Device) nextSequence() byte {
	d.sequenceMutex.Lock()
	defer d.sequenceMutex.Unlock()
	for {
		d.sequence += 0x08
		if d.sequence != 0x00 {
			return d.sequence
		}
	}
}

// setSequence will set sequence with given value from packet response
func (d *Device) setSequence(value byte) {
	d.sequenceMutex.Lock()
	defer d.sequenceMutex.Unlock()
	d.sequence = value
}

// newHidPacket will create a new HID packet and append data to it
func (d *Device) newHidPacket(data []byte) []byte {
	buf := make([]byte, HidBufferSize)
	copy(buf[1:], data)
	return buf
}

// calculateChecksum will calculate CRC checksum
func (d *Device) calculateChecksum(data []byte) byte {
	var val uint8 = 0
	for _, b := range data {
		val = crcTable[val^b]
	}
	return val
}

// deviceSpeedPacket will create a new byte packet for device speed modification
func (d *Device) deviceSpeedPacket(values map[int]*SpeedMode) []byte {
	// Main packet
	buffer := make([]byte, BufferLength-4)

	// Speed data
	speedBuffer := make([]byte, BufferLength-13)
	for i := range speedBuffer {
		speedBuffer[i] = 0x00
	}

	// Sort device list
	keys := make([]int, 0)
	for k := range values {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	index := 0
	for _, k := range keys {
		if values[k].Pump {
			speedBuffer[12] = values[k].Value
			speedBuffer[13] = 0xff
			speedBuffer[14] = 0xff
			speedBuffer[17] = 0xff
			speedBuffer[18] = 0x07
		} else {
			speedBuffer[index] = 0x02
			if values[k].ZeroRpm {
				// More testing is needed on what temp are fans triggered
				// For now, this is disabled
				//buf[index] = 0x03
			}
			index += 5
			speedBuffer[index] = byte(common.FractionOfByte(float64(values[k].Value)/100, nil))
			index += 1
		}
	}

	// Fill the main packet with 0xff
	for i := range buffer {
		buffer[i] = 0xff
	}

	// Static commands
	buffer[0] = 0x14
	buffer[1] = 0x00
	buffer[2] = 0xff
	buffer[3] = 0x05

	// Append speed data
	if len(speedBuffer) > 0 {
		copy(buffer[9:], speedBuffer)
	}
	return buffer
}

// setSpeed will modify device speed
func (d *Device) setSpeed(data map[int]*SpeedMode) {
	if d.Exit {
		return
	}

	buffer := make(map[int]*SpeedMode, 1)

	if len(d.Devices) == 4 {
		// Pump + 3 fans
		buffer[0] = data[3]
		buf := d.deviceSpeedPacket(buffer)
		d.transfer(modeSetSpeed[1], buf)
	}

	for i := 0; i < 3; i++ {
		buffer[i] = data[i]
	}

	buf := d.deviceSpeedPacket(buffer)
	d.transfer(modeSetSpeed[0], buf)
}

// getLiquidTemperature will fetch temperature from AIO device
func (d *Device) getLiquidTemperature() float32 {
	for _, device := range d.Devices {
		if device.ChannelId == 0 {
			return float32(device.Temperature)
		}
	}
	return 0
}

// updateDeviceSpeed will update device speed based on a temperature reading
func (d *Device) updateDeviceSpeed() {
	d.timerSpeed = time.NewTicker(time.Duration(temperaturePullingInterval) * time.Millisecond)
	tmp := make(map[int]string)
	channelSpeeds := make(map[int]*SpeedMode, len(d.Devices))
	var change = false

	go func() {
		for {
			select {
			case <-d.timerSpeed.C:
				var temp float32 = 0
				for _, device := range d.Devices {
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
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial}).Warn("Unable to get CPU temperature.")
							}
						}
					case temperatures.SensorTypeLiquidTemperature:
						{
							temp = d.getLiquidTemperature()
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial}).Warn("Unable to get liquid temperature.")
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
					case temperatures.SensorTypeExternalHwMon:
						{
							temp = temperatures.GetHwMonTemperature(profiles.Device)
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "hwmonDeviceId": profiles.Device}).Warn("Unable to get hwmon temperature.")
							}
						}
					case temperatures.SensorTypeExternalExecutable:
						{
							temp = temperatures.GetExternalBinaryTemperature(profiles.Device)
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "binary": profiles.Device}).Warn("Unable to get temperature from binary.")
							}
						}
					}

					// All temps failed, default to 50
					if temp == 0 {
						temp = 50
					}

					if device.ChannelId == 0 {
						cp := fmt.Sprintf("%s-%d", device.Profile, device.ChannelId)
						if ok := tmp[device.ChannelId]; ok != cp {
							tmp[device.ChannelId] = cp
							speedMode := &SpeedMode{}
							speedMode.Value = d.getPumpMode(device.ChannelId, device.Profile)
							speedMode.ZeroRpm = false
							speedMode.Pump = true
							channelSpeeds[device.ChannelId] = speedMode
							change = true
						}
					} else {
						if config.GetConfig().GraphProfiles {
							fansValue := temperatures.Interpolate(profiles.Points[1], temp)
							fans := int(math.Round(float64(fansValue)))

							// Failsafe
							if fans < 20 {
								fans = 20
							}
							if fans > 100 {
								fans = 100
							}
							cp := fmt.Sprintf("%s-%d-%f", device.Profile, device.ChannelId, temp)
							if ok := tmp[device.ChannelId]; ok != cp {
								speedMode := &SpeedMode{}
								tmp[device.ChannelId] = cp
								speedMode.ZeroRpm = profiles.ZeroRpm
								speedMode.Value = byte(fans)
								speedMode.Pump = false
								channelSpeeds[device.ChannelId] = speedMode
								change = true
							}

						} else {
							for i := 0; i < len(profiles.Profiles); i++ {
								profile := profiles.Profiles[i]
								minimum := profile.Min + 0.1
								if common.InBetween(temp, minimum, profile.Max) {
									cp := fmt.Sprintf("%s-%d-%d", device.Profile, device.ChannelId, profile.Fans)
									if ok := tmp[device.ChannelId]; ok != cp {
										speedMode := &SpeedMode{}
										tmp[device.ChannelId] = cp
										speedMode.ZeroRpm = profiles.ZeroRpm
										speedMode.Value = byte(profile.Fans)
										speedMode.Pump = false
										channelSpeeds[device.ChannelId] = speedMode
										change = true
									}
								}
							}
						}
					}

				}
				if change {
					change = false
					d.setSpeed(channelSpeeds)
				}
			case <-d.speedRefreshChan:
				d.timerSpeed.Stop()
				return
			}
		}
	}()
}

// UpdateDeviceSpeed will update device channel speed.
func (d *Device) UpdateDeviceSpeed(channelId int, value uint16) uint8 {
	if device, ok := manualSpeedModes[channelId]; ok {
		if device.Pump {
			if value < 0 || value > 2 {
				value = 1
			}
			manualSpeedModes[channelId].Value = byte(value)
		} else {
			manualSpeedModes[channelId].Value = byte(value)
		}
		d.setSpeed(manualSpeedModes)
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

// read will read data from a device and return data as a byte array
func (d *Device) read(command byte, data []byte) []byte {
	bufferR := d.transfer(command, data)
	crc := bufferR[len(bufferR)-1]
	crcForCalc := d.calculateChecksum(bufferR[1 : len(bufferR)-1])

	if crc != crcForCalc {
		logger.Log(logger.Fields{"crc": crc, "calc": crcForCalc}).Error("Invalid CRC checksum")
	}

	d.setSequence(bufferR[1])
	return bufferR
}

func (d *Device) writeColor(data []byte) {
	if d.Exit {
		return
	}

	chunks := common.ProcessMultiChunkPacket(data, 60)
	for i, chunk := range chunks {
		command := cmdWriteColor + byte(i)
		d.transfer(command, chunk)
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(command byte, data []byte) []byte {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	buffer := make([]byte, BufferSize)

	// Make everything 0xff on init
	for i := 0; i < len(buffer); i++ {
		if buffer[i] == 0 {
			buffer[i] = 0x00
		}
	}
	buffer[0] = byte(BufferLength)
	buffer[1] = d.nextSequence() | command

	copy(buffer[2:], data)
	buffer[len(buffer)-1] = d.calculateChecksum(buffer[1 : len(buffer)-1])

	bufferR := make([]byte, BufferSize)
	reports := make([]byte, 1)
	bufferW := d.newHidPacket(buffer)

	// Every now and often, we get a HUD_REPORT that break sequence chain.
	err := d.dev.SetNonblock(true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to SetNonblock")
	}

	for {
		n, err := d.dev.Read(reports)
		if err != nil {
			if n < 0 {
				// discarding packet
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

	if _, err = d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
	}

	// Get data from a device
	if _, err = d.dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read data from device")
	}

	return bufferR
}
