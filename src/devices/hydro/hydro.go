package hydro

// Package: CORSAIR Hydro AIOs
// This is the primary package for CORSAIR Hydro AIO devices.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/stats"
	"OpenLinkHub/src/temperatures"
	"OpenLinkHub/src/usb"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DeviceDataObject struct {
	LiquidTemperature float64
	PumpSpeed         uint16
	FanSpeed          uint16
}

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
	ProductId   uint16 `json:"productId"`
	Product     string `json:"product"`
	Fans        uint8  `json:"fans"`
	FanLeds     uint8  `json:"fanLeds"`
	PumpLeds    uint8  `json:"pumpLeds"`
	EndpointIn  int    `json:"endpointIn"`
	EndpointOut int    `json:"endpointOut"`
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
	instance          *common.Device
}

var (
	pwd                        = ""
	cmdSetFanSpeed             = byte(0x11)
	cmdSetPumpSpeed            = byte(0x13)
	cmdSetConfiguration        = byte(0x10)
	cmdGetDeviceData           = byte(0x20)
	BufferSize                 = 64
	deviceRefreshInterval      = 2000
	temperaturePullingInterval = 3000
	rgbModes                   = []string{
		"static",
	}
	supportedDevices = []SupportedDevice{
		// 2 physical fans, 1 logical via daisy chain
		{ProductId: 3080, Product: "H80i Hydro", Fans: 1, FanLeds: 0, PumpLeds: 1, EndpointIn: 0x82, EndpointOut: 0x02},
		{ProductId: 3081, Product: "H100i Hydro", Fans: 1, FanLeds: 0, PumpLeds: 1, EndpointIn: 0x82, EndpointOut: 0x02},
		{ProductId: 3082, Product: "H115i Hydro", Fans: 1, FanLeds: 0, PumpLeds: 1, EndpointIn: 0x82, EndpointOut: 0x02},
	}
	deviceList = []DeviceList{
		{
			Name:    "Pump",
			Channel: -1,
			Index:   0,
			Type:    0,
			Pump:    true,
			Desc:    "Pump",
			PumpModes: map[byte]string{
				40: "Quiet",
				66: "Performance",
			},
			HasSpeed: true,
			HasTemps: true,
		},
		{
			Name:      "Fans",
			Channel:   0,
			Index:     1,
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
		Template: "hydro.html",
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
		ProductId:        productId,
	}

	supportedDevice := d.getSupportedDevice(d.ProductId)
	if supportedDevice == nil {
		return nil
	}
	dev.SetEndpoints(supportedDevice.EndpointOut, supportedDevice.EndpointIn)

	// Bootstrap
	d.getManufacturer()     // Manufacturer
	d.getProduct()          // Product
	d.getSerial()           // Serial
	d.loadRgb()             // Load RGB
	d.setFans()             // Number of fans
	d.loadDeviceProfiles()  // Load all device profiles
	d.getDeviceFirmware()   // Firmware
	d.getDevices()          // Get devices
	d.setAutoRefresh()      // Set auto device refresh
	d.getTemperatureProbe() // Devices with temperature probes
	d.saveDeviceProfile()   // Save profile
	d.setConfiguration()    // Configuration
	if config.GetConfig().Manual {
		fmt.Println(
			fmt.Sprintf("[%s [%s]] Manual flag enabled. Process will not monitor temperature or adjust fan speed.", d.Serial, d.Product),
		)
	} else {
		d.updateDeviceSpeed() // Update device speed
	}
	d.createDevice() // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeHydro,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-device.svg",
		Instance:    d,
		GetDevice:   d,
	}
}

// GetRgbProfiles will return RGB profiles for a target device
func (d *Device) GetRgbProfiles() interface{} {
	return d.Rgb
}

// GetDeviceTemplate will return device template name
func (d *Device) GetDeviceTemplate() string {
	return d.Template
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")
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
		})
	}()

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

// UpdateSpeedProfile will update device channel speed.
func (d *Device) UpdateSpeedProfile(channelId int, profile string) uint8 {
	// Check if the profile exists
	profiles := temperatures.GetTemperatureProfile(profile)
	if profiles == nil {
		return 0
	}

	// If the profile is liquid temperature, check for the presence of AIOs
	if profiles.Sensor == temperatures.SensorTypeLiquidTemperature {
		valid := false
		for _, device := range d.Devices {
			if device.ChannelId == 0 { // Pump
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

	// Check if actual channelId exists in the device list
	if _, ok := d.Devices[channelId]; ok {
		d.Devices[channelId].Profile = profile
	}

	d.saveDeviceProfile()
	return 1
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
	d.setConfiguration() // Restart RGB
	return 1
}

// ChangeDeviceBrightnessValue will change device brightness via slider
func (d *Device) ChangeDeviceBrightnessValue(value uint8) uint8 {
	if value < 0 || value > 100 {
		return 0
	}

	d.DeviceProfile.BrightnessSlider = &value
	d.saveDeviceProfile()
	d.setConfiguration() // Restart RGB

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
	d.setConfiguration() // Restart RGB
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

// GetTemperatureProbes will return a list of temperature probes
func (d *Device) GetTemperatureProbes() *[]TemperatureProbe {
	return d.TemperatureProbes
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
func (d *Device) getSerial() {
	d.Serial = strconv.Itoa(int(d.ProductId))
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	buf := d.transfer(cmdGetDeviceData, nil)
	d.Firmware = fmt.Sprintf("%d.%d.%d", buf[4], buf[5], buf[6])
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

	// First save, assign saved profile to a device
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
		logger.Log(logger.Fields{"error": err, "location": deviceProfile.Path}).Fatal("Unable to close file handle")
	}
	d.loadDeviceProfiles() // Reload
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

		deviceData := d.getDeviceDataObject()

		if deviceList[device].Pump {
			rpm = deviceData.PumpSpeed
			temp = deviceData.LiquidTemperature
		} else {
			rpm = deviceData.FanSpeed
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

// getPumpSpeed will return pump speed
func (d *Device) getPumpSpeed() uint16 {
	buf := d.transfer(cmdGetDeviceData, nil)
	fmt.Println(fmt.Sprintf("% 2x", buf))

	fanRpm := binary.BigEndian.Uint16(buf[0:2])
	pumpRpm := binary.BigEndian.Uint16(buf[8:10])
	liquidTempCelsius := float32(buf[10]) + float32(buf[14])*0.1

	fmt.Println("Temperature: ", liquidTempCelsius)
	fmt.Println("Fan speed: ", fanRpm)
	fmt.Println("Pump speed: ", pumpRpm)

	d.Firmware = fmt.Sprintf("%d.%d.%d.%d", buf[3], buf[4], buf[5], buf[6])
	fmt.Println(d.Firmware)
	return binary.BigEndian.Uint16(buf[3:])
}

// getDeviceDataObject will get device data and return as DeviceDataObject
func (d *Device) getDeviceDataObject() *DeviceDataObject {
	buf := d.transfer(cmdGetDeviceData, nil)
	fanSpeed := binary.BigEndian.Uint16(buf[0:2])
	pumpSpeed := binary.BigEndian.Uint16(buf[8:10])
	temperature := float32(buf[10]) + float32(buf[14])*0.1

	data := &DeviceDataObject{
		LiquidTemperature: float64(temperature),
		PumpSpeed:         pumpSpeed,
		FanSpeed:          fanSpeed,
	}
	return data
}

// getDeviceData will get device data
func (d *Device) getDeviceData() {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if d.Exit {
		return
	}

	deviceData := d.getDeviceDataObject()
	for device := range deviceList {
		var rpm uint16 = 0
		temp := 0.0
		if deviceList[device].Pump {
			rpm = deviceData.PumpSpeed
			temp = deviceData.LiquidTemperature
		} else {
			rpm = deviceData.FanSpeed
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

// setPwdMode will set PWM mode of fans and set static color
func (d *Device) setConfiguration() {
	buf := make([]byte, 18)

	profile := d.GetRgbProfile("static")
	if profile == nil {
		// Set to white if profile fails
		buf[0] = 0xff // R
		buf[1] = 0xff // G
		buf[2] = 0xff // B
	} else {
		profile.StartColor.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
		profileColor := rgb.ModifyBrightness(profile.StartColor)
		buf[0] = byte(profileColor.Red)   // R
		buf[1] = byte(profileColor.Green) // G
		buf[2] = byte(profileColor.Blue)  // B
	}

	buf[4] = 0xff
	buf[5] = 0xff
	buf[6] = 0xff

	buf[9] = 0x2d
	buf[10] = 0x0a
	buf[11] = 0x05
	buf[12] = 0x01

	// PWM mode, 0x00 for 3pin mode. 3pin mode runs at 100%
	buf[17] = 0x01
	d.transfer(cmdSetConfiguration, buf)
}

// setSpeed will modify device speed
func (d *Device) setSpeed(data map[int]*SpeedMode) {
	//		    max			   fans
	// 11 00 00 64 00 00 00 00 50 50 00 00 00 00

	// pump mode
	// 13 28 - Quiet
	// 13 46 - Extreme
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	if d.Exit {
		return
	}
	for _, value := range data {
		if value.Pump {
			// Pump
			buf := make([]byte, 1)
			buf[0] = value.Value
			d.transfer(cmdSetPumpSpeed, buf)
		} else {
			// Fans
			buf := make([]byte, 13)
			buf[2] = 0x64 // Maximum %, 100
			buf[7] = value.Value
			buf[8] = value.Value
			d.transfer(cmdSetFanSpeed, buf)
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

					if device.ChannelId == 0 {
						cp := fmt.Sprintf("%s-%d", device.Profile, device.ChannelId)
						if ok := tmp[device.ChannelId]; ok != cp {
							tmp[device.ChannelId] = cp
							speedMode := &SpeedMode{}
							speedMode.Value = d.getPumpMode(device.ChannelId, device.Profile)
							speedMode.ZeroRpm = false
							speedMode.Pump = true
							channelSpeeds[device.ChannelId] = speedMode
							change = true
						}
					} else {
						if config.GetConfig().GraphProfiles {
							fansValue := temperatures.Interpolate(profiles.Points[1], temp)
							fans := int(math.Round(float64(fansValue)))

							// Failsafe
							if fans < 20 {
								fans = 20
							}
							if fans > 100 {
								fans = 100
							}
							cp := fmt.Sprintf("%s-%d-%f", device.Profile, device.ChannelId, temp)
							if ok := tmp[device.ChannelId]; ok != cp {
								speedMode := &SpeedMode{}
								tmp[device.ChannelId] = cp
								speedMode.ZeroRpm = profiles.ZeroRpm
								speedMode.Value = byte(fans)
								speedMode.Pump = false
								channelSpeeds[device.ChannelId] = speedMode
								change = true
							}
						} else {
							for i := 0; i < len(profiles.Profiles); i++ {
								profile := profiles.Profiles[i]
								minimum := profile.Min + 0.1
								if common.InBetween(temp, minimum, profile.Max) {
									cp := fmt.Sprintf("%s-%d-%d", device.Profile, device.ChannelId, profile.Fans)
									if ok := tmp[device.ChannelId]; ok != cp {
										speedMode := &SpeedMode{}
										tmp[device.ChannelId] = cp
										speedMode.ZeroRpm = profiles.ZeroRpm
										speedMode.Value = byte(profile.Fans)
										speedMode.Pump = false
										channelSpeeds[device.ChannelId] = speedMode
										change = true
									}
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

	if err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
		return bufferR
	}

	// Get data from a device
	if err := d.dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to read data from device")
		return bufferR
	}

	return bufferR
}
