package lt100

// Package: CORSAIR LT100 Smart Lighting Tower
// This is the primary package for CORSAIR LT100 Smart Lighting Tower.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
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

type DeviceProfile struct {
	Active      bool
	Path        string
	Product     string
	Serial      string
	Brightness  uint8
	RGBProfiles map[int]string
	Labels      map[int]string
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
	Template                string
	Brightness              map[int]string
	HasLCD                  bool
	CpuTemp                 float32
	GpuTemp                 float32
	Keepalive               bool
	Rgb                     *rgb.RGB
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
	cmdGetDevices           = []byte{0x00, 0x02}
	dataFlush               = []byte{0xff}
	dataBrightness          = byte(0x64)
	mutex                   sync.Mutex
	deviceRefreshInterval   = 1000
	bufferSize              = 64
	readBufferSize          = 16
	bufferSizeWrite         = bufferSize + 1
	maxBufferSizePerRequest = 50
	authRefreshChan         = make(chan bool)
	keepAliveChan           = make(chan bool)
	timer                   = &time.Ticker{}
	timerKeepAlive          = &time.Ticker{}
	ledsPerTower            = 27
	deviceKeepAlive         = 1000
)

// Init will initialize a new device
func Init(vendorId, productId uint16, serial string) *Device {
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
		Product: "LT100 RGB",
	}

	// Bootstrap
	d.getManufacturer()    // Manufacturer
	d.getSerial()          // Serial
	d.loadRgb()            // Load RGB
	d.loadDeviceProfiles() // Load all device profiles
	d.wakeUpDevice()       // Wake up
	d.getDeviceFirmware()  // Firmware
	d.resetLeds()          // Reset all LEDs
	if d.getDevices() > 0 {
		d.setAutoRefresh()    // Set auto device refresh
		d.saveDeviceProfile() // Create device profile
		d.startColors()       // Start color processing
		d.setDeviceColor()    // Device color
		logger.Log(logger.Fields{"device": d}).Info("Device successfully initialized")
	} else {
		logger.Log(logger.Fields{"device": d}).Warn("Unable to get amount of connected towers. Closing device...")
		d.Stop()
		return nil
	}
	return d
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	logger.Log(logger.Fields{"serial": d.Serial}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}

	if d.Keepalive {
		keepAliveChan <- true
		timerKeepAlive.Stop()
	}

	timer.Stop()
	authRefreshChan <- true

	d.shutdownLed()
	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
		}
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
	if timerKeepAlive != nil && d.Keepalive {
		d.Keepalive = false
		keepAliveChan <- true
		timerKeepAlive.Stop()
	}
}

// setAutoRefresh will keep a device alive
func (d *Device) setKeepAlive() {
	timerKeepAlive = time.NewTicker(time.Duration(deviceKeepAlive) * time.Millisecond)
	keepAliveChan = make(chan bool)
	d.Keepalive = true
	go func() {
		for {
			select {
			case <-timerKeepAlive.C:
				d.keepAlive()
			case <-keepAliveChan:
				timerKeepAlive.Stop()
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
	d.write(cmdLedReset, buf) // Reset
	d.write(cmdStart, buf)    // Start

	// Brightness
	buf[1] = dataBrightness
	d.write(cmdBrightness, buf)

	// Set protocol
	buf[1] = 0x01
	d.write(cmdProtocol, buf)

	// Set port state
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

	// Wait for 2 seconds
	time.Sleep(2 * time.Second)
}

// startColors will initiate interface for receiving colors
func (d *Device) startColors() {
	buf := make([]byte, 8)
	buf[0] = 0x00
	d.write(cmdStart, buf)
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
	profileList := make(map[string]*DeviceProfile, 0)
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

// getProduct will return device name
func (d *Device) getProduct() {
	product, err := d.dev.GetProductStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get product")
	}
	d.Product = product
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
func (d *Device) wakeUpDevice() {
	_, err := d.transfer(0x3c, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
		return
	}
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	fw, err := d.transfer(cmdGetFirmware, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
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
		Product:     d.Product,
		Serial:      d.Serial,
		RGBProfiles: rgbProfiles,
		Labels:      labels,
		Path:        profilePath,
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
		deviceProfile.Active = d.DeviceProfile.Active
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
	m := 0
	towers := 0
	for _, cmdRefreshPort := range cmdRefreshPorts {
		d.write(cmdRefreshPort, nil)
	}

	result, err := d.transfer(cmdPortState, cmdGetDevices)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write red color to device")
	}

	d.write(cmdSave, dataFlush)
	if result[1] == 0 {
		// This device will randomly return 0 LEDs on towers, so we need to enumerate multiple times.
		// I'm unable to verify if this is a proper way to enumerate available towers or not, but no
		// other method worked. Only another option is to user define the number of connected towers
		// and this function is completely removed, but this isn't user-friendly.
		for {
			result, err = d.transfer(cmdPortState, cmdGetDevices)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to write red color to device")
				continue
			}
			if result[1] != 0 || m >= 100 {
				break
			}
			m++
			time.Sleep(500 * time.Millisecond)
		}
	}

	leds := result[1]
	switch leds {
	case 27:
		towers = 1
	case 54:
		towers = 2
	case 81:
		towers = 3
	case 108:
		towers = 4
	}

	if towers == 0 {
		return 0
	}

	logger.Log(logger.Fields{"towers": towers, "ledChannels": leds}).Info("Towers detected")

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
	mutex.Lock()
	defer mutex.Unlock()

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

	// Are all devices under static mode?
	// In static mode, we only need to send color once;
	// there is no need for continuous packet sending.
	s, l := 0, 0
	for _, k := range keys {
		if d.Devices[k].LedChannels > 0 {
			l++ // device has LED
			if d.Devices[k].RGB == "static" {
				s++ // led profile is set to static
			}
		}
	}

	if s > 0 || l > 0 { // We have some values
		if s == l { // number of devices matches number of devices with static profile
			static := map[int][]byte{}
			profile := d.GetRgbProfile("static")
			if d.DeviceProfile.Brightness != 0 {
				profile.StartColor.Brightness = rgb.GetBrightnessValue(d.DeviceProfile.Brightness)
			}
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
	}

	go func(lightChannels int) {
		lock := sync.Mutex{}
		startTime := time.Now()
		reverse := map[int]bool{}
		counterColorpulse := map[int]int{}
		counterFlickering := map[int]int{}
		counterColorshift := map[int]int{}
		counterCircleshift := map[int]int{}
		counterCircle := map[int]int{}
		counterColorwarp := map[int]int{}
		counterSpinner := map[int]int{}
		counterCpuTemp := map[int]int{}
		counterGpuTemp := map[int]int{}
		temperatureKeys := map[int]*rgb.Color{}
		colorwarpGeneratedReverse := false
		d.activeRgb = rgb.Exit()

		// Generate random colors
		d.activeRgb.RGBStartColor = rgb.GenerateRandomColor(1)
		d.activeRgb.RGBEndColor = rgb.GenerateRandomColor(1)
		hue := 1
		wavePosition := 0.0
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
					if d.DeviceProfile.Brightness > 0 {
						r.RGBBrightness = rgb.GetBrightnessValue(d.DeviceProfile.Brightness)
						r.RGBStartColor.Brightness = r.RGBBrightness
						r.RGBEndColor.Brightness = r.RGBBrightness
					}

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
					case "watercolor":
						{
							r.Watercolor(startTime)
							buff = append(buff, r.Output...)
						}
					case "cpu-temperature":
						{
							lock.Lock()
							counterCpuTemp[k]++
							if counterCpuTemp[k] >= r.Smoothness {
								counterCpuTemp[k] = 0
							}

							if _, ok := temperatureKeys[k]; !ok {
								temperatureKeys[k] = r.RGBStartColor
							}

							r.MinTemp = profile.MinTemp
							r.MaxTemp = profile.MaxTemp
							res := r.Temperature(float64(d.CpuTemp), counterCpuTemp[k], temperatureKeys[k])
							temperatureKeys[k] = res
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "gpu-temperature":
						{
							lock.Lock()
							counterGpuTemp[k]++
							if counterGpuTemp[k] >= r.Smoothness {
								counterGpuTemp[k] = 0
							}

							if _, ok := temperatureKeys[k]; !ok {
								temperatureKeys[k] = r.RGBStartColor
							}

							r.MinTemp = profile.MinTemp
							r.MaxTemp = profile.MaxTemp
							res := r.Temperature(float64(d.GpuTemp), counterGpuTemp[k], temperatureKeys[k])
							temperatureKeys[k] = res
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "colorpulse":
						{
							lock.Lock()
							counterColorpulse[k]++
							if counterColorpulse[k] >= r.Smoothness {
								counterColorpulse[k] = 0
							}

							r.Colorpulse(counterColorpulse[k])
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "static":
						{
							r.Static()
							buff = append(buff, r.Output...)
						}
					case "rotator":
						{
							r.Rotator(hue)
							buff = append(buff, r.Output...)
						}
					case "wave":
						{
							r.Wave(wavePosition)
							buff = append(buff, r.Output...)
						}
					case "storm":
						{
							r.Storm()
							buff = append(buff, r.Output...)
						}
					case "flickering":
						{
							lock.Lock()
							if counterFlickering[k] >= r.Smoothness {
								counterFlickering[k] = 0
							} else {
								counterFlickering[k]++
							}

							r.Flickering(counterFlickering[k])
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "colorshift":
						{
							lock.Lock()
							if counterColorshift[k] >= r.Smoothness && !reverse[k] {
								counterColorshift[k] = 0
								reverse[k] = true
							} else if counterColorshift[k] >= r.Smoothness && reverse[k] {
								counterColorshift[k] = 0
								reverse[k] = false
							}

							r.Colorshift(counterColorshift[k], reverse[k])
							counterColorshift[k]++
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "circleshift":
						{
							lock.Lock()
							if counterCircleshift[k] >= int(d.Devices[k].LedChannels) {
								counterCircleshift[k] = 0
							} else {
								counterCircleshift[k]++
							}

							r.Circle(counterCircleshift[k])
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "circle":
						{
							lock.Lock()
							if counterCircle[k] >= int(d.Devices[k].LedChannels) {
								counterCircle[k] = 0
							} else {
								counterCircle[k]++
							}

							r.Circle(counterCircle[k])
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "spinner":
						{
							lock.Lock()
							if counterSpinner[k] >= int(d.Devices[k].LedChannels) {
								counterSpinner[k] = 0
							} else {
								counterSpinner[k]++
							}
							r.Spinner(counterSpinner[k])
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					case "colorwarp":
						{
							lock.Lock()
							if counterColorwarp[k] >= r.Smoothness {
								if !colorwarpGeneratedReverse {
									colorwarpGeneratedReverse = true
									d.activeRgb.RGBStartColor = d.activeRgb.RGBEndColor
									d.activeRgb.RGBEndColor = rgb.GenerateRandomColor(r.RGBBrightness)
								}
								counterColorwarp[k] = 0
							} else if counterColorwarp[k] == 0 && colorwarpGeneratedReverse == true {
								colorwarpGeneratedReverse = false
							} else {
								counterColorwarp[k]++
							}

							r.Colorwarp(counterColorwarp[k], d.activeRgb.RGBStartColor, d.activeRgb.RGBEndColor)
							lock.Unlock()
							buff = append(buff, r.Output...)
						}
					}
				}
				// Send it
				d.writeColor(buff)
				time.Sleep(20 * time.Millisecond)
				hue++
				wavePosition += 0.2
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
	timer = time.NewTicker(time.Duration(deviceRefreshInterval) * time.Millisecond)
	authRefreshChan = make(chan bool)
	go func() {
		for {
			select {
			case <-timer.C:
				d.setTemperatures()
			case <-authRefreshChan:
				timer.Stop()
				return
			}
		}
	}()
}

// writeColor will write data to the device with a specific endpoint.
func (d *Device) writeColor(data []byte) {
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
			logger.Log(logger.Fields{"error": err}).Error("Unable to write red color to device")
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
			logger.Log(logger.Fields{"error": err}).Error("Unable to write red color to device")
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
	mutex.Lock()
	defer mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[0] = endpoint

	if buffer != nil && len(buffer) > 0 {
		copy(bufferW[1:], buffer)
	}
	// Create read buffer
	bufferR := make([]byte, readBufferSize)

	// Send command to a device
	if _, err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return nil, err
	}

	// Get data from a device
	if _, err := d.dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		return nil, err
	}

	return bufferR, nil
}
