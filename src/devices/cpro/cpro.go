package cpro

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/temperatures"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Package: Corsair Commander Pro
// This is the primary package for Corsair Commander Pro.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

// ExternalLedDevice contains a list of supported external-LED devices connected to a HUB
type ExternalLedDevice struct {
	Index   int
	Name    string
	Total   int
	Command byte
}

type ExternalHubData struct {
	PortId                  byte
	ExternalHubDeviceType   int
	ExternalHubDeviceAmount int
}

type DeviceProfile struct {
	Active        bool
	Path          string
	Product       string
	Serial        string
	Brightness    uint8
	RGBProfiles   map[int]string
	SpeedProfiles map[int]string
	ExternalHubs  map[int]*ExternalHubData
	Labels        map[int]string
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
	ExternalLedDevice       []ExternalLedDevice
	activeRgb               map[int]*rgb.ActiveRGB
	Template                string
	Brightness              map[int]string
}

type Devices struct {
	ChannelId          int             `json:"channelId"`
	Type               byte            `json:"type"`
	Model              byte            `json:"-"`
	DeviceId           string          `json:"deviceId"`
	Name               string          `json:"name"`
	DefaultValue       byte            `json:"-"`
	Rpm                int16           `json:"rpm"`
	Temperature        float32         `json:"temperature"`
	LedChannels        uint8           `json:"-"`
	ContainsPump       bool            `json:"-"`
	Description        string          `json:"description"`
	HubId              string          `json:"-"`
	PumpModes          map[byte]string `json:"-"`
	Profile            string          `json:"profile"`
	RGB                string          `json:"rgb"`
	Label              string          `json:"label"`
	PortId             byte            `json:"-"`
	HasSpeed           bool
	HasTemps           bool
	IsTemperatureProbe bool
	ExternalLed        bool
}

var (
	pwd, _                     = os.Getwd()
	cmdGetFirmware             = byte(0x02)
	cmdInitDevice              = byte(0x03)
	cmdGetConnectedFans        = byte(0x20)
	cmdGetConnectedProbes      = byte(0x10)
	cmdLedReset                = byte(0x37)
	cmdGetSpeed                = byte(0x21)
	cmdGetTemperature          = byte(0x11)
	cmdSetSpeed                = byte(0x23)
	cmdFanMode                 = byte(0x28)
	cmdWriteLedConfig          = byte(0x35)
	cmdWriteColor              = byte(0x32)
	cmdRefresh                 = byte(0x33)
	cmdRefresh2                = byte(0x34)
	cmdPortState               = byte(0x38)
	fanModePwm                 = byte(0x02)
	mutex                      sync.Mutex
	deviceRefreshInterval      = 1500
	temperaturePullingInterval = 3000
	bufferSize                 = 64
	readBufferSite             = 16
	fanChannels                = 6
	temperatureChannels        = 4
	maxBufferSizePerRequest    = 50
	defaultSpeedValue          = 100
	authRefreshChan            = make(chan bool)
	speedRefreshChan           = make(chan bool)
	timer                      = &time.Ticker{}
	timerSpeed                 = &time.Ticker{}
	maximumLedAmount           = 408
	externalLedDevices         = []ExternalLedDevice{
		{
			Index: 1,
			Name:  "HD RGB Series Fan",
			Total: 12,
		},
		{
			Index: 2,
			Name:  "LL RGB Series Fan",
			Total: 16,
		},
		{
			Index: 3,
			Name:  "ML PRO RGB Series Fan",
			Total: 4,
		},
		{
			Index: 4,
			Name:  "QL RGB Series Fan",
			Total: 34,
		},
		{
			Index: 5,
			Name:  "8-LED Series Fan",
			Total: 8,
		},
		{
			Index: 6,
			Name:  "SP RGB Series Fan",
			Total: 1,
		},
	}
)

// Init will initialize a new device
func Init(vendorId, productId uint16, serial string) *Device {
	// Open device, return if failure
	dev, err := hid.Open(vendorId, productId, serial)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId, "serial": serial}).Error("Unable to open HID device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:               dev,
		Template:          "cpro.html",
		ExternalLedDevice: externalLedDevices,
		ExternalLedDeviceAmount: map[int]string{
			0: "No Device",
			1: "1 Device",
			2: "2 Devices",
			3: "3 Devices",
			4: "4 Devices",
			5: "5 Devices",
			6: "6 Devices",
		},
		activeRgb: make(map[int]*rgb.ActiveRGB, 2),
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
	}

	// Bootstrap
	d.getManufacturer()    // Manufacturer
	d.getProduct()         // Product
	d.getSerial()          // Serial
	d.loadDeviceProfiles() // Load all device profiles
	d.getDeviceFirmware()  // Firmware
	d.getDevices()         // Get devices connected to a hub
	d.setFanMode()         // Set default fan mode
	d.saveDeviceProfile()  // Create device profile
	d.setColorEndpoint()   // Setup lightning
	d.setDefaults()        // Set default speed value
	d.setAutoRefresh()     // Set auto device refresh
	d.setDeviceColor()     // Device color
	if config.GetConfig().Manual {
		fmt.Println(
			fmt.Sprintf("[%s] Manual flag enabled. Process will not monitor temperature or adjust fan speed.", d.Serial),
		)
	} else {
		d.updateDeviceSpeed() // Update device speed
	}
	logger.Log(logger.Fields{"device": d}).Info("Device successfully initialized")
	return d
}

// ShutdownLed will reset LED ports and set device in 'hardware' mode
func (d *Device) ShutdownLed() {
	for i := 0; i < len(d.DeviceProfile.ExternalHubs); i++ {
		externalHub := d.DeviceProfile.ExternalHubs[i]
		lightChannels := 0
		for _, device := range d.Devices {
			if device.PortId == externalHub.PortId {
				lightChannels += int(device.LedChannels)
			}
		}
		cfg := []byte{externalHub.PortId, 0x00, byte(lightChannels), 0x04, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff}
		_, err := d.transfer(cmdLedReset, []byte{externalHub.PortId})
		if err != nil {
			return
		}
		_, err = d.transfer(cmdRefresh2, []byte{externalHub.PortId})
		if err != nil {
			return
		}
		_, err = d.transfer(cmdPortState, []byte{externalHub.PortId, 0x01})
		if err != nil {
			return
		}
		_, err = d.transfer(cmdWriteLedConfig, cfg)
		if err != nil {
			return
		}
		_, err = d.transfer(cmdRefresh, []byte{0xff})
		if err != nil {
			return
		}
	}
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	logger.Log(logger.Fields{"serial": d.Serial}).Info("Stopping device...")
	for i := 0; i < len(d.DeviceProfile.ExternalHubs); i++ {
		if d.activeRgb[i] != nil {
			d.activeRgb[i].Exit <- true // Exit current RGB mode
			d.activeRgb[i] = nil
		}
	}

	d.ShutdownLed()
	timer.Stop()
	if !config.GetConfig().Manual {
		timerSpeed.Stop()
		speedRefreshChan <- true
	}
	authRefreshChan <- true

	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
		}
	}
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
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to get manufacturer")
	}
	d.Manufacturer = manufacturer
}

// getProduct will return device name
func (d *Device) getProduct() {
	product, err := d.dev.GetProductStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to get product")
	}
	d.Product = product
}

// getSerial will return device serial number
func (d *Device) getSerial() {
	serial, err := d.dev.GetSerialNbr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to get device serial number")
	}
	d.Serial = serial
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	_, err := d.transfer(cmdInitDevice, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
		return
	}

	fw, err := d.transfer(cmdGetFirmware, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
		return
	}

	v1, v2, v3 := int(fw[1]), int(fw[2]), int(fw[3])
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) setFanMode() {
	for device := range d.Devices {
		if d.Devices[device].HasSpeed {
			_, err := d.transfer(cmdFanMode, []byte{0x02, byte(device), fanModePwm})
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to set device mode")
				continue
			}
			/*
				_, err = d.transfer(byte(0x26), []byte{byte(device), 0x0f, 0xfa})
				if err != nil {
					logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to set device mode")
					continue
				}
			*/
		}
	}
}

// setDefaults will set default mode for all devices
func (d *Device) setDefaults() {
	channelDefaults := map[int]byte{}
	for device := range d.Devices {
		if d.Devices[device].HasSpeed {
			channelDefaults[device] = byte(defaultSpeedValue)
		}
	}
	d.setSpeed(channelDefaults)
}

// setSpeed will generate a speed buffer and send it to a device
func (d *Device) setSpeed(data map[int]byte) {
	for channel, value := range data {
		_, err := d.transfer(cmdSetSpeed, []byte{byte(channel), value})
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to set device speed")
			continue
		}
	}
}

// UpdateRgbProfile will update device RGB profile
func (d *Device) UpdateRgbProfile(channelId int, profile string) uint8 {
	if rgb.GetRgbProfile(profile) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}

	if _, ok := d.Devices[channelId]; ok {
		// Update channel with new profile
		d.Devices[channelId].RGB = profile
	} else {
		return 0
	}

	d.DeviceProfile.RGBProfiles[channelId] = profile // Set profile
	d.saveDeviceProfile()                            // Save profile
	for i := 0; i < len(d.DeviceProfile.ExternalHubs); i++ {
		if d.activeRgb[i] != nil {
			d.activeRgb[i].Exit <- true // Exit current RGB mode
			d.activeRgb[i] = nil
		}
	}
	d.setDeviceColor() // Restart RGB
	return 1
}

// UpdateDeviceSpeed will change device speed profile
func (d *Device) UpdateDeviceSpeed(channelId int, value uint16) uint8 {
	// Check if actual channelId exists in the device list
	if device, ok := d.Devices[channelId]; ok {
		if device.IsTemperatureProbe {
			return 0
		}
		channelSpeeds := map[int][]byte{}

		if value < 20 {
			value = 20
		}

		// Minimal pump speed should be 50%
		if device.ContainsPump {
			if value < 50 {
				value = 50
			}
		}
		channelSpeeds[device.ChannelId] = []byte{byte(value)}
		//d.setSpeed(channelSpeeds, 0)
		return 1
	}
	return 0
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

// ChangeDeviceBrightness will change device brightness
func (d *Device) ChangeDeviceBrightness(mode uint8) uint8 {
	d.DeviceProfile.Brightness = mode
	d.saveDeviceProfile()
	for i := 0; i < len(d.DeviceProfile.ExternalHubs); i++ {
		if d.activeRgb[i] != nil {
			d.activeRgb[i].Exit <- true // Exit current RGB mode
			d.activeRgb[i] = nil
		}
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
		for i := 0; i < len(d.DeviceProfile.ExternalHubs); i++ {
			if d.activeRgb[i] != nil {
				d.activeRgb[i].Exit <- true // Exit current RGB mode
				d.activeRgb[i] = nil
			}
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
		// Speed reset
		if !config.GetConfig().Manual {
			timerSpeed.Stop()
			d.updateDeviceSpeed() // Update device speed
		}
		return 1
	}
	return 0
}

// UpdateSpeedProfile will update device channel speed.
func (d *Device) UpdateSpeedProfile(channelId int, profile string) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	// Check if the profile exists
	profiles := temperatures.GetTemperatureProfile(profile)
	if profiles == nil {
		return 0
	}

	// If the profile is liquid temperature, check for the presence of AIOs
	if profiles.Sensor == temperatures.SensorTypeLiquidTemperature {
		// This device does not have an option for AIO pump
		return 2
	}

	// Check if actual channelId exists in the device list
	if _, ok := d.Devices[channelId]; ok {
		if d.Devices[channelId].HasSpeed {
			d.Devices[channelId].Profile = profile
		}
	}
	d.saveDeviceProfile()
	return 1
}

// UpdateDeviceMetrics will update device metrics
func (d *Device) UpdateDeviceMetrics() {
	for _, device := range d.Devices {
		header := &metrics.Header{
			Product:          d.Product,
			Serial:           d.Serial,
			Firmware:         d.Firmware,
			ChannelId:        strconv.Itoa(device.ChannelId),
			Name:             device.Name,
			Description:      device.Description,
			Profile:          device.Profile,
			Label:            device.Label,
			RGB:              device.RGB,
			AIO:              strconv.FormatBool(device.ContainsPump),
			ContainsPump:     strconv.FormatBool(device.ContainsPump),
			Temperature:      float64(device.Temperature),
			LedChannels:      strconv.Itoa(int(device.LedChannels)),
			Rpm:              device.Rpm,
			TemperatureProbe: strconv.FormatBool(device.IsTemperatureProbe),
		}
		metrics.Populate(header)
	}
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

	speedProfiles := make(map[int]string, len(d.Devices))
	rgbProfiles := make(map[int]string, len(d.Devices))
	labels := make(map[int]string, len(d.Devices))

	for _, device := range d.Devices {
		if device.LedChannels > 0 {
			rgbProfiles[device.ChannelId] = device.RGB
		}

		if device.HasSpeed {
			speedProfiles[device.ChannelId] = device.Profile
		}

		labels[device.ChannelId] = device.Label
	}

	deviceProfile := &DeviceProfile{
		Product:       d.Product,
		Serial:        d.Serial,
		RGBProfiles:   rgbProfiles,
		SpeedProfiles: speedProfiles,
		ExternalHubs:  make(map[int]*ExternalHubData, 2),
		Labels:        labels,
		Path:          profilePath,
	}

	// First save, assign saved profile to a device
	if d.DeviceProfile == nil {
		for _, device := range d.Devices {
			if device.LedChannels > 0 {
				rgbProfiles[device.ChannelId] = "static"
			}
			labels[device.ChannelId] = "Not Set"
		}
		deviceProfile.Active = true

		for i := 0; i < 2; i++ {
			externalHubs := &ExternalHubData{
				PortId:                  byte(i),
				ExternalHubDeviceType:   0,
				ExternalHubDeviceAmount: 0,
			}
			deviceProfile.ExternalHubs[i] = externalHubs
		}
		d.DeviceProfile = deviceProfile
	} else {
		deviceProfile.ExternalHubs = d.DeviceProfile.ExternalHubs
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
	buffer, err := json.Marshal(deviceProfile)
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

// getDeviceData will fetch device data
func (d *Device) getDeviceData() {
	m := 0

	// Fans
	response, err := d.transfer(cmdGetConnectedFans, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to get connected devices")
		return
	}
	for s, i := 0, 1; s < fanChannels; s, i = s+1, i+1 {
		connected := response[i] > 0x00
		if connected {
			rpm, e := d.transfer(cmdGetSpeed, []byte{byte(s)})
			if e != nil {
				logger.Log(logger.Fields{"error": e, "channel": s, "serial": d.Serial}).Error("Unable to read get device speed")
				continue
			}
			val := binary.BigEndian.Uint16(rpm[1:])
			if _, ok := d.Devices[m]; ok {
				if val > 1 {
					d.Devices[m].Rpm = int16(val)
				}
			}
		}
		m++
	}

	// Temperature probes
	response, err = d.transfer(cmdGetConnectedProbes, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to get connected temperature probes")
		return
	}
	for z, k := 0, 1; z < temperatureChannels; z, k = z+1, k+1 {
		connected := response[k] == 0x01
		if connected {
			temp, e := d.transfer(cmdGetTemperature, []byte{byte(z)})
			if e != nil {
				logger.Log(logger.Fields{"error": e, "channel": z, "serial": d.Serial}).Error("Unable to read get device speed")
				continue
			}
			val := binary.BigEndian.Uint16(temp[1:]) / 100
			if _, ok := d.Devices[m]; ok {
				if val > 1 {
					d.Devices[m].Temperature = float32(val)
				}
			}
		}
		m++
	}
}

// getDevices will fetch all devices connected to a hub
func (d *Device) getDevices() int {
	var devices = make(map[int]*Devices, 0)
	m := 0

	// Fans
	response, err := d.transfer(cmdGetConnectedFans, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to get connected devices")
		return 0
	}
	for s, i := 0, 1; s < fanChannels; s, i = s+1, i+1 {
		connected := response[i] > 0x00
		if connected {
			rpm, e := d.transfer(cmdGetSpeed, []byte{byte(s)})
			if e != nil {
				logger.Log(logger.Fields{"error": e, "channel": s, "serial": d.Serial}).Error("Unable to read get device speed")
				continue
			}
			val := binary.BigEndian.Uint16(rpm[1:])
			if val > 0 {
				speedProfile := "Normal"
				label := "Not Set"
				if d.DeviceProfile != nil {
					// Profile is set
					if sp, ok := d.DeviceProfile.SpeedProfiles[m]; ok {
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
					if lb, ok := d.DeviceProfile.Labels[m]; ok {
						if len(lb) > 0 {
							label = lb
						}
					}
				} else {
					logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
				}

				// Build device object
				device := &Devices{
					ChannelId:   s,
					DeviceId:    fmt.Sprintf("%s-%v", "Fan", s),
					Name:        fmt.Sprintf("Fan %d", s+1),
					Rpm:         int16(val),
					Temperature: 0,
					Description: "Fan",
					LedChannels: 0,
					HubId:       d.Serial,
					Profile:     speedProfile,
					Label:       label,
					HasSpeed:    true,
					HasTemps:    false,
				}
				devices[m] = device
			}
		}
		m++
	}

	// Temperature probes
	response, err = d.transfer(cmdGetConnectedProbes, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to get connected temperature probes")
		return 0
	}
	for s, i := 0, 1; s < temperatureChannels; s, i = s+1, i+1 {
		connected := response[i] == 0x01
		if connected {
			temp, e := d.transfer(cmdGetTemperature, []byte{byte(s)})
			if e != nil {
				logger.Log(logger.Fields{"error": e, "channel": s, "serial": d.Serial}).Error("Unable to read get device speed")
				continue
			}
			val := binary.BigEndian.Uint16(temp[1:]) / 100

			label := "Not Set"
			if d.DeviceProfile != nil {
				// Device label
				if lb, ok := d.DeviceProfile.Labels[m]; ok {
					if len(lb) > 0 {
						label = lb
					}
				}
			}

			// Build device object
			device := &Devices{
				ChannelId:   m,
				DeviceId:    fmt.Sprintf("%s-%v", "Probe", s),
				Name:        fmt.Sprintf("Temperature Probe %d", s),
				Rpm:         0,
				Temperature: float32(val),
				Description: "Probe",
				LedChannels: 0,
				HubId:       d.Serial,
				HasSpeed:    false,
				HasTemps:    true,
				Label:       label,
			}
			devices[m] = device
		}
		m++
	}

	// RGB
	if d.DeviceProfile != nil {
		for i := 0; i < len(d.DeviceProfile.ExternalHubs); i++ {
			externalHub := d.DeviceProfile.ExternalHubs[i]
			externalDeviceType := d.getExternalLedDevice(externalHub.ExternalHubDeviceType)
			if externalDeviceType != nil {
				LedChannels := uint8(externalDeviceType.Total)
				for z := 0; z < externalHub.ExternalHubDeviceAmount; z++ {
					rgbProfile := "static"
					label := "Not Set"

					if rp, ok := d.DeviceProfile.RGBProfiles[m]; ok {
						if rgb.GetRgbProfile(rp) != nil { // Speed profile exists in configuration
							// Speed profile exists in configuration
							rgbProfile = rp
						}
					}

					if lb, ok := d.DeviceProfile.Labels[m]; ok {
						label = lb
					}

					device := &Devices{
						ChannelId:   m,
						DeviceId:    fmt.Sprintf("%s-%v", "LED", m),
						Name:        externalDeviceType.Name,
						Description: "LED",
						HubId:       d.Serial,
						LedChannels: LedChannels,
						RGB:         rgbProfile,
						PortId:      externalHub.PortId,
						HasSpeed:    false,
						HasTemps:    false,
						Label:       label,
					}
					devices[m] = device
					m++
				}
			}
		}
	}

	d.Devices = devices
	return len(devices)
}

// getExternalLedDevice will return ExternalLedDevice based on given device index
func (d *Device) getExternalLedDevice(index int) *ExternalLedDevice {
	for _, externalLedDevice := range externalLedDevices {
		if externalLedDevice.Index == index {
			return &externalLedDevice
		}
	}
	return nil
}

// setColorEndpoint will activate hub color endpoint for further usage
func (d *Device) setColorEndpoint() {
	for _, externalHub := range d.DeviceProfile.ExternalHubs {
		if externalHub.ExternalHubDeviceAmount > 0 {
			lightChannels := 0
			for _, device := range d.Devices {
				if device.PortId == externalHub.PortId {
					lightChannels += int(device.LedChannels)
				}
			}
			cfg := []byte{externalHub.PortId, 0x00, byte(lightChannels), 0x04, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff}
			_, err := d.transfer(cmdLedReset, []byte{externalHub.PortId})
			if err != nil {
				return
			}

			_, err = d.transfer(cmdWriteLedConfig, cfg)
			if err != nil {
				return
			}
		}
	}
}

// ResetRgb will reset the current rgb mode
func (d *Device) ResetRgb() {
	for i := 0; i < len(d.DeviceProfile.ExternalHubs); i++ {
		if d.activeRgb[i] != nil {
			d.activeRgb[i].Exit <- true // Exit current RGB mode
			d.activeRgb[i] = nil
		}
	}
	d.getDevices()        // Reload devices
	d.saveDeviceProfile() // Save profile
	d.setDeviceColor()    // Restart RGB
}

// UpdateExternalHubDeviceType will update a device type connected to the external-LED hub
func (d *Device) UpdateExternalHubDeviceType(portId, externalType int) uint8 {
	if d.DeviceProfile != nil {
		if externalType == 0 {
			d.DeviceProfile.ExternalHubs[portId].ExternalHubDeviceType = externalType
			d.ResetRgb()
			return 1
		}
		if d.getExternalLedDevice(externalType) != nil {
			d.DeviceProfile.ExternalHubs[portId].ExternalHubDeviceType = externalType
			d.ResetRgb()
			return 1
		} else {
			return 2
		}
	}
	return 0
}

// UpdateExternalHubDeviceAmount will update device amount connected to an external-LED hub and trigger RGB reset
func (d *Device) UpdateExternalHubDeviceAmount(portId, externalDevices int) uint8 {
	if d.DeviceProfile != nil {
		if _, ok := d.DeviceProfile.ExternalHubs[portId]; ok {
			// Store current amount
			currentAmount := d.DeviceProfile.ExternalHubs[portId].ExternalHubDeviceAmount

			// Init number of LED channels
			lightChannels := 0

			// Set new device amount
			d.DeviceProfile.ExternalHubs[portId].ExternalHubDeviceAmount = externalDevices

			// Validate the maximum number of LED channels
			for i := 0; i < len(d.DeviceProfile.ExternalHubs); i++ {
				externalHub := d.DeviceProfile.ExternalHubs[i]
				if deviceType := d.getExternalLedDevice(externalHub.ExternalHubDeviceType); deviceType != nil {
					lightChannels += deviceType.Total * externalHub.ExternalHubDeviceAmount
				}
			}
			if lightChannels > maximumLedAmount {
				d.DeviceProfile.ExternalHubs[portId].ExternalHubDeviceAmount = currentAmount
				logger.Log(logger.Fields{"serial": d.Serial, "portId": portId}).Info("You have exceeded maximum amount of supported LED channels.")
				return 2
			}

			d.DeviceProfile.ExternalHubs[portId].ExternalHubDeviceAmount = externalDevices
			for i := 0; i < len(d.DeviceProfile.ExternalHubs); i++ {
				if d.activeRgb[i] != nil {
					d.activeRgb[i].Exit <- true // Exit current RGB mode
					d.activeRgb[i] = nil
				}
			}
			d.getDevices()        // Reload devices
			d.saveDeviceProfile() // Save profile
			d.setDeviceColor()    // Restart RGB
			return 1
		}
	}
	return 0
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	// Reset
	reset := map[int][]byte{}
	var buffer []byte

	// Get the number of LED channels we have
	lightChannels := 0
	for _, device := range d.Devices {
		if device.LedChannels > 0 {
			lightChannels += int(device.LedChannels)
		}
	}

	// Do we have any RGB component in the system?
	if lightChannels == 0 {
		logger.Log(logger.Fields{}).Info("No RGB compatible devices found")
		return
	}

	// Reset all channels
	color := &rgb.Color{
		Red:        0,
		Green:      0,
		Blue:       0,
		Brightness: 0,
	}

	for i := 0; i < len(d.DeviceProfile.ExternalHubs); i++ {
		externalHub := d.DeviceProfile.ExternalHubs[i]
		lightChannels = 0
		for _, device := range d.Devices {
			if device.PortId == externalHub.PortId {
				lightChannels += int(device.LedChannels)
			}
		}
		for i := 0; i < lightChannels; i++ {
			reset[i] = []byte{
				byte(color.Red),
				byte(color.Green),
				byte(color.Blue),
			}
		}

		buffer = rgb.SetColor(reset)
		d.writeColor(buffer, lightChannels, externalHub.PortId)
	}

	// Are all devices under static mode?
	// In static mode, we only need to send color once;
	// there is no need for continuous packet sending.
	ledEnabledDevices, ledEnabledStaticDevices := 0, 0
	for _, device := range d.Devices {
		if device.LedChannels > 0 {
			ledEnabledDevices++ // device has LED
			if device.RGB == "static" {
				ledEnabledStaticDevices++ // led profile is set to static
			}
		}
	}

	if ledEnabledDevices > 0 || ledEnabledStaticDevices > 0 {
		if ledEnabledDevices == ledEnabledStaticDevices {
			profile := rgb.GetRgbProfile("static")
			if d.DeviceProfile.Brightness != 0 {
				profile.StartColor.Brightness = rgb.GetBrightnessValue(d.DeviceProfile.Brightness)
			}
			profileColor := rgb.ModifyBrightness(profile.StartColor)
			for i := 0; i < len(d.DeviceProfile.ExternalHubs); i++ {
				externalHub := d.DeviceProfile.ExternalHubs[i]
				lightChannels = 0
				for _, device := range d.Devices {
					if device.PortId == externalHub.PortId {
						lightChannels += int(device.LedChannels)
					}
				}

				for i := 0; i < lightChannels; i++ {
					reset[i] = []byte{
						byte(profileColor.Red),
						byte(profileColor.Green),
						byte(profileColor.Blue),
					}
				}

				buffer = rgb.SetColor(reset)
				d.writeColor(buffer, lightChannels, externalHub.PortId)
			}
			return
		}
	}

	for i := 0; i < len(d.DeviceProfile.ExternalHubs); i++ {
		externalHub := d.DeviceProfile.ExternalHubs[i]
		if externalHub.ExternalHubDeviceAmount < 1 {
			continue
		}

		go func(externalHub ExternalHubData, i int) {
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
			colorwarpGeneratedReverse := false
			d.activeRgb[i] = rgb.Exit()

			// Generate random colors
			d.activeRgb[i].RGBStartColor = rgb.GenerateRandomColor(1)
			d.activeRgb[i].RGBEndColor = rgb.GenerateRandomColor(1)

			lc := 0
			keys := make([]int, 0)
			rgbSettings := make(map[int]*rgb.ActiveRGB)

			for k := range d.Devices {
				if d.Devices[k].PortId == externalHub.PortId && d.Devices[k].LedChannels > 0 {
					rgbCustomColor := true
					lc += int(d.Devices[k].LedChannels)
					keys = append(keys, k)

					profile := rgb.GetRgbProfile(d.Devices[k].RGB)
					if profile == nil {
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
						r.RGBStartColor = d.activeRgb[i].RGBStartColor
						r.RGBEndColor = d.activeRgb[i].RGBEndColor
					}

					// Brightness
					if d.DeviceProfile.Brightness > 0 {
						r.RGBBrightness = rgb.GetBrightnessValue(d.DeviceProfile.Brightness)
						r.RGBStartColor.Brightness = r.RGBBrightness
						r.RGBEndColor.Brightness = r.RGBBrightness
					}
					rgbSettings[k] = r
				} else {
					continue
				}
			}
			sort.Ints(keys)

			for {
				buff := make([]byte, 0)
				select {
				case <-d.activeRgb[i].Exit:
					return
				default:
					for _, k := range keys {
						r := rgbSettings[k]
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
										d.activeRgb[i].RGBStartColor = d.activeRgb[i].RGBEndColor
										d.activeRgb[i].RGBEndColor = rgb.GenerateRandomColor(r.RGBBrightness)
									}
									counterColorwarp[k] = 0
								} else if counterColorwarp[k] == 0 && colorwarpGeneratedReverse == true {
									colorwarpGeneratedReverse = false
								} else {
									counterColorwarp[k]++
								}

								r.Colorwarp(counterColorwarp[k], d.activeRgb[i].RGBStartColor, d.activeRgb[i].RGBEndColor)
								lock.Unlock()
								buff = append(buff, r.Output...)
							}
						}
					}
				}
				d.writeColor(buff, lc, externalHub.PortId)
				time.Sleep(10 * time.Millisecond)
			}
		}(*externalHub, i)
	}
}

// setAutoRefresh will refresh device data
func (d *Device) setAutoRefresh() {
	timer = time.NewTicker(time.Duration(deviceRefreshInterval) * time.Millisecond)
	authRefreshChan = make(chan bool)
	go func() {
		for {
			select {
			case <-timer.C:
				d.getDeviceData()
			case <-authRefreshChan:
				timer.Stop()
				return
			}
		}
	}()
}

// updateDeviceSpeed will update device speed based on a temperature reading
func (d *Device) updateDeviceSpeed() {
	timerSpeed = time.NewTicker(time.Duration(temperaturePullingInterval) * time.Millisecond)
	go func() {
		tmp := make(map[int]string, 0)
		for {
			select {
			case <-timerSpeed.C:
				var temp float32 = 0
				for _, device := range d.Devices {
					if device.HasSpeed {
						channelSpeeds := map[int]byte{}
						profiles := temperatures.GetTemperatureProfile(device.Profile)
						if profiles == nil {
							// No such profile, default to Normal
							profiles = temperatures.GetTemperatureProfile("Normal")
						}

						switch profiles.Sensor {
						case temperatures.SensorTypeGPU:
							{
								temp = temperatures.GetNVIDIAGpuTemperature()
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
							}
						case temperatures.SensorTypeStorage:
							{
								temp = temperatures.GetStorageTemperature(profiles.Device)
								if temp == 0 {
									logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "hwmonDeviceId": profiles.Device}).Warn("Unable to get storage temperature.")
								}
							}
						}

						// All temps failed, default to 50
						if temp == 0 {
							temp = 50
						}

						for i := 0; i < len(profiles.Profiles); i++ {
							profile := profiles.Profiles[i]
							minimum := profile.Min + 0.1
							if common.InBetween(temp, minimum, profile.Max) {
								cp := fmt.Sprintf("%s-%d-%d-%d-%d", device.Profile, device.ChannelId, profile.Id, profile.Fans, profile.Pump)
								if ok := tmp[device.ChannelId]; ok != cp {
									tmp[device.ChannelId] = cp
									channelSpeeds[device.ChannelId] = byte(profile.Fans)
									d.setSpeed(channelSpeeds)
								}
							}
						}
					}
				}
			case <-speedRefreshChan:
				timerSpeed.Stop()
				return
			}
		}
	}()
}

// writeColor will write data to the device with a specific endpoint.
func (d *Device) writeColor(data []byte, lightChannels int, portId byte) {
	r := make([]byte, lightChannels)
	g := make([]byte, lightChannels)
	b := make([]byte, lightChannels)
	m := 0

	for i := 0; i < lightChannels; i++ {
		// Red
		r[i] = data[m]
		m++

		// Green
		g[i] = data[m]
		m++

		// Blue
		b[i] = data[m]
		m++
	}

	chunksR := common.ProcessMultiChunkPacket(r, maxBufferSizePerRequest)
	chunksG := common.ProcessMultiChunkPacket(g, maxBufferSizePerRequest)
	chunksB := common.ProcessMultiChunkPacket(b, maxBufferSizePerRequest)

	packetsR := make(map[int][]byte, len(chunksR))
	for i, chunk := range chunksR {
		chunkLen := len(chunk)
		chunkPacket := make([]byte, chunkLen+4)
		chunkPacket[0] = portId
		chunkPacket[1] = byte(i * maxBufferSizePerRequest)
		chunkPacket[2] = byte(chunkLen)
		chunkPacket[3] = 0x00 // R
		copy(chunkPacket[4:], chunk)
		packetsR[i] = chunkPacket
	}

	packetsG := make(map[int][]byte, len(chunksR))
	for i, chunk := range chunksG {
		chunkLen := len(chunk)
		chunkPacket := make([]byte, chunkLen+4)
		chunkPacket[0] = portId
		chunkPacket[1] = byte(i * maxBufferSizePerRequest)
		chunkPacket[2] = byte(chunkLen)
		chunkPacket[3] = 0x01 // G
		copy(chunkPacket[4:], chunk)
		packetsG[i] = chunkPacket
	}

	packetsB := make(map[int][]byte, len(chunksB))
	for i, chunk := range chunksB {
		chunkLen := len(chunk)
		chunkPacket := make([]byte, chunkLen+4)
		chunkPacket[0] = portId
		chunkPacket[1] = byte(i * maxBufferSizePerRequest)
		chunkPacket[2] = byte(chunkLen)
		chunkPacket[3] = 0x02 // B
		copy(chunkPacket[4:], chunk)
		packetsB[i] = chunkPacket
	}

	for z := 0; z < len(chunksR); z++ {
		_, err := d.transfer(cmdWriteColor, packetsR[z])
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to write red color to device")
		}

		_, err = d.transfer(cmdWriteColor, packetsG[z])
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to write green color to device")
		}

		_, err = d.transfer(cmdWriteColor, packetsB[z])
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to write blue color to device")
		}
	}

	_, err := d.transfer(cmdRefresh, []byte{0xff})
	if err != nil {
		return
	}

	_, err = d.transfer(cmdPortState, []byte{portId, 0x02})
	if err != nil {
		return
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint byte, commands []byte) ([]byte, error) {
	mutex.Lock()
	defer mutex.Unlock()

	bufferW := make([]byte, bufferSize)
	bufferW[1] = endpoint
	index := 2
	if commands != nil {
		if len(commands) > bufferSize-1 {
			commands = commands[:bufferSize-1]
		}
		copy(bufferW[index:index+len(commands)], commands)
	}

	bufferR := make([]byte, readBufferSite)

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
