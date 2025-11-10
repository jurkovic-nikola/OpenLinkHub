package k70pmW

// Package: K70 RGB PRO MINI
// This is the primary package for K70 RGB PRO MINI
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
	"OpenLinkHub/src/stats"
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
	Active               bool
	Path                 string
	Product              string
	Serial               string
	LCDMode              uint8
	LCDRotation          uint8
	Brightness           uint8
	RGBProfile           string
	SlipstreamRGBProfile string
	BrightnessSlider     *uint8
	SleepMode            int
	OriginalBrightness   uint8
	Label                string
	Layout               string
	Keyboards            map[string]*keyboards.Keyboard
	Profile              string
	PollingRate          int
	Profiles             []string
	DisableAltTab        bool
	DisableAltF4         bool
	DisableShiftTab      bool
	DisableWinKey        bool
	Performance          bool
}

type Device struct {
	Debug                  bool
	dev                    *hid.Device
	listener               *hid.Device
	Manufacturer           string `json:"manufacturer"`
	Product                string `json:"product"`
	Serial                 string `json:"serial"`
	Firmware               string `json:"firmware"`
	activeRgb              *rgb.ActiveRGB
	UserProfiles           map[string]*DeviceProfile `json:"userProfiles"`
	Devices                map[int]string            `json:"devices"`
	DeviceProfile          *DeviceProfile
	OriginalProfile        *DeviceProfile
	Template               string
	VendorId               uint16
	ProductId              uint16
	SlipstreamId           uint16
	RGBModes               map[string]string
	SleepModes             map[int]string
	KeyAmount              int
	Brightness             map[int]string
	PollingRates           map[int]string
	LEDChannels            int
	CpuTemp                float32
	GpuTemp                float32
	Layouts                []string
	Rgb                    *rgb.RGB
	rgbMutex               sync.RWMutex
	Endpoint               byte
	mutex                  sync.Mutex
	Exit                   bool
	UIKeyboard             string
	UIKeyboardRow          string
	BatteryLevel           uint16
	FunctionKey            bool
	KeyAssignmentModifiers map[int]string
	KeyAssignmentTypes     map[int]string
	ModifierIndex          *big.Int
	KeyboardKey            *keyboards.Key
	PressLoop              bool
	MacroTracker           map[int]macro.Tracker
	Connected              bool
	deviceLock             sync.Mutex
	RGBDirection           map[byte]string
	mouseLoopActive        bool
	mouseLoopMutex         sync.Mutex
	mouseLoopStopCh        chan struct{}
}

var (
	pwd                     = ""
	cmdSoftwareMode         = []byte{0x01, 0x03, 0x00, 0x06}
	cmdHardwareMode         = []byte{0x01, 0x03, 0x00, 0x01}
	cmdOpenColorEndpoint    = []byte{0x0d, 0x01}
	cmdSetLeds              = []byte{0x13}
	cmdRead                 = []byte{0x09, 0x01}
	cmdActivateLed          = []byte{0x65, 0x6d}
	cmdGetFirmware          = []byte{0x02, 0x13}
	dataTypeSetColor        = []byte{0x12, 0x00}
	cmdWriteColor           = []byte{0x06, 0x01}
	cmdFlush                = []byte{0x15, 0x01}
	cmdBatteryLevel         = []byte{0x02, 0x0f}
	cmdSetPollingRate       = []byte{0x01, 0x01, 0x00}
	cmdPerformance          = []byte{0x01, 0x4a, 0x00}
	cmdWritePerformance     = []byte{0x01}
	cmdOpenEndpoint         = []byte{0x0d, 0x02, 0x02}
	cmdKeyAssignment        = []byte{0x06, 0x02}
	cmdWriteNext            = []byte{0x07, 0x01}
	cmdCloseEndpoint        = []byte{0x05, 0x01, 0x02}
	cmdCloseColorEndpoint   = []byte{0x05, 0x01, 0x01}
	cmdInitProtocol         = []byte{0x0b, 0x65, 0x6d}
	cmdConnectionMode       = []byte{0x01, 0x3a}
	cmdSleep                = []byte{0x01, 0x0e, 0x00}
	cmdBrightness           = []byte{0x01, 0x02, 0x00}
	bufferSize              = 64
	bufferSizeWrite         = bufferSize + 1
	headerSize              = 2
	headerWriteSize         = 4
	maxBufferSizePerRequest = 61
	keyboardKey             = "k70pm-default"
	defaultLayout           = "k70pm-default-US"
	maxKeyAssignmentLen     = 61
	rgbProfileUpgrade       = []string{"tlk", "tlr", "spiralrainbow", "rainbowwave", "rain", "visor", "colorwave"}
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
		Template:     "k70pmW.html",
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
		RGBDirection: map[byte]string{
			1: "Top to Bottom",
			2: "Bottom to Top",
			4: "Left to Right",
			5: "Right to Left",
		},
		Product: "K70 PRO MINI",
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
		UIKeyboardRow: "keyboard-row-18",
		KeyAssignmentTypes: map[int]string{
			0:  "None",
			1:  "Media Keys",
			3:  "Keyboard",
			9:  "Mouse",
			10: "Macro",
		},
		MacroTracker: make(map[int]macro.Tracker),
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
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}

	if d.Connected {
		d.setHardwareMode()
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
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
	found := false
	enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
		found = true
		return nil
	})

	// Enumerate all Corsair devices
	err := hid.Enumerate(d.VendorId, d.ProductId, enum)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Warn("Unable to enumerate devices")
	}

	if found {
		logger.Log(logger.Fields{"vendorId": d.VendorId, "productId": d.ProductId, "serial": d.Serial}).Warn("Connect() not allowed while in USB protocol")
	}

	if !d.Connected && !found {
		d.Connected = true
		d.setSoftwareMode()    // Activate software mode
		d.getBatterLevel()     // Battery level
		d.getDeviceFirmware()  // Firmware
		d.setKeyAmount()       // Set number of keys
		d.initLeds()           // Init LED ports
		d.setDeviceColor()     // Device color
		d.setBrightnessLevel() // Brightness
		d.setSleepTimer()      // Sleep
		d.setupPerformance()   // Init default key action
	}
}

// setupKeyAssignment will setup keyboard keys
func (d *Device) setupKeyAssignment() {
	if d.DeviceProfile == nil {
		return
	}
	if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; !ok {
		return
	}

	keyboard, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]
	if !ok {
		return
	}

	buf := make([]byte, keyboard.KeyAssignmentLength)
	for i := range buf {
		buf[i] = 0x01
	}

	for _, row := range keyboard.Row {
		for _, key := range row.Keys {
			if key.KeyData == nil || len(key.KeyData) == 0 {
				continue
			}
			if key.Default {
				buf[key.KeyData[0]] = byte(key.KeyData[1])
			} else {
				buf[key.KeyData[0]] = byte(key.KeyData[2])
			}

			if key.RetainOriginal {
				if !key.Default {
					buf[key.KeyData[0]] = byte(key.KeyData[1])
				}
			} else {
				if key.ModifierKey > 0 {
					shift := d.getModifierKeyShift(int(key.ModifierKey))
					if shift > 0 {
						buf[key.KeyData[0]] = shift + 2
						if key.BrightnessKey || key.RgbKey || key.ProfileKey || key.MacroRecordingKey || key.MediaKey || key.IsLock {
							buf[key.KeyData[0]] = shift
						}
					}
				}
			}
		}
	}
	d.writeKeyAssignment(buf)
}

func (d *Device) getModifierKeyShift(modifierKey int) byte {
	for _, value := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
		for keyIndex, key := range value.Keys {
			if keyIndex == modifierKey {
				return key.ModifierShift
			}
		}
	}
	return 0
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
			d.Rgb.Profiles[profile] = *rgb.GetRgbProfile(profile)
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
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	} else {
		d.Connected = true
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
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

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

	buffer := make([]byte, len(dataTypeSetColor)+len(buf)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(buf)))
	copy(buffer[headerWriteSize:headerWriteSize+len(dataTypeSetColor)], dataTypeSetColor)
	copy(buffer[headerWriteSize+len(dataTypeSetColor):], buf)

	_, err = d.transfer(cmdWriteColor, buffer)
	if err != nil {
		return
	}

	_, err = d.transfer(cmdCloseColorEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close endpoint")
	}

	_, err = d.transfer(cmdInitProtocol, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to init color protocol")
	}
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
		deviceProfile.SlipstreamRGBProfile = "keyboard"
		deviceProfile.Label = "Keyboard"
		deviceProfile.Active = true
		keyboardMap["default"] = keyboards.GetKeyboard(defaultLayout)
		deviceProfile.Keyboards = keyboardMap
		deviceProfile.Profile = "default"
		deviceProfile.Profiles = []string{"default"}
		deviceProfile.Layout = "US"
		deviceProfile.PollingRate = 4
		deviceProfile.SleepMode = 15
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
		deviceProfile.OriginalBrightness = d.DeviceProfile.OriginalBrightness
		deviceProfile.Brightness = d.DeviceProfile.Brightness
		deviceProfile.RGBProfile = d.DeviceProfile.RGBProfile
		deviceProfile.SlipstreamRGBProfile = d.DeviceProfile.SlipstreamRGBProfile
		deviceProfile.Label = d.DeviceProfile.Label
		deviceProfile.Profile = d.DeviceProfile.Profile
		deviceProfile.Profiles = d.DeviceProfile.Profiles
		deviceProfile.Keyboards = d.DeviceProfile.Keyboards
		deviceProfile.PollingRate = d.DeviceProfile.PollingRate
		deviceProfile.DisableAltTab = d.DeviceProfile.DisableAltTab
		deviceProfile.DisableAltF4 = d.DeviceProfile.DisableAltF4
		deviceProfile.DisableShiftTab = d.DeviceProfile.DisableShiftTab
		deviceProfile.DisableWinKey = d.DeviceProfile.DisableWinKey
		deviceProfile.Performance = d.DeviceProfile.Performance
		deviceProfile.SleepMode = d.DeviceProfile.SleepMode

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
				if d.DeviceProfile.Keyboards["default"] == nil {
					currentLayout := fmt.Sprintf("%s-%s", keyboardKey, d.DeviceProfile.Layout)
					layout := keyboards.GetKeyboard(currentLayout)
					d.DeviceProfile.Keyboards["default"] = layout
				}
			}
		}
	}
}

// setCpuTemperature will store current CPU temperature
func (d *Device) setTemperatures() {
	d.CpuTemp = temperatures.GetCpuTemperature()
	d.GpuTemp = temperatures.GetGpuTemperature()
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
	if !d.Connected {
		return 0
	}

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
	d.setBrightnessLevel()
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
	d.setBrightnessLevel()
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
		d.setupPerformance()
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
				d.setupPerformance()
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

// ProcessGetKeyAssignmentModifiers will get key assignment modifiers
func (d *Device) ProcessGetKeyAssignmentModifiers() interface{} {
	if d.DeviceProfile == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to set color. DeviceProfile is null!")
		return nil
	}

	modifiers := make(map[int]string)
	modifiers[0] = "None"
	if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
		for _, rows := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
			for keyId, key := range rows.Keys {
				if key.Modifier {
					modifiers[keyId] = key.KeyNameInternal
					if len(key.KeyNameInternal) == 0 {
						modifiers[keyId] = key.KeyName
					}
				}
			}
		}
	}
	return modifiers
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

// isFunctionKey will check if given modifier key is Function Key
func (d *Device) isFunctionKey(keyIndex int) bool {
	if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
		for _, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
			for keyId, key := range row.Keys {
				if keyIndex == keyId {
					if key.FunctionKey {
						return true
					}
				}
			}
		}
	}
	return false
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
				key.ModifierKey = keyAssignment.ModifierKey
				key.RetainOriginal = keyAssignment.RetainOriginal
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
		val := uint16(*d.DeviceProfile.BrightnessSlider) * 10
		binary.LittleEndian.PutUint16(buf[0:2], val)
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
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, 99)
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
						buf = make([]byte, 107)
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
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, 99)
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
						buf = make([]byte, 107)
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
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, 99)
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
						buf = make([]byte, 107)
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
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
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

				var buf = make([]byte, 99)
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
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, 99)
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
						buf = make([]byte, 107)
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
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, 99)
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
						buf = make([]byte, 107)
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
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, 99)
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
						buf = make([]byte, 107)
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
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
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
				var buf = make([]byte, 99)
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
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, 99)
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
						buf = make([]byte, 107)
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
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

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
			_, err := d.transfer(cmdWriteColor, chunk)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
			}
		} else {
			// Chunks don't use cmdWriteColor, they use static cmdWriteNext
			_, err := d.transfer(cmdWriteNext, chunk)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to endpoint")
			}
		}
	}

	_, err = d.transfer(cmdCloseColorEndpoint, nil)
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

// writeKeyAssignment will write Key Assignment data
func (d *Device) writeKeyAssignment(data []byte) {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

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
			_, err := d.transfer(cmdWriteNext, chunk)
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

// getModifierPosition will return key modifier packet position in backendListener
func (d *Device) getModifierPosition() uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	if val, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
		return val.ModifierPosition
	}
	return 0
}

// getModifierKey will return modifier key value
func (d *Device) getModifierKey(modifierIndex uint8) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	if val, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
		for _, rows := range val.Row {
			for keyId, key := range rows.Keys {
				if key.ModifierPacketValue == modifierIndex {
					return uint8(keyId)
				}
			}
		}
	}
	return 0
}

// addToMacroTracker adds or updates an entry in MacroTracker
func (d *Device) addToMacroTracker(key int, value uint16, actionType uint8) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.MacroTracker == nil {
		d.MacroTracker = make(map[int]macro.Tracker)
	}
	d.MacroTracker[key] = macro.Tracker{
		Value: value,
		Type:  actionType,
	}
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
		switch d.MacroTracker[key].Type {
		case 1, 3:
			inputmanager.InputControlKeyboardHold(d.MacroTracker[key].Value, false)
			break
		case 9:
			inputmanager.InputControlMouseHold(d.MacroTracker[key].Value, false)
			break
		}
		d.deleteFromMacroTracker(key)
	}
}

// mouseEventLoop will send mouse action until stopped
func mouseEventLoop(stopCh <-chan struct{}, actionCommand, actionSleep uint16) {
	// Send input once
	inputmanager.InputControlMouse(actionCommand)

	// Timer
	ticker := time.NewTicker(time.Duration(actionSleep) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			inputmanager.InputControlMouse(actionCommand)
		case <-stopCh:
			// Loop stopped
			return
		}
	}
}

// changeConnectionMode will change device connection mode, from Slipstream to Bluetooth profiles
func (d *Device) changeConnectionMode(mode byte) {
	buf := make([]byte, 2)
	buf[0] = 0x00
	buf[1] = mode
	_, err := d.transfer(cmdConnectionMode, buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to change connection mode")
		return
	}
}

// TriggerKeyAssignment will trigger key assignment if defined
func (d *Device) TriggerKeyAssignment(value []byte) {
	functionKey := value[17] == 0x04
	if functionKey != d.FunctionKey {
		d.FunctionKey = functionKey
	}

	var modifierKey uint8 = 0
	modifierIndex := value[d.getModifierPosition()]
	if modifierIndex > 0 {
		modifierKey = d.getModifierKey(modifierIndex)
	}

	raw := make([]byte, len(value))
	if value[1] == 0x02 {
		raw = value[2:22]
	}

	// Cleanup FN
	if d.FunctionKey {
		raw[15] = 0x00
	}

	// Cleanup modifiers
	if modifierKey > 0 {
		raw[13] = 0x00
	}

	// Hash it
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
		key := d.getKeyData(val.String())
		if key == nil {
			return
		}

		// Lock
		if key.IsLock && functionKey {
			d.DeviceProfile.Performance = !d.DeviceProfile.Performance
			d.saveDeviceProfile()
			d.setupPerformance()
			return
		}

		// Bluetooth profile 1
		if key.BluetoothProfile1 && functionKey {
			d.changeConnectionMode(2)
			return
		}

		// Bluetooth profile 2
		if key.BluetoothProfile2 && functionKey {
			d.changeConnectionMode(3)
			return
		}

		// Bluetooth profile 3
		if key.BluetoothProfile3 && functionKey {
			d.changeConnectionMode(8)
			return
		}

		// USB protocol
		if key.SlipstreamProfile && functionKey {
			d.changeConnectionMode(0)
			return
		}

		// Brightness
		if key.BrightnessKey && functionKey {
			if *d.DeviceProfile.BrightnessSlider >= 100 {
				*d.DeviceProfile.BrightnessSlider = 0
			} else {
				*d.DeviceProfile.BrightnessSlider += 20
			}
			d.saveDeviceProfile()
			d.setBrightnessLevel()
			return
		}

		// Sub-action keys with FN combination
		if key.HasSubAction && functionKey && key.FnActionCommand > 0 && key.Default {
			d.addToMacroTracker(0, key.FnActionCommand, key.ActionType)
			inputmanager.InputControlKeyboard(key.FnActionCommand, true)
			return
		}

		// Default action
		if key.Default {
			return
		}

		// If modifier is function key and functionKey == true, set modifierKey
		if d.isFunctionKey(int(key.ModifierKey)) && functionKey {
			modifierKey = key.ModifierKey
		}

		// Return if modifier keys are incompatible
		if key.ModifierKey > 0 && key.ModifierKey != modifierKey {
			return
		}

		// If a modifier is used, but the key doesn't expect one
		if modifierKey > 0 && key.ModifierKey == 0 {
			if key.RetainOriginal || !key.Default {
				return
			}
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
			if key.ActionHold {
				d.mouseLoopMutex.Lock()
				defer d.mouseLoopMutex.Unlock()
				if !d.mouseLoopActive {
					d.mouseLoopActive = true
					d.mouseLoopStopCh = make(chan struct{})
					go mouseEventLoop(d.mouseLoopStopCh, key.ActionCommand, key.ToggleDelay)
				} else {
					// Stop sending events
					d.mouseLoopActive = false
					close(d.mouseLoopStopCh)
				}
			} else {
				inputmanager.InputControlMouse(key.ActionCommand)
			}
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
						d.addToMacroTracker(i, v.ActionCommand, v.ActionType)
					}

					switch v.ActionType {
					case 1, 3:
						inputmanager.InputControlKeyboard(v.ActionCommand, v.ActionHold)
						break
					case 9:
						if v.ActionRepeat > 0 && !v.ActionHold {
							for z := 0; z < int(v.ActionRepeat); z++ {
								inputmanager.InputControlMouse(v.ActionCommand)
								if v.ActionRepeatDelay > 0 && v.ActionRepeat > 1 {
									time.Sleep(time.Duration(v.ActionRepeatDelay) * time.Millisecond)
								}
							}
						} else if v.ActionHold && v.ActionRepeat == 0 {
							inputmanager.InputControlMouseHold(v.ActionCommand, v.ActionHold)
						} else {
							inputmanager.InputControlMouse(v.ActionCommand)
						}
						break
					case 5:
						if v.ActionDelay > 0 {
							time.Sleep(time.Duration(v.ActionDelay) * time.Millisecond)
						}
						break
					case 6:
						if v.ActionRepeat > 0 {
							for z := 0; z < int(v.ActionRepeat); z++ {
								inputmanager.InputControlKeyboardText(v.ActionText)
								if v.ActionRepeatDelay > 0 && v.ActionRepeat > 1 {
									time.Sleep(time.Duration(v.ActionRepeatDelay) * time.Millisecond)
								}
							}
						} else {
							inputmanager.InputControlKeyboardText(v.ActionText)
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

	if !d.Exit {
		// Get data from a device
		if _, err := d.dev.Read(bufferR); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
			return bufferR, err
		}
	}
	return bufferR, nil
}

// ModifyBatteryLevel will modify battery level
func (d *Device) ModifyBatteryLevel(batteryLevel uint16) {
	d.BatteryLevel = batteryLevel
	stats.UpdateBatteryStats(d.Serial, d.Product, d.BatteryLevel, 0)
}
