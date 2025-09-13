package cc

// Package: CORSAIR iCUE COMMANDER CORE
// This is the primary package for CORSAIR iCUE COMMANDER CORE.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices/lcd"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/stats"
	"OpenLinkHub/src/systeminfo"
	"OpenLinkHub/src/temperatures"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"OpenLinkHub/src/cluster"
	"OpenLinkHub/src/openrgb"
	"github.com/sstallion/go-hid"
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
	cmdSetLedPorts             = []byte{0x1e}
	modeGetLeds                = []byte{0x20}
	modeGetSpeeds              = []byte{0x17}
	modeSetSpeed               = []byte{0x18}
	modeGetTemperatures        = []byte{0x21}
	modeGetFans                = []byte{0x1a}
	modeSetColor               = []byte{0x22}
	dataTypeSetSpeed           = []byte{0x07, 0x00}
	dataTypeSetColor           = []byte{0x12, 0x00}
	dataTypeSubColor           = []byte{0x07, 0x00}
	bufferSize                 = 64
	bufferSizeWrite            = bufferSize + 1
	ledInit                    = 500
	headerSize                 = 2
	headerWriteSize            = 4
	deviceRefreshInterval      = 1000
	lcdRefreshInterval         = 1000
	defaultSpeedValue          = 50
	temperaturePullingInterval = 3000
	ledStartIndex              = 6
	maxBufferSizePerRequest    = 61
	lcdHeaderSize              = 8
	lcdBufferSize              = 1024
	maxLCDBufferSizePerRequest = lcdBufferSize - lcdHeaderSize
	i2cPrefix                  = "i2c"
	rgbProfileUpgrade          = []string{"nebula", "marquee", "rotarystack", "sequential", "spiralrainbow"}
	rgbModes                   = []string{
		"circle",
		"circleshift",
		"colorpulse",
		"colorshift",
		"colorwarp",
		"cpu-temperature",
		"flickering",
		"gpu-temperature",
		"liquid-temperature",
		"marquee",
		"nebula",
		"led",
		"off",
		"rainbow",
		"rotarystack",
		"rotator",
		"sequential",
		"spinner",
		"spiralrainbow",
		"static",
		"storm",
		"watercolor",
		"wave",
	}
	aioList = []AIOList{
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

type RGBOverride struct {
	Enabled       bool
	RGBStartColor rgb.Color
	RGBEndColor   rgb.Color
	RgbModeSpeed  float64
}

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

// LedChannel struct for LED pump and fan data
type LedChannel struct {
	Total   uint8
	Command byte
	Name    string
}

// DeviceProfile struct contains all device profile
type DeviceProfile struct {
	Active             bool
	Path               string
	Product            string
	Serial             string
	LCDMode            uint8
	LCDImage           string
	LCDRotation        uint8
	Brightness         uint8
	BrightnessSlider   *uint8
	OriginalBrightness uint8
	RGBProfiles        map[int]string
	SpeedProfiles      map[int]string
	Labels             map[int]string
	RGBLabels          map[int]string
	CustomLEDs         map[int]int
	MultiRGB           string
	MultiProfile       string
	RGBOverride        map[int]map[int]RGBOverride
	OpenRGBIntegration bool
	RGBCluster         bool
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
	Debug              bool
	dev                *hid.Device
	lcd                *hid.Device
	Manufacturer       string                    `json:"manufacturer"`
	Product            string                    `json:"product"`
	Serial             string                    `json:"serial"`
	Path               string                    `json:"path"`
	Firmware           string                    `json:"firmware"`
	AIOType            string                    `json:"-"`
	Devices            map[int]*Devices          `json:"devices"`
	RgbDevices         map[int]*Devices          `json:"rgbDevices"`
	UserProfiles       map[string]*DeviceProfile `json:"userProfiles"`
	ExternalLedDevice  []ExternalLedDevice
	DeviceProfile      *DeviceProfile
	TemperatureProbes  *[]TemperatureProbe
	activeRgb          *rgb.ActiveRGB
	Template           string
	HasLCD             bool
	VendorId           uint16
	LCDModes           map[int]string
	LCDRotations       map[int]string
	Brightness         map[int]string
	CpuTemp            float32
	GpuTemp            float32
	FreeLedPorts       map[int]string
	FreeLedPortLEDs    map[int]string
	Rgb                *rgb.RGB
	rgbMutex           sync.RWMutex
	LCDImage           *lcd.ImageData
	Exit               bool
	mutex              sync.Mutex
	mutexLcd           sync.Mutex
	deviceLock         sync.Mutex
	autoRefreshChan    chan struct{}
	speedRefreshChan   chan struct{}
	lcdRefreshChan     chan struct{}
	lcdImageChan       chan struct{}
	timer              *time.Ticker
	timerSpeed         *time.Ticker
	lcdTimer           *time.Ticker
	internalLedDevices map[int]*LedChannel
	RGBModes           []string
	queue              chan []byte
	instance           *common.Device
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
func Init(vendorId, productId uint16, serial, path string) *common.Device {
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
		Path:              path,
		ExternalLedDevice: externalLedDevices,
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
		RGBModes:           rgbModes,
		FreeLedPorts:       make(map[int]string, 6),
		FreeLedPortLEDs:    make(map[int]string, 34),
		internalLedDevices: make(map[int]*LedChannel, 7),
		autoRefreshChan:    make(chan struct{}),
		speedRefreshChan:   make(chan struct{}),
		timer:              &time.Ticker{},
		lcdTimer:           &time.Ticker{},
		timerSpeed:         &time.Ticker{},
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
	d.loadRgb()             // Load RGB
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
	d.saveDeviceProfile()   // Save
	d.setColorEndpoint()    // Set device color endpoint
	d.setDefaults()         // Set default speed and color values for fans and pumps
	d.setAutoRefresh()      // Set auto device refresh
	d.getTemperatureProbe() // Devices with temperature probes
	d.resetLEDPorts()       // Reset device LED
	if config.GetConfig().Manual {
		fmt.Println(
			fmt.Sprintf("[%s [%s]] Manual flag enabled. Process will not monitor temperature or adjust fan speed.", d.Serial, d.Product),
		)
	} else {
		d.updateDeviceSpeed() // Update device speed
	}
	d.setDeviceColor() // Device color
	if d.HasLCD {
		if d.DeviceProfile.LCDMode == lcd.DisplayImage {
			if d.loadLcdImage() != 1 {
				logger.Log(logger.Fields{"serial": d.Serial}).Warn("Unable to load LCD image from profile")
			} else {
				d.setupLCDImage()
			}
		} else {
			d.setupLCD(false)
		}
	}
	d.setupOpenRGBController() // OpenRGB Controller
	d.setupClusterController() // RGB Cluster
	d.createDevice()           // Device register
	d.startQueueWorker()       // Queue
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeCC,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-device.svg",
		Instance:    d,
		GetDevice:   d,
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
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.autoRefreshChan != nil {
				close(d.autoRefreshChan)
			}
			if !config.GetConfig().Manual {
				d.timerSpeed.Stop()
				if d.speedRefreshChan != nil {
					close(d.speedRefreshChan)
				}
			}
			if d.queue != nil {
				close(d.queue)
			}
		})
	}()

	if d.HasLCD {
		if d.DeviceProfile.LCDMode == lcd.DisplayImage {
			if d.lcdImageChan != nil {
				close(d.lcdImageChan)
			}
		} else {
			if d.lcdRefreshChan != nil {
				close(d.lcdRefreshChan)
			}
		}
		d.lcdTimer.Stop()

		lcdReports := map[int][]byte{0: {0x03, 0x1e, 0x01, 0x01}, 1: {0x03, 0x1d, 0x00, 0x01}}
		for i := 0; i <= 1; i++ {
			_, e := d.lcd.SendFeatureReport(lcdReports[i])
			if e != nil {
				logger.Log(logger.Fields{"error": e}).Error("Unable to send report to LCD HID device")
			}
		}

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
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
}

// StopDirty will stop device in a dirty way
func (d *Device) StopDirty() uint8 {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}

	d.timer.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.autoRefreshChan != nil {
				close(d.autoRefreshChan)
			}
			if !config.GetConfig().Manual {
				d.timerSpeed.Stop()
				if d.speedRefreshChan != nil {
					close(d.speedRefreshChan)
				}
			}
			if d.queue != nil {
				close(d.queue)
			}
		})
	}()

	if d.HasLCD {
		if d.DeviceProfile.LCDMode == lcd.DisplayImage {
			if d.lcdImageChan != nil {
				close(d.lcdImageChan)
			}
		} else {
			if d.lcdRefreshChan != nil {
				close(d.lcdRefreshChan)
			}
		}
		d.lcdTimer.Stop()

		lcdReports := map[int][]byte{0: {0x03, 0x1e, 0x01, 0x01}, 1: {0x03, 0x1d, 0x00, 0x01}}
		for i := 0; i <= 1; i++ {
			_, e := d.lcd.SendFeatureReport(lcdReports[i])
			if e != nil {
				logger.Log(logger.Fields{"error": e}).Error("Unable to send report to LCD HID device")
			}
		}

		err := d.lcd.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close LCD HID device")
		}
	}
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
			template := rgb.GetRgbProfile(profile)
			if template == nil {
				d.Rgb.Profiles[profile] = rgb.Profile{}
			} else {
				d.Rgb.Profiles[profile] = *template
			}
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
	var serial = ""
	var productIds = []uint16{3129, 3123}

	enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
		if info.InterfaceNbr == 0 {
			serial = info.SerialNbr
		}
		return nil
	})

	for _, productId := range productIds {
		err := hid.Enumerate(d.VendorId, productId, enum)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Fatal("Unable to enumerate LCD devices")
			continue
		}

		if len(serial) > 0 {
			lcdPanel, e := hid.Open(d.VendorId, productId, serial)
			if e != nil {
				logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId, "productId": productId}).Error("Unable to open LCD HID device")
				d.HasLCD = false
				continue
			}
			d.lcd = lcdPanel
			d.HasLCD = true
			break
		}
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
		"getDeviceFirmware",
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
	lc := d.read(modeGetLeds, "getLedDevices")
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
			d.internalLedDevices[i] = leds
		} else {
			// Add to a device map
			d.internalLedDevices[i] = leds
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

// isRgbStatic will return true or false if all devices are set to static RGB mode
func (d *Device) isRgbStatic() bool {
	s, l := 0, 0

	keys := make([]int, 0)
	for k := range d.RgbDevices {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		if d.RgbDevices[k].LedChannels > 0 {
			l++ // device has LED
			if d.RgbDevices[k].RGB == "static" {
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

// setupOpenRGBController will create Cluster Controller for RGB Cluster
func (d *Device) setupClusterController() {
	if d.DeviceProfile == nil {
		return
	}

	if !d.DeviceProfile.RGBCluster {
		return
	}

	lightChannels := 0
	for k := range d.RgbDevices {
		lightChannels += int(d.RgbDevices[k].LedChannels)
	}
	clusterController := &common.ClusterController{
		Product:      d.Product,
		Serial:       d.Serial,
		LedChannels:  uint32(lightChannels),
		WriteColorEx: d.writeColorCluster,
	}

	cluster.Get().AddDeviceController(clusterController)
}

// setupOpenRGBController will create RGBController object for OpenRGB Client Integration
func (d *Device) setupOpenRGBController() {
	lightChannels := 0
	keys := make([]int, 0)

	// For proper packet positioning
	for k := range d.RgbDevices {
		lightChannels += int(d.RgbDevices[k].LedChannels)
		keys = append(keys, k)
	}
	sort.Ints(keys)

	controller := &common.OpenRGBController{
		Name:         d.Product,
		Vendor:       "Corsair", // Static value
		Description:  "OpenLinkHub Backend Device",
		FwVersion:    d.Firmware,
		Serial:       d.Serial,
		Location:     fmt.Sprintf("HID: %s", d.Path),
		Zones:        nil,
		Colors:       make([]byte, lightChannels*3),
		ActiveMode:   0,
		WriteColorEx: d.writeColorEx,
		DeviceType:   common.DeviceTypeCooler,
		ColorMode:    common.ColorModePerLed,
	}

	for _, k := range keys {
		zone := common.OpenRGBZone{
			Name:     d.RgbDevices[k].Name,
			NumLEDs:  uint32(d.RgbDevices[k].LedChannels),
			ZoneType: common.ZoneTypeLinear,
		}
		if d.RgbDevices[k].ContainsPump {
			zone.ZoneType = common.ZoneTypeMatrix
		}
		controller.Zones = append(controller.Zones, zone)
	}
	// Send it
	openrgb.AddDeviceController(controller)
}

// modifyOpenRGBController will modify existing controller
func (d *Device) modifyOpenRGBController() {
	ctrl := openrgb.GetDeviceController(d.Serial)
	if ctrl == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Warn("No such controller found")
		return
	}

	lightChannels := 0
	keys := make([]int, 0)

	// For proper packet positioning
	for k := range d.RgbDevices {
		lightChannels += int(d.RgbDevices[k].LedChannels)
		keys = append(keys, k)
	}
	sort.Ints(keys)

	ctrl.Colors = make([]byte, lightChannels*3)
	ctrl.Zones = nil

	for _, k := range keys {
		zone := common.OpenRGBZone{
			Name:    d.RgbDevices[k].Name,
			NumLEDs: uint32(d.RgbDevices[k].LedChannels),
		}
		ctrl.Zones = append(ctrl.Zones, zone)
	}
	openrgb.UpdateDeviceController(d.Serial, ctrl)
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	// Reset
	reset := map[int][]byte{}
	var buffer []byte
	lightChannels := 0

	s, l := 0, 0
	keys := make([]int, 0)
	for k, device := range d.RgbDevices {
		lightChannels += int(device.LedChannels)
		keys = append(keys, k)
		l++ // device has LED
		if device.RGB == "static" {
			s++ // led profile is set to static
		}
	}
	sort.Ints(keys)

	// Do we have any RGB component in the system?
	if lightChannels == 0 {
		logger.Log(logger.Fields{}).Info("No RGB compatible devices found")
		return
	}

	// Reset all channels
	color := &rgb.Color{Red: 0, Green: 0, Blue: 0, Brightness: 0}
	for i := 0; i < lightChannels; i++ {
		reset[i] = []byte{
			byte(color.Red),
			byte(color.Green),
			byte(color.Blue),
		}
	}

	buffer = rgb.SetColor(reset)
	d.writeColor(buffer)

	// OpenRGB
	if d.DeviceProfile.OpenRGBIntegration {
		logger.Log(logger.Fields{}).Info("Exiting setDeviceColor() due to OpenRGB client")
		return
	}

	// RGB Cluster
	if d.DeviceProfile.RGBCluster {
		logger.Log(logger.Fields{}).Info("Exiting setDeviceColor() due to RGB Cluster")
		return
	}

	// Are all devices under static mode?
	// In static mode, we only need to send color once;
	// there is no need for continuous packet sending.
	if s == l { // number of devices matches number of devices with static profile
		profile := d.GetRgbProfile("static")
		if profile == nil {
			return
		}

		profile.StartColor.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
		profileColor := rgb.ModifyBrightness(profile.StartColor)
		m := 0

		for _, k := range keys {
			var c *rgb.Color
			rgbOverride := d.getRgbOverride(k, 0)
			if rgbOverride != nil && rgbOverride.Enabled && d.RgbDevices[k].LedChannels > 0 {
				profileOverride := d.GetRgbProfile("static")
				if profileOverride == nil {
					return
				}
				profileOverride.StartColor = rgbOverride.RGBStartColor
				c = rgb.ModifyBrightness(profileOverride.StartColor)
			} else {
				c = profileColor
			}
			for i := 0; i < int(d.RgbDevices[k].LedChannels); i++ {
				reset[m] = []byte{
					byte(c.Red),
					byte(c.Green),
					byte(c.Blue),
				}
				m++
			}
		}
		buffer = rgb.SetColor(reset)
		d.writeColor(buffer) // Write color once
		return
	}

	go func(lightChannels int) {
		startTime := time.Now()
		d.activeRgb = rgb.Exit()

		// Generate random colors
		d.activeRgb.RGBStartColor = rgb.GenerateRandomColor(1)
		d.activeRgb.RGBEndColor = rgb.GenerateRandomColor(1)
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
					profile := d.GetRgbProfile(d.RgbDevices[k].RGB)
					if profile == nil {
						for i := 0; i < int(d.RgbDevices[k].LedChannels); i++ {
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

					index := 0
					rgbOverride := d.getRgbOverride(k, index)
					if rgbOverride != nil && rgbOverride.Enabled && d.RgbDevices[k].LedChannels > 0 {
						r.RGBStartColor = &rgbOverride.RGBStartColor
						r.RGBEndColor = &rgbOverride.RGBEndColor
						r.RgbModeSpeed = common.FClamp(rgbOverride.RgbModeSpeed, 0.1, 10)
					}

					// Brightness
					r.RGBBrightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
					r.RGBStartColor.Brightness = r.RGBBrightness
					r.RGBEndColor.Brightness = r.RGBBrightness
					r.ChannelId = k
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
					case "spiralrainbow":
						{
							r.SpiralRainbow(startTime)
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
					case "nebula":
						{
							r.Nebula(&startTime)
							buff = append(buff, r.Output...)
						}
					case "marquee":
						{
							r.Marquee(&startTime)
							buff = append(buff, r.Output...)
						}
					case "rotarystack":
						{
							r.RotaryStack(&startTime)
							buff = append(buff, r.Output...)
						}
					case "sequential":
						{
							r.Sequential(&startTime)
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

// getRgbDevices will get all RGB devices
func (d *Device) getRgbDevices() {
	var devices = make(map[int]*Devices)
	var m = 0
	amount := 7

	for i := 0; i < amount; i++ {
		if internalLedDevice, ok := d.internalLedDevices[i]; ok {
			if internalLedDevice.Total > 0 {
				rgbProfile := "static"
				label := "Set Label"
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
									if d.GetRgbProfile(rp) != nil { // Speed profile exists in configuration
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
	var devices = make(map[int]*Devices)
	var m = 0

	// Fans
	response := d.read(modeGetFans, "getDevices")
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
	response = d.read(modeGetTemperatures, "getDevices")
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

	response := d.write(modeSetSpeed, dataTypeSetSpeed, buffer, true, "setSpeed")
	if d.Debug {
		logger.Log(logger.Fields{"serial": d.Serial, "data": fmt.Sprintf("%2x", response), "len": len(response), "device": d.Product}).Info("setSpeed() - Speed")
	}

	if len(response) >= 4 {
		if response[5] != 0x07 && response[2] != 0x00 {
			if d.Debug {
				logger.Log(logger.Fields{"serial": d.Serial, "data": fmt.Sprintf("%2x", response), "len": len(response), "device": d.Product}).Info("setSpeed() - Speed Retry")
			}
			m := 0
			for {
				m++
				response = d.write(modeSetSpeed, dataTypeSetSpeed, buffer, true, "setSpeed")
				if d.Debug {
					logger.Log(logger.Fields{"serial": d.Serial, "data": fmt.Sprintf("%2x", response), "len": len(response), "device": d.Product, "retry": m}).Info("setSpeed() - Speed Retry")
				}
				if response[5] == 0x07 || response[2] == 0x00 || m > 20 {
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

// getRgbOverride will return RGBOverride object
func (d *Device) getRgbOverride(deviceId, subDeviceId int) *RGBOverride {
	if value, ok := d.DeviceProfile.RGBOverride[deviceId]; ok {
		if val, found := value[subDeviceId]; found {
			return &val
		}
	}
	return nil
}

// setRgbOverride will set RGBOverride object
func (d *Device) setRgbOverride(deviceId, subDeviceId int, rgbOverride RGBOverride) {
	if value, ok := d.DeviceProfile.RGBOverride[deviceId]; ok {
		value[subDeviceId] = rgbOverride
		d.DeviceProfile.RGBOverride[deviceId] = value
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
	d.timerSpeed = time.NewTicker(time.Duration(temperaturePullingInterval) * time.Millisecond)
	go func() {
		tmp := make(map[int]string)
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
			case <-d.timerSpeed.C:
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
							temp = temperatures.GetNVIDIAGpuTemperature(0)
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
							gpuTemp := temperatures.GetNVIDIAGpuTemperature(0)
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
					case temperatures.SensorTypeMultiGPU:
						{
							temp = temperatures.GetGpuTemperatureIndex(int(profiles.GPUIndex))
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "hwmonDeviceId": profiles.Device}).Warn("Unable to get hwmon temperature.")
							}
						}
					case temperatures.SensorTypeGlobalTemperature:
						{
							temp = stats.GetDeviceTemperature(profiles.Device, profiles.ChannelId)
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "hwmonDeviceId": profiles.Device}).Warn("Unable to get hwmon temperature.")
							}
						}
					}

					// All temps failed, default to 50
					if temp == 0 {
						temp = 50
					}

					if config.GetConfig().GraphProfiles {
						pumpValue := temperatures.Interpolate(profiles.Points[0], temp)
						fansValue := temperatures.Interpolate(profiles.Points[1], temp)

						pump := int(math.Round(float64(pumpValue)))
						fans := int(math.Round(float64(fansValue)))

						// Failsafe
						if fans < 20 && !profiles.ZeroRpm {
							fans = 20
						}

						if device.ContainsPump {
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

						cp := fmt.Sprintf("%s-%d-%f", device.Profile, device.ChannelId, temp)
						if ok := tmp[device.ChannelId]; ok != cp {
							tmp[device.ChannelId] = cp
							if device.ContainsPump {
								channelSpeeds[device.ChannelId] = byte(pump)
							} else {
								channelSpeeds[device.ChannelId] = byte(fans)
							}
							d.setSpeed(channelSpeeds, 0)
						}
						if d.Debug {
							logger.Log(logger.Fields{"serial": d.Serial, "pump": pump, "fans": fans, "temp": temp, "device": device.Name, "zeroRpm": profiles.ZeroRpm}).Info("updateDeviceSpeed()")
						}
					} else {
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

									if profile.Fans < 20 && !profiles.ZeroRpm {
										profile.Fans = 20
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
				}
			case <-d.speedRefreshChan:
				d.timerSpeed.Stop()
				return
			}
		}
	}()
}

// setHardwareMode will switch a device to hardware mode
func (d *Device) setHardwareMode() {
	_, err := d.transfer(cmdHardwareMode, nil, "setHardwareMode")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	_, err := d.transfer(cmdSoftwareMode, nil, "setHardwareMode")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// getDeviceType will set a type of AIO
func (d *Device) getDeviceType() {
	deviceType, err := d.transfer(cmdGetPumpVersion, nil, "getDeviceType")
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
	}
	pumpVersion := int16(deviceType[3])
	if d.Debug {
		logger.Log(logger.Fields{"serial": d.Serial, "data": fmt.Sprintf("%2x", deviceType)}).Info("getDeviceType() - Pump Version")
	}

	deviceType, err = d.transfer(cmdGetRadiatorType, nil, "getDeviceType")
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
	}
	radiatorSize := int16(binary.LittleEndian.Uint16(deviceType[3:5]))

	if d.Debug {
		logger.Log(logger.Fields{"serial": d.Serial, "data": fmt.Sprintf("%2x", deviceType)}).Info("getDeviceType() - Radiator size")
	}

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
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	// Close any RGB endpoint
	_, err := d.transfer(cmdCloseEndpoint, modeSetColor, "setColorEndpoint")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
	}

	// Open RGB endpoint
	_, err = d.transfer(cmdOpenColorEndpoint, modeSetColor, "setColorEndpoint")
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
	if d.Exit {
		return
	}

	// Channels
	channels := d.read(modeGetFans, "getDeviceData")
	if channels == nil {
		return
	}
	var m = 0

	// Speed
	response := d.read(modeGetSpeeds, "getDeviceData")
	if response == nil {
		return
	}

	amount := d.getChannelAmount(channels)
	sensorData := response[6:]

	if d.Debug {
		logger.Log(logger.Fields{"serial": d.Serial, "data": fmt.Sprintf("%2x", response), "amount": amount, "device": d.Product}).Info("getDeviceData() - Speed")
	}

	for i, s := 0, 0; i < amount; i, s = i+1, s+2 {
		if d.Exit {
			break
		}
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
	response = d.read(modeGetTemperatures, "getDeviceData")
	if response == nil {
		return
	}

	amount = d.getChannelAmount(response)
	sensorData = response[6:]
	for i, s := 0, 0; i < amount; i, s = i+1, s+3 {
		if d.Exit {
			break
		}
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

	// Update stats
	for key, value := range d.Devices {
		temperatureString := ""
		rpmString := ""

		if value.Rpm > 0 || value.Temperature > 0 {
			if value.Temperature > 0 {
				temperatureString = dashboard.GetDashboard().TemperatureToString(value.Temperature)
			}
			if value.Rpm > 0 {
				rpmString = fmt.Sprintf("%v RPM", value.Rpm)
			}
			stats.UpdateAIOStats(d.Serial, value.Name, temperatureString, rpmString, value.Label, key, value.Temperature)
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
	if _, ok := d.Devices[channelId]; !ok {
		return 0
	}

	d.Devices[channelId].Label = label
	d.saveDeviceProfile()
	return 1
}

// UpdateRGBDeviceLabel will set / update device label
func (d *Device) UpdateRGBDeviceLabel(channelId int, label string) uint8 {
	if _, ok := d.RgbDevices[channelId]; !ok {
		return 0
	}

	d.RgbDevices[channelId].Label = label
	d.saveDeviceProfile()
	return 1
}

// UpdateDeviceLcd will update device LCD
func (d *Device) UpdateDeviceLcd(_ int, mode uint8) uint8 {
	if d.HasLCD {
		value := d.DeviceProfile.LCDMode
		if mode == lcd.DisplayImage {
			if len(lcd.GetLcdImages()) == 0 {
				return 0
			}

			if len(d.DeviceProfile.LCDImage) == 0 {
				lcdImage := lcd.GetLcdImages()[0]
				d.DeviceProfile.LCDImage = lcdImage.Name
				d.LCDImage = &lcdImage
			}

			if lcd.GetLcdImage(d.DeviceProfile.LCDImage) == nil {
				lcdImage := lcd.GetLcdImages()[0]
				d.DeviceProfile.LCDImage = lcdImage.Name
				d.LCDImage = &lcdImage
			}

			// Stop lcd timer and switch to animation loop
			if d.lcdRefreshChan != nil {
				close(d.lcdRefreshChan)
			}

			d.lcdTimer.Stop()
			d.setupLCDImage()
		} else {
			// Reset if old value was Animation and new mode is not
			if value == lcd.DisplayImage && value != mode {
				d.setupLCD(true)
			}
		}

		d.DeviceProfile.LCDMode = mode
		d.saveDeviceProfile()
		return 1
	} else {
		return 2
	}
}

// UpdateDeviceLcdImage will update device LCD image
func (d *Device) UpdateDeviceLcdImage(_ int, image string) uint8 {
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

		d.DeviceProfile.LCDImage = image
		d.LCDImage = lcdImage
		d.saveDeviceProfile()
		return 1
	} else {
		return 0
	}
}

// UpdateDeviceLcdRotation will update device LCD rotation
func (d *Device) UpdateDeviceLcdRotation(_ int, rotation uint8) uint8 {
	if d.HasLCD {
		d.DeviceProfile.LCDRotation = rotation
		d.saveDeviceProfile()
		d.setLcdRotation()
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

// ChangeDeviceBrightnessValue will change device brightness via slider
func (d *Device) ChangeDeviceBrightnessValue(value uint8) uint8 {
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

// getTemperatureProbe will return all devices with a temperature probe
func (d *Device) getTemperatureProbe() {
	var probes []TemperatureProbe

	keys := make([]int, 0)
	for k := range d.Devices {
		keys = append(keys, k)
	}

	for _, k := range keys {
		if d.Devices[k].IsTemperatureProbe || d.Devices[k].HasTemps {
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

// setLcdRotation will change LCD rotation
func (d *Device) setLcdRotation() {
	if d.lcd != nil {
		lcdReport := []byte{0x03, 0x0c, d.DeviceProfile.LCDRotation, 0x01}
		_, err := d.lcd.SendFeatureReport(lcdReport)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId, "serial": d.Serial}).Error("Unable to change LCD rotation")
		}
	}
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

	if pf := d.GetRgbProfile(profileName); pf == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}

	pf := d.GetRgbProfile(profileName)
	if pf == nil {
		return 0
	}
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
	if d.DeviceProfile == nil {
		return 0
	}

	if d.DeviceProfile.OpenRGBIntegration {
		return 4
	}

	if d.DeviceProfile.RGBCluster {
		return 5
	}

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

// ProcessGetRgbOverride will get rgb override data
func (d *Device) ProcessGetRgbOverride(channelId, subDeviceId int) interface{} {
	return d.getRgbOverride(channelId, subDeviceId)
}

// ProcessSetOpenRgbIntegration will update OpenRGB integration status
func (d *Device) ProcessSetOpenRgbIntegration(enabled bool) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if d.DeviceProfile.RGBCluster {
		return 2
	}

	d.DeviceProfile.OpenRGBIntegration = enabled
	d.saveDeviceProfile() // Save profile
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
	return 1
}

// ProcessSetRgbCluster will update OpenRGB integration status
func (d *Device) ProcessSetRgbCluster(enabled bool) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	if d.DeviceProfile.OpenRGBIntegration {
		return 2
	}

	d.DeviceProfile.RGBCluster = enabled
	d.saveDeviceProfile() // Save profile
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB

	if enabled {
		lightChannels := 0
		for k := range d.RgbDevices {
			lightChannels += int(d.RgbDevices[k].LedChannels)
		}

		clusterController := &common.ClusterController{
			Product:      d.Product,
			Serial:       d.Serial,
			LedChannels:  uint32(lightChannels),
			WriteColorEx: d.writeColorCluster,
		}

		cluster.Get().AddDeviceController(clusterController)
	} else {
		cluster.Get().RemoveDeviceControllerBySerial(d.Serial)
	}
	return 1
}

// ProcessSetRgbOverride will update RGB override settings
func (d *Device) ProcessSetRgbOverride(channelId, subDeviceId int, enabled bool, startColor, endColor rgb.Color, speed float64) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	rgbOverride := d.getRgbOverride(channelId, subDeviceId)
	if rgbOverride == nil {
		return 0
	}

	if speed < 0 || speed > 10 {
		return 0
	}

	rgbOverride.Enabled = enabled
	rgbOverride.RGBStartColor = startColor
	rgbOverride.RGBEndColor = endColor
	rgbOverride.RgbModeSpeed = speed
	rgbOverride.RGBStartColor.Brightness = 1
	rgbOverride.RGBEndColor.Brightness = 1

	d.setRgbOverride(channelId, subDeviceId, *rgbOverride)
	d.saveDeviceProfile()
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

		d.resetLEDPorts()           // Reset LED ports
		d.saveDeviceProfile()       // Save profile
		d.setDeviceColor()          // Restart RGB
		d.modifyOpenRGBController() // Notify OpenRGB
		return 1
	} else {
		return 2 // No such free port
	}
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
		_, err := d.transfer(command, nil, "initLedPorts")
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "port": i}).Error("Failed to initialize LED ports")
		}
	}
	time.Sleep(time.Duration(ledInit) * time.Millisecond)
}

// resetLEDPorts will reset hub LED ports and configure currently connected LED device
func (d *Device) resetLEDPorts() {
	var buf []byte

	buf = append(buf, 0x0d)
	buf = append(buf, 0x00)
	buf = append(buf, 0x07)

	// Start at 1, since 0 is the pump, and iterate through all 6 physical connectors
	for i := 0; i <= 6; i++ {
		if z, ok := d.internalLedDevices[i]; ok {
			if z.Total > 0 {
				// Channel activation
				buf = append(buf, 0x01)
				// Fan LED command code, each LED device has different command code
				buf = append(buf, z.Command)
			} else {
				if deviceType, valid := d.DeviceProfile.CustomLEDs[i]; valid {
					if deviceType > 0 {
						externalDeviceType := d.getExternalLedDevice(d.DeviceProfile.CustomLEDs[i])
						if externalDeviceType == nil {
							buf = append(buf, 0x00)
						} else {
							// Channel activation
							buf = append(buf, 0x01)
							buf = append(buf, externalDeviceType.Command)
						}
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
	d.write(cmdSetLedPorts, nil, buf, false, "resetLEDPorts")
}

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	noOverride := false
	var defaultBrightness = uint8(100)
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"

	speedProfiles := make(map[int]string, len(d.Devices))
	rgbProfiles := make(map[int]string, len(d.Devices))
	labels := make(map[int]string, len(d.Devices))
	rgbLabels := make(map[int]string, len(d.Devices))
	customLEDs := make(map[int]int, len(d.Devices))
	rgbOverride := make(map[int]map[int]RGBOverride, len(d.RgbDevices))

	if d.DeviceProfile == nil || d.DeviceProfile.RGBOverride == nil {
		noOverride = true
	}

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
		if noOverride {
			rgbOverride[device.ChannelId] = map[int]RGBOverride{
				0: {
					Enabled: false,
					RGBStartColor: rgb.Color{
						Red:        0,
						Green:      255,
						Blue:       255,
						Brightness: 1,
						Hex:        "#00ffff",
					},
					RGBEndColor: rgb.Color{
						Red:        0,
						Green:      255,
						Blue:       255,
						Brightness: 1,
						Hex:        "#00ffff",
					},
					RgbModeSpeed: 3,
				},
			}
		} else {
			if _, ok := d.DeviceProfile.RGBOverride[device.ChannelId]; !ok {
				rgbOverride[device.ChannelId] = map[int]RGBOverride{
					0: {
						Enabled: false,
						RGBStartColor: rgb.Color{
							Red:        0,
							Green:      255,
							Blue:       255,
							Brightness: 1,
							Hex:        "#00ffff",
						},
						RGBEndColor: rgb.Color{
							Red:        0,
							Green:      255,
							Blue:       255,
							Brightness: 1,
							Hex:        "#00ffff",
						},
						RgbModeSpeed: 3,
					},
				}
			} else {
				rgbOverride[device.ChannelId] = d.DeviceProfile.RGBOverride[device.ChannelId]
			}
		}
	}

	for _, device := range d.Devices {
		labels[device.ChannelId] = device.Label
	}

	deviceProfile := &DeviceProfile{
		Product:            d.Product,
		Serial:             d.Serial,
		SpeedProfiles:      speedProfiles,
		RGBProfiles:        rgbProfiles,
		Labels:             labels,
		RGBLabels:          rgbLabels,
		Path:               profilePath,
		BrightnessSlider:   &defaultBrightness,
		OriginalBrightness: 100,
		RGBOverride:        rgbOverride,
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
		deviceProfile.LCDImage = ""
		d.DeviceProfile = deviceProfile
	} else {
		if d.DeviceProfile.BrightnessSlider == nil {
			deviceProfile.BrightnessSlider = &defaultBrightness
			d.DeviceProfile.BrightnessSlider = &defaultBrightness
		} else {
			deviceProfile.BrightnessSlider = d.DeviceProfile.BrightnessSlider
		}
		deviceProfile.OriginalBrightness = d.DeviceProfile.OriginalBrightness

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
		deviceProfile.LCDImage = d.DeviceProfile.LCDImage
		deviceProfile.MultiProfile = d.DeviceProfile.MultiProfile
		deviceProfile.MultiRGB = d.DeviceProfile.MultiRGB
		deviceProfile.OpenRGBIntegration = d.DeviceProfile.OpenRGBIntegration
		deviceProfile.RGBCluster = d.DeviceProfile.RGBCluster
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
		logger.Log(logger.Fields{"error": fileErr, "location": deviceProfile.Path}).Error("Unable to create new device profile")
		return
	}

	// WriteWrite JSON buffer to file
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

// read will read data from a device and return data as a byte array
func (d *Device) read(endpoint []byte, caller string) []byte {
	// Lock it
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	// Endpoint data
	var buffer []byte

	if d.Exit {
		return nil
	}

	// Close specified endpoint
	_, err := d.transfer(cmdCloseEndpoint, endpoint, caller)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to close endpoint")
		return nil
	}

	// Open endpoint
	_, err = d.transfer(cmdOpenEndpoint, endpoint, caller)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to open endpoint")
		return nil
	}

	// Read data from endpoint
	buffer, err = d.transfer(cmdRead, endpoint, caller)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read endpoint")
		return nil
	}

	// Close specified endpoint
	_, err = d.transfer(cmdCloseEndpoint, endpoint, caller)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to close endpoint")
		return nil
	}
	return buffer
}

// writeColor will write data to the device with a specific endpoint.
// writeColor does not require endpoint closing and opening like normal Write requires.
// Endpoint is open only once. Once the endpoint is open, color can be sent continuously.
func (d *Device) writeColor(data []byte) {
	// Lock it
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if d.Exit {
		return
	}

	buffer := make([]byte, len(dataTypeSetColor)+len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)+2))
	copy(buffer[headerWriteSize:headerWriteSize+len(dataTypeSetColor)], dataTypeSetColor)
	copy(buffer[headerWriteSize+len(dataTypeSetColor):], data)

	// Split packet into chunks
	chunks := common.ProcessMultiChunkPacket(buffer, maxBufferSizePerRequest)
	for i, chunk := range chunks {
		if i == 0 {
			// Initial packet is using cmdWriteColor
			_, err := d.transfer(cmdWriteColor, chunk, "writeColor")
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
			}
		} else {
			// Chunks don't use cmdWriteColor, they use static dataTypeSubColor
			_, err := d.transfer(dataTypeSubColor, chunk, "writeColor")
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to endpoint")
			}
		}
	}
}

// writeColorEx will write data to the device from OpenRGB client
func (d *Device) writeColorEx(data []byte, _ int) {
	if !d.DeviceProfile.OpenRGBIntegration {
		return
	}
	if d.Exit {
		return
	}

	// Copy data to avoid race conditions, since the caller might reuse the slice
	copyData := make([]byte, len(data))
	copy(copyData, data)

	select {
	case d.queue <- copyData:
	default:
	}
}

// writeColorCluster will write data to the device from cluster client
func (d *Device) writeColorCluster(data []byte, _ int) {
	if !d.DeviceProfile.RGBCluster {
		return
	}

	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if d.Exit {
		return
	}

	buffer := make([]byte, len(dataTypeSetColor)+len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)+2))
	copy(buffer[headerWriteSize:headerWriteSize+len(dataTypeSetColor)], dataTypeSetColor)
	copy(buffer[headerWriteSize+len(dataTypeSetColor):], data)

	// Split packet into chunks
	chunks := common.ProcessMultiChunkPacket(buffer, maxBufferSizePerRequest)
	for i, chunk := range chunks {
		if i == 0 {
			// Initial packet is using cmdWriteColor
			_, err := d.transfer(cmdWriteColor, chunk, "writeColor")
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
			}
		} else {
			// Chunks don't use cmdWriteColor, they use static dataTypeSubColor
			_, err := d.transfer(dataTypeSubColor, chunk, "writeColor")
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to endpoint")
			}
		}
	}
}

// clearQueue will clear queue
func (d *Device) clearQueue() {
	for {
		select {
		case <-d.queue:
		default:
			return
		}
	}
}

// startQueueWorker will initialize queue system and control packet flow towards the device
func (d *Device) startQueueWorker() {
	d.queue = make(chan []byte, 10)

	go func() {
		for data := range d.queue {
			d.deviceLock.Lock()

			if d.Exit {
				d.deviceLock.Unlock()
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
					_, err := d.transfer(cmdWriteColor, chunk, "writeColor")
					if err != nil {
						logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
					}
				} else {
					// Chunks don't use cmdWriteColor, they use static dataTypeSubColor
					_, err := d.transfer(dataTypeSubColor, chunk, "writeColor")
					if err != nil {
						logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to endpoint")
					}
				}
			}
			d.deviceLock.Unlock()
			time.Sleep(20 * time.Millisecond)
		}
	}()
}

// write will write data to the device with specific endpoint
func (d *Device) write(endpoint, bufferType, data []byte, extra bool, caller string) []byte {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

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
	if d.Exit {
		return bufferR
	}

	// Close endpoint
	_, err := d.transfer(cmdCloseEndpoint, endpoint, caller)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to close endpoint")
		return bufferR
	}

	// Open endpoint
	_, err = d.transfer(cmdOpenEndpoint, endpoint, caller)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to open endpoint")
		return bufferR
	}

	// Send it
	bufferR, err = d.transfer(cmdWrite, buffer, caller)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to endpoint")
		return bufferR
	}

	// Close endpoint
	_, err = d.transfer(cmdCloseEndpoint, endpoint, caller)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to close endpoint")
		return bufferR
	}

	return bufferR
}

// setupLCD will activate and configure LCD
func (d *Device) setupLCD(reload bool) {
	if reload {
		close(d.lcdImageChan)
	}
	d.lcdTimer = time.NewTicker(time.Duration(lcdRefreshInterval) * time.Millisecond)
	d.lcdRefreshChan = make(chan struct{})
	go func() {
		for {
			select {
			case <-d.lcdTimer.C:
				switch d.DeviceProfile.LCDMode {
				case lcd.DisplayCPU:
					{
						buffer := lcd.GenerateScreenImage(
							lcd.DisplayCPU,
							int(temperatures.GetCpuTemperature()),
							0,
							0,
							0,
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
						)
						d.transferToLcd(buffer)
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
						d.transferToLcd(buffer)
					}
				case lcd.DisplayArc:
					{
						val := 0
						arcType := 0
						sensor := 0
						switch lcd.GetArc().Sensor {
						case 0: // CPU temperature
							val = int(temperatures.GetCpuTemperature())
							break
						case 1: // GPU temperature
							val = int(temperatures.GetGpuTemperature())
							arcType = 1
							break
						case 2: // Liquid temperature
							val = int(d.getLiquidTemperature())
							arcType = 2
							sensor = 2
							break
						case 3: // CPU utilization
							val = int(systeminfo.GetCpuUtilization())
							sensor = 3
							break
						case 4: // GPU utilization
							val = systeminfo.GetGPUUtilization()
							sensor = 4
						}
						image := lcd.GenerateArcScreenImage(arcType, sensor, val)
						if image == nil {
							break // Fail
						}
						d.transferToLcd(image)
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
							d.transferToLcd(image)
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
								d.transferToLcd(image[i].Buffer)
								if i != imageLen-1 {
									if image[i].Delay > 0 {
										time.Sleep(time.Duration(image[i].Delay) * time.Millisecond)
									}
								}
							}
						}
					}
				}
			case <-d.lcdRefreshChan:
				d.lcdTimer.Stop()
				return
			}
		}
	}()
}

// loadLcdImage will load current LCD image
func (d *Device) loadLcdImage() uint8 {
	if len(d.DeviceProfile.LCDImage) == 0 {
		return 0
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", d.DeviceProfile.LCDImage); !m {
		return 0
	}

	lcdImage := lcd.GetLcdImage(d.DeviceProfile.LCDImage)
	if lcdImage == nil {
		d.DeviceProfile.LCDMode = 0
		d.saveDeviceProfile()
		d.setupLCD(false)
		return 0
	}
	d.LCDImage = lcdImage
	return 1
}

// setupLCDImage will set up lcd image
func (d *Device) setupLCDImage() {
	d.lcdImageChan = make(chan struct{})
	lcdImage := d.DeviceProfile.LCDImage
	if len(lcdImage) == 0 {
		logger.Log(logger.Fields{"serial": d.Serial}).Warn("Invalid LCD image")
		return
	}

	if d.LCDImage == nil {
		d.loadLcdImage()
	}

	go func() {
		for {
			select {
			default:
				if d.LCDImage.Frames > 1 {
					for i := 0; i < d.LCDImage.Frames; i++ {
						data := d.LCDImage.Buffer[i]
						buffer := data.Buffer
						delay := data.Delay

						d.transferToLcd(buffer)
						if delay > 0 {
							time.Sleep(time.Duration(delay) * time.Millisecond)
						} else {
							// Single frame, static image, generate 100ms of delay
							time.Sleep(100 * time.Millisecond)
						}
					}
				} else {
					data := d.LCDImage.Buffer[0]
					buffer := data.Buffer
					delay := data.Delay
					d.transferToLcd(buffer)
					if delay > 0 {
						time.Sleep(time.Duration(delay) * time.Millisecond)
					} else {
						// Single frame, static image, generate 100ms of delay
						time.Sleep(100 * time.Millisecond)
					}
				}
			case <-d.lcdImageChan:
				return
			}
		}
	}()
}

// transferToLcd will transfer data to LCD panel
func (d *Device) transferToLcd(buffer []byte) {
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
func (d *Device) transfer(endpoint, buffer []byte, caller string) ([]byte, error) {
	// Packet control, mandatory for this device
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Create read buffer
	bufferR := make([]byte, bufferSize)

	if d.Exit {
		// Create write buffer
		bufferW := make([]byte, bufferSizeWrite)
		bufferW[1] = 0x08
		endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
		copy(endpointHeaderPosition, endpoint)
		if len(buffer) > 0 {
			copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
		}

		// Send command to a device
		if _, err := d.dev.Write(bufferW); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "caller": caller}).Error("Unable to write to a device")
			return nil, err
		}
	} else {
		// Create write buffer
		bufferW := make([]byte, bufferSizeWrite)
		bufferW[1] = 0x08
		endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
		copy(endpointHeaderPosition, endpoint)
		if len(buffer) > 0 {
			copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
		}

		// Send command to a device
		if _, err := d.dev.Write(bufferW); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "caller": caller}).Error("Unable to write to a device")
			return bufferR, err
		}

		// Get data from a device
		if _, err := d.dev.Read(bufferR); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "caller": caller}).Error("Unable to read data from device")
			return bufferR, err
		}
	}
	return bufferR, nil
}
