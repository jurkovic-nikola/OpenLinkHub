package k65plusW

// Package: K65 PLUS WIRELESS
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
	"path/filepath"
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
	Label                string
	Layout               string
	Keyboards            map[string]*keyboards.Keyboard
	Profile              string
	Profiles             []string
	ControlDial          int
	BrightnessLevel      uint16
	SleepMode            int
	DisableAltTab        bool
	DisableAltF4         bool
	DisableShiftTab      bool
	DisableWinKey        bool
	Performance          bool
	RGBCluster           bool
}

type Device struct {
	Debug                  bool
	dev                    *hid.Device
	listener               *hid.Device
	Manufacturer           string `json:"manufacturer"`
	Product                string `json:"product"`
	Serial                 string `json:"serial"`
	Firmware               string `json:"firmware"`
	DongleFirmware         string `json:"dongleFirmware"`
	activeRgb              *rgb.ActiveRGB
	UserProfiles           map[string]*DeviceProfile `json:"userProfiles"`
	Devices                map[int]string            `json:"devices"`
	DeviceProfile          *DeviceProfile
	OriginalProfile        *DeviceProfile
	Template               string
	VendorId               uint16
	Brightness             map[int]string
	LEDChannels            int
	CpuTemp                float32
	GpuTemp                float32
	Layouts                []string
	ProductId              uint16
	SlipstreamId           uint16
	ControlDialOptions     map[int]string
	RGBModes               map[string]string
	SleepModes             map[int]string
	KeyAmount              int
	Rgb                    *rgb.RGB
	rgbMutex               sync.RWMutex
	Endpoint               byte
	Exit                   bool
	timer                  *time.Ticker
	autoRefreshChan        chan struct{}
	mutex                  sync.Mutex
	deviceLock             sync.Mutex
	Connected              bool
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
	instance               *common.Device
	mouseLoopActive        bool
	mouseLoopMutex         sync.Mutex
	mouseLoopStopCh        chan struct{}
	Usb                    bool
}

var (
	pwd                     = ""
	cmdSoftwareMode         = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode         = []byte{0x01, 0x03, 0x00, 0x01}
	cmdActivateLed          = []byte{0x0d, 0x01, 0x60, 0x6d}
	cmdBrightness           = []byte{0x01, 0x02, 0x00}
	cmdGetFirmware          = []byte{0x02, 0x13}
	dataTypeSetColor        = []byte{0x7e, 0x20, 0x01}
	dataTypeSubColor        = []byte{0x07, 0x01}
	cmdWriteColor           = []byte{0x06, 0x01}
	cmdSleep                = []byte{0x01, 0x0e, 0x00}
	cmdBatteryLevel         = []byte{0x02, 0x0f}
	cmdPerformance          = []byte{0x01, 0x4a, 0x00}
	cmdWritePerformance     = []byte{0x01}
	cmdOpenEndpoint         = []byte{0x0d, 0x01, 0x02}
	cmdCloseEndpoint        = []byte{0x05, 0x01, 0x01}
	cmdKeyAssignment        = []byte{0x06, 0x01}
	cmdKeyAssignmentNext    = []byte{0x07, 0x01}
	deviceRefreshInterval   = 1000
	transferTimeout         = 500
	bufferSize              = 64
	bufferSizeWrite         = bufferSize + 1
	headerSize              = 2
	headerWriteSize         = 4
	maxBufferSizePerRequest = 61
	keyboardKey             = "k65plus-default"
	defaultLayout           = "k65plus-default-US"
	KeyAssignment           = 123
	maxKeyAssignmentLen     = 61
	rgbProfileUpgrade       = []string{"gradient", "pastelrainbow", "pastelspiralrainbow"}
	rgbModes                = []string{
		"watercolor",
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
		Template:     "k65plusW.html",
		VendorId:     vendorId,
		ProductId:    productId,
		SlipstreamId: slipstreamId,
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
		Serial:      serial,
		Endpoint:    endpoint,
		Product:     "K65 PLUS",
		LEDChannels: 123,
		Layouts:     keyboards.GetLayouts(keyboardKey),
		ControlDialOptions: map[int]string{
			1: "Volume Control",
			2: "Brightness",
			3: "Scroll",
			4: "Zoom",
			5: "Screen Brightness",
			6: "Media Control",
			7: "Horizontal Scroll",
		},
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
		},
		SleepModes: map[int]string{
			5:  "5 minutes",
			10: "10 minutes",
			15: "15 minutes",
			30: "30 minutes",
			60: "1 hour",
		},
		autoRefreshChan: make(chan struct{}),
		UIKeyboard:      "keyboard-6",
		UIKeyboardRow:   "keyboard-row-17",
		KeyAssignmentTypes: map[int]string{
			0:  "None",
			1:  "Media Keys",
			3:  "Keyboard",
			8:  "Sniper",
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

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeK65PlusW,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-keyboard.svg",
		Instance:    d,
	}
}

// GetRgbProfiles will return RGB profiles for a target device
func (d *Device) GetRgbProfiles() interface{} {
	tmp := *d.Rgb

	// Filter unsupported modes out
	profiles := make(map[string]rgb.Profile, len(tmp.Profiles))
	for key, value := range tmp.Profiles {
		if slices.Contains(rgbModes, key) {
			profiles[key] = value
		}
	}
	tmp.Profiles = profiles
	return tmp
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {}

// StopInternal will stop all device operations and switch a device back to hardware mode
func (d *Device) StopInternal() {
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
		})
	}()

	d.initLeds()
	if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
		var buf = make([]byte, 93)
		buf[2] = 0x01
		buf[3] = 0xff
		buf[4] = 0xff
		buf[5] = 0xff
		buf[6] = 0xff
		dataTypeSetColor = []byte{0x22, 0x00, 0x03, 0x04}
		d.writeColor(buf)
	}

	_, err := d.transfer(cmdCloseEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change color")
	}

	d.setHardwareMode()
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
			if d.autoRefreshChan != nil {
				close(d.autoRefreshChan)
			}
		})
	}()

	d.Connected = false
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 1
}

// setupKeyAssignment will setup keyboard keys
func (d *Device) setupKeyAssignment() {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

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

	buf := make([]byte, KeyAssignment)
	for i := range buf {
		buf[i] = 0x01
	}

	for _, row := range keyboard.Row {
		for _, key := range row.Keys {
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
		d.setAutoRefresh()     // Auto refresh
		d.setSoftwareMode()    // Activate software mode
		d.getBatterLevel()     // Battery level
		d.getDeviceFirmware()  // Firmware
		d.setKeyAmount()       // Set number of keys
		d.initLeds()           // Init LED ports
		d.setDeviceColor()     // Device color
		d.setBrightnessLevel() // Brightness
		d.setSleepTimer()      // Sleep
		d.setupPerformance()   // Performance
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

		if err := common.SaveJsonData(rgbFilename, profile); err != nil {
			logger.Log(logger.Fields{"error": err, "location": rgbFilename}).Error("Unable to write rgb profile data")
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
		if err := common.SaveJsonData(path, d.Rgb); err != nil {
			logger.Log(logger.Fields{"error": err, "location": path}).Error("Unable to upgrade rgb profile data")
			return
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
			if key.NoColor {
				continue
			}

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
	d.deviceLock.Lock()

	if d.DeviceProfile == nil {
		d.deviceLock.Unlock()
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

	// Key control
	control := make([]byte, 3)
	if d.DeviceProfile.Performance {
		control = []byte{0x45, 0x00, 0x01}
	} else {
		control = []byte{0x45, 0x00, 0x00}
	}

	_, err = d.transfer(cmdWritePerformance, control)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to setup keyboard performance")
	}

	d.deviceLock.Unlock()
	d.setupKeyAssignment()
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
	_, err := d.transfer(cmdHardwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
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
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	fw, err := d.transfer(
		cmdGetFirmware,
		nil,
	)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
		return
	}

	v1, v2, v3 := int(fw[3]), int(fw[4]), int(binary.LittleEndian.Uint16(fw[5:7]))
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

// initLeds will initialize LED ports
func (d *Device) initLeds() {
	_, err := d.transfer(cmdActivateLed, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
		return
	}
	time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
	_, err = d.transfer([]byte{0x09, 0x01}, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
		return
	}
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

	if d.DeviceProfile == nil {
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
		deviceProfile.DisableAltTab = d.DeviceProfile.DisableAltTab
		deviceProfile.DisableAltF4 = d.DeviceProfile.DisableAltF4
		deviceProfile.DisableShiftTab = d.DeviceProfile.DisableShiftTab
		deviceProfile.DisableWinKey = d.DeviceProfile.DisableWinKey
		deviceProfile.Performance = d.DeviceProfile.Performance
		deviceProfile.RGBCluster = d.DeviceProfile.RGBCluster

		if len(d.DeviceProfile.Path) < 1 {
			deviceProfile.Path = profilePath
			d.DeviceProfile.Path = profilePath
		} else {
			deviceProfile.Path = d.DeviceProfile.Path
		}
		deviceProfile.LCDMode = d.DeviceProfile.LCDMode
		deviceProfile.LCDRotation = d.DeviceProfile.LCDRotation
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

	d.loadDeviceProfiles()
}

// loadDeviceProfiles will load custom user profiles
func (d *Device) loadDeviceProfiles() {
	profileList := make(map[string]*DeviceProfile)
	userProfileDirectory := pwd + "/database/profiles/"

	files, err := os.ReadDir(userProfileDirectory)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": userProfileDirectory, "serial": d.Serial}).Error("Unable to read content of a folder")
		return
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

// setSleepTimer will set device sleep timer
func (d *Device) setSleepTimer() uint8 {
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
		if err := common.SaveJsonData(rgbFilename, d.Rgb); err != nil {
			logger.Log(logger.Fields{"error": err, "location": rgbFilename}).Error("Unable to write rgb profile data")
			return
		}
	}
}

// ProcessNewGradientColor will create new gradient color
func (d *Device) ProcessNewGradientColor(profileName string) (uint8, uint) {
	if d.GetRgbProfile(profileName) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profileName}).Warn("Non-existing RGB profile")
		return 0, 0
	}

	pf := d.GetRgbProfile(profileName)
	if pf == nil {
		return 0, 0
	}

	if pf.Gradients == nil {
		return 0, 0
	}

	// find next available key
	nextID := 0
	for k := range pf.Gradients {
		if k >= nextID {
			nextID = k + 1
		}
	}
	pf.Gradients[nextID] = rgb.Color{Red: 0, Green: 255, Blue: 255}

	d.Rgb.Profiles[profileName] = *pf
	d.saveRgbProfile()
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}
	d.setDeviceColor()
	return 1, uint(nextID)
}

// ProcessDeleteGradientColor will delete gradient color
func (d *Device) ProcessDeleteGradientColor(profileName string) (uint8, uint) {
	if d.GetRgbProfile(profileName) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profileName}).Warn("Non-existing RGB profile")
		return 0, 0
	}

	pf := d.GetRgbProfile(profileName)
	if pf == nil {
		return 0, 0
	}

	if len(pf.Gradients) < 3 {
		return 2, 0
	}

	maxKey := -1
	for k := range pf.Gradients {
		if k > maxKey {
			maxKey = k
		}
	}
	delete(pf.Gradients, maxKey)

	d.Rgb.Profiles[profileName] = *pf
	d.saveRgbProfile()
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}
	d.setDeviceColor()
	return 1, uint(maxKey)
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
	pf.Gradients = profile.Gradients

	d.Rgb.Profiles[profileName] = *pf
	d.saveRgbProfile()
	if d.Connected {
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true
			d.activeRgb = nil
		}
		d.setDeviceColor()
	}
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
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}
	d.setDeviceColor()
	return 1

}

// ChangeDeviceBrightness will change device brightness
func (d *Device) ChangeDeviceBrightness(mode uint8) uint8 {
	d.DeviceProfile.Brightness = mode
	d.DeviceProfile.BrightnessLevel = 1000

	if mode == 4 {
		d.DeviceProfile.BrightnessLevel = 0
	}

	d.saveDeviceProfile()
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}
	d.setDeviceColor()
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
	if !d.Connected {
		return 0
	}

	if profile, ok := d.UserProfiles[profileName]; ok {
		currentProfile := d.DeviceProfile
		currentProfile.Active = false
		d.DeviceProfile = currentProfile
		d.saveDeviceProfile()

		// RGB reset
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true
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

// DeleteDeviceProfile deletes a device profile and its JSON file
func (d *Device) DeleteDeviceProfile(profileName string) uint8 {
	if !d.Connected {
		return 0
	}

	profile, ok := d.UserProfiles[profileName]
	if !ok {
		return 0
	}

	if !common.IsValidExtension(profile.Path, ".json") {
		return 0
	}

	if profile.Active {
		return 2
	}

	if err := os.Remove(profile.Path); err != nil {
		return 3
	}

	delete(d.UserProfiles, profileName)

	return 1
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
		d.activeRgb.Exit <- true
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
		d.activeRgb.Exit <- true
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
				key.DeviceId = keyAssignment.DeviceId
				key.ActionHold = keyAssignment.ActionHold
				key.ModifierKey = keyAssignment.ModifierKey
				key.RetainOriginal = keyAssignment.RetainOriginal

				if key.Default {
					key.ColorOffOnFunctionKey = key.ColorOffOnFunctionKeyInternal
				} else {
					key.ColorOffOnFunctionKey = !d.isFunctionKey(int(keyAssignment.ModifierKey))
				}

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
func (d *Device) UpdateDeviceColor(_ int, keyOption int, color rgb.Color, _ []int) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	switch keyOption {
	case 2:
		{
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Color = color
				if d.activeRgb != nil {
					d.activeRgb.Exit <- true
					d.activeRgb = nil
				}
				d.saveDeviceProfile()
				d.setDeviceColor()
				return 1
			}
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

	if !slices.Contains(rgbModes, d.DeviceProfile.SlipstreamRGBProfile) {
		d.DeviceProfile.SlipstreamRGBProfile = "keyboard"
	}

	// Init
	d.initLeds()
	switch d.DeviceProfile.SlipstreamRGBProfile {
	case "off":
		{
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, 93)
				buf[3] = 0x01
				buf[4] = 0xff
				buf[5] = 0x00
				buf[6] = 0x00
				buf[7] = 0x00
				dataTypeSetColor = []byte{0x7e, 0x20, 0x01}
				d.writeColor(buf)
				return
			}
		}
	case "keyboard":
		{
			if keyboard, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				var buf = make([]byte, 93)
				buf[3] = 0x01
				buf[4] = 0xff
				buf[5] = byte(keyboard.Color.Blue)
				buf[6] = byte(keyboard.Color.Green)
				buf[7] = byte(keyboard.Color.Red)
				dataTypeSetColor = []byte{0x7e, 0x20, 0x01}
				d.writeColor(buf)
				return
			}
		}
	case "rain":
		{
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				rain := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				var speed = byte(0x04)
				if rain != nil {
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
				}
				var buf = make([]byte, 89)
				dataTypeSetColor = []byte{0x7e, 0xa0, 0x02, speed, 0x01}
				d.writeColor(buf)
				return
			}
		}
	case "tlk":
		{
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				tlk := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				var speed = byte(0x04)
				if tlk != nil {
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
				}

				var buf = make([]byte, 89)
				dataTypeSetColor = []byte{0xf9, 0xb1, 0x02, speed}
				d.writeColor(buf)
				return
			}
		}
	case "tlr":
		{
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				tlr := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				var speed = byte(0x04)
				if tlr != nil {
					switch tlr.Speed {
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

				var buf = make([]byte, 89)
				dataTypeSetColor = []byte{0xa2, 0x09, 0x02, speed}
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

				var buf = make([]byte, 89)
				dataTypeSetColor = []byte{0x87, 0xab, 0x00, speed, 0x06}
				d.writeColor(buf)
				return
			}
		}
	case "colorpulse":
		{
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				colorpulse := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				var speed = byte(0x04)
				if colorpulse != nil {
					if colorpulse.Speed == 1 {
						speed = 0x05 // Fast
					}
					if colorpulse.Speed > 1 {
						speed = 0x04 // Medium
					}
					if colorpulse.Speed > 5 {
						speed = 0x03 // Slow
					}
				}

				var buf = make([]byte, 89)
				dataTypeSetColor = []byte{0x4f, 0xad, 0x02, speed}
				d.writeColor(buf)
				return
			}
		}
	case "colorshift":
		{
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				colorshift := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				var speed = byte(0x04)
				if colorshift != nil {
					if colorshift.Speed == 1 {
						speed = 0x05 // Fast
					}
					if colorshift.Speed > 1 {
						speed = 0x04 // Medium
					}
					if colorshift.Speed > 5 {
						speed = 0x03 // Slow
					}
				}

				var buf = make([]byte, 89)
				dataTypeSetColor = []byte{0xfa, 0xa5, 0x02, speed}
				d.writeColor(buf)
				return
			}
		}
	case "colorwave":
		{
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				colorwave := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				var speed = byte(0x04)
				if colorwave != nil {
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
				}

				var buf = make([]byte, 89)
				dataTypeSetColor = []byte{0xff, 0x7b, 0x02, speed, 0x04}
				d.writeColor(buf)
				return
			}
		}
	case "rainbowwave":
		{
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				rainbowwave := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				var speed = byte(0x04)
				if rainbowwave != nil {
					switch rainbowwave.Speed {
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

				var buf = make([]byte, 89)
				dataTypeSetColor = []byte{0x4c, 0xb9, 0x00, speed, 0x04}
				d.writeColor(buf)
				return
			}
		}
	case "watercolor":
		{
			if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
				watercolor := d.GetRgbProfile(d.DeviceProfile.SlipstreamRGBProfile)
				var speed = byte(0x04)
				if watercolor != nil {
					if watercolor.Speed == 1 {
						speed = 0x05 // Fast
					}
					if watercolor.Speed > 1 {
						speed = 0x04 // Medium
					}
					if watercolor.Speed > 5 {
						speed = 0x03 // Slow
					}
				}

				var buf = make([]byte, 93)
				buf[2] = 0x01
				buf[3] = 0xff
				buf[4] = 0xff
				buf[5] = 0xff
				buf[6] = 0xff
				dataTypeSetColor = []byte{0x22, 0x00, 0x03, speed}
				d.writeColor(buf)
				return
			}
		}
	}
	_, err := d.transfer(cmdCloseEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change color")
	}
}

// setBrightnessLevel will set global brightness level
func (d *Device) setBrightnessLevel() {
	if d.Exit {
		return
	}

	if d.DeviceProfile != nil {
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf[0:2], d.DeviceProfile.BrightnessLevel)
		_, err := d.transfer(cmdBrightness, buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change brightness")
		}
	}
}

// writeColor will write color data to the device
func (d *Device) writeColor(data []byte) {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	buffer := make([]byte, len(dataTypeSetColor)+len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	copy(buffer[headerWriteSize:headerWriteSize+len(dataTypeSetColor)], dataTypeSetColor)
	copy(buffer[headerWriteSize+len(dataTypeSetColor):], data)

	chunks := common.ProcessMultiChunkPacket(buffer, maxBufferSizePerRequest)
	for i, chunk := range chunks {
		if i == 0 {
			_, err := d.transfer(cmdWriteColor, chunk)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
			}
		} else {
			_, err := d.transfer(dataTypeSubColor, chunk)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to endpoint")
			}
		}
	}
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

// triggerKeyAssignment will trigger key assignment if defined
func (d *Device) triggerKeyAssignment(value []byte, functionKey bool, modifierKey uint8) {
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

		// Function Key
		if functionKey {
			switch key.ActionType {
			case 11: // Brightness +
				if d.DeviceProfile.BrightnessLevel+100 > 1000 {
					d.DeviceProfile.BrightnessLevel = 1000
				} else {
					d.DeviceProfile.BrightnessLevel += 100
				}
			case 12: // Brightness -
				if d.DeviceProfile.BrightnessLevel < 100 {
					d.DeviceProfile.BrightnessLevel = 0
				} else {
					d.DeviceProfile.BrightnessLevel -= 100
				}
			}

			// Brightness
			if key.ActionType == 11 || key.ActionType == 12 {
				d.saveDeviceProfile()
				d.setBrightnessLevel()
				return
			}

			// Performance Lock
			if key.IsLock {
				d.DeviceProfile.Performance = !d.DeviceProfile.Performance
				d.saveDeviceProfile()
				d.setupPerformance()
				return
			}

			// Media Keys, ignore any hold action
			if key.MediaKey {
				inputmanager.InputControlKeyboard(key.FnActionCommand, false)
				return
			}
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

// writeKeyAssignment will write Key Assignment data
func (d *Device) writeKeyAssignment(data []byte) {
	buffer := make([]byte, len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	copy(buffer[headerWriteSize:], data)

	_, err := d.transfer(cmdOpenEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to open write endpoint")
		return
	}

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

	_, err = d.transfer(cmdCloseEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to close write endpoint")
		return
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint, buffer []byte) ([]byte, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = d.Endpoint
	endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
	}

	bufferR := make([]byte, bufferSize)

	if _, err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return bufferR, err
	}

	if _, err := d.dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		return bufferR, err
	}
	return bufferR, nil
}

// ModifyBatteryLevel will modify battery level
func (d *Device) ModifyBatteryLevel(batteryLevel uint16) {
	d.BatteryLevel = batteryLevel
	stats.UpdateBatteryStats(d.Serial, d.Product, d.BatteryLevel, 0)
}

// TriggerKeyAssignment will trigger key assignment
func (d *Device) TriggerKeyAssignment(data []byte) {
	// FN color change
	functionKey := data[17] == 0x04
	if functionKey != d.FunctionKey {
		d.FunctionKey = functionKey
	}

	var modifierKey uint8 = 0
	modifierIndex := data[d.getModifierPosition()]
	if modifierIndex > 0 {
		modifierKey = d.getModifierKey(modifierIndex)
	}

	if data[1] == 0x02 {
		d.triggerKeyAssignment(data, functionKey, modifierKey)
	} else if data[1] == 0x05 {
		// Knob
		value := data[4]
		switch d.DeviceProfile.ControlDial {
		case 1:
			{
				switch value {
				case 1:
					inputmanager.InputControlKeyboard(inputmanager.VolumeUp, false)
					break
				case 255:
					inputmanager.InputControlKeyboard(inputmanager.VolumeDown, false)
					break
				}
			}
		case 2:
			{
				switch value {
				case 1:
					if d.DeviceProfile.BrightnessLevel+100 > 1000 {
						d.DeviceProfile.BrightnessLevel = 1000
					} else {
						d.DeviceProfile.BrightnessLevel += 100
					}
				case 255:
					if d.DeviceProfile.BrightnessLevel < 100 {
						d.DeviceProfile.BrightnessLevel = 0
					} else {
						d.DeviceProfile.BrightnessLevel -= 100
					}
				}
				d.saveDeviceProfile()
				d.setBrightnessLevel()
			}
		case 3:
			{
				switch value {
				case 1:
					inputmanager.InputControlScroll(false)
				case 255:
					inputmanager.InputControlScroll(true)
				}
			}
		case 4:
			{
				switch value {
				case 1:
					inputmanager.InputControlZoom(true)
				case 255:
					inputmanager.InputControlZoom(false)
				}
			}
		case 5:
			{
				switch value {
				case 1:
					inputmanager.InputControlKeyboard(inputmanager.KeyScreenBrightnessUp, false)
				case 255:
					inputmanager.InputControlKeyboard(inputmanager.KeyScreenBrightnessDown, false)
				}
			}
		case 6:
			{
				switch value {
				case 1:
					inputmanager.InputControlKeyboard(inputmanager.MediaNext, false)
				case 255:
					inputmanager.InputControlKeyboard(inputmanager.MediaPrev, false)
				}
			}
		case 7:
			{
				switch value {
				case 1:
					inputmanager.InputControlScrollHorizontal(true)
				case 255:
					inputmanager.InputControlScrollHorizontal(false)
				}
			}
		}
	}
}
