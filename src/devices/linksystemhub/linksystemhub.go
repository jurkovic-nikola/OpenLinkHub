package linksystemhub

// Package: iCUE Link System Hub
// This is the primary package for iCUE Link System Hub.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/temperatures"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"os"
	"sort"
	"sync"
	"time"
)

// DeviceMonitor struct contains the shared variable and synchronization primitives
type DeviceMonitor struct {
	Status byte
	Lock   sync.Mutex
	Cond   *sync.Cond
}

type DeviceProfile struct {
	Product       string
	Serial        string
	SpeedProfiles map[int]string
	RGBProfiles   map[int]string
}

// SupportedDevice contains definition of supported devices
type SupportedDevice struct {
	DeviceId       byte   `json:"deviceId"`
	Model          byte   `json:"deviceModel"`
	Name           string `json:"deviceName"`
	LedChannels    uint8  `json:"ledChannels"`
	LcdLedChannels uint8  `json:"lcdLedChannels"`
	ContainsPump   bool   `json:"containsPump"`
	Desc           string `json:"desc"`
	AIO            bool   `json:"aio"`
}

// Devices contain information about devices connected to an iCUE Link
type Devices struct {
	ChannelId    int             `json:"channelId"`
	Type         byte            `json:"type"`
	Model        byte            `json:"-"`
	DeviceId     string          `json:"deviceId"`
	Name         string          `json:"name"`
	DefaultValue byte            `json:"-"`
	Rpm          int16           `json:"rpm"`
	Temperature  float32         `json:"temperature"`
	LedChannels  uint8           `json:"-"`
	ContainsPump bool            `json:"-"`
	Description  string          `json:"description"`
	HubId        string          `json:"-"`
	PumpModes    map[byte]string `json:"-"`
	Profile      string          `json:"profile"`
	RGB          string          `json:"rgb"`
	HasSpeed     bool
	HasTemps     bool
	AIO          bool
}

type Device struct {
	dev           *hid.Device
	lcd           *hid.Device
	Manufacturer  string           `json:"manufacturer"`
	Product       string           `json:"product"`
	Serial        string           `json:"serial"`
	Firmware      string           `json:"firmware"`
	AIO           bool             `json:"aio"`
	Devices       map[int]*Devices `json:"devices"`
	DeviceProfile *DeviceProfile
	deviceMonitor *DeviceMonitor
	profileConfig string
	activeRgb     *rgb.ActiveRGB
	Template      string
	HasLCD        bool
	VendorId      uint16
}

var (
	cmdOpenEndpoint            = []byte{0x0d, 0x01}
	cmdOpenColorEndpoint       = []byte{0x0d, 0x00}
	cmdCloseEndpoint           = []byte{0x05, 0x01, 0x01}
	cmdGetFirmware             = []byte{0x02, 0x13}
	cmdSoftwareMode            = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode            = []byte{0x01, 0x03, 0x00, 0x01}
	cmdWrite                   = []byte{0x06, 0x01}
	cmdWriteColor              = []byte{0x06, 0x00}
	cmdRead                    = []byte{0x08, 0x01}
	cmdGetDeviceMode           = []byte{0x01, 0x08, 0x01}
	cmdRefreshDevices          = []byte{0x1a, 0x01}
	cmdWaitForDevice           = []byte{0x12, 0x00}
	modeGetDevices             = []byte{0x36}
	modeGetTemperatures        = []byte{0x21}
	modeGetSpeeds              = []byte{0x17}
	modeSetSpeed               = []byte{0x18}
	modeSetColor               = []byte{0x22}
	dataTypeGetDevices         = []byte{0x21, 0x00}
	dataTypeGetTemperatures    = []byte{0x10, 0x00}
	dataTypeGetSpeeds          = []byte{0x25, 0x00}
	dataTypeSetSpeed           = []byte{0x07, 0x00}
	dataTypeSetColor           = []byte{0x12, 0x00}
	mutex                      sync.Mutex
	authRefreshChan            = make(chan bool)
	speedRefreshChan           = make(chan bool)
	bufferSize                 = 512
	headerSize                 = 3
	headerWriteSize            = 4
	bufferSizeWrite            = bufferSize + 1
	transferTimeout            = 500
	maxBufferSizePerRequest    = 508
	defaultSpeedValue          = 70
	temperaturePullingInterval = 3000
	deviceRefreshInterval      = 1000
	timer                      = &time.Ticker{}
	timerSpeed                 = &time.Ticker{}
	supportedDevices           = []SupportedDevice{
		{DeviceId: 1, Model: 0, Name: "iCUE LINK QX RGB", LedChannels: 34, ContainsPump: false, Desc: "Fan"},
		{DeviceId: 19, Model: 0, Name: "iCUE LINK RX", LedChannels: 0, ContainsPump: false, Desc: "Fan"},
		{DeviceId: 15, Model: 0, Name: "iCUE LINK RX RGB", LedChannels: 8, ContainsPump: false, Desc: "Fan"},
		{DeviceId: 4, Model: 0, Name: "iCUE LINK RX MAX", LedChannels: 8, ContainsPump: false, Desc: "Fan"},
		{DeviceId: 7, Model: 2, Name: "iCUE LINK H150i", LedChannels: 20, LcdLedChannels: 24, ContainsPump: true, Desc: "AIO", AIO: true},
		{DeviceId: 7, Model: 5, Name: "iCUE LINK H150i", LedChannels: 20, LcdLedChannels: 24, ContainsPump: true, Desc: "AIO", AIO: true},
		{DeviceId: 7, Model: 1, Name: "iCUE LINK H115i", LedChannels: 20, LcdLedChannels: 24, ContainsPump: true, Desc: "AIO", AIO: true},
		{DeviceId: 7, Model: 3, Name: "iCUE LINK H170i", LedChannels: 20, LcdLedChannels: 24, ContainsPump: true, Desc: "AIO", AIO: true},
		{DeviceId: 7, Model: 0, Name: "iCUE LINK H100i", LedChannels: 20, LcdLedChannels: 24, ContainsPump: true, Desc: "AIO", AIO: true},
		{DeviceId: 7, Model: 4, Name: "iCUE LINK H100i", LedChannels: 20, LcdLedChannels: 24, ContainsPump: true, Desc: "AIO", AIO: true},
		{DeviceId: 9, Model: 0, Name: "XC7 Elite", LedChannels: 24, ContainsPump: false, Desc: "CPU Block"},
		{DeviceId: 9, Model: 1, Name: "XC7 Elite", LedChannels: 24, ContainsPump: false, Desc: "CPU Block"},
		{DeviceId: 13, Model: 1, Name: "XG7", LedChannels: 16, ContainsPump: false, Desc: "GPU Block"},
		{DeviceId: 12, Model: 0, Name: "XD5 Elite", LedChannels: 22, ContainsPump: true, Desc: "Pump/Res"},
		{DeviceId: 12, Model: 1, Name: "XD5 Elite", LedChannels: 22, ContainsPump: true, Desc: "Pump/Res"},
		{DeviceId: 14, Model: 0, Name: "XD5 Elite LCD", LedChannels: 22, ContainsPump: true, Desc: "Pump/Res"},
		{DeviceId: 14, Model: 1, Name: "XD5 Elite LCD", LedChannels: 22, ContainsPump: true, Desc: "Pump/Res"},
		{DeviceId: 16, Model: 0, Name: "VRM Cooler Module", LedChannels: 0, ContainsPump: false, Desc: "Fan"},
		{DeviceId: 11, Model: 0, Name: "iCUE LINK TITAN H100i", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true},
		{DeviceId: 11, Model: 4, Name: "iCUE LINK TITAN H100i", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true},
		{DeviceId: 11, Model: 2, Name: "iCUE LINK TITAN H150i", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true},
		{DeviceId: 11, Model: 5, Name: "iCUE LINK TITAN H150i", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true},
		{DeviceId: 11, Model: 1, Name: "iCUE LINK TITAN H115i", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true},
		{DeviceId: 11, Model: 3, Name: "iCUE LINK TITAN H170i", LedChannels: 20, ContainsPump: true, Desc: "AIO", AIO: true},
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
		dev:      dev,
		Template: "lsh.html",
	}

	// Bootstrap
	d.getManufacturer()   // Manufacturer
	d.getProduct()        // Product
	d.getSerial()         // Serial
	d.getDeviceLcd()      // Check if LCD pump cover is installed
	d.setProfileConfig()  // Device profile
	d.getDeviceProfile()  // Get device profile if any
	d.getDeviceFirmware() // Firmware
	d.setSoftwareMode()   // Activate software mode
	d.getDevices()        // Get devices connected to a hub
	d.setColorEndpoint()  // Set device color endpoint
	d.setDefaults()       // Set default speed and color values for fans and pumps
	d.setAutoRefresh()    // Set auto device refresh
	if config.GetConfig().Manual {
		fmt.Println(
			fmt.Sprintf("[%s] Manual flag enabled. Process will not monitor temperature or adjust fan speed.", d.Serial),
		)
	} else {
		d.updateDeviceSpeed() // Update device speed
	}
	d.saveDeviceProfile() // Create device profile
	d.setDeviceColor()    // Activate device RGB
	d.newDeviceMonitor()  // Device monitor
	logger.Log(logger.Fields{"device": d}).Info("Device successfully initialized")
	return d
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	logger.Log(logger.Fields{"serial": d.Serial}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}
	timer.Stop()

	if !config.GetConfig().Manual {
		timerSpeed.Stop()
		speedRefreshChan <- true
	}

	authRefreshChan <- true
	d.setHardwareMode()
	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Fatal("Unable to close HID device")
		}
	}

	/*
		if d.lcd != nil {
			err := d.lcd.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Fatal("Unable to close LCD HID device")
			}
		}
	*/
}

// getDeviceLcd will check if AIO has LCD pump cover
func (d *Device) getDeviceLcd() {
	//lcdSerialNumber := ""
	var lcdProductId uint16 = 3150

	enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
		if info.InterfaceNbr == 0 {
			d.HasLCD = true
			//lcdSerialNumber = info.SerialNbr
		}
		return nil
	})

	// Enumerate all Corsair devices
	err := hid.Enumerate(d.VendorId, lcdProductId, enum)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId, "serial": d.Serial}).Fatal("Unable to enumerate LCD devices")
		return
	}

	/*
		TO-DO: Implement basic LED options like temperatures, speeds, etc...
			if d.HasLCD {
				logger.Log(logger.Fields{"vendorId": d.VendorId, "productId": lcdProductId, "serial": d.Serial}).Info("LCD pump cover detected")
				lcd, e := hid.Open(d.VendorId, lcdProductId, lcdSerialNumber)
				if e != nil {
					logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId, "productId": lcdProductId, "serial": d.Serial}).Error("Unable to open LCD HID device")
					return
				}
				d.lcd = lcd
			}
	*/
}

// setProfileConfig will set a static path for JSON configuration file
func (d *Device) setProfileConfig() {
	pwd, err := os.Getwd()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to get working directory")
		return
	}
	d.profileConfig = pwd + "/database/profiles/" + d.Serial + ".json"
}

// getDeviceProfile will load persistent device configuration
func (d *Device) getDeviceProfile() {
	if common.FileExists(d.profileConfig) {
		f, err := os.Open(d.profileConfig)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Unable to load profile")
			return
		}
		if err = json.NewDecoder(f).Decode(&d.DeviceProfile); err != nil {
			logger.Log(logger.Fields{"serial": d.Serial}).Warn("Unable to decode profile json")
		}
		fmt.Println("[Profiles] Device profile successfully loaded", d.profileConfig)
		err = f.Close()
		if err != nil {
			logger.Log(logger.Fields{"location": d.profileConfig}).Warn("Failed to close file handle")
		}
	} else {
		logger.Log(logger.Fields{"serial": d.Serial}).Warn("No profile found for device. Probably initial start")
	}
}

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	speedProfiles := make(map[int]string, len(d.Devices))
	rgbProfiles := make(map[int]string, len(d.Devices))
	for _, device := range d.Devices {
		speedProfiles[device.ChannelId] = device.Profile
		rgbProfiles[device.ChannelId] = device.RGB
	}
	deviceProfile := &DeviceProfile{
		Product:       d.Product,
		Serial:        d.Serial,
		SpeedProfiles: speedProfiles,
		RGBProfiles:   rgbProfiles,
	}

	// First save, assign saved profile to a device
	if d.DeviceProfile == nil {
		for _, device := range d.Devices {
			rgbProfiles[device.ChannelId] = "static"
		}
		d.DeviceProfile = deviceProfile
	}

	// Convert to JSON
	buffer, err := json.Marshal(deviceProfile)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
		return
	}

	// Create profile filename
	file, fileErr := os.Create(d.profileConfig)
	if fileErr != nil {
		logger.Log(logger.Fields{"error": err, "location": d.profileConfig}).Error("Unable to create new device profile")
		return
	}

	// Write JSON buffer to file
	_, err = file.Write(buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": d.profileConfig}).Error("Unable to write data")
		return
	}

	// Close file
	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": d.profileConfig}).Fatal("Unable to close file handle")
	}
}

// ResetSpeedProfiles will reset channel speed profile if it matches with the current speed profile
// This is used when speed profile is deleted from the UI
func (d *Device) ResetSpeedProfiles(profile string) {
	mutex.Lock()
	defer mutex.Unlock()

	i := 0
	for _, device := range d.Devices {
		if device.Profile == profile {
			d.Devices[device.ChannelId].Profile = "Normal"
			i++
		}
	}

	if i > 0 {
		// Save only if something was changed
		d.saveDeviceProfile()
	}
}

// UpdateDeviceSpeed will update device channel speed.
func (d *Device) UpdateDeviceSpeed(channelId int, value uint16) uint8 {
	// Check if actual channelId exists in the device list
	if device, ok := d.Devices[channelId]; ok {
		channelSpeeds := map[int]byte{}

		if value < 20 {
			value = 20
		}
		// Minimal pump speed should be 50%
		if device.ContainsPump {
			if value < 50 {
				value = 50
			}
		}
		channelSpeeds[device.ChannelId] = byte(value)
		d.setSpeed(channelSpeeds, 0)
		return 1
	}
	return 0
}

// UpdateSpeedProfile will update device channel speed.
// If channelId is 0, all device channels will be updated
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
		valid := false
		for _, device := range d.Devices {
			if device.AIO {
				valid = true
				break
			}
		}

		if !valid {
			return 2
		}
	}

	if channelId < 0 {
		// All devices
		for _, device := range d.Devices {
			d.Devices[device.ChannelId].Profile = profile
		}
	} else {
		// Check if actual channelId exists in the device list
		if _, ok := d.Devices[channelId]; ok {
			// Update channel with new profile
			d.Devices[channelId].Profile = profile
		}
	}

	// Save to profile
	d.saveDeviceProfile()
	return 1
}

// UpdateRgbProfile will update device RGB profile
func (d *Device) UpdateRgbProfile(channelId int, profile string) {
	if rgb.GetRgbProfile(profile) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return
	}

	if _, ok := d.Devices[channelId]; ok {
		// Update channel with new profile
		d.Devices[channelId].RGB = profile
	}

	d.DeviceProfile.RGBProfiles[channelId] = profile // Set profile
	d.saveDeviceProfile()                            // Save profile
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
}

// getLiquidTemperature will fetch temperature from AIO device
func (d *Device) getLiquidTemperature() float32 {
	for _, device := range d.Devices {
		if device.AIO {
			return device.Temperature
		}
	}
	return 0
}

// updateDeviceSpeed will update device speed based on a temperature reading
func (d *Device) updateDeviceSpeed() {
	timerSpeed = time.NewTicker(time.Duration(temperaturePullingInterval) * time.Millisecond)
	go func() {
		channelSpeeds := map[int]byte{}

		// Init speed channels
		for _, device := range d.Devices {
			channelSpeeds[device.ChannelId] = byte(defaultSpeedValue)
		}
		for {
			select {
			case <-timerSpeed.C:
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
							temp = temperatures.GetNVIDIAGpuTemperature()
							if temp == 0 {
								temp = temperatures.GetAMDGpuTemperature()
								if temp == 0 {
									logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial}).Warn("Unable to get GPU sensor temperature. Going to fallback to CPU")
									temp = temperatures.GetCpuTemperature()
									if temp == 0 {
										temp = 70
									}
								}
							}
						}
					case temperatures.SensorTypeCPU:
						{
							temp = temperatures.GetCpuTemperature()
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial}).Warn("Unable to get CPU sensor temperature.")
							}
						}
					case temperatures.SensorTypeLiquidTemperature:
						{
							temp = d.getLiquidTemperature()
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial}).Warn("Unable to get liquid temperature.")
							}
						}
					}

					// All temps failed, default to 50
					if temp == 0 {
						logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial}).Warn("ALl temperature sensors failed. Defaulting to 50")
						temp = 50
					}

					for i := 0; i < len(profiles.Profiles); i++ {
						profile := profiles.Profiles[i]
						if common.InBetween(temp, profile.Min, profile.Max) {
							// Validation
							if profile.Mode < 0 || profile.Mode > 1 {
								profile.Mode = 0
							}

							if profile.Pump < 50 || profile.Pump > 100 {
								profile.Pump = 70
							}

							var speed byte = 0x00
							if device.ContainsPump {
								speed = byte(profile.Pump)
							} else {
								speed = byte(profile.Fans)
							}
							if channelSpeeds[device.ChannelId] != speed {
								channelSpeeds[device.ChannelId] = speed
								d.setSpeed(channelSpeeds, 0)
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

// setDefaults will set default mode for all devices
func (d *Device) setDefaults() {
	channelDefaults := map[int]byte{}
	for device := range d.Devices {
		channelDefaults[device] = byte(defaultSpeedValue)
	}
	d.setSpeed(channelDefaults, 0)
}

// setSpeed will generate a speed buffer and send it to a device
func (d *Device) setSpeed(data map[int]byte, mode uint8) {
	buffer := make([]byte, len(data)*4+1)
	buffer[0] = byte(len(data))
	i := 1

	keys := make([]int, 0)

	for k := range data {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		buffer[i] = byte(k)
		buffer[i+1] = mode
		buffer[i+2] = data[k]
		i += 4
	}

	// Validate device response. In case of error, repeat packet
	response := d.write(modeSetSpeed, dataTypeSetSpeed, buffer)
	if len(response) >= 4 {
		if response[3] != 0x00 {
			m := 0
			for {
				m++
				response = d.write(modeSetSpeed, dataTypeSetSpeed, buffer)
				if response[3] == 0x00 || m > 20 {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
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
				d.setDeviceStatus()
				d.getDeviceData()
			case <-authRefreshChan:
				timer.Stop()
				return
			}
		}
	}()
}

// setDefaults will set default mode for all devices
func (d *Device) getDeviceData() {
	// Speed
	response := d.read(modeGetSpeeds, dataTypeGetSpeeds)
	amount := response[6]
	sensorData := response[7:]
	valid := response[7]
	if valid == 0x01 {
		for i := 0; i < int(amount); i++ {
			currentSensor := sensorData[i*3 : (i+1)*3]
			status := currentSensor[0]
			if status == 0x00 {
				if _, ok := d.Devices[i]; ok {
					rpm := int16(binary.LittleEndian.Uint16(currentSensor[1:3]))
					if rpm > 1 {
						d.Devices[i].Rpm = rpm
					}
				}
			}
		}
	}

	// Temperature
	response = d.read(modeGetTemperatures, dataTypeGetTemperatures)
	amount = response[6]
	sensorData = response[7:]
	valid = response[7]
	if valid == 0x01 {
		for i, s := 0, 0; i < int(amount); i, s = i+1, s+3 {
			currentSensor := sensorData[s : s+3]
			status := currentSensor[0]
			if status == 0x00 {
				if _, ok := d.Devices[i]; ok {
					temp := float32(int16(binary.LittleEndian.Uint16(currentSensor[1:3]))) / 10.0
					if temp > 1 {
						d.Devices[i].Temperature = temp
					}
				}
			}
		}
	}
}

// getSupportedDevice will return supported device or nil pointer
func (d *Device) getSupportedDevice(deviceId byte, deviceModel byte) *SupportedDevice {
	for _, device := range supportedDevices {
		if device.DeviceId == deviceId && device.Model == deviceModel {
			return &device
		}
	}
	return nil
}

// getDevices will fetch all devices connected to a hub
func (d *Device) getDevices() int {
	var devices = make(map[int]*Devices, 0)

	response := d.read(modeGetDevices, dataTypeGetDevices)
	channel := response[6]
	index := response[7:]
	position := 0

	for i := 1; i <= int(channel); i++ {
		deviceIdLen := index[position+7]
		if deviceIdLen == 0 {
			position += 8
			continue
		}
		deviceTypeModel := index[position : position+8]
		deviceId := index[position+8 : position+8+int(deviceIdLen)]

		// Get device definition
		deviceMeta := d.getSupportedDevice(deviceTypeModel[2], deviceTypeModel[3])
		if deviceMeta == nil {
			if deviceIdLen > 0 {
				position += 8 + int(deviceIdLen)
			} else {
				position += 8
			}
			continue
		}

		// Get a persistent speed profile. Fallback to Normal is anything fails
		speedProfile := "Normal"
		if d.DeviceProfile != nil {
			// Profile is set
			if sp, ok := d.DeviceProfile.SpeedProfiles[i]; ok {
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
		} else {
			logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
		}

		// Get a persistent speed profile. Fallback to Normal is anything fails
		rgbProfile := "static"
		if d.DeviceProfile != nil {
			// Profile is set
			if rp, ok := d.DeviceProfile.RGBProfiles[i]; ok {
				// Profile device channel exists
				if rgb.GetRgbProfile(rp) != nil { // Speed profile exists in configuration
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

		ledChannels := deviceMeta.LedChannels
		if d.HasLCD {
			if deviceMeta.LcdLedChannels > 0 {
				ledChannels = deviceMeta.LcdLedChannels
			} else {
				ledChannels = deviceMeta.LedChannels
			}
		}

		// Build device object
		device := &Devices{
			ChannelId:    i,
			Type:         deviceTypeModel[2],
			Model:        deviceTypeModel[3],
			DeviceId:     string(deviceId),
			Name:         deviceMeta.Name,
			DefaultValue: 0,
			Rpm:          0,
			Temperature:  0,
			LedChannels:  ledChannels,
			ContainsPump: deviceMeta.ContainsPump,
			Description:  deviceMeta.Desc,
			HubId:        d.Serial,
			Profile:      speedProfile,
			RGB:          rgbProfile,
			HasSpeed:     true,
			HasTemps:     true,
			AIO:          deviceMeta.AIO,
		}
		devices[i] = device
		position += 8 + int(deviceIdLen)
	}

	d.Devices = devices
	return len(devices)
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

	for _, device := range d.Devices {
		LedChannels := device.LedChannels
		if LedChannels > 0 {
			for i := 0; i < int(LedChannels); i++ {
				reset[i] = []byte{
					byte(color.Red),
					byte(color.Green),
					byte(color.Blue),
				}
			}
		}
	}
	buffer = rgb.SetColor(reset)
	d.writeColor(buffer)

	// Get the number of LED channels we have
	lightChannels := 0
	for _, device := range d.Devices {
		match := d.getSupportedDevice(device.Type, device.Model)
		if match == nil {
			continue
		}
		lightChannels += int(device.LedChannels)
	}

	// Do we have any RGB component in the system?
	if lightChannels == 0 {
		logger.Log(logger.Fields{}).Info("No RGB compatible devices found")
		return
	}

	// Are all devices under static mode?
	// In static mode, we only need to send color once;
	// there is no need for continuous packet sending.
	s, l := 0, 0
	for _, device := range d.Devices {
		if device.LedChannels > 0 {
			l++ // device has LED
			if device.RGB == "static" {
				s++ // led profile is set to static
			}
		}
	}
	if s > 0 || l > 0 { // We have some values
		if s == l { // number of devices matches number of devices with static profile
			profile := rgb.GetRgbProfile("static")
			profileColor := rgb.ModifyBrightness(profile.StartColor)
			for i := 0; i < lightChannels; i++ {
				reset[i] = []byte{
					byte(profileColor.Red),
					byte(profileColor.Green),
					byte(profileColor.Blue),
				}
			}
			buffer = rgb.SetColor(reset)
			d.writeColor(buffer) // Write color once
			return
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
		colorwarpGeneratedReverse := false
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
				keys := make([]int, 0)

				for k := range d.Devices {
					keys = append(keys, k)
				}
				sort.Ints(keys)

				for _, k := range keys {
					rgbCustomColor := true
					profile := rgb.GetRgbProfile(d.Devices[k].RGB)
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

					switch d.Devices[k].RGB {
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
				time.Sleep(40 * time.Millisecond)
			}
		}
	}(lightChannels)
}

// setColorEndpoint will activate hub color endpoint for further usage
func (d *Device) setColorEndpoint() {
	// Close any RGB endpoint
	_, err := d.transfer(cmdCloseEndpoint, modeSetColor, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
	}

	// Open RGB endpoint
	_, err = d.transfer(cmdOpenColorEndpoint, modeSetColor, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to open endpoint")
	}
}

// setHardwareMode will switch a device to hardware mode
func (d *Device) setHardwareMode() {
	_, err := d.transfer(cmdHardwareMode, nil, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	_, err := d.transfer(cmdSoftwareMode, nil, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
	time.Sleep(time.Duration(transferTimeout) * time.Millisecond)

	if config.GetConfig().RefreshOnStart {
		// If set to true in config.json, this will re-initialize the hub before device enumeration.
		// Experimental for now.
		// This is handy if you need to reconnect cables on HUB without power-cycle.
		_, err = d.transfer(cmdRefreshDevices, nil, nil)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
		}

		for {
			time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
			res, err := d.transfer(cmdWaitForDevice, nil, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Fatal("Unable to wait for device status")
			}
			if res[1] == 0 {
				// Device is initialized
				time.Sleep(time.Duration(transferTimeout*2) * time.Millisecond)
				break
			}
		}
	}
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
	fw, err := d.transfer(
		cmdGetFirmware,
		nil,
		nil,
	)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to a device")
	}

	v1, v2, v3 := int(fw[4]), int(fw[5]), int(binary.LittleEndian.Uint16(fw[6:8]))
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

// newDeviceMonitor initializes and returns a new Monitor
func (d *Device) newDeviceMonitor() {
	m := &DeviceMonitor{}
	m.Cond = sync.NewCond(&m.Lock)
	go d.waitForDevice(func() {
		// Device woke up after machine was sleeping
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true
		}
		d.setSoftwareMode()  // Activate software mode
		d.setColorEndpoint() // Set device color endpoint
		d.setDeviceColor()   // Set RGB
		d.newDeviceMonitor() // Device monitor
	})
	d.deviceMonitor = m
}

// setDeviceStatus sets the status and notifies a waiting goroutine if necessary
func (d *Device) setDeviceStatus() {
	mode, err := d.transfer(cmdGetDeviceMode, nil, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
	}
	if len(mode) == 0 {
		return
	}

	d.deviceMonitor.Lock.Lock()
	defer d.deviceMonitor.Lock.Unlock()
	d.deviceMonitor.Status = mode[1]
	d.deviceMonitor.Cond.Broadcast()
}

// waitForDevice waits for the status to change from zero to one and back to zero before running the action
func (d *Device) waitForDevice(action func()) {
	d.deviceMonitor.Lock.Lock()
	for d.deviceMonitor.Status != 1 {
		d.deviceMonitor.Cond.Wait()
	}
	d.deviceMonitor.Lock.Unlock()

	d.deviceMonitor.Lock.Lock()
	for d.deviceMonitor.Status != 0 {
		d.deviceMonitor.Cond.Wait()
	}
	d.deviceMonitor.Lock.Unlock()
	action()
}

// read will read data from a device and return data as a byte array
func (d *Device) read(endpoint, bufferType []byte) []byte {
	// Endpoint data
	var buffer []byte

	// Close specified endpoint
	_, err := d.transfer(cmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
	}

	// Open endpoint
	_, err = d.transfer(cmdOpenEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to open endpoint")
	}

	// Read data from endpoint
	buffer, err = d.transfer(cmdRead, endpoint, bufferType)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to read endpoint")
	}

	// Close specified endpoint
	_, err = d.transfer(cmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
	}
	return buffer
}

// write will write data to the device with specific endpoint
func (d *Device) write(endpoint, bufferType, data []byte) []byte {
	// Buffer
	buffer := make([]byte, len(bufferType)+len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)+2))
	copy(buffer[headerWriteSize:headerWriteSize+len(bufferType)], bufferType)
	copy(buffer[headerWriteSize+len(bufferType):], data)

	// Create read buffer
	bufferR := make([]byte, bufferSize)

	// Close endpoint
	_, err := d.transfer(cmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
	}

	// Open endpoint
	_, err = d.transfer(cmdOpenEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to open endpoint")
	}

	// Send it
	bufferR, err = d.transfer(cmdWrite, buffer, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to endpoint")
	}

	// Close endpoint
	_, err = d.transfer(cmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
	}
	return bufferR
}

// writeColor will write data to the device with a specific endpoint.
// writeColor does not require endpoint closing and opening like normal Write requires.
// Endpoint is open only once. Once the endpoint is open, color can be sent continuously.
func (d *Device) writeColor(data []byte) {
	// Buffer
	buffer := make([]byte, len(dataTypeSetColor)+len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)+2))
	copy(buffer[headerWriteSize:headerWriteSize+len(dataTypeSetColor)], dataTypeSetColor)
	copy(buffer[headerWriteSize+len(dataTypeSetColor):], data)

	// Process buffer and create a chunked array if needed
	writeColorEp := cmdWriteColor
	colorEp := make([]byte, len(writeColorEp))
	copy(colorEp, writeColorEp)

	chunks := common.ProcessMultiChunkPacket(buffer, maxBufferSizePerRequest)
	for i, chunk := range chunks {
		// Next color endpoint based on number of chunks
		colorEp[0] = colorEp[0] + byte(i)
		// Send it
		_, err := d.transfer(colorEp, chunk, nil)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to endpoint")
		}
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint, buffer, bufferType []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	mutex.Lock()
	defer mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[2] = 0x01
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
	}

	// Get data from a device
	if _, err := d.dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
	}

	// Read remaining data from a device
	if len(bufferType) == 2 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(transferTimeout)*time.Millisecond)
		defer cancel()

		for ctx.Err() != nil && !responseMatch(bufferR, bufferType) {
			if _, err := d.dev.Read(bufferR); err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
			}
		}
		if ctx.Err() != nil {
			logger.Log(logger.Fields{"error": ctx.Err(), "serial": d.Serial}).Error("Unable to read data from device")
		}
	}
	return bufferR, nil
}

// responseMatch will check if two byte arrays match
func responseMatch(response, expected []byte) bool {
	responseBuffer := response[5:7]
	return bytes.Equal(responseBuffer, expected)
}
