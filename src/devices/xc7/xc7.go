package xc7

// Package: Corsair XC7
// This is the primary package for Corsair XC7 CPU Block.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices/lcd"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/systeminfo"
	"OpenLinkHub/src/temperatures"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/godbus/dbus/v5"
	"github.com/sstallion/go-hid"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// DeviceProfile struct contains all device profile
type DeviceProfile struct {
	Active      bool
	Path        string
	Product     string
	Serial      string
	LCDMode     uint8
	LCDRotation uint8
	Brightness  uint8
	RGBProfile  string
	Label       string
}

type TemperatureProbe struct {
	ChannelId int
	Name      string
	Label     string
	Serial    string
	Product   string
}

type Device struct {
	Debug             bool
	dev               *hid.Device
	Manufacturer      string                    `json:"manufacturer"`
	Product           string                    `json:"product"`
	Serial            string                    `json:"serial"`
	Firmware          string                    `json:"firmware"`
	AIO               bool                      `json:"aio"`
	UserProfiles      map[string]*DeviceProfile `json:"userProfiles"`
	Devices           map[int]string            `json:"devices"`
	DeviceProfile     *DeviceProfile
	OriginalProfile   *DeviceProfile
	TemperatureProbes *[]TemperatureProbe
	activeRgb         *rgb.ActiveRGB
	Template          string
	HasLCD            bool
	VendorId          uint16
	ProductId         uint16
	LCDModes          map[int]string
	LCDRotations      map[int]string
	Brightness        map[int]string
	GlobalBrightness  float64
	FirmwareInternal  []int
	Temperature       float32
	TemperatureString string `json:"temperatureString"`
	LEDChannels       int
	CpuTemp           float32
	GpuTemp           float32
	Rgb               *rgb.RGB
}

var (
	pwd                        = ""
	lcdRefreshInterval         = 1000
	mutex                      sync.Mutex
	authRefreshChan            = make(chan bool)
	lcdRefreshChan             = make(chan bool)
	deviceRefreshInterval      = 1000
	timer                      = &time.Ticker{}
	lcdTimer                   = &time.Ticker{}
	lcdHeaderSize              = 8
	lcdBufferSize              = 1024
	temperatureReportId        = byte(24)
	firmwareReportId           = byte(5)
	featureReportSize          = 32
	maxLCDBufferSizePerRequest = lcdBufferSize - lcdHeaderSize
	deviceWakeupDelay          = 5000
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
		dev:       dev,
		Template:  "xc7.html",
		VendorId:  vendorId,
		ProductId: productId,
		LCDModes: map[int]string{
			0: "Liquid Temperature",
			2: "CPU Temperature",
			3: "GPU Temperature",
			5: "Combined",
			6: "CPU / GPU Temp",
			7: "CPU / GPU Load",
			8: "CPU / GPU Load/Temp",
			9: "Time",
		},
		LCDRotations: map[int]string{
			0: "default",
			1: "90 degrees",
			2: "180 degrees",
			3: "270 degrees",
		},
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
	}

	if productId == 3138 {
		// CORSAIR XC7 ELITE LCD CPU Water Block
		d.HasLCD = true
		d.LEDChannels = 31
	}

	// Bootstrap
	d.getManufacturer()     // Manufacturer
	d.getProduct()          // Product
	d.getSerial()           // Serial
	d.loadRgb()             // Load RGB
	d.getDeviceFirmware()   // Firmware
	d.loadDeviceProfiles()  // Load all device profiles
	d.setAutoRefresh()      // Set auto device refresh
	d.saveDeviceProfile()   // Save profile
	d.getTemperatureProbe() // Devices with temperature probes
	d.setDeviceColor()      // Device color
	d.setupLCD()            // LCD
	if config.GetConfig().DbusMonitor {
		d.dbusDeviceMonitor()
	}
	logger.Log(logger.Fields{"device": d}).Info("Device successfully initialized")
	return d
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	logger.Log(logger.Fields{"serial": d.Serial}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}
	authRefreshChan <- true

	if d.dev != nil {
		lcdRefreshChan <- true
		lcdTimer.Stop()

		// Switch LCD back to hardware mode
		lcdReports := map[int][]byte{0: {0x03, 0x1e, 0x01, 0x01}, 1: {0x03, 0x1d, 0x00, 0x01}}
		for i := 0; i <= 1; i++ {
			_, e := d.dev.SendFeatureReport(lcdReports[i])
			if e != nil {
				logger.Log(logger.Fields{"error": e}).Error("Unable to send report to LCD HID device")
			}
		}

		// Close it
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close LCD HID device")
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
	product = strings.Replace(product, "CORSAIR ", "", -1)
	product = strings.Replace(product, " CPU Water Block", "", -1)
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

	deviceProfile := &DeviceProfile{
		Product: d.Product,
		Serial:  d.Serial,
		Path:    profilePath,
	}

	// First save, assign saved profile to a device
	if d.DeviceProfile == nil {
		// RGB, Label
		deviceProfile.RGBProfile = "static"
		deviceProfile.Label = "CPU Block"

		// LCD
		if d.HasLCD {
			deviceProfile.LCDMode = 0
			deviceProfile.LCDRotation = 0
		}

		deviceProfile.Active = true
		d.DeviceProfile = deviceProfile
	} else {
		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.Brightness = d.DeviceProfile.Brightness
		deviceProfile.RGBProfile = d.DeviceProfile.RGBProfile
		deviceProfile.Label = d.DeviceProfile.Label
		if len(d.DeviceProfile.Path) < 1 {
			deviceProfile.Path = profilePath
			d.DeviceProfile.Path = profilePath
		} else {
			deviceProfile.Path = d.DeviceProfile.Path
		}
		deviceProfile.LCDMode = d.DeviceProfile.LCDMode
		deviceProfile.LCDRotation = d.DeviceProfile.LCDRotation
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
		logger.Log(logger.Fields{"error": err, "location": deviceProfile.Path}).Fatal("Unable to close file handle")
	}

	d.loadDeviceProfiles() // Reload
}

// getTemperatureProbe will request a feature report for temperature probe
func (d *Device) getTemperatureProbeData() {
	buf := make([]byte, featureReportSize+1)
	buf[0] = temperatureReportId
	n, err := d.dev.GetFeatureReport(buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to get temperature probe feature report")
		return
	}
	buffer := buf[:n]
	temp := float32(int16(binary.LittleEndian.Uint16(buffer[2:4]))) / 10.0

	d.Temperature = temp
	d.TemperatureString = dashboard.GetDashboard().TemperatureToString(temp)
}

// getDeviceFirmware will get device firmware
func (d *Device) getDeviceFirmware() {
	buf := make([]byte, featureReportSize+1)
	buf[0] = firmwareReportId
	n, err := d.dev.GetFeatureReport(buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to firmware details")
		return
	}
	buffer := buf[:n]

	v1, v2, v3, v4 := string(buffer[6]), string(buffer[8]), string(buffer[10]), string(buffer[12:14])
	d.Firmware = fmt.Sprintf("%s.%s.%s.%s", v1, v2, v3, v4)
}

// setCpuTemperature will store current CPU temperature
func (d *Device) setTemperatures() {
	d.CpuTemp = temperatures.GetCpuTemperature()
	d.GpuTemp = temperatures.GetGpuTemperature()
}

// getLiquidTemperature will fetch temperature from AIO device
func (d *Device) getLiquidTemperature() float32 {
	return d.Temperature
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
				d.getTemperatureProbeData()
			case <-authRefreshChan:
				timer.Stop()
				return
			}
		}
	}()
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
	d.transferToColor(buffer)

	if d.DeviceProfile.RGBProfile == "static" {
		profile := d.GetRgbProfile("static")
		if d.DeviceProfile.Brightness != 0 {
			profile.StartColor.Brightness = rgb.GetBrightnessValue(d.DeviceProfile.Brightness)
		}

		profileColor := rgb.ModifyBrightness(profile.StartColor)
		for i := 0; i < d.LEDChannels; i++ {
			reset[i] = []byte{
				byte(profileColor.Red),
				byte(profileColor.Green),
				byte(profileColor.Blue),
			}
		}
		buffer = rgb.SetColor(reset)
		d.transferToColor(buffer) // Write color once
		return
	}

	go func(lightChannels int) {
		lock := sync.Mutex{}
		startTime := time.Now()
		reverse := false
		counterColorpulse := 0
		counterFlickering := 0
		counterColorshift := 0
		counterCircleshift := 0
		counterCircle := 0
		counterColorwarp := 0
		counterSpinner := 0
		counterCpuTemp := 0
		counterGpuTemp := 0
		counterLiquidTemp := 0
		var temperatureKeys *rgb.Color
		colorwarpGeneratedReverse := false
		d.activeRgb = rgb.Exit()

		// Generate random colors
		d.activeRgb.RGBStartColor = rgb.GenerateRandomColor(1)
		d.activeRgb.RGBEndColor = rgb.GenerateRandomColor(1)

		hue := 1
		wavePosition := 0.0
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
					logger.Log(logger.Fields{"profile": d.DeviceProfile.RGBProfile, "serial": d.Serial}).Warn("No such RGB profile found")
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
				if d.DeviceProfile.Brightness > 0 {
					r.RGBBrightness = rgb.GetBrightnessValue(d.DeviceProfile.Brightness)
					r.RGBStartColor.Brightness = r.RGBBrightness
					r.RGBEndColor.Brightness = r.RGBBrightness
				}

				switch d.DeviceProfile.RGBProfile {
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
				case "watercolor":
					{
						r.Watercolor(startTime)
						buff = append(buff, r.Output...)
					}
				case "liquid-temperature":
					{
						lock.Lock()
						counterLiquidTemp++
						if counterLiquidTemp >= r.Smoothness {
							counterLiquidTemp = 0
						}

						if temperatureKeys == nil {
							temperatureKeys = r.RGBStartColor
						}

						r.MinTemp = profile.MinTemp
						r.MaxTemp = profile.MaxTemp
						res := r.Temperature(float64(d.getLiquidTemperature()), counterLiquidTemp, temperatureKeys)
						temperatureKeys = res
						lock.Unlock()
						buff = append(buff, r.Output...)
					}
				case "cpu-temperature":
					{
						lock.Lock()
						counterCpuTemp++
						if counterCpuTemp >= r.Smoothness {
							counterCpuTemp = 0
						}

						if temperatureKeys == nil {
							temperatureKeys = r.RGBStartColor
						}

						r.MinTemp = profile.MinTemp
						r.MaxTemp = profile.MaxTemp
						res := r.Temperature(float64(d.CpuTemp), counterCpuTemp, temperatureKeys)
						temperatureKeys = res
						lock.Unlock()
						buff = append(buff, r.Output...)
					}
				case "gpu-temperature":
					{
						lock.Lock()
						counterGpuTemp++
						if counterGpuTemp >= r.Smoothness {
							counterGpuTemp = 0
						}

						if temperatureKeys == nil {
							temperatureKeys = r.RGBStartColor
						}

						r.MinTemp = profile.MinTemp
						r.MaxTemp = profile.MaxTemp
						res := r.Temperature(float64(d.GpuTemp), counterGpuTemp, temperatureKeys)
						temperatureKeys = res
						lock.Unlock()
						buff = append(buff, r.Output...)
					}
				case "colorpulse":
					{
						lock.Lock()
						counterColorpulse++
						if counterColorpulse >= r.Smoothness {
							counterColorpulse = 0
						}

						r.Colorpulse(counterColorpulse)
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
						if counterFlickering >= r.Smoothness {
							counterFlickering = 0
						} else {
							counterFlickering++
						}

						r.Flickering(counterFlickering)
						lock.Unlock()
						buff = append(buff, r.Output...)
					}
				case "colorshift":
					{
						lock.Lock()
						if counterColorshift >= r.Smoothness && !reverse {
							counterColorshift = 0
							reverse = true
						} else if counterColorshift >= r.Smoothness && reverse {
							counterColorshift = 0
							reverse = false
						}

						r.Colorshift(counterColorshift, reverse)
						counterColorshift++
						lock.Unlock()
						buff = append(buff, r.Output...)
					}
				case "circleshift":
					{
						lock.Lock()
						if counterCircleshift >= lightChannels {
							counterCircleshift = 0
						} else {
							counterCircleshift++
						}

						r.Circle(counterCircleshift)
						lock.Unlock()
						buff = append(buff, r.Output...)
					}
				case "circle":
					{
						lock.Lock()
						if counterCircle >= lightChannels {
							counterCircle = 0
						} else {
							counterCircle++
						}

						r.Circle(counterCircle)
						lock.Unlock()
						buff = append(buff, r.Output...)
					}
				case "spinner":
					{
						lock.Lock()
						if counterSpinner >= lightChannels {
							counterSpinner = 0
						} else {
							counterSpinner++
						}
						r.Spinner(counterSpinner)
						lock.Unlock()
						buff = append(buff, r.Output...)
					}
				case "colorwarp":
					{
						lock.Lock()
						if counterColorwarp >= r.Smoothness {
							if !colorwarpGeneratedReverse {
								colorwarpGeneratedReverse = true
								d.activeRgb.RGBStartColor = d.activeRgb.RGBEndColor
								d.activeRgb.RGBEndColor = rgb.GenerateRandomColor(r.RGBBrightness)
							}
							counterColorwarp = 0
						} else if counterColorwarp == 0 && colorwarpGeneratedReverse == true {
							colorwarpGeneratedReverse = false
						} else {
							counterColorwarp++
						}

						r.Colorwarp(counterColorwarp, d.activeRgb.RGBStartColor, d.activeRgb.RGBEndColor)
						lock.Unlock()
						buff = append(buff, r.Output...)
					}
				}
				// Send it
				d.transferToColor(buff)
				time.Sleep(20 * time.Millisecond)
				hue++
				wavePosition += 0.2
			}
		}
	}(d.LEDChannels)
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

		newProfile := profile
		newProfile.Active = true
		d.DeviceProfile = newProfile
		d.saveDeviceProfile()
		d.setDeviceColor()
		return 1
	}
	return 0
}

// UpdateDeviceLcd will update device LCD
func (d *Device) UpdateDeviceLcd(_, mode uint8) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	if d.HasLCD {
		d.DeviceProfile.LCDMode = mode
		d.saveDeviceProfile()
		return 1
	} else {
		return 2
	}
}

// UpdateDeviceLcdRotation will update device LCD rotation
func (d *Device) UpdateDeviceLcdRotation(_, rotation uint8) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	if d.HasLCD {
		d.DeviceProfile.LCDRotation = rotation
		d.saveDeviceProfile()
		return 1
	} else {
		return 2
	}
}

// UpdateDeviceLabel will set / update device label
func (d *Device) UpdateDeviceLabel(_, label string) uint8 {
	mutex.Lock()
	defer mutex.Unlock()

	d.DeviceProfile.Label = label
	d.saveDeviceProfile()
	return 1
}

// UpdateRgbProfile will update device RGB profile
func (d *Device) UpdateRgbProfile(_ int, profile string) uint8 {
	if d.GetRgbProfile(profile) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}

	d.DeviceProfile.RGBProfile = profile // Set profile
	d.saveDeviceProfile()                // Save profile
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
	return 1
}

// getTemperatureProbe will return all devices with a temperature probe
func (d *Device) getTemperatureProbe() {
	var probes []TemperatureProbe
	probe := TemperatureProbe{
		ChannelId: 0,
		Name:      d.Product,
		Label:     d.DeviceProfile.Label,
		Serial:    d.Serial,
		Product:   d.Product,
	}
	probes = append(probes, probe)
	d.TemperatureProbes = &probes
}

// getLCDRotation will return rotation value based on rotation mode
func (d *Device) getLCDRotation() int {
	switch d.DeviceProfile.LCDRotation {
	case 0:
		return 0
	case 1:
		return 90
	case 2:
		return 180
	case 3:
		return 270
	}
	return 0
}

// setupLCD will activate and configure LCD
func (d *Device) setupLCD() {
	lcdTimer = time.NewTicker(time.Duration(lcdRefreshInterval) * time.Millisecond)
	lcdRefreshChan = make(chan bool)
	go func() {
		for {
			select {
			case <-lcdTimer.C:
				switch d.DeviceProfile.LCDMode {
				case lcd.DisplayCPU:
					{
						cpuTemp := int(temperatures.GetCpuTemperature())
						buffer := lcd.GenerateScreenImage(
							lcd.DisplayCPU,
							cpuTemp,
							0,
							0,
							0,
							d.getLCDRotation(),
						)
						d.transferToLcd(buffer)
					}
				case lcd.DisplayGPU:
					{
						gpuTemp := int(temperatures.GetGpuTemperature())
						buffer := lcd.GenerateScreenImage(
							lcd.DisplayGPU,
							gpuTemp,
							0,
							0,
							0,
							d.getLCDRotation(),
						)
						d.transferToLcd(buffer)
					}
				case lcd.DisplayLiquid:
					{
						liquidTemp := int(d.Temperature)
						buffer := lcd.GenerateScreenImage(
							lcd.DisplayLiquid,
							liquidTemp,
							0,
							0,
							0,
							d.getLCDRotation(),
						)
						d.transferToLcd(buffer)
					}
				case lcd.DisplayLiquidCPU:
					{
						cpuTemp := int(temperatures.GetCpuTemperature())
						liquidTemp := int(d.Temperature)
						buffer := lcd.GenerateScreenImage(
							lcd.DisplayLiquidCPU,
							liquidTemp,
							cpuTemp,
							0,
							0,
							d.getLCDRotation(),
						)
						d.transferToLcd(buffer)
					}
				case lcd.DisplayCpuGpuTemp:
					{
						cpuTemp := int(temperatures.GetCpuTemperature())
						gpuTemp := int(temperatures.GetGpuTemperature())
						buffer := lcd.GenerateScreenImage(
							lcd.DisplayCpuGpuTemp,
							cpuTemp,
							gpuTemp,
							0,
							0,
							d.getLCDRotation(),
						)
						d.transferToLcd(buffer)
					}
				case lcd.DisplayCpuGpuLoad:
					{
						cpuUtil := int(systeminfo.GetCpuUtilization())
						gpuUtil := systeminfo.GetGPUUtilization()
						buffer := lcd.GenerateScreenImage(
							lcd.DisplayCpuGpuLoad,
							cpuUtil,
							gpuUtil,
							0,
							0,
							d.getLCDRotation(),
						)
						d.transferToLcd(buffer)
					}
				case lcd.DisplayCpuGpuLoadTemp:
					{
						cpuTemp := int(temperatures.GetCpuTemperature())
						gpuTemp := int(temperatures.GetGpuTemperature())
						cpuUtil := int(systeminfo.GetCpuUtilization())
						gpuUtil := systeminfo.GetGPUUtilization()
						buffer := lcd.GenerateScreenImage(
							lcd.DisplayCpuGpuLoadTemp,
							cpuTemp,
							gpuTemp,
							cpuUtil,
							gpuUtil,
							d.getLCDRotation(),
						)
						d.transferToLcd(buffer)
					}
				case lcd.DisplayTime:
					{
						buffer := lcd.GenerateScreenImage(
							lcd.DisplayTime,
							0,
							0,
							0,
							0,
							d.getLCDRotation(),
						)
						d.transferToLcd(buffer)
					}
				}
			case <-lcdRefreshChan:
				lcdTimer.Stop()
				return
			}
		}
	}()
}

// dbusDeviceMonitor will monitor dbus events for suspend and resume
func (d *Device) dbusDeviceMonitor() {
	go func() {
		// Connect to the session bus
		conn, err := dbus.ConnectSystemBus()
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Fatal("Failed to connect to system bus")
		}
		defer func(conn *dbus.Conn) {
			err = conn.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Fatal("Error closing dbus")
			}
		}(conn)

		// Listen for the PrepareForSleep signal
		_ = conn.Object("org.freedesktop.login1", "/org/freedesktop/login1")
		ch := make(chan *dbus.Signal, 10)
		conn.Signal(ch)

		match := "type='signal',interface='org.freedesktop.login1.Manager',member='PrepareForSleep'"
		err = conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, match).Store()
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Fatal("Failed to add D-Bus match")
		}

		for signal := range ch {
			if len(signal.Body) > 0 {
				if isSleeping, ok := signal.Body[0].(bool); ok {
					if isSleeping {
						//
					} else {
						// Wait for 5 seconds until the hub wakes up
						time.Sleep(time.Duration(deviceWakeupDelay) * time.Millisecond)

						// Device woke up after machine was sleeping
						if d.activeRgb != nil {
							d.activeRgb.Exit <- true
							d.activeRgb = nil
						}
						d.setDeviceColor() // Set RGB
					}
				}
			}
		}
	}()
}

// transferToLcd will transfer data to LCD panel
func (d *Device) transferToLcd(buffer []byte) {
	mutex.Lock()
	defer mutex.Unlock()
	chunks := common.ProcessMultiChunkPacket(buffer, maxLCDBufferSizePerRequest)
	for i, chunk := range chunks {
		bufferW := make([]byte, lcdBufferSize)
		bufferW[0] = 0x02
		bufferW[1] = 0x05 // LCD data

		// The last packet needs to end with 0x01 in order for display to render data
		if len(chunk) < maxLCDBufferSizePerRequest {
			bufferW[3] = 0x01
		}

		bufferW[4] = byte(i)
		binary.LittleEndian.PutUint16(bufferW[6:8], uint16(len(chunk)))
		copy(bufferW[8:], chunk)

		if _, err := d.dev.Write(bufferW); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
			break
		}
	}
}

// transferToColor will transfer color data to a device
func (d *Device) transferToColor(buffer []byte) {
	mutex.Lock()
	defer mutex.Unlock()

	bufferW := make([]byte, lcdBufferSize)
	bufferW[0] = 0x02
	bufferW[1] = 0x07                // RGB Data
	bufferW[2] = byte(d.LEDChannels) // Number of LEDs on the block
	copy(bufferW[3:], buffer)        // Color buffer
	if _, err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return
	}
}
