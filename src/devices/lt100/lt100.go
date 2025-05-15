package lt100

// Package: CORSAIR LT100 Smart Lighting Tower
// This is the primary package for CORSAIR LT100 Smart Lighting Tower.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/led"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/temperatures"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"math/rand"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// DeviceInfo represents a USB device
type DeviceInfo struct {
	Bus         string
	Device      string
	ID          string
	Vendor      string
	Product     string
	Description string
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
	Labels             map[int]string
}

type Devices struct {
	ChannelId    int    `json:"channelId"`
	Type         byte   `json:"type"`
	Model        byte   `json:"-"`
	DeviceId     string `json:"deviceId"`
	Name         string `json:"name"`
	LedChannels  uint8  `json:"-"`
	Description  string `json:"description"`
	HubId        string `json:"-"`
	Profile      string `json:"profile"`
	RGB          string `json:"rgb"`
	Label        string `json:"label"`
	PortId       byte   `json:"-"`
	CellSize     uint8
	ContainsPump bool
}

type Device struct {
	dev                     *hid.Device
	Manufacturer            string                    `json:"manufacturer"`
	Product                 string                    `json:"product"`
	Serial                  string                    `json:"serial"`
	Firmware                string                    `json:"firmware"`
	Devices                 map[int]*Devices          `json:"devices"`
	UserProfiles            map[string]*DeviceProfile `json:"userProfiles"`
	DeviceProfile           *DeviceProfile
	ExternalLedDeviceAmount map[int]string
	activeRgb               *rgb.ActiveRGB
	ledProfile              *led.Device
	Template                string
	Brightness              map[int]string
	HasLCD                  bool
	CpuTemp                 float32
	GpuTemp                 float32
	Keepalive               bool
	Rgb                     *rgb.RGB
	VendorId                uint16
	ProductId               uint16
	Path                    string
	Exit                    bool
	mutex                   sync.Mutex
	autoRefreshChan         chan struct{}
	keepAliveChan           chan struct{}
	timer                   *time.Ticker
	timerKeepAlive          *time.Ticker
}

var (
	pwd                     = ""
	cmdGetFirmware          = byte(0x02)
	cmdLedReset             = byte(0x37)
	cmdPortState            = byte(0x38)
	cmdWriteLedConfig       = byte(0x35)
	cmdWriteColor           = byte(0x32)
	cmdSave                 = byte(0x33)
	cmdStart                = byte(0x34)
	cmdProtocol             = byte(0x3b)
	cmdBrightness           = byte(0x39)
	cmdRefreshPorts         = []byte{0x3c, 0x3d}
	dataFlush               = []byte{0xff}
	dataBrightness          = byte(0x64)
	deviceRefreshInterval   = 1000
	bufferSize              = 64
	readBufferSize          = 17
	bufferSizeWrite         = bufferSize + 1
	maxBufferSizePerRequest = 50
	ledsPerTower            = 27
	deviceKeepAlive         = 2000
	rgbProfileUpgrade       = []string{"custom"}
)

// Init will initialize a new device
func Init(vendorId, productId uint16, serial, path string) *Device {
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
		dev:      dev,
		Template: "lt100.html",
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
		Product:         "LT100 RGB",
		Path:            path,
		VendorId:        vendorId,
		ProductId:       productId,
		keepAliveChan:   make(chan struct{}),
		autoRefreshChan: make(chan struct{}),
	}

	// Bootstrap
	d.getManufacturer()    // Manufacturer
	d.getSerial()          // Serial
	d.loadRgb()            // Load RGB
	d.loadDeviceProfiles() // Load all device profiles
	d.getDeviceFirmware()  // Firmware
	d.resetLeds()          // Reset all LEDs
	if d.getDevices() > 0 {
		d.setAutoRefresh()    // Set auto device refresh
		d.saveDeviceProfile() // Create device profile
		d.setupLedProfile()   // LED profile
		d.setDeviceColor()    // Device color
		logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")
	} else {
		logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Warn("Unable to get amount of connected towers. Closing device...\"")
		d.Stop()
		return nil
	}
	return d
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
		d.saveLedProfile()                       // Save profile
		d.ledProfile = led.LoadProfile(d.Serial) // Reload
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
	lightChannels := 0
	keys := make([]int, 0)
	for k := range d.Devices {
		lightChannels += int(d.Devices[k].LedChannels)
		keys = append(keys, k)
	}
	sort.Ints(keys)

	device := led.Device{
		Serial:     d.Serial,
		DeviceName: d.Product,
	}

	devices := map[int]led.DeviceData{}

	for _, k := range keys {
		channels := map[int]rgb.Color{}
		deviceData := led.DeviceData{}
		deviceData.LedChannels = d.Devices[k].LedChannels
		deviceData.Tower = true
		for i := 0; i < int(d.Devices[k].LedChannels); i++ {
			channels[i] = rgb.Color{
				Red:   0,
				Green: 255,
				Blue:  255,
				Hex:   fmt.Sprintf("#%02x%02x%02x", 0, 255, 255),
			}
		}
		deviceData.Channels = channels
		devices[k] = deviceData
	}
	device.Devices = devices
	led.SaveProfile(d.Serial, device)
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
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.autoRefreshChan != nil {
				close(d.autoRefreshChan)
			}
			if d.Keepalive {
				close(d.keepAliveChan)
				d.timerKeepAlive.Stop()
			}
		})
	}()

	d.shutdownLed()
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
			if d.Keepalive {
				close(d.keepAliveChan)
				d.timerKeepAlive.Stop()
			}
		})
	}()
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 1
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
			d.Rgb.Profiles[profile] = rgb.Profile{}
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

// unsetKeepAlive will stop keepalive timer
func (d *Device) unsetKeepAlive() {
	if d.timerKeepAlive != nil && d.Keepalive {
		d.Keepalive = false
		d.timerKeepAlive.Stop()
	}
}

// setAutoRefresh will keep a device alive
func (d *Device) setKeepAlive() {
	d.timerKeepAlive = time.NewTicker(time.Duration(deviceKeepAlive) * time.Millisecond)
	d.Keepalive = true
	go func() {
		for {
			select {
			case <-d.timerKeepAlive.C:
				if d.Exit {
					return
				}
				d.keepAlive()
			case <-d.keepAliveChan:
				d.timerKeepAlive.Stop()
				return
			}
		}
	}()
}

// keepAlive will keep a device alive
func (d *Device) keepAlive() {
	_, err := d.transfer(cmdSave, dataFlush)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return
	}
}

// resetLeds will reset LED port
func (d *Device) resetLeds() {
	buf := make([]byte, 8)
	buf[0] = 0x00
	d.write(byte(0x06), buf)  // Reset
	d.write(cmdLedReset, buf) // Reset
	d.write(cmdStart, buf)    // Start

	// Brightness
	buf[1] = dataBrightness
	d.write(cmdBrightness, buf)

	buf[1] = 0x01
	d.write(cmdProtocol, buf)

	// Set port state
	buf[1] = 0x01
	d.write(cmdPortState, buf)

	// Write configuration
	buf[1] = 0x00
	buf[2] = 0x6c
	buf[3] = 0x0b
	buf[4] = 0x00
	buf[5] = 0x01
	buf[6] = 0x01
	d.write(cmdWriteLedConfig, buf)

	// Flush it
	d.write(cmdSave, dataFlush)

	// Set port state
	buf[1] = 0x02
	d.write(cmdPortState, buf)
}

// shutdownLed will reset LED ports and set device in 'hardware' mode
func (d *Device) shutdownLed() {
	lightChannels := 0
	for _, device := range d.Devices {
		lightChannels += int(device.LedChannels)
	}

	buf := make([]byte, 8)
	buf[0] = 0x00
	d.write(cmdLedReset, buf) // Reset
	d.write(cmdStart, buf)    // Start

	// Set port state
	buf[1] = 0x01
	d.write(cmdPortState, buf)

	// Write configuration
	buf[1] = 0x00
	buf[2] = byte(lightChannels)
	buf[3] = 0x0b
	buf[4] = 0x00
	buf[5] = 0x01
	buf[6] = 0x01
	d.write(cmdWriteLedConfig, buf)

	// Flush it
	d.write(cmdSave, dataFlush)
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

// getManufacturer will return device manufacturer
func (d *Device) getManufacturer() {
	manufacturer, err := d.dev.GetMfrStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get manufacturer")
	}
	d.Manufacturer = manufacturer
}

// getSerial will return device serial number
func (d *Device) getSerial() {
	serial, err := d.dev.GetSerialNbr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device serial number")
	}
	d.Serial = serial
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	fw, err := d.transfer(cmdGetFirmware, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get device firmware")
		return
	}
	v1, v2, v3 := int(fw[1]), int(fw[2]), int(fw[3])
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

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	var defaultBrightness = uint8(100)
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"

	rgbProfiles := make(map[int]string, len(d.Devices))
	labels := make(map[int]string, len(d.Devices))

	for _, device := range d.Devices {
		if device.LedChannels > 0 {
			rgbProfiles[device.ChannelId] = device.RGB
		}
		labels[device.ChannelId] = device.Label
	}

	deviceProfile := &DeviceProfile{
		Product:            d.Product,
		Serial:             d.Serial,
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
		deviceProfile.OriginalBrightness = d.DeviceProfile.OriginalBrightness
		deviceProfile.Brightness = d.DeviceProfile.Brightness
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

// getDevices will fetch all devices connected to a hub
func (d *Device) getDevices() int {
	towers := 0
	m := 0
	for {
		for _, cmdRefreshPort := range cmdRefreshPorts {
			res, err := d.transfer(cmdRefreshPort, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to write red color to device")
			}
			if res[1] != 0 {
				switch res[1] {
				case 27:
					towers = 1
				case 54:
					towers = 2
				case 81:
					towers = 3
				case 108:
					towers = 4
				}
			}
		}
		m++
		if towers != 0 || m > 100 {
			break
		}
	}

	if towers == 0 {
		return 0
	}

	logger.Log(logger.Fields{"towers": towers, "iterations": m}).Info("Towers detected")

	var devices = make(map[int]*Devices, 2)
	for i := 0; i < 2; i++ {
		rgbProfile := "static"
		label := "Set Label"

		if d.DeviceProfile != nil {
			if rp, ok := d.DeviceProfile.RGBProfiles[i]; ok {
				if d.GetRgbProfile(rp) != nil { // Speed profile exists in configuration
					// Speed profile exists in configuration
					rgbProfile = rp
				}
			}

			// Device label
			if lb, ok := d.DeviceProfile.Labels[i]; ok {
				if len(lb) > 0 {
					label = lb
				}
			}
		} else {
			logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
		}

		device := &Devices{
			ChannelId:   i,
			DeviceId:    fmt.Sprintf("%s-%v", "LED", i),
			Name:        fmt.Sprintf("%s #%v", "LT 100", i),
			Description: "LED",
			HubId:       d.Serial,
			LedChannels: uint8(ledsPerTower),
			RGB:         rgbProfile,
			CellSize:    2,
			PortId:      0,
			Label:       label,
		}
		devices[i] = device
	}
	d.Devices = devices
	return len(devices)
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
	if d.GetRgbProfile(profileName) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}

	pf := d.GetRgbProfile(profileName)
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

	if channelId < 0 {
		for _, device := range d.Devices {
			if device.LedChannels > 0 {
				d.DeviceProfile.RGBProfiles[device.ChannelId] = profile
				d.Devices[device.ChannelId].RGB = profile
			}
		}
	} else {
		if _, ok := d.Devices[channelId]; ok {
			d.DeviceProfile.RGBProfiles[channelId] = profile // Set profile
			d.Devices[channelId].RGB = profile
		} else {
			return 0
		}
	}

	d.DeviceProfile.RGBProfiles[channelId] = profile // Set profile
	d.saveDeviceProfile()                            // Save profile
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}

	d.setDeviceColor() // Restart RGB
	return 1
}

// UpdateDeviceLabel will set / update device label
func (d *Device) UpdateDeviceLabel(channelId int, label string) uint8 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if _, ok := d.Devices[channelId]; !ok {
		return 0
	}

	d.Devices[channelId].Label = label
	d.saveDeviceProfile()
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

	if d.isRgbStatic() {
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
	if d.isRgbStatic() {
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true // Exit current RGB mode
			d.activeRgb = nil
		}
		d.setDeviceColor() // Restart RGB
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
			d.activeRgb.Exit <- true // Exit current RGB mode
			d.activeRgb = nil
		}
		for _, device := range d.Devices {
			if device.LedChannels > 0 {
				d.Devices[device.ChannelId].RGB = profile.RGBProfiles[device.ChannelId]
			}
			d.Devices[device.ChannelId].Label = profile.Labels[device.ChannelId]
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

// isRgbStatic will return true or false if all devices are set to static RGB mode
func (d *Device) isRgbStatic() bool {
	s, l := 0, 0

	keys := make([]int, 0)
	for k := range d.Devices {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		if d.Devices[k].LedChannels > 0 {
			l++ // device has LED
			if d.Devices[k].RGB == "static" {
				s++ // led profile is set to static
			}
		}
	}

	if s > 0 || l > 0 { // We have some values
		if s == l {
			return true
		}
	}
	return false
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	// Reset
	reset := map[int][]byte{}
	var buffer []byte
	lightChannels := 0

	keys := make([]int, 0)
	for k := range d.Devices {
		lightChannels += int(d.Devices[k].LedChannels)
		keys = append(keys, k)
	}
	sort.Ints(keys)

	// Do we have any RGB component in the system?
	if lightChannels == 0 {
		logger.Log(logger.Fields{}).Info("No RGB compatible devices found")
		return
	}

	// Reset color
	color := &rgb.Color{Red: 0, Green: 0, Blue: 0, Brightness: 0}
	for i := 0; i < lightChannels; i++ {
		reset[i] = []byte{
			byte(color.Red),
			byte(color.Green),
			byte(color.Blue),
		}
	}
	buffer = rgb.SetColor(reset)
	d.writeColor(buffer)

	if d.isRgbStatic() {
		static := map[int][]byte{}
		profile := d.GetRgbProfile("static")
		if profile == nil {
			return
		}

		profile.StartColor.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
		profileColor := rgb.ModifyBrightness(profile.StartColor)
		m := 0
		for _, k := range keys {
			for i := 0; i < int(d.Devices[k].LedChannels); i++ {
				static[m] = []byte{
					byte(profileColor.Red),
					byte(profileColor.Green),
					byte(profileColor.Blue),
				}
				m++
			}
		}
		buffer = rgb.SetColor(static)
		d.writeColor(buffer)
		d.setKeepAlive()
		return
	} else {
		d.unsetKeepAlive()
	}

	go func(lightChannels int) {
		startTime := time.Now()
		d.activeRgb = rgb.Exit()

		// Generate random colors
		d.activeRgb.RGBStartColor = rgb.GenerateRandomColor(1)
		d.activeRgb.RGBEndColor = rgb.GenerateRandomColor(1)
		rand.New(rand.NewSource(time.Now().UnixNano()))
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
						logger.Log(logger.Fields{"profile": d.Devices[k].RGB, "serial": d.Serial}).Warn("No such RGB profile found")
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

					switch d.Devices[k].RGB {
					case "custom":
						{
							for n := 0; n < int(d.Devices[k].LedChannels); n++ {
								value := d.getLedProfileColor(k, n)
								value.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
								val := rgb.ModifyBrightness(*value)
								buff = append(buff, []byte{byte(val.Red), byte(val.Green), byte(val.Blue)}...)
							}
						}
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
				}
				// Send it
				d.writeColor(buff)
				time.Sleep(10 * time.Millisecond)
			}
		}
	}(lightChannels)
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
				d.setTemperatures()
			case <-d.autoRefreshChan:
				d.timer.Stop()
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
	packetLen := len(data) / 3
	r := make([]byte, packetLen)
	g := make([]byte, packetLen)
	b := make([]byte, packetLen)
	m := 0

	for i := 0; i < packetLen; i++ {
		r[i] = data[m]
		m++
		g[i] = data[m]
		m++
		b[i] = data[m]
		m++
	}

	chunksR := common.ProcessMultiChunkPacket(r, maxBufferSizePerRequest)
	chunksG := common.ProcessMultiChunkPacket(g, maxBufferSizePerRequest)
	chunksB := common.ProcessMultiChunkPacket(b, maxBufferSizePerRequest)

	// Prepare for packets
	_, err := d.transfer(cmdPortState, []byte{0x00, 0x02})
	if err != nil {
		return
	}

	for p := 0; p < len(chunksR); p++ {
		chunkPacket := make([]byte, len(chunksR[p])+4)
		chunkPacket[1] = byte(p * maxBufferSizePerRequest)
		chunkPacket[2] = byte(maxBufferSizePerRequest)
		chunkPacket[3] = 0x00
		copy(chunkPacket[4:], chunksR[p])
		_, err = d.transfer(cmdWriteColor, chunkPacket)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to write red color to device")
		}
	}

	for p := 0; p < len(chunksG); p++ {
		chunkPacket := make([]byte, len(chunksG[p])+4)
		chunkPacket[1] = byte(p * maxBufferSizePerRequest)
		chunkPacket[2] = byte(maxBufferSizePerRequest)
		chunkPacket[3] = 0x01
		copy(chunkPacket[4:], chunksG[p])
		_, err = d.transfer(cmdWriteColor, chunkPacket)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to write green color to device")
		}
	}

	for p := 0; p < len(chunksB); p++ {
		chunkPacket := make([]byte, len(chunksB[p])+4)
		chunkPacket[1] = byte(p * maxBufferSizePerRequest)
		chunkPacket[2] = byte(maxBufferSizePerRequest)
		chunkPacket[3] = 0x02
		copy(chunkPacket[4:], chunksB[p])
		_, err = d.transfer(cmdWriteColor, chunkPacket)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to write blue color to device")
		}
	}

	// Flush everything
	flush := make([]byte, 4)
	flush[1] = 0x64
	flush[2] = 0x08
	for end := 0x00; end < 0x03; end++ {
		flush[3] = byte(end)
		_, err = d.transfer(cmdWriteColor, flush)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to flush packet to a device")
		}
	}
	d.write(cmdSave, dataFlush)
}

// write will send data to a device
func (d *Device) write(endpoint byte, buffer []byte) {
	_, err := d.transfer(endpoint, buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write data to device")
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint byte, buffer []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	if endpoint > 0 {
		bufferW[1] = endpoint
	}

	if buffer != nil && len(buffer) > 0 {
		copy(bufferW[2:], buffer)
	}
	// Create read buffer
	bufferR := make([]byte, readBufferSize)

	// Send command to a device
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
