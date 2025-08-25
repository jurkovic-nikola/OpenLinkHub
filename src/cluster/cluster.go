package cluster

// Package: cluster
// This is the primary package for RGB cluster.
// Cluster RGB is controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/temperatures"
	"encoding/json"
	"math/rand"
	"os"
	"sync"
	"time"
)

var (
	pwd                   = ""
	d                     *Device
	deviceRefreshInterval = 1000
)

type DeviceProfile struct {
	RGBProfile string
}

type Device struct {
	Product         string `json:"product"`
	Serial          string `json:"serial"`
	DeviceProfile   *DeviceProfile
	Rgb             *rgb.RGB
	activeRgb       *rgb.ActiveRGB
	Controllers     []*common.ClusterController
	mutex           sync.RWMutex
	Exit            bool
	RGBModes        []string
	CpuTemp         float32
	GpuTemp         float32
	timer           *time.Ticker
	autoRefreshChan chan struct{}
}

func Init() *Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath
	d = &Device{
		Product: "Cluster",
		Serial:  "cluster",
		RGBModes: []string{
			"circle",
			"circleshift",
			"colorpulse",
			"colorshift",
			"colorwarp",
			"cpu-temperature",
			"flickering",
			"gpu-temperature",
			"marquee",
			"nebula",
			"rainbow",
			"rotator",
			"sequential",
			"spinner",
			"static",
			"storm",
			"visor",
			"watercolor",
			"wave",
		},
		autoRefreshChan: make(chan struct{}),
		timer:           &time.Ticker{},
	}
	d.loadRgb()
	d.loadDeviceProfile()
	d.saveDeviceProfile()
	d.setAutoRefresh()
	return d
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}

	d.timer.Stop()
	close(d.autoRefreshChan)
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
}

func Get() *Device {
	return d
}

// AddDeviceController will add a new Cluster Controller
func (d *Device) AddDeviceController(controller *common.ClusterController) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.Controllers = append(d.Controllers, controller)

	if len(d.Controllers) == 1 {
		d.setDeviceColor()
	}
}

// RemoveDeviceControllerBySerial removes a controller by its serial
func (d *Device) RemoveDeviceControllerBySerial(serial string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for i, c := range d.Controllers {
		if c.Serial == serial {
			d.Controllers = append(d.Controllers[:i], d.Controllers[i+1:]...)
			if len(d.Controllers) == 0 {
				if d.activeRgb != nil {
					d.activeRgb.Exit <- true
					d.activeRgb = nil
				}
			}
			return
		}
	}
}

// GetRgbProfiles will return RGB profiles for a target device
func (d *Device) GetRgbProfiles() interface{} {
	return d.Rgb
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

// UpdateRgbProfileData will update RGB profile data
func (d *Device) UpdateRgbProfileData(profileName string, profile rgb.Profile) uint8 {
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
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
	return 1
}

// UpdateRgbProfile will update device RGB profile
func (d *Device) UpdateRgbProfile(_ int, profile string) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	pf := d.GetRgbProfile(profile)
	if pf == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}
	d.DeviceProfile.RGBProfile = profile
	d.saveDeviceProfile()

	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
	return 1
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

// distributeColors splits the generated buffer across all controllers
func (d *Device) distributeColors(buff []byte) {
	d.mutex.RLock()
	controllers := make([]*common.ClusterController, len(d.Controllers))
	copy(controllers, d.Controllers) // copy slice to avoid race
	d.mutex.RUnlock()

	var wg sync.WaitGroup
	offset := 0

	for _, c := range controllers {
		length := int(c.LedChannels) * 3

		if offset+length > len(buff) {
			break
		}

		slice := buff[offset : offset+length]

		if c.WriteColorEx != nil {
			wg.Add(1)
			go func(ctrl *common.ClusterController, data []byte) {
				defer wg.Done()
				ctrl.WriteColorEx(data, ctrl.ChannelId)
			}(c, slice)
		}

		offset += length
	}

	wg.Wait() // wait for all goroutines to finish
}

// setDeviceColor will set cluster rgb effect
func (d *Device) setDeviceColor() {
	if d.DeviceProfile == nil {
		return
	}

	profile := d.GetRgbProfile(d.DeviceProfile.RGBProfile)
	if profile == nil {
		return
	}

	go func() {
		startTime := time.Now()
		d.activeRgb = rgb.Exit()
		d.activeRgb.RGBStartColor = rgb.GenerateRandomColor(1)
		d.activeRgb.RGBEndColor = rgb.GenerateRandomColor(1)
		rand.New(rand.NewSource(time.Now().UnixNano()))

		for {
			lightChannels := 0
			for k := range d.Controllers {
				lightChannels += int(d.Controllers[k].LedChannels)
			}

			select {
			case <-d.activeRgb.Exit:
				return
			default:
				if d.Exit {
					return
				}
				buff := d.generateRgbEffect(lightChannels, &startTime, d.DeviceProfile.RGBProfile)
				d.distributeColors(buff)
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()
}

// generateRgbEffect will generate RGB effect for given device index
func (d *Device) generateRgbEffect(channels int, startTime *time.Time, rgbProfile string) []byte {
	buff := make([]byte, 0)
	rgbCustomColor := true

	profile := d.GetRgbProfile(rgbProfile)
	if profile == nil {
		for i := 0; i < channels; i++ {
			buff = []byte{0, 0, 0}
		}
		return buff
	}
	rgbModeSpeed := common.FClamp(profile.Speed, 0.1, 10)
	if (rgb.Color{}) == profile.StartColor || (rgb.Color{}) == profile.EndColor {
		rgbCustomColor = false
	}

	r := rgb.New(
		channels,
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
	r.RGBBrightness = 1 //rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
	r.RGBStartColor.Brightness = r.RGBBrightness
	r.RGBEndColor.Brightness = r.RGBBrightness

	r.MinTemp = profile.MinTemp
	r.MaxTemp = profile.MaxTemp

	switch rgbProfile {
	case "off":
		{
			for n := 0; n < channels; n++ {
				buff = append(buff, []byte{0, 0, 0}...)
			}
		}
	case "rainbow":
		{
			r.Rainbow(*startTime)
			buff = r.Output
		}
	case "watercolor":
		{
			r.Watercolor(*startTime)
			buff = r.Output
		}
	case "cpu-temperature":
		{
			r.Temperature(float64(d.CpuTemp))
			buff = r.Output
		}
	case "gpu-temperature":
		{
			r.Temperature(float64(d.GpuTemp))
			buff = r.Output
		}
	case "colorpulse":
		{
			r.Colorpulse(startTime)
			buff = r.Output
		}
	case "static":
		{
			r.Static()
			buff = r.Output
		}
	case "rotator":
		{
			r.Rotator(startTime)
			buff = r.Output
		}
	case "wave":
		{
			r.Wave(startTime)
			buff = r.Output
		}
	case "storm":
		{
			r.Storm()
			buff = r.Output
		}
	case "flickering":
		{
			r.Flickering(startTime)
			buff = r.Output
		}
	case "colorshift":
		{
			r.Colorshift(startTime, d.activeRgb)
			buff = r.Output
		}
	case "circleshift":
		{
			r.CircleShift(startTime)
			buff = r.Output
		}
	case "circle":
		{
			r.Circle(startTime)
			buff = r.Output
		}
	case "spinner":
		{
			r.Spinner(startTime)
			buff = r.Output
		}
	case "colorwarp":
		{
			r.Colorwarp(startTime, d.activeRgb)
			buff = r.Output
		}
	case "nebula":
		{
			r.Nebula(startTime)
			buff = r.Output
		}
	case "visor":
		{
			r.Visor(startTime)
			buff = r.Output
		}
	case "marquee":
		{
			r.Marquee(startTime)
			buff = r.Output
		}
	case "sequential":
		{
			r.Sequential(startTime)
			buff = r.Output
		}
	}
	return buff
}

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"
	deviceProfile := &DeviceProfile{}

	if d.DeviceProfile == nil {
		deviceProfile.RGBProfile = "rainbow"
		d.DeviceProfile = deviceProfile
	} else {
		deviceProfile.RGBProfile = d.DeviceProfile.RGBProfile
	}

	buffer, err := json.MarshalIndent(deviceProfile, "", "    ")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
		return
	}

	file, fileErr := os.Create(profilePath)
	if fileErr != nil {
		logger.Log(logger.Fields{"error": fileErr, "location": profilePath}).Error("Unable to create new device profile")
		return
	}

	_, err = file.Write(buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": profilePath}).Error("Unable to write data")
		return
	}

	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": profilePath}).Error("Unable to close file handle")
	}
	d.loadDeviceProfile()
}

// loadDeviceProfiles will load custom user profiles
func (d *Device) loadDeviceProfile() {
	profileLocation := pwd + "/database/profiles/" + d.Serial + ".json"

	pf := &DeviceProfile{}

	if !common.IsValidExtension(profileLocation, ".json") {
		return
	}

	file, err := os.Open(profileLocation)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": profileLocation}).Warn("Unable to load profile")
		return
	}

	if err = json.NewDecoder(file).Decode(pf); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": profileLocation}).Warn("Unable to decode profile")
		return
	}

	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"location": profileLocation, "serial": d.Serial}).Warn("Failed to close file handle")
	}

	d.DeviceProfile = pf
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
			case <-d.autoRefreshChan:
				d.timer.Stop()
				return
			}
		}
	}()
}
