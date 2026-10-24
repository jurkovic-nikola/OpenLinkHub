package k95

// Package: K95 RGB
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dispatcher"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/keyboards"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/macro"
	"OpenLinkHub/src/openrgb"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/temperatures"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sstallion/go-hid"
)

type KeyPos struct {
	Row int
	Col int
}

// DeviceProfile struct contains all device profile
type DeviceProfile struct {
	OpenRGBIntegration bool
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
	RgbOff             bool
}

type Device struct {
	Path               string
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
	autoRefreshChan    chan struct{}
	mutex              sync.Mutex
	deviceLock         sync.Mutex
	UIKeyboard         string
	UIKeyboardRow      string
	KeyboardKey        *keyboards.Key
	PressLoop          bool
	ModifierIndex      *big.Int
	KeyAssignmentTypes map[int]string
	MacroTracker       map[int]macro.Tracker
	RGBModes           []string
	instance           *common.Device
	mouseLoopActive    bool
	mouseLoopMutex     sync.Mutex
	mouseLoopStopCh    chan struct{}
	stopRepeat         chan struct{}
	stopRepeatMutex    sync.Mutex
	dispatch           dispatcher.DeviceDispatcher
	queue              chan []byte
	queueStop          chan struct{}
	queueDone          chan struct{}
	queueStopOnce      sync.Once
}

var (
	pwd                   = ""
	cmdSoftwareMode       = []byte{0x04, 0x02}
	cmdHardwareMode       = []byte{0x04, 0x01}
	cmdActivateLed        = []byte{0x05, 0x09}
	cmdKeyAssignment      = []byte{0x40, 0x1e, 0x00}
	cmdWrite              = byte(0x07)
	cmdWriteColor         = byte(0x7f)
	dataTypeColor         = []byte{0x28, 0x01, 0x03, 0x02}
	cmdPerformance        = []byte{0x48}
	cmdKeyControl         = []byte{0x40, 0x01, 0x00, 0x60, 0xc0}
	cmdSetPollingRate     = []byte{0x0a, 0x00, 0x00}
	deviceRefreshInterval = 1000
	transferTimeout       = 500
	bufferSize            = 64
	bufferSizeWrite       = bufferSize + 1
	headerSize            = 2
	colorPackets          = 3
	keyboardKey           = "k95-default"
	defaultLayout         = "k95-default-US"
	maximumPacketSize     = 60
	rgbProfileUpgrade     = []string{"gradient", "pastelrainbow", "pastelspiralrainbow"}
	rgbModes              = []string{
		"circle",
		"circleshift",
		"colorpulse",
		"colorshift",
		"colorwarp",
		"cpu-temperature",
		"flickering",
		"gpu-temperature",
		"gradient",
		"keyboard",
		"off",
		"rainbow",
		"pastelrainbow",
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
		Path:      path,
		Template:  "k95.html",
		VendorId:  vendorId,
		ProductId: productId,
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
		Product:         "K95",
		LEDChannels:     168,
		Layouts:         keyboards.GetLayouts(keyboardKey),
		autoRefreshChan: make(chan struct{}),
		UIKeyboard:      "keyboard-7",
		UIKeyboardRow:   "keyboard-row-27",
		RGBModes:        rgbModes,
		PollingRates: map[int]string{
			0: "Not Set",
			8: "125 Hz / 8 msec",
			4: "250 Hz / 4 msec",
			2: "500 Hz / 2 msec",
			1: "1000 Hz / 1 msec",
		},
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

	d.getDebugMode()           // Debug mode
	d.getManufacturer()        // Manufacturer
	d.getSerial()              // Serial
	d.loadRgb()                // Load RGB
	d.setSoftwareMode()        // Activate software mode
	d.getDeviceFirmware()      // Firmware
	d.loadDeviceProfiles()     // Load all device profiles
	d.saveDeviceProfile()      // Save profile
	d.setAutoRefresh()         // Set auto device refresh
	d.setDeviceColor()         // Device color
	d.setupPerformance()       // Performance
	d.backendListener()        // Control buttons
	d.startQueueWorker()       // Queue
	d.setupOpenRGBController() // OpenRGB Controller
	d.createDevice()           // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeK95,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-keyboard.svg",
		Instance:    d,
	}
}

// SetDispatcher will set device dispatcher
func (d *Device) SetDispatcher(ds dispatcher.DeviceDispatcher) {
	d.dispatch = ds
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
func (d *Device) Stop() {
	d.Exit = true
	d.stopQueueWorker()
	if d.queueDone != nil {
		<-d.queueDone
	}
	openrgb.RemoveDeviceControllerBySerial(d.Serial)
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
	d.stopQueueWorker()
	if d.queueDone != nil {
		<-d.queueDone
	}
	openrgb.RemoveDeviceControllerBySerial(d.Serial)
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

	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 2
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

	buf := make([]byte, 0)
	keyMap := make(map[uint16]byte)
	for _, value := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
		for _, key := range value.Keys {
			var val byte = 0x00
			if key.Default {
				val = 0xc0
			} else {
				val = 0x40
			}
			if key.OnlyColor {
				continue
			}
			keyMap[key.KeyData[1]] = val
		}
	}

	keys := make([]uint16, 0, len(keyMap))
	for k := range keyMap {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	for _, k := range keys {
		buf = append(buf, byte(k), keyMap[k])
	}

	chunks := common.ProcessMultiChunkPacket(buf, 60)
	for i, chunk := range chunks {
		cmdKeyAssignment[1] = byte(len(chunk) / 2)
		err := d.transfer(cmdWrite, cmdKeyAssignment, chunk)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "index": i, "packet": fmt.Sprintf("%2x", chunks)}).Error("Unable to send key assignment packet")
			break
		}
	}
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
	for key, val := range d.Rgb.Profiles {
		template := rgb.GetRgbProfile(key)
		if template == nil {
			continue
		}

		if val.Version != template.Version {
			d.Rgb.Profiles[key] = *template
			save = true
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
		base = byte(160)
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
	err := d.transfer(cmdWrite, cmdPerformance, buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to setup keyboard performance")
	}
	d.deviceLock.Unlock()
	d.setupKeyAssignment()

	if d.isRgbStatic() && !d.DeviceProfile.OpenRGBIntegration {
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true
			d.activeRgb = nil
		}
		d.setDeviceColor()
	}
	d.setKeyControl()
}

// isRgbStatic will return if RGB effect is static
func (d *Device) isRgbStatic() bool {
	if d.DeviceProfile.RGBProfile == "keyboard" || d.DeviceProfile.RGBProfile == "static" {
		return true
	}
	return false
}

func (d *Device) setKeyControl() {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	err := d.transfer(cmdWrite, cmdKeyControl, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to reset device color")
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
	err := d.transfer(cmdWrite, cmdHardwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	err := d.transfer(cmdWrite, cmdSoftwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	info, err := d.dev.GetDeviceInfo()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device info")
		return
	}
	if info == nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device info")
		return
	}

	fw, err := common.GetBcdDevice(info.Path)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device firmware")
	}
	d.Firmware = fw
}

// initLeds will initialize LED ports
func (d *Device) initLeds() {
	err := d.transfer(cmdWrite, cmdActivateLed, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
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

	if d.DeviceProfile == nil {
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
			deviceProfile.OpenRGBIntegration = d.DeviceProfile.OpenRGBIntegration
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

	d.loadDeviceProfiles()
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
		if !common.AlphanumericDashRegex.MatchString(fileName) {
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

// UpdatePollingRate will set device polling rate
func (d *Device) UpdatePollingRate(pullingRate int) uint8 {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if _, ok := d.PollingRates[pullingRate]; ok {
		if d.DeviceProfile == nil {
			return 0
		}
		d.Exit = true
		d.stopQueueWorker()
		openrgb.RemoveDeviceControllerBySerial(d.Serial)
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true
			d.activeRgb = nil
		}
		time.Sleep(40 * time.Millisecond)

		d.DeviceProfile.PollingRate = pullingRate
		d.saveDeviceProfile()
		buf := make([]byte, 1)
		buf[0] = byte(pullingRate)
		err := d.transfer(cmdWrite, cmdSetPollingRate, buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to set polling rate")
			return 0
		}
		return 1
	}
	return 0
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

	if profile.StartColor.Temperature < 0 || profile.StartColor.Temperature > 105 {
		return 0
	}

	if profile.MiddleColor.Temperature < 0 || profile.MiddleColor.Temperature > 105 {
		return 0
	}

	if profile.EndColor.Temperature < 0 || profile.EndColor.Temperature > 105 {
		return 0
	}

	profile.StartColor.Brightness = pf.StartColor.Brightness
	profile.EndColor.Brightness = pf.EndColor.Brightness
	profile.MiddleColor.Brightness = pf.MiddleColor.Brightness
	pf.StartColor = profile.StartColor
	pf.EndColor = profile.EndColor
	pf.MiddleColor = profile.MiddleColor
	pf.Speed = profile.Speed
	pf.Gradients = profile.Gradients

	d.Rgb.Profiles[profileName] = *pf
	d.saveRgbProfile()
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}
	d.setDeviceColor()
	return 1
}

// UpdateRgbProfile will update device RGB profile
func (d *Device) UpdateRgbProfile(_ int, profile string) uint8 {
	if d.GetRgbProfile(profile) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}
	if d.DeviceProfile.OpenRGBIntegration {
		return 4
	}
	d.DeviceProfile.RGBProfile = profile // Set profile
	d.saveDeviceProfile()                // Save profile
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
	d.saveDeviceProfile()
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}
	d.setDeviceColor()
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
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}
	d.setDeviceColor()
	return 1
}

// ProcessSetOpenRgbIntegration will update OpenRGB integration status
func (d *Device) ProcessSetOpenRgbIntegration(enabled bool) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if d.activeRgb != nil {
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}

	d.clearQueue()
	d.DeviceProfile.OpenRGBIntegration = enabled
	d.saveDeviceProfile()
	d.setDeviceColor()
	return 1
}

// newOpenRGBController will create new OpenRGB controller object
func (d *Device) newOpenRGBController() *common.OpenRGBController {
	if d.DeviceProfile == nil {
		return nil
	}
	leds, matrix, offsets, colors := d.buildOpenRGBKeyboard(d.getCurrentKeyboard())

	if len(leds) == 0 {
		return nil
	}

	controller := &common.OpenRGBController{
		Name:        d.Product,
		Vendor:      "Corsair",
		Description: "OpenLinkHub Backend Device",
		FwVersion:   d.Firmware,
		Serial:      d.Serial,
		Location:    fmt.Sprintf("HID: %s", d.Path),
		DeviceType:  common.DeviceTypeKeyboard,
		ColorMode:   common.ColorModePerLed,
		LEDs:        leds, Colors: colors,
		Zones: []common.OpenRGBZone{
			{
				Name:     "Keyboard",
				NumLEDs:  uint32(len(leds)),
				ZoneType: common.ZoneTypeMatrix, Matrix: matrix,
			},
		},
		WriteColorEx: func(data []byte, _ int) {
			d.writeColorEx(data, offsets)
		},
	}

	return controller
}

// setupOpenRGBController will setup new OpenRGB controller
func (d *Device) setupOpenRGBController() {
	if controller := d.newOpenRGBController(); controller != nil {
		openrgb.AddDeviceController(controller)
		openrgb.SendToOpenRGB()
	}
}

// refreshOpenRGBController will refresh device in OpenRGB
func (d *Device) refreshOpenRGBController() {
	d.clearQueue()
	if controller := d.newOpenRGBController(); controller != nil {
		openrgb.UpdateDeviceController(d.Serial, controller)
	}
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
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}
	d.setDeviceColor()
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
			d.activeRgb.Exit <- true
			d.activeRgb = nil
		}

		newProfile := profile
		newProfile.Active = true
		d.DeviceProfile = newProfile
		d.saveDeviceProfile()
		d.refreshOpenRGBController()
		d.setDeviceColor()
		d.setupPerformance()
		return 1
	}
	return 0
}

// DeleteDeviceProfile deletes a device profile and its JSON file
func (d *Device) DeleteDeviceProfile(profileName string) uint8 {
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
				if d.activeRgb != nil {
					d.activeRgb.Exit <- true
					d.activeRgb = nil
				}
				d.refreshOpenRGBController()
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
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}
	d.refreshOpenRGBController()
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
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}
	d.refreshOpenRGBController()
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
				key.ToggleDelay = keyAssignment.ToggleDelay
				d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row[rowId].Keys[keyId] = key
				d.saveDeviceProfile()
				d.setupKeyAssignment()
				return 1
			}
		}
	}
	return 0
}

// buildKeyIndexMap will build keyboard map with rows and columns
func (d *Device) buildKeyIndexMap() map[int]KeyPos {
	keyIndexMap := make(map[int]KeyPos)
	keyboard := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]
	for r, row := range keyboard.Row {
		for c := range row.Keys {
			keyIndexMap[c] = KeyPos{
				Row: r,
				Col: c,
			}
		}
	}
	return keyIndexMap
}

// UpdateDeviceColor will update device color based on selected input
func (d *Device) UpdateDeviceColor(keyId, keyOption int, color rgb.Color, selections []int) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; !ok {
		return 0
	}

	switch keyOption {
	case 0:
		{
			for rowIndex, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
				for keyIndex, key := range row.Keys {
					if keyIndex == keyId {
						if key.NoColor {
							return 0
						}
						key.Color = rgb.Color{
							Red:        color.Red,
							Green:      color.Green,
							Blue:       color.Blue,
							Brightness: 0,
						}
						d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row[rowIndex].Keys[keyIndex] = key
						if d.activeRgb != nil {
							d.activeRgb.Exit <- true
							d.activeRgb = nil
						}
						d.setDeviceColor()
						return 1
					}
				}
			}
		}
	case 1:
		{
			rowId := -1
			for rowIndex, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
				for keyIndex, key := range row.Keys {
					if keyIndex == keyId {
						if key.NoColor {
							return 0
						}
						rowId = rowIndex
						break
					}
				}
			}

			if rowId < 0 {
				return 0
			}

			for keyIndex, key := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row[rowId].Keys {
				key.Color = rgb.Color{
					Red:        color.Red,
					Green:      color.Green,
					Blue:       color.Blue,
					Brightness: 0,
				}
				d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row[rowId].Keys[keyIndex] = key
			}
			if d.activeRgb != nil {
				d.activeRgb.Exit <- true
				d.activeRgb = nil
			}
			d.setDeviceColor()
			return 1
		}
	case 2:
		{
			for rowIndex, row := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
				for keyIndex, key := range row.Keys {
					if key.NoColor {
						continue
					}

					key.Color = rgb.Color{
						Red:        color.Red,
						Green:      color.Green,
						Blue:       color.Blue,
						Brightness: 0,
					}
					d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row[rowIndex].Keys[keyIndex] = key
				}
			}
			if d.activeRgb != nil {
				d.activeRgb.Exit <- true
				d.activeRgb = nil
			}
			d.setDeviceColor()
			return 1
		}
	case 3:
		{
			updated := 0
			keyIndexMap := d.buildKeyIndexMap()
			for _, val := range selections {
				pos, ok := keyIndexMap[val]
				if !ok {
					continue
				}

				key, valid := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row[pos.Row].Keys[pos.Col]
				if !valid {
					continue
				}

				key.Color = rgb.Color{
					Red:   color.Red,
					Green: color.Green,
					Blue:  color.Blue,
				}

				d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row[pos.Row].Keys[pos.Col] = key
				updated++
			}

			if updated > 0 {
				if d.activeRgb != nil {
					d.activeRgb.Exit <- true
					d.activeRgb = nil
				}
				d.setDeviceColor()
				return 1
			}
		}
	}
	return 0
}

// ControlDeviceRgb will change device brightness via schedulerSchedulerBrightness
func (d *Device) ControlDeviceRgb(value bool) {
	if d.DeviceProfile == nil {
		return
	}

	d.DeviceProfile.RgbOff = value
	d.saveDeviceProfile()

	if d.activeRgb != nil {
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}
	d.setDeviceColor()
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	// Reset
	var buf = make(map[int][]byte, colorPackets)
	for i := 0; i < colorPackets; i++ {
		buf[i] = make([]byte, 168)
	}

	if d.DeviceProfile == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to set color. DeviceProfile is null!")
		return
	}

	if d.GetRgbProfile(d.DeviceProfile.RGBProfile) == nil {
		d.DeviceProfile.RGBProfile = "keyboard"
	}

	if d.DeviceProfile.OpenRGBIntegration {
		d.writeColor(buf)
		return
	}
	if d.DeviceProfile.RgbOff {
		if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
			for _, rows := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
				for _, keys := range rows.Keys {
					for _, packetIndex := range keys.PacketIndex {
						buf[0][packetIndex] = 0x00
						buf[1][packetIndex] = 0x00
						buf[2][packetIndex] = 0x00
					}
				}
			}
			d.writeColor(buf) // Write color once
			return
		} else {
			logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to set color. Unknown keyboard")
			return
		}
	}

	if d.DeviceProfile.RGBProfile == "keyboard" {
		if _, ok := d.DeviceProfile.Keyboards[d.DeviceProfile.Profile]; ok {
			for _, rows := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
				for _, keys := range rows.Keys {
					keys.Color.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
					keys.Color = *rgb.ModifyBrightness(keys.Color)
					for _, packetIndex := range keys.PacketIndex {
						buf[0][packetIndex] = byte(keys.Color.Red)
						buf[1][packetIndex] = byte(keys.Color.Green)
						buf[2][packetIndex] = byte(keys.Color.Blue)
					}
				}
			}
			d.writeColor(buf) // Write color once
			return
		} else {
			logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to set color. Unknown keyboard")
			return
		}
	}

	if d.DeviceProfile.RGBProfile == "static" {
		profile := d.GetRgbProfile("static")
		if profile == nil {
			return
		}

		profile.StartColor.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
		profileColor := rgb.ModifyBrightness(profile.StartColor)
		for _, rows := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
			for _, keys := range rows.Keys {
				keys.Color.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
				keys.Color = *rgb.ModifyBrightness(keys.Color)

				for _, packetIndex := range keys.PacketIndex {
					buf[0][packetIndex] = byte(profileColor.Red)
					buf[1][packetIndex] = byte(profileColor.Green)
					buf[2][packetIndex] = byte(profileColor.Blue)
				}
			}
		}
		d.writeColor(buf) // Write color once
		return
	}

	if d.DeviceProfile.RGBProfile == "off" {
		for _, rows := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
			for _, keys := range rows.Keys {
				for _, packetIndex := range keys.PacketIndex {
					buf[0][packetIndex] = byte(0)
					buf[1][packetIndex] = byte(0)
					buf[2][packetIndex] = byte(0)
				}
			}
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
				rgbCustomColor := true
				profile := d.GetRgbProfile(d.DeviceProfile.RGBProfile)
				if profile == nil {
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
					r.RGBMiddleColor = &profile.MiddleColor
				} else {
					r.RGBStartColor = d.activeRgb.RGBStartColor
					r.RGBEndColor = d.activeRgb.RGBEndColor
					r.RGBMiddleColor = d.activeRgb.RGBMiddleColor
				}

				if r.RGBMiddleColor == nil {
					r.RGBMiddleColor = &rgb.Color{}
				}

				// Brightness
				r.RGBBrightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
				r.RGBStartColor.Brightness = r.RGBBrightness
				r.RGBEndColor.Brightness = r.RGBBrightness
				r.RGBMiddleColor.Brightness = r.RGBBrightness

				switch d.DeviceProfile.RGBProfile {
				case "rainbow":
					{
						r.Rainbow(startTime)
					}
				case "pastelrainbow":
					{
						r.PastelRainbow(startTime)
					}
				case "watercolor":
					{
						r.Watercolor(startTime)
					}
				case "gradient":
					{
						r.ColorshiftGradient(startTime, profile.Gradients, profile.Speed)
					}
				case "cpu-temperature":
					{
						r.MinTemp = profile.MinTemp
						r.MaxTemp = profile.MaxTemp
						r.Temperature(float64(d.CpuTemp))
					}
				case "gpu-temperature":
					{
						r.MinTemp = profile.MinTemp
						r.MaxTemp = profile.MaxTemp
						r.Temperature(float64(d.GpuTemp))
					}
				case "colorpulse":
					{
						r.Colorpulse(&startTime)
					}
				case "static":
					{
						r.Static()
					}
				case "rotator":
					{
						r.Rotator(&startTime)
					}
				case "wave":
					{
						r.Wave(&startTime)
					}
				case "storm":
					{
						r.Storm()
					}
				case "flickering":
					{
						r.Flickering(&startTime)
					}
				case "colorshift":
					{
						r.Colorshift(&startTime, d.activeRgb)
					}
				case "circleshift":
					{
						r.CircleShift(&startTime)
					}
				case "circle":
					{
						r.Circle(&startTime)
					}
				case "spinner":
					{
						r.Spinner(&startTime)
					}
				case "colorwarp":
					{
						r.Colorwarp(&startTime, d.activeRgb)
					}
				}

				if len(r.Raw) == 0 {
					continue
				}

				for _, rows := range d.DeviceProfile.Keyboards[d.DeviceProfile.Profile].Row {
					for _, keys := range rows.Keys {
						for _, packetIndex := range keys.PacketIndex {
							buf[0][packetIndex] = r.Raw[packetIndex][0]
							buf[1][packetIndex] = r.Raw[packetIndex][1]
							buf[2][packetIndex] = r.Raw[packetIndex][2]
						}
					}
				}

				d.writeColor(buf)
				time.Sleep(20 * time.Millisecond)
			}
		}
	}(d.LEDChannels)
}

// writeColorEx will write data to the device from OpenRGB client
func (d *Device) writeColorEx(data []byte, offsets []int) {
	if d.Exit || d.DeviceProfile == nil || !d.DeviceProfile.OpenRGBIntegration {
		return
	}

	select {
	case <-d.queueStop:
		return
	default:
	}

	buffer := d.openRGBColorBuffer(data, offsets)
	if buffer == nil {
		return
	}
	select {
	case <-d.queueStop:
		return
	case d.queue <- buffer:
	default:
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

// stopQueueWorker will stop queue worker for OpenRGB client
func (d *Device) stopQueueWorker() {
	if d.queueStop == nil {
		return
	}
	d.queueStopOnce.Do(func() { close(d.queueStop) })
	d.clearQueue()
}

// startQueueWorker will initialize queue system and control packet flow towards the device
func (d *Device) startQueueWorker() {
	d.queue = make(chan []byte, 10)
	d.queueStop = make(chan struct{})
	d.queueDone = make(chan struct{})
	go func() {
		defer close(d.queueDone)
		for {
			var data []byte
			select {
			case <-d.queueStop:
				return
			case data = <-d.queue:
			}

			d.deviceLock.Lock()
			select {
			case <-d.queueStop:
				d.deviceLock.Unlock()
				return
			default:
			}
			if d.Exit || d.DeviceProfile == nil || !d.DeviceProfile.OpenRGBIntegration {
				d.deviceLock.Unlock()
				continue
			}
			if d.DeviceProfile.RgbOff {
				clear(data)
			}
			d.writeColorBuffer(map[int][]byte{0: data[:168], 1: data[168:336], 2: data[336:]})
			d.deviceLock.Unlock()

			select {
			case <-d.queueStop:
				return
			case <-time.After(20 * time.Millisecond):
			}
		}
	}()
}

// openRGBColorBuffer generates OpenRGB buffer for colors
func (d *Device) openRGBColorBuffer(data []byte, offsets []int) []byte {
	if len(data) != len(offsets)*3 {
		return nil
	}
	buffer := make([]byte, 168*3)
	for i, offset := range offsets {
		if offset < 0 || offset >= 168 {
			return nil
		}
		buffer[offset] = data[i*3]
		buffer[168+offset] = data[i*3+1]
		buffer[2*168+offset] = data[i*3+2]
	}

	return buffer
}

// writeColor will write color data to the device
func (d *Device) writeColor(data map[int][]byte) {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if d.Exit {
		return
	}

	d.writeColorBuffer(data)
}

// writeColorBuffer writes color buffer to the device
func (d *Device) writeColorBuffer(data map[int][]byte) {
	// Always override Lock color
	if d.DeviceProfile.Performance {
		data[0][8] = 255
		data[1][8] = 0
		data[2][8] = 0
	}

	for i := 0; i < len(data); i++ {
		chunks := common.ProcessMultiChunkPacket(data[i], maximumPacketSize)
		for m, chunk := range chunks {
			buf := make([]byte, 63)
			buf[0] = byte(m + 1)
			buf[1] = byte(len(chunk))
			buf[2] = 0x00
			copy(buf[3:], chunk)
			err := d.transfer(cmdWriteColor, buf, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
			}
		}

		if i == 0 || i == 1 {
			dataTypeColor[3] = 0x01
		} else {
			dataTypeColor[3] = 0x02
		}
		dataTypeColor[1] = byte(i + 1)
		e := d.transfer(cmdWrite, dataTypeColor, nil)
		if e != nil {
			logger.Log(logger.Fields{"error": e, "serial": d.Serial}).Error("Unable to write to color endpoint")
		}
		time.Sleep(time.Millisecond * 2)
	}
}

// getListenerData will listen for keyboard events and return data on success or nil on failure.
// ReadWithTimeout is mandatory due to the nature of listening for events
func (d *Device) getListenerData() []byte {
	if d.listener == nil {
		return nil
	}
	data := make([]byte, bufferSize)
	n, err := d.listener.ReadWithTimeout(data, 100*time.Millisecond)
	if err != nil || n == 0 {
		return nil
	}
	return data
}

// controlButtonListener will listen for events from the control buttons
func (d *Device) backendListener() {
	go func() {
		enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
			if info.InterfaceNbr == 0 {
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

		if d.listener == nil {
			logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to open device listener")
			return
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
					time.Sleep(5 * time.Millisecond)
					continue
				}

				d.triggerKeyAssignment(data)
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()
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
func (d *Device) triggerKeyAssignment(value []byte) {
	raw := make([]byte, len(value))
	if value[0] == 0x03 {
		raw = value[1:21]
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
			case 8:
				if d.dispatch != nil {
					d.dispatch(d.KeyboardKey.DeviceId, "CallSniperMode", d.PressLoop)
				}
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
				d.activeRgb.Exit <- true
				d.activeRgb = nil
			}
			d.setDeviceColor()
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
		case 8:
			if key.ActionHold {
				d.KeyboardKey = key
			}
			if d.dispatch != nil && len(key.DeviceId) > 0 {
				d.dispatch(key.DeviceId, "CallSniperMode", key.ActionHold)
			}
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
						if v.ActionRepeat > 0 && !v.ActionHold {
							d.stopRepeatMutex.Lock()
							if d.stopRepeat != nil {
								close(d.stopRepeat)
							}

							d.stopRepeat = make(chan struct{})
							localStop := d.stopRepeat
							d.stopRepeatMutex.Unlock()

							go func() {
								for z := 0; z < int(v.ActionRepeat); z++ {
									select {
									case <-localStop:
										return
									default:
										inputmanager.InputControlKeyboard(v.ActionCommand, false)
									}
									if v.ActionRepeatDelay > 0 && v.ActionRepeat > 1 {
										time.Sleep(time.Duration(v.ActionRepeatDelay) * time.Millisecond)
									}
								}
							}()

						} else {
							inputmanager.InputControlKeyboard(v.ActionCommand, v.ActionHold)
						}
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
					case 20:
						if v.ActionRepeat > 0 && !v.ActionHold {
							d.stopRepeatMutex.Lock()
							if d.stopRepeat != nil {
								close(d.stopRepeat)
							}

							d.stopRepeat = make(chan struct{})
							localStop := d.stopRepeat
							d.stopRepeatMutex.Unlock()

							go func() {
								for z := 0; z < int(v.ActionRepeat); z++ {
									select {
									case <-localStop:
										return
									default:
										if v.MousePositionAbsolute {
											inputmanager.InputControlMoveAbsolute(int32(v.MousePositionX), int32(v.MousePositionY))
										} else {
											inputmanager.InputControlMove(int32(v.MousePositionX), int32(v.MousePositionY))
										}
									}
									if v.ActionRepeatDelay > 0 && v.ActionRepeat > 1 {
										time.Sleep(time.Duration(v.ActionRepeatDelay) * time.Millisecond)
									}
								}
							}()
						} else {
							if v.MousePositionAbsolute {
								inputmanager.InputControlMoveAbsolute(int32(v.MousePositionX), int32(v.MousePositionY))
							} else {
								inputmanager.InputControlMove(int32(v.MousePositionX), int32(v.MousePositionY))
							}
						}
						break
					}
				}
			}
			break
		}
	}
}

// buildOpenRGBKeyboard will build OpenRGB compatibly keyboard layout
func (d *Device) buildOpenRGBKeyboard(keyboard *keyboards.Keyboard) ([]common.OpenRGGLed, [][]uint32, []int, []byte) {
	if keyboard == nil {
		return nil, nil, nil, nil
	}
	const na = -1

	positions := [7][27]int{
		{11, 23, 35, 47, 137, 8, na, na, na, na, na, na, na, na, na, na, na, na, na, na, na, na, na, 20, na, na, na},
		{10, 22, 34, 0, na, 12, 24, 36, 48, 60, 72, 84, 96, 108, 120, 132, 6, na, na, 18, 30, 42, na, 32, 44, 56, 68},
		{46, 58, 70, 1, 13, 25, 37, 49, 61, 73, 85, 97, 109, 121, 133, 7, na, 31, na, 54, 66, 78, na, 80, 92, 104, 116},
		{82, 94, 106, 2, na, 14, 26, 38, 50, 62, 74, 86, 98, 110, 122, 134, 90, na, na, 43, 55, 67, na, 9, 21, 33, 128},
		{118, 59, 71, 3, na, 15, 27, 39, 51, 63, 75, 87, 99, 111, 123, 135, 114, 126, na, na, na, na, na, 57, 69, 81, na},
		{83, 95, 107, 4, 16, 28, 40, 52, 64, 76, 88, 100, 112, 124, 136, 79, na, na, na, na, 103, na, na, 93, 105, 117, 140},
		{119, 131, 143, 5, 17, 29, na, na, na, 53, na, na, 89, 101, 113, na, 91, na, na, 115, 127, 139, na, 129, na, 141, na},
	}
	byAddress := make(map[int]keyboards.Key)
	for _, row := range keyboard.Row {
		for _, key := range row.Keys {
			if key.NoColor {
				continue
			}
			for _, offset := range key.PacketIndex {
				if offset >= 0 && offset < 168 {
					byAddress[offset] = key
				}
			}
		}
	}

	_, ansi := byAddress[102]
	if ansi {
		positions[3][17] = 102
		positions[4][16], positions[4][17] = 126, na
	}

	matrix := make([][]uint32, len(positions))
	for y := range matrix {
		matrix[y] = make([]uint32, len(positions[y]))
		for x := range matrix[y] {
			matrix[y][x] = common.NA
		}
	}

	var leds []common.OpenRGGLed
	var offsets []int
	var colors []byte

	for x := 0; x < len(positions[0]); x++ {
		for y := range positions {
			address := positions[y][x]
			key, ok := byAddress[address]
			if !ok {
				continue
			}

			matrix[y][x] = uint32(len(leds))
			leds = append(leds,
				common.OpenRGGLed{
					Name:  d.openRGBKeyName(key, address, ansi),
					Value: uint32(address),
				},
			)

			offsets = append(offsets, address)
			colors = append(colors, byte(key.Color.Red), byte(key.Color.Green), byte(key.Color.Blue))
		}
	}

	return leds, matrix, offsets, colors
}

// openRGBKeyName handles mismatch in key names
func (d *Device) openRGBKeyName(key keyboards.Key, address int, ansi bool) string {
	controls := map[int]string{
		10:  "Key: G1",
		22:  "Key: G2",
		34:  "Key: G3",
		46:  "Key: G4",
		58:  "Key: G5",
		70:  "Key: G6",
		82:  "Key: G7",
		94:  "Key: G8",
		106: "Key: G9",
		118: "Key: G10",
		59:  "Key: G11",
		71:  "Key: G12",
		83:  "Key: G13",
		95:  "Key: G14",
		107: "Key: G15",
		119: "Key: G16",
		131: "Key: G17",
		143: "Key: G18",
		11:  "Key: MR",
		23:  "Key: M1",
		35:  "Key: M2",
		47:  "Key: M3",
	}
	if name, ok := controls[address]; ok {
		return name
	}

	addresses := map[int]int{
		0:   41,
		1:   53,
		2:   43,
		3:   57,
		4:   106,
		5:   105,
		6:   69,
		7:   46,
		8:   114,
		9:   95,
		12:  58,
		13:  30,
		14:  20,
		15:  4,
		16:  100,
		17:  108,
		18:  70,
		20:  102,
		21:  96,
		24:  59,
		25:  31,
		26:  26,
		27:  22,
		28:  29,
		29:  107,
		30:  71,
		31:  42,
		32:  123,
		33:  97,
		36:  60,
		37:  32,
		38:  8,
		39:  7,
		40:  27,
		42:  72,
		43:  76,
		44:  126,
		47:  137,
		48:  61,
		49:  33,
		50:  21,
		51:  9,
		52:  6,
		53:  44,
		54:  73,
		55:  77,
		56:  124,
		57:  92,
		59:  138,
		60:  62,
		61:  34,
		62:  23,
		63:  10,
		64:  25,
		66:  74,
		67:  78,
		68:  125,
		69:  93,
		72:  63,
		73:  35,
		74:  28,
		75:  11,
		76:  5,
		78:  75,
		79:  110,
		80:  83,
		81:  94,
		84:  64,
		85:  36,
		86:  24,
		87:  13,
		88:  17,
		89:  111,
		90:  48,
		91:  109,
		92:  84,
		93:  89,
		96:  65,
		97:  37,
		98:  12,
		99:  14,
		100: 16,
		101: 112,
		102: 49,
		103: 82,
		104: 85,
		105: 90,
		108: 66,
		109: 38,
		110: 18,
		111: 15,
		112: 54,
		113: 101,
		114: 50,
		115: 80,
		116: 86,
		117: 91,
		120: 67,
		121: 39,
		122: 19,
		123: 51,
		124: 55,
		125: 128,
		126: 40,
		127: 81,
		128: 87,
		129: 98,
		132: 68,
		133: 45,
		134: 47,
		135: 52,
		136: 56,
		137: 113,
		139: 79,
		140: 88,
		141: 99,
	}
	if mapped, ok := addresses[address]; ok {
		address = mapped
	} else {
		return fmt.Sprintf("LED %d", address)
	}

	if address >= 58 && address <= 69 {
		return fmt.Sprintf("Key: F%d", address-57)
	}

	if address >= 89 && address <= 97 {
		return fmt.Sprintf("Key: Number Pad %d", address-88)
	}

	special := map[int]string{
		70:  "Print Screen",
		76:  "Delete",
		83:  "Num Lock",
		84:  "Number Pad /",
		85:  "Number Pad *",
		86:  "Number Pad -",
		87:  "Number Pad +",
		88:  "Number Pad Enter",
		98:  "Number Pad 0",
		99:  "Number Pad .",
		80:  "Left Arrow",
		81:  "Down Arrow",
		82:  "Up Arrow",
		79:  "Right Arrow",
		123: "Media Stop",
		128: "Profile",
		113: "Brightness",
		114: "Windows Lock",
		126: "Media Previous",
		124: "Media Play/Pause",
		125: "Media Next",
		102: "Media Mute",
		71:  "Scroll Lock",
		72:  "Pause/Break",
		73:  "Insert",
		74:  "Home",
		75:  "Page Up",
		77:  "End",
		78:  "Page Down",
		131: "G1",
		132: "G2",
		133: "G3",
		134: "G4",
		41:  "Escape",
		42:  "Backspace",
		43:  "Tab",
		57:  "Caps Lock",
		106: "Left Shift",
		110: "Right Shift",
		105: "Left Control",
		108: "Left Windows",
		107: "Left Alt",
		111: "Right Alt",
		112: "Right Windows",
		122: "Right Fn",
		101: "Menu",
		109: "Right Control",
		44:  "Space",
	}

	if address == 137 || address == 138 || address == 140 || address == 141 {
		return ""
	}

	if address == 40 {
		if ansi {
			return "Key: Enter"
		}
		return "Key: Enter (ISO)"
	}

	if name, ok := special[address]; ok {
		return "Key: " + name
	}

	if address == 100 && strings.HasPrefix(key.KeyName, "\\") {
		return "Key: \\ (ISO)"
	}

	switch key.KeyName {
	case "Sz \\ ?", "Sz":
		return "Key: ß"
	case "'`":
		return "Key: `"
	case "< >", "<>|", "<":
		return "Key: < >"
	}

	fields := strings.Fields(key.KeyName)
	if len(fields) == 0 {
		return fmt.Sprintf("LED %d", address)
	}

	return "Key: " + fields[0]
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(command byte, endpoint, buffer []byte) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = command
	endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
	}

	if _, err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return err
	}
	return nil
}
