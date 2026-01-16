package sabrergbproW

// Package: CORSAIR SABRE RGB PRO WIRELESS
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/cluster"
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/macro"
	"OpenLinkHub/src/openrgb"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/stats"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"math/bits"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
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
	PollingRate        int
	ZoneColors         map[int]ZoneColors
	Profiles           map[int]DPIProfile
	SleepMode          int
	AngleSnapping      int
	ButtonOptimization int
	LiftHeight         int
	KeyAssignmentHash  string
	OpenRGBIntegration bool
	RGBCluster         bool
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
	Path                  string `json:"path"`
	Firmware              string `json:"firmware"`
	activeRgb             *rgb.ActiveRGB
	UserProfiles          map[string]*DeviceProfile `json:"userProfiles"`
	ProfileOrder          []string                  `json:"profileOrder"`
	Devices               map[int]string            `json:"devices"`
	DeviceProfile         *DeviceProfile
	OriginalProfile       *DeviceProfile
	Template              string
	VendorId              uint16
	ProductId             uint16
	SlipstreamId          uint16
	Brightness            map[int]string
	SwitchModes           map[int]string
	KeyAssignmentTypes    map[int]string
	LEDChannels           int
	ChangeableLedChannels int
	CpuTemp               float32
	GpuTemp               float32
	Layouts               []string
	Rgb                   *rgb.RGB
	rgbMutex              sync.RWMutex
	Endpoint              byte
	SleepModes            map[int]string
	LiftHeights           map[int]string
	Connected             bool
	mutex                 sync.Mutex
	deviceLock            sync.Mutex
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
	queue                 chan []byte
	Usb                   bool
	MinDPI                int
	MaxDPI                int
	ZoneAmount            int
	DPIAmount             int
}

var (
	pwd                       = ""
	cmdSoftwareMode           = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode           = []byte{0x01, 0x03, 0x00, 0x01}
	cmdSleepMode              = []byte{0x01, 0x03, 0x00, 0x04}
	cmdGetFirmware            = []byte{0x02, 0x13}
	cmdWriteColor             = []byte{0x06, 0x00}
	cmdOpenEndpoint           = []byte{0x0d, 0x00, 0x01}
	cmdOpenSleepWriteEndpoint = []byte{0x01, 0x0d, 0x00, 0x01}
	cmdSetDpi                 = map[int][]byte{0: {0x01, 0x20, 0x00}}
	cmdSleep                  = map[int][]byte{0: {0x01, 0x37, 0x00}, 1: {0x01, 0x0e, 0x00}}
	cmdAngleSnapping          = []byte{0x01, 0x07, 0x00}
	cmdButtonOptimization     = []byte{0x01, 0xb0, 0x00}
	cmdLiftHeight             = []byte{0x01, 0x06, 0x00}
	cmdOpenWriteEndpoint      = []byte{0x0d, 0x01, 0x02}
	cmdWrite                  = []byte{0x06, 0x01}
	cmdCloseEndpoint          = []byte{0x05, 0x01, 0x01}
	cmdBatteryLevel           = []byte{0x02, 0x0f}
	bufferSize                = 64
	bufferSizeWrite           = bufferSize + 1
	headerSize                = 2
	headerWriteSize           = 4
	keyAmount                 = 7
	minDpiValue               = 100
	maxDpiValue               = 26000
	rgbProfileUpgrade         = []string{"gradient", "pastelrainbow", "pastelspiralrainbow"}
	rgbModes                  = []string{
		"colorpulse",
		"colorshift",
		"colorwarp",
		"cpu-temperature",
		"flickering",
		"gpu-temperature",
		"gradient",
		"mouse",
		"off",
		"rainbow",
		"pastelrainbow",
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
		Path:         strconv.Itoa(int(endpoint)),
		Template:     "sabrergbproW.html",
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
		Product: "SABRE RGB PRO",
		SleepModes: map[int]string{
			1:  "1 minute",
			5:  "5 minutes",
			10: "10 minutes",
			15: "15 minutes",
			30: "30 minutes",
			60: "1 hour",
		},
		LiftHeights: map[int]string{
			2: "Low",
			3: "Medium",
			4: "High",
			6: "Calibrated",
		},
		RGBModes:              rgbModes,
		LEDChannels:           4,
		ChangeableLedChannels: 2,
		KeyAssignmentTypes: map[int]string{
			0:  "None",
			1:  "Media Keys",
			2:  "DPI",
			3:  "Keyboard",
			8:  "Sniper",
			9:  "Mouse",
			10: "Macro",
			11: "Profile Switch",
		},
		InputActions:      inputmanager.GetInputActions(),
		keyAssignmentFile: "/database/key-assignments/sabrergbpro.json",
		MacroTracker:      make(map[int]uint16),
		MinDPI:            minDpiValue,
		MaxDPI:            maxDpiValue,
		ZoneAmount:        1,
		DPIAmount:         6,
	}

	d.getDebugMode()       // Debug mode
	d.loadRgb()            // Load RGB
	d.loadDeviceProfiles() // Load all device profiles
	d.saveDeviceProfile()  // Save profile
	d.loadKeyAssignments() // Key Assignments
	d.startQueueWorker()   // Queue
	return d
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
	if !value {
		cluster.Get().RemoveDeviceControllerBySerial(d.Serial)
		openrgb.RemoveDeviceControllerBySerial(d.Serial)
	}

	if d.activeRgb != nil {
		d.activeRgb.Exit <- true
	}
	d.Connected = value
}

// Connect will connect to a device
func (d *Device) Connect() {
	if !d.Connected {
		d.Connected = true
		d.getDeviceFirmware()      // Firmware
		d.setSoftwareMode()        // Activate software mode
		d.getBatterLevel()         // Battery level
		d.setLiftHeight()          // Lift Height
		d.setAngleSnapping()       // Angle snapping
		d.setButtonOptimization()  // Button optimization
		d.initLeds()               // Init LED ports
		d.setDeviceColor()         // Device color
		d.toggleDPI()              // DPI
		d.setupKeyAssignment()     // Setup key assignments
		d.setupOpenRGBController() // OpenRGB Controller
		d.setupClusterController() // RGB Cluster
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
		d.toggleDPI()
		d.loadKeyAssignments()
		d.setupKeyAssignment()
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

// rotateDeviceProfile will rotate and activate next user profile
func (d *Device) rotateDeviceProfile() {
	if d.DeviceProfile == nil || len(d.ProfileOrder) == 0 || len(d.UserProfiles) == 0 {
		return
	}

	var currentName string
	for name, profile := range d.UserProfiles {
		if profile.Active {
			currentName = name
			break
		}
	}

	if currentName == "" {
		next := d.ProfileOrder[0]
		d.ChangeDeviceProfile(next)
		return
	}
	idx := -1
	for i, name := range d.ProfileOrder {
		if name == currentName {
			idx = i
			break
		}
	}

	if idx == -1 {
		next := d.ProfileOrder[0]
		d.ChangeDeviceProfile(next)
		return
	}

	nextIdx := (idx + 1) % len(d.ProfileOrder)
	next := d.ProfileOrder[nextIdx]

	d.ChangeDeviceProfile(next)
	return
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

	if d.GetRgbProfile(profile) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}

	if d.DeviceProfile.RGBCluster {
		return 5
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

// setupOpenRGBController will create Cluster Controller for RGB Cluster
func (d *Device) setupClusterController() {
	if d.DeviceProfile == nil {
		return
	}

	if !d.DeviceProfile.RGBCluster {
		return
	}

	clusterController := &common.ClusterController{
		Product:      d.Product,
		Serial:       d.Serial,
		LedChannels:  uint32(d.ChangeableLedChannels),
		WriteColorEx: d.writeColorCluster,
	}

	cluster.Get().AddDeviceController(clusterController)
}

// setupOpenRGBController will create RGBController object for OpenRGB Client Integration
func (d *Device) setupOpenRGBController() {
	controller := &common.OpenRGBController{
		Name:         d.Product,
		Vendor:       "Corsair", // Static value
		Description:  "OpenLinkHub Backend Device",
		FwVersion:    d.Firmware,
		Serial:       d.Serial,
		Location:     fmt.Sprintf("Slipstream Ep: %s", d.Path), // Slipstream endpoint
		Zones:        nil,
		Colors:       make([]byte, d.ChangeableLedChannels*3),
		ActiveMode:   0,
		WriteColorEx: d.writeColorEx,
		DeviceType:   common.DeviceTypeMouse,
		ColorMode:    common.ColorModePerLed,
	}

	zone := []common.OpenRGBZone{
		{
			Name:     "Logo",
			NumLEDs:  uint32(1),
			ZoneType: common.ZoneTypeLinear,
		},
	}
	controller.Zones = zone
	openrgb.AddDeviceController(controller)
}

// ProcessSetOpenRgbIntegration will update OpenRGB integration status
func (d *Device) ProcessSetOpenRgbIntegration(enabled bool) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	if d.DeviceProfile.RGBCluster {
		return 2
	}

	d.clearQueue()
	d.DeviceProfile.OpenRGBIntegration = enabled
	d.saveDeviceProfile() // Save profile
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}
	d.setDeviceColor()
	return 1
}

// ProcessSetRgbCluster will update OpenRGB integration status
func (d *Device) ProcessSetRgbCluster(enabled bool) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	if d.DeviceProfile.OpenRGBIntegration {
		return 2
	}

	d.DeviceProfile.RGBCluster = enabled
	d.saveDeviceProfile() // Save profile
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}
	d.setDeviceColor()

	if enabled {
		clusterController := &common.ClusterController{
			Product:      d.Product,
			Serial:       d.Serial,
			LedChannels:  uint32(d.ChangeableLedChannels),
			WriteColorEx: d.writeColorCluster,
		}

		cluster.Get().AddDeviceController(clusterController)
	} else {
		cluster.Get().RemoveDeviceControllerBySerial(d.Serial)
	}
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

	if d.DeviceProfile.RGBProfile == "static" || d.DeviceProfile.RGBProfile == "mouse" {
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
	if d.DeviceProfile.RGBProfile == "static" || d.DeviceProfile.RGBProfile == "mouse" {
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

// SaveMouseZoneColors will save mouse zone colors
func (d *Device) SaveMouseZoneColors(dpi rgb.Color, zoneColors map[int]rgb.Color) uint8 {
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
	stats.UpdateBatteryStats(d.Serial, d.Product, d.BatteryLevel, 1)
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
	_, _ = d.transfer(cmdAngleSnapping, buf)
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
	_, _ = d.transfer(cmdButtonOptimization, buf)
}

// setLiftHeight will change mouse lift height
func (d *Device) setLiftHeight() {
	if d.DeviceProfile == nil {
		return
	}

	if d.DeviceProfile.LiftHeight < 2 || d.DeviceProfile.LiftHeight > 6 {
		return
	}

	buf := make([]byte, 1)
	buf[0] = byte(d.DeviceProfile.LiftHeight)
	_, _ = d.transfer(cmdLiftHeight, buf)
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
		deviceProfile.RGBProfile = "mouse"
		deviceProfile.Label = "Mouse"
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
		}
		deviceProfile.Profiles = map[int]DPIProfile{
			0: {
				Name:        "Stage 1",
				Value:       400,
				PackerIndex: 1,
				ColorIndex: map[int][]int{
					0: {1, 4, 7},
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
					0: {1, 4, 7},
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
					0: {1, 4, 7},
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
					0: {1, 4, 7},
				},
				Color: &rgb.Color{
					Red:        255,
					Green:      0,
					Blue:       255,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 255, 0, 255),
				},
			},
			4: {
				Name:        "Stage 5",
				Value:       3200,
				PackerIndex: 5,
				ColorIndex: map[int][]int{
					0: {1, 4, 7},
				},
				Color: &rgb.Color{
					Red:        0,
					Green:      255,
					Blue:       255,
					Brightness: 1,
					Hex:        fmt.Sprintf("#%02x%02x%02x", 0, 255, 255),
				},
			},
			5: {
				Name:        "Sniper",
				Value:       200,
				PackerIndex: 6,
				ColorIndex: map[int][]int{
					0: {1, 4, 7},
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
		deviceProfile.Profile = 1
		deviceProfile.SleepMode = 15
		deviceProfile.PollingRate = 4
		deviceProfile.LiftHeight = 2
	} else {
		if d.DeviceProfile.LiftHeight == 0 {
			deviceProfile.LiftHeight = 2
		} else {
			deviceProfile.LiftHeight = d.DeviceProfile.LiftHeight
		}

		// Upgrade DPI profile
		d.upgradeDpiProfiles()

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
		deviceProfile.ZoneColors = d.DeviceProfile.ZoneColors
		deviceProfile.AngleSnapping = d.DeviceProfile.AngleSnapping
		deviceProfile.ButtonOptimization = d.DeviceProfile.ButtonOptimization
		deviceProfile.KeyAssignmentHash = d.DeviceProfile.KeyAssignmentHash
		if len(d.DeviceProfile.Path) < 1 {
			deviceProfile.Path = profilePath
			d.DeviceProfile.Path = profilePath
		} else {
			deviceProfile.Path = d.DeviceProfile.Path
		}
		deviceProfile.OpenRGBIntegration = d.DeviceProfile.OpenRGBIntegration
		deviceProfile.RGBCluster = d.DeviceProfile.RGBCluster
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

// upgradeDpiProfiles will perform upgrade of DPI profiles in needed
func (d *Device) upgradeDpiProfiles() {
	found := false
	for _, profile := range d.DeviceProfile.Profiles {
		if strings.EqualFold(profile.Name, "Sniper") {
			found = true
			break
		}
	}

	if !found {
		index := len(d.DeviceProfile.Profiles)
		sniper := DPIProfile{
			Name:        "Sniper",
			Value:       200,
			PackerIndex: 6,
			ColorIndex: map[int][]int{
				0: {1, 4, 7},
			},
			Color: &rgb.Color{
				Red:        255,
				Green:      255,
				Blue:       0,
				Brightness: 1,
				Hex:        fmt.Sprintf("#%02x%02x%02x", 255, 255, 0),
			},
			Sniper: true,
		}
		d.DeviceProfile.Profiles[index] = sniper
	}
}

// UpdateDeviceKeyAssignment will update device key assignments
func (d *Device) UpdateDeviceKeyAssignment(keyIndex int, keyAssignment inputmanager.KeyAssignment) uint8 {
	if val, ok := d.KeyAssignment[keyIndex]; ok {
		val.Default = keyAssignment.Default
		val.ActionHold = keyAssignment.ActionHold
		val.ActionType = keyAssignment.ActionType
		val.ActionCommand = keyAssignment.ActionCommand
		val.IsMacro = keyAssignment.IsMacro
		val.OnRelease = keyAssignment.OnRelease
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

	if err := common.SaveJsonData(keyAssignmentsFile, d.KeyAssignment); err != nil {
		logger.Log(logger.Fields{"error": err, "location": keyAssignmentsFile}).Error("Unable to write key assignment data")
		return
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
			64: {
				Name:          "Profile Switch",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
				ProfileSwitch: true,
			},
			32: {
				Name:          "DPI Button",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
			},
			16: {
				Name:          "Back Button",
				Default:       true,
				ActionType:    0,
				ActionCommand: 0,
				ActionHold:    false,
			},
			8: {
				Name:          "Forward Button",
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

		if err := common.SaveJsonData(keyAssignmentsFile, keyAssignment); err != nil {
			logger.Log(logger.Fields{"error": err, "location": keyAssignmentsFile}).Error("Unable to save key assignments data")
			return
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
func (d *Device) setSleepTimer() uint8 {
	if d.Exit {
		return 0
	}
	if d.DeviceProfile != nil {
		changed := 0
		_, err := d.transfer(cmdOpenSleepWriteEndpoint, nil)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change device sleep timer")
			return 0
		}

		buf := make([]byte, 4)
		sleep := d.DeviceProfile.SleepMode * (60 * 1000)
		binary.LittleEndian.PutUint32(buf, uint32(sleep))

		for i := 0; i < 2; i++ {
			command := cmdSleep[i]
			_, err = d.transfer(command, buf)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Warn("Unable to change device sleep timer")
				continue
			}
			changed++
		}

		if changed > 0 {
			return 1
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
	d.rebuildProfileOrder()
	d.getDeviceProfile()
}

// rebuildProfileOrder will return profile order
func (d *Device) rebuildProfileOrder() {
	d.ProfileOrder = d.ProfileOrder[:0]
	for name := range d.UserProfiles {
		d.ProfileOrder = append(d.ProfileOrder, name)
	}
	sort.Strings(d.ProfileOrder)
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

// getSniperColor will get sniper dpi color
func (d *Device) getSniperColor() *rgb.Color {
	for _, val := range d.DeviceProfile.Profiles {
		if val.Sniper {
			return val.Color
		}
	}
	return nil
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	buf := make([]byte, d.LEDChannels*3)
	if d.DeviceProfile == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to set color. DeviceProfile is null!")
		return
	}

	// OpenRGB
	if d.DeviceProfile.OpenRGBIntegration {
		// DPI
		dpiColor := d.DeviceProfile.Profiles[d.DeviceProfile.Profile].Color
		if d.SniperMode {
			dpiColor = d.getSniperColor()
		}
		if dpiColor == nil {
			return
		}

		dpiColor.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
		dpiColor = rgb.ModifyBrightness(*dpiColor)

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

		zoneKeys := make([]int, 0, len(d.DeviceProfile.ZoneColors))
		for key := range d.DeviceProfile.ZoneColors {
			zoneKeys = append(zoneKeys, key)
		}
		sort.Ints(zoneKeys)

		m := 0
		for _, key := range zoneKeys {
			zoneColor := d.DeviceProfile.ZoneColors[key]
			for _, zoneColorIndex := range zoneColor.ColorIndex {
				buf[zoneColorIndex] = 0x00
				m++
			}
		}
		d.writeColor(buf)
		logger.Log(logger.Fields{}).Info("Exiting setDeviceColor() due to OpenRGB client")
		return
	}

	// RGB Cluster
	if d.DeviceProfile.RGBCluster {
		logger.Log(logger.Fields{}).Info("Exiting setDeviceColor() due to RGB Cluster")
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

	dpiColor.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
	dpiColor = rgb.ModifyBrightness(*dpiColor)

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

	if d.DeviceProfile.RGBProfile == "mouse" {
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
				case "pastelrainbow":
					{
						r.PastelRainbow(startTime)
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
				d.deviceLock.Lock()
				value := profile.Value

				// Send DPI packet
				if value < uint16(minDpiValue) {
					value = uint16(minDpiValue)
				}
				if value > uint16(maxDpiValue) {
					value = uint16(maxDpiValue)
				}

				buf := make([]byte, 2)
				binary.LittleEndian.PutUint16(buf[0:2], value)
				for i := 0; i <= 1; i++ {
					_, err := d.transfer(cmdSetDpi[i], buf)
					if err != nil {
						logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to set dpi")
					}
				}
				d.deviceLock.Unlock()

				if d.activeRgb != nil {
					d.activeRgb.Exit <- true
					d.activeRgb = nil
				}
				d.setDeviceColor()
			}
		}
	} else {
		// Reset to normal DPI mode
		d.toggleDPI()
	}
}

// toggleDPI will change DPI mode
func (d *Device) toggleDPI() {
	if d.DeviceProfile != nil {
		d.deviceLock.Lock()
		profile := d.DeviceProfile.Profiles[d.DeviceProfile.Profile]
		value := profile.Value

		// Send DPI packet
		if value < uint16(minDpiValue) {
			value = uint16(minDpiValue)
		}
		if value > uint16(maxDpiValue) {
			value = uint16(maxDpiValue)
		}

		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf[0:2], value)
		for i := 0; i <= 1; i++ {
			_, err := d.transfer(cmdSetDpi[i], buf)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to set dpi")
			}
		}
		d.deviceLock.Unlock()

		if d.activeRgb != nil {
			d.activeRgb.Exit <- true
			d.activeRgb = nil
		}
		d.setDeviceColor()
	}
}

// writeColorEx will write data to the device from OpenRGB client
func (d *Device) writeColorEx(data []byte, _ int) {
	if !d.DeviceProfile.OpenRGBIntegration {
		return
	}
	if d.Exit {
		return
	}

	// Copy data to avoid race conditions, since the caller might reuse the slice
	copyData := make([]byte, len(data))
	copy(copyData, data)

	select {
	case d.queue <- copyData:
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

// startQueueWorker will initialize queue system and control packet flow towards the device
func (d *Device) startQueueWorker() {
	d.queue = make(chan []byte, 10)

	go func() {
		for data := range d.queue {
			d.deviceLock.Lock()
			buf := make([]byte, d.LEDChannels*3)
			// DPI
			dpiColor := d.DeviceProfile.Profiles[d.DeviceProfile.Profile].Color
			if d.SniperMode {
				dpiColor = d.getSniperColor()
			}
			if dpiColor == nil {
				return
			}

			dpiColor.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
			dpiColor = rgb.ModifyBrightness(*dpiColor)

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

			zoneKeys := make([]int, 0, len(d.DeviceProfile.ZoneColors))
			for key := range d.DeviceProfile.ZoneColors {
				zoneKeys = append(zoneKeys, key)
			}
			sort.Ints(zoneKeys)

			m := 0
			for _, key := range zoneKeys {
				zoneColor := d.DeviceProfile.ZoneColors[key]
				for _, zoneColorIndex := range zoneColor.ColorIndex {
					if m >= len(data) {
						break
					}
					buf[zoneColorIndex] = data[m]
					m++
				}
			}

			buffer := make([]byte, len(buf)+headerWriteSize)
			binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(buf)))
			copy(buffer[headerWriteSize:], buf)

			_, err := d.transferWithTimeout(cmdWriteColor, buffer)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
			}
			d.deviceLock.Unlock()
			time.Sleep(20 * time.Millisecond)
		}
	}()
}

// writeColorCluster will write cluster color
func (d *Device) writeColorCluster(data []byte, _ int) {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if !d.DeviceProfile.RGBCluster {
		return
	}

	if d.Exit {
		return
	}

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

	dpiColor.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
	dpiColor = rgb.ModifyBrightness(*dpiColor)

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

	zoneKeys := make([]int, 0, len(d.DeviceProfile.ZoneColors))
	for key := range d.DeviceProfile.ZoneColors {
		zoneKeys = append(zoneKeys, key)
	}
	sort.Ints(zoneKeys)

	m := 0
	for _, key := range zoneKeys {
		zoneColor := d.DeviceProfile.ZoneColors[key]
		for _, zoneColorIndex := range zoneColor.ColorIndex {
			if m >= len(data) {
				break
			}
			buf[zoneColorIndex] = data[m]
			m++
		}
	}

	buffer := make([]byte, len(buf)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(buf)))
	copy(buffer[headerWriteSize:], buf)

	_, err := d.transferWithTimeout(cmdWriteColor, buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
	}
}

// writeColor will write color data to the device
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

// transferWithTimeout will send data to a device and retrieve device output with timeout
func (d *Device) transferWithTimeout(endpoint, buffer []byte) ([]byte, error) {
	if d.Exit {
		return nil, nil
	}

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

	if _, err := d.dev.ReadWithTimeout(bufferR, 100*time.Millisecond); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		return bufferR, err
	}
	return bufferR, nil
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

// writeKeyAssignmentData will write key assignment to the device.
func (d *Device) writeKeyAssignmentData(data []byte) {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if d.Exit {
		return
	}

	_, err := d.transfer(cmdOpenWriteEndpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Unable to open write endpoint")
		return
	}

	buffer := make([]byte, len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	copy(buffer[headerWriteSize:], data)
	_, err = d.transfer(cmdWrite, buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to data endpoint")
	}

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

		if val.OnRelease {
			isPressed, isReleased = isReleased, isPressed
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
			if val.Default && val.ProfileSwitch {
				d.rotateDeviceProfile()
				continue
			}

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
			case 11:
				d.rotateDeviceProfile()
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
