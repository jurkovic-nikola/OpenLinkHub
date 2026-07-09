package nvidiagpu

// Package: nvidiagpu
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dispatcher"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"
	"time"
)

type nativeGPU struct {
	Handle          uintptr
	DeviceID        uint32
	SubSystemID     uint32
	RevisionID      uint32
	ExtDeviceID     uint32
	Name            string
	Zones           []int
	TreatsRGBWAsRGB bool
}

type Devices struct {
	ChannelId   int    `json:"channelId"`
	DeviceId    int    `json:"deviceId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	LedChannels uint8  `json:"ledChannels"`
	RGB         string `json:"rgb"`
	Label       string `json:"label"`
	ZoneType    int    `json:"zoneType"`
	HasSpeed    bool
	HasTemps    bool
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
	MultiRGB           string
	RgbOff             bool
}

type Device struct {
	Manufacturer     string                    `json:"manufacturer"`
	Product          string                    `json:"product"`
	Serial           string                    `json:"serial"`
	Path             string                    `json:"path"`
	Firmware         string                    `json:"firmware"`
	RGB              string                    `json:"rgb"`
	AIO              bool                      `json:"aio"`
	Devices          map[int]*Devices          `json:"devices"`
	UserProfiles     map[string]*DeviceProfile `json:"userProfiles"`
	DeviceProfile    *DeviceProfile
	RGBDeviceOnly    bool
	Template         string
	HasLCD           bool
	RGBModes         []string
	Rgb              *rgb.RGB
	GlobalBrightness float64
	Exit             bool
	activeRgb        *rgb.ActiveRGB
	instance         *common.Device
	native           nativeGPU
	mutex            sync.Mutex
	rgbMutex         sync.RWMutex
}

var (
	pwd      = ""
	rgbModes = []string{
		"circle",
		"circleshift",
		"colorpulse",
		"colorshift",
		"colorwarp",
		"gradient",
		"led",
		"marquee",
		"nebula",
		"off",
		"rainbow",
		"pastelrainbow",
		"pastelspiralrainbow",
		"rotator",
		"spinner",
		"spiralrainbow",
		"static",
		"storm",
		"watercolor",
		"wave",
	}
)

func InitAll() []*common.Device {
	pwd = config.GetConfig().ConfigPath

	gpus, err := detectNativeGPUs()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Warn("Unable to initialize NVIDIA GPU RGB")
		return nil
	}
	if len(gpus) == 0 {
		return nil
	}

	devices := make([]*common.Device, 0, len(gpus))
	for i, gpu := range gpus {
		device := newDevice(i, gpu)
		device.loadRgb()
		device.loadDeviceProfiles()
		device.ensureDeviceProfile()
		device.saveDeviceProfile()
		device.setDeviceColor()
		device.createDevice()
		logger.Log(logger.Fields{
			"serial":            device.Serial,
			"product":           device.Product,
			"deviceId":          fmt.Sprintf("0x%08x", gpu.DeviceID),
			"subSystemId":       fmt.Sprintf("0x%08x", gpu.SubSystemID),
			"illuminationZones": len(gpu.Zones),
		}).Info("NVIDIA GPU RGB device successfully initialized")
		devices = append(devices, device.instance)
	}

	return devices
}

func newDevice(index int, gpu nativeGPU) *Device {
	serial := fmt.Sprintf("nvidiagpu%d", index)
	devices := make(map[int]*Devices, len(gpu.Zones))
	for zone, zoneType := range gpu.Zones {
		devices[zone] = &Devices{
			ChannelId:   zone,
			DeviceId:    zone,
			Name:        fmt.Sprintf("GPU Zone %d", zone+1),
			Description: "NVIDIA illumination zone",
			LedChannels: 1,
			RGB:         "rainbow",
			Label:       fmt.Sprintf("Zone %d", zone+1),
			ZoneType:    zoneType,
		}
	}

	return &Device{
		Manufacturer:  "NVIDIA",
		Product:       gpu.Name,
		Serial:        serial,
		Path:          fmt.Sprintf("pci:%08x:%08x", gpu.DeviceID, gpu.SubSystemID),
		Firmware:      "0",
		Template:      "nvidiagpu.html",
		Devices:       devices,
		UserProfiles:  make(map[string]*DeviceProfile),
		RGBDeviceOnly: true,
		RGBModes:      rgbModes,
		native:        gpu,
	}
}

func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeNvidiaGPU,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-device.svg",
		Instance:    d,
		GetDevice:   d,
		DeviceType:  common.DeviceTypeGpu,
	}
}

func (d *Device) SetDispatcher(_ dispatcher.DeviceDispatcher) {
}

func (d *Device) GetDeviceTemplate() string {
	return d.Template
}

func (d *Device) GetRgbProfiles() interface{} {
	if d.Rgb == nil {
		return nil
	}

	tmp := *d.Rgb
	profiles := make(map[string]rgb.Profile, len(tmp.Profiles))
	for key, value := range tmp.Profiles {
		if slices.Contains(rgbModes, key) {
			profiles[key] = value
		}
	}
	tmp.Profiles = profiles
	return tmp
}

func (d *Device) GetRgbProfile(profile string) *rgb.Profile {
	if d.Rgb == nil {
		return nil
	}
	if val, ok := d.Rgb.Profiles[profile]; ok {
		return &val
	}
	return nil
}

func (d *Device) Stop() {
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")
	d.stopActiveRgb()
	d.writeAllZones([]byte{0, 0, 0})
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
}

func (d *Device) StopDirty() uint8 {
	d.Stop()
	return 1
}

func (d *Device) UpdateDeviceMetrics() {
}

func (d *Device) SchedulerBrightness(value uint8) uint8 {
	if d.DeviceProfile == nil || d.DeviceProfile.BrightnessSlider == nil {
		return 0
	}
	if value == 0 {
		d.DeviceProfile.OriginalBrightness = *d.DeviceProfile.BrightnessSlider
		d.DeviceProfile.BrightnessSlider = &value
	} else {
		d.DeviceProfile.BrightnessSlider = &d.DeviceProfile.OriginalBrightness
	}
	d.saveDeviceProfile()
	d.restartRgb()
	return 1
}

func (d *Device) ControlDeviceRgb(value bool) {
	if d.DeviceProfile == nil {
		return
	}
	d.DeviceProfile.RgbOff = value
	d.saveDeviceProfile()
	d.restartRgb()
}

func (d *Device) ChangeDeviceBrightness(mode uint8) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	d.DeviceProfile.Brightness = mode
	d.saveDeviceProfile()
	d.restartRgb()
	return 1
}

func (d *Device) ChangeDeviceBrightnessValue(value uint8) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	if d.GlobalBrightness != 0 {
		return 2
	}
	if value > 100 {
		return 0
	}
	d.DeviceProfile.BrightnessSlider = &value
	d.saveDeviceProfile()
	d.restartRgb()
	return 1
}

func (d *Device) UpdateRgbProfileData(profileName string, profile rgb.Profile) uint8 {
	d.rgbMutex.Lock()
	defer d.rgbMutex.Unlock()

	current := d.GetRgbProfile(profileName)
	if current == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profileName}).Warn("Non-existing RGB profile")
		return 0
	}
	if profile.StartColor.Temperature < 0 || profile.StartColor.Temperature > 105 {
		return 0
	}
	if profile.MiddleColor.Temperature < 0 || profile.MiddleColor.Temperature > 105 {
		return 0
	}
	if profile.EndColor.Temperature < 0 || profile.EndColor.Temperature > 105 {
		return 0
	}

	profile.StartColor.Brightness = current.StartColor.Brightness
	profile.EndColor.Brightness = current.EndColor.Brightness
	profile.MiddleColor.Brightness = current.MiddleColor.Brightness
	current.StartColor = profile.StartColor
	current.EndColor = profile.EndColor
	current.MiddleColor = profile.MiddleColor
	current.Speed = profile.Speed
	current.Gradients = profile.Gradients
	current.MinTemp = profile.MinTemp
	current.MaxTemp = profile.MaxTemp
	current.AlternateColors = profile.AlternateColors
	current.RgbDirection = profile.RgbDirection

	d.Rgb.Profiles[profileName] = *current
	d.saveRgbProfile()
	d.restartRgb()
	return 1
}

func (d *Device) UpdateRgbProfile(channelId int, profile string) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	if d.GetRgbProfile(profile) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}

	if channelId < 0 {
		d.DeviceProfile.MultiRGB = profile
		for _, device := range d.Devices {
			d.DeviceProfile.RGBProfiles[device.ChannelId] = profile
			device.RGB = profile
		}
	} else {
		device, ok := d.Devices[channelId]
		if !ok {
			return 0
		}
		d.DeviceProfile.RGBProfiles[channelId] = profile
		device.RGB = profile
	}

	d.saveDeviceProfile()
	d.restartRgb()
	return 1
}

func (d *Device) UpdateRgbProfileBulk(channelIds []int, profile string) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	if d.GetRgbProfile(profile) == nil {
		return 0
	}
	for _, channelId := range channelIds {
		device, ok := d.Devices[channelId]
		if !ok {
			return 0
		}
		d.DeviceProfile.RGBProfiles[channelId] = profile
		device.RGB = profile
	}
	d.saveDeviceProfile()
	d.restartRgb()
	return 1
}

func (d *Device) UpdateDeviceLabel(channelId int, label string) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}
	device, ok := d.Devices[channelId]
	if !ok {
		return 0
	}
	d.DeviceProfile.Labels[channelId] = label
	device.Label = label
	d.saveDeviceProfile()
	return 1
}

func (d *Device) UpdateRGBDeviceLabel(channelId int, label string) uint8 {
	return d.UpdateDeviceLabel(channelId, label)
}

func (d *Device) ChangeDeviceProfile(profileName string) uint8 {
	profile, ok := d.UserProfiles[profileName]
	if !ok {
		return 0
	}

	if d.DeviceProfile != nil {
		d.DeviceProfile.Active = false
		d.saveDeviceProfile()
	}

	for _, device := range d.Devices {
		device.RGB = profile.RGBProfiles[device.ChannelId]
		device.Label = profile.Labels[device.ChannelId]
	}

	profile.Active = true
	d.DeviceProfile = profile
	d.saveDeviceProfile()
	d.restartRgb()
	return 1
}

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

func (d *Device) SaveUserProfile(profileName string) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	profilePath := filepath.Join(pwd, "database", "profiles", d.Serial+"-"+profileName+".json")
	newProfile := *d.DeviceProfile
	newProfile.Path = profilePath
	newProfile.Active = false

	if err := common.SaveJsonData(profilePath, &newProfile); err != nil {
		logger.Log(logger.Fields{"error": err, "location": profilePath}).Error("Unable to create new device profile")
		return 0
	}

	d.loadDeviceProfiles()
	return 1
}

func (d *Device) loadRgb() {
	rgbFilename := filepath.Join(pwd, "database", "rgb", d.Serial+".json")
	if !common.IsValidExtension(rgbFilename, ".json") {
		return
	}

	if !common.FileExists(rgbFilename) {
		profile := rgb.GetRGB()
		profile.Device = d.Product
		if err := common.SaveJsonData(rgbFilename, profile); err != nil {
			logger.Log(logger.Fields{"error": err, "location": rgbFilename}).Error("Unable to write RGB profile data")
			return
		}
	}

	file, err := os.Open(rgbFilename)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": rgbFilename}).Warn("Unable to load RGB")
		return
	}
	defer file.Close()

	if err = json.NewDecoder(file).Decode(&d.Rgb); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": rgbFilename}).Warn("Unable to decode RGB profile")
	}
}

func (d *Device) saveRgbProfile() {
	if d.Rgb == nil {
		return
	}
	rgbFilename := filepath.Join(pwd, "database", "rgb", d.Serial+".json")
	if err := common.SaveJsonData(rgbFilename, d.Rgb); err != nil {
		logger.Log(logger.Fields{"error": err, "location": rgbFilename}).Error("Unable to write RGB profile data")
	}
}

func (d *Device) loadDeviceProfiles() {
	profileList := make(map[string]*DeviceProfile)
	profileDirectory := filepath.Join(pwd, "database", "profiles")
	files, err := os.ReadDir(profileDirectory)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": profileDirectory, "serial": d.Serial}).Warn("Unable to read profile directory")
		d.UserProfiles = profileList
		return
	}

	for _, fi := range files {
		if fi.IsDir() {
			continue
		}
		profileLocation := filepath.Join(profileDirectory, fi.Name())
		if !common.IsValidExtension(profileLocation, ".json") {
			continue
		}
		fileName := fi.Name()[:len(fi.Name())-len(filepath.Ext(fi.Name()))]
		if !common.AlphanumericDashRegex.MatchString(fileName) {
			continue
		}

		fileSerial := fileName
		profileName := "default"
		if split := slices.Index([]rune(fileName), '-'); split != -1 {
			runes := []rune(fileName)
			fileSerial = string(runes[:split])
			profileName = string(runes[split+1:])
		}
		if fileSerial != d.Serial {
			continue
		}

		profile := &DeviceProfile{}
		file, err := os.Open(profileLocation)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": profileLocation}).Warn("Unable to load profile")
			continue
		}
		if err = json.NewDecoder(file).Decode(profile); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": profileLocation}).Warn("Unable to decode profile")
			file.Close()
			continue
		}
		file.Close()
		profile.Path = profileLocation
		profileList[profileName] = profile
		if profile.Active {
			d.DeviceProfile = profile
		}
	}

	d.UserProfiles = profileList
}

func (d *Device) ensureDeviceProfile() {
	if d.DeviceProfile == nil {
		brightness := uint8(100)
		d.DeviceProfile = &DeviceProfile{
			Active:             true,
			Path:               filepath.Join(pwd, "database", "profiles", d.Serial+".json"),
			Product:            d.Product,
			Serial:             d.Serial,
			Brightness:         3,
			BrightnessSlider:   &brightness,
			OriginalBrightness: brightness,
			RGBProfiles:        make(map[int]string),
			Labels:             make(map[int]string),
			MultiRGB:           "rainbow",
		}
	}
	if d.DeviceProfile.BrightnessSlider == nil {
		brightness := uint8(100)
		d.DeviceProfile.BrightnessSlider = &brightness
	}
	if d.DeviceProfile.RGBProfiles == nil {
		d.DeviceProfile.RGBProfiles = make(map[int]string)
	}
	if d.DeviceProfile.Labels == nil {
		d.DeviceProfile.Labels = make(map[int]string)
	}

	d.DeviceProfile.Product = d.Product
	d.DeviceProfile.Serial = d.Serial
	for channelId, device := range d.Devices {
		if _, ok := d.DeviceProfile.RGBProfiles[channelId]; !ok {
			d.DeviceProfile.RGBProfiles[channelId] = d.DeviceProfile.MultiRGB
		}
		if d.DeviceProfile.RGBProfiles[channelId] == "" {
			d.DeviceProfile.RGBProfiles[channelId] = "rainbow"
		}
		if _, ok := d.DeviceProfile.Labels[channelId]; !ok {
			d.DeviceProfile.Labels[channelId] = device.Label
		}
		device.RGB = d.DeviceProfile.RGBProfiles[channelId]
		device.Label = d.DeviceProfile.Labels[channelId]
	}
	if _, ok := d.UserProfiles["default"]; !ok && filepath.Base(d.DeviceProfile.Path) == d.Serial+".json" {
		d.UserProfiles["default"] = d.DeviceProfile
	}
}

func (d *Device) saveDeviceProfile() {
	if d.DeviceProfile == nil {
		return
	}
	if d.DeviceProfile.Path == "" {
		d.DeviceProfile.Path = filepath.Join(pwd, "database", "profiles", d.Serial+".json")
	}
	if err := common.SaveJsonData(d.DeviceProfile.Path, d.DeviceProfile); err != nil {
		logger.Log(logger.Fields{"error": err, "location": d.DeviceProfile.Path}).Error("Unable to save device profile")
	}
	d.loadDeviceProfiles()
}

func (d *Device) restartRgb() {
	d.stopActiveRgb()
	d.setDeviceColor()
}

func (d *Device) stopActiveRgb() {
	d.mutex.Lock()
	active := d.activeRgb
	d.activeRgb = nil
	d.mutex.Unlock()

	if active != nil {
		close(active.Exit)
	}
}

func (d *Device) setActiveRgb(active *rgb.ActiveRGB) {
	d.mutex.Lock()
	d.activeRgb = active
	d.mutex.Unlock()
}

func (d *Device) setDeviceColor() {
	if d.DeviceProfile == nil {
		return
	}

	keys := make([]int, 0, len(d.Devices))
	for key := range d.Devices {
		keys = append(keys, key)
	}
	sort.Ints(keys)

	if d.DeviceProfile.RgbOff {
		d.writeAllZones([]byte{0, 0, 0})
		return
	}

	staticOnly := true
	for _, key := range keys {
		profileName := d.Devices[key].RGB
		if profileName != "static" && profileName != "off" && profileName != "led" {
			staticOnly = false
			break
		}
	}

	if staticOnly {
		for _, key := range keys {
			buffer := d.renderZone(d.Devices[key], time.Now(), nil)
			d.writeColor(buffer, key)
		}
		return
	}

	active := rgb.Exit()
	active.RGBStartColor = rgb.GenerateRandomColor(1)
	active.RGBEndColor = rgb.GenerateRandomColor(1)
	d.setActiveRgb(active)

	go func(active *rgb.ActiveRGB, keys []int) {
		startTime := time.Now()
		ticker := time.NewTicker(30 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-active.Exit:
				return
			case <-ticker.C:
				for _, key := range keys {
					buffer := d.renderZone(d.Devices[key], startTime, active)
					d.writeColor(buffer, key)
				}
			}
		}
	}(active, keys)
}

func (d *Device) renderZone(device *Devices, startTime time.Time, active *rgb.ActiveRGB) []byte {
	if device == nil || device.LedChannels == 0 {
		return []byte{0, 0, 0}
	}

	profile := d.GetRgbProfile(device.RGB)
	if profile == nil {
		return []byte{0, 0, 0}
	}

	brightness := rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
	if d.GlobalBrightness != 0 {
		brightness = d.GlobalBrightness
	}

	if device.RGB == "off" {
		return []byte{0, 0, 0}
	}

	if device.RGB == "static" || device.RGB == "led" {
		color := profile.StartColor
		color.Brightness = brightness
		profileColor := rgb.ModifyBrightness(color)
		return []byte{byte(profileColor.Red), byte(profileColor.Green), byte(profileColor.Blue)}
	}

	rgbCustomColor := true
	if (rgb.Color{}) == profile.StartColor || (rgb.Color{}) == profile.EndColor {
		rgbCustomColor = false
	}

	rgbModeSpeed := common.FClamp(profile.Speed, 0.1, 10)
	r := rgb.New(
		int(device.LedChannels),
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
		r.RGBMiddleColor = &profile.MiddleColor
	} else if active != nil {
		r.RGBStartColor = active.RGBStartColor
		r.RGBEndColor = active.RGBEndColor
		r.RGBMiddleColor = active.RGBMiddleColor
	}
	if r.RGBMiddleColor == nil {
		r.RGBMiddleColor = &rgb.Color{}
	}
	r.RGBBrightness = brightness
	r.RGBStartColor.Brightness = brightness
	r.RGBEndColor.Brightness = brightness
	r.RGBMiddleColor.Brightness = brightness
	r.ChannelId = device.ChannelId

	switch device.RGB {
	case "rainbow":
		r.Rainbow(startTime)
	case "pastelrainbow":
		r.PastelRainbow(startTime)
	case "spiralrainbow":
		r.SpiralRainbow(startTime)
	case "pastelspiralrainbow":
		r.PastelSpiralRainbow(startTime)
	case "watercolor":
		r.Watercolor(startTime)
	case "gradient":
		r.ColorshiftGradient(startTime, profile.Gradients, profile.Speed)
	case "colorpulse":
		r.Colorpulse(&startTime)
	case "rotator":
		r.Rotator(&startTime)
	case "wave":
		r.Wave(&startTime)
	case "storm":
		r.Storm()
	case "colorshift":
		if active != nil {
			r.Colorshift(&startTime, active)
		}
	case "circleshift":
		r.CircleShift(&startTime)
	case "circle":
		r.Circle(&startTime)
	case "spinner":
		r.Spinner(&startTime)
	case "colorwarp":
		if active != nil {
			r.Colorwarp(&startTime, active)
		}
	case "nebula":
		r.Nebula(&startTime)
	case "marquee":
		r.Marquee(&startTime)
	default:
		r.Static()
	}

	if len(r.Output) < 3 {
		return []byte{0, 0, 0}
	}
	return r.Output[:3]
}

func (d *Device) writeAllZones(color []byte) {
	for channelId := range d.Devices {
		d.writeColor(color, channelId)
	}
}

func (d *Device) writeColor(data []byte, channelId int) {
	if d.Exit || len(data) < 3 {
		return
	}

	brightness := d.currentBrightnessPercent()
	if data[0] == 0 && data[1] == 0 && data[2] == 0 {
		brightness = 0
	}

	if err := setNativeZone(d.native.Handle, channelId, data[0], data[1], data[2], brightness, d.native.TreatsRGBWAsRGB); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "zone": channelId}).Warn("Unable to set NVIDIA GPU RGB zone")
	}
}

func (d *Device) currentBrightnessPercent() uint8 {
	if d.DeviceProfile == nil || d.DeviceProfile.BrightnessSlider == nil {
		return 100
	}
	if d.GlobalBrightness != 0 {
		return uint8(common.Clamp(int(d.GlobalBrightness*100), 0, 100))
	}
	return uint8(common.Clamp(int(*d.DeviceProfile.BrightnessSlider), 0, 100))
}
