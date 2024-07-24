package cc

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

// Package: CORSAIR iCUE COMMANDER CORE
// This is the primary package for CORSAIR iCUE COMMANDER CORE.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later
// Supported devices:
// - iCUE COMMANDER CORE - 0c32
// - iCUE COMMANDER CORE - 0c1c
// - iCUE H100i ELITE CAPELLIX
// - iCUE H115i ELITE CAPELLIX
// - iCUE H150i ELITE CAPELLIX
// - iCUE H170i ELITE CAPELLIX
// - H100i ELITE LCD
// - H150i ELITE LCD
// - H170i ELITE LCD
// - iCUE H100i ELITE CAPELLIX XT
// - iCUE H115i ELITE CAPELLIX XT
// - iCUE H150i ELITE CAPELLIX XT
// - iCUE H170i ELITE CAPELLIX XT
// - 1x Temperature Probe

var (
	cmdOpenEndpoint            = []byte{0x0d, 0x01}
	cmdOpenColorEndpoint       = []byte{0x0d, 0x00}
	cmdCloseEndpoint           = []byte{0x05, 0x01, 0x01}
	cmdGetFirmware             = []byte{0x02, 0x13}
	cmdDeviceType              = []byte{0x02, 0x58, 0x00}
	cmdSoftwareMode            = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode            = []byte{0x01, 0x03, 0x00, 0x01}
	cmdWrite                   = []byte{0x06, 0x01}
	cmdWriteColor              = []byte{0x06, 0x00}
	cmdRead                    = []byte{0x08, 0x01}
	cmdGetDeviceMode           = []byte{0x02, 0x03, 0x00}
	cmdSetLedPorts             = []byte{0x1e}
	modeGetLeds                = []byte{0x20}
	modeGetSpeeds              = []byte{0x17}
	modeSetSpeed               = []byte{0x18}
	modeGetTemperatures        = []byte{0x21}
	modeGetFans                = []byte{0x1a}
	modeSetColor               = []byte{0x22}
	dataTypeGetTemperatures    = []byte{0x10, 0x00}
	dataTypeGetSpeeds          = []byte{0x06, 0x00}
	dataTypeSetSpeed           = []byte{0x07, 0x00}
	dataTypeGetFans            = []byte{0x09, 0x00}
	dataTypeGetLeds            = []byte{0x0f, 0x00}
	dataTypeSetColor           = []byte{0x12, 0x00}
	dataTypeSubColor           = []byte{0x07, 0x00}
	mutex                      sync.Mutex
	bufferSize                 = 64
	bufferSizeWrite            = bufferSize + 1
	transferTimeout            = 500
	headerSize                 = 2
	headerWriteSize            = 4
	authRefreshChan            = make(chan bool)
	speedRefreshChan           = make(chan bool)
	deviceRefreshInterval      = 1000
	defaultSpeedValue          = 50
	temperaturePullingInterval = 3000
	ledStartIndex              = 6
	maxBufferSizePerRequest    = 61
	timer                      = &time.Ticker{}
	timerSpeed                 = &time.Ticker{}
	internalLedDevices         = make(map[int]*LedChannel, 7)
	aioList                    = []AIOList{
		{Name: "H100i ELITE CAPELLIX", PumpVersion: 1, RadiatorSize: 240},
		{Name: "H100i ELITE CAPELLIX", PumpVersion: 2, RadiatorSize: 240},
		{Name: "H115i ELITE CAPELLIX", PumpVersion: 1, RadiatorSize: 280},
		{Name: "H150i ELITE CAPELLIX", PumpVersion: 1, RadiatorSize: 360},
		{Name: "H150i ELITE CAPELLIX", PumpVersion: 2, RadiatorSize: 360},
		{Name: "H170i ELITE CAPELLIX", PumpVersion: 1, RadiatorSize: 420},
		{Name: "H100i ELITE LCD", PumpVersion: 3, RadiatorSize: 240},
		{Name: "H150i ELITE LCD", PumpVersion: 3, RadiatorSize: 360},
		{Name: "H170i ELITE LCD", PumpVersion: 3, RadiatorSize: 420},
		{Name: "H100i ELITE LCD XT", PumpVersion: 5, RadiatorSize: 240},
		{Name: "H100i ELITE LCD XT", PumpVersion: 6, RadiatorSize: 240},
		{Name: "H115i ELITE LCD XT", PumpVersion: 5, RadiatorSize: 280},
		{Name: "H150i ELITE LCD XT", PumpVersion: 5, RadiatorSize: 360},
		{Name: "H150i ELITE LCD XT", PumpVersion: 6, RadiatorSize: 360},
		{Name: "H170i ELITE LCD XT", PumpVersion: 5, RadiatorSize: 420},
	}
)

// DeviceMonitor struct contains the shared variable and synchronization primitives
type DeviceMonitor struct {
	Status byte
	Lock   sync.Mutex
	Cond   *sync.Cond
}

type LedChannel struct {
	Total   uint8
	Command byte
}

type DeviceProfile struct {
	Product       string
	Serial        string
	RGBProfiles   map[int]string
	SpeedProfiles map[int]string
}

// Devices contain information about devices connected to a Commander Core
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
	HasSpeed           bool
	HasTemps           bool
	IsTemperatureProbe bool
}

type Device struct {
	dev           *hid.Device
	Manufacturer  string           `json:"manufacturer"`
	Product       string           `json:"product"`
	Serial        string           `json:"serial"`
	Firmware      string           `json:"firmware"`
	RGB           string           `json:"rgb"`
	AIOType       string           `json:"-"`
	Devices       map[int]*Devices `json:"devices"`
	DeviceProfile *DeviceProfile
	deviceMonitor *DeviceMonitor
	profileConfig string
	activeRgb     *rgb.ActiveRGB
	Template      string
}

type AIOList struct {
	Name         string
	PumpVersion  int16
	RadiatorSize int16
}

/*
// Hard reset of all device LED ports
// Uses ONLY if you brick your device regarding LED (e.g., stuck on red color permanently)
func (d *Device) hardLedReset() {
	buf := make([]byte, 16)
	m := 4
	buf[0] = 0x0d
	buf[1] = 0x00
	buf[2] = 0x07
	buf[3] = 0x00
	for i := 0; i < 6; i++ {
		buf[m] = 0x01
		m++
		buf[m] = 0x06
		m++
	}
	d.write(cmdSetLedPorts, nil, buf, false)
}
*/

// Init will initialize a new device
func Init(vendorId, productId uint16, serial string) *Device {
	// Open device, return if failure
	dev, err := hid.Open(vendorId, productId, serial)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId}).Error("Unable to open HID device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:      dev,
		Template: "cc.html",
	}

	// There are 2 CCs. One has a packet size of 64 and the other has 96.
	// This matters only for RGB operations due to packet chunking.
	if productId == 3100 { // 0c1c
		bufferSize = 96
		bufferSizeWrite = bufferSize + 1
		maxBufferSizePerRequest = 93
	}

	// Bootstrap
	d.getManufacturer()   // Manufacturer
	d.getProduct()        // Product
	d.getSerial()         // Serial
	d.setProfileConfig()  // Device profile
	d.getDeviceProfile()  // Get device profile if any
	d.getDeviceFirmware() // Firmware
	d.setSoftwareMode()   // Activate software mode
	d.initLedPorts()      // Init LED ports
	d.getDeviceType()     // Find an AIO device type
	d.getLedDevices()     // Get LED devices
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
	d.resetLEDPorts()     // Reset device LED
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

	v1, v2, v3 := int(fw[3]), int(fw[4]), int(binary.LittleEndian.Uint16(fw[5:7]))
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

// getLedDevices will get all connected LED data
func (d *Device) getLedDevices() {
	// LED channels
	lc := d.read(modeGetLeds, dataTypeGetLeds)
	ld := lc[ledStartIndex:] // Channel data starts from position 10 and 4x increments per channel
	amount := 7
	for i := 0; i < amount; i++ {
		var numLEDs uint16 = 0
		var command byte = 00

		// Initialize LED channel data
		leds := &LedChannel{
			Total:   0,
			Command: 00,
		}

		// Check if device status is 2, aka connected
		connected := binary.LittleEndian.Uint16(ld[i*4:i*4+2]) == 2
		if connected {
			// Get number of LEDs
			numLEDs = binary.LittleEndian.Uint16(ld[i*4+2 : i*4+2+2])

			// Each LED device has different command code
			switch numLEDs {
			case 4:
				{
					command = 03
				}
			case 8:
				{
					command = 05
				}
			case 10:
				{
					command = 01
				}
			case 12:
				{
					command = 04
				}
			case 16:
				{
					command = 02
				}
			case 29:
				{
					// Pump, no command codes here
				}
			case 34:
				{
					command = 06
				}
			}

			// Set values
			leds.Total = uint8(numLEDs)
			leds.Command = command
		}

		// Add to a device map
		internalLedDevices[i] = leds
	}
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
					if d.Devices[k].IsTemperatureProbe {
						continue
					}

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

// getDevices will fetch all devices connected to a hub
func (d *Device) getDevices() int {
	var devices = make(map[int]*Devices, 0)
	var m = 0

	// Fans
	response := d.read(modeGetFans, dataTypeGetFans)
	amount := d.getChannelAmount(response)
	for i := 0; i < amount; i++ {
		status := response[6:][i]
		if status == 0x07 {
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

			// Get LED data
			var LedChannels uint8 = 0
			if internalLedDevice, ok := internalLedDevices[m]; ok {
				LedChannels = internalLedDevice.Total
			}

			// Build device object
			device := &Devices{
				ChannelId:   i,
				DeviceId:    fmt.Sprintf("%s-%v", "Fan", i),
				Name:        fmt.Sprintf("Fan %d", i),
				Rpm:         0,
				Temperature: 0,
				Description: "Fan",
				LedChannels: LedChannels,
				HubId:       d.Serial,
				Profile:     speedProfile,
				HasSpeed:    true,
				HasTemps:    false,
			}

			if i == 0 { // Pump is on channel 0
				device.HasSpeed = true
				device.HasTemps = true
				device.Description = "AIO"
				device.Name = d.AIOType
				device.DeviceId = fmt.Sprintf("%s-%v", "AIO", i)
				device.ContainsPump = true
			}

			device.RGB = rgbProfile
			devices[m] = device
			m++
		}
	}

	// Temperature probe
	response = d.read(modeGetTemperatures, dataTypeGetTemperatures)
	sensorData := response[9:]
	for i, s := 0, 0; i < 1; i, s = i+1, s+3 {
		status := sensorData[s : s+3][0]
		if status == 0x00 {
			// Build device object
			device := &Devices{
				ChannelId:          m,
				DeviceId:           fmt.Sprintf("%s-%v", "Probe", i),
				Name:               fmt.Sprintf("Temperature Probe %d", i),
				Rpm:                0,
				Temperature:        0,
				Description:        "Probe",
				HubId:              d.Serial,
				HasSpeed:           false,
				HasTemps:           true,
				IsTemperatureProbe: true,
			}
			devices[m] = device
			m++
		}
	}

	d.Devices = devices
	return len(devices)
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
	d.write(modeSetSpeed, dataTypeSetSpeed, buffer, true)
}

// ResetSpeedProfiles will reset channel speed profile if it matches with the current speed profile
// This is used when speed profile is deleted from the UI
func (d *Device) ResetSpeedProfiles(profile string) {
	mutex.Lock()
	defer mutex.Unlock()

	i := 0
	for _, device := range d.Devices {
		if device.HasSpeed {
			if device.Profile == profile {
				d.Devices[device.ChannelId].Profile = "Normal"
				i++
			}
		}
	}

	if i > 0 {
		// Save only if something was changed
		d.saveDeviceProfile()
	}
}

// updateDeviceSpeed will update device speed based on a temperature reading
func (d *Device) updateDeviceSpeed() {
	timerSpeed = time.NewTicker(time.Duration(temperaturePullingInterval) * time.Millisecond)
	tmp := make(map[int]string, 0)

	go func() {
		for {
			select {
			case <-timerSpeed.C:
				var temp float32 = 0
				//if temp > 0 {
				for _, device := range d.Devices {
					if device.IsTemperatureProbe {
						continue
					}

					channelSpeeds := map[int][]byte{}
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
					}

					for i := 0; i < len(profiles.Profiles); i++ {
						profile := profiles.Profiles[i]
						if common.InBetween(temp, profile.Min, profile.Max) {
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

								if device.ContainsPump {
									channelSpeeds[device.ChannelId] = []byte{byte(profile.Pump)}
								} else {
									channelSpeeds[device.ChannelId] = []byte{byte(profile.Fans)}
								}
								d.setSpeed(channelSpeeds, 0)
							}
						}
					}
				}
				//}
			case <-speedRefreshChan:
				timerSpeed.Stop()
				return
			}
		}
	}()
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

// getDeviceType will set a type of AIO
func (d *Device) getDeviceType() {
	deviceType, err := d.transfer(cmdDeviceType, nil, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Fatal("Unable to write to a device")
	}

	// We match a device with radiator size and pump version
	radiatorSize := int16(binary.LittleEndian.Uint16(deviceType[3:6]))
	pumpVersion := int16(deviceType[1])

	for _, aioType := range aioList {
		if aioType.RadiatorSize == radiatorSize && aioType.PumpVersion == pumpVersion {
			d.AIOType = aioType.Name
			break
		}
	}

	if len(d.AIOType) == 0 {
		d.AIOType = "Unknown AIO"
	}
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

// getChannelAmount will return a number of available channels
func (d *Device) getChannelAmount(data []byte) int {
	return int(data[5])
}

// getDeviceData will fetch device data
func (d *Device) getDeviceData() {
	// Channels
	channels := d.read(modeGetFans, dataTypeGetFans)
	if channels == nil {
		return
	}
	var m = 0

	// Speed
	response := d.read(modeGetSpeeds, dataTypeGetSpeeds)
	amount := d.getChannelAmount(channels)
	sensorData := response[6:]
	for i, s := 0, 0; i < amount; i, s = i+1, s+2 {
		currentSensor := sensorData[s : s+2]
		status := channels[6:][i]
		if status == 0x07 {
			if _, ok := d.Devices[m]; ok {
				d.Devices[m].Rpm = int16(binary.LittleEndian.Uint16(currentSensor))
				m++
			}
		}
	}

	// Temperature
	response = d.read(modeGetTemperatures, dataTypeGetTemperatures)
	amount = d.getChannelAmount(response)
	sensorData = response[6:]
	for i, s := 0, 0; i < amount; i, s = i+1, s+3 {
		currentSensor := sensorData[s : s+3]
		status := currentSensor[0]
		if status == 0x00 {
			if i == 0 {
				if _, ok := d.Devices[i]; ok {
					d.Devices[i].Temperature = float32(int16(binary.LittleEndian.Uint16(currentSensor[1:3]))) / 10.0
				}
			} else {
				if _, ok := d.Devices[m]; ok {
					d.Devices[m].Temperature = float32(int16(binary.LittleEndian.Uint16(currentSensor[1:3]))) / 10.0
				}
			}
		}
	}
}

// setDefaults will set default mode for all devices
func (d *Device) setDefaults() {
	channelDefaults := map[int][]byte{}
	for device := range d.Devices {
		if d.Devices[device].HasSpeed {
			channelDefaults[device] = []byte{byte(defaultSpeedValue)}
		}
	}
	d.setSpeed(channelDefaults, 0)
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

// UpdateDeviceSpeed will update device channel speed.
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
		d.setSpeed(channelSpeeds, 0)
		return 1
	}
	return 0
}

// UpdateSpeedProfile will update device channel speed.
// If channelId is 0, all device channels will be updated
func (d *Device) UpdateSpeedProfile(channelId int, profile string) {
	mutex.Lock()
	defer mutex.Unlock()

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

// initLedPorts will prep LED physical ports for reading
func (d *Device) initLedPorts() {
	for i := 1; i <= 6; i++ {
		var command = []byte{0x14, byte(i), 0x01}
		_, err := d.transfer(command, nil, nil)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "port": i}).Error("Failed to initialize LED ports")
		}
	}

	// We need to wait around 500 ms for physical ports to re-initialize
	// After that we can grab any new connected / disconnected device values
	time.Sleep(time.Duration(transferTimeout) * time.Millisecond)
}

// resetLEDPorts will reset hubs LED ports and configure currently connected LED device
func (d *Device) resetLEDPorts() {
	var buf []byte

	buf = append(buf, 0x0d)
	buf = append(buf, 0x00)
	buf = append(buf, 0x07)
	buf = append(buf, 0x01)
	buf = append(buf, 0x08)

	// Start at 1, since 0 is the pump, and iterate through all 6 physical connectors
	for i := 1; i <= 6; i++ {
		if z, ok := internalLedDevices[i]; ok {
			// Channel activation
			buf = append(buf, 0x01)

			// Fan LED command code, each LED device has different command code
			buf = append(buf, z.Command)
		} else {
			// Channel is not active
			buf = append(buf, 0x00)
		}
	}
	d.write(cmdSetLedPorts, nil, buf, false)
}

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	speedProfiles := make(map[int]string, len(d.Devices))
	rgbProfiles := make(map[int]string, len(d.Devices))
	for _, device := range d.Devices {
		if device.IsTemperatureProbe {
			continue
		}
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
			if device.IsTemperatureProbe {
				continue
			}
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

// newDeviceMonitor initializes and returns a new Monitor
func (d *Device) newDeviceMonitor() {
	m := &DeviceMonitor{}
	m.Cond = sync.NewCond(&m.Lock)
	go d.waitForDevice(func() {
		// Device woke up after machine was sleeping
		if d.activeRgb != nil {
			d.activeRgb.Exit <- true
			d.activeRgb = nil
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
	d.deviceMonitor.Status = mode[3]
	d.deviceMonitor.Cond.Broadcast()
}

// waitForDevice waits for the status to change from zero to one and back to zero before running the action
func (d *Device) waitForDevice(action func()) {
	d.deviceMonitor.Lock.Lock()
	for d.deviceMonitor.Status != 2 {
		d.deviceMonitor.Cond.Wait()
	}
	d.deviceMonitor.Lock.Unlock()

	d.deviceMonitor.Lock.Lock()
	for d.deviceMonitor.Status != 1 {
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
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to close endpoint")
	}

	// Open endpoint
	_, err = d.transfer(cmdOpenEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to open endpoint")
	}

	// Read data from endpoint
	buffer, err = d.transfer(cmdRead, endpoint, bufferType)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read endpoint")
	}

	// Close specified endpoint
	_, err = d.transfer(cmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to close endpoint")
	}
	return buffer
}

// writeColor will write data to the device with a specific endpoint.
// writeColor does not require endpoint closing and opening like normal Write requires.
// Endpoint is open only once. Once the endpoint is open, color can be sent continuously.
func (d *Device) writeColor(data []byte) {
	buffer := make([]byte, len(dataTypeSetColor)+len(data)+headerWriteSize)
	binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)+2))
	copy(buffer[headerWriteSize:headerWriteSize+len(dataTypeSetColor)], dataTypeSetColor)
	copy(buffer[headerWriteSize+len(dataTypeSetColor):], data)

	// Split packet into chunks
	chunks := common.ProcessMultiChunkPacket(buffer, maxBufferSizePerRequest)
	for i, chunk := range chunks {
		if i == 0 {
			// Initial packet is using cmdWriteColor
			_, err := d.transfer(cmdWriteColor, chunk, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
			}
		} else {
			// Chunks don't use cmdWriteColor, they use static dataTypeSubColor
			_, err := d.transfer(dataTypeSubColor, chunk, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to endpoint")
			}
		}
	}
}

// write will write data to the device with specific endpoint
func (d *Device) write(endpoint, bufferType, data []byte, extra bool) {
	// Buffer
	buffer := make([]byte, len(bufferType)+len(data)+headerWriteSize)
	if extra {
		binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)+2))
	} else {
		binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(data)))
	}
	copy(buffer[headerWriteSize:headerWriteSize+len(bufferType)], bufferType)
	copy(buffer[headerWriteSize+len(bufferType):], data)

	// Close endpoint
	_, err := d.transfer(cmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to close endpoint")
	}

	// Open endpoint
	_, err = d.transfer(cmdOpenEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to open endpoint")
	}

	// Send it
	_, err = d.transfer(cmdWrite, buffer, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to endpoint")
	}

	// Close endpoint
	_, err = d.transfer(cmdCloseEndpoint, endpoint, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to close endpoint")
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint, buffer, bufferType []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	mutex.Lock()
	defer mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = 0x08
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
		return nil, err
	}

	// Get data from a device
	if _, err := d.dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		return nil, err
	}

	// Read remaining data from a device
	if len(bufferType) == 2 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(transferTimeout)*time.Millisecond)
		defer cancel()

		for ctx.Err() != nil && !responseMatch(bufferR, bufferType) {
			if _, err := d.dev.Read(bufferR); err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
				return nil, err
			}
		}
		if ctx.Err() != nil {
			logger.Log(logger.Fields{"error": ctx.Err(), "serial": d.Serial}).Error("Unable to read data from device")
			return nil, ctx.Err()
		}
	}

	return bufferR, nil
}

// responseMatch will check if two byte arrays match
func responseMatch(response, expected []byte) bool {
	responseBuffer := response[4:6]
	return bytes.Equal(responseBuffer, expected)
}
