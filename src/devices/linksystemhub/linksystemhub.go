package linksystemhub

// Package: iCUE Link System Hub
// This is the primary package for iCUE Link System Hub.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
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
	Product         string
	Serial          string
	SpeedProfiles   map[int]string
	RGBProfiles     map[int]string
	ExternalHub     bool
	ExternalType    int
	ExternalDevices int
}

// SupportedDevice contains definition of supported devices
type SupportedDevice struct {
	DeviceId     byte   `json:"deviceId"`
	Model        byte   `json:"deviceModel"`
	Name         string `json:"deviceName"`
	LedChannels  uint8  `json:"ledChannels"`
	ContainsPump bool   `json:"containsPump"`
	Desc         string `json:"desc"`
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
}

type Device struct {
	dev           *hid.Device
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
	ExternalHub   bool
	RGBDeviceOnly bool
	Template      string
}

var (
	cmdOpenEndpoint                   = []byte{0x0d, 0x01}
	cmdOpenColorEndpoint              = []byte{0x0d, 0x00}
	cmdCloseEndpoint                  = []byte{0x05, 0x01, 0x01}
	cmdGetFirmware                    = []byte{0x02, 0x13}
	cmdSoftwareMode                   = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode                   = []byte{0x01, 0x03, 0x00, 0x01}
	cmdWrite                          = []byte{0x06, 0x01}
	cmdWriteColor                     = []byte{0x06, 0x00}
	cmdRead                           = []byte{0x08, 0x01}
	cmdGetDeviceMode                  = []byte{0x01, 0x08, 0x01}
	modeGetDevices                    = []byte{0x36}
	modeGetTemperatures               = []byte{0x21}
	modeGetSpeeds                     = []byte{0x17}
	modeSetSpeed                      = []byte{0x18}
	modeSetColor                      = []byte{0x22}
	dataTypeGetDevices                = []byte{0x21, 0x00}
	dataTypeGetTemperatures           = []byte{0x10, 0x00}
	dataTypeGetSpeeds                 = []byte{0x25, 0x00}
	dataTypeSetSpeed                  = []byte{0x07, 0x00}
	dataTypeSetColor                  = []byte{0x12, 0x00}
	mutex                             sync.Mutex
	authRefreshChan                   = make(chan bool)
	speedRefreshChan                  = make(chan bool)
	bufferSize                        = 512
	headerSize                        = 3
	headerWriteSize                   = 4
	bufferSizeWrite                   = bufferSize + 1
	transferTimeout                   = 500
	maxBufferSizePerRequest           = 508
	defaultSpeedValue                 = 70
	defaultTemperaturePullingInterval = 3000
	defaultDeviceRefreshInterval      = 1000
	timer                             = &time.Ticker{}
	timerSpeed                        = &time.Ticker{}
	supportedDevices                  = []SupportedDevice{
		{
			DeviceId:     1,
			Model:        0,
			Name:         "QX Fan",
			LedChannels:  34,
			ContainsPump: false,
			Desc:         "Fan",
		},
		{
			DeviceId:     19,
			Model:        0,
			Name:         "RX Fan",
			LedChannels:  0,
			ContainsPump: false,
			Desc:         "Fan",
		},
		{
			DeviceId:     15,
			Model:        0,
			Name:         "RX RGB Fan",
			LedChannels:  8,
			ContainsPump: false,
			Desc:         "Fan",
		},
		{
			DeviceId:     7,
			Model:        2,
			Name:         "H150i",
			LedChannels:  20,
			ContainsPump: true,
			Desc:         "AIO",
		},
		{
			DeviceId:     7,
			Model:        5,
			Name:         "H150i",
			LedChannels:  20,
			ContainsPump: true,
			Desc:         "AIO",
		},
		{
			DeviceId:     7,
			Model:        1,
			Name:         "H115i",
			LedChannels:  20,
			ContainsPump: true,
			Desc:         "AIO",
		},
		{
			DeviceId:     7,
			Model:        3,
			Name:         "H170i",
			LedChannels:  20,
			ContainsPump: true,
			Desc:         "AIO",
		},
		{
			DeviceId:     7,
			Model:        0,
			Name:         "H100i",
			LedChannels:  20,
			ContainsPump: true,
			Desc:         "AIO",
		},
		{
			DeviceId:     7,
			Model:        4,
			Name:         "H100i",
			LedChannels:  20,
			ContainsPump: true,
			Desc:         "AIO",
		},
		{
			DeviceId:     9,
			Model:        0,
			Name:         "XC7 Elite",
			LedChannels:  24,
			ContainsPump: false,
			Desc:         "CPU Block",
		},
		{
			DeviceId:     9,
			Model:        1,
			Name:         "XC7 Elite",
			LedChannels:  24,
			ContainsPump: false,
			Desc:         "CPU Block",
		},
		{
			DeviceId:     13,
			Model:        1,
			Name:         "XG7",
			LedChannels:  16,
			ContainsPump: false,
			Desc:         "GPU Block",
		},
		{
			DeviceId:     12,
			Model:        0,
			Name:         "XD5 Elite",
			LedChannels:  22,
			ContainsPump: true,
			Desc:         "Pump/Res",
		},
		{
			DeviceId:     12,
			Model:        1,
			Name:         "XD5 Elite",
			LedChannels:  22,
			ContainsPump: true,
			Desc:         "Pump/Res",
		},
		{
			DeviceId:     14,
			Model:        0,
			Name:         "XD5 Elite LCD",
			LedChannels:  22,
			ContainsPump: true,
			Desc:         "Pump/Res",
		},
		{
			DeviceId:     14,
			Model:        1,
			Name:         "XD5 Elite LCD",
			LedChannels:  22,
			ContainsPump: true,
			Desc:         "Pump/Res",
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
		dev:      dev,
		Template: "lsh.html",
	}

	// Bootstrap
	d.getManufacturer()   // Manufacturer
	d.getProduct()        // Product
	d.getSerial()         // Serial
	d.setProfileConfig()  // Device profile
	d.getDeviceProfile()  // Get device profile if any
	d.getDeviceFirmware() // Firmware
	d.setSoftwareMode()   // Activate software mode
	d.getDevices()        // Get devices connected to a hub
	d.setColorEndpoint()  // Set device color endpoint
	d.setDefaults()       // Set default speed and color values for fans and pumps
	d.setAutoRefresh()    // Set auto device refresh
	d.updateDeviceSpeed() // Update device speed
	d.saveDeviceProfile() // Create device profile
	d.setDeviceColor()    // Activate device RGB

	d.newDeviceMonitor() // Device monitor
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
	timerSpeed.Stop()
	authRefreshChan <- true
	speedRefreshChan <- true
	d.setHardwareMode()
	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Fatal("Unable to close HID device")
		}
	}
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
	for _, linkDevice := range d.Devices {
		speedProfiles[linkDevice.ChannelId] = linkDevice.Profile
		rgbProfiles[linkDevice.ChannelId] = linkDevice.RGB
	}
	deviceProfile := &DeviceProfile{
		Product:       d.Product,
		Serial:        d.Serial,
		SpeedProfiles: speedProfiles,
		RGBProfiles:   rgbProfiles,
	}

	// First save, assign saved profile to a device
	if d.DeviceProfile == nil {
		for _, linkDevice := range d.Devices {
			rgbProfiles[linkDevice.ChannelId] = "static"
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
	for _, linkDevice := range d.Devices {
		if linkDevice.Profile == profile {
			d.Devices[linkDevice.ChannelId].Profile = "Normal"
			i++
		}
	}

	if i > 0 {
		// Save only if something was changed
		d.saveDeviceProfile()
	}
}

// UpdateSpeedProfile will update device channel speed.
// If channelId is 0, all device channels will be updated
func (d *Device) UpdateSpeedProfile(channelId int, profile string) {
	mutex.Lock()
	defer mutex.Unlock()

	if channelId < 0 {
		// All devices
		for _, linkDevice := range d.Devices {
			d.Devices[linkDevice.ChannelId].Profile = profile
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
	d.activeRgb.Exit <- true                         // Exit current RGB mode
	d.setDeviceColor()                               // Restart RGB
}

// updateDeviceSpeed will update device speed based on a temperature reading
func (d *Device) updateDeviceSpeed() {
	timerSpeed = time.NewTicker(time.Duration(defaultTemperaturePullingInterval) * time.Millisecond)
	tmp := make(map[int]string, 0)

	go func() {
		for {
			select {
			case <-timerSpeed.C:
				var temp float32 = 0
				for _, linkDevice := range d.Devices {
					channelSpeeds := map[int][]byte{}
					profiles := temperatures.GetTemperatureProfile(linkDevice.Profile)
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
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial}).Warn("Unable to get sensor temperature. Defaulting to 70")
								temp = 70
							}
						}
					}

					for i := 0; i < len(profiles.Profiles); i++ {
						profile := profiles.Profiles[i]
						if common.InBetween(temp, profile.Min, profile.Max) {
							cp := fmt.Sprintf("%s-%d-%d-%d-%d", linkDevice.Profile, linkDevice.ChannelId, profile.Id, profile.Fans, profile.Pump)
							if ok := tmp[linkDevice.ChannelId]; ok != cp {
								tmp[linkDevice.ChannelId] = cp

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

								if linkDevice.ContainsPump {
									channelSpeeds[linkDevice.ChannelId] = []byte{byte(profile.Pump)}
								} else {
									channelSpeeds[linkDevice.ChannelId] = []byte{byte(profile.Fans)}
								}
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
	channelDefaults := map[int][]byte{}
	for linkDevice := range d.Devices {
		channelDefaults[linkDevice] = []byte{byte(defaultSpeedValue)}
	}
	d.setSpeed(channelDefaults, 0)
}

// setSpeed will generate a speed buffer and send it to a device
func (d *Device) setSpeed(data map[int][]byte, mode uint8) {
	buffer := make([]byte, len(data)*4+1)
	buffer[0] = byte(len(data))
	i := 1
	for channel, speed := range data {
		v := 2
		buffer[i] = byte(channel)
		buffer[i+1] = mode // Either percent mode or RPM mode
		for value := range speed {
			buffer[i+v] = speed[value]
			v++
		}
		i += 4 // Move to the next place
	}
	d.write(modeSetSpeed, dataTypeSetSpeed, buffer)
}

// setAutoRefresh will refresh device data
func (d *Device) setAutoRefresh() {
	timer = time.NewTicker(time.Duration(defaultDeviceRefreshInterval) * time.Millisecond)
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
	for i := 0; i < int(amount); i++ {
		currentSensor := sensorData[i*3 : (i+1)*3]
		status := currentSensor[0]
		if status == 0x00 {
			if _, ok := d.Devices[i]; ok {
				d.Devices[i].Rpm = int16(binary.LittleEndian.Uint16(currentSensor[1:3]))
			}
		}
	}

	// Temperature
	response = d.read(modeGetTemperatures, dataTypeGetTemperatures)
	amount = response[6]
	sensorData = response[7:]
	for i, s := 0, 0; i < int(amount); i, s = i+1, s+3 {
		currentSensor := sensorData[s : s+3]
		status := currentSensor[0]
		if status == 0x00 {
			if _, ok := d.Devices[i]; ok {
				d.Devices[i].Temperature = float32(int16(binary.LittleEndian.Uint16(currentSensor[1:3]))) / 10.0
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

		// Build device object
		linkHubDevice := &Devices{
			ChannelId:    i,
			Type:         deviceTypeModel[2],
			Model:        deviceTypeModel[3],
			DeviceId:     string(deviceId),
			Name:         deviceMeta.Name,
			DefaultValue: 0,
			Rpm:          0,
			Temperature:  0,
			LedChannels:  deviceMeta.LedChannels,
			ContainsPump: deviceMeta.ContainsPump,
			Description:  deviceMeta.Desc,
			HubId:        d.Serial,
			Profile:      speedProfile,
			RGB:          rgbProfile,
			HasSpeed:     true,
			HasTemps:     true,
		}
		devices[i] = linkHubDevice
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

	d.activeRgb = rgb.Exit()

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
func (d *Device) write(endpoint, bufferType, data []byte) {
	// Buffer
	buffer := make([]byte, len(bufferType)+len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)+2))
	copy(buffer[headerWriteSize:headerWriteSize+len(bufferType)], bufferType)
	copy(buffer[headerWriteSize+len(bufferType):], data)

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
	_, err = d.transfer(cmdWrite, buffer, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to endpoint")
	}

	// Close endpoint
	_, err = d.transfer(cmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to close endpoint")
	}
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
