package k55core

// Package: K55 CORE
// This is the primary package for K55 CORE.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/keyboards"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/macro"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/temperatures"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"math/big"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// DeviceProfile struct contains all device profile
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
	RGBProfile         string
	Label              string
	Layout             string
	Keyboards          map[string]*keyboards.Keyboard
	Profile            string
	PollingRate        int
	Profiles           []string
	DisableAltTab      bool
	DisableAltF4       bool
	DisableShiftTab    bool
	DisableWinKey      bool
	Performance        bool
}

type Device struct {
	Debug              bool
	dev                *hid.Device
	listener           *hid.Device
	Manufacturer       string `json:"manufacturer"`
	Product            string `json:"product"`
	Serial             string `json:"serial"`
	Firmware           string `json:"firmware"`
	activeRgb          *rgb.ActiveRGB
	UserProfiles       map[string]*DeviceProfile `json:"userProfiles"`
	Devices            map[int]string            `json:"devices"`
	DeviceProfile      *DeviceProfile
	OriginalProfile    *DeviceProfile
	Template           string
	VendorId           uint16
	ProductId          uint16
	Brightness         map[int]string
	PollingRates       map[int]string
	LEDChannels        int
	CpuTemp            float32
	GpuTemp            float32
	Layouts            []string
	Rgb                *rgb.RGB
	rgbMutex           sync.RWMutex
	Exit               bool
	timer              *time.Ticker
	timerKeepAlive     *time.Ticker
	autoRefreshChan    chan struct{}
	keepAliveChan      chan struct{}
	mutex              sync.Mutex
	UIKeyboard         string
	UIKeyboardRow      string
	KeyboardKey        *keyboards.Key
	PressLoop          bool
	ModifierIndex      *big.Int
	KeyAssignmentTypes map[int]string
	MacroTracker       map[int]uint16
	RGBModes           []string
	instance           *common.Device
}

var (
	pwd                     = ""
	cmdSoftwareMode         = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode         = []byte{0x01, 0x03, 0x00, 0x01}
	cmdActivateLed          = []byte{0x0d, 0x01, 0x22}
	cmdGetFirmware          = []byte{0x02, 0x13}
	dataTypeSetColor        = []byte{0x12, 0x00}
	dataTypeSubColor        = []byte{0x07, 0x00}
	cmdWriteColor           = []byte{0x06, 0x01}
	cmdSetPollingRate       = []byte{0x01, 0x01, 0x00}
	cmdKeepAlive            = []byte{0x12}
	cmdOpenEndpoint         = []byte{0x0d, 0x02, 0x02}
	cmdCloseEndpoint        = []byte{0x05, 0x01, 0x02}
	cmdKeyAssignment        = []byte{0x06, 0x02}
	cmdKeyAssignmentNext    = []byte{0x07, 0x02}
	cmdWritePerformance     = []byte{0x01}
	cmdPerformance          = []byte{0x01, 0x4a, 0x00}
	deviceRefreshInterval   = 1000
	deviceKeepAlive         = 20000
	transferTimeout         = 500
	bufferSize              = 64
	bufferSizeWrite         = bufferSize + 1
	headerSize              = 2
	headerWriteSize         = 4
	maxBufferSizePerRequest = 61
	colorPacketLength       = 35
	keyboardKey             = "k55core-default"
	defaultLayout           = "k55core-default-US"
	KeyAssignment           = 125
	maxKeyAssignmentLen     = 61
	rgbModes                = []string{
		"circle",
		"circleshift",
		"colorpulse",
		"colorshift",
		"colorwarp",
		"cpu-temperature",
		"flickering",
		"gpu-temperature",
		"keyboard",
		"off",
		"rainbow",
		"rotator",
		"spinner",
		"static",
		"storm",
		"watercolor",
		"wave",
	}
)

func Init(vendorId, productId uint16, _, path string) *common.Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	dev, err := hid.OpenPath(path)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId, "path": path}).Error("Unable to open HID device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:       dev,
		Template:  "k55core.html",
		VendorId:  vendorId,
		ProductId: productId,
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
		Product:         "K55 CORE RGB",
		LEDChannels:     10,
		Layouts:         keyboards.GetLayouts(keyboardKey),
		autoRefreshChan: make(chan struct{}),
		keepAliveChan:   make(chan struct{}),
		UIKeyboard:      "keyboard-6",
		UIKeyboardRow:   "keyboard-row-25",
		RGBModes:        rgbModes,
		PollingRates: map[int]string{
			0: "Not Set",
			1: "125 Hz / 8 msec",
			2: "250 Hu / 4 msec",
			3: "500 Hz / 2 msec",
			4: "1000 Hz / 1 msec",
		},
		KeyAssignmentTypes: map[int]string{
			0:  "None",
			1:  "Media Keys",
			3:  "Keyboard",
			9:  "Mouse",
			10: "Macro",
		},
		MacroTracker: make(map[int]uint16),
	}

	d.getDebugMode()       // Debug mode
	d.getManufacturer()    // Manufacturer
	d.getSerial()          // Serial
	d.loadRgb()            // Load RGB
	d.setSoftwareMode()    // Activate software mode
	d.initLeds()           // Init LED ports
	d.getDeviceFirmware()  // Firmware
	d.loadDeviceProfiles() // Load all device profiles
	d.saveDeviceProfile()  // Save profile
	d.setAutoRefresh()     // Set auto device refresh
	d.setKeepAlive()       // Keepalive
	d.setDeviceColor()     // Device color
	d.setupPerformance()   // Performance
	d.backendListener()    // Control buttons
	d.createDevice()       // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeK55Core,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-keyboard.svg",
		Instance:    d,
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
	d.timerKeepAlive.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.autoRefreshChan != nil {
				close(d.autoRefreshChan)
			}
			close(d.keepAliveChan)
		})
	}()

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
	d.timerKeepAlive.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.autoRefreshChan != nil {
				close(d.autoRefreshChan)
			}
			close(d.keepAliveChan)
		})
	}()
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 2
}

// setupKeyAssignment will setup keyboard keys
func (d *Device) setupKeyAssignment() {
	if d.DeviceProfile == nil {
		return
	}
	if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; !ok {
		return
	}

	buf := make([]byte, KeyAssignment)
	for i := 0; i < KeyAssignment; i++ {
		buf[i] = 0x01
	}
	for _, value := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
		for _, key := range value.Keys {
			var val byte = 0x00
			if key.Default {
				val = 0x01
			}
			buf[key.KeyData[0]] = val
		}
	}
	d.writeKeyAssignment(buf)
}

// getKeyData will return key data for given key hash
func (d *Device) getKeyData(keyHash string) *keyboards.Key {
	for _, value := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
		for _, key := range value.Keys {
			for _, hash := range key.KeyHash {
				if hash == keyHash {
					return &key
				}
			}
		}
	}
	return nil
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

// getManufacturer will return device manufacturer
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
}

// setupPerformance will set up keyboard performance mode
func (d *Device) setupPerformance() {
	if d.DeviceProfile == nil {
		return
	}

	base := byte(0)
	if d.DeviceProfile.Performance {
		if d.DeviceProfile.DisableWinKey {
			base = base + 1
		}

		if d.DeviceProfile.DisableAltTab {
			base = base + 2
		}

		if d.DeviceProfile.DisableAltF4 {
			base = base + 4
		}

		if d.DeviceProfile.DisableShiftTab {
			base = base + 8
		}
	}

	buf := make([]byte, 1)
	buf[0] = base
	_, err := d.transfer(cmdPerformance, buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to setup keyboard performance")
	}
	d.setupKeyAssignment()

	control := make(map[int][]byte, 2)
	if d.DeviceProfile.Performance {
		control = map[int][]byte{
			0: {0x45, 0x00, 0x01},
		}
	} else {
		control = map[int][]byte{
			0: {0x45, 0x00, 0x00},
		}
	}

	for i := 0; i < len(control); i++ {
		_, err := d.transfer(cmdWritePerformance, control[i])
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to setup keyboard performance")
		}
	}

	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
}

// getManufacturer will return device manufacturer
func (d *Device) getManufacturer() {
	manufacturer, err := d.dev.GetMfrStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to get manufacturer")
	}
	d.Manufacturer = manufacturer
}

// getSerial will return device serial number
func (d *Device) getSerial() {
	serial, err := d.dev.GetSerialNbr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to get device serial number")
	}
	d.Serial = serial
}

// setHardwareMode will switch a device to hardware mode
func (d *Device) setHardwareMode() {
	_, err := d.transfer(cmdHardwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	_, err := d.transfer(cmdSoftwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	fw, err := d.transfer(
		cmdGetFirmware,
		nil,
	)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to a device")
	}

	v1, v2, v3 := int(fw[3]), int(fw[4]), int(binary.LittleEndian.Uint16(fw[5:7]))
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

// initLeds will initialize LED ports
func (d *Device) initLeds() {
	_, err := d.transfer(cmdActivateLed, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
	// We need to wait around 500 ms for physical ports to re-initialize
	// After that we can grab any new connected / disconnected device values
	time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
}

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	var defaultBrightness = uint8(100)
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"
	keyboardMap := make(map[string]*keyboards.Keyboard)

	deviceProfile := &DeviceProfile{
		Product:            d.Product,
		Serial:             d.Serial,
		Path:               profilePath,
		BrightnessSlider:   &defaultBrightness,
		OriginalBrightness: 100,
	}

	// First save, assign saved profile to a device
	if d.DeviceProfile == nil {
		// RGB, Label
		deviceProfile.RGBProfile = "keyboard"
		deviceProfile.Label = "Keyboard"
		deviceProfile.Active = true
		keyboardMap["default"] = keyboards.GetKeyboard(defaultLayout)
		deviceProfile.Keyboards = keyboardMap
		deviceProfile.Profile = "default"
		deviceProfile.Profiles = []string{"default"}
		deviceProfile.Layout = "US"
		deviceProfile.PollingRate = 4
	} else {
		if d.DeviceProfile.BrightnessSlider == nil {
			deviceProfile.BrightnessSlider = &defaultBrightness
			d.DeviceProfile.BrightnessSlider = &defaultBrightness
		} else {
			deviceProfile.BrightnessSlider = d.DeviceProfile.BrightnessSlider
		}

		if len(d.DeviceProfile.Layout) == 0 {
			deviceProfile.Layout = "US"
		} else {
			deviceProfile.Layout = d.DeviceProfile.Layout
		}

		// Upgrade process
		currentLayout := fmt.Sprintf("%s-%s", keyboardKey, d.DeviceProfile.Layout)
		layout := keyboards.GetKeyboard(currentLayout)
		if layout == nil {
			return
		}
		if d.DeviceProfile.Keyboards["default"].Version != layout.Version {
			logger.Log(
				logger.Fields{
					"current":  d.DeviceProfile.Keyboards["default"].Version,
					"expected": layout.Version,
					"serial":   d.Serial,
				},
			).Info("Upgrading keyboard profile version")
			d.DeviceProfile.Keyboards["default"] = layout
		} else {
			logger.Log(
				logger.Fields{
					"current":  d.DeviceProfile.Keyboards["default"].Version,
					"expected": layout.Version,
					"serial":   d.Serial,
				},
			).Info("Keyboard profile version is OK")
		}

		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.Brightness = d.DeviceProfile.Brightness
		deviceProfile.RGBProfile = d.DeviceProfile.RGBProfile
		deviceProfile.Label = d.DeviceProfile.Label
		deviceProfile.Profile = d.DeviceProfile.Profile
		deviceProfile.Profiles = d.DeviceProfile.Profiles
		deviceProfile.Keyboards = d.DeviceProfile.Keyboards
		deviceProfile.OriginalBrightness = d.DeviceProfile.OriginalBrightness
		deviceProfile.PollingRate = d.DeviceProfile.PollingRate
		deviceProfile.DisableAltTab = d.DeviceProfile.DisableAltTab
		deviceProfile.DisableAltF4 = d.DeviceProfile.DisableAltF4
		deviceProfile.DisableShiftTab = d.DeviceProfile.DisableShiftTab
		deviceProfile.DisableWinKey = d.DeviceProfile.DisableWinKey
		deviceProfile.Performance = d.DeviceProfile.Performance

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
		logger.Log(logger.Fields{"error": fileErr, "location": deviceProfile.Path}).Error("Unable to create new device profile")
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

// keepAlive will keep a device alive
func (d *Device) keepAlive() {
	if d.Exit {
		return
	}
	_, err := d.transfer(cmdKeepAlive, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
	}
}

// setAutoRefresh will refresh device data
func (d *Device) setKeepAlive() {
	d.timerKeepAlive = time.NewTicker(time.Duration(deviceKeepAlive) * time.Millisecond)
	go func() {
		for {
			select {
			case <-d.timerKeepAlive.C:
				d.keepAlive()
			case <-d.keepAliveChan:
				d.timerKeepAlive.Stop()
				return
			}
		}
	}()
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
			case <-d.autoRefreshChan:
				d.timer.Stop()
				return
			}
		}
	}()
}

// setCpuTemperature will store current CPU temperature
func (d *Device) setTemperatures() {
	d.CpuTemp = temperatures.GetCpuTemperature()
	d.GpuTemp = temperatures.GetGpuTemperature()
}

// UpdateDeviceLabel will set / update device label
func (d *Device) UpdateDeviceLabel(_ int, label string) uint8 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.DeviceProfile.Label = label
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

// UpdatePollingRate will set device polling rate
func (d *Device) UpdatePollingRate(pullingRate int) uint8 {
	if _, ok := d.PollingRates[pullingRate]; ok {
		if d.DeviceProfile == nil {
			return 0
		}
		d.Exit = true
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true // Exit current RGB mode
			d.activeRgb = nil
		}
		time.Sleep(40 * time.Millisecond)

		d.DeviceProfile.PollingRate = pullingRate
		d.saveDeviceProfile()
		buf := make([]byte, 1)
		buf[0] = byte(pullingRate)
		_, err := d.transfer(cmdSetPollingRate, buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to set polling rate")
			return 0
		}
		return 1
	}
	return 0
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
func (d *Device) UpdateRgbProfile(_ int, profile string) uint8 {
	if d.GetRgbProfile(profile) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}
	d.DeviceProfile.RGBProfile = profile // Set profile
	d.saveDeviceProfile()                // Save profile
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
	return 1

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

		newProfile := profile
		newProfile.Active = true
		d.DeviceProfile = newProfile
		d.saveDeviceProfile()
		d.setDeviceColor()
		return 1
	}
	return 0
}

// ChangeKeyboardLayout will change keyboard layout
func (d *Device) ChangeKeyboardLayout(layout string) uint8 {
	layouts := keyboards.GetLayouts(keyboardKey)
	if len(layouts) < 1 {
		return 2
	}

	if slices.Contains(layouts, layout) {
		if d.DeviceProfile != nil {
			if _, ok := d.DeviceProfile.Keyboards["default"]; ok {
				layoutKey := fmt.Sprintf("%s-%s", keyboardKey, layout)
				keyboardLayout := keyboards.GetKeyboard(layoutKey)
				if keyboardLayout == nil {
					logger.Log(logger.Fields{"serial": d.Serial}).Error("Trying to apply non-existing keyboard layout")
					return 2
				}

				d.DeviceProfile.Keyboards["default"] = keyboardLayout
				d.DeviceProfile.Layout = layout
				d.saveDeviceProfile()
				return 1
			}
		} else {
			logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is null")
			return 0
		}
	} else {
		logger.Log(logger.Fields{"serial": d.Serial}).Warn("No such layout")
		return 2
	}
	return 0
}

// getCurrentKeyboard will return current active keyboard
func (d *Device) getCurrentKeyboard() *keyboards.Keyboard {
	if keyboard, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
		return keyboard
	}
	return nil
}

// SaveDeviceProfile will save a new keyboard profile
func (d *Device) SaveDeviceProfile(profileName string, new bool) uint8 {
	if new {
		if d.DeviceProfile == nil {
			return 0
		}

		if slices.Contains(d.DeviceProfile.Profiles, profileName) {
			return 2
		}

		if _, ok := d.DeviceProfile.Keyboards[profileName]; ok {
			return 2
		}

		d.DeviceProfile.Profiles = append(d.DeviceProfile.Profiles, profileName)
		d.DeviceProfile.Keyboards[profileName] = d.getCurrentKeyboard()
		d.saveDeviceProfile()
		return 1
	} else {
		d.saveDeviceProfile()
		return 1
	}
}

// UpdateKeyboardProfile will change keyboard profile
func (d *Device) UpdateKeyboardProfile(profileName string) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if !slices.Contains(d.DeviceProfile.Profiles, profileName) {
		return 2
	}

	if _, ok := d.DeviceProfile.Keyboards[profileName]; !ok {
		return 2
	}

	d.DeviceProfile.Profile = profileName
	d.saveDeviceProfile()
	// RGB reset
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor()
	return 1
}

// DeleteKeyboardProfile will delete keyboard profile
func (d *Device) DeleteKeyboardProfile(profileName string) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if profileName == "default" {
		return 3
	}

	if !slices.Contains(d.DeviceProfile.Profiles, profileName) {
		return 2
	}

	if _, ok := d.DeviceProfile.Keyboards[profileName]; !ok {
		return 2
	}

	index := common.IndexOfString(d.DeviceProfile.Profiles, profileName)
	if index < 0 {
		return 0
	}

	d.DeviceProfile.Profile = "default"
	d.DeviceProfile.Profiles = append(d.DeviceProfile.Profiles[:index], d.DeviceProfile.Profiles[index+1:]...)
	delete(d.DeviceProfile.Keyboards, profileName)

	d.saveDeviceProfile()
	// RGB reset
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor()
	return 1
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

// ProcessGetKeyboardKey will get key data
func (d *Device) ProcessGetKeyboardKey(keyId int) interface{} {
	for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
		for keyIndex, key := range row.Keys {
			if keyIndex == keyId {
				return key
			}
		}
	}
	return nil
}

// ProcessGetKeyAssignmentTypes will get KeyAssignmentTypes
func (d *Device) ProcessGetKeyAssignmentTypes() interface{} {
	return d.KeyAssignmentTypes
}

// ProcessGetKeyboardPerformance will get keyboard performance values
func (d *Device) ProcessGetKeyboardPerformance() interface{} {
	values := []common.KeyboardPerformance{
		{
			Name:     "Disable Win Key",
			Type:     "checkbox",
			Value:    d.DeviceProfile.DisableWinKey,
			Internal: "perf_winKey",
		},
		{
			Name:     "Disable Shift + Tab",
			Type:     "checkbox",
			Value:    d.DeviceProfile.DisableShiftTab,
			Internal: "perf_shiftTab",
		},
		{
			Name:     "Disable Alt + Tab",
			Type:     "checkbox",
			Value:    d.DeviceProfile.DisableAltTab,
			Internal: "perf_altTab",
		},
		{
			Name:     "Disable Alt + F4",
			Type:     "checkbox",
			Value:    d.DeviceProfile.DisableAltF4,
			Internal: "perf_altF4",
		},
	}
	return values
}

// ProcessSetKeyboardPerformance will set keyboard performance values
func (d *Device) ProcessSetKeyboardPerformance(performance common.KeyboardPerformanceData) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	d.DeviceProfile.DisableWinKey = performance.WinKey
	d.DeviceProfile.DisableShiftTab = performance.ShiftTab
	d.DeviceProfile.DisableAltF4 = performance.AltF4
	d.DeviceProfile.DisableAltTab = performance.AltTab
	d.saveDeviceProfile()
	d.setupPerformance()
	return 1
}

// UpdateDeviceKeyAssignment will update device key assignments
func (d *Device) UpdateDeviceKeyAssignment(keyIndex int, keyAssignment inputmanager.KeyAssignment) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; !ok {
		return 0
	}
	for rowId, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
		for keyId, key := range row.Keys {
			if keyIndex == keyId {
				if key.OnlyColor {
					return 2
				}
				key.Default = keyAssignment.Default
				key.ActionType = keyAssignment.ActionType
				key.ActionCommand = keyAssignment.ActionCommand
				key.ActionHold = keyAssignment.ActionHold
				d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row[rowId].Keys[keyId] = key
				d.saveDeviceProfile()
				d.setupKeyAssignment()
				return 1
			}
		}
	}
	return 0
}

// UpdateDeviceColor will update device color based on selected input
func (d *Device) UpdateDeviceColor(keyId, keyOption int, color rgb.Color) uint8 {
	zoneId := -1
	switch keyOption {
	case 0:
		{
			for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
				for keyIndex, key := range row.Keys {
					if keyIndex == keyId {
						zoneId = key.Zone
						break
					}
				}
			}

			cl := rgb.Color{
				Red:        color.Red,
				Green:      color.Green,
				Blue:       color.Blue,
				Brightness: 0,
			}

			zone := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Zones[zoneId]
			zone.Color = cl
			d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Zones[zoneId] = zone
			if d.activeRgb != nil {
				d.activeRgb.Exit <- true // Exit current RGB mode
				d.activeRgb = nil
			}
			d.setDeviceColor() // Restart RGB
			return 1
		}
	case 2:
		{
			for zoneIndex, zone := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Zones {
				zone.Color = rgb.Color{
					Red:        color.Red,
					Green:      color.Green,
					Blue:       color.Blue,
					Brightness: 0,
				}
				d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Zones[zoneIndex] = zone
			}
			if d.activeRgb != nil {
				d.activeRgb.Exit <- true // Exit current RGB mode
				d.activeRgb = nil
			}
			d.setDeviceColor() // Restart RGB
			return 1
		}
	}
	return 0
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	if d.DeviceProfile == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to set color. DeviceProfile is null!")
		return
	}

	if d.DeviceProfile.RGBProfile == "keyboard" {
		var buf = make([]byte, colorPacketLength)
		keys := make([]int, 0, len(d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Zones))
		for k := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Zones {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		i := 0
		for _, k := range keys {
			colors := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Zones[k].Color
			colors.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
			profileColor := rgb.ModifyBrightness(colors)
			buf[i] = byte(profileColor.Red)
			buf[i+1] = byte(profileColor.Green)
			buf[i+2] = byte(profileColor.Blue)
			i += 3
		}
		d.writeColor(buf) // Write color once
		return
	}

	if d.DeviceProfile.RGBProfile == "static" {
		profile := d.GetRgbProfile("static")
		if profile == nil {
			return
		}

		profile.StartColor.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
		profileColor := rgb.ModifyBrightness(profile.StartColor)
		var buf = make([]byte, colorPacketLength)
		keys := make([]int, 0, len(d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Zones))
		for k := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Zones {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		i := 0
		for range keys {
			buf[i] = byte(profileColor.Red)
			buf[i+1] = byte(profileColor.Green)
			buf[i+2] = byte(profileColor.Blue)
			i += 3
		}
		d.writeColor(buf) // Write color once
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

				rgbCustomColor := true
				profile := d.GetRgbProfile(d.DeviceProfile.RGBProfile)
				if profile == nil {
					for i := 0; i < d.LEDChannels; i++ {
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
					d.LEDChannels,
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

				switch d.DeviceProfile.RGBProfile {
				case "off":
					{
						for n := 0; n < d.LEDChannels; n++ {
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

				// Send it
				var buf = make([]byte, colorPacketLength)
				keys := make([]int, 0, len(d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Zones))
				for k := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Zones {
					keys = append(keys, k)
				}
				sort.Ints(keys)
				i := 0
				for range keys {
					buf[i] = buff[i]
					buf[i+1] = buff[i+1]
					buf[i+2] = buff[i+2]
					i += 3
				}
				d.writeColor(buf)
				time.Sleep(40 * time.Millisecond)
			}
		}
	}(d.LEDChannels)
}

// writeColor will write data to the device with a specific endpoint.
// writeColor does not require endpoint closing and opening like normal Write requires.
// Endpoint is open only once. Once the endpoint is open, color can be sent continuously.
func (d *Device) writeColor(data []byte) {
	if d.Exit {
		return
	}

	buff := make([]byte, 0)
	if d.DeviceProfile.Performance {
		buff = append(buff, []byte{0xff, 0xff, 0xff}...)
	} else {
		buff = append(buff, []byte{0x00, 0x00, 0x00}...)
	}
	buff = append(buff, data...)

	buffer := make([]byte, len(dataTypeSetColor)+len(buff)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(colorPacketLength))
	copy(buffer[headerWriteSize:headerWriteSize+len(dataTypeSetColor)], dataTypeSetColor)
	copy(buffer[headerWriteSize+len(dataTypeSetColor):], buff)

	// Split packet into chunks
	chunks := common.ProcessMultiChunkPacket(buffer, maxBufferSizePerRequest)
	for i, chunk := range chunks {
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

// writeKeyAssignment will write Key Assignment data
func (d *Device) writeKeyAssignment(data []byte) {
	buffer := make([]byte, len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	copy(buffer[headerWriteSize:], data)

	// Open endpoint
	_, err := d.transfer(cmdOpenEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to open write endpoint")
		return
	}

	// Split packet into chunks
	chunks := common.ProcessMultiChunkPacket(buffer, maxKeyAssignmentLen)
	for i, chunk := range chunks {
		if i == 0 {
			_, err := d.transfer(cmdKeyAssignment, chunk)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
			}
		} else {
			_, err := d.transfer(cmdKeyAssignmentNext, chunk)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to endpoint")
			}
		}
	}

	// Close endpoint
	_, err = d.transfer(cmdCloseEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close write endpoint")
		return
	}
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

// backendListener will listen for events from the control buttons
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

				d.triggerKeyAssignment(data)
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()
}

// addToMacroTracker adds or updates an entry in MacroTracker
func (d *Device) addToMacroTracker(key int, value uint16) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.MacroTracker == nil {
		d.MacroTracker = make(map[int]uint16)
	}
	d.MacroTracker[key] = value
}

// deleteFromMacroTracker deletes an entry from MacroTracker
func (d *Device) deleteFromMacroTracker(key int) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.MacroTracker == nil || len(d.MacroTracker) == 0 {
		return
	}
	delete(d.MacroTracker, key)
}

// releaseMacroTracker will release current MacroTracker
func (d *Device) releaseMacroTracker() {
	d.mutex.Lock()
	if d.MacroTracker == nil {
		d.mutex.Unlock()
		return
	}
	keys := make([]int, 0, len(d.MacroTracker))
	for key := range d.MacroTracker {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	d.mutex.Unlock()

	for _, key := range keys {
		inputmanager.InputControlKeyboardHold(d.MacroTracker[key], false)
		d.deleteFromMacroTracker(key)
	}
}

// triggerKeyAssignment will trigger key assignment if defined
func (d *Device) triggerKeyAssignment(value []byte) {
	raw := make([]byte, len(value))
	if value[1] == 0x02 {
		raw = value[2:22]
	}
	for i, j := 0, len(raw)-1; i < j; i, j = i+1, j-1 {
		raw[i], raw[j] = raw[j], raw[i]
	}
	val := new(big.Int).SetBytes(raw)

	// Check if we have any queue in macro tracker. If yes, release those keys
	if len(d.MacroTracker) > 0 {
		d.releaseMacroTracker()
	}

	if d.ModifierIndex != val {
		if d.KeyboardKey != nil {
			switch d.KeyboardKey.ActionType {
			case 1, 3:
				inputmanager.InputControlKeyboard(d.KeyboardKey.ActionCommand, d.PressLoop)
				break
			}
		}
		d.KeyboardKey = nil
	}
	d.ModifierIndex = val
	if val.Cmp(big.NewInt(0)) > 0 {
		if d.Debug {
			logger.Log(logger.Fields{"keyHash": val.String(), "vendorId": d.VendorId, "serial": d.Serial}).Error("Logging key hash")
		}
		key := d.getKeyData(val.String())
		if key == nil {
			return
		}

		// Brightness
		if key.ActionType == 11 {
			if *d.DeviceProfile.BrightnessSlider >= 100 {
				*d.DeviceProfile.BrightnessSlider = 0
			} else {
				*d.DeviceProfile.BrightnessSlider += 20
			}

			d.saveDeviceProfile()
			if d.activeRgb != nil {
				d.activeRgb.Exit <- true // Exit current RGB mode
				d.activeRgb = nil
			}
			d.setDeviceColor() // Restart RGB
			return
		}

		// Performance Lock
		if key.IsLock {
			d.DeviceProfile.Performance = !d.DeviceProfile.Performance
			d.saveDeviceProfile()
			d.setupPerformance()
			return
		}

		if key.Default {
			return // Default key action
		}

		if key.OnlyColor {
			return // Color only
		}

		// Process it
		switch key.ActionType {
		case 1, 3:
			if key.ActionHold {
				d.KeyboardKey = key
			}
			inputmanager.InputControlKeyboard(key.ActionCommand, key.ActionHold)
			break
		case 9:
			inputmanager.InputControlMouse(key.ActionCommand)
			break
		case 10:
			macroProfile := macro.GetProfile(int(key.ActionCommand))
			if macroProfile == nil {
				logger.Log(logger.Fields{"serial": d.Serial}).Error("Invalid macro profile")
				return
			}
			for i := 0; i < len(macroProfile.Actions); i++ {
				if v, valid := macroProfile.Actions[i]; valid {
					// Add to macro tracker for easier release
					if v.ActionHold {
						d.addToMacroTracker(i, v.ActionCommand)
					}

					switch v.ActionType {
					case 1, 3:
						inputmanager.InputControlKeyboard(v.ActionCommand, v.ActionHold)
						break
					case 9:
						inputmanager.InputControlMouse(v.ActionCommand)
						break
					case 5:
						if v.ActionDelay > 0 {
							time.Sleep(time.Duration(v.ActionDelay) * time.Millisecond)
						}
						break
					}
				}
			}
			break
		}
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint, buffer []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	d.mutex.Lock()
	defer d.mutex.Unlock()

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
		return bufferR, err
	}

	// Get data from a device
	if _, err := d.dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		return bufferR, err
	}

	return bufferR, nil
}
