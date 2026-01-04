package lnpro

// Package: CORSAIR Lightning Node Pro
// This is the primary package for CORSAIR Lightning Node Pro.
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
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// ExternalLedDevice contains a list of supported external-LED devices connected to a HUB
type ExternalLedDevice struct {
	Index    int
	Name     string
	Total    int
	Command  byte
	Triangle bool
}

type ExternalHubData struct {
	PortId                  byte
	ExternalHubDeviceType   int
	ExternalHubDeviceAmount int
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
	ExternalHubs       map[int]*ExternalHubData
	HardwareMode       int
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
	IsTriangle   bool
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
	activeRgb               *rgb.ActiveRGB
	Template                string
	Brightness              map[int]string
	HasLCD                  bool
	CpuTemp                 float32
	GpuTemp                 float32
	Rgb                     *rgb.RGB
	rgbMutex                sync.RWMutex
	Exit                    bool
	mutex                   sync.Mutex
	autoRefreshChan         chan struct{}
	timer                   *time.Ticker
	State                   map[byte]bool
	HardwareModes           map[int]string
	RGBModes                []string
	instance                *common.Device
}

var (
	pwd                     = ""
	cmdGetFirmware          = byte(0x02)
	cmdLedReset             = byte(0x37)
	cmdPortState            = byte(0x38)
	cmdWriteLedConfig       = byte(0x35)
	cmdWriteColor           = byte(0x32)
	cmdSave                 = byte(0x33)
	cmdRefresh2             = byte(0x34)
	dataFlush               = []byte{0xff}
	deviceRefreshInterval   = 1000
	bufferSize              = 64
	bufferSizeWrite         = bufferSize + 1
	readBufferSize          = 17
	maxBufferSizePerRequest = 50
	maximumLedAmount        = 204
	deviceUpdateDelay       = 5
	rgbProfileUpgrade       = []string{"gradient"}
	rgbModes                = []string{
		"circle",
		"circleshift",
		"colorpulse",
		"colorshift",
		"colorwarp",
		"cpu-temperature",
		"flickering",
		"gpu-temperature",
		"gradient",
		"off",
		"rainbow",
		"pastelrainbow",
		"rotator",
		"spinner",
		"static",
		"storm",
		"watercolor",
		"wave",
	}
	hardwareLights = map[int][]byte{
		0: {0x02, 0x01, 0x00, 0x01, 0x00, 0x00},
		1: {0x01, 0x01, 0x00, 0x01, 0x00, 0x00},
		2: {0x03, 0x01, 0x01, 0x01, 0x00, 0x00},
		3: {0x07, 0x01, 0x00, 0x00, 0x00, 0xff},
		4: {0x0a, 0x01, 0x00, 0x01, 0x00, 0x00},
		5: {0x00, 0x01, 0x01, 0x01, 0x00, 0x00},
		6: {0x09, 0x01, 0x01, 0x01, 0x00, 0x00},
		7: {0x08, 0x01, 0x00, 0x01, 0x00, 0x00},
		8: {0x06, 0x01, 0x01, 0x01, 0x00, 0x00},
	}
)

// Init will initialize a new device
func Init(vendorId, productId uint16, serial, _ string) *common.Device {
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
		Template: "lnpro.html",
		ExternalLedDeviceAmount: map[int]string{
			0: "No Device",
			1: "1 Device",
			2: "2 Devices",
			3: "3 Devices",
			4: "4 Devices",
			5: "5 Devices",
			6: "6 Devices",
			7: "7 Devices",
			8: "8 Devices",
			9: "9 Devices",
		},
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
		timer:           &time.Ticker{},
		autoRefreshChan: make(chan struct{}),
		State:           make(map[byte]bool, 2),
		HardwareModes: map[int]string{
			0: "Color Pulse",
			1: "Color Shift",
			2: "Color Wave",
			3: "Marquee",
			4: "Rainbow",
			5: "Rainbow Wave",
			6: "Sequential",
			7: "Strobing",
			8: "Visor",
		},
		RGBModes: rgbModes,
	}

	// Bootstrap
	d.getManufacturer()     // Manufacturer
	d.getProduct()          // Product
	d.getSerial()           // Serial
	d.loadExternalDevices() // External metadata
	d.loadRgb()             // Load RGB
	d.loadDeviceProfiles()  // Load all device profiles
	d.getDeviceFirmware()   // Firmware
	d.getDevices()          // Get devices connected to a hub
	d.setAutoRefresh()      // Set auto device refresh
	d.saveDeviceProfile()   // Create device profile
	d.setColorEndpoint()    // Setup lightning
	d.setDeviceColor(true)  // Device color
	d.createDevice()        // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeLnPro,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-device.svg",
		Instance:    d,
	}
}

// ShutdownLed will reset LED ports and set device in 'hardware' mode
func (d *Device) ShutdownLed() {
	var hardwareLight []byte
	if hardwareMode, ok := hardwareLights[d.DeviceProfile.HardwareMode]; ok {
		hardwareLight = hardwareMode
	} else {
		hardwareLight = []byte{0x00, 0x01, 0x01, 0x01, 0x00, 0x00}
	}

	for i := 0; i < len(d.DeviceProfile.ExternalHubs); i++ {
		externalHub := d.DeviceProfile.ExternalHubs[i]
		lightChannels := 0
		for _, device := range d.Devices {
			if device.PortId == externalHub.PortId {
				lightChannels += int(device.LedChannels)
			}
		}
		if lightChannels > 0 {
			buff := make([]byte, 9)
			buff[0] = externalHub.PortId
			buff[1] = 0x00
			buff[2] = byte(lightChannels)
			copy(buff[3:], hardwareLight)

			_, err := d.transfer(cmdLedReset, []byte{externalHub.PortId}, false)
			if err != nil {
				return
			}
			_, err = d.transfer(cmdRefresh2, []byte{externalHub.PortId}, false)
			if err != nil {
				return
			}
			_, err = d.transfer(cmdPortState, []byte{externalHub.PortId, 0x01}, false)
			if err != nil {
				return
			}
			_, err = d.transfer(cmdWriteLedConfig, buff, false)
			if err != nil {
				return
			}
		}
	}
	_, err := d.transfer(cmdSave, dataFlush, false)
	if err != nil {
		return
	}
}

// GetRgbProfiles will return RGB profiles for a target device
func (d *Device) GetRgbProfiles() interface{} {
	tmp := *d.Rgb

	// Filter unsupported modes out
	profiles := make(map[string]rgb.Profile, len(tmp.Profiles))
	for key, value := range tmp.Profiles {
		if slices.Contains(rgbModes, key) {
			profiles[key] = value
		}
	}
	tmp.Profiles = profiles
	return tmp
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.ShutdownLed()

	d.timer.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.autoRefreshChan != nil {
				close(d.autoRefreshChan)
			}
		})
	}()

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
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.timer.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.autoRefreshChan != nil {
				close(d.autoRefreshChan)
			}
		})
	}()
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 1
}

// loadExternalDevices will load external device definitions
func (d *Device) loadExternalDevices() {
	externalDevicesFile := pwd + "/database/external/lnpro.json"
	if common.FileExists(externalDevicesFile) {
		file, err := os.Open(externalDevicesFile)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": externalDevicesFile}).Warn("Unable to load external devices metadata")
			return
		}
		if err = json.NewDecoder(file).Decode(&d.ExternalLedDevice); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": externalDevicesFile}).Warn("Unable to decode external devices metadata")
			return
		}
		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"location": externalDevicesFile, "serial": d.Serial}).Warn("Failed to close external devices metadata")
		}
	} else {
		logger.Log(logger.Fields{"serial": d.Serial, "location": externalDevicesFile}).Warn("Unable to load external devices metadata")
		d.ExternalLedDevice = []ExternalLedDevice{
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
				Name:  "SP RGB Series Fan (1 LED)",
				Total: 1,
			},
			{
				Index: 7,
				Name:  "LC100 Accent Triangles",
				Total: 9,
			},
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
			template := rgb.GetRgbProfile(profile)
			if template == nil {
				d.Rgb.Profiles[profile] = rgb.Profile{}
			} else {
				d.Rgb.Profiles[profile] = *template
			}
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
func (d *Device) getDeviceFirmware() {
	fw, err := d.transfer(
		cmdGetFirmware,
		nil,
		true,
	)

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
		ExternalHubs:       make(map[int]*ExternalHubData),
		Labels:             labels,
		Path:               profilePath,
		BrightnessSlider:   &defaultBrightness,
		OriginalBrightness: 100,
		HardwareMode:       0,
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
		if d.DeviceProfile.BrightnessSlider == nil {
			deviceProfile.BrightnessSlider = &defaultBrightness
			d.DeviceProfile.BrightnessSlider = &defaultBrightness
		} else {
			deviceProfile.BrightnessSlider = d.DeviceProfile.BrightnessSlider
		}
		deviceProfile.ExternalHubs = d.DeviceProfile.ExternalHubs
		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.Brightness = d.DeviceProfile.Brightness
		deviceProfile.OriginalBrightness = d.DeviceProfile.OriginalBrightness
		deviceProfile.HardwareMode = d.DeviceProfile.HardwareMode
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
		logger.Log(logger.Fields{"error": fileErr, "location": deviceProfile.Path}).Error("Unable to create new device profile")
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
	var devices = make(map[int]*Devices)
	if d.DeviceProfile != nil {
		m := 0
		var LedChannels uint8 = 0

		for i := 0; i < len(d.DeviceProfile.ExternalHubs); i++ {
			externalHub := d.DeviceProfile.ExternalHubs[i]
			externalDeviceType := d.getExternalLedDevice(externalHub.ExternalHubDeviceType)
			if externalDeviceType != nil {
				LedChannels = uint8(externalDeviceType.Total)
				for z := 0; z < externalHub.ExternalHubDeviceAmount; z++ {
					rgbProfile := "static"
					label := "Set Label"

					if rp, ok := d.DeviceProfile.RGBProfiles[m]; ok {
						if d.GetRgbProfile(rp) != nil { // Speed profile exists in configuration
							// Speed profile exists in configuration
							rgbProfile = rp
						}
					}

					// Device label
					if lb, ok := d.DeviceProfile.Labels[m]; ok {
						if len(lb) > 0 {
							label = lb
						}
					}

					device := &Devices{
						ChannelId:   m,
						DeviceId:    fmt.Sprintf("%s-%v", "LED", m),
						Name:        externalDeviceType.Name,
						Description: "LED",
						HubId:       d.Serial,
						LedChannels: LedChannels,
						RGB:         rgbProfile,
						CellSize:    2,
						PortId:      externalHub.PortId,
						Label:       label,
						IsTriangle:  externalDeviceType.Triangle,
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
	for _, externalLedDevice := range d.ExternalLedDevice {
		if externalLedDevice.Index == index {
			return &externalLedDevice
		}
	}
	return nil
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

// ProcessNewGradientColor will create new gradient color
func (d *Device) ProcessNewGradientColor(profileName string) (uint8, uint) {
	if d.GetRgbProfile(profileName) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profileName}).Warn("Non-existing RGB profile")
		return 0, 0
	}

	pf := d.GetRgbProfile(profileName)
	if pf == nil {
		return 0, 0
	}

	if pf.Gradients == nil {
		return 0, 0
	}

	// find next available key
	nextID := 0
	for k := range pf.Gradients {
		if k >= nextID {
			nextID = k + 1
		}
	}
	pf.Gradients[nextID] = rgb.Color{Red: 0, Green: 255, Blue: 255}

	d.Rgb.Profiles[profileName] = *pf
	d.saveRgbProfile()
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor(false) // Restart RGB
	return 1, uint(nextID)
}

// ProcessDeleteGradientColor will delete gradient color
func (d *Device) ProcessDeleteGradientColor(profileName string) (uint8, uint) {
	if d.GetRgbProfile(profileName) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profileName}).Warn("Non-existing RGB profile")
		return 0, 0
	}

	pf := d.GetRgbProfile(profileName)
	if pf == nil {
		return 0, 0
	}

	if len(pf.Gradients) < 3 {
		return 2, 0
	}

	maxKey := -1
	for k := range pf.Gradients {
		if k > maxKey {
			maxKey = k
		}
	}
	delete(pf.Gradients, maxKey)

	d.Rgb.Profiles[profileName] = *pf
	d.saveRgbProfile()
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor(false)
	return 1, uint(maxKey)
}

// UpdateRgbProfileData will update RGB profile data
func (d *Device) UpdateRgbProfileData(profileName string, profile rgb.Profile) uint8 {
	d.rgbMutex.Lock()
	defer d.rgbMutex.Unlock()

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
	pf.Gradients = profile.Gradients

	d.Rgb.Profiles[profileName] = *pf
	d.saveRgbProfile()
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor(true) // Restart RGB
	return 1
}

// UpdateRgbProfile will update device RGB profile
func (d *Device) UpdateRgbProfile(channelId int, profile string) uint8 {
	if d.GetRgbProfile(profile) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
	}

	if profile == "liquid-temperature" {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Unable to apply liquid-temperature profile without a pump of AIO")
		return 2
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

	d.setDeviceColor(true) // Restart RGB
	return 1
}

// ResetRgb will reset the current rgb mode
func (d *Device) ResetRgb() {
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}

	d.getDevices()        // Reload devices
	d.saveDeviceProfile() // Save profile
	d.ShutdownLed()
	d.setColorEndpoint()
	d.setDeviceColor(true) // Restart RGB
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

// UpdateHardwareRgbProfile will update a device type connected to the external-LED hub
func (d *Device) UpdateHardwareRgbProfile(hardwareLight int) uint8 {
	if d.DeviceProfile != nil {
		if _, ok := hardwareLights[hardwareLight]; ok {
			if d.DeviceProfile.HardwareMode != hardwareLight {
				d.DeviceProfile.HardwareMode = hardwareLight
				d.saveDeviceProfile()
				return 1
			}
		}
	}
	return 0
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

	d.setDeviceColor(true) // Restart RGB
	return 1
}

// ChangeDeviceBrightnessValue will change device brightness via slider
func (d *Device) ChangeDeviceBrightnessValue(value uint8) uint8 {
	if value < 0 || value > 100 {
		return 0
	}

	d.DeviceProfile.BrightnessSlider = &value
	d.saveDeviceProfile()

	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}

	d.setDeviceColor(false) // Restart RGB
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
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}

	d.setDeviceColor(false) // Restart RGB
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
		d.setDeviceColor(true)
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
			if d.activeRgb != nil {
				d.activeRgb.Exit <- true // Exit current RGB mode
				d.activeRgb = nil
			}

			d.ResetRgb()
			return 1
		}
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
func (d *Device) setDeviceColor(resetColor bool) {
	lightChannels := 0
	keys := make([]int, 0)
	for k := range d.Devices {
		lightChannels += int(d.Devices[k].LedChannels)
		keys = append(keys, k)
	}
	sort.Ints(keys)

	if lightChannels == 0 {
		logger.Log(logger.Fields{}).Info("No RGB compatible devices found")
		return
	}

	if resetColor {
		color := &rgb.Color{Red: 0, Green: 0, Blue: 0, Brightness: 0}
		buff := make(map[byte][]byte)
		for _, k := range keys {
			for i := 0; i < int(d.Devices[k].LedChannels); i++ {
				buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], []byte{
					byte(color.Red),
					byte(color.Green),
					byte(color.Blue),
				}...)
			}
		}
		d.writeColor(buff)
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
			profile := d.GetRgbProfile("static")
			if profile == nil {
				return
			}
			profile.StartColor.Brightness = rgb.GetBrightnessValueFloat(*d.DeviceProfile.BrightnessSlider)
			profileColor := rgb.ModifyBrightness(profile.StartColor)
			buff := make(map[byte][]byte)
			for _, k := range keys {
				for i := 0; i < int(d.Devices[k].LedChannels); i++ {
					buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], []byte{
						byte(profileColor.Red),
						byte(profileColor.Green),
						byte(profileColor.Blue),
					}...)
				}
			}
			d.writeColor(buff)
			return
		}
	}
	go func() {
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
				buff := make(map[byte][]byte)
				for _, k := range keys {
					rgbCustomColor := true
					profile := d.GetRgbProfile(d.Devices[k].RGB)
					if profile == nil {
						for i := 0; i < int(d.Devices[k].LedChannels); i++ {
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], []byte{0, 0, 0}...)
						}
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
					r.ChannelId = k

					switch d.Devices[k].RGB {
					case "off":
						{
							for n := 0; n < int(d.Devices[k].LedChannels); n++ {
								buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], []byte{0, 0, 0}...)
							}
						}
					case "rainbow":
						{
							r.Rainbow(startTime)
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "pastelrainbow":
						{
							r.PastelRainbow(startTime)
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "watercolor":
						{
							r.Watercolor(startTime)
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "gradient":
						{
							r.ColorshiftGradient(startTime, profile.Gradients, profile.Speed)
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "cpu-temperature":
						{
							r.MinTemp = profile.MinTemp
							r.MaxTemp = profile.MaxTemp
							r.Temperature(float64(d.CpuTemp))
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "gpu-temperature":
						{
							r.MinTemp = profile.MinTemp
							r.MaxTemp = profile.MaxTemp
							r.Temperature(float64(d.GpuTemp))
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "colorpulse":
						{
							r.Colorpulse(&startTime)
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "static":
						{
							r.Static()
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "rotator":
						{
							r.Rotator(&startTime)
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "wave":
						{
							r.Wave(&startTime)
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "storm":
						{
							r.Storm()
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "flickering":
						{
							r.Flickering(&startTime)
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "colorshift":
						{
							r.Colorshift(&startTime, d.activeRgb)
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "circleshift":
						{
							r.CircleShift(&startTime)
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "circle":
						{
							r.Circle(&startTime)
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "spinner":
						{
							r.Spinner(&startTime)
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					case "colorwarp":
						{
							r.Colorwarp(&startTime, d.activeRgb)
							buff[d.Devices[k].PortId] = append(buff[d.Devices[k].PortId], r.Output...)
						}
					}
				}
				// Send it
				d.writeColor(buff)
				time.Sleep(time.Duration(deviceUpdateDelay) * time.Millisecond)
			}
		}
	}()
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
				_, err := d.transfer(cmdSave, dataFlush, false)
				if err != nil {
					return
				}
			case <-d.autoRefreshChan:
				d.timer.Stop()
				return
			}
		}
	}()
}

// writeColor will write data to the device with a specific endpoint.
func (d *Device) writeColor(buffer map[byte][]byte) {
	if d.Exit {
		return
	}

	for portId, data := range buffer {
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

		_, err := d.transfer(cmdPortState, []byte{portId, 0x02}, false)
		if err != nil {
			return
		}

		if !d.State[portId] {
			d.State[portId] = true
			_, err = d.transfer(cmdPortState, []byte{portId}, false)
			if err != nil {
				return
			}
		}

		for z := 0; z < len(chunksR); z++ {
			if d.Exit {
				return
			}
			chunkPacket := make([]byte, len(chunksR[z])+4)
			chunkPacket[0] = portId
			chunkPacket[1] = byte(z * maxBufferSizePerRequest)
			chunkPacket[2] = byte(len(chunksR[z]))
			chunkPacket[3] = 0x00
			copy(chunkPacket[4:], chunksR[z])
			_, err = d.transfer(cmdWriteColor, chunkPacket, false)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to write color to device")
			}

			chunkPacket[0] = portId
			chunkPacket[1] = byte(z * maxBufferSizePerRequest)
			chunkPacket[2] = byte(len(chunksG[z]))
			chunkPacket[3] = 0x01
			copy(chunkPacket[4:], chunksG[z])
			_, err = d.transfer(cmdWriteColor, chunkPacket, false)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to write color to device")
			}

			chunkPacket[0] = portId
			chunkPacket[1] = byte(z * maxBufferSizePerRequest)
			chunkPacket[2] = byte(len(chunksB[z]))
			chunkPacket[3] = 0x02
			copy(chunkPacket[4:], chunksB[z])
			_, err = d.transfer(cmdWriteColor, chunkPacket, false)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to write color to device")
			}
		}

		_, err = d.transfer(cmdSave, dataFlush, false)
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to write blue color to device")
		}
	}
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
			_, err := d.transfer(cmdLedReset, []byte{externalHub.PortId}, false)
			if err != nil {
				return
			}

			_, err = d.transfer(cmdWriteLedConfig, cfg, false)
			if err != nil {
				return
			}
		}
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint byte, buffer []byte, read bool) ([]byte, error) {
	// Packet control, mandatory for this device
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = endpoint

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

	if read {
		// Get data from a device
		if _, err := d.dev.Read(bufferR); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
			return bufferR, err
		}
	}
	return bufferR, nil
}
