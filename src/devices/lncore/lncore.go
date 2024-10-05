package lncore

// Package: CORSAIR Lightning Node Core
// This is the primary package for CORSAIR Lightning Node Core
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ExternalLedDevice contains a list of supported external-LED devices connected to a HUB
type ExternalLedDevice struct {
	Index   int
	Name    string
	Total   int
	Command byte
}

type DeviceProfile struct {
	Active                  bool
	Path                    string
	Product                 string
	Serial                  string
	Brightness              uint8
	RGBProfiles             map[int]string
	Labels                  map[int]string
	ExternalHubDeviceType   int
	ExternalHubDeviceAmount int
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
	ExternalLed  bool
	ContainsPump bool
	CellSize     uint8
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
}

var (
	pwd, _                  = os.Getwd()
	cmdGetFirmware          = byte(0x02)
	cmdLedReset             = byte(0x37)
	cmdPortState            = byte(0x38)
	cmdWriteLedConfig       = byte(0x35)
	cmdWriteColor           = byte(0x32)
	cmdRefresh              = byte(0x33)
	cmdRefresh2             = byte(0x34)
	mutex                   sync.Mutex
	deviceRefreshInterval   = 1000
	bufferSize              = 64
	bufferSizeWrite         = bufferSize + 1
	maxBufferSizePerRequest = 50
	authRefreshChan         = make(chan bool)
	timer                   = &time.Ticker{}
	externalLedDevices      = []ExternalLedDevice{
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
		Template:          "lncore.html",
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
	d.setAutoRefresh()     // Set auto device refresh
	d.setColorEndpoint()   // Setup lightning
	d.saveDeviceProfile()  // Save profile
	d.setDeviceColor()     // Device color
	logger.Log(logger.Fields{"device": d}).Info("Device successfully initialized")
	return d
}

// ShutdownLed will reset LED ports and set device in 'hardware' mode
func (d *Device) ShutdownLed() {
	lightChannels := 0
	for _, device := range d.Devices {
		if device.LedChannels > 0 {
			lightChannels += int(device.LedChannels)
		}
	}

	config := []byte{0x00, 0x00, byte(lightChannels), 0x04, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff}

	_, err := d.transfer(cmdLedReset, []byte{0x00})
	if err != nil {
		return
	}
	_, err = d.transfer(cmdRefresh2, []byte{0x00})
	if err != nil {
		return
	}
	_, err = d.transfer(cmdPortState, []byte{0x00, 0x01})
	if err != nil {
		return
	}
	_, err = d.transfer(cmdWriteLedConfig, config)
	if err != nil {
		return
	}
	_, err = d.transfer(cmdRefresh, []byte{0xff})
	if err != nil {
		return
	}
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	logger.Log(logger.Fields{"serial": d.Serial}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}

	d.ShutdownLed()
	timer.Stop()
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
	product = strings.Replace(product, "CORSAIR ", "", -1)
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
		deviceProfile.ExternalHubDeviceAmount = d.DeviceProfile.ExternalHubDeviceAmount
		deviceProfile.ExternalHubDeviceType = d.DeviceProfile.ExternalHubDeviceType
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

// getDevices will fetch all devices connected to a hub
func (d *Device) getDevices() int {
	var devices = make(map[int]*Devices, 0)
	if d.DeviceProfile != nil {
		// External LED hub
		externalDeviceType := d.getExternalLedDevice(d.DeviceProfile.ExternalHubDeviceType)
		var LedChannels uint8 = 0
		if externalDeviceType != nil {
			LedChannels = uint8(externalDeviceType.Total)
		}

		if LedChannels > 0 {
			rgbProfile := "static"
			label := "Set Label"
			for z := 0; z < d.DeviceProfile.ExternalHubDeviceAmount; z++ {
				if d.DeviceProfile != nil {
					// Profile is set
					if rp, ok := d.DeviceProfile.RGBProfiles[z]; ok {
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

					// Device label
					if lb, ok := d.DeviceProfile.Labels[z]; ok {
						if len(lb) > 0 {
							label = lb
						}
					}
				} else {
					logger.Log(logger.Fields{"serial": d.Serial}).Warn("DeviceProfile is not set, probably first startup")
				}

				device := &Devices{
					ChannelId:   z,
					DeviceId:    fmt.Sprintf("%s-%v", "LED", z),
					Name:        externalDeviceType.Name,
					Description: "LED",
					HubId:       d.Serial,
					LedChannels: LedChannels,
					RGB:         rgbProfile,
					ExternalLed: true,
					CellSize:    2,
					Label:       label,
				}
				devices[z] = device
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
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
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

// UpdateExternalHubDeviceType will update a device type connected to the external-LED hub
func (d *Device) UpdateExternalHubDeviceType(externalType int) uint8 {
	if d.DeviceProfile != nil {
		if d.getExternalLedDevice(externalType) != nil {
			d.DeviceProfile.ExternalHubDeviceType = externalType
			if d.activeRgb != nil {
				d.activeRgb.Exit <- true // Exit current RGB mode
				d.activeRgb = nil
			}
			d.getDevices()        // Reload devices
			d.saveDeviceProfile() // Save profile
			d.setDeviceColor()    // Restart RGB
			return 1
		} else {
			return 2
		}
	}
	return 0
}

// UpdateExternalHubDeviceAmount will update device amount connected to an external-LED hub and trigger RGB reset
func (d *Device) UpdateExternalHubDeviceAmount(externalDevices int) uint8 {
	if d.DeviceProfile != nil {
		if d.DeviceProfile.ExternalHubDeviceType > 0 {
			d.DeviceProfile.ExternalHubDeviceAmount = externalDevices
			if d.activeRgb != nil {
				d.activeRgb.Exit <- true // Exit current RGB mode
				d.activeRgb = nil
			}
			d.getDevices()        // Reload devices
			d.saveDeviceProfile() // Save profile
			d.setDeviceColor()    // Restart RGB
			return 1
		}
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

	// Get the number of LED channels we have
	lightChannels := 0
	for _, device := range d.Devices {
		if device.LedChannels > 0 {
			lightChannels += int(device.LedChannels)
		}
	}

	// Reset all channels
	color := &rgb.Color{
		Red:        0,
		Green:      0,
		Blue:       0,
		Brightness: 0,
	}

	for i := 0; i < lightChannels; i++ {
		reset[i] = []byte{
			byte(color.Red),
			byte(color.Green),
			byte(color.Blue),
		}
	}

	buffer = rgb.SetColor(reset)
	d.writeColor(buffer, lightChannels)

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
			if d.DeviceProfile.Brightness != 0 {
				profile.StartColor.Brightness = rgb.GetBrightnessValue(d.DeviceProfile.Brightness)
			}

			profileColor := rgb.ModifyBrightness(profile.StartColor)
			for i := 0; i < lightChannels; i++ {
				reset[i] = []byte{
					byte(profileColor.Red),
					byte(profileColor.Green),
					byte(profileColor.Blue),
				}
			}
			buffer = rgb.SetColor(reset)
			d.writeColor(buffer, lightChannels) // Write color once
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
				externalKeys := make([]int, 0)
				internalKeys := make([]int, 0)
				for k := range d.Devices {
					if d.Devices[k].ExternalLed {
						externalKeys = append(externalKeys, k)
					} else {
						if d.Devices[k].LedChannels > 0 {
							internalKeys = append(internalKeys, k)
						}
					}
				}
				// Sort internal LED keys
				sort.Ints(internalKeys)

				// Sort external LED keys
				sort.Ints(externalKeys)

				// Append to main
				keys = append(keys, externalKeys...)
				keys = append(keys, internalKeys...)

				for _, k := range keys {
					rgbCustomColor := true
					profile := rgb.GetRgbProfile(d.Devices[k].RGB)
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
				d.writeColor(buff, lightChannels)
				time.Sleep(10 * time.Millisecond)
			}
		}
	}(lightChannels)
}

// setAutoRefresh will refresh device data
func (d *Device) setAutoRefresh() {
	timer = time.NewTicker(time.Duration(deviceRefreshInterval) * time.Millisecond)
	authRefreshChan = make(chan bool)
	go func() {
		for {
			select {
			case <-timer.C:
				_, err := d.transfer(byte(0x33), []byte{0xff})
				if err != nil {
					return
				}
			case <-authRefreshChan:
				timer.Stop()
				return
			}
		}
	}()
}

// writeColor will write data to the device with a specific endpoint.
func (d *Device) writeColor(data []byte, lightChannels int) {
	// Packets are sent like:
	// 50 packets of red, 50 packets on green, 50 packets of blue
	// Repeat until the buffer is empty.
	r := make([]byte, lightChannels)
	g := make([]byte, lightChannels)
	b := make([]byte, lightChannels)
	m := 0

	// FIll R. G. B buffer
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

	// Split each buffer into chunks
	chunksR := common.ProcessMultiChunkPacket(r, maxBufferSizePerRequest)
	packetsR := make(map[int][]byte, len(chunksR))
	for i, chunk := range chunksR {
		chunkLen := len(chunk)
		chunkPacket := make([]byte, chunkLen+4)
		chunkPacket[0] = 0x00
		chunkPacket[1] = byte(i * maxBufferSizePerRequest)
		chunkPacket[2] = byte(chunkLen)
		chunkPacket[3] = 0x00
		copy(chunkPacket[4:], chunk)
		packetsR[i] = chunkPacket
	}

	chunksG := common.ProcessMultiChunkPacket(g, maxBufferSizePerRequest)
	packetsG := make(map[int][]byte, len(chunksR))
	for i, chunk := range chunksG {
		chunkLen := len(chunk)
		chunkPacket := make([]byte, chunkLen+4)
		chunkPacket[0] = 0x00
		chunkPacket[1] = byte(i * maxBufferSizePerRequest)
		chunkPacket[2] = byte(chunkLen)
		chunkPacket[3] = 0x01
		copy(chunkPacket[4:], chunk)
		packetsG[i] = chunkPacket
	}

	chunksB := common.ProcessMultiChunkPacket(b, maxBufferSizePerRequest)
	packetsB := make(map[int][]byte, len(chunksB))
	for i, chunk := range chunksB {
		chunkLen := len(chunk)
		chunkPacket := make([]byte, chunkLen+4)
		chunkPacket[0] = 0x00
		chunkPacket[1] = byte(i * maxBufferSizePerRequest)
		chunkPacket[2] = byte(chunkLen)
		chunkPacket[3] = 0x02
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

	_, err = d.transfer(cmdPortState, []byte{0x00, 0x02})
	if err != nil {
		return
	}
}

// setColorEndpoint will activate hub color endpoint for further usage
func (d *Device) setColorEndpoint() {
	// Get the number of LED channels we have
	lightChannels := 0
	for _, device := range d.Devices {
		if device.LedChannels > 0 {
			lightChannels += int(device.LedChannels)
		}
	}

	config := []byte{0x00, 0x00, byte(lightChannels), 0x04, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff}

	_, err := d.transfer(cmdLedReset, []byte{0x00})
	if err != nil {
		return
	}

	_, err = d.transfer(cmdWriteLedConfig, config)
	if err != nil {
		return
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint byte, buffer []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	mutex.Lock()
	defer mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = endpoint

	if buffer != nil && len(buffer) > 0 {
		if len(buffer) > bufferSize-1 {
			buffer = buffer[:bufferSize-1]
		}
		copy(bufferW[2:], buffer)
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

	return bufferR, nil
}
