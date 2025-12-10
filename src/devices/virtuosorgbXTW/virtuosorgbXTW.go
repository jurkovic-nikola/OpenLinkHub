package virtuosorgbXTW

// Package: CORSAIR VIRTUOSO RGB WIRELESS XT
// This is the primary package for CORSAIR VIRTUOSO RGB WIRELESS XT.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/stats"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"os"
	"regexp"
	"slices"
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
	ZoneColors         map[int]ZoneColors
	Profiles           map[int]DPIProfile
	SleepMode          int
	MuteIndicator      int
	SideTone           int
	SideToneValue      int
}

type DPIProfile struct {
	Name        string `json:"name"`
	Value       uint16
	PackerIndex int
	ColorIndex  map[int][]int
	Color       *rgb.Color
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
	SlipstreamId          uint16
	Brightness            map[int]string
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
	Exit                  bool
	MuteStatus            byte
	MuteIndicators        map[int]string
	SideToneModes         map[int]string
	BatteryLevel          uint16
	RGBModes              []string
}

var (
	pwd                  = ""
	cmdSoftwareMode      = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode      = []byte{0x01, 0x03, 0x00, 0x01}
	cmdSleepMode         = []byte{0x01, 0x03, 0x00, 0x04}
	cmdGetFirmware       = []byte{0x02, 0x13}
	cmdWriteColor        = []byte{0x06, 0x00}
	cmdOpenEndpoint      = []byte{0x0d, 0x00, 0x01}
	cmdOpenWriteEndpoint = []byte{0x01, 0x0d, 0x00, 0x01}
	cmdSleep             = []byte{0x01, 0x0e, 0x00}
	cmdBatteryLevel      = []byte{0x02, 0x0f}
	cmdSidetoneMode      = []byte{0x01, 0x46, 0x00}
	cmdSidetone          = []byte{0x01, 0x47, 0x00}
	bufferSize           = 64
	bufferSizeWrite      = bufferSize + 1
	headerSize           = 3
	headerWriteSize      = 4
	rgbProfileUpgrade    = []string{"gradient"}
	rgbModes             = []string{
		"colorpulse",
		"colorshift",
		"colorwarp",
		"cpu-temperature",
		"flickering",
		"gpu-temperature",
		"gradient",
		"headset",
		"off",
		"rainbow",
		"rotator",
		"static",
		"storm",
		"watercolor",
		"wave",
	}
)

func Init(vendorId, slipstreamId, productId uint16, dev *hid.Device, endpoint byte, serial string) *Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	// Init new struct with HID device
	d := &Device{
		dev:          dev,
		Template:     "virtuosorgbXTW.html",
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
		Product: "VIRTUOSO RGB WIRELESS XT",
		SleepModes: map[int]string{
			1:  "1 minute",
			5:  "5 minutes",
			10: "10 minutes",
			15: "15 minutes",
			30: "30 minutes",
			60: "1 hour",
		},
		RGBModes:              rgbModes,
		LEDChannels:           3,
		ChangeableLedChannels: 3,
		MuteIndicators: map[int]string{
			0: "Disabled",
			1: "Enabled",
		},
		SideToneModes: map[int]string{
			0: "Disabled",
			1: "Enabled",
		},
	}

	d.getDebugMode()       // Debug mode
	d.loadRgb()            // Load RGB
	d.loadDeviceProfiles() // Load all device profiles
	d.saveDeviceProfile()  // Save profile
	return d
}

// configureHeadset will configure headset sidetone and active noise cancellation
func (d *Device) configureHeadset() {
	if d.DeviceProfile == nil {
		return
	}

	if d.DeviceProfile.SideTone == 1 {
		// Sidetone is enabled, ANC needs to be disabled
		buf := make([]byte, 1)
		buf[0] = 0x00

		// Setup Sidetone
		_, err := d.transfer(cmdSidetoneMode, buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to enable side tone")
			return
		}

		if d.DeviceProfile.SideToneValue < 0 {
			d.DeviceProfile.SideToneValue = 0
		}

		if d.DeviceProfile.SideToneValue > 100 {
			d.DeviceProfile.SideToneValue = 100
		}

		// Setup Sidetone value
		buf = make([]byte, 2)
		binary.LittleEndian.PutUint16(buf[0:2], uint16(d.DeviceProfile.SideToneValue*10)) // Sidetone 0-100 * 10
		_, err = d.transfer(cmdSidetone, buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to set side tone value")
			return
		}
	} else {
		// Disable sidetone
		buf := make([]byte, 1)
		buf[0] = 0x01
		_, err := d.transfer(cmdSidetoneMode, buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to enable side tone")
			return
		}
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

// StopDirty will stop devices in a dirty way
func (d *Device) StopDirty() uint8 {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device (dirty)...")

	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}

	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	d.Connected = false
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
		d.getDeviceFirmware() // Firmware
		d.setSoftwareMode()   // Activate software mode
		d.getBatterLevel()    // Battery level
		d.initLeds()          // Init LED ports
		d.setDeviceColor()    // Device color
		d.configureHeadset()  // Headset config
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
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
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
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
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
	pf.Gradients = profile.Gradients

	d.Rgb.Profiles[profileName] = *pf
	d.saveRgbProfile()
	if d.Connected {
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true // Exit current RGB mode
			d.activeRgb = nil
		}
		d.setDeviceColor() // Restart RGB
	}
	return 1
}

// UpdateRgbProfile will update device RGB profile
func (d *Device) UpdateRgbProfile(_ int, profile string) uint8 {
	if !d.Connected {
		return 0
	}

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

	if d.DeviceProfile.RGBProfile == "static" || d.DeviceProfile.RGBProfile == "headset" {
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
	if d.DeviceProfile.RGBProfile == "static" || d.DeviceProfile.RGBProfile == "headset" {
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true // Exit current RGB mode
			d.activeRgb = nil
		}
		d.setDeviceColor() // Restart RGB
	}
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

// SaveHeadsetZoneColors will save headset zone colors
func (d *Device) SaveHeadsetZoneColors(zoneColors map[int]rgb.Color) uint8 {
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

// getManufacturer will return device manufacturer
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
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
	stats.UpdateBatteryStats(d.Serial, d.Product, d.BatteryLevel, 2)
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	_, err := d.transfer(cmdSoftwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
}

// SetSleepMode will switch a device to sleep mode
func (d *Device) SetSleepMode() {
	_, err := d.transfer(cmdSleepMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
	//d.Connected = false
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
		deviceProfile.RGBProfile = "headset"
		deviceProfile.Label = "Headset"
		deviceProfile.Active = true
		deviceProfile.ZoneColors = map[int]ZoneColors{
			0: { // Logo
				ColorIndex: []int{0, 3, 6},
				Color: &rgb.Color{
					Red:        255,
					Green:      255,
					Blue:       0,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 255, 255, 0),
				},
				Name: "Logo",
			},
			1: { // Microphone
				ColorIndex: []int{2, 5, 8},
				Color: &rgb.Color{
					Red:        0,
					Green:      255,
					Blue:       255,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 0, 255, 255),
				},
				Name: "Microphone",
			},
			2: { // Indicator LED
				ColorIndex: []int{1, 4, 7},
				Color: &rgb.Color{
					Red:        0,
					Green:      255,
					Blue:       255,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 0, 255, 255),
				},
				Name: "Indicator LED",
			},
		}

		deviceProfile.SleepMode = 15
		deviceProfile.MuteIndicator = 0
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
		deviceProfile.RGBProfile = d.DeviceProfile.RGBProfile
		deviceProfile.Label = d.DeviceProfile.Label
		deviceProfile.ZoneColors = d.DeviceProfile.ZoneColors
		deviceProfile.SleepMode = d.DeviceProfile.SleepMode
		deviceProfile.MuteIndicator = d.DeviceProfile.MuteIndicator
		deviceProfile.SideTone = d.DeviceProfile.SideTone
		deviceProfile.SideToneValue = d.DeviceProfile.SideToneValue

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

// UpdateMuteIndicator will update device mute indicator
func (d *Device) UpdateMuteIndicator(minutes int) uint8 {
	if d.DeviceProfile != nil {
		d.DeviceProfile.MuteIndicator = minutes
		d.saveDeviceProfile()
		return 1
	}
	return 0
}

// UpdateSidetone will update device side tone
func (d *Device) UpdateSidetone(value int) uint8 {
	if d.DeviceProfile != nil {
		d.DeviceProfile.SideTone = value
		d.saveDeviceProfile()
		d.configureHeadset()
		return 1
	}
	return 0
}

// UpdateSidetoneValue will update device sidetone value
func (d *Device) UpdateSidetoneValue(value int) uint8 {
	if d.DeviceProfile != nil {
		if d.DeviceProfile.SideTone == 1 {
			d.DeviceProfile.SideToneValue = value
			d.saveDeviceProfile()
			d.configureHeadset()
			return 1
		} else {
			return 2
		}
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

// setSleepTimer will set device sleep timer
func (d *Device) setSleepTimer() uint8 {
	if d.Exit {
		return 0
	}
	if d.DeviceProfile != nil {
		_, err := d.transfer(cmdOpenWriteEndpoint, nil)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change device sleep timer")
			return 0
		}

		buf := make([]byte, 4)
		sleep := d.DeviceProfile.SleepMode * (60 * 1000)
		binary.LittleEndian.PutUint32(buf, uint32(sleep))

		_, err = d.transfer(cmdSleep, buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change device sleep timer")
			return 0
		}
		return 1

	}
	return 0
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
	_, err := d.transfer(cmdOpenEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to change device mode")
	}
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	buf := make([]byte, d.LEDChannels*3)
	if d.DeviceProfile == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to set color. DeviceProfile is null!")
		return
	}

	if d.DeviceProfile.RGBProfile == "headset" {
		for _, zoneColor := range d.DeviceProfile.ZoneColors {
			zoneColor.Color.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
			zoneColor.Color = rgb.ModifyBrightness(*zoneColor.Color)
			zoneColorIndexRange := zoneColor.ColorIndex
			for key, zoneColorIndex := range zoneColorIndexRange {
				switch key {
				case 0: // Red
					buf[zoneColorIndex] = byte(zoneColor.Color.Red)
				case 1: // Green
					buf[zoneColorIndex] = byte(zoneColor.Color.Green)
				case 2: // Blue
					buf[zoneColorIndex] = byte(zoneColor.Color.Blue)
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

		profile.StartColor.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
		profileColor := rgb.ModifyBrightness(profile.StartColor)
		for _, zoneColor := range d.DeviceProfile.ZoneColors {
			zoneColorIndexRange := zoneColor.ColorIndex
			for key, zoneColorIndex := range zoneColorIndexRange {
				switch key {
				case 0: // Red
					buf[zoneColorIndex] = byte(profileColor.Red)
				case 1: // Green
					buf[zoneColorIndex] = byte(profileColor.Green)
				case 2: // Blue
					buf[zoneColorIndex] = byte(profileColor.Blue)
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
				r.RGBBrightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
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
				case "gradient":
					{
						r.ColorshiftGradient(startTime, profile.Gradients, profile.Speed)
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
				zoneKeys := make([]int, 0, len(d.DeviceProfile.ZoneColors))
				for key := range d.DeviceProfile.ZoneColors {
					zoneKeys = append(zoneKeys, key)
				}
				sort.Ints(zoneKeys)

				m := 0
				for _, key := range zoneKeys {
					zoneColor := d.DeviceProfile.ZoneColors[key]
					for _, zoneColorIndex := range zoneColor.ColorIndex {
						if m >= len(buff) {
							break
						}
						buf[zoneColorIndex] = buff[m]
						m++
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
	if d.Exit {
		return
	}

	// When mic is muted and MuteIndicator is set
	if d.DeviceProfile.MuteIndicator == 1 && d.MuteStatus == 1 {
		data[2] = 0xff
		data[5] = 0x00
		data[8] = 0x00
	}

	buffer := make([]byte, len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	copy(buffer[headerWriteSize:], data)

	_, err := d.transfer(cmdWriteColor, buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
	}
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

	reports := make([]byte, 1)
	err := d.dev.SetNonblock(true)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to SetNonblock")
	}

	for {
		n, e := d.dev.Read(reports)
		if e != nil {
			if n < 0 {
				//
			}
			if e == hid.ErrTimeout || n == 0 {
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

// NotifyMuteChanged will change microphone LED based on microphone status
func (d *Device) NotifyMuteChanged(status byte) {
	d.MuteStatus = status
	// RGB reset
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	buf := make([]byte, d.LEDChannels*3)
	if status == 1 {
		buf[2] = 0xff
		buf[5] = 0x00
		buf[8] = 0x00
	} else {
		buf[2] = 0x00
		buf[5] = 0xff
		buf[8] = 0x00
	}

	d.writeColor(buf)
	time.Sleep(1500 * time.Millisecond)
	d.setDeviceColor()
}

// ModifyBatteryLevel will modify battery level
func (d *Device) ModifyBatteryLevel(batteryLevel uint16) {
	d.BatteryLevel = batteryLevel
	stats.UpdateBatteryStats(d.Serial, d.Product, d.BatteryLevel, 2)
}
