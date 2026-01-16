package st100

// Package: Corsair ST100
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/cluster"
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/led"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/temperatures"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

type Device struct {
	Debug           bool
	dev             *hid.Device
	Manufacturer    string `json:"manufacturer"`
	Product         string `json:"product"`
	Serial          string `json:"serial"`
	Firmware        string `json:"firmware"`
	activeRgb       *rgb.ActiveRGB
	ledProfile      *led.Device
	DeviceProfile   *DeviceProfile
	UserProfiles    map[string]*DeviceProfile `json:"userProfiles"`
	Brightness      map[int]string
	Template        string
	VendorId        uint16
	ProductId       uint16
	LEDChannels     int
	CpuTemp         float32
	GpuTemp         float32
	Rgb             *rgb.RGB
	rgbMutex        sync.RWMutex
	Exit            bool
	timer           *time.Ticker
	autoRefreshChan chan struct{}
	mutex           sync.Mutex
	RGBModes        []string
	instance        *common.Device
}

type ZoneColor struct {
	Color       *rgb.Color
	PacketIndex int
}

// DeviceProfile struct contains all device profile
type DeviceProfile struct {
	Active             bool
	Path               string
	Product            string
	Serial             string
	Brightness         uint8
	BrightnessSlider   *uint8
	OriginalBrightness uint8
	RGBProfile         string
	Label              string
	Stand              *Stand
	RGBCluster         bool
}

type Stand struct {
	Row map[int]Row `json:"row"`
}

type Row struct {
	Zones map[int]Zones `json:"zones"`
}

type Zones struct {
	Name        string    `json:"name"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Left        int       `json:"left"`
	Top         int       `json:"top"`
	PacketIndex []int     `json:"packetIndex"`
	Color       rgb.Color `json:"color"`
}

var (
	pwd                   = ""
	bufferSize            = 64
	bufferSizeWrite       = bufferSize + 1
	headerSize            = 2
	headerWriteSize       = 3
	deviceRefreshInterval = 1000
	cmdWrite              = byte(0x07)
	cmdRead               = byte(0x0e)
	cmdSoftwareMode       = []byte{0x04, 0x02}
	cmdHardwareMode       = []byte{0x04, 0x01}
	cmdGetFirmware        = []byte{0x01, 0x05}
	cmdWriteColor         = []byte{0x22, 0x14}
	colorPacketLength     = 28
	rgbProfileUpgrade     = []string{"gradient", "spiralrainbow", "pastelrainbow", "pastelspiralrainbow"}
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
		"off",
		"rainbow",
		"pastelrainbow",
		"rotator",
		"spinner",
		"spiralrainbow",
		"stand",
		"static",
		"storm",
		"watercolor",
		"wave",
	}
)

func Init(vendorId, productId uint16, serial, _ string) *common.Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	// Open device, return if failure
	dev, err := hid.Open(vendorId, productId, serial)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId, "serial": serial}).Error("Unable to open HID device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:       dev,
		VendorId:  vendorId,
		ProductId: productId,
		Product:   "ST100 RGB",
		Template:  "st100.html",
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
		RGBModes:        rgbModes,
		LEDChannels:     9,
		autoRefreshChan: make(chan struct{}),
		timer:           &time.Ticker{},
	}

	d.getDebugMode()           // Debug mode
	d.getManufacturer()        // Manufacturer
	d.getSerial()              // Serial
	d.loadRgb()                // Load RGB
	d.setSoftwareMode()        // Activate software mode
	d.getDeviceFirmware()      // Firmware
	d.loadDeviceProfiles()     // Load all device profiles
	d.saveDeviceProfile()      // Save profile
	d.setupLedProfile()        // LED profile
	d.setAutoRefresh()         // Set auto device refresh
	d.setDeviceColor()         // Device color
	d.setupClusterController() // RGB Cluster
	d.createDevice()           // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeST100,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-headphone-stand.svg",
		Instance:    d,
	}
}

// GetDeviceLedData will return led profiles as interface
func (d *Device) GetDeviceLedData() interface{} {
	return d.ledProfile
}

// getLedProfileColor will get RGB color based on channelId and ledId
func (d *Device) getLedProfileColor(channelId int, ledId int) *rgb.Color {
	if channels, ok := d.ledProfile.Devices[channelId]; ok {
		if color, found := channels.Channels[ledId]; found {
			return &color
		}
	}
	return nil
}

// setupLedProfile will init and load LED profile
func (d *Device) setupLedProfile() {
	d.ledProfile = led.LoadProfile(d.Serial)
	if d.ledProfile == nil {
		d.saveLedProfile()
		d.ledProfile = led.LoadProfile(d.Serial)
	}
}

// saveLedProfile will save new LED profile
func (d *Device) saveLedProfile() {
	// Default profile
	profile := d.GetRgbProfile("static")
	if profile == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Error("Unable to load static rgb profile")
		return
	}

	// Init
	device := led.Device{
		Serial:     d.Serial,
		DeviceName: d.Product,
	}

	devices := map[int]led.DeviceData{}

	for i := 0; i < d.LEDChannels; i++ {
		channels := map[int]rgb.Color{}
		deviceData := led.DeviceData{}
		deviceData.LedChannels = 1
		deviceData.Stand = true
		channels[0] = rgb.Color{
			Red:   0,
			Green: 255,
			Blue:  255,
			Hex:   fmt.Sprintf("#%02x%02x%02x", 0, 255, 255),
		}
		deviceData.Channels = channels
		devices[i] = deviceData
	}
	device.Devices = devices
	led.SaveProfile(d.Serial, device)
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
	return d.DeviceProfile.Stand
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

// UpdateDeviceMetrics will update device metrics
func (d *Device) UpdateDeviceMetrics() {
	header := &metrics.Header{
		Product:  d.Product,
		Serial:   d.Serial,
		Firmware: d.Firmware,
	}
	metrics.Populate(header)
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

// getManufacturer will return device manufacturer
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
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

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	_, err := d.transfer(cmdWrite, cmdSoftwareMode, nil, false)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// setHardwareMode will switch a device to hardware mode
func (d *Device) setHardwareMode() {
	_, err := d.transfer(cmdWrite, cmdHardwareMode, nil, false)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	v1, v2, v3 := 0, 0, 0
	for key, cmdFirmware := range cmdGetFirmware {
		fw, err := d.transfer(
			cmdRead,
			[]byte{cmdFirmware},
			nil,
			true,
		)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to a device")
		}
		switch key {
		case 0:
			v1, v2 = int(fw[9]), int(fw[8])
		case 1:
			v3 = int(fw[4])
		}
	}
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
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
		deviceProfile.RGBProfile = "stand"
		deviceProfile.Label = "Headset Stand"
		deviceProfile.Active = true
		deviceProfile.Stand = &Stand{
			Row: map[int]Row{
				1: {
					Zones: map[int]Zones{
						1: {Name: "Logo", Width: 150, Height: 30, Left: 170, Top: -21, PacketIndex: []int{12}, Color: rgb.Color{Red: 255, Green: 255, Blue: 0, Brightness: 1}},
					},
				},
				2: {
					Zones: map[int]Zones{
						2: {Name: "", Width: 150, Height: 150, Left: 0, Top: 20, PacketIndex: []int{24}, Color: rgb.Color{Red: 0, Green: 255, Blue: 255, Brightness: 1}},
						3: {Name: "", Width: 150, Height: 150, Left: 20, Top: 20, PacketIndex: []int{0}, Color: rgb.Color{Red: 0, Green: 255, Blue: 255, Brightness: 1}},
						4: {Name: "", Width: 150, Height: 150, Left: 20, Top: 20, PacketIndex: []int{3}, Color: rgb.Color{Red: 0, Green: 255, Blue: 255, Brightness: 1}},
					},
				},
				3: {
					map[int]Zones{
						5: {Name: "", Width: 150, Height: 150, Left: 0, Top: 20, PacketIndex: []int{21}, Color: rgb.Color{Red: 0, Green: 255, Blue: 255, Brightness: 1}},
						6: {Name: "", Width: 150, Height: 150, Left: 190, Top: 20, PacketIndex: []int{6}, Color: rgb.Color{Red: 0, Green: 255, Blue: 255, Brightness: 1}},
					},
				},
				4: {
					map[int]Zones{
						7: {Name: "", Width: 150, Height: 150, Left: 0, Top: 20, PacketIndex: []int{18}, Color: rgb.Color{Red: 0, Green: 255, Blue: 255, Brightness: 1}},
						8: {Name: "", Width: 150, Height: 150, Left: 20, Top: 20, PacketIndex: []int{15}, Color: rgb.Color{Red: 0, Green: 255, Blue: 255, Brightness: 1}},
						9: {Name: "", Width: 150, Height: 150, Left: 20, Top: 20, PacketIndex: []int{9}, Color: rgb.Color{Red: 0, Green: 255, Blue: 255, Brightness: 1}},
					},
				},
			},
		}
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
		deviceProfile.Stand = d.DeviceProfile.Stand
		if len(d.DeviceProfile.Path) < 1 {
			deviceProfile.Path = profilePath
			d.DeviceProfile.Path = profilePath
		} else {
			deviceProfile.Path = d.DeviceProfile.Path
		}
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

// SaveDeviceProfile will save a new device profile
func (d *Device) SaveDeviceProfile(_ string, _ bool) uint8 {
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
	if d.DeviceProfile == nil {
		return 0
	}

	if d.GetRgbProfile(profile) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}

	if d.DeviceProfile.RGBCluster {
		return 5
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
		LedChannels:  uint32(colorPacketLength),
		WriteColorEx: d.writeColorCluster,
	}

	cluster.Get().AddDeviceController(clusterController)
}

// ProcessSetRgbCluster will update OpenRGB integration status
func (d *Device) ProcessSetRgbCluster(enabled bool) uint8 {
	if d.DeviceProfile == nil {
		return 0
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
			LedChannels:  uint32(colorPacketLength),
			WriteColorEx: d.writeColorCluster,
		}

		cluster.Get().AddDeviceController(clusterController)
	} else {
		cluster.Get().RemoveDeviceControllerBySerial(d.Serial)
	}
	return 1
}

// UpdateDeviceColor will update device color based on selected input
func (d *Device) UpdateDeviceColor(keyId, keyOption int, color rgb.Color) uint8 {
	switch keyOption {
	case 0:
		{
			for rowIndex, row := range d.DeviceProfile.Stand.Row {
				for keyIndex, key := range row.Zones {
					if keyIndex == keyId {
						key.Color = rgb.Color{
							Red:        color.Red,
							Green:      color.Green,
							Blue:       color.Blue,
							Brightness: 0,
						}
						d.DeviceProfile.Stand.Row[rowIndex].Zones[keyIndex] = key
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
			for rowIndex, row := range d.DeviceProfile.Stand.Row {
				for keyIndex := range row.Zones {
					if keyIndex == keyId {
						rowId = rowIndex
						break
					}
				}
			}

			if rowId < 0 {
				return 0
			}

			for keyIndex, key := range d.DeviceProfile.Stand.Row[rowId].Zones {
				key.Color = rgb.Color{
					Red:        color.Red,
					Green:      color.Green,
					Blue:       color.Blue,
					Brightness: 0,
				}
				d.DeviceProfile.Stand.Row[rowId].Zones[keyIndex] = key
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
			for rowIndex, row := range d.DeviceProfile.Stand.Row {
				for keyIndex, key := range row.Zones {
					key.Color = rgb.Color{
						Red:        color.Red,
						Green:      color.Green,
						Blue:       color.Blue,
						Brightness: 0,
					}
					d.DeviceProfile.Stand.Row[rowIndex].Zones[keyIndex] = key
				}
			}
			if d.activeRgb != nil {
				d.activeRgb.Exit <- true
				d.activeRgb = nil
			}
			d.setDeviceColor()
			return 1
		}
	}
	return 0
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

	if d.DeviceProfile.RGBProfile == "static" || d.DeviceProfile.RGBProfile == "stand" {
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

	if d.DeviceProfile.RGBProfile == "static" || d.DeviceProfile.RGBProfile == "stand" {
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true
			d.activeRgb = nil
		}
		d.setDeviceColor()
	}
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

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
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

	for i := 0; i < d.LEDChannels; i++ {
		reset[i] = []byte{
			byte(color.Red),
			byte(color.Green),
			byte(color.Blue),
		}
	}

	buffer = rgb.SetColor(reset)
	_, err := d.transfer(cmdWrite, cmdWriteColor, buffer, false)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
	}

	if d.DeviceProfile == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to set color. DeviceProfile is null!")
		return
	}

	// RGB Cluster
	if d.DeviceProfile.RGBCluster {
		logger.Log(logger.Fields{}).Info("Exiting setDeviceColor() due to RGB Cluster")
		return
	}

	if d.DeviceProfile.RGBProfile == "stand" {
		var buf = make([]byte, colorPacketLength)
		for _, rows := range d.DeviceProfile.Stand.Row {
			for _, keys := range rows.Zones {
				color = &rgb.Color{
					Red:        keys.Color.Red,
					Green:      keys.Color.Green,
					Blue:       keys.Color.Blue,
					Brightness: keys.Color.Brightness,
				}

				color.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
				profileColor := rgb.ModifyBrightness(*color)
				for _, packetIndex := range keys.PacketIndex {
					buf[packetIndex] = byte(profileColor.Red)
					buf[packetIndex+1] = byte(profileColor.Green)
					buf[packetIndex+2] = byte(profileColor.Blue)
				}
			}
		}
		_, err = d.transfer(cmdWrite, cmdWriteColor, buf, false)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
		}
		return
	}

	if d.DeviceProfile.RGBProfile == "static" {
		profile := d.GetRgbProfile("static")
		if profile == nil {
			return
		}

		profile.StartColor.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
		profileColor := rgb.ModifyBrightness(profile.StartColor)
		for i := 0; i < d.LEDChannels; i++ {
			reset[i] = []byte{
				byte(profileColor.Red),
				byte(profileColor.Green),
				byte(profileColor.Blue),
			}
		}
		buffer = rgb.SetColor(reset)
		_, _ = d.transfer(cmdWrite, cmdWriteColor, buffer, false)
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
					for i := 0; i < d.LEDChannels; i++ {
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
				} else {
					r.RGBStartColor = d.activeRgb.RGBStartColor
					r.RGBEndColor = d.activeRgb.RGBEndColor
				}

				// Brightness
				r.RGBBrightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
				r.RGBStartColor.Brightness = r.RGBBrightness
				r.RGBEndColor.Brightness = r.RGBBrightness

				switch d.DeviceProfile.RGBProfile {
				case "custom":
					{
						for n := 0; n < d.LEDChannels; n++ {
							value := d.getLedProfileColor(n, 0) // This ledId is always 0
							value.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
							val := rgb.ModifyBrightness(*value)
							buff = append(buff, []byte{byte(val.Red), byte(val.Green), byte(val.Blue)}...)
						}
					}
				case "off":
					{
						for n := 0; n < d.LEDChannels; n++ {
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
				case "spiralrainbow":
					{
						r.SpiralRainbow(startTime)
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
				if d.Exit {
					return
				}

				_, err = d.transfer(cmdWrite, cmdWriteColor, buff, false)
				if err != nil {
					logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}(d.LEDChannels)
}

// writeColorCluster will write data to the device from cluster client
func (d *Device) writeColorCluster(data []byte, _ int) {
	if !d.DeviceProfile.RGBCluster {
		return
	}

	if d.Exit {
		return
	}

	var buf = make([]byte, colorPacketLength)
	for _, rows := range d.DeviceProfile.Stand.Row {
		for _, keys := range rows.Zones {
			for _, packetIndex := range keys.PacketIndex {
				buf[packetIndex] = data[packetIndex]
				buf[packetIndex+1] = data[packetIndex+1]
				buf[packetIndex+2] = data[packetIndex+2]
			}
		}
	}

	_, err := d.transfer(cmdWrite, cmdWriteColor, buf, false)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(command byte, endpoint, buffer []byte, read bool) ([]byte, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = command
	endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[headerWriteSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
	}

	bufferR := make([]byte, bufferSize)

	if _, err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return bufferR, err
	}

	if read {
		if _, err := d.dev.Read(bufferR); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
			return bufferR, err
		}
	}
	return bufferR, nil
}
