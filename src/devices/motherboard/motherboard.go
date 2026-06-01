package motherboard

// Package: motherboard
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/motherboards"
	"OpenLinkHub/src/stats"
	"OpenLinkHub/src/temperatures"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	pwd                        = ""
	defaultSpeedValue          = 70
	temperaturePullingInterval = 3000
	i2cPrefix                  = "i2c"
)

type DeviceProfile struct {
	Active        bool
	Path          string
	Product       string
	Serial        string
	SpeedProfiles map[int]string
	HeaderModes   map[int]int
	Labels        map[int]string
	MultiProfile  string
	RgbOff        bool
}

type TemperatureProbe struct {
	ChannelId int
	Name      string
	Label     string
	Serial    string
	Product   string
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
	TemperatureString  string          `json:"temperatureString"`
	LedChannels        uint8           `json:"-"`
	ContainsPump       bool            `json:"-"`
	Description        string          `json:"description"`
	HubId              string          `json:"-"`
	PumpModes          map[byte]string `json:"-"`
	Profile            string          `json:"profile"`
	RGB                string          `json:"rgb"`
	Label              string          `json:"label"`
	ProbeId            int             `json:"probeId"`
	MinTemp            float64         `json:"minTemp"`
	MaxTemp            float64         `json:"maxTemp"`
	HeaderMode         int             `json:"headerMode"`
	OperatingModes     map[int]string  `json:"operatingModes"`
	HasSpeed           bool
	HasTemps           bool
	IsTemperatureProbe bool
	ExternalLed        bool
	CellSize           uint8
}

type Device struct {
	Debug             bool
	dev               *motherboards.Motherboard
	Manufacturer      string                    `json:"manufacturer"`
	Product           string                    `json:"product"`
	Serial            string                    `json:"serial"`
	Path              string                    `json:"path"`
	Firmware          string                    `json:"firmware"`
	RGB               string                    `json:"rgb"`
	AIO               bool                      `json:"aio"`
	Devices           map[int]*Devices          `json:"devices"`
	UserProfiles      map[string]*DeviceProfile `json:"userProfiles"`
	DeviceProfile     *DeviceProfile
	TemperatureProbes *[]TemperatureProbe
	RGBDeviceOnly     bool
	Template          string
	HasLCD            bool
	CpuTemp           float32
	GpuTemp           float32
	mutex             sync.Mutex
	autoRefreshChan   chan struct{}
	speedRefreshChan  chan struct{}
	timer             *time.Ticker
	timerSpeed        *time.Ticker
	Exit              bool
	deviceLock        sync.Mutex
	instance          *common.Device
}

// Init will initialize a new device
func Init() *common.Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	dev := motherboards.GetMotherboard()
	if dev == nil {
		logger.Log(logger.Fields{}).Error("Unable to open motherboard device")
		return nil
	}

	if len(motherboards.GetMotherboardSerial()) < 1 {
		logger.Log(logger.Fields{}).Error("Unable to open motherboard device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:              dev,
		Template:         "motherboard.html",
		autoRefreshChan:  make(chan struct{}),
		speedRefreshChan: make(chan struct{}),
		timer:            &time.Ticker{},
		timerSpeed:       &time.Ticker{},
	}

	// Bootstrap
	d.getDebugMode()       // Debug mode
	d.getProduct()         // Product
	d.getSerial()          // Serial
	d.loadDeviceProfiles() // Load all device profiles
	d.getDevices()         // Get devices connected to a hub
	d.setPwmMode()         // Set PWM mode
	d.setDefaults()        // Set default speed and color values for fans and pumps
	d.setAutoRefresh()     // Set auto device refresh
	d.saveDeviceProfile()  // Save profile
	if config.GetConfig().Manual {
		fmt.Println(
			fmt.Sprintf("[%s [%s]] Manual flag enabled. Process will not monitor temperature or adjust fan speed.", d.Serial, d.Product),
		)
	} else {
		d.updateDeviceSpeed()
	}
	d.createDevice() // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeMotherboard,
		Product:     d.Product,
		Serial:      d.Serial,
		Image:       "icon-motherboard.svg",
		Instance:    d,
		GetDevice:   d,
	}
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

	if config.GetConfig().MotherboardBiosOnExit {
		d.setBiosMode()
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
}

// StopDirty will stop device in a dirty way
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

	if config.GetConfig().MotherboardBiosOnExit {
		d.setBiosMode()
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 1
}

// GetDeviceTemplate will return device template name
func (d *Device) GetDeviceTemplate() string {
	return d.Template
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
		if !common.AlphanumericDashRegex.MatchString(fileName) {
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

// getProduct will return device name
func (d *Device) getProduct() {
	d.Product = d.dev.DisplayName
}

// getSerial will return device serial number
func (d *Device) getSerial() {
	d.Serial = motherboards.GetMotherboardSerial()
}

// getManufacturer will return device manufacturer
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
}

// getBiosOperatingMode will return BIOS operating mode for header
func (d *Device) getBiosOperatingMode(channelId int) int {
	if val, ok := d.dev.Headers[channelId]; ok {
		for k, v := range val.HeaderModes {
			if v == "BIOS" {
				return k
			}
		}
	}
	return 0
}

// getPwmOperatingMode will return PWM operating mode for header
func (d *Device) getPwmOperatingMode(channelId int) int {
	if val, ok := d.dev.Headers[channelId]; ok {
		for k, v := range val.HeaderModes {
			if v == "PWM" {
				return k
			}
		}
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

// setDefaults will set default mode for all devices
func (d *Device) getDeviceData() {
	if d.Exit {
		return
	}

	for key, value := range d.Devices {
		rpm := motherboards.GetMotherboardHeaderValue(value.ChannelId)
		d.Devices[key].Rpm = rpm
	}

	// Update stats
	for key, value := range d.Devices {
		temperatureString := ""
		rpmString := ""
		if value.Rpm > 0 || value.Temperature > 0 {
			if value.Temperature > 0 {
				temperatureString = dashboard.GetDashboard().TemperatureToString(value.Temperature)
			}
			if value.Rpm > 0 {
				rpmString = fmt.Sprintf("%v RPM", value.Rpm)
			}
			stats.UpdateDeviceStats(d.Serial, value.Name, temperatureString, rpmString, value.Label, key, value.Temperature)
		}
	}
}

// setBiosMode will set headers to BIOS mode
func (d *Device) setBiosMode() {
	for _, value := range d.Devices {
		if motherboards.GetMotherboardHeaderMode(value.ChannelId) == d.getBiosOperatingMode(value.ChannelId) {
			continue
		}
		motherboards.SetMotherboardHeaderMode(value.ChannelId, d.getBiosOperatingMode(value.ChannelId))
	}
}

// setPwmMode will set headers to PWM mode
func (d *Device) setPwmMode() {
	for _, value := range d.Devices {
		if motherboards.GetMotherboardHeaderMode(value.ChannelId) == value.HeaderMode {
			continue
		}
		motherboards.SetMotherboardHeaderMode(value.ChannelId, value.HeaderMode)
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

// setCpuTemperature will store current CPU temperature
func (d *Device) setTemperatures() {
	d.CpuTemp = temperatures.GetCpuTemperature()
	d.GpuTemp = temperatures.GetGpuTemperature()
}

// setAutoRefresh will refresh device data
func (d *Device) setAutoRefresh() {
	d.timer = time.NewTicker(time.Duration(d.dev.Interval) * time.Millisecond)
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

// UpdateOperatingMode will update device operating mode.
func (d *Device) UpdateOperatingMode(channelId, mode int) uint8 {
	if _, ok := d.Devices[channelId]; ok {
		d.Devices[channelId].HeaderMode = mode
		if motherboards.SetMotherboardHeaderMode(channelId, mode) > 0 {
			d.saveDeviceProfile()
			return 1
		}
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

	if motherboards.GetMotherboardHeaderMode(channelId) == d.getBiosOperatingMode(channelId) {
		return 0
	}

	// If the profile is liquid temperature, check for the presence of AIOs
	if profiles.Sensor == temperatures.SensorTypeLiquidTemperature {
		// This device does not have an option for AIO pump
		return 2
	}

	// Block PSU profile type
	if profiles.Sensor == temperatures.SensorTypePSU {
		return 6
	}

	if profiles.Sensor == temperatures.SensorTypeTemperatureProbe {
		if strings.HasPrefix(profiles.Device, i2cPrefix) {
			if temperatures.GetMemoryTemperature(profiles.ChannelId) == 0 {
				return 5
			}
		} else {
			if profiles.Device != d.Serial {
				return 3
			}

			if _, ok := d.Devices[profiles.ChannelId]; !ok {
				return 4
			}
		}
	}

	if channelId < 0 {
		d.DeviceProfile.MultiProfile = profile
		for _, device := range d.Devices {
			d.Devices[device.ChannelId].Profile = profile
		}
	} else {
		if _, ok := d.Devices[channelId]; ok {
			d.Devices[channelId].Profile = profile
		}
	}

	// Save to profile
	d.saveDeviceProfile()
	return 1
}

// UpdateSpeedProfileBulk will update device channel speed.
func (d *Device) UpdateSpeedProfileBulk(channelIds []int, profile string) uint8 {
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

	// Block PSU profile type
	if profiles.Sensor == temperatures.SensorTypePSU {
		return 6
	}

	if profiles.Sensor == temperatures.SensorTypeTemperatureProbe {
		if strings.HasPrefix(profiles.Device, i2cPrefix) {
			if temperatures.GetMemoryTemperature(profiles.ChannelId) == 0 {
				return 5
			}
		} else {
			if profiles.Device != d.Serial {
				return 3
			}

			if _, ok := d.Devices[profiles.ChannelId]; !ok {
				return 4
			}
		}
	}

	if len(channelIds) > 0 {
		d.DeviceProfile.MultiProfile = profile
		for _, channelId := range channelIds {
			if _, ok := d.Devices[channelId]; ok {
				// Update channel with new profile
				d.Devices[channelId].Profile = profile
			} else {
				return 0
			}
		}
	} else {
		return 0
	}

	d.saveDeviceProfile()
	return 1
}

// ResetSpeedProfiles will reset channel speed profile if it matches with the current speed profile
// This is used when speed profile is deleted from the UI
func (d *Device) ResetSpeedProfiles(profile string) {
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
		d.saveDeviceProfile()
	}
}

// getDevices will fetch all devices connected to a hub
func (d *Device) getDevices() int {
	var devices = make(map[int]*Devices)

	mobo := motherboards.GetMotherboard()
	if mobo != nil {
		for i := 1; i <= len(mobo.Headers); i++ {
			speedProfile := "Normal"
			label := "Set Label"
			headerMode := motherboards.GetMotherboardHeaderMode(i)
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

				// Device label
				if lb, ok := d.DeviceProfile.Labels[i]; ok {
					if len(lb) > 0 {
						label = lb
					}
				}

				if mode, ok := d.DeviceProfile.HeaderModes[i]; ok {
					headerMode = mode
				}
			} else {
				logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
			}

			device := &Devices{
				ChannelId:      i,
				DeviceId:       fmt.Sprintf("%s-%v", "Fan", i),
				Name:           mobo.Headers[i].HeaderName,
				Rpm:            0,
				Temperature:    0,
				Description:    "Fan",
				HubId:          d.Serial,
				Profile:        speedProfile,
				HasSpeed:       true,
				Label:          label,
				HeaderMode:     headerMode,
				OperatingModes: mobo.Headers[i].HeaderModes,
			}
			if label == "Pump" {
				device.ContainsPump = true
			}
			devices[i] = device
		}
	}
	d.Devices = devices
	return len(devices)
}

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"

	speedProfiles := make(map[int]string, len(d.Devices))
	headerModes := make(map[int]int, len(d.Devices))
	labels := make(map[int]string, len(d.Devices))

	for _, device := range d.Devices {
		speedProfiles[device.ChannelId] = device.Profile
		headerModes[device.ChannelId] = device.HeaderMode
	}

	for _, device := range d.Devices {
		labels[device.ChannelId] = device.Label
	}

	deviceProfile := &DeviceProfile{
		Product:       d.Product,
		Serial:        d.Serial,
		SpeedProfiles: speedProfiles,
		HeaderModes:   headerModes,
		Labels:        labels,
		Path:          profilePath,
	}

	if d.DeviceProfile == nil {
		for _, device := range d.Devices {
			labels[device.ChannelId] = "Set Label"
		}
		deviceProfile.Active = true
	} else {
		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.MultiProfile = d.DeviceProfile.MultiProfile
		if len(d.DeviceProfile.Path) < 1 {
			deviceProfile.Path = profilePath
			d.DeviceProfile.Path = profilePath
		} else {
			deviceProfile.Path = d.DeviceProfile.Path
		}
		deviceProfile.RgbOff = d.DeviceProfile.RgbOff
	}

	d.DeviceProfile = deviceProfile

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

// setSpeed will generate a speed buffer and send it to a device
func (d *Device) setSpeed(data map[int]byte) {
	if d.Exit {
		return
	}

	for key, value := range data {
		if motherboards.GetMotherboardHeaderMode(key) == d.getBiosOperatingMode(key) {
			// BIOS mode can not be updated from user-space
			continue
		}
		motherboards.SetMotherboardHeaderValue(key, int(value))
	}
}

// updateDeviceSpeed will update device speed based on a temperature reading
func (d *Device) updateDeviceSpeed() {
	d.timerSpeed = time.NewTicker(time.Duration(temperaturePullingInterval) * time.Millisecond)
	go func() {
		tmp := make(map[int]string)
		channelSpeeds := map[int]byte{}

		for _, device := range d.Devices {
			if device.IsTemperatureProbe {
				continue
			}
			channelSpeeds[device.ChannelId] = byte(defaultSpeedValue)
		}
		for {
			select {
			case <-d.timerSpeed.C:
				var temp float32 = 0
				for _, device := range d.Devices {
					if device.HasTemps {
						continue // Temperature probes
					}

					profiles := temperatures.GetTemperatureProfile(device.Profile)
					if profiles == nil {
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
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "hwmonDeviceId": profiles.Device}).Warn("Unable to get CPU temperature.")
							}
						}
					case temperatures.SensorTypeStorage:
						{
							temp = temperatures.GetStorageTemperature(profiles.Device)
							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "hwmonDeviceId": profiles.Device}).Warn("Unable to get storage temperature.")
							}
						}
					case temperatures.SensorTypeTemperatureProbe:
						{
							if strings.HasPrefix(profiles.Device, i2cPrefix) {
								temp = temperatures.GetMemoryTemperature(profiles.ChannelId)
							} else {
								if d.Devices[profiles.ChannelId].IsTemperatureProbe {
									temp = d.Devices[profiles.ChannelId].Temperature
								}
							}

							if temp == 0 {
								logger.Log(logger.Fields{"temperature": temp, "serial": d.Serial, "channelId": profiles.ChannelId}).Warn("Unable to get probe temperature.")
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
					case temperatures.SensorTypeMultiGPUs:
						{
							maxGpuTemp := float32(0)

							for _, index := range config.GetConfig().NvidiaGpuIndex {
								gpuTemp := temperatures.GetNVIDIAGpuTemperature(index)
								if gpuTemp == 0 {
									logger.Log(logger.Fields{
										"temperature": gpuTemp,
										"serial":      d.Serial,
										"channelId":   profiles.ChannelId,
										"gpuIndex":    index,
									}).Warn("Unable to get GPU temperature. Setting to 50")
									gpuTemp = 50
								}

								if gpuTemp > maxGpuTemp {
									maxGpuTemp = gpuTemp
								}
							}
							temp = maxGpuTemp
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
						if fans < 20 && !profiles.ZeroRpm {
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
							if device.ContainsPump {
								channelSpeeds[device.ChannelId] = byte(pump)
							} else {
								channelSpeeds[device.ChannelId] = byte(fans)
							}
							d.setSpeed(channelSpeeds)
						}

						if d.Debug {
							logger.Log(logger.Fields{"serial": d.Serial, "pump": pump, "fans": fans, "temp": temp, "device": device.Name, "zeroRpm": profiles.ZeroRpm}).Info("updateDeviceSpeed()")
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

									if profile.Fans < 20 && !profiles.ZeroRpm {
										profile.Fans = 20
									}

									if profile.Pump < 50 {
										profile.Pump = 50
									}

									if profile.Pump > 100 {
										profile.Pump = 100
									}

									if device.ContainsPump {
										channelSpeeds[device.ChannelId] = byte(profile.Pump)
									} else {
										channelSpeeds[device.ChannelId] = byte(profile.Fans)
									}
									d.setSpeed(channelSpeeds)
								}
							}
						}
					}
				}
			case <-d.speedRefreshChan:
				d.timerSpeed.Stop()
				return
			}
		}
	}()
}

// UpdateDeviceSpeed will update device channel speed.
func (d *Device) UpdateDeviceSpeed(channelId int, value uint16) uint8 {
	if device, ok := d.Devices[channelId]; ok {
		if device.IsTemperatureProbe {
			return 0
		}
		channelSpeeds := map[int]byte{}

		if value < 20 {
			value = 20
		}

		if device.ContainsPump {
			if value < 50 {
				value = 50
			}
		}
		channelSpeeds[device.ChannelId] = byte(value)
		d.setSpeed(channelSpeeds)
		return 1
	}
	return 0
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

// ChangeDeviceProfile will change device profile
func (d *Device) ChangeDeviceProfile(profileName string) uint8 {
	if profile, ok := d.UserProfiles[profileName]; ok {
		currentProfile := d.DeviceProfile
		currentProfile.Active = false
		d.DeviceProfile = currentProfile
		d.saveDeviceProfile()

		for _, device := range d.Devices {
			if device.HasSpeed {
				d.Devices[device.ChannelId].Profile = profile.SpeedProfiles[device.ChannelId]
			}
			d.Devices[device.ChannelId].Label = profile.Labels[device.ChannelId]
		}

		newProfile := profile
		newProfile.Active = true
		d.DeviceProfile = newProfile
		d.saveDeviceProfile()

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

// UpdateDeviceMetrics will update device metrics
func (d *Device) UpdateDeviceMetrics() {
	if d.Exit {
		return
	}
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
