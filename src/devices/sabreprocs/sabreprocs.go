package sabreprocs

// Package: CORSAIR SABRE PRO Gaming Mouse
// This is the primary package for CORSAIR SABRE PRO Gaming Mouse.
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

// DeviceProfile struct contains all device profile
type DeviceProfile struct {
	Active             bool
	Path               string
	Product            string
	Serial             string
	Label              string
	Profile            int
	PollingRate        int
	DPIColor           *rgb.Color
	SniperColor        *rgb.Color
	Profiles           map[int]DPIProfile
	AngleSnapping      int
	ButtonOptimization int
	KeyAssignmentHash  string
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
	Debug                 bool
	dev                   *hid.Device
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
	Brightness            map[int]string
	PollingRates          map[int]string
	SwitchModes           map[int]string
	KeyAssignmentTypes    map[int]string
	LEDChannels           int
	ChangeableLedChannels int
	CpuTemp               float32
	GpuTemp               float32
	Layouts               []string
	Rgb                   *rgb.RGB
	rgbMutex              sync.RWMutex
	SleepModes            map[int]string
	timer                 *time.Ticker
	autoRefreshChan       chan struct{}
	mutex                 sync.Mutex
	timerKeepAlive        *time.Ticker
	keepAliveChan         chan struct{}
	Exit                  bool
	KeyAssignment         map[int]inputmanager.KeyAssignment
	InputActions          map[uint16]inputmanager.InputAction
	PressLoop             bool
	keyAssignmentFile     string
	KeyAssignmentData     *inputmanager.KeyAssignment
	ModifierIndex         uint32
	SniperMode            bool
	MacroTracker          map[int]uint16
	instance              *common.Device
}

var (
	pwd                   = ""
	cmdSoftwareMode       = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode       = []byte{0x01, 0x03, 0x00, 0x01}
	cmdGetFirmware        = []byte{0x02, 0x13}
	cmdWriteColor         = []byte{0x06, 0x00}
	cmdOpenEndpoint       = []byte{0x0d, 0x00, 0x01}
	cmdHeartbeat          = []byte{0x12}
	cmdSetDpi             = []byte{0x01}
	modeDpi               = []byte{0x21, 0x22}
	cmdAngleSnapping      = []byte{0x01, 0x07, 0x00}
	cmdButtonOptimization = []byte{0x01, 0xb0, 0x00}
	cmdSetPollingRate     = []byte{0x01, 0x01, 0x00}
	cmdOpenWriteEndpoint  = []byte{0x0d, 0x01, 0x02}
	cmdWrite              = []byte{0x06, 0x01}
	cmdCloseEndpoint      = []byte{0x05, 0x01, 0x01}
	cmdStart              = []byte{0x02, 0x03}
	bufferSize            = 64
	bufferSizeWrite       = bufferSize + 1
	headerSize            = 2
	headerWriteSize       = 4
	keyAmount             = 6
	minDpiValue           = 100
	maxDpiValue           = 18000
	deviceRefreshInterval = 1000
	deviceKeepAlive       = 20000
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
		Template:  "sabreprocs.html",
		VendorId:  vendorId,
		ProductId: productId,
		Firmware:  "n/a",
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
		Product: "SABRE PRO CS",
		SleepModes: map[int]string{
			1:  "1 minute",
			5:  "5 minutes",
			10: "10 minutes",
			15: "15 minutes",
			30: "30 minutes",
			60: "1 hour",
		},
		LEDChannels:           1,
		ChangeableLedChannels: 1,
		PollingRates: map[int]string{
			0: "Not Set",
			1: "125 Hz / 8 msec",
			2: "250 Hu / 4 msec",
			3: "500 Hz / 2 msec",
			4: "1000 Hz / 1 msec",
			5: "2000 Hz / 0.5 msec",
			6: "4000 Hz / 0.25 msec",
			7: "8000 Hz / 0.125 msec",
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

		InputActions:      inputmanager.GetInputActions(),
		keyAssignmentFile: "/database/key-assignments/sabreprocs.json",
		MacroTracker:      make(map[int]uint16),
	}

	d.getDebugMode()          // Debug mode
	d.getManufacturer()       // Manufacturer
	d.getSerial()             // Serial
	d.loadRgb()               // Load RGB
	d.loadDeviceProfiles()    // Load all device profiles
	d.saveDeviceProfile()     // Save profile
	d.setSoftwareMode()       // Activate software mode
	d.setAutoRefresh()        // Set auto device refresh
	d.getDeviceFirmware()     // Firmware
	d.setAngleSnapping()      // Angle snapping
	d.setButtonOptimization() // Button optimization
	d.initLeds()              // Init LED ports
	d.setDeviceColor()        // Device color
	d.backendListener()       // Control listener
	d.setKeepAlive()          // Keepalive
	d.toggleDPI()             // DPI
	d.loadKeyAssignments()    // Key Assignments
	d.setupKeyAssignment()    // Setup key assignments
	d.createDevice()          // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeSabreProCs,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-mouse.svg",
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
			if d.keepAliveChan != nil {
				close(d.keepAliveChan)
			}
		})
	}()

	d.setHardwareMode()
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
			if d.keepAliveChan != nil {
				close(d.keepAliveChan)
			}
		})
	}()
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 2
}

// getSerial will return device serial number
func (d *Device) getSerial() {
	serial, err := d.dev.GetSerialNbr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device serial number")
	}
	d.Serial = serial
}

// getManufacturer will return device manufacturer
func (d *Device) getManufacturer() {
	manufacturer, err := d.dev.GetMfrStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get manufacturer")
	}
	d.Manufacturer = manufacturer
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
		d.loadKeyAssignments()
		d.setupKeyAssignment()
		return 1
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
			profile.Value = stage
			d.DeviceProfile.Profiles[key] = profile
			i++
		}
	}

	if i > 0 {
		d.saveDeviceProfile()
		d.toggleDPI()
		return 1
	}
	return 0
}

// SaveMouseDpiColors will save mouse dpi colors
func (d *Device) SaveMouseDpiColors(dpi rgb.Color, dpiColors map[int]rgb.Color) uint8 {
	i := 0
	if d.DeviceProfile == nil {
		return 0
	}
	if dpi.Red > 255 ||
		dpi.Green > 255 ||
		dpi.Blue > 255 ||
		dpi.Red < 0 ||
		dpi.Green < 0 ||
		dpi.Blue < 0 {
		return 0
	}

	// Zone Colors
	for key, zone := range dpiColors {
		if zone.Red > 255 ||
			zone.Green > 255 ||
			zone.Blue > 255 ||
			zone.Red < 0 ||
			zone.Green < 0 ||
			zone.Blue < 0 {
			continue
		}
		if profileColor, ok := d.DeviceProfile.Profiles[key]; ok {
			profileColor.Color.Red = zone.Red
			profileColor.Color.Green = zone.Green
			profileColor.Color.Blue = zone.Blue
			profileColor.Color.Hex = fmt.Sprintf("#%02x%02x%02x", int(zone.Red), int(zone.Green), int(zone.Blue))
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

// getManufacturer will return device manufacturer
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
}

// setHardwareMode will switch a device to hardware mode
func (d *Device) setHardwareMode() {
	_, err := d.transfer(cmdHardwareMode, nil, false)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	i := 0
	_, err := d.transfer(cmdStart, nil, true)
	if err != nil {
		for {
			if i >= 20 {
				break
			}
			_, err = d.transfer(cmdStart, nil, true)
			if err == nil {
				break
			}
			time.Sleep(2 * time.Second)
			i++
		}
	}

	if i == 20 {
		logger.Log(logger.Fields{"error": "failed to read data from device"}).Error("Unable to read device data after 20 attempts")
		return
	}

	_, err = d.transfer(cmdSoftwareMode, nil, true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	fw, err := d.transfer(
		cmdGetFirmware,
		nil,
		false,
	)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
	}

	v1, v2, v3 := int(fw[3]), int(fw[4]), int(binary.LittleEndian.Uint16(fw[5:7]))
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

// setAngleSnapping will change Angle Snapping mode
func (d *Device) setAngleSnapping() {
	if d.DeviceProfile == nil {
		return
	}

	if d.DeviceProfile.AngleSnapping < 0 || d.DeviceProfile.AngleSnapping > 1 {
		return
	}

	buf := make([]byte, 1)
	buf[0] = byte(d.DeviceProfile.AngleSnapping)
	_, _ = d.transfer(cmdAngleSnapping, buf, false)
}

// setButtonOptimization will change Button Response Optimization mode
func (d *Device) setButtonOptimization() {
	if d.DeviceProfile == nil {
		return
	}

	if d.DeviceProfile.ButtonOptimization < 0 || d.DeviceProfile.ButtonOptimization > 1 {
		return
	}

	buf := make([]byte, 1)
	buf[0] = byte(d.DeviceProfile.ButtonOptimization)
	_, _ = d.transfer(cmdButtonOptimization, buf, false)
}

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"

	deviceProfile := &DeviceProfile{
		Product: d.Product,
		Serial:  d.Serial,
		Path:    profilePath,
	}

	// First save, assign saved profile to a device
	if d.DeviceProfile == nil {
		// RGB, Label
		deviceProfile.Label = "Mouse"
		deviceProfile.Active = true

		deviceProfile.DPIColor = &rgb.Color{
			Red:        0,
			Green:      255,
			Blue:       255,
			Brightness: 1,
			Hex:        fmt.Sprintf("#%02x%02x%02x", 0, 255, 255),
		}

		deviceProfile.SniperColor = &rgb.Color{
			Red:        255,
			Green:      255,
			Blue:       0,
			Brightness: 1,
			Hex:        fmt.Sprintf("#%02x%02x%02x", 255, 255, 0),
		}

		deviceProfile.Profiles = map[int]DPIProfile{
			0: {
				Name:        "Stage 1",
				Value:       400,
				PackerIndex: 1,
				ColorIndex: map[int][]int{
					0: {0, 1, 2},
				},
				Color: &rgb.Color{
					Red:        255,
					Green:      0,
					Blue:       0,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 255, 0, 0),
				},
			},
			1: {
				Name:        "Stage 2",
				Value:       800,
				PackerIndex: 2,
				ColorIndex: map[int][]int{
					0: {0, 1, 2},
				},
				Color: &rgb.Color{
					Red:        255,
					Green:      255,
					Blue:       255,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 255, 255, 255),
				},
			},
			2: {
				Name:        "Stage 3",
				Value:       1200,
				PackerIndex: 3,
				ColorIndex: map[int][]int{
					0: {0, 1, 2},
				},
				Color: &rgb.Color{
					Red:        0,
					Green:      255,
					Blue:       0,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 0, 255, 0),
				},
			},
			3: {
				Name:        "Stage 4",
				Value:       1600,
				PackerIndex: 4,
				ColorIndex: map[int][]int{
					0: {0, 1, 2},
				},
				Color: &rgb.Color{
					Red:        128,
					Green:      0,
					Blue:       128,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 128, 0, 128),
				},
			},
			4: {
				Name:        "Stage 5",
				Value:       3200,
				PackerIndex: 5,
				ColorIndex: map[int][]int{
					0: {0, 1, 2},
				},
				Color: &rgb.Color{
					Red:        0,
					Green:      191,
					Blue:       255,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 0, 191, 255),
				},
			},
			5: {
				Name:        "Sniper",
				Value:       200,
				PackerIndex: 6,
				ColorIndex: map[int][]int{
					0: {0, 1, 2},
				},
				Color: &rgb.Color{
					Red:        255,
					Green:      255,
					Blue:       0,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 255, 255, 0),
				},
				Sniper: true,
			},
		}

		deviceProfile.Profile = 2
		deviceProfile.PollingRate = 4
	} else {
		if d.DeviceProfile.PollingRate == 0 {
			deviceProfile.PollingRate = 4
		} else {
			deviceProfile.PollingRate = d.DeviceProfile.PollingRate
		}

		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.Label = d.DeviceProfile.Label
		deviceProfile.Profiles = d.DeviceProfile.Profiles
		deviceProfile.Profile = d.DeviceProfile.Profile
		deviceProfile.DPIColor = d.DeviceProfile.DPIColor
		deviceProfile.SniperColor = d.DeviceProfile.SniperColor
		deviceProfile.AngleSnapping = d.DeviceProfile.AngleSnapping
		deviceProfile.ButtonOptimization = d.DeviceProfile.ButtonOptimization
		deviceProfile.KeyAssignmentHash = d.DeviceProfile.KeyAssignmentHash
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
		logger.Log(logger.Fields{"error": err, "location": deviceProfile.Path}).Error("Unable to close file handle")
	}

	d.loadDeviceProfiles() // Reload
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
			32: {
				Name:          "DPI",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
			},
			16: {
				Name:          "Forward Button",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
			},
			8: {
				Name:          "Back Button",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
			},
			4: {
				Name:          "Middle Button",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
			},
			2: {
				Name:          "Right Button",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
			},
			1: {
				Name:          "Left Button",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
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

// initLeds will initialize LED endpoint
func (d *Device) initLeds() {
	_, err := d.transfer(cmdOpenEndpoint, nil, false)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
}

// getSniperColor will get sniper dpi color
func (d *Device) getSniperColor() *rgb.Color {
	for _, val := range d.DeviceProfile.Profiles {
		if val.Sniper {
			return val.Color
		}
	}
	return nil
}

// getSniperColorIndex will get sniper dpi color slice
func (d *Device) getSniperColorIndex() DPIProfile {
	for _, val := range d.DeviceProfile.Profiles {
		if val.Sniper {
			return val
		}
	}
	return DPIProfile{}
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	buf := make([]byte, d.LEDChannels*3)
	if d.DeviceProfile == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to set color. DeviceProfile is null!")
		return
	}

	// DPI
	dpiColor := d.DeviceProfile.Profiles[d.DeviceProfile.Profile].Color
	if d.SniperMode {
		dpiColor = d.getSniperColor()
	}
	if dpiColor == nil {
		return
	}

	dpiLeds := d.DeviceProfile.Profiles[d.DeviceProfile.Profile]
	for i := 0; i < len(dpiLeds.ColorIndex); i++ {
		dpiColorIndexRange := dpiLeds.ColorIndex[i]
		for key, dpiColorIndex := range dpiColorIndexRange {
			switch key {
			case 0: // Red
				buf[dpiColorIndex] = byte(dpiColor.Red)
			case 1: // Green
				buf[dpiColorIndex] = byte(dpiColor.Green)
			case 2: // Blue
				buf[dpiColorIndex] = byte(dpiColor.Blue)
			}
		}
	}

	d.writeColor(buf)
}

// Close will close all timers and channels before restart
func (d *Device) Close() {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	time.Sleep(500 * time.Millisecond)
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
		_, err := d.transfer(cmdSetPollingRate, buf, false)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to set mouse polling rate")
			return 0
		}
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

// UpdateButtonOptimization will update button response optimization mode
func (d *Device) UpdateButtonOptimization(buttonOptimizationMode int) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if d.DeviceProfile.ButtonOptimization == buttonOptimizationMode {
		return 0
	}

	d.DeviceProfile.ButtonOptimization = buttonOptimizationMode
	d.saveDeviceProfile()
	d.setButtonOptimization()
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

				for _, dpiCode := range modeDpi {
					buf := make([]byte, 4)
					buf[0] = dpiCode
					buf[1] = 0x00
					binary.LittleEndian.PutUint16(buf[2:4], value)
					_, err := d.transfer(cmdSetDpi, buf, false)
					if err != nil {
						logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to set dpi")
					}
				}
				if d.activeRgb != nil {
					d.activeRgb.Exit <- true // Exit current RGB mode
					d.activeRgb = nil
				}
				d.setDeviceColor() // Restart RGB
			}
		}
	} else {
		// Reset to normal DPI mode
		d.toggleDPI()
	}
}

// toggleDPI will change DPI mode
func (d *Device) toggleDPI() {
	if d.Exit {
		return
	}
	if d.DeviceProfile != nil {
		profile := d.DeviceProfile.Profiles[d.DeviceProfile.Profile]
		value := profile.Value

		// Send DPI packet
		if value < uint16(minDpiValue) {
			value = uint16(minDpiValue)
		}
		if value > uint16(maxDpiValue) {
			value = uint16(maxDpiValue)
		}

		for _, dpiCode := range modeDpi {
			buf := make([]byte, 4)
			buf[0] = dpiCode
			buf[1] = 0x00
			binary.LittleEndian.PutUint16(buf[2:4], value)
			_, err := d.transfer(cmdSetDpi, buf, false)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to set dpi")
			}
		}
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true // Exit current RGB mode
			d.activeRgb = nil
		}
		d.setDeviceColor() // Restart RGB
	}
}

// keepAlive will keep a device alive
func (d *Device) keepAlive() {
	if d.Exit {
		return
	}
	_, err := d.transfer(cmdHeartbeat, nil, false)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write heartbeat to a device")
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

// writeColor will write data to the device with a specific endpoint.
func (d *Device) writeColor(data []byte) {
	if d.Exit {
		return
	}
	buffer := make([]byte, len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	copy(buffer[headerWriteSize:], data)

	_, err := d.transfer(cmdWriteColor, buffer, false)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
	}
}

// writeKeyAssignmentData will write key assignment to the device.
func (d *Device) writeKeyAssignmentData(data []byte) {
	if d.Exit {
		return
	}

	// Open endpoint
	_, err := d.transfer(cmdOpenWriteEndpoint, nil, false)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to open write endpoint")
		return
	}

	// Send data
	buffer := make([]byte, len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	copy(buffer[headerWriteSize:], data)
	_, err = d.transfer(cmdWrite, buffer, false)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to data endpoint")
	}

	// Close endpoint
	_, err = d.transfer(cmdCloseEndpoint, nil, false)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to close endpoint")
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

// backendListener will listen for events from the device
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

				if data[1] == 0x02 {
					d.triggerKeyAssignment(binary.LittleEndian.Uint32(data[2:6]))
				}
			}
		}
	}()
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

	buf := make([]byte, keyAmount)
	i := 0
	for _, k := range keys {
		value := d.KeyAssignment[k]
		if value.Default {
			buf[i] = byte(1)
		} else {
			buf[i] = byte(0)
		}
		i++
	}
	d.writeKeyAssignmentData(buf)
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
func (d *Device) triggerKeyAssignment(value uint32) {
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
			case 8:
				d.sniperMode(false)
			case 9:
				inputmanager.InputControlMouseHold(val.ActionCommand, false)
			}
		}

		if isPressed {
			if mask == 0x20 && val.Default {
				d.ModifyDpi()
				continue
			}
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
							inputmanager.InputControlKeyboard(v.ActionCommand, v.ActionHold)
						case 9:
							inputmanager.InputControlMouse(v.ActionCommand)
						case 5:
							if v.ActionDelay > 0 {
								time.Sleep(time.Duration(v.ActionDelay) * time.Millisecond)
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
func (d *Device) transfer(endpoint, buffer []byte, timeout bool) ([]byte, error) {
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

	if timeout {
		// Get data from a device
		if _, err := d.dev.ReadWithTimeout(bufferR, 100*time.Millisecond); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
			return bufferR, err
		}
	} else {
		// Get data from a device
		if _, err := d.dev.Read(bufferR); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
			return bufferR, err
		}
	}
	return bufferR, nil
}
