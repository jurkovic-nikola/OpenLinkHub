package sabrev2proWU

// Package: CORSAIR SABRE V2 PRO
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/macro"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/stats"
	"encoding/json"
	"fmt"
	"math/bits"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sstallion/go-hid"
)

const (
	EvKey      = 0x01
	BtnLeft    = 0x110
	BtnRight   = 0x111
	BtnMiddle  = 0x112
	BtnBack    = 0x113 // 275
	BtnForward = 0x114 // 276
)

type ZoneColors struct {
	Color      *rgb.Color
	ColorIndex []int
	Name       string
}

type ButtonProto struct {
	Cmd  byte
	Tail byte
}

// DeviceProfile struct contains all device profile
type DeviceProfile struct {
	Active             bool
	Path               string
	Product            string
	Serial             string
	Brightness         uint8
	RGBProfile         string
	BrightnessSlider   *uint8
	OriginalBrightness uint8
	Label              string
	Profile            int
	PollingRate        int
	Profiles           map[int]DPIProfile
	SleepMode          int
	AngleSnapping      int
	ButtonOptimization int
	LiftHeight         int
	RippleControl      int
	MotionSync         int
	KeyAssignmentHash  string
	RgbOff             bool
}

type DPIProfile struct {
	Name        string `json:"name"`
	Value       uint16
	PackerIndex int
	ColorIndex  map[int][]int
	Color       *rgb.Color
	Sniper      bool
}

type Device struct {
	Debug              bool
	dev                *hid.Device
	mouse              *os.File
	Manufacturer       string                    `json:"manufacturer"`
	Product            string                    `json:"product"`
	Serial             string                    `json:"serial"`
	Firmware           string                    `json:"firmware"`
	UserProfiles       map[string]*DeviceProfile `json:"userProfiles"`
	Devices            map[int]string            `json:"devices"`
	DeviceProfile      *DeviceProfile
	OriginalProfile    *DeviceProfile
	Template           string
	VendorId           uint16
	ProductId          uint16
	Brightness         map[int]string
	PollingRates       map[int]string
	SwitchModes        map[int]string
	KeyAssignmentTypes map[int]string
	CpuTemp            float32
	GpuTemp            float32
	Layouts            []string
	SleepModes         map[int]string
	LiftHeights        map[int]string
	mutex              sync.Mutex
	timerKeepAlive     *time.Ticker
	keepAliveChan      chan struct{}
	Exit               bool
	KeyAssignment      map[int]inputmanager.KeyAssignment
	InputActions       map[uint16]inputmanager.InputAction
	PressLoop          bool
	keyAssignmentFile  string
	BatteryLevel       uint16
	KeyAssignmentData  *inputmanager.KeyAssignment
	ModifierIndex      byte
	SniperMode         bool
	MacroTracker       map[int]uint16
	RGBModes           []string
	instance           *common.Device
	Usb                bool
	Connected          bool
	stopRepeat         chan struct{}
	stopRepeatMutex    sync.Mutex
	MinDPI             int
	MaxDPI             int
	ZoneAmount         int
	DPIAmount          int
	DPIValue           uint16
}

var (
	pwd              = ""
	cmdSetDpi        = []byte{0x04, 0x02}
	cmdWriteDpi      = byte(0x04)
	cmdWrite         = []byte{0x07}
	cmdRead          = []byte{0x04}
	cmdAngleSnapping = byte(0xAF)
	cmdRippleControl = byte(0xB1)
	cmdMotionSync    = byte(0xAB)
	bufferSize       = 64
	bufferSizeWrite  = bufferSize + 1
	headerSize       = 2
	keyAmount        = 5
	minDpiValue      = 100
	maxDpiValue      = 33000
	deviceKeepAlive  = 20000
	buttonByKey      = map[int]ButtonProto{
		2:  {Cmd: 0x64, Tail: 0x89}, // Right
		4:  {Cmd: 0x68, Tail: 0x85}, // Middle
		8:  {Cmd: 0x6C, Tail: 0x81}, // Back
		16: {Cmd: 0x70, Tail: 0x7D}, // Forward
	}
	rgbModes = []string{
		"colorpulse",
		"colorshift",
		"colorwarp",
		"cpu-temperature",
		"gpu-temperature",
		"mouse",
		"off",
		"rainbow",
		"static",
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
		Usb:       true,
		Connected: true,
		dev:       dev,
		Template:  "sabrev2proW.html",
		VendorId:  vendorId,
		ProductId: productId,
		Firmware:  "n/a",
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
		Product: "SABRE V2 PRO",
		SleepModes: map[int]string{
			1:  "1 minute",
			3:  "3 minutes",
			5:  "5 minutes",
			10: "10 minutes",
			20: "20 minutes",
			30: "30 minutes",
			40: "40 minutes",
		},
		RGBModes: rgbModes,
		PollingRates: map[int]string{
			0: "Not Set",
			1: "1000 Hz / 1 msec",
			2: "2000 Hz / 0.5 msec",
			3: "4000 Hz / 0.25 msec",
			4: "8000 Hz / 0.125 msec",
		},
		SwitchModes: map[int]string{
			0: "Disabled",
			1: "Enabled",
		},
		KeyAssignmentTypes: map[int]string{
			0:  "None",
			1:  "Media Keys",
			2:  "DPI",
			3:  "Keyboard",
			8:  "Sniper",
			9:  "Mouse",
			10: "Macro",
		},
		LiftHeights: map[int]string{
			1: "Low",
			2: "Medium",
			3: "High",
		},
		InputActions:      inputmanager.GetInputActions(),
		keyAssignmentFile: "/database/key-assignments/sabrev2pro.json",
		MacroTracker:      make(map[int]uint16),
		MinDPI:            minDpiValue,
		MaxDPI:            maxDpiValue,
		DPIAmount:         6,
	}

	d.getDebugMode()       // Debug mode
	d.getManufacturer()    // Manufacturer
	d.getSerial()          // Serial
	d.loadDeviceProfiles() // Load all device profiles
	d.saveDeviceProfile()  // Save profile
	d.getDeviceFirmware()  // Firmware
	d.setLiftHeight()      // Lift Height
	d.setAngleSnapping()   // Angle snapping
	d.setRippleControl()   // Ripple Control
	d.setMotionSync()      // Motion Sync
	d.setSleepTimer()      // Sleep timer
	d.getBatterLevel()     // Battery level
	d.writeDPI()           // Write DPI
	d.toggleDPI()          // DPI
	d.mouseListener()      // Mouse listener
	d.setKeepAlive()       // Keepalive
	d.loadKeyAssignments() // Key Assignments
	d.setupKeyAssignment() // Setup key assignments
	d.createDevice()       // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeSabreV2Pro,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-mouse.svg",
		Instance:    d,
	}
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")

	d.timerKeepAlive.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.keepAliveChan != nil {
				close(d.keepAliveChan)
			}
		})
	}()

	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			return
		}
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
}

// StopDirty will stop device in a dirty way
func (d *Device) StopDirty() uint8 {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")

	d.timerKeepAlive.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.keepAliveChan != nil {
				close(d.keepAliveChan)
			}
		})
	}()
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 1
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
	d.Serial = "11048"
}

// GetDeviceTemplate will return device template name
func (d *Device) GetDeviceTemplate() string {
	return d.Template
}

// ChangeDeviceProfile will change device profile
func (d *Device) ChangeDeviceProfile(profileName string) uint8 {
	if profile, ok := d.UserProfiles[profileName]; ok {
		currentProfile := d.DeviceProfile
		currentProfile.Active = false
		d.DeviceProfile = currentProfile
		d.saveDeviceProfile()

		newProfile := profile
		newProfile.Active = true
		d.DeviceProfile = newProfile
		d.saveDeviceProfile()
		d.loadKeyAssignments()
		d.setupKeyAssignment()
		return 1
	}
	return 0
}

// SaveUserProfile will generate a new user profile configuration and save it to a file
func (d *Device) SaveUserProfile(profileName string) uint8 {
	if d.DeviceProfile != nil {
		profilePath := pwd + "/database/profiles/" + d.Serial + "-" + profileName + ".json"
		keyAssignmentHash := common.GenerateRandomMD5()

		newProfile := d.DeviceProfile
		newProfile.Path = profilePath
		newProfile.Active = false
		newProfile.KeyAssignmentHash = keyAssignmentHash

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
		d.saveKeyAssignments()
		d.loadDeviceProfiles()
		return 1
	}
	return 0
}

// SaveMouseDPI will save mouse DPI
func (d *Device) SaveMouseDPI(stages map[int]uint16) uint8 {
	i := 0
	if d.DeviceProfile == nil {
		return 0
	}

	if len(stages) == 0 {
		return 0
	}

	for key, stage := range stages {
		if _, ok := d.DeviceProfile.Profiles[key]; ok {
			profile := d.DeviceProfile.Profiles[key]
			if stage > uint16(maxDpiValue) {
				continue
			}
			if stage < uint16(minDpiValue) {
				continue
			}

			profile.Value = ((stage + 25) / 50) * 50 // This mouse increments DPI by 50
			d.DeviceProfile.Profiles[key] = profile
			i++
		}
	}

	if i > 0 {
		d.saveDeviceProfile()
		d.writeDPI()
		d.toggleDPI()
		return 1
	}
	return 0
}

// getManufacturer will return device manufacturer
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
}

// getBatterLevel will return initial battery level
func (d *Device) getBatterLevel() {
	buf := make([]byte, 15)
	buf[14] = 0x49
	batteryLevel, err := d.transfer(cmdRead, buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get battery level")
	}

	d.BatteryLevel = uint16(batteryLevel[6])
	stats.UpdateBatteryStats(d.Serial, d.Product, d.BatteryLevel, 1)
}

// GetSleepMode will return current sleep mode
func (d *Device) GetSleepMode() int {
	if d.DeviceProfile != nil {
		return d.DeviceProfile.SleepMode
	}
	return 0
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

// setLiftHeight will change mouse lift height
func (d *Device) setLiftHeight() {
	if d.DeviceProfile == nil {
		return
	}

	if d.DeviceProfile.LiftHeight < 1 || d.DeviceProfile.LiftHeight > 3 {
		return
	}

	buf := d.buildLiftHeightPacket(d.DeviceProfile.LiftHeight)
	_, _ = d.transfer(cmdWrite, buf)
}

// setAngleSnapping will change Angle Snapping mode
func (d *Device) setAngleSnapping() {
	if d.DeviceProfile == nil {
		return
	}

	if d.DeviceProfile.AngleSnapping < 0 || d.DeviceProfile.AngleSnapping > 1 {
		return
	}

	buf := d.buildTogglePacket(cmdAngleSnapping, d.DeviceProfile.AngleSnapping == 1, 0x40)
	_, _ = d.transfer(cmdWrite, buf)
}

// setRippleControl will change ripple control
func (d *Device) setRippleControl() {
	if d.DeviceProfile == nil {
		return
	}

	if d.DeviceProfile.RippleControl < 0 || d.DeviceProfile.RippleControl > 1 {
		return
	}

	buf := d.buildTogglePacket(cmdRippleControl, d.DeviceProfile.RippleControl == 1, 0x3E)
	_, _ = d.transfer(cmdWrite, buf)
}

// setMotionSync will change motion sync
func (d *Device) setMotionSync() {
	if d.DeviceProfile == nil {
		return
	}

	if d.DeviceProfile.MotionSync < 0 || d.DeviceProfile.MotionSync > 1 {
		return
	}

	buf := d.buildTogglePacket(cmdMotionSync, d.DeviceProfile.MotionSync == 1, 0x44)
	_, _ = d.transfer(cmdWrite, buf)
}

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	var defaultBrightness = uint8(100)
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"

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
		deviceProfile.RGBProfile = "mouse"
		deviceProfile.Label = "Mouse"
		deviceProfile.Active = true
		deviceProfile.Profiles = map[int]DPIProfile{
			0: {
				Name:        "Stage 1",
				Value:       400,
				PackerIndex: 12,
				Color: &rgb.Color{
					Red:        255,
					Green:      40,
					Blue:       79,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 255, 40, 79),
				},
			},
			1: {
				Name:        "Stage 2",
				Value:       800,
				PackerIndex: 16,
				Color: &rgb.Color{
					Red:        61,
					Green:      100,
					Blue:       255,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 61, 100, 255),
				},
			},
			2: {
				Name:        "Stage 3",
				Value:       1200,
				PackerIndex: 20,
				Color: &rgb.Color{
					Red:        1,
					Green:      182,
					Blue:       96,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 1, 182, 96),
				},
			},
			3: {
				Name:        "Stage 4",
				Value:       1600,
				PackerIndex: 24,
				Color: &rgb.Color{
					Red:        255,
					Green:      214,
					Blue:       0,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 255, 214, 0),
				},
			},
			4: {
				Name:        "Stage 5",
				Value:       2000,
				PackerIndex: 28,
				Color: &rgb.Color{
					Red:        255,
					Green:      62,
					Blue:       235,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 255, 62, 235),
				},
			},
			5: {
				Name:        "Sniper",
				Value:       200,
				PackerIndex: 6,
				Sniper:      true,
				Color: &rgb.Color{
					Red:        0,
					Green:      0,
					Blue:       0,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 0, 0, 0),
				},
			},
		}
		deviceProfile.Profile = 1
		deviceProfile.SleepMode = 5
		deviceProfile.PollingRate = 1
		deviceProfile.LiftHeight = 2
		deviceProfile.MotionSync = 1
	} else {
		if d.DeviceProfile.BrightnessSlider == nil {
			deviceProfile.BrightnessSlider = &defaultBrightness
			d.DeviceProfile.BrightnessSlider = &defaultBrightness
		} else {
			deviceProfile.BrightnessSlider = d.DeviceProfile.BrightnessSlider
		}

		if d.DeviceProfile.SleepMode == 0 {
			deviceProfile.SleepMode = 15
		} else {
			deviceProfile.SleepMode = d.DeviceProfile.SleepMode
		}

		if d.DeviceProfile.PollingRate == 0 {
			deviceProfile.PollingRate = 4
		} else {
			deviceProfile.PollingRate = d.DeviceProfile.PollingRate
		}

		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.Brightness = d.DeviceProfile.Brightness
		deviceProfile.OriginalBrightness = d.DeviceProfile.OriginalBrightness
		deviceProfile.RGBProfile = d.DeviceProfile.RGBProfile
		deviceProfile.Label = d.DeviceProfile.Label
		deviceProfile.Profiles = d.DeviceProfile.Profiles
		deviceProfile.Profile = d.DeviceProfile.Profile
		deviceProfile.AngleSnapping = d.DeviceProfile.AngleSnapping
		deviceProfile.ButtonOptimization = d.DeviceProfile.ButtonOptimization
		deviceProfile.KeyAssignmentHash = d.DeviceProfile.KeyAssignmentHash
		deviceProfile.LiftHeight = d.DeviceProfile.LiftHeight
		deviceProfile.RippleControl = d.DeviceProfile.RippleControl
		deviceProfile.MotionSync = d.DeviceProfile.MotionSync

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

// UpdateDeviceKeyAssignment will update device key assignments
func (d *Device) UpdateDeviceKeyAssignment(keyIndex int, keyAssignment inputmanager.KeyAssignment) uint8 {
	if inputmanager.GetVirtualMouse() == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Warn("Virtual mouse is not available. Key AssignmentS are blocked.")
		return 0
	}

	if val, ok := d.KeyAssignment[keyIndex]; ok {
		val.Default = keyAssignment.Default
		val.ActionHold = keyAssignment.ActionHold
		val.ActionType = keyAssignment.ActionType
		val.ActionCommand = keyAssignment.ActionCommand
		val.IsMacro = keyAssignment.IsMacro
		d.KeyAssignment[keyIndex] = val
		d.saveKeyAssignments()
		d.setupKeyAssignment()
		return 1
	}
	return 0
}

func (d *Device) saveKeyAssignments() {
	keyAssignmentsFile := pwd + d.keyAssignmentFile
	if len(d.DeviceProfile.KeyAssignmentHash) > 0 {
		fileFormat := fmt.Sprintf("/database/key-assignments/%s.json", d.DeviceProfile.KeyAssignmentHash)
		keyAssignmentsFile = pwd + fileFormat
	}

	// Convert to JSON
	buffer, err := json.MarshalIndent(d.KeyAssignment, "", "    ")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
		return
	}

	// Create profile filename
	file, err := os.Create(keyAssignmentsFile)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": keyAssignmentsFile}).Error("Unable to create new device profile")
		return
	}

	// Write JSON buffer to file
	_, err = file.Write(buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": keyAssignmentsFile}).Error("Unable to write data")
		return
	}

	// Close file
	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": keyAssignmentsFile}).Error("Unable to close file handle")
	}
}

// loadKeyAssignments will load custom key assignments
func (d *Device) loadKeyAssignments() {
	if d.DeviceProfile == nil {
		return
	}
	keyAssignmentsFile := pwd + d.keyAssignmentFile
	if len(d.DeviceProfile.KeyAssignmentHash) > 0 {
		fileFormat := fmt.Sprintf("/database/key-assignments/%s.json", d.DeviceProfile.KeyAssignmentHash)
		keyAssignmentsFile = pwd + fileFormat
	}

	if common.FileExists(keyAssignmentsFile) {
		file, err := os.Open(keyAssignmentsFile)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": keyAssignmentsFile}).Warn("Unable to load JSON file")
			return
		}

		if err = json.NewDecoder(file).Decode(&d.KeyAssignment); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": keyAssignmentsFile}).Warn("Unable to decode key assignments JSON")
			return
		}

		// Prevent left click modifications
		if !d.KeyAssignment[1].Default {
			logger.Log(logger.Fields{"serial": d.Serial, "value": d.KeyAssignment[1].Default, "expectedValue": 1}).Warn("Restoring left button to original value")
			var val = d.KeyAssignment[1]
			val.Default = true
			d.KeyAssignment[1] = val
		}

		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"location": keyAssignmentsFile, "serial": d.Serial}).Warn("Failed to close file handle")
		}
	} else {
		var keyAssignment = map[int]inputmanager.KeyAssignment{
			16: {
				Name:          "Forward Button",
				Default:       true,
				ActionType:    9,
				ActionCommand: 95,
				ActionHold:    false,
				ButtonIndex:   12,
			},
			8: {
				Name:          "Back Button",
				Default:       true,
				ActionType:    9,
				ActionCommand: 94,
				ActionHold:    false,
				ButtonIndex:   8,
			},
			4: {
				Name:          "Middle Button",
				Default:       true,
				ActionType:    9,
				ActionCommand: 93,
				ActionHold:    false,
				ButtonIndex:   4,
			},
			2: {
				Name:          "Right Button",
				Default:       true,
				ActionType:    9,
				ActionCommand: 92,
				ActionHold:    false,
				ButtonIndex:   0,
			},
			1: {
				Name:          "Left Button",
				Default:       true,
				ActionType:    9,
				ActionCommand: 91,
				ActionHold:    true,
				ButtonIndex:   0,
			},
		}

		// Convert to JSON
		buffer, err := json.MarshalIndent(keyAssignment, "", "    ")
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
			return
		}

		file, err := os.Create(keyAssignmentsFile)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": keyAssignmentsFile}).Error("Unable to create new key assignment file")
			return
		}

		_, err = file.Write(buffer)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": keyAssignmentsFile}).Error("Unable to write data tp key assignment file")
			return
		}

		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": keyAssignmentsFile}).Error("Unable to close key assignment file")
		}
		d.KeyAssignment = keyAssignment
	}
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

// setSleepTimer will set device sleep timer
func (d *Device) setSleepTimer() {
	if d.DeviceProfile != nil {
		buf := d.buildSleepPacket(d.DeviceProfile.SleepMode)
		_, _ = d.transfer(cmdWrite, buf)
	}
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

// UpdatePollingRate will set device polling rate
func (d *Device) UpdatePollingRate(pullingRate int) uint8 {
	if _, ok := d.PollingRates[pullingRate]; ok {
		if d.DeviceProfile == nil {
			return 0
		}
		d.Exit = true
		time.Sleep(40 * time.Millisecond)

		d.DeviceProfile.PollingRate = pullingRate
		d.saveDeviceProfile()
		buf := d.buildPollingPacket(d.DeviceProfile.PollingRate)
		_, _ = d.transfer(cmdWrite, buf)
		return 1
	}
	return 0
}

// UpdateAngleSnapping will update angle snapping mode
func (d *Device) UpdateAngleSnapping(angleSnappingMode int) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if d.DeviceProfile.AngleSnapping == angleSnappingMode {
		return 0
	}

	d.DeviceProfile.AngleSnapping = angleSnappingMode
	d.saveDeviceProfile()
	d.setAngleSnapping()
	return 1
}

// UpdateRippleControl will update ripple control
func (d *Device) UpdateRippleControl(rippleControl int) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if d.DeviceProfile.RippleControl == rippleControl {
		return 0
	}

	d.DeviceProfile.RippleControl = rippleControl
	d.saveDeviceProfile()
	d.setRippleControl()
	return 1
}

// UpdateMotionSync will update motion sync
func (d *Device) UpdateMotionSync(motionSync int) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if d.DeviceProfile.MotionSync == motionSync {
		return 0
	}

	d.DeviceProfile.MotionSync = motionSync
	d.saveDeviceProfile()
	d.setMotionSync()
	return 1
}

// UpdateLiftHeight will update lift height
func (d *Device) UpdateLiftHeight(liftHeight int) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if liftHeight < 2 || liftHeight > 6 {
		return 0
	}
	if d.DeviceProfile.LiftHeight == liftHeight {
		return 0
	}

	d.DeviceProfile.LiftHeight = liftHeight
	d.saveDeviceProfile()
	d.setLiftHeight()
	return 1
}

func (d *Device) ModifyDpi() {
	if d.DeviceProfile.Profile >= 4 {
		d.DeviceProfile.Profile = 0
	} else {
		d.DeviceProfile.Profile++
	}

	d.saveDeviceProfile()
	d.toggleDPI()
}

// sniperMode will set mouse DPI to sniper mode
func (d *Device) sniperMode(active bool) {
	d.SniperMode = active
	if active {
		for _, profile := range d.DeviceProfile.Profiles {
			if profile.Sniper {
				value := profile.Value

				// Send DPI packet
				if value < uint16(minDpiValue) {
					value = uint16(minDpiValue)
				}
				if value > uint16(maxDpiValue) {
					value = uint16(maxDpiValue)
				}
				d.DPIValue = d.DeviceProfile.Profiles[d.DeviceProfile.Profile].Value

				pf := d.DeviceProfile.Profiles[d.DeviceProfile.Profile]
				pf.Value = value
				d.DeviceProfile.Profiles[d.DeviceProfile.Profile] = pf

				d.writeDPI()
				d.toggleDPI()
			}
		}
	} else {
		pf := d.DeviceProfile.Profiles[d.DeviceProfile.Profile]
		pf.Value = d.DPIValue

		d.DeviceProfile.Profiles[d.DeviceProfile.Profile] = pf
		d.writeDPI()
		d.toggleDPI()
	}
}

// writeDPI will write DPI data to the device
func (d *Device) writeDPI() {
	if d.DeviceProfile == nil {
		return
	}

	for key, value := range d.DeviceProfile.Profiles {
		if value.Sniper {
			continue
		}

		base := 225
		step := (value.Value / 50) - 1
		buf := make([]byte, 15)
		buf[2] = byte(value.PackerIndex)
		buf[3] = cmdWriteDpi
		buf[4] = byte(step & 0xFF)
		buf[5] = buf[4]
		buf[6] = byte((step >> 8) * 0x44)
		buf[7] = byte((85 - 2*int(buf[4]) - int(buf[6])) & 0xFF)
		buf[14] = byte(base - (4 * key))
		_, err := d.transfer(cmdWrite, buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to write dpi")
		}
	}
}

// toggleDPI will change DPI mode
func (d *Device) toggleDPI() {
	if d.DeviceProfile != nil {
		if _, ok := d.DeviceProfile.Profiles[d.DeviceProfile.Profile]; ok {
			base := 85
			buf := make([]byte, 15)
			copy(buf[2:4], cmdSetDpi)
			buf[4] = byte(d.DeviceProfile.Profile)
			buf[5] = byte(base - d.DeviceProfile.Profile)
			buf[14] = 0xeb

			_, err := d.transfer(cmdWrite, buf)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to set dpi")
			}
		}
	}
}

// keepAlive will keep a device alive
func (d *Device) keepAlive() {
	buf := make([]byte, 15)
	buf[14] = 0x49
	batteryLevel, err := d.transfer(cmdRead, buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get battery level")
	}

	d.BatteryLevel = uint16(batteryLevel[6])
	stats.UpdateBatteryStats(d.Serial, d.Product, d.BatteryLevel, 1)
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

// setupKeyAssignment will setup mouse keys
func (d *Device) setupKeyAssignment() {
	if len(d.KeyAssignment) != keyAmount {
		logger.Log(logger.Fields{
			"vendorId": d.VendorId,
			"keys":     len(d.KeyAssignment),
			"expected": keyAmount,
		}).Warn("Key amount mismatch")
		return
	}

	for key, value := range d.KeyAssignment {
		// Left click (bit 1) is not configurable
		if key == 1 {
			continue
		}

		if _, ok := buttonByKey[key]; !ok {
			logger.Log(logger.Fields{"bit": key}).Warn("Unknown mouse button bit")
			continue
		}

		buf := d.buildKeyAssignmentPacket(key, value.Default)
		if _, err := d.transfer(cmdWrite, buf); err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to write key assignment")
		}
	}
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
func (d *Device) triggerKeyAssignment(value byte) {
	var bitDiff = value ^ d.ModifierIndex
	var pressedKeys = bitDiff & value
	var releasedKeys = bitDiff & ^value
	d.ModifierIndex = value

	for keys := pressedKeys | releasedKeys; keys != 0; {
		bitIdx := bits.TrailingZeros8(keys)
		mask := uint8(1) << bitIdx
		keys &^= mask

		isPressed := pressedKeys&mask != 0
		isReleased := releasedKeys&mask != 0

		val, ok := d.KeyAssignment[int(mask)]
		if !ok {
			continue
		}

		if isReleased {
			// Check if we have any queue in macro tracker. If yes, release those keys
			if len(d.MacroTracker) > 0 {
				d.releaseMacroTracker()
			}

			if !val.ActionHold {
				continue
			}
			switch val.ActionType {
			case 1, 3:
				inputmanager.InputControlKeyboardHold(val.ActionCommand, false)
			case 8:
				d.sniperMode(false)
			case 9:
				inputmanager.InputControlMouseHold(val.ActionCommand, false)
			}
		}

		if isPressed {
			switch val.ActionType {
			case 1, 3:
				if val.ActionHold {
					inputmanager.InputControlKeyboardHold(val.ActionCommand, true)
				} else {
					inputmanager.InputControlKeyboard(val.ActionCommand, false)
				}
				break
			case 2:
				d.ModifyDpi()
				break
			case 8:
				d.sniperMode(true)
				break
			case 9:
				if val.ActionHold {
					inputmanager.InputControlMouseHold(val.ActionCommand, true)
				} else {
					inputmanager.InputControlMouse(val.ActionCommand)
				}
				break
			case 10:
				macroProfile := macro.GetProfile(int(val.ActionCommand))
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
						}
					}
				}
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

// mouseListener will listen for events from mouse
func (d *Device) mouseListener() {
	if inputmanager.GetVirtualMouse() == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Warn("Virtual mouse is not available. Key AssignmentS are blocked")
		return
	}

	go func() {
		inputEvent := ""
		enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
			if info.InterfaceNbr == 2 {
				events, err := common.FindEventsByHidraw(info.Path)
				if err != nil {
					return err
				}
				if len(events) > 0 {
					inputEvent = events[0]
				}
			}
			return nil
		})

		err := hid.Enumerate(d.VendorId, d.ProductId, enum)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to enumerate devices")
		}

		if len(inputEvent) > 0 {
			f, err := os.OpenFile(inputEvent, os.O_RDONLY, 0)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to open input event listener")
				return
			}
			err = inputmanager.IocTlInt(f.Fd(), inputmanager.EvIocGrab, 1)
			if err != nil {
				_ = f.Close()
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("EvIocGrab failed")
				return
			}
			d.mouse = f
		}

		if d.mouse != nil {
			for {
				select {
				default:
					if d.Exit {
						err = d.mouse.Close()
						if err != nil {
							logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Failed to close listener")
							return
						}
						return
					}

					ev, err := inputmanager.ReadEvent(d.mouse)
					if err != nil {
						return
					}

					// Buttons
					if ev.Type == EvKey && (ev.Code == BtnBack || ev.Code == BtnForward || ev.Code == BtnMiddle || ev.Code == BtnLeft || ev.Code == BtnRight) {
						var val byte = 0
						switch ev.Code {
						case 272:
							val = 1
						case 273:
							val = 2
						case 274:
							val = 4
						case 275:
							val = 8
						case 276:
							val = 16
						}

						if ev.Value == 1 {
							d.triggerKeyAssignment(val)
						} else {
							d.triggerKeyAssignment(0)
						}
					}

					// Mouse position
					if ev.Type == 0x02 {
						if ev.Code == 0x08 {
							direction := ev.Value == 0x01
							inputmanager.InputControlScroll(direction)
						} else {
							val := map[int]int32{}

							if ev.Code == 0 {
								val[0] = ev.Value
							}

							if ev.Code == 1 {
								val[1] = ev.Value
							}
							inputmanager.InputControlMove(val[0], val[1])
						}
					}
				}
			}
		}
	}()
}

// buildKeyAssignmentPacket will build key assignment packet
func (d *Device) buildKeyAssignmentPacket(bit int, enabled bool) []byte {
	def := buttonByKey[bit]

	buf := make([]byte, 14)
	buf[2] = def.Cmd
	buf[3] = 0x04

	if enabled {
		buf[4] = 0x01
		buf[5] = byte(bit)
		buf[7] = byte(0x54 - bit)
	} else {
		buf[4] = 0x00
		buf[5] = 0x00
		buf[7] = 0x55
	}
	buf[13] = def.Tail

	return buf
}

// buildTogglePacket will build packet for toggle actions
func (d *Device) buildTogglePacket(cmd byte, enabled bool, tail byte) []byte {
	v := byte(0x00)
	if enabled {
		v = 0x01
	}
	buf := make([]byte, 14)
	buf[2] = cmd
	buf[3] = 0x02
	buf[4] = v
	buf[5] = 0x55 - v
	buf[13] = tail

	return buf
}

// buildLiftHeightPacket will build Lift Height packet
func (d *Device) buildLiftHeightPacket(LiftHeight int) []byte {
	val := byte(0x00)
	switch LiftHeight {
	case 1:
		val = 0x00
	case 2:
		val = 0x01
	case 3:
		val = 0x03
	}

	buf := make([]byte, 14)
	buf[2] = 0x0A
	buf[3] = 0x02
	buf[4] = val
	buf[5] = 0x55 - val
	buf[13] = 0xE5

	return buf
}

// buildSleepPacket will build sleep packet
func (d *Device) buildSleepPacket(minutes int) []byte {
	val := byte(minutes * 6)
	buf := make([]byte, 14)

	buf[2] = 0xAD
	buf[3] = 0x02
	buf[4] = val
	buf[5] = 0x55 - val
	buf[13] = 0x42

	return buf
}

// buildPollingPacket will create polling rate packet
func (d *Device) buildPollingPacket(pollingRate int) []byte {
	buf := make([]byte, 14)
	val := byte(0x00)
	switch pollingRate {
	case 1:
		val = 0x01
	case 2:
		val = 0x10
	case 3:
		val = 0x20
	case 4:
		val = 0x40
	}
	buf[2] = 0x00
	buf[3] = 0x02
	buf[4] = val
	buf[5] = 0x55 - val
	buf[13] = 0xEF

	return buf
}
