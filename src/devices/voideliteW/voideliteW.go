package voideliteW

// Package: CORSAIR VOID ELITE WIRELESS
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/audio"
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"OpenLinkHub/src/stats"
	"github.com/sstallion/go-hid"
)

type SideTone struct {
	Min int
	Max int
	Val byte
}

type ZoneColors struct {
	Color      *rgb.Color
	ColorIndex []int
	Name       string
}

type Equalizer struct {
	Name  string
	Value float64
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
	Equalizers         map[int]Equalizer
	SleepMode          int
	MuteIndicator      int
	SideTone           int
	SideToneValue      int
	RgbOff             bool
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
	mutex                 sync.Mutex
	Exit                  bool
	MuteStatus            byte
	MuteIndicators        map[int]string
	RGBModes              []string
	BatteryLevel          uint16
	instance              *common.Device
	ZoneAmount            int
	Connected             bool
	ResetColor            bool
}

var (
	pwd                 = ""
	cmdSoftwareMode     = []byte{0x01, 0x00}
	cmdHardwareMode     = []byte{0x00, 0x00}
	cmdDeviceMode       = byte(0xc8)
	cmdInitLed          = byte(0xc9)
	cmdWriteColor       = byte(0xcb)
	cmdSideTone         = byte(0xca)
	cmdSideToneReportId = byte(0xff)
	dataTypeLedChannels = []byte{0x64, 0x66}
	dataTypeColorRedL   = byte(0x1c)
	dataTypeColorGreenL = byte(0x16)
	dataTypeColorBlueL  = byte(0x17)
	dataTypeColorRedR   = byte(0x1d)
	dataTypeColorGreenR = byte(0x18)
	dataTypeColorBlueR  = byte(0x19)
	dataTypeColors      = map[int]byte{
		0: dataTypeColorRedL,
		1: dataTypeColorGreenL,
		2: dataTypeColorBlueL,
		3: dataTypeColorRedR,
		4: dataTypeColorGreenR,
		5: dataTypeColorBlueR,
	}
	bufferSize = 32
	rgbModes   = []string{
		"headset",
		"static",
	}
	sideTone = []SideTone{
		{0, 0, 0xc0},
		{1, 1, 0xd9},
		{2, 2, 0xdf},
		{3, 3, 0xe2},
		{4, 4, 0xe5},
		{5, 5, 0xe7},
		{6, 6, 0xe8},
		{7, 7, 0xe9},
		{8, 8, 0xeb},
		{9, 9, 0xec},
		{10, 11, 0xed},
		{12, 12, 0xee},
		{13, 14, 0xef},
		{15, 15, 0xf0},
		{16, 17, 0xf1},
		{18, 19, 0xf2},
		{20, 22, 0xf3},
		{23, 25, 0xf4},
		{26, 28, 0xf5},
		{29, 31, 0xf6},
		{32, 35, 0xf7},
		{36, 39, 0xf8},
		{40, 44, 0xf9},
		{45, 50, 0xfa},
		{51, 56, 0xfb},
		{57, 63, 0xfc},
		{64, 70, 0xfd},
		{71, 79, 0xfe},
		{80, 89, 0xff},
		{90, 100, 0x00},
	}

	// dataTypeColorRed    = byte(0x1c) - L
	// dataTypeColorRed    = byte(0x1d) - R
	// dataTypeColorRed    = byte(0x1e) - Mic

	// dataTypeColorGreen  = byte(0x16) - L
	// dataTypeColorGreen  = byte(0x18) - R

	// dataTypeColorBlue   = byte(0x17) - L
	// dataTypeColorBlue   = byte(0x19) - R

	// dataTypeColorGreen  = byte(0x1b) - Indicator
)

func Init(vendorId, slipstreamId, productId uint16, dev *hid.Device, endpoint byte, serial string) *Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	// Init new struct with HID device
	d := &Device{
		dev:          dev,
		Template:     "voideliteW.html",
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
		Product:               "VOID ELITE WIRELESS",
		RGBModes:              rgbModes,
		LEDChannels:           2,
		ChangeableLedChannels: 2,
		MuteIndicators: map[int]string{
			0: "Disabled",
			1: "Enabled",
		},
		ZoneAmount: 2,
	}

	d.getDebugMode()       // Debug mode
	d.loadRgb()            // Load RGB
	d.loadDeviceProfiles() // Load all device profiles
	d.saveDeviceProfile()  // Save profile
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	// Placeholder
}

// StopInternal will stop all device operations and switch a device back to hardware mode
func (d *Device) StopInternal() {
	d.Exit = true

	if d.activeRgb != nil {
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}

	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")
	if d.Connected {
		d.setHardwareMode()
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeHS80RGB,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-headphone.svg",
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

// GetZoneColors will return current device zone colors
func (d *Device) GetZoneColors() interface{} {
	if d.DeviceProfile == nil {
		return nil
	}
	return d.DeviceProfile.ZoneColors
}

// StopDirty will device in a dirty way
func (d *Device) StopDirty() uint8 {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device (dirty)...")
	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 2
}

// SetConnected will change connected status
func (d *Device) SetConnected(value bool) {
	if d.Connected {
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true
			d.activeRgb = nil
		}
		d.Connected = value
		time.Sleep(1000 * time.Millisecond)
	}
}

// Connect will connect to a device
func (d *Device) Connect() {
	if !d.Connected {
		d.Connected = true
		d.getDeviceFirmware()   // Firmware
		d.setSoftwareMode()     // Activate software mode
		d.initLeds()            // Init LED ports
		d.setDeviceColor()      // Device color
		d.setDeviceBrightness() // Brightness
		d.configureHeadset()    // Headset config
		d.setEqualizer()        // Equalizer
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
		d.setDeviceColor()
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

	if d.DeviceProfile.RGBProfile == "static" || d.DeviceProfile.RGBProfile == "headset" {
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true
			d.activeRgb = nil
		}
		d.setDeviceColor()
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
			d.activeRgb.Exit <- true
			d.activeRgb = nil
		}
		d.setDeviceColor()
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
			d.activeRgb.Exit <- true
			d.activeRgb = nil
		}
		d.setDeviceColor()
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
	err := d.transfer(cmdDeviceMode, cmdHardwareMode)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to change device operating state")
	}
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	err := d.transfer(cmdDeviceMode, cmdSoftwareMode)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to change device operating state")
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

	fw, err := common.GetBcdDeviceHex(info.Path)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device firmware")
	}
	d.Firmware = fw
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

	if d.DeviceProfile == nil {
		deviceProfile.RGBProfile = "headset"
		deviceProfile.Label = "Headset"
		deviceProfile.Active = true
		deviceProfile.ZoneColors = map[int]ZoneColors{
			0: { // Logo L
				ColorIndex: []int{0, 1, 2},
				Color: &rgb.Color{
					Red:        255,
					Green:      255,
					Blue:       0,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 255, 255, 0),
				},
				Name: "Logo L",
			},

			1: { // Logo R
				ColorIndex: []int{3, 4, 5},
				Color: &rgb.Color{
					Red:        255,
					Green:      255,
					Blue:       0,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 255, 255, 0),
				},
				Name: "Logo R",
			},
		}
		deviceProfile.SleepMode = 15
		deviceProfile.MuteIndicator = 0
		deviceProfile.Equalizers = map[int]Equalizer{
			1:  {Name: "32", Value: 0},
			2:  {Name: "64", Value: 0},
			3:  {Name: "125", Value: 0},
			4:  {Name: "250", Value: 0},
			5:  {Name: "500", Value: 0},
			6:  {Name: "1K", Value: 0},
			7:  {Name: "2K", Value: 0},
			8:  {Name: "4K", Value: 0},
			9:  {Name: "8K", Value: 0},
			10: {Name: "16K", Value: 0},
		}

		deviceProfile.SideTone = 0
		deviceProfile.SideToneValue = 100
	} else {
		if d.DeviceProfile.Equalizers == nil {
			deviceProfile.Equalizers = map[int]Equalizer{
				1:  {Name: "32", Value: 0},
				2:  {Name: "64", Value: 0},
				3:  {Name: "125", Value: 0},
				4:  {Name: "250", Value: 0},
				5:  {Name: "500", Value: 0},
				6:  {Name: "1K", Value: 0},
				7:  {Name: "2K", Value: 0},
				8:  {Name: "4K", Value: 0},
				9:  {Name: "8K", Value: 0},
				10: {Name: "16K", Value: 0},
			}
		} else {
			deviceProfile.Equalizers = d.DeviceProfile.Equalizers
		}

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

		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.Brightness = d.DeviceProfile.Brightness
		deviceProfile.OriginalBrightness = d.DeviceProfile.OriginalBrightness
		deviceProfile.RGBProfile = d.DeviceProfile.RGBProfile
		deviceProfile.Label = d.DeviceProfile.Label
		deviceProfile.ZoneColors = d.DeviceProfile.ZoneColors
		deviceProfile.MuteIndicator = d.DeviceProfile.MuteIndicator

		if len(d.DeviceProfile.Path) < 1 {
			deviceProfile.Path = profilePath
			d.DeviceProfile.Path = profilePath
		} else {
			deviceProfile.Path = d.DeviceProfile.Path
		}

		deviceProfile.SideTone = d.DeviceProfile.SideTone
		deviceProfile.SideToneValue = d.DeviceProfile.SideToneValue
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

// setEqualizer will set audio equalizer
func (d *Device) setEqualizer() {
	if d.DeviceProfile == nil {
		return
	}

	if d.DeviceProfile.Equalizers == nil {
		return
	}

	if !audio.GetAudio().Enabled {
		return
	}

	for k, v := range d.DeviceProfile.Equalizers {
		audio.SetBand(k, v.Value)
	}
}

// GetEqualizers will return equalizers
func (d *Device) GetEqualizers() interface{} {
	if d.DeviceProfile == nil || d.DeviceProfile.Equalizers == nil {
		return nil
	}
	return d.DeviceProfile.Equalizers
}

// UpdateEqualizer will update device equalizer
func (d *Device) UpdateEqualizer(values map[int]float64) uint8 {
	tmp := map[int]float64{}

	if d.DeviceProfile == nil || d.DeviceProfile.Equalizers == nil {
		return 0
	}

	updated := 0

	for key, value := range values {
		if key < 1 || key > 10 {
			continue
		}

		if value > 12 || value < -12 {
			value = 0
		}

		equalizer, ok := d.DeviceProfile.Equalizers[key]
		if !ok {
			continue
		}

		if equalizer.Value == value {
			continue
		}

		equalizer.Value = value
		d.DeviceProfile.Equalizers[key] = equalizer

		tmp[key] = value
		updated++
	}

	if updated > 0 {
		d.saveDeviceProfile()
		if len(tmp) > 0 && audio.GetAudio().Enabled {
			for k, v := range tmp {
				audio.SetBand(k, v)
			}
		}
		return 1
	} else {
		return 2
	}
}

func (d *Device) configureHeadset() {
	if d.DeviceProfile == nil {
		return
	}

	if d.DeviceProfile.SideTone == 1 {
		buf := make([]byte, 4)
		buf[0] = 0x05
		buf[1] = 0x00
		buf[2] = 0x00
		buf[3] = 0x00
		err := d.transfer(cmdSideTone, buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to change side tone")
			return
		}

		if d.DeviceProfile.SideToneValue > 100 || d.DeviceProfile.SideToneValue < 1 {
			return
		}
		var sideToneValue byte = 0x00

		for _, r := range sideTone {
			if d.DeviceProfile.SideToneValue >= r.Min && d.DeviceProfile.SideToneValue <= r.Max {
				sideToneValue = r.Val
			}
		}

		report := make([]byte, 64)
		report[0] = cmdSideToneReportId
		report[1] = 0x0b
		report[2] = 0x00
		report[3] = 0xff
		report[4] = 0x04
		report[5] = 0x0e
		report[6] = 0x01
		report[7] = 0x05
		report[8] = 0x01
		report[9] = 0x04
		report[10] = 0x00
		report[11] = sideToneValue

		_, err = d.dev.SendFeatureReport(report)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to set side tone value")
			return
		}
	} else {
		buf := make([]byte, 4)
		buf[0] = 0x05
		buf[1] = 0x01
		buf[2] = 0x00
		buf[3] = 0x00
		err := d.transfer(cmdSideTone, buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to change side tone")
			return
		}
	}
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

// initLeds will initialize LED endpoint
func (d *Device) initLeds() {
	for i := 0; i < len(dataTypeLedChannels); i++ {
		err := d.transfer(cmdInitLed, []byte{dataTypeLedChannels[i]})
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to init led endpoints")
			continue
		}
	}
}

// setDeviceBrightness will set device brightness
func (d *Device) setDeviceBrightness() {
	buf := make([]byte, 13)
	buf[0] = 0x06 // Header len

	// L-R
	buf[1] = 0x2c
	buf[2] = 0xaf
	// R-R
	buf[3] = 0x26
	buf[4] = 0xaf

	// L-G
	buf[5] = 0x27
	buf[6] = 0xaf
	// R-G
	buf[7] = 0x2d
	buf[8] = 0xaf

	// L-B
	buf[9] = 0x28
	buf[10] = 0xaf
	// R-B
	buf[11] = 0x29
	buf[12] = 0xaf

	time.Sleep(500 * time.Millisecond)
	err := d.transfer(cmdWriteColor, buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
		return
	}
}

// ControlDeviceRgb will change device brightness via schedulerSchedulerBrightness
func (d *Device) ControlDeviceRgb(value bool) {
	if d.DeviceProfile == nil {
		return
	}

	d.DeviceProfile.RgbOff = value
	d.saveDeviceProfile()

	if d.Connected {
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true
			d.activeRgb = nil
		}
		d.setDeviceColor()
	}
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	buf := make([]byte, d.LEDChannels*3)
	if d.DeviceProfile == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to set color. DeviceProfile is null!")
		return
	}

	if d.DeviceProfile.RgbOff {
		for _, zoneColor := range d.DeviceProfile.ZoneColors {
			zoneColorIndexRange := zoneColor.ColorIndex
			for key, zoneColorIndex := range zoneColorIndexRange {
				switch key {
				case 0: // Red
					buf[zoneColorIndex] = 0x00
				case 1: // Green
					buf[zoneColorIndex] = 0x00
				case 2: // Blue
					buf[zoneColorIndex] = 0x00
				}
			}
		}
		d.writeColor(buf)
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
}

// writeColor will write color data to the device
func (d *Device) writeColor(data []byte) {
	if d.Exit {
		return
	}

	m := 1
	buffer := make([]byte, bufferSize)
	buffer[0] = 0x06 // Header len

	for i := 0; i < 3; i++ {
		buffer[m] = dataTypeColors[i]
		buffer[m+1] = data[i]
		buffer[m+2] = dataTypeColors[i+3]
		buffer[m+3] = data[i+3]
		m += 4
	}

	err := d.transfer(cmdWriteColor, buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
		return
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(command byte, data []byte) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	bufferW := make([]byte, bufferSize)

	bufferW[0] = command
	if len(data) > 0 {
		copy(bufferW[1:], data)
	}

	if _, err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return err
	}
	return nil
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

// NotifyMuteChanged will change mute status
func (d *Device) NotifyMuteChanged(value byte) {
	if value == 0 {
		d.setMicrophoneColor(false)
		d.MuteControl(false)
	} else {
		d.setMicrophoneColor(true)
		d.MuteControl(true)
	}
}

func (d *Device) MuteControl(muted bool) {
	mute, err := common.GetPulseAudioMuteStatus()
	if err == nil {
		if mute != muted {
			_, _ = common.MuteWithPulseAudioEx()
		}
	} else {
		mute, err = common.GetAlsaMuteStatus()
		if err == nil {
			if mute != muted {
				_, _ = common.MuteWithPulseAudioEx()
			}
		}
	}
}

// setMicrophoneColor will set microphone LED color
func (d *Device) setMicrophoneColor(muted bool) {
	buf := make([]byte, bufferSize)
	buf[0] = 0x01
	buf[1] = 0x1e
	if muted {
		buf[2] = 0xff
	}

	err := d.transfer(cmdWriteColor, buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
		return
	}
}

// ModifyBatteryLevel will modify battery level
func (d *Device) ModifyBatteryLevel(batteryLevel byte, charging byte) {
	if charging == 0x05 && !d.ResetColor {
		d.ResetColor = true
		d.setDeviceColor()
	} else if charging == 0x01 && d.ResetColor {
		d.ResetColor = false
		d.setDeviceColor()
	}

	d.BatteryLevel = uint16(batteryLevel)
	stats.UpdateBatteryStats(d.Serial, d.Product, uint16(batteryLevel), 2)
}

// SetDeviceFirmware will set device firmware
func (d *Device) SetDeviceFirmware(data []byte) {
	v1, v2 := fmt.Sprintf("%x", data[3]), fmt.Sprintf("%02x", data[4])
	d.Firmware = v1 + "." + v2
}
