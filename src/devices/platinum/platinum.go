package platinum

// Package: CORSAIR Platinum AIOs
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/openrgb"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/stats"
	"OpenLinkHub/src/temperatures"
	"OpenLinkHub/src/usb"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
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

type Shutdown struct {
	command byte
	data    []byte
}

type SpeedMode struct {
	Value   byte
	ZeroRpm bool
	Pump    bool
}

type DeviceProfile struct {
	Active             bool
	Path               string
	Product            string
	Serial             string
	Brightness         uint8
	BrightnessSlider   *uint8
	OriginalBrightness uint8
	RGBProfiles        map[int]string
	SpeedProfiles      map[int]string
	Labels             map[int]string
	OpenRGBIntegration bool
}

type DeviceList struct {
	Name      string
	Channel   int
	Index     int
	Type      byte
	Pump      bool
	Desc      string
	PumpModes map[byte]string
	HasSpeed  bool
	HasTemps  bool
}

type SupportedDevice struct {
	ProductId uint16 `json:"productId"`
	Product   string `json:"product"`
	Fans      uint8  `json:"fans"`
	FanLeds   uint8  `json:"fanLeds"`
	PumpLeds  uint8  `json:"pumpLeds"`
}

type Devices struct {
	ChannelId          int     `json:"channelId"`
	Channel            int     `json:"channel"`
	DeviceId           string  `json:"deviceId"`
	Type               byte    `json:"type"`
	Mode               byte    `json:"-"`
	Name               string  `json:"name"`
	Rpm                uint16  `json:"rpm"`
	Temperature        float64 `json:"temperature"`
	TemperatureString  string  `json:"temperatureString"`
	LedChannels        uint8   `json:"-"`
	ContainsPump       bool    `json:"-"`
	Description        string  `json:"description"`
	Profile            string  `json:"profile"`
	RGB                string  `json:"rgb"`
	Label              string  `json:"label"`
	PumpModes          map[byte]string
	HasSpeed           bool
	HasTemps           bool
	IsTemperatureProbe bool
}

type TemperatureProbe struct {
	ChannelId int
	Name      string
	Label     string
	Serial    string
	Product   string
}

type Device struct {
	dev               *usb.Device
	ProductId         uint16
	Manufacturer      string                    `json:"manufacturer"`
	Product           string                    `json:"product"`
	Serial            string                    `json:"serial"`
	Path              string                    `json:"path"`
	Firmware          string                    `json:"firmware"`
	RGB               string                    `json:"rgb"`
	Fans              int                       `json:"fans"`
	RequireActivation bool                      `json:"requireActivation"`
	AIO               bool                      `json:"aio"`
	Devices           map[int]*Devices          `json:"devices"`
	UserProfiles      map[string]*DeviceProfile `json:"userProfiles"`
	ActiveDevice      SupportedDevice
	activeRgb         *rgb.ActiveRGB
	sequence          byte
	DeviceProfile     *DeviceProfile
	TemperatureProbes *[]TemperatureProbe
	ExternalHub       bool
	RGBDeviceOnly     bool
	Template          string
	Brightness        map[int]string
	HasLCD            bool
	CpuTemp           float32
	GpuTemp           float32
	Rgb               *rgb.RGB
	rgbMutex          sync.RWMutex
	mutex             sync.Mutex
	deviceLock        sync.Mutex
	autoRefreshChan   chan struct{}
	speedRefreshChan  chan struct{}
	timer             *time.Ticker
	timerSpeed        *time.Ticker
	Exit              bool
	RGBModes          []string
	queue             chan []byte
	instance          *common.Device
}

var (
	pwd                        = ""
	cmdGetFirmware             = byte(0xaa)
	cmdGetLiquidTemperature    = byte(0xa9)
	cmdSetPumpSpeed            = byte(0x30)
	cmdGetPumpSpeed            = byte(0x31)
	cmdGetFanSpeed             = byte(0x41)
	cmdSetFanSpeed             = byte(0x42)
	cmdSetColor                = byte(0x62)
	cmdInitColor               = byte(0x61)
	cmdColorModeSoftware       = byte(0x01)
	cmdColorModeHardware       = byte(0x00)
	BufferSize                 = 64
	deviceRefreshInterval      = 1000
	temperaturePullingInterval = 3000
	rgbProfileUpgrade          = []string{"gradient", "pastelrainbow", "pastelspiralrainbow"}
	rgbModes                   = []string{
		"circle",
		"colorpulse",
		"colorshift",
		"colorwarp",
		"cpu-temperature",
		"gpu-temperature",
		"gradient",
		"liquid-temperature",
		"off",
		"rainbow",
		"pastelrainbow",
		"rotator",
		"static",
		"watercolor",
	}
	supportedDevices = []SupportedDevice{
		{ProductId: 3090, Product: "H150i PLATINUM", Fans: 3, FanLeds: 0, PumpLeds: 1},
		{ProductId: 3091, Product: "H115i PLATINUM", Fans: 2, FanLeds: 0, PumpLeds: 1},
		{ProductId: 3093, Product: "H100i PLATINUM", Fans: 2, FanLeds: 0, PumpLeds: 1},
	}
	deviceList = []DeviceList{
		{
			Name:      "Pump",
			Channel:   -1,
			Index:     0,
			Type:      0,
			Pump:      true,
			Desc:      "Pump",
			PumpModes: map[byte]string{},
			HasSpeed:  true,
			HasTemps:  true,
		},
		{
			Name:      "Fan 1",
			Channel:   0,
			Index:     1,
			Type:      1,
			Pump:      false,
			Desc:      "Fan",
			PumpModes: map[byte]string{},
			HasSpeed:  true,
		},
		{
			Name:      "Fan 2",
			Channel:   1,
			Index:     2,
			Type:      1,
			Pump:      false,
			Desc:      "Fan",
			PumpModes: map[byte]string{},
			HasSpeed:  true,
		},
		{
			Name:      "Fan 3",
			Channel:   2,
			Index:     3,
			Type:      1,
			Pump:      false,
			Desc:      "Fan",
			PumpModes: map[byte]string{},
			HasSpeed:  true,
		},
	}
)

func Init(vendorId, productId uint16, _, path string) *common.Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	// Open device, return if failure
	dev, err := usb.Open(path)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId}).Error("Unable to open HID device")
		return nil
	}

	// Set device control
	err = dev.SetDeviceControl(0x02, 0x02, 0, 0, 0)
	if err != nil {
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:      dev,
		AIO:      true,
		Path:     path,
		Template: "platinum.html",
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
		RGBModes:         rgbModes,
		autoRefreshChan:  make(chan struct{}),
		speedRefreshChan: make(chan struct{}),
		timer:            &time.Ticker{},
		timerSpeed:       &time.Ticker{},
	}

	d.ProductId = productId

	// Bootstrap
	d.getManufacturer()    // Manufacturer
	d.getProduct()         // Product
	d.getSerial()          // Serial
	d.loadRgb()            // Load RGB
	d.setFans()            // Number of fans
	d.loadDeviceProfiles() // Load all device profiles
	if d.getDeviceFirmware() == 1 {
		d.getPumpSpeed()
		d.getDevices()          // Get devices
		d.setAutoRefresh()      // Set auto device refresh
		d.getTemperatureProbe() // Devices with temperature probes
		d.saveDeviceProfile()   // Save profile
		d.initLeds()            // Init LED
		d.setDeviceColor()      // Device color
		if config.GetConfig().Manual {
			fmt.Println(
				fmt.Sprintf("[%s [%s]] Manual flag enabled. Process will not monitor temperature or adjust fan speed.", d.Serial, d.Product),
			)
		} else {
			d.updateDeviceSpeed()
		}
		d.setupOpenRGBController() // OpenRGB Controller
		d.createDevice()           // Device register
		d.startQueueWorker()       // Queue
		logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")
		return d.instance
	} else {
		logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Error("Unable to get device firmware.")
		return nil
	}
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypePlatinum,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-radiator.svg",
		Instance:    d,
		GetDevice:   d,
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

// GetDeviceTemplate will return device template name
func (d *Device) GetDeviceTemplate() string {
	return d.Template
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}

	d.timer.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if !config.GetConfig().Manual {
				d.timerSpeed.Stop()
				if d.speedRefreshChan != nil {
					close(d.speedRefreshChan)
				}
			}

			if d.autoRefreshChan != nil {
				close(d.autoRefreshChan)
			}

			if d.queue != nil {
				close(d.queue)
			}
		})
	}()

	d.transfer(cmdInitColor, []byte{cmdColorModeHardware})
	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
		}
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
}

// StopDirty will device in a dirty way
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
			if !config.GetConfig().Manual {
				d.timerSpeed.Stop()
				if d.speedRefreshChan != nil {
					close(d.speedRefreshChan)
				}
			}

			if d.autoRefreshChan != nil {
				close(d.autoRefreshChan)
			}

			if d.queue != nil {
				close(d.queue)
			}
		})
	}()
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 1
}

// UpdateDeviceLabel will set / update device label
func (d *Device) UpdateDeviceLabel(channelId int, label string) uint8 {
	if _, ok := d.Devices[channelId]; !ok {
		return 0
	}

	d.Devices[channelId].Label = label
	d.saveDeviceProfile()
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

// UpdateSpeedProfile will update device channel speed.
// If channelId is 0, all device channels will be updated
func (d *Device) UpdateSpeedProfile(channelId int, profile string) uint8 {
	// Check if the profile exists
	profiles := temperatures.GetTemperatureProfile(profile)
	if profiles == nil {
		return 0
	}

	// Avoid linear profile for this device
	if profiles.Linear {
		return 0
	}

	// If the profile is liquid temperature, check for the presence of AIOs
	if profiles.Sensor == temperatures.SensorTypeLiquidTemperature {
		valid := false
		for _, device := range d.Devices {
			if device.ContainsPump {
				valid = true
				break
			}
		}

		if !valid {
			return 2
		}
	}

	// Block PSU profile type
	if profiles.Sensor == temperatures.SensorTypePSU {
		return 6
	}

	if profiles.Sensor == temperatures.SensorTypeTemperatureProbe {
		if profiles.Device != d.Serial {
			return 3
		}

		if _, ok := d.Devices[profiles.ChannelId]; !ok {
			return 4
		}
	}

	// Check if actual channelId exists in the device list
	if _, ok := d.Devices[channelId]; ok {
		// Update channel with new profile
		d.Devices[channelId].Profile = profile
	}

	// Save to profile
	d.saveDeviceProfile()
	return 1
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
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}
	d.setDeviceColor()
	return 1
}

// UpdateRgbProfile will update device RGB profile
func (d *Device) UpdateRgbProfile(channelId int, profile string) uint8 {
	if d.GetRgbProfile(profile) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}
	hasPump := false

	for _, device := range d.Devices {
		if device.ContainsPump {
			hasPump = true
			break
		}
	}

	if profile == "liquid-temperature" {
		if !hasPump {
			logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Unable to apply liquid-temperature profile without a pump of AIO")
			return 2
		}
	}

	if _, ok := d.Devices[channelId]; ok {
		d.DeviceProfile.RGBProfiles[channelId] = profile // Set profile
		d.Devices[channelId].RGB = profile
	} else {
		return 0
	}

	d.DeviceProfile.RGBProfiles[channelId] = profile // Set profile
	d.saveDeviceProfile()                            // Save profile
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

		for _, device := range d.Devices {
			if device.LedChannels > 0 {
				d.Devices[device.ChannelId].RGB = profile.RGBProfiles[device.ChannelId]
			}
			if device.HasSpeed {
				d.Devices[device.ChannelId].Profile = profile.SpeedProfiles[device.ChannelId]
			}
			d.Devices[device.ChannelId].Label = profile.Labels[device.ChannelId]
		}

		newProfile := profile
		newProfile.Active = true
		d.DeviceProfile = newProfile
		d.saveDeviceProfile()
		d.setDeviceColor()

		if !config.GetConfig().Manual {
			d.timerSpeed.Stop()
			d.updateDeviceSpeed()
		}
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

// getTemperatureProbe will return all devices with a temperature probe
func (d *Device) getTemperatureProbe() {
	var probes []TemperatureProbe

	keys := make([]int, 0)
	for k := range d.Devices {
		keys = append(keys, k)
	}

	for _, k := range keys {
		if d.Devices[k].IsTemperatureProbe || d.Devices[k].HasTemps || d.Devices[k].ContainsPump {
			probe := TemperatureProbe{
				ChannelId: d.Devices[k].ChannelId,
				Name:      d.Devices[k].Name,
				Label:     d.Devices[k].Label,
				Serial:    d.Serial,
				Product:   d.Product,
			}
			probes = append(probes, probe)
		}
	}
	d.TemperatureProbes = &probes
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

// GetTemperatureProbes will return a list of temperature probes
func (d *Device) GetTemperatureProbes() *[]TemperatureProbe {
	return d.TemperatureProbes
}

// setupOpenRGBController will create RGBController object for OpenRGB Client Integration
func (d *Device) setupOpenRGBController() {
	lightChannels := 0
	keys := make([]int, 0)

	// For proper packet positioning
	for k := range d.Devices {
		lightChannels += int(d.Devices[k].LedChannels)
		keys = append(keys, k)
	}
	sort.Ints(keys)

	controller := &common.OpenRGBController{
		Name:         d.Product,
		Vendor:       "Corsair", // Static value
		Description:  "OpenLinkHub Backend Device",
		FwVersion:    d.Firmware,
		Serial:       d.Serial,
		Location:     d.Path,
		Zones:        nil,
		Colors:       make([]byte, lightChannels*3),
		ActiveMode:   0,
		WriteColorEx: d.writeColorEx,
		DeviceType:   common.DeviceTypeCooler,
		ColorMode:    common.ColorModePerLed,
	}

	for _, k := range keys {
		if d.Devices[k].LedChannels > 0 {
			zone := common.OpenRGBZone{
				Name:    d.Devices[k].Name,
				NumLEDs: uint32(d.Devices[k].LedChannels),
			}
			controller.Zones = append(controller.Zones, zone)
		}
	}
	openrgb.AddDeviceController(controller)
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	// Release existing queue
	d.clearQueue()

	// Reset
	reset := map[int][]byte{}
	var buffer []byte

	// Reset all channels
	color := &rgb.Color{
		Red:        0,
		Green:      0,
		Blue:       0,
		Brightness: 0,
	}

	// Get the number of LED channels we have
	lightChannels := 0
	m := 0
	for _, device := range d.Devices {
		lightChannels += int(device.LedChannels)
	}

	// Do we have any RGB component in the system?
	if lightChannels == 0 {
		logger.Log(logger.Fields{}).Info("No RGB compatible devices found")
		return
	}

	if lightChannels > 0 {
		for i := 0; i < lightChannels; i++ {
			reset[i] = []byte{
				byte(color.Red),
				byte(color.Green),
				byte(color.Blue),
			}
		}
		m++
	}

	buffer = rgb.SetColor(reset)
	d.transfer(cmdSetColor, buffer)

	// OpenRGB
	if d.DeviceProfile.OpenRGBIntegration {
		logger.Log(logger.Fields{}).Info("Exiting setDeviceColor() due to OpenRGB client")
		return
	}

	go func(lightChannels int) {
		startTime := time.Now()
		d.activeRgb = rgb.Exit()

		// Generate random colors
		d.activeRgb.RGBStartColor = rgb.GenerateRandomColor(1)
		d.activeRgb.RGBEndColor = rgb.GenerateRandomColor(1)

		keys := make([]int, 0)

		for k := range d.Devices {
			keys = append(keys, k)
		}
		sort.Ints(keys)

		for {
			select {
			case <-d.activeRgb.Exit:
				return
			default:
				buff := make([]byte, 0)
				for _, k := range keys {
					rgbCustomColor := true
					profile := d.GetRgbProfile(d.Devices[k].RGB)
					if profile == nil {
						for i := 0; i < int(d.Devices[k].LedChannels); i++ {
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
						int(d.Devices[k].LedChannels),
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
					r.ChannelId = k

					switch d.Devices[k].RGB {
					case "off":
						{
							for n := 0; n < int(d.Devices[k].LedChannels); n++ {
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
					case "liquid-temperature":
						{
							r.MinTemp = profile.MinTemp
							r.MaxTemp = profile.MaxTemp
							r.Temperature(float64(d.getLiquidTemperature()))
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
				}

				d.writeColor(buff)
				time.Sleep(40 * time.Millisecond)
			}
		}
	}(lightChannels)
}

// writeColor will write color data to the device
func (d *Device) writeColor(data []byte) {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()
	d.transfer(cmdSetColor, data)
}

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

// getSupportedDevice will return supported device or nil pointer
func (d *Device) getSupportedDevice(productId uint16) *SupportedDevice {
	for _, device := range supportedDevices {
		if device.ProductId == productId {
			return &device
		}
	}
	return nil
}

// getManufacturer will return device manufacturer
func (d *Device) getManufacturer() {
	d.Manufacturer = d.dev.GetMfrStr()
}

// setFans will set number of fans
func (d *Device) setFans() {
	product := d.getSupportedDevice(d.ProductId)
	if product == nil {
		d.Fans = 0
	} else {
		d.Fans = int(product.Fans)
	}
}

// getProduct will set product name
func (d *Device) getProduct() {
	product := d.getSupportedDevice(d.ProductId)
	if product == nil {
		d.Product = d.dev.GetProductStr()
	} else {
		d.Product = product.Product
	}
}

// getSerial will set the device serial number.
// In case of no serial, productId will be placed as serial number
func (d *Device) getSerial() {
	d.Serial = d.dev.GetSerialNbr()
	if len(d.Serial) == 0 {
		d.Serial = strconv.Itoa(int(d.ProductId))
	}
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() uint8 {
	response := d.transfer(cmdGetFirmware, nil)
	if response == nil {
		logger.Log(logger.Fields{}).Error("Unable to get device firmware")
	}

	if cmdGetFirmware == response[0] {
		d.Firmware = fmt.Sprintf("%d.%d.%d.%d", response[3], response[4], response[5], response[6])
		return 1
	} else {
		logger.Log(logger.Fields{"expected": cmdGetFirmware, "received": response[0]}).Error("Received unexpected response")
		return 0
	}
}

// initLeds will initialize LED channels
func (d *Device) initLeds() {
	d.transfer(cmdInitColor, []byte{cmdColorModeSoftware})
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
			}
		}
	}
}

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	var defaultBrightness = uint8(100)
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"

	speedProfiles := make(map[int]string, len(d.Devices))
	rgbProfiles := make(map[int]string, len(d.Devices))
	labels := make(map[int]string, len(d.Devices))

	for _, device := range d.Devices {
		speedProfiles[device.ChannelId] = device.Profile
		labels[device.ChannelId] = device.Label
		if device.LedChannels > 0 {
			rgbProfiles[device.ChannelId] = device.RGB
		}
	}
	deviceProfile := &DeviceProfile{
		Product:            d.Product,
		Serial:             d.Serial,
		SpeedProfiles:      speedProfiles,
		RGBProfiles:        rgbProfiles,
		Labels:             labels,
		Path:               profilePath,
		BrightnessSlider:   &defaultBrightness,
		OriginalBrightness: 100,
	}

	if d.DeviceProfile == nil {
		for _, device := range d.Devices {
			if device.LedChannels > 0 {
				rgbProfiles[device.ChannelId] = "static"
			}
			labels[device.ChannelId] = "Set Label"
		}
		deviceProfile.Active = true
		d.DeviceProfile = deviceProfile
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
		if len(d.DeviceProfile.Path) < 1 {
			deviceProfile.Path = profilePath
			d.DeviceProfile.Path = profilePath
		} else {
			deviceProfile.Path = d.DeviceProfile.Path
		}
		deviceProfile.OpenRGBIntegration = d.DeviceProfile.OpenRGBIntegration
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

// getDevices will fetch all devices connected to a hub
func (d *Device) getDevices() int {
	var devices = make(map[int]*Devices)

	for device := range deviceList {
		if deviceList[device].Index > d.Fans {
			// Depending on AIO type, skip last fan in an array
			continue
		}

		// Get a persistent speed profile. Fallback to Normal is anything fails
		speedProfile := "Normal"
		label := "Set Label"
		speedMode := &SpeedMode{
			ZeroRpm: false,
			Pump:    deviceList[device].Pump,
			Value:   70,
		}

		if d.DeviceProfile != nil {
			// Profile is set
			if sp, ok := d.DeviceProfile.SpeedProfiles[deviceList[device].Index]; ok {
				// Profile device channel exists
				if temperatures.GetTemperatureProfile(sp) != nil { // Speed profile exists in configuration
					// Speed profile exists in configuration
					speedProfile = sp
				} else {
					logger.Log(logger.Fields{"serial": d.Serial, "profile": sp}).Warn("Tried to apply non-existing profile")
				}
			} else {
				logger.Log(logger.Fields{"serial": d.Serial, "profile": sp}).Warn("Tried to apply non-existing channel")
			}
			// Device label
			if lb, ok := d.DeviceProfile.Labels[deviceList[device].Index]; ok {
				if len(lb) > 0 {
					label = lb
				}
			}
		} else {
			logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
		}

		rgbProfile := "static"
		var ledChannels uint8 = 0
		// LED channels
		supportedDevice := d.getSupportedDevice(d.ProductId)
		if supportedDevice != nil {
			if deviceList[device].Pump {
				speedMode.Value = 1
				ledChannels = supportedDevice.PumpLeds
			} else {
				ledChannels = supportedDevice.FanLeds
			}
		} else {
			ledChannels = 1
		}

		if ledChannels > 0 {
			// Get a persistent speed profile. Fallback to Normal is anything fails
			if d.DeviceProfile != nil {
				// Profile is set
				if rp, ok := d.DeviceProfile.RGBProfiles[deviceList[device].Index]; ok {
					// Profile device channel exists
					if d.GetRgbProfile(rp) != nil { // Speed profile exists in configuration
						// Speed profile exists in configuration
						rgbProfile = rp
					} else {
						logger.Log(logger.Fields{"serial": d.Serial, "profile": rp}).Warn("Tried to apply non-existing rgb profile")
					}
				} else {
					logger.Log(logger.Fields{"serial": d.Serial, "profile": rp}).Warn("Tried to apply rgb profile to the non-existing channel")
				}
			} else {
				logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
			}
		}

		var rpm uint16 = 0
		temp := 0.0
		if deviceList[device].Pump {
			rpm = d.getPumpSpeed()
			temp = d.getTemperature()
		} else {
			rpm = d.getFanSpeed(deviceList[device].Channel)
		}

		// Device object
		dev := &Devices{
			ChannelId:         deviceList[device].Index,
			Channel:           deviceList[device].Channel,
			Type:              deviceList[device].Type,
			DeviceId:          fmt.Sprintf("%s-%v", deviceList[device].Desc, deviceList[device].Index),
			Mode:              0,
			Name:              deviceList[device].Name,
			Rpm:               rpm,
			Temperature:       temp,
			TemperatureString: dashboard.GetDashboard().TemperatureToString(float32(temp)),
			LedChannels:       ledChannels,
			ContainsPump:      deviceList[device].Pump,
			Description:       deviceList[device].Desc,
			Profile:           speedProfile,
			PumpModes:         deviceList[device].PumpModes,
			HasSpeed:          deviceList[device].HasSpeed,
			HasTemps:          deviceList[device].HasTemps,
			RGB:               rgbProfile,
			Label:             label,
		}

		// Add to array
		devices[deviceList[device].Index] = dev
	}

	d.Devices = devices
	return len(devices)
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

// getFanSpeed will return fan speed based on channel index
func (d *Device) getFanSpeed(index int) uint16 {
	data := make([]byte, 1)
	data[0] = byte(index)
	buf := d.transfer(cmdGetFanSpeed, data)
	return binary.BigEndian.Uint16(buf[4:6])
}

// getPumpSpeed will return pump speed
func (d *Device) getPumpSpeed() uint16 {
	buf := d.transfer(cmdGetPumpSpeed, nil)
	return binary.BigEndian.Uint16(buf[3:])
}

// getTemperature will return AIO liquid temperature
func (d *Device) getTemperature() float64 {
	buf := d.transfer(cmdGetLiquidTemperature, nil)
	temp := buf[3]
	addon := buf[4]
	addonData := 0.0

	if float64(temp) < 0.0 {
		addonData = -0.1
	} else {
		addonData = 0.1
	}
	return float64(temp) + float64(addon)*addonData
}

// getDeviceData will get device data
func (d *Device) getDeviceData() {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if d.Exit {
		return
	}

	for device := range deviceList {
		var rpm uint16 = 0
		temp := 0.0
		if deviceList[device].Pump {
			rpm = d.getPumpSpeed()
			temp = d.getTemperature()
		} else {
			rpm = d.getFanSpeed(deviceList[device].Channel)
		}

		// Update
		if _, ok := d.Devices[deviceList[device].Index]; ok {
			if rpm > 0 {
				d.Devices[deviceList[device].Index].Rpm = rpm
			}

			if temp > 0 {
				d.Devices[deviceList[device].Index].Temperature = temp
				d.Devices[deviceList[device].Index].TemperatureString = dashboard.GetDashboard().TemperatureToString(float32(temp))
			}

			rpmString := fmt.Sprintf("%v RPM", d.Devices[deviceList[device].Index].Rpm)

			stats.UpdateAIOStats(
				d.Serial,
				d.Devices[deviceList[device].Index].Name,
				d.Devices[deviceList[device].Index].TemperatureString,
				rpmString,
				d.Devices[deviceList[device].Index].Label,
				deviceList[device].Index,
				float32(d.Devices[deviceList[device].Index].Temperature),
			)
		}
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
				d.getDeviceData()
			case <-d.autoRefreshChan:
				d.timer.Stop()
				return
			}
		}
	}()
}

// setSpeed will modify device speed
func (d *Device) setSpeed(data map[int]*SpeedMode) {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if d.Exit {
		return
	}
	for key, value := range data {
		if value.Pump {
			d.transfer(cmdSetPumpSpeed, []byte{value.Value})
		} else {
			d.transfer(cmdSetFanSpeed, []byte{byte(key), value.Value})
		}
	}
}

// ResetSpeedProfiles will reset channel speed profile if it matches with the current speed profile
// This is used when speed profile is deleted from the UI
func (d *Device) ResetSpeedProfiles(profile string) {
	i := 0
	for _, device := range d.Devices {
		if device.ChannelId != 0 {
			if device.HasSpeed {
				if device.Profile == profile {
					d.Devices[device.ChannelId].Profile = "Normal"
					i++
				}
			}
		}
	}

	if i > 0 {
		// Save only if something was changed
		d.saveDeviceProfile()
	}
}

// getLiquidTemperature will fetch temperature from AIO device
func (d *Device) getLiquidTemperature() float32 {
	for _, device := range d.Devices {
		if device.ChannelId == 0 {
			return float32(device.Temperature)
		}
	}
	return 0
}

// getPumpMode will return byte pump mode based on a profile name
func (d *Device) getPumpMode(index int, profile string) byte {
	for device := range deviceList {
		if deviceList[device].Index == index {
			for pumpMode, modeName := range deviceList[device].PumpModes {
				if modeName == profile {
					return pumpMode
				}
			}
		}
	}
	return 0
}

// ProcessSetOpenRgbIntegration will update OpenRGB integration status
func (d *Device) ProcessSetOpenRgbIntegration(enabled bool) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	d.DeviceProfile.OpenRGBIntegration = enabled
	d.saveDeviceProfile() // Save profile
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true
		d.activeRgb = nil
	}
	d.setDeviceColor()
	return 1
}

// updateDeviceSpeed will update device speed based on a temperature reading
func (d *Device) updateDeviceSpeed() {
	d.timerSpeed = time.NewTicker(time.Duration(temperaturePullingInterval) * time.Millisecond)
	tmp := make(map[int]string)
	channelSpeeds := make(map[int]*SpeedMode, len(d.Devices))
	var change = false

	go func() {
		for {
			select {
			case <-d.timerSpeed.C:
				var temp float32 = 0
				for _, device := range d.Devices {
					profiles := temperatures.GetTemperatureProfile(device.Profile)
					if profiles == nil {
						// No such profile, default to Normal
						profiles = temperatures.GetTemperatureProfile("Normal")
					}

					switch profiles.Sensor {
					case temperatures.SensorTypeGPU:
						{
							temp = temperatures.GetNVIDIAGpuTemperature(0)
							if temp == 0 {
								temp = temperatures.GetAMDGpuTemperature()
								if temp == 0 {
									logger.Log(logger.Fields{"temperature": temp}).Warn("Unable to get sensor temperature. Going to fallback to CPU")
									temp = temperatures.GetCpuTemperature()
								}
							}
						}
					case temperatures.SensorTypeCPU:
						{
							temp = temperatures.GetCpuTemperature()
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial}).Warn("Unable to get CPU temperature.")
							}
						}
					case temperatures.SensorTypeLiquidTemperature:
						{
							temp = d.getLiquidTemperature()
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial}).Warn("Unable to get liquid temperature.")
							}
						}
					case temperatures.SensorTypeCpuGpu:
						{
							cpuTemp := temperatures.GetCpuTemperature()
							gpuTemp := temperatures.GetNVIDIAGpuTemperature(0)
							if gpuTemp == 0 {
								gpuTemp = temperatures.GetAMDGpuTemperature()
							}

							if gpuTemp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "channelId": profiles.ChannelId, "cpu": cpuTemp, "gpu": gpuTemp}).Warn("Unable to get GPU temperature. Setting to 50")
								gpuTemp = 50
							}

							temp = float32(math.Max(float64(cpuTemp), float64(gpuTemp)))
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "channelId": profiles.ChannelId, "cpu": cpuTemp, "gpu": gpuTemp}).Warn("Unable to get maximum temperature value out of 2 numbers.")
							}
						}
					case temperatures.SensorTypeExternalHwMon:
						{
							temp = temperatures.GetHwMonTemperature(profiles.Device)
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "hwmonDeviceId": profiles.Device}).Warn("Unable to get hwmon temperature.")
							}
						}
					case temperatures.SensorTypeExternalExecutable:
						{
							temp = temperatures.GetExternalBinaryTemperature(profiles.Device)
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "binary": profiles.Device}).Warn("Unable to get temperature from binary.")
							}
						}
					case temperatures.SensorTypeMultiGPU:
						{
							temp = temperatures.GetGpuTemperatureIndex(int(profiles.GPUIndex))
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "hwmonDeviceId": profiles.Device}).Warn("Unable to get hwmon temperature.")
							}
						}
					case temperatures.SensorTypeGlobalTemperature:
						{
							temp = stats.GetDeviceTemperature(profiles.Device, profiles.ChannelId)
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "hwmonDeviceId": profiles.Device}).Warn("Unable to get hwmon temperature.")
							}
						}
					}

					// All temps failed, default to 50
					if temp == 0 {
						temp = 50
					}

					if config.GetConfig().GraphProfiles {
						pumpValue := temperatures.Interpolate(profiles.Points[0], temp)
						fansValue := temperatures.Interpolate(profiles.Points[1], temp)

						pump := int(math.Round(float64(pumpValue)))
						fans := int(math.Round(float64(fansValue)))

						// Failsafe
						if fans < 20 {
							fans = 20
						}
						if device.ContainsPump {
							if pump < 50 {
								pump = 70
							}
						} else {
							if pump < 20 {
								pump = 30
							}
						}
						if pump > 100 {
							pump = 100
						}
						if fans > 100 {
							fans = 100
						}

						cp := fmt.Sprintf("%s-%d-%f", device.Profile, device.ChannelId, temp)
						if ok := tmp[device.ChannelId]; ok != cp {
							tmp[device.ChannelId] = cp
							speedMode := &SpeedMode{}
							speedMode.ZeroRpm = profiles.ZeroRpm
							speedMode.Value = byte(fans)
							speedMode.Pump = false

							if device.ContainsPump {
								speedMode.Value = byte(pump)
								speedMode.Pump = true
							} else {
								speedMode.Value = byte(fans)
							}
							channelSpeeds[device.Channel] = speedMode
							change = true
						}
					} else {
						for i := 0; i < len(profiles.Profiles); i++ {
							profile := profiles.Profiles[i]
							minimum := profile.Min + 0.1
							if common.InBetween(temp, minimum, profile.Max) {
								cp := fmt.Sprintf("%s-%d-%d-%d-%d", device.Profile, device.ChannelId, profile.Id, profile.Fans, profile.Pump)
								if ok := tmp[device.ChannelId]; ok != cp {
									tmp[device.ChannelId] = cp

									// Validation
									if profile.Mode < 0 || profile.Mode > 1 {
										profile.Mode = 0
									}

									if profile.Pump < 50 {
										profile.Pump = 50
									}

									if profile.Pump > 100 {
										profile.Pump = 100
									}

									speedMode := &SpeedMode{}
									speedMode.ZeroRpm = profiles.ZeroRpm
									speedMode.Value = byte(profile.Fans)
									speedMode.Pump = false

									if device.ContainsPump {
										speedMode.Value = byte(profile.Pump)
										speedMode.Pump = true
									} else {
										speedMode.Value = byte(profile.Fans)
									}
									channelSpeeds[device.Channel] = speedMode
									change = true
								}
							}
						}
					}
				}
				if change {
					change = false
					d.setSpeed(channelSpeeds)
				}
			case <-d.speedRefreshChan:
				d.timerSpeed.Stop()
				return
			}
		}
	}()
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

			if d.Exit {
				d.deviceLock.Unlock()
				return
			}

			d.transfer(cmdSetColor, data)
			d.deviceLock.Unlock()
			time.Sleep(20 * time.Millisecond)
		}
	}()
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(command byte, data []byte) []byte {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	bufferW := make([]byte, len(data)+1)

	bufferW[0] = command
	if len(data) > 0 {
		copy(bufferW[1:], data)
	}

	bufferR := make([]byte, BufferSize)

	// Check if something can be read before new write
	if err := d.dev.ReadNonBlock(bufferR, 5); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read data from device")
		return bufferR
	}

	// Write data to the device
	if err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
		return bufferR
	}

	// Get data from device device
	if err := d.dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read data from device")
		return bufferR
	}

	return bufferR
}
