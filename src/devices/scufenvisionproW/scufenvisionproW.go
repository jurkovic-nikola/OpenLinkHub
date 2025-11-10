package scufenvisionproW

// Package: SCUF Gaming SCUF Envision Pro Controller
// This is the primary package for SCUF Gaming SCUF Envision Pro Controller.
// All device actions are controlled from this package.
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
	"OpenLinkHub/src/temperatures"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"math/bits"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type ZoneColors struct {
	Color      *rgb.Color
	ColorIndex []int
	Name       string
}

type AnalogData struct {
	DeadZoneMin uint8
	DeadZoneMax uint8
	Points      map[int]common.CurveData `json:"points"`
}

type DeviceProfile struct {
	Active                      bool
	Path                        string
	Product                     string
	Serial                      string
	Brightness                  uint8
	RGBProfile                  string
	BrightnessSlider            uint8
	OriginalBrightness          uint8
	Label                       string
	ZoneColors                  map[int]ZoneColors
	SleepMode                   int
	LeftVibrationValue          uint8
	RightVibrationValue         uint8
	KeyAssignmentHash           string
	LeftThumbStickMode          uint8
	LeftThumbStickSensitivityX  uint8
	LeftThumbStickSensitivityY  uint8
	LeftThumbStickInvertY       bool
	RightThumbStickMode         uint8
	RightThumbStickSensitivityX uint8
	RightThumbStickSensitivityY uint8
	RightThumbStickInvertY      bool
	AnalogData                  map[int]AnalogData
}

type Device struct {
	Debug                 bool
	dev                   *hid.Device
	analogListener        *hid.Device
	listener              *hid.Device
	Manufacturer          string `json:"manufacturer"`
	Product               string `json:"product"`
	Serial                string `json:"serial"`
	Firmware              string `json:"firmware"`
	activeRgb             *rgb.ActiveRGB
	UserProfiles          map[string]*DeviceProfile `json:"userProfiles"`
	Devices               map[int]string            `json:"devices"`
	DeviceProfile         *DeviceProfile
	OriginalProfile       *DeviceProfile
	Template              string
	VendorId              uint16
	ProductId             uint16
	SlipstreamId          uint16
	Brightness            map[int]string
	KeyAssignmentTypes    map[int]string
	ThumbStickModes       map[int]string
	LEDChannels           int
	ChangeableLedChannels int
	CpuTemp               float32
	GpuTemp               float32
	Layouts               []string
	Rgb                   *rgb.RGB
	rgbMutex              sync.RWMutex
	Endpoint              byte
	SleepModes            map[int]string
	Connected             bool
	mutex                 sync.Mutex
	deviceLock            sync.Mutex
	timerKeepAlive        *time.Ticker
	keepAliveChan         chan struct{}
	timer                 *time.Ticker
	autoRefreshChan       chan struct{}
	Exit                  bool
	KeyAssignment         map[int]inputmanager.KeyAssignment
	InputActions          map[uint16]inputmanager.InputAction
	PressLoop             bool
	keyAssignmentFile     string
	BatteryLevel          uint16
	KeyAssignmentData     *inputmanager.KeyAssignment
	ModifierIndex         uint32
	SniperMode            bool
	MacroTracker          map[int]uint16
	RGBModes              []string
	instance              *common.Device
}

var (
	pwd                          = ""
	cmdSoftwareMode              = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode              = []byte{0x01, 0x03, 0x00, 0x01}
	cmdSleepMode                 = []byte{0x01, 0x03, 0x00, 0x04}
	cmdGetFirmware               = []byte{0x02, 0x13}
	cmdInitWrite                 = []byte{0x01}
	cmdLeftVibrationModule       = []byte{0x84, 0x00}
	cmdRightVibrationModule      = []byte{0x85, 0x00}
	cmdWriteColor                = []byte{0x06, 0x00}
	cmdOpenEndpoint              = []byte{0x0d, 0x00, 0x01}
	cmdHeartbeat                 = []byte{0x12}
	cmdBatteryLevel              = []byte{0x02, 0x0f}
	cmdCloseEndpoint             = []byte{0x05, 0x01, 0x01}
	cmdOpenKeyAssignmentEndpoint = []byte{0x0d, 0x01, 0x02}
	cmdOpenAnalogDataEndpoint    = []byte{0x0d, 0x01, 0x2b}
	cmdOpenWriteEndpoint         = []byte{0x0d, 0x00, 0x01}
	cmdBeginWrite                = []byte{0x09, 0x01}
	cmdWrite                     = []byte{0x06, 0x01}
	cmdWriteNext                 = []byte{0x07, 0x01}
	cmdSleep                     = []byte{0x0e, 0x00}
	cmdEcoMode                   = []byte{0x0b, 0x00, 0x00}
	cmdDeadZones                 = map[int]map[int][]byte{
		0: {
			0: {0x7c, 0x00},
			1: {0xdd, 0x00},
		},
		1: {
			0: {0x7d, 0x00},
			1: {0xde, 0x00},
		},
		2: {
			0: {0x7a, 0x00},
			1: {0xdb, 0x00},
		},
		3: {
			0: {0x7b, 0x00},
			1: {0xdc, 0x00},
		},
	}
	cmdInitDeadZones = map[int][]byte{
		0: {0x80, 0x00},
		1: {0x81, 0x00},
		2: {0x7e, 0x00},
		3: {0x7f, 0x00},
	}
	bufferSize              = 64
	bufferSizeWrite         = bufferSize + 1
	headerSize              = 3
	headerWriteSize         = 4
	keyAmount               = 30
	keyAmountLen            = 32
	deviceRefreshInterval   = 1000
	scufVendorId            = uint16(11925)
	maxBufferSizePerRequest = 60
	rgbModes                = []string{
		"colorpulse",
		"colorshift",
		"colorwarp",
		"cpu-temperature",
		"flickering",
		"gpu-temperature",
		"controller",
		"off",
		"rainbow",
		"rotator",
		"static",
		"storm",
		"watercolor",
		"wave",
	}
)

func Init(_, slipstreamId, productId uint16, dev *hid.Device, endpoint byte, serial string) *Device {
	pwd = config.GetConfig().ConfigPath

	// Init new struct with HID device
	d := &Device{
		dev:          dev,
		Template:     "scufenvisionproW.html",
		VendorId:     scufVendorId,
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
		Product: "SCUF ENVISION PRO",
		SleepModes: map[int]string{
			1:  "1 minute",
			5:  "5 minutes",
			10: "10 minutes",
			15: "15 minutes",
			30: "30 minutes",
			60: "1 hour",
		},
		RGBModes:              rgbModes,
		LEDChannels:           9,
		ChangeableLedChannels: 9,
		autoRefreshChan:       make(chan struct{}),
		timer:                 &time.Ticker{},
		keepAliveChan:         make(chan struct{}),
		timerKeepAlive:        &time.Ticker{},
		KeyAssignmentTypes: map[int]string{
			0:  "None",
			1:  "Media Keys",
			2:  "DPI",
			3:  "Keyboard",
			8:  "Sniper",
			9:  "Mouse",
			10: "Macro",
		},
		ThumbStickModes: map[int]string{
			0: "None",
			1: "Mouse",
		},
		InputActions:      inputmanager.GetInputActions(),
		keyAssignmentFile: "/database/key-assignments/scufenvisionpro.json",
		MacroTracker:      make(map[int]uint16),
	}

	d.getDebugMode()       // Debug mode
	d.loadRgb()            // Load RGB
	d.loadDeviceProfiles() // Load all device profiles
	d.saveDeviceProfile()  // Save profile
	d.setAutoRefresh()     // Set auto device refresh
	d.loadKeyAssignments() // Key Assignments
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d
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
		d.getDeviceFirmware()        // Firmware
		d.setSoftwareMode()          // Activate software mode
		d.getBatterLevel()           // Battery level
		d.initLeds()                 // Init LED ports
		d.setDeviceColor()           // Device color
		d.setVibrationModuleValues() // Vibration module
		d.setupAnalogDevices()       // Analog devices
		d.setupKeyAssignment()       // Setup key assignments
		d.setSleepTimer()            // Sleep timer
		d.setEcoMode()               // Eco mode Off
		d.setAnalogDevice()          // Analog device
		d.analogDataListener()       // Analog listener
	}
}

// GetRgbProfiles will return RGB profiles for a target device
func (d *Device) GetRgbProfiles() interface{} {
	return d.Rgb
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

	d.timerKeepAlive.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.keepAliveChan != nil {
				close(d.keepAliveChan)
			}
		})
	}()

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
	return 2
}

// ProcessControllerEmulation will process controller emulation
func (d *Device) ProcessControllerEmulation(module, mode, sensitivityX, sensitivityY uint8, invertYAxis bool) uint8 {
	if d.DeviceProfile != nil {
		if mode < 0 || mode > 2 {
			return 0
		}

		if sensitivityX < 5 || sensitivityX > 50 {
			return 0
		}

		if sensitivityY < 5 || sensitivityY > 50 {
			return 0
		}

		switch module {
		case 0:
			d.DeviceProfile.LeftThumbStickMode = mode
			d.DeviceProfile.LeftThumbStickSensitivityX = sensitivityX
			d.DeviceProfile.LeftThumbStickSensitivityY = sensitivityY
			d.DeviceProfile.LeftThumbStickInvertY = invertYAxis
			break
		case 1:
			d.DeviceProfile.RightThumbStickMode = mode
			d.DeviceProfile.RightThumbStickSensitivityX = sensitivityX
			d.DeviceProfile.RightThumbStickSensitivityY = sensitivityY
			d.DeviceProfile.RightThumbStickInvertY = invertYAxis
			break
		}

		d.saveDeviceProfile()
		return 1
	}
	return 0
}

// ProcessControllerVibration will update left or right vibration module
func (d *Device) ProcessControllerVibration(module, value uint8) uint8 {
	if d.DeviceProfile != nil {
		if value < 0 || value > 100 {
			value = 50
		}

		switch module {
		case 0:
			d.DeviceProfile.LeftVibrationValue = value
			break
		case 1:
			d.DeviceProfile.RightVibrationValue = value
			break
		}
		d.saveDeviceProfile()
		d.setVibrationModuleValues()
		d.triggerHapticEngine()
		return 1
	}
	return 0
}

// SaveControllerZoneColors will save controller zone colors
func (d *Device) SaveControllerZoneColors(zoneColors map[int]rgb.Color) uint8 {
	i := 0
	if d.DeviceProfile == nil {
		return 0
	}

	// Zone Colors
	for key, zone := range zoneColors {
		if zone.Red > 255 ||
			zone.Green > 255 ||
			zone.Blue > 255 ||
			zone.Red < 0 ||
			zone.Green < 0 ||
			zone.Blue < 0 {
			continue
		}
		if zoneColor, ok := d.DeviceProfile.ZoneColors[key]; ok {
			zoneColor.Color.Red = zone.Red
			zoneColor.Color.Green = zone.Green
			zoneColor.Color.Blue = zone.Blue
			zoneColor.Color.Hex = fmt.Sprintf("#%02x%02x%02x", int(zone.Red), int(zone.Green), int(zone.Blue))
			d.DeviceProfile.ZoneColors[key] = zoneColor
		}
		i++
	}

	if i > 0 {
		d.saveDeviceProfile()
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true // Exit current RGB mode
			d.activeRgb = nil
		}
		d.setDeviceColor() // Restart RGB
		return 1
	}
	return 0
}

// UpdateDeviceKeyAssignment will update device key assignments
func (d *Device) UpdateDeviceKeyAssignment(keyIndex int, keyAssignment inputmanager.KeyAssignment) uint8 {
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

	d.DeviceProfile.BrightnessSlider = value
	d.saveDeviceProfile()
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
	return 1
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

// ProcessGetControllerGraph will return analog data
func (d *Device) ProcessGetControllerGraph() interface{} {
	if d.DeviceProfile == nil {
		return nil
	}

	if d.DeviceProfile.AnalogData == nil {
		return nil
	}

	data := make(map[int][]common.CurveData, 4)
	for i := 0; i < len(d.DeviceProfile.AnalogData); i++ {
		var points []common.CurveData
		for m := 0; m < len(d.DeviceProfile.AnalogData[i].Points); m++ {
			points = append(points, common.CurveData{
				X: d.DeviceProfile.AnalogData[i].Points[m].X,
				Y: d.DeviceProfile.AnalogData[i].Points[m].Y,
			})
		}
		data[i] = points
	}
	return data
}

// ProcessSetControllerGraph will set analog data
func (d *Device) ProcessSetControllerGraph(analogDevice int, deadZoneMin, deadZoneMax uint8, curveData []common.CurveData) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if d.DeviceProfile.AnalogData == nil {
		return 0
	}

	if device, ok := d.DeviceProfile.AnalogData[analogDevice]; ok {
		device.DeadZoneMin = deadZoneMin
		device.DeadZoneMax = deadZoneMax
		for i := 0; i < len(curveData); i++ {
			device.Points[i] = curveData[i]
		}
		d.DeviceProfile.AnalogData[analogDevice] = device
		d.saveDeviceProfile()
		d.setupAnalogDevices()
	}
	return 1
}

// setupAnalogDevices will set up analog devices and their dead zones
func (d *Device) setupAnalogDevices() {
	if d.DeviceProfile == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Error("d.DeviceProfile is null")
		return
	}

	if d.DeviceProfile.AnalogData == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Error("d.DeviceProfile.AnalogData is null")
		return
	}

	// Dead zones
	for i := 0; i < len(cmdInitDeadZones); i++ {
		analogData := d.DeviceProfile.AnalogData[i]
		deadZoneMin := analogData.DeadZoneMin
		deadZoneMax := analogData.DeadZoneMax

		if deadZoneMin < 0 || deadZoneMin > 15 {
			deadZoneMin = 5
		}

		if deadZoneMax < 0 || deadZoneMax > 15 {
			deadZoneMax = 5
		}

		// Init endpoint
		_, err := d.transfer(cmdInitWrite, cmdInitDeadZones[i])
		if err != nil {
			logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Failed to init left thumbstick endpoint")
			return
		}

		// Write min / max
		for m := 0; m < len(cmdDeadZones[i]); m++ {
			buf := make([]byte, 3)
			copy(buf[0:1], cmdDeadZones[i][m])
			buf[2] = deadZoneMin
			if m == 1 {
				buf[2] = deadZoneMax
			}
			_, err = d.transfer(cmdInitWrite, buf)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Failed to set left thumbstick dead zone")
				return
			}

		}
	}

	// Analog curve data
	buf := make([]byte, 59)
	buf[0] = 0x26
	buf[1] = 0x00
	buf[2] = byte(len(cmdInitDeadZones))
	var curveData []byte

	for i := 0; i < len(cmdInitDeadZones); i++ {
		curveData = append(curveData, byte(i))
		curveData = append(curveData, byte(len(d.DeviceProfile.AnalogData[i].Points)))
		for m := 0; m < len(d.DeviceProfile.AnalogData[i].Points); m++ {
			curveData = append(curveData, d.DeviceProfile.AnalogData[i].Points[m].X)
			curveData = append(curveData, d.DeviceProfile.AnalogData[i].Points[m].Y)
		}
	}
	copy(buf[3:], curveData)
	d.writeAnalogData(buf)
}

// setAnalogDevice will open analog device
func (d *Device) setAnalogDevice() {
	enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
		if info.InterfaceNbr == 3 {
			listener, err := hid.OpenPath(info.Path)
			if err != nil {
				return err
			}
			d.analogListener = listener
		}
		return nil
	})

	err := hid.Enumerate(scufVendorId, d.SlipstreamId, enum)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to enumerate devices")
		return
	}
}

// saveKeyAssignments will save key mappings
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
			2147483648: {
				Name:          "Profile",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   31,
			},
			1073741824: {
				Name:          "G5",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   30,
			},
			536870912: {
				Name:          "G4",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   29,
			},
			268435456: {
				Name:          "G3",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   28,
			},
			134217728: {
				Name:          "G2",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   27,
			},
			67108864: {
				Name:          "G1",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   26,
			},
			16777216: {
				Name:          "Power",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   24,
			},
			8388608: {
				Name:          "S2",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   23,
			},
			4194304: {
				Name:          "S1",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   22,
			},
			2097152: {
				Name:          "P4",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   21,
			},
			1048576: {
				Name:          "P3",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   20,
			},
			524288: {
				Name:          "P2",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   19,
			},
			262144: {
				Name:          "P1",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   18,
			},
			131072: {
				Name:          "MENU",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   17,
			},
			65536: {
				Name:          "LOCK",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   16,
			},
			16384: {
				Name:          "RS (Press)",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   14,
			},
			8192: {
				Name:          "LS (Press)",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   13,
			},
			4096: {
				Name:          "RT",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   12,
			},
			2048: {
				Name:          "LT",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   11,
			},
			1024: {
				Name:          "RB",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   10,
			},
			512: {
				Name:          "LB",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   9,
			},
			256: {
				Name:          "B",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   8,
			},
			128: {
				Name:          "Y",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   7,
			},
			64: {
				Name:          "X",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   6,
			},
			32: {
				Name:          "A",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   5,
			},
			16: {
				Name:          "DPAD Right",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   4,
			},
			8: {
				Name:          "DPAD Left",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   3,
			},
			4: {
				Name:          "DPAD Down",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   2,
			},
			2: {
				Name:          "DPAD Up",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ButtonIndex:   1,
			},
			1: {
				Name:          "Left Button",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
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

// getManufacturer will return device manufacturer
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
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

// triggerHapticEngine will trigger vibration motors
func (d *Device) triggerHapticEngine() {
	go func() {
		buf := make([]byte, 13)
		buf[0] = 0x09
		buf[1] = 0x00
		buf[2] = 0x6a
		buf[3] = 0x09
		buf[4] = 0x00
		buf[5] = 0x03
		buf[6] = 0x00
		buf[7] = 0x00
		buf[8] = 0xff // Left Haptic Engine On
		buf[9] = 0xff // Right Haptic Engine On
		buf[10] = 0x10
		buf[11] = 0x00
		buf[12] = 0xeb

		if d.analogListener != nil {
			_, err := d.analogListener.Write(buf)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to write to haptic device")
				return
			}

			time.Sleep(1 * time.Second)
			buf[8] = 0x00 // Left Haptic Engine Off
			buf[9] = 0x00 // Right Haptic Engine Off
			_, err = d.analogListener.Write(buf)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to write to haptic device")
				return
			}
		}
	}()
}

// setEcoMode will set device eco mode
func (d *Device) setEcoMode() {
	_, err := d.transfer(cmdInitWrite, cmdEcoMode)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change device sleep timer")
	}
}

// setSleepTimer will set device sleep timer
func (d *Device) setSleepTimer() uint8 {
	if d.DeviceProfile != nil {
		if d.DeviceProfile.SleepMode == 0 {
			sleepCmd := cmdOpenWriteEndpoint
			sleepCmd[2] = 0x00
			_, err := d.transfer(cmdInitWrite, sleepCmd)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change device sleep timer")
				return 0
			}
			return 1
		} else {
			sleepCmd := cmdOpenWriteEndpoint
			sleepCmd[2] = 0x01
			_, err := d.transfer(cmdInitWrite, sleepCmd)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change device sleep timer")
				return 0
			}

			buf := make([]byte, 6)
			copy(buf[0:1], cmdSleep)
			sleep := d.DeviceProfile.SleepMode * (60 * 1000)
			binary.LittleEndian.PutUint32(buf[2:], uint32(sleep))

			_, err = d.transfer(cmdInitWrite, buf)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change device sleep timer")
				return 0
			}

			_, err = d.transfer([]byte{0x02, 0x40}, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change device sleep timer")
				return 0
			}

			_, err = d.transfer(cmdSleepMode, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change device sleep timer")
				return 0
			}
			return 1
		}
	}
	return 0
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

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	_, err := d.transfer(cmdSoftwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
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
	d.BatteryLevel = binary.LittleEndian.Uint16(batteryLevel[4:6]) / 10
	stats.UpdateBatteryStats(d.Serial, d.Product, d.BatteryLevel, 3)
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	fw, err := d.transfer(
		cmdGetFirmware,
		nil,
	)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
	}
	v1, v2, v3 := int(fw[4]), int(fw[5]), int(fw[6])
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
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

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"

	deviceProfile := &DeviceProfile{
		Product:            d.Product,
		Serial:             d.Serial,
		Path:               profilePath,
		BrightnessSlider:   100,
		OriginalBrightness: 100,
	}

	// First save, assign saved profile to a device
	if d.DeviceProfile == nil {
		// RGB, Label
		deviceProfile.RGBProfile = "controller"
		deviceProfile.Label = "Controller"
		deviceProfile.Active = true
		deviceProfile.ZoneColors = map[int]ZoneColors{
			0: { // Controller
				ColorIndex: []int{0, 9, 18},
				Color: &rgb.Color{
					Red:        0,
					Green:      255,
					Blue:       255,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 0, 255, 255),
				},
				Name: "Controller",
			},
		}
		deviceProfile.SleepMode = 15
		deviceProfile.LeftVibrationValue = 100
		deviceProfile.RightVibrationValue = 100
		deviceProfile.LeftThumbStickSensitivityX = 5
		deviceProfile.LeftThumbStickSensitivityY = 5
		deviceProfile.RightThumbStickSensitivityX = 5
		deviceProfile.RightThumbStickSensitivityY = 5
		deviceProfile.AnalogData = map[int]AnalogData{
			0: { // Left Thumbstick
				DeadZoneMin: 5,
				DeadZoneMax: 5,
				Points: map[int]common.CurveData{
					0: {
						X: 0,
						Y: 0,
					},
					1: {
						X: 20,
						Y: 20,
					},
					2: {
						X: 40,
						Y: 40,
					},
					3: {
						X: 60,
						Y: 60,
					},
					4: {
						X: 80,
						Y: 80,
					},
					5: {
						X: 100,
						Y: 100,
					},
				},
			},
			1: { // Right Thumbstick
				DeadZoneMin: 5,
				DeadZoneMax: 5,
				Points: map[int]common.CurveData{
					0: {
						X: 0,
						Y: 0,
					},
					1: {
						X: 20,
						Y: 20,
					},
					2: {
						X: 40,
						Y: 40,
					},
					3: {
						X: 60,
						Y: 60,
					},
					4: {
						X: 80,
						Y: 80,
					},
					5: {
						X: 100,
						Y: 100,
					},
				},
			},
			2: { // Left Trigger
				DeadZoneMin: 2,
				DeadZoneMax: 2,
				Points: map[int]common.CurveData{
					0: {
						X: 0,
						Y: 0,
					},
					1: {
						X: 20,
						Y: 20,
					},
					2: {
						X: 40,
						Y: 40,
					},
					3: {
						X: 60,
						Y: 60,
					},
					4: {
						X: 80,
						Y: 80,
					},
					5: {
						X: 100,
						Y: 100,
					},
				},
			},
			3: { // Right Trigger
				DeadZoneMin: 2,
				DeadZoneMax: 2,
				Points: map[int]common.CurveData{
					0: {
						X: 0,
						Y: 0,
					},
					1: {
						X: 20,
						Y: 20,
					},
					2: {
						X: 40,
						Y: 40,
					},
					3: {
						X: 60,
						Y: 60,
					},
					4: {
						X: 80,
						Y: 80,
					},
					5: {
						X: 100,
						Y: 100,
					},
				},
			},
		}
	} else {
		deviceProfile.BrightnessSlider = d.DeviceProfile.BrightnessSlider
		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.Brightness = d.DeviceProfile.Brightness
		deviceProfile.OriginalBrightness = d.DeviceProfile.OriginalBrightness
		deviceProfile.RGBProfile = d.DeviceProfile.RGBProfile
		deviceProfile.Label = d.DeviceProfile.Label
		deviceProfile.ZoneColors = d.DeviceProfile.ZoneColors
		deviceProfile.SleepMode = d.DeviceProfile.SleepMode
		deviceProfile.Path = d.DeviceProfile.Path
		deviceProfile.LeftVibrationValue = d.DeviceProfile.LeftVibrationValue
		deviceProfile.RightVibrationValue = d.DeviceProfile.RightVibrationValue
		deviceProfile.LeftThumbStickMode = d.DeviceProfile.LeftThumbStickMode
		deviceProfile.LeftThumbStickSensitivityX = d.DeviceProfile.LeftThumbStickSensitivityX
		deviceProfile.LeftThumbStickSensitivityY = d.DeviceProfile.LeftThumbStickSensitivityY
		deviceProfile.RightThumbStickMode = d.DeviceProfile.RightThumbStickMode
		deviceProfile.RightThumbStickSensitivityX = d.DeviceProfile.RightThumbStickSensitivityX
		deviceProfile.RightThumbStickSensitivityY = d.DeviceProfile.RightThumbStickSensitivityY
		deviceProfile.LeftThumbStickInvertY = d.DeviceProfile.LeftThumbStickInvertY
		deviceProfile.RightThumbStickInvertY = d.DeviceProfile.RightThumbStickInvertY
		deviceProfile.AnalogData = d.DeviceProfile.AnalogData
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

// setVibrationModuleValues will set left and right vibration modules values
func (d *Device) setVibrationModuleValues() {
	if d.DeviceProfile == nil {
		return
	}

	if d.DeviceProfile.LeftVibrationValue < 0 || d.DeviceProfile.LeftVibrationValue > 100 {
		d.DeviceProfile.LeftVibrationValue = 50
	}

	if d.DeviceProfile.RightVibrationValue < 0 || d.DeviceProfile.RightVibrationValue > 100 {
		d.DeviceProfile.RightVibrationValue = 50
	}

	// Left module
	buf := make([]byte, 3)
	buf[0] = cmdLeftVibrationModule[0]          // Left module
	buf[1] = cmdLeftVibrationModule[1]          // Header
	buf[2] = d.DeviceProfile.LeftVibrationValue // Value
	_, err := d.transfer(cmdInitWrite, buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to set left vibration module")
		return
	}

	// Right module
	buf[0] = cmdRightVibrationModule[0]          // Right module
	buf[1] = cmdRightVibrationModule[1]          // Header
	buf[2] = d.DeviceProfile.RightVibrationValue // Value
	_, err = d.transfer(cmdInitWrite, buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to set right vibration module")
	}
}

// keepAlive will keep a device alive
func (d *Device) keepAlive() {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if d.Exit {
		return
	}
	_, err := d.transfer(cmdHeartbeat, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write heartbeat to a device")
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
			case <-d.autoRefreshChan:
				d.timer.Stop()
				return
			}
		}
	}()
}

// initLeds will initialize LED endpoint
func (d *Device) initLeds() {
	_, err := d.transfer(cmdOpenEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to initialize device color endpoint")
	}
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	buf := make([]byte, d.LEDChannels*3)
	if d.DeviceProfile == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to set color. DeviceProfile is null!")
		return
	}

	if d.DeviceProfile.RGBProfile == "controller" {
		for _, zoneColor := range d.DeviceProfile.ZoneColors {
			zoneColor.Color.Brightness = rgb.GetBrightnessValueFloat(d.DeviceProfile.BrightnessSlider)
			zoneColor.Color = rgb.ModifyBrightness(*zoneColor.Color)
			zoneColorIndexRange := zoneColor.ColorIndex
			for key, zoneColorIndex := range zoneColorIndexRange {
				switch key {
				case 0: // Red
					for i := zoneColorIndex; i < 9; i++ {
						buf[i] = byte(zoneColor.Color.Red)
					}
				case 1: // Green
					for i := zoneColorIndex; i < 9*2; i++ {
						buf[i] = byte(zoneColor.Color.Green)
					}
				case 2: // Blue
					for i := zoneColorIndex; i < 9*3; i++ {
						buf[i] = byte(zoneColor.Color.Blue)
					}
				}
			}
		}
		d.writeColor(buf)
		return
	}

	if d.DeviceProfile.RGBProfile == "static" {
		profile := d.GetRgbProfile("static")
		if profile == nil {
			return
		}

		profile.StartColor.Brightness = rgb.GetBrightnessValueFloat(d.DeviceProfile.BrightnessSlider)
		profileColor := rgb.ModifyBrightness(profile.StartColor)
		for _, zoneColor := range d.DeviceProfile.ZoneColors {
			zoneColorIndexRange := zoneColor.ColorIndex
			for key, zoneColorIndex := range zoneColorIndexRange {
				switch key {
				case 0: // Red
					for i := zoneColorIndex; i < 9; i++ {
						buf[i] = byte(profileColor.Red)
					}
				case 1: // Green
					for i := zoneColorIndex; i < 9*2; i++ {
						buf[i] = byte(profileColor.Green)
					}
				case 2: // Blue
					for i := zoneColorIndex; i < 9*3; i++ {
						buf[i] = byte(profileColor.Blue)
					}
				}
			}
		}
		d.writeColor(buf)
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
					for i := 0; i < d.ChangeableLedChannels*3; i++ {
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
					d.ChangeableLedChannels,
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
				r.RGBBrightness = rgb.GetBrightnessValueFloat(d.DeviceProfile.BrightnessSlider)
				r.RGBStartColor.Brightness = r.RGBBrightness
				r.RGBEndColor.Brightness = r.RGBBrightness

				switch d.DeviceProfile.RGBProfile {
				case "off":
					{
						for n := 0; n < d.ChangeableLedChannels; n++ {
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

				for _, zoneColor := range d.DeviceProfile.ZoneColors {
					zoneColorIndexRange := zoneColor.ColorIndex

					for key, zoneColorIndex := range zoneColorIndexRange {
						switch key {
						case 0: // Red
							for i := zoneColorIndex; i < 9; i++ {
								buf[i] = buff[i]
							}
						case 1: // Green
							for i := zoneColorIndex; i < 9*2; i++ {
								buf[i] = buff[i]
							}
						case 2: // Blue
							for i := zoneColorIndex; i < 9*3; i++ {
								buf[i] = buff[i]
							}
						}
					}
				}

				d.writeColor(buf)
				time.Sleep(40 * time.Millisecond)
			}
		}
	}(d.ChangeableLedChannels)
}

// writeColor will write data to the device with a specific endpoint.
func (d *Device) writeColor(data []byte) {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if d.Exit {
		return
	}

	buffer := make([]byte, len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	copy(buffer[headerWriteSize:], data)
	_, err := d.transfer(cmdWriteColor, buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
	}
}

// writeKeyAssignmentData will write key assignment to the device.
func (d *Device) writeKeyAssignmentData(data []byte) {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if d.Exit {
		return
	}

	// Open endpoint
	_, err := d.transfer(cmdOpenKeyAssignmentEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to open write endpoint")
		return
	}

	// Send data
	buffer := make([]byte, len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	copy(buffer[headerWriteSize:], data)
	_, err = d.transfer(cmdWrite, buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to data endpoint")
	}

	// Close endpoint
	_, err = d.transfer(cmdCloseEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to close endpoint")
		return
	}
}

// writeAnalogData will write analog data to the device.
func (d *Device) writeAnalogData(data []byte) {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if d.Exit {
		return
	}

	// Open endpoint
	_, err := d.transfer(cmdOpenAnalogDataEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to open write endpoint")
		return
	}

	// Begin write
	_, err = d.transfer(cmdBeginWrite, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to open write endpoint")
		return
	}

	// Write data
	buffer := make([]byte, len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	copy(buffer[headerWriteSize:], data)

	// Split packet into chunks
	chunks := common.ProcessMultiChunkPacket(buffer, maxBufferSizePerRequest)
	for i, chunk := range chunks {
		if i == 0 {
			_, err := d.transfer(cmdWrite, chunk)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to analog endpoint")
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
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to close endpoint")
		return
	}
}

// setupKeyAssignment will setup mouse keys
func (d *Device) setupKeyAssignment() {
	// Prevent modifications if key amount does not match the expected key amount
	definedKeyAmount := len(d.KeyAssignment)
	if definedKeyAmount < keyAmount || definedKeyAmount > keyAmount {
		logger.Log(logger.Fields{"vendorId": d.VendorId, "keys": definedKeyAmount, "expected": keyAmount}).Warn("Expected key amount does not match the expected key amount.")
		return
	}

	keys := make([]int, 0)
	for k := range d.KeyAssignment {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	buf := make([]byte, keyAmountLen)
	for i := range buf {
		buf[i] = 0x01
	}

	for _, k := range keys {
		value := d.KeyAssignment[k]
		if value.Default {
			buf[value.ButtonIndex] = byte(1)
		} else {
			buf[value.ButtonIndex] = byte(0)
		}
	}
	d.writeKeyAssignmentData(buf)
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint, buffer []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = 0x02
	bufferW[2] = d.Endpoint
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

// getListenerData will listen for keyboard events and return data on success or nil on failure.
// ReadWithTimeout is mandatory due to the nature of listening for events
func (d *Device) getAnalogData() []byte {
	data := make([]byte, bufferSize)
	if d.analogListener != nil {
		n, err := d.analogListener.ReadWithTimeout(data, 100*time.Millisecond)
		if err != nil || n == 0 {
			return nil
		}
	}
	return data
}

func (d *Device) decodeStick(data []byte) (int16, int16) {
	if len(data) < 4 {
		return 0, 0
	}

	// little endian signed 16-bit
	x := int16(uint16(data[0]) | uint16(data[1])<<8)
	y := int16(uint16(data[2]) | uint16(data[3])<<8)
	return x, y
}

func (d *Device) stickToMoveData(module uint8, xRaw, yRaw int16) (int32, int32) {
	var scaleX = float32(0.00001)
	var scaleY = float32(0.00001)

	switch module {
	case 0:
		scaleX = scaleX * float32(d.DeviceProfile.LeftThumbStickSensitivityX)
		scaleY = scaleY * float32(d.DeviceProfile.LeftThumbStickSensitivityY)
		break
	case 1:
		scaleX = scaleX * float32(d.DeviceProfile.RightThumbStickSensitivityX)
		scaleY = scaleY * float32(d.DeviceProfile.RightThumbStickSensitivityY)
		break
	}
	dx := int32(float32(xRaw) * scaleX)
	dy := int32(float32(yRaw) * scaleY)
	return dx, dy
}

func (d *Device) handleAnalogData(module uint8, data []byte) {
	xRaw, yRaw := d.decodeStick(data)
	dx, dy := d.stickToMoveData(module, xRaw, yRaw)

	if dx != 0 || dy != 0 {
		switch module {
		case 0:
			switch d.DeviceProfile.LeftThumbStickMode {
			case 1:
				if d.DeviceProfile.LeftThumbStickInvertY {
					dy = -dy
				}
				inputmanager.InputControlMove(dx, dy)
				break
			}
			break
		case 1:
			switch d.DeviceProfile.RightThumbStickMode {
			case 1:
				if d.DeviceProfile.RightThumbStickInvertY {
					dy = -dy
				}
				inputmanager.InputControlMove(dx, dy)
				break
			}
			break
		}
	}
}

// analogListener will listen for analog events from the device
func (d *Device) analogDataListener() {
	go func() {
		for {
			select {
			default:
				if d.Exit {
					if d.analogListener != nil {
						err := d.analogListener.Close()
						if err != nil {
							logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Failed to close listener")
							return
						}
					}
					return
				}

				data := d.getAnalogData()
				if len(data) == 0 || data == nil {
					continue
				}

				// Left thumbstick
				if data[1] > 0x00 || data[2] > 0x00 || data[3] > 0x00 || data[4] > 0x00 {
					d.handleAnalogData(0, data[1:5])
				}

				// Right thumbstick
				if data[5] > 0x00 || data[6] > 0x00 || data[7] > 0x00 || data[8] > 0x00 {
					d.handleAnalogData(1, data[5:9])
				}
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

// TriggerKeyAssignment will trigger key assignment if defined
func (d *Device) TriggerKeyAssignment(value uint32) {
	var bitDiff = value ^ d.ModifierIndex
	var pressedKeys = bitDiff & value
	var releasedKeys = bitDiff & ^value
	d.ModifierIndex = value

	for keys := pressedKeys | releasedKeys; keys != 0; {
		bitIdx := bits.TrailingZeros32(keys)
		mask := uint32(1) << bitIdx
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

			if val.Default || !val.ActionHold {
				continue
			}
			switch val.ActionType {
			case 1, 3:
				inputmanager.InputControlKeyboardHold(val.ActionCommand, false)
			case 9:
				inputmanager.InputControlMouseHold(val.ActionCommand, false)
			}
		}

		if isPressed {
			if val.Default {
				continue
			}

			switch val.ActionType {
			case 1, 3:
				if val.ActionHold {
					inputmanager.InputControlKeyboardHold(val.ActionCommand, true)
				} else {
					inputmanager.InputControlKeyboard(val.ActionCommand, false)
				}
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
							inputmanager.InputControlKeyboard(v.ActionCommand, v.ActionHold)
						case 9:
							inputmanager.InputControlMouse(v.ActionCommand)
						case 5:
							if v.ActionDelay > 0 {
								time.Sleep(time.Duration(v.ActionDelay) * time.Millisecond)
							}
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
						}
					}
				}
				break
			}
		}
	}
}

// ModifyBatteryLevel will modify battery level
func (d *Device) ModifyBatteryLevel(batteryLevel uint16) {
	d.BatteryLevel = batteryLevel
	stats.UpdateBatteryStats(d.Serial, d.Product, d.BatteryLevel, 1)
}
