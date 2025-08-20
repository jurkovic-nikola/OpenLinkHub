package k100airW

// Package: K100 AIR Wireless
// This is the primary package for K100 AIR Wireless.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/keyboards"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/stats"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/sstallion/go-hid"
)

// DeviceProfile struct contains all device profile
type DeviceProfile struct {
	Active               bool
	Path                 string
	Product              string
	Serial               string
	LCDMode              uint8
	LCDRotation          uint8
	Brightness           uint8
	RGBProfile           string
	SlipstreamRGBProfile string
	Label                string
	Layout               string
	Keyboards            map[string]*keyboards.Keyboard
	Profile              string
	Profiles             []string
	ControlDial          int
	BrightnessLevel      uint16
	SleepMode            int
}

type Device struct {
	Debug              bool
	dev                *hid.Device
	listener           *hid.Device
	Manufacturer       string `json:"manufacturer"`
	Product            string `json:"product"`
	Serial             string `json:"serial"`
	Firmware           string `json:"firmware"`
	DongleFirmware     string `json:"dongleFirmware"`
	activeRgb          *rgb.ActiveRGB
	UserProfiles       map[string]*DeviceProfile `json:"userProfiles"`
	Devices            map[int]string            `json:"devices"`
	DeviceProfile      *DeviceProfile
	OriginalProfile    *DeviceProfile
	Template           string
	VendorId           uint16
	Brightness         map[int]string
	CpuTemp            float32
	GpuTemp            float32
	Layouts            []string
	ProductId          uint16
	SlipstreamId       uint16
	ControlDialOptions map[int]string
	RGBModes           map[string]string
	SleepModes         map[int]string
	KeyAmount          int
	Connected          bool
	Rgb                *rgb.RGB
	rgbMutex           sync.RWMutex
	Endpoint           byte
	mutex              sync.Mutex
	Exit               bool
	UIKeyboard         string
	UIKeyboardRow      string
	BatteryLevel       uint16
	KeyAssignment      map[int]inputmanager.KeyAssignment
	InputActions       map[uint16]inputmanager.InputAction
	PressLoop          bool
	keyAssignmentFile  string
}

var (
	pwd                     = ""
	cmdCloseEndpoint        = []byte{0x05, 0x01, 0x01}
	cmdSoftwareMode         = []byte{0x01, 0x03, 0x00, 0x06}
	cmdHardwareMode         = []byte{0x01, 0x03, 0x00, 0x01}
	cmdOpenColorEndpoint    = []byte{0x0d, 0x01}
	cmdSetLeds              = []byte{0x13}
	cmdInitProtocol         = []byte{0x0b, 0x65, 0x6d}
	cmdFlush                = []byte{0x15, 0x01}
	cmdRead                 = []byte{0x09, 0x01}
	cmdActivateLed          = []byte{0x65, 0x6d}
	cmdBrightness           = []byte{0x01, 0x02, 0x00}
	cmdGetFirmware          = []byte{0x02, 0x13}
	dataTypeSetColor        = []byte{0x7e, 0x20, 0x01}
	cmdSleep                = []byte{0x01, 0x0e, 0x00}
	cmdBatteryLevel         = []byte{0x02, 0x0f}
	cmdOpenWriteEndpoint    = []byte{0x0d, 0x01, 0x02}
	cmdWrite                = []byte{0x06, 0x01}
	cmdWriteExtra           = []byte{0x07, 0x01}
	cmdBluetoothMode        = []byte{0x01, 0x3a, 0x00, 0x02}
	cmdUsbMode              = map[int][]byte{0: {0x02, 0x3a, 0x00, 0x00}, 1: {0x01, 0x3a, 0x00, 0x00}}
	bufferSize              = 64
	bufferSizeWrite         = bufferSize + 1
	headerSize              = 2
	headerWriteSize         = 4
	maxBufferSizePerRequest = 61
	keyboardKey             = "k100air-default"
	defaultLayout           = "k100air-default-US"
	keyAssignmentLength     = 135
	rgbModes                = []string{
		"watercolor",
		"visor",
		"rainbowwave",
		"colorwave",
		"colorshift",
		"colorpulse",
		"spiralrainbow",
		"tlr",
		"tlk",
		"rain",
		"keyboard",
		"off",
	}
)

func Init(vendorId, slipstreamId, productId uint16, dev *hid.Device, endpoint byte, serial string) *Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	// Init new struct with HID device
	d := &Device{
		dev:          dev,
		Template:     "k100airW.html",
		VendorId:     vendorId,
		ProductId:    productId,
		SlipstreamId: slipstreamId,
		Serial:       serial,
		Endpoint:     endpoint,
		Firmware:     "n/a",
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
		Product: "K100 AIR",
		Layouts: keyboards.GetLayouts(keyboardKey),
		RGBModes: map[string]string{
			"watercolor":    "Watercolor",
			"colorpulse":    "Color Pulse",
			"colorshift":    "Color Shift",
			"colorwave":     "Color Wave",
			"rain":          "Rain",
			"rainbowwave":   "Rainbow Wave",
			"spiralrainbow": "Spiral Rainbow",
			"tlk":           "Type Lighting - Key",
			"tlr":           "Type Lighting - Ripple",
			"keyboard":      "Keyboard",
			"off":           "Off",
			"visor":         "Visor",
		},
		SleepModes: map[int]string{
			1:  "1 minute",
			5:  "5 minutes",
			10: "10 minutes",
			15: "15 minutes",
			30: "30 minutes",
			60: "1 hour",
		},
		UIKeyboard:    "keyboard-7",
		UIKeyboardRow: "keyboard-row-25",
	}

	d.getDebugMode()       // Debug mode
	d.loadRgb()            // Load RGB
	d.loadDeviceProfiles() // Load all device profiles
	d.saveDeviceProfile()  // Save profile
	return d
}

// GetRgbProfiles will return RGB profiles for a target device
func (d *Device) GetRgbProfiles() interface{} {
	return d.Rgb
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	// Placeholder
}

// StopInternal will stop all device operations and switch a device back to hardware mode
func (d *Device) StopInternal() {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}
	d.setHardwareMode()
}

// StopDirty will stop all device operations in dirty way
func (d *Device) StopDirty() uint8 {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial}).Info("Stopping device (dirty)...")

	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}

	d.Connected = false
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 1
}

// SetConnected will change connected status
func (d *Device) SetConnected(value bool) {
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true
	}
	d.Connected = value
}

// Connect will connect to a device
func (d *Device) Connect() {
	if !d.Connected {
		d.Connected = true
		d.setSoftwareMode()    // Activate software mode
		d.getBatterLevel()     // Battery level
		d.getDeviceFirmware()  // Firmware
		d.setKeyAmount()       // Set number of keys
		d.initLeds()           // Init LED ports
		d.setDeviceColor()     // Device color
		d.setBrightnessLevel() // Brightness
		d.setSleepTimer()      // Sleep
		d.initKeys()           // Init default key action
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

// setKeyAmount will set global key amount
func (d *Device) setKeyAmount() {
	index := 0
	for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
		for _, key := range row.Keys {
			for range key.PacketIndex {
				index++
			}
		}
	}
	d.KeyAmount = index
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

// setHardwareMode will switch a device to hardware mode
func (d *Device) setHardwareMode() {
	if d.Connected {
		_, err := d.transfer(cmdHardwareMode, nil)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
		}
	}
}

// getBatterLevel will return initial battery level
func (d *Device) getBatterLevel() {
	batteryLevel, err := d.transfer(cmdBatteryLevel, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get battery level")
	}
	d.BatteryLevel = binary.LittleEndian.Uint16(batteryLevel[3:5]) / 10
	stats.UpdateBatteryStats(d.Serial, d.Product, d.BatteryLevel, 0)
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	_, err := d.transfer(cmdSoftwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
	d.Connected = true
}

// GetSleepMode will return current sleep mode
func (d *Device) GetSleepMode() int {
	if d.DeviceProfile != nil {
		return d.DeviceProfile.SleepMode
	}
	return 0
}

// setSoftwareMode will switch a device to bluetooth mode
func (d *Device) setBluetoothMode() {
	if d.Connected {
		_, err := d.transfer(cmdBluetoothMode, nil)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
		}
		d.Connected = false
	}
}

// setUsbMode will switch a device to usb mode
func (d *Device) setUsbMode() {
	if d.Connected {
		for i := 0; i < len(cmdUsbMode); i++ {
			_, err := d.transfer(cmdUsbMode[i], nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
			}
		}
	}
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	fw, err := d.transfer(cmdGetFirmware, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
	}

	v1, v2, v3 := int(fw[3]), int(fw[4]), int(binary.LittleEndian.Uint16(fw[5:7]))
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

// initLeds will initialize LED ports
func (d *Device) initLeds() {
	_, err := d.transfer(cmdOpenColorEndpoint, cmdSetLeds)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to open endpoint")
	}

	_, err = d.transfer(cmdRead, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
		return
	}

	buf := make([]byte, 8)
	buf[0] = 0x69
	buf[1] = 0x6c
	buf[2] = 0x01
	buf[4] = 0x08
	buf[6] = 0x65
	buf[7] = 0x6d

	dataTypeSetColor = []byte{}
	d.writeColor(buf)

	_, err = d.transfer(cmdCloseEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close endpoint")
	}

	_, err = d.transfer(cmdInitProtocol, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to init color protocol")
	}
}

// initKeys will init default key action for all keys
func (d *Device) initKeys() {
	var val byte = 0
	buf := make([]byte, keyAssignmentLength)
	for i := range buf {
		buf[i] = 0x01 // 0x01 complete slice
	}

	// Keys
	for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
		for keyIndex, key := range row.Keys {
			switch keyIndex {
			case 4:
				continue
			case 87:
				val = 0x3f
			case 98:
				val = 0x3f
			case 104:
				val = 0x3f
			case 106:
				val = 0x3f
			case 108:
				val = 0x3f
			case 109:
				val = 0x3f
			case 111:
				val = 0x3f
			default:
				val = 0x39
			}
			bufPosition := key.PacketIndex[0] / 3
			buf[bufPosition] = val
		}
	}

	// Wheel
	buf[103] = 0x39
	buf[104] = 0x39
	d.writeKeyAssignmentData(buf)
}

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"
	keyboardMap := make(map[string]*keyboards.Keyboard)

	deviceProfile := &DeviceProfile{
		Product: d.Product,
		Serial:  d.Serial,
		Path:    profilePath,
	}

	// First save, assign saved profile to a device
	if d.DeviceProfile == nil {
		// RGB, Label
		deviceProfile.RGBProfile = "keyboard"
		deviceProfile.SlipstreamRGBProfile = "keyboard"
		deviceProfile.Label = "Keyboard"
		deviceProfile.Active = true
		keyboardMap["default"] = keyboards.GetKeyboard(defaultLayout)
		deviceProfile.Keyboards = keyboardMap
		deviceProfile.Profile = "default"
		deviceProfile.Profiles = []string{"default"}
		deviceProfile.Layout = "US"
		deviceProfile.ControlDial = 1
		deviceProfile.BrightnessLevel = 1000
		deviceProfile.SleepMode = 15
	} else {
		if len(d.DeviceProfile.Layout) == 0 {
			deviceProfile.Layout = "US"
		} else {
			deviceProfile.Layout = d.DeviceProfile.Layout
		}

		if d.DeviceProfile.SleepMode == 0 {
			deviceProfile.SleepMode = 15
		} else {
			deviceProfile.SleepMode = d.DeviceProfile.SleepMode
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
		deviceProfile.SlipstreamRGBProfile = d.DeviceProfile.SlipstreamRGBProfile
		deviceProfile.Label = d.DeviceProfile.Label
		deviceProfile.Profile = d.DeviceProfile.Profile
		deviceProfile.Profiles = d.DeviceProfile.Profiles
		deviceProfile.Keyboards = d.DeviceProfile.Keyboards
		deviceProfile.ControlDial = d.DeviceProfile.ControlDial
		deviceProfile.BrightnessLevel = d.DeviceProfile.BrightnessLevel

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
		logger.Log(logger.Fields{"error": err, "location": deviceProfile.Path}).Error("Unable to close file handle")
	}

	d.loadDeviceProfiles() // Reload
}

// loadDeviceProfiles will load custom user profiles
func (d *Device) loadDeviceProfiles() {
	profileList := make(map[string]*DeviceProfile)
	userProfileDirectory := pwd + "/database/profiles/"

	files, err := os.ReadDir(userProfileDirectory)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": userProfileDirectory, "serial": d.Serial}).Error("Unable to read content of a folder")
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

// setSleepTimer will set device sleep timer
func (d *Device) setSleepTimer() uint8 {
	if d.Exit {
		return 0
	}

	if d.DeviceProfile != nil {
		buf := make([]byte, 4)
		sleep := d.DeviceProfile.SleepMode * (60 * 1000)
		binary.LittleEndian.PutUint32(buf, uint32(sleep))
		_, err := d.transfer(cmdSleep, buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change device sleep timer")
			return 0
		}
		return 1
	}
	return 0
}

// UpdateSleepTimer will update device sleep timer
func (d *Device) UpdateSleepTimer(minutes int) uint8 {
	if d.DeviceProfile != nil {
		d.DeviceProfile.SleepMode = minutes
		d.saveDeviceProfile()
		d.setSleepTimer()
		return 1
	}
	return 0
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
	pf.AlternateColors = profile.AlternateColors
	pf.RgbDirection = profile.RgbDirection

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
	if _, ok := d.RGBModes[profile]; !ok {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}

	d.DeviceProfile.SlipstreamRGBProfile = profile // Set profile
	d.saveDeviceProfile()                          // Save profile
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
	d.DeviceProfile.BrightnessLevel = 1000

	switch mode {
	case 1:
		d.DeviceProfile.BrightnessLevel = 300
	case 2:
		d.DeviceProfile.BrightnessLevel = 600
	case 3:
		d.DeviceProfile.BrightnessLevel = 1000
	case 4:
		d.DeviceProfile.BrightnessLevel = 0
	}

	d.saveDeviceProfile()
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf[0:2], d.DeviceProfile.BrightnessLevel)
	_, err := d.transfer(cmdBrightness, buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change brightness")
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
				d.setKeyAmount()

				// RGB reset
				if d.activeRgb != nil {
					d.activeRgb.Exit <- true // Exit current RGB mode
					d.activeRgb = nil
				}
				d.setDeviceColor()
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

// UpdateControlDial will update control dial function
func (d *Device) UpdateControlDial(value int) uint8 {
	d.DeviceProfile.ControlDial = value
	d.saveDeviceProfile()
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

// UpdateDeviceColor will update device color based on selected input
func (d *Device) UpdateDeviceColor(_, keyOption int, color rgb.Color) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	switch keyOption {
	case 2:
		{
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Color = color
				if d.activeRgb != nil {
					d.activeRgb.Exit <- true // Exit current RGB mode
					d.activeRgb = nil
				}
				d.setDeviceColor() // Restart RGB
				return 1
			}
		}
	}
	return 0
}

// setBrightnessLevel will set global brightness level
func (d *Device) setBrightnessLevel() {
	if d.DeviceProfile != nil {
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf[0:2], d.DeviceProfile.BrightnessLevel)
		_, err := d.transfer(cmdBrightness, buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change brightness")
		}
	}
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	if d.DeviceProfile == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to set color. DeviceProfile is null!")
		return
	}

	if !slices.Contains(rgbModes, d.DeviceProfile.SlipstreamRGBProfile) {
		d.DeviceProfile.SlipstreamRGBProfile = "keyboard"
	}

	switch d.DeviceProfile.SlipstreamRGBProfile {
	case "off":
		{
			if keyboard, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, keyboard.BufferSize)
				buf[3] = 0x01
				buf[4] = 0xff
				buf[5] = 0
				buf[6] = 0
				buf[7] = 0
				buf[8] = byte(d.KeyAmount)
				start := 9
				for _, row := range keyboard.Row {
					for _, key := range row.Keys {
						for packet := range key.PacketIndex {
							value := key.PacketIndex[packet] / 3
							buf[start] = byte(value)
							start++
						}
					}
				}
				dataTypeSetColor = []byte{0x7e, 0x20, 0x01}
				d.writeColor(buf)
				return
			}
		}
	case "keyboard":
		{
			if keyboard, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var colorIndex = make([]byte, 0)
				var buf = make([]byte, keyboard.BufferSize)
				buf[3] = 0x01
				buf[4] = 0xff
				buf[5] = byte(keyboard.Color.Blue)
				buf[6] = byte(keyboard.Color.Green)
				buf[7] = byte(keyboard.Color.Red)
				buf[8] = byte(d.KeyAmount)
				for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
					for _, key := range row.Keys {
						for packet := range key.PacketIndex {
							value := key.PacketIndex[packet] / 3
							colorIndex = append(colorIndex, byte(value))
						}
					}
				}
				// Sort in descending order
				sort.Slice(colorIndex, func(i, j int) bool {
					return colorIndex[i] > colorIndex[j] // Reverse order
				})
				copy(buf[9:], colorIndex)

				dataTypeSetColor = []byte{0x7e, 0x20, 0x01}
				d.writeColor(buf)
				return
			}
		}
	case "rain":
		{
			if keyboard, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, keyboard.BufferSize-4)
				dataTypeSetColor = []byte{0x7e, 0xa0, 0x02, 0x04, 0x01}
				start := 3

				rain := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				if rain == nil {
					buf[2] = byte(d.KeyAmount)
					start = 3
				} else {
					var speed = byte(0x04)
					switch rain.Speed {
					case 1:
						speed = 0x05 // Fast
						break
					case 2:
						speed = 0x04 // Medium
						break
					case 3:
						speed = 0x03 // Slow
						break
					}

					if rain.AlternateColors {
						buf = make([]byte, (keyboard.BufferSize-4)+8)
						dataTypeSetColor = []byte{0x7e, 0xa0, 0x01, speed, 0x01} // Custom colors
						buf[1] = 0x02                                            // 2 colors
						buf[2] = 0xff                                            // Marker
						buf[3] = byte(rain.StartColor.Blue)                      // Color 1 Blue
						buf[4] = byte(rain.StartColor.Green)                     // Color 1 Green
						buf[5] = byte(rain.StartColor.Red)                       // Color 1 Red
						buf[6] = 0xff                                            // Marker
						buf[7] = byte(rain.EndColor.Blue)                        // Color 2 Blue
						buf[8] = byte(rain.EndColor.Green)                       // Color 2 Green
						buf[9] = byte(rain.EndColor.Red)                         // Color 2 Red
						buf[10] = byte(d.KeyAmount)                              // Key amount
						start = 11                                               // Keyboard data start position
					} else {
						dataTypeSetColor = []byte{0x7e, 0xa0, 0x02, speed, 0x01} // Random colors
						start = 3
						buf[2] = byte(d.KeyAmount)
					}
				}

				for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
					for _, key := range row.Keys {
						for packet := range key.PacketIndex {
							value := key.PacketIndex[packet] / 3
							buf[start] = byte(value)
							start++
						}
					}
				}
				d.writeColor(buf)
				return
			}
		}
	case "tlk":
		{
			if keyboard, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, keyboard.BufferSize-4)
				dataTypeSetColor = []byte{0xf9, 0xb1, 0x02, 0x04} // Random colors
				start := 4

				tlk := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				if tlk == nil {
					logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to find tlk rgb profile definition. Using device defaults")
					start = 4
					buf[3] = byte(d.KeyAmount)
				} else {
					var speed = byte(0x04)
					switch tlk.Speed {
					case 1:
						speed = 0x05 // Fast
						break
					case 2:
						speed = 0x04 // Medium
						break
					case 3:
						speed = 0x03 // Slow
						break
					}
					if tlk.AlternateColors {
						buf = make([]byte, (keyboard.BufferSize-4)+8)
						dataTypeSetColor = []byte{0xf9, 0xb1, 0x01, speed} // Custom colors
						buf[2] = 0x02                                      // 2 colors
						buf[3] = 0xff                                      // Marker
						buf[4] = byte(tlk.StartColor.Blue)                 // Color 1 Blue
						buf[5] = byte(tlk.StartColor.Green)                // Color 1 Green
						buf[6] = byte(tlk.StartColor.Red)                  // Color 1 Red
						buf[7] = 0xff                                      // Marker
						buf[8] = byte(tlk.EndColor.Blue)                   // Color 2 Blue
						buf[9] = byte(tlk.EndColor.Green)                  // Color 2 Green
						buf[10] = byte(tlk.EndColor.Red)                   // Color 2 Red
						buf[11] = byte(d.KeyAmount)                        // Key amount
						start = 12                                         // Keyboard data start position
					} else {
						dataTypeSetColor = []byte{0xf9, 0xb1, 0x02, speed} // Random colors
						start = 4
						buf[3] = byte(d.KeyAmount)
					}
				}

				for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
					for _, key := range row.Keys {
						for packet := range key.PacketIndex {
							value := key.PacketIndex[packet] / 3
							buf[start] = byte(value)
							start++
						}
					}
				}

				d.writeColor(buf)
				return
			}
		}
	case "tlr":
		{
			if keyboard, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, keyboard.BufferSize-4)
				dataTypeSetColor = []byte{0xa2, 0x09, 0x02, 0x04}
				start := 4
				tlk := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				if tlk == nil {
					logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to find tlk rgb profile definition. Using device defaults")
					start = 4
					buf[3] = byte(d.KeyAmount)
				} else {
					var speed = byte(0x04)
					switch tlk.Speed {
					case 1:
						speed = 0x05 // Fast
						break
					case 2:
						speed = 0x04 // Medium
						break
					case 3:
						speed = 0x03 // Slow
						break
					}

					if tlk.AlternateColors {
						buf = make([]byte, (keyboard.BufferSize-4)+8)
						dataTypeSetColor = []byte{0xa2, 0x09, 0x01, speed} // Custom colors
						buf[2] = 0x02                                      // 2 colors
						buf[3] = 0xff                                      // Marker
						buf[4] = byte(tlk.StartColor.Blue)                 // Color 1 Blue
						buf[5] = byte(tlk.StartColor.Green)                // Color 1 Green
						buf[6] = byte(tlk.StartColor.Red)                  // Color 1 Red
						buf[7] = 0xff                                      // Marker
						buf[8] = byte(tlk.EndColor.Blue)                   // Color 2 Blue
						buf[9] = byte(tlk.EndColor.Green)                  // Color 2 Green
						buf[10] = byte(tlk.EndColor.Red)                   // Color 2 Red
						buf[11] = byte(d.KeyAmount)                        // Key amount
						start = 12                                         // Keyboard data start position
					} else {
						dataTypeSetColor = []byte{0xa2, 0x09, 0x02, speed} // Random colors
						start = 4
						buf[3] = byte(d.KeyAmount)
					}

				}
				for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
					for _, key := range row.Keys {
						for packet := range key.PacketIndex {
							value := key.PacketIndex[packet] / 3
							buf[start] = byte(value)
							start++
						}
					}
				}
				d.writeColor(buf)
				return
			}
		}
	case "spiralrainbow":
		{
			if keyboard, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, keyboard.BufferSize-4)
				spiralrainbow := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				var speed = byte(0x04)
				if spiralrainbow != nil {
					switch spiralrainbow.Speed {
					case 1:
						speed = 0x05 // Fast
						break
					case 2:
						speed = 0x04 // Medium
						break
					case 3:
						speed = 0x03 // Slow
						break
					}
				}
				dataTypeSetColor = []byte{0x87, 0xab, 0x00, speed, 0x06}
				buf[2] = byte(d.KeyAmount)
				start := 3
				for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
					for _, key := range row.Keys {
						for packet := range key.PacketIndex {
							value := key.PacketIndex[packet] / 3
							buf[start] = byte(value)
							start++
						}
					}
				}
				d.writeColor(buf)
				return
			}
		}
	case "colorpulse":
		{
			if keyboard, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, keyboard.BufferSize-4)
				dataTypeSetColor = []byte{0x4f, 0xad, 0x02, 0x04}
				start := 4
				colorpulse := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				if colorpulse == nil {
					logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to find tlk rgb profile definition. Using device defaults")
					start = 4
					buf[3] = byte(d.KeyAmount)
				} else {
					var speed = byte(0x04)
					if colorpulse.Speed == 1 {
						speed = 0x05 // Fast
					}
					if colorpulse.Speed > 1 {
						speed = 0x04 // Medium
					}
					if colorpulse.Speed > 5 {
						speed = 0x03 // Slow
					}

					if colorpulse.AlternateColors {
						buf = make([]byte, (keyboard.BufferSize-4)+8)
						dataTypeSetColor = []byte{0x4f, 0xad, 0x01, speed} // Custom colors
						buf[2] = 0x02                                      // 2 colors
						buf[3] = 0xff                                      // Marker
						buf[4] = byte(colorpulse.StartColor.Blue)          // Color 1 Blue
						buf[5] = byte(colorpulse.StartColor.Green)         // Color 1 Green
						buf[6] = byte(colorpulse.StartColor.Red)           // Color 1 Red
						buf[7] = 0xff                                      // Marker
						buf[8] = byte(colorpulse.EndColor.Blue)            // Color 2 Blue
						buf[9] = byte(colorpulse.EndColor.Green)           // Color 2 Green
						buf[10] = byte(colorpulse.EndColor.Red)            // Color 2 Red
						buf[11] = byte(d.KeyAmount)                        // Key amount
						start = 12                                         // Keyboard data start position
					} else {
						dataTypeSetColor = []byte{0x4f, 0xad, 0x02, speed} // Random colors
						start = 4
						buf[3] = byte(d.KeyAmount)
					}

				}
				for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
					for _, key := range row.Keys {
						for packet := range key.PacketIndex {
							value := key.PacketIndex[packet] / 3
							buf[start] = byte(value)
							start++
						}
					}
				}
				d.writeColor(buf)
				return
			}
		}
	case "colorshift":
		{
			if keyboard, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, keyboard.BufferSize-4)
				dataTypeSetColor = []byte{0xfa, 0xa5, 0x02, 0x04}
				start := 4
				colorpulse := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				if colorpulse == nil {
					logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to find tlk rgb profile definition. Using device defaults")
					start = 4
					buf[3] = byte(d.KeyAmount)
				} else {
					var speed = byte(0x04)
					if colorpulse.Speed == 1 {
						speed = 0x05 // Fast
					}
					if colorpulse.Speed > 1 {
						speed = 0x04 // Medium
					}
					if colorpulse.Speed > 5 {
						speed = 0x03 // Slow
					}

					if colorpulse.AlternateColors {
						buf = make([]byte, (keyboard.BufferSize-4)+8)
						dataTypeSetColor = []byte{0xfa, 0xa5, 0x01, speed} // Custom colors
						buf[2] = 0x02                                      // 2 colors
						buf[3] = 0xff                                      // Marker
						buf[4] = byte(colorpulse.StartColor.Blue)          // Color 1 Blue
						buf[5] = byte(colorpulse.StartColor.Green)         // Color 1 Green
						buf[6] = byte(colorpulse.StartColor.Red)           // Color 1 Red
						buf[7] = 0xff                                      // Marker
						buf[8] = byte(colorpulse.EndColor.Blue)            // Color 2 Blue
						buf[9] = byte(colorpulse.EndColor.Green)           // Color 2 Green
						buf[10] = byte(colorpulse.EndColor.Red)            // Color 2 Red
						buf[11] = byte(d.KeyAmount)                        // Key amount
						start = 12                                         // Keyboard data start position
					} else {
						dataTypeSetColor = []byte{0xfa, 0xa5, 0x02, speed} // Random colors
						start = 4
						buf[3] = byte(d.KeyAmount)
					}

				}
				for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
					for _, key := range row.Keys {
						for packet := range key.PacketIndex {
							value := key.PacketIndex[packet] / 3
							buf[start] = byte(value)
							start++
						}
					}
				}
				d.writeColor(buf)
				return
			}
		}
	case "colorwave":
		{
			if keyboard, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, keyboard.BufferSize-4)
				dataTypeSetColor = []byte{0xff, 0x7b, 0x02, 0x04, 0x04}
				start := 3

				colorwave := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				if colorwave == nil {
					buf[2] = byte(d.KeyAmount)
					start = 3
				} else {
					var speed = byte(0x04)
					switch colorwave.Speed {
					case 1:
						speed = 0x05 // Fast
						break
					case 2:
						speed = 0x04 // Medium
						break
					case 3:
						speed = 0x03 // Slow
						break
					}

					direction := byte(0x04)
					if colorwave.RgbDirection > 0 && colorwave.RgbDirection < 6 {
						direction = colorwave.RgbDirection
					}

					if colorwave.AlternateColors {
						buf = make([]byte, (keyboard.BufferSize-4)+8)
						dataTypeSetColor = []byte{0xff, 0x7b, 0x01, speed, direction} // Custom colors
						buf[1] = 0x02                                                 // 2 colors
						buf[2] = 0xff                                                 // Marker
						buf[3] = byte(colorwave.StartColor.Blue)                      // Color 1 Blue
						buf[4] = byte(colorwave.StartColor.Green)                     // Color 1 Green
						buf[5] = byte(colorwave.StartColor.Red)                       // Color 1 Red
						buf[6] = 0xff                                                 // Marker
						buf[7] = byte(colorwave.EndColor.Blue)                        // Color 2 Blue
						buf[8] = byte(colorwave.EndColor.Green)                       // Color 2 Green
						buf[9] = byte(colorwave.EndColor.Red)                         // Color 2 Red
						buf[10] = byte(d.KeyAmount)                                   // Key amount
						start = 11                                                    // Keyboard data start position
					} else {
						dataTypeSetColor = []byte{0xff, 0x7b, 0x02, speed, direction} // Random colors
						start = 3
						buf[2] = byte(d.KeyAmount)
					}
				}

				for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
					for _, key := range row.Keys {
						for packet := range key.PacketIndex {
							value := key.PacketIndex[packet] / 3
							buf[start] = byte(value)
							start++
						}
					}
				}
				d.writeColor(buf)
				return
			}
		}
	case "rainbowwave":
		{
			if keyboard, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, keyboard.BufferSize-4)
				spiralrainbow := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				var speed = byte(0x04)
				if spiralrainbow != nil {
					switch spiralrainbow.Speed {
					case 1:
						speed = 0x05 // Fast
						break
					case 2:
						speed = 0x04 // Fast
						break
					case 3:
						speed = 0x03 // Fast
						break
					}
				}

				dataTypeSetColor = []byte{0x4c, 0xb9, 0x00, speed, 0x04}
				buf[2] = byte(d.KeyAmount)
				start := 3
				for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
					for _, key := range row.Keys {
						for packet := range key.PacketIndex {
							value := key.PacketIndex[packet] / 3
							buf[start] = byte(value)
							start++
						}
					}
				}
				d.writeColor(buf)
				return
			}
		}
	case "visor":
		{
			if keyboard, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, keyboard.BufferSize-4)
				dataTypeSetColor = []byte{0xc0, 0x90, 0x02, 0x04, 0x04}
				start := 3

				visor := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				if visor == nil {
					buf[2] = byte(d.KeyAmount)
					start = 3
				} else {
					var speed = byte(0x04)
					switch visor.Speed {
					case 1:
						speed = 0x05 // Fast
						break
					case 2:
						speed = 0x04 // Medium
						break
					case 3:
						speed = 0x03 // Slow
						break
					}

					if visor.AlternateColors {
						buf = make([]byte, (keyboard.BufferSize-4)+8)
						dataTypeSetColor = []byte{0xc0, 0x90, 0x01, speed, 0x04} // Custom colors
						buf[1] = 0x02                                            // 2 colors
						buf[2] = 0xff                                            // Marker
						buf[3] = byte(visor.StartColor.Blue)                     // Color 1 Blue
						buf[4] = byte(visor.StartColor.Green)                    // Color 1 Green
						buf[5] = byte(visor.StartColor.Red)                      // Color 1 Red
						buf[6] = 0xff                                            // Marker
						buf[7] = byte(visor.EndColor.Blue)                       // Color 2 Blue
						buf[8] = byte(visor.EndColor.Green)                      // Color 2 Green
						buf[9] = byte(visor.EndColor.Red)                        // Color 2 Red
						buf[10] = byte(d.KeyAmount)                              // Key amount
						start = 11                                               // Keyboard data start position
					} else {
						dataTypeSetColor = []byte{0xc0, 0x90, 0x02, speed, 0x04} // Random colors
						start = 3
						buf[2] = byte(d.KeyAmount)
					}
				}

				for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
					for _, key := range row.Keys {
						for packet := range key.PacketIndex {
							value := key.PacketIndex[packet] / 3
							buf[start] = byte(value)
							start++
						}
					}
				}
				d.writeColor(buf)
				return
			}
		}
	case "watercolor":
		{
			if keyboard, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, keyboard.BufferSize)
				buf[2] = 0x01
				buf[3] = 0xff
				buf[4] = 0xff
				buf[5] = 0xff
				buf[6] = 0xff
				buf[7] = byte(d.KeyAmount)
				start := 8
				for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
					for _, key := range row.Keys {
						for packet := range key.PacketIndex {
							value := key.PacketIndex[packet] / 3
							buf[start] = byte(value)
							start++
						}
					}
				}
				dataTypeSetColor = []byte{0x22, 0x00, 0x03, 0x04}
				d.writeColor(buf)
				return
			}
		}
	}
}

// writeColor will write data to the device with a specific endpoint.
func (d *Device) writeColor(data []byte) {
	if d.Exit {
		return
	}
	buffer := make([]byte, len(dataTypeSetColor)+len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	copy(buffer[headerWriteSize:headerWriteSize+len(dataTypeSetColor)], dataTypeSetColor)
	copy(buffer[headerWriteSize+len(dataTypeSetColor):], data)

	_, err := d.transfer(cmdOpenColorEndpoint, cmdActivateLed)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
		return
	}

	_, err = d.transfer(cmdRead, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
		return
	}

	// Split packet into chunks
	chunks := common.ProcessMultiChunkPacket(buffer, maxBufferSizePerRequest)
	for i, chunk := range chunks {
		if i == 0 {
			// Initial packet is using cmdWriteColor
			_, err := d.transfer(cmdWrite, chunk)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
			}
		} else {
			// Chunks don't use cmdWriteColor, they use static dataTypeSubColor
			_, err := d.transfer(cmdWriteExtra, chunk)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to endpoint")
			}
		}
	}

	_, err = d.transfer(cmdCloseEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
		return
	}

	// Close endpoint
	_, err = d.transfer(cmdFlush, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to close endpoint")
	}
}

// writeKeyAssignmentData will write key assignment to the device.
func (d *Device) writeKeyAssignmentData(data []byte) {
	if d.Exit {
		return
	}
	buffer := make([]byte, len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	copy(buffer[headerWriteSize:], data)

	_, err := d.transfer(cmdOpenWriteEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "endpoint": cmdOpenWriteEndpoint, "serial": d.Serial}).Error("Unable to open endpoint")
		return
	}

	// Split packet into chunks
	chunks := common.ProcessMultiChunkPacket(buffer, maxBufferSizePerRequest)
	for i, chunk := range chunks {
		if i == 0 {
			_, err := d.transfer(cmdWrite, chunk)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to the device")
			}
		} else {
			_, err := d.transfer(cmdWriteExtra, chunk)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to the device")
			}
		}
	}

	_, err = d.transfer(cmdCloseEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "endpoint": cmdCloseEndpoint, "serial": d.Serial}).Error("Unable to close endpoint")
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint, buffer []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = d.Endpoint
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

// modifyBrightness will modify brightness via control button
func (d *Device) modifyBrightness() {
	if d.DeviceProfile != nil {
		if d.DeviceProfile.BrightnessLevel >= 1000 {
			d.DeviceProfile.BrightnessLevel = 0
		} else {
			d.DeviceProfile.BrightnessLevel += 200
		}
		d.saveDeviceProfile()
		d.setBrightnessLevel()
	}
}

// TriggerKeyAssignment will trigger key assignment
func (d *Device) TriggerKeyAssignment(data []byte) {
	if data[16] == 0x02 {
		d.modifyBrightness()
	} else if data[17] == 0x04 && data[18] == 0x08 {
		d.setUsbMode()
	} else if data[17] == 0x04 && data[18] == 0x10 {
		cmdBluetoothMode[3] = 0x02 // Profile 1
		d.setBluetoothMode()
	} else if data[17] == 0x04 && data[18] == 0x20 {
		cmdBluetoothMode[3] = 0x03 // Profile 2
		d.setBluetoothMode()
	} else if data[17] == 0x04 && data[18] == 0x40 {
		cmdBluetoothMode[3] = 0x08 // Profile 3
		d.setBluetoothMode()
	}
}

// ModifyBatteryLevel will modify battery level
func (d *Device) ModifyBatteryLevel(batteryLevel uint16) {
	d.BatteryLevel = batteryLevel
	stats.UpdateBatteryStats(d.Serial, d.Product, d.BatteryLevel, 0)
}
