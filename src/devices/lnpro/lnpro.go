package lnpro

// Package: CORSAIR Lightning Node Pro
// This is the primary package for CORSAIR Lightning Node Pro
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
	"sort"
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

type ExternalHubData struct {
	PortId                  byte
	ExternalHubDeviceType   int
	ExternalHubDeviceAmount int
}

type DeviceProfile struct {
	Product      string
	Serial       string
	RGBProfiles  map[int]string
	Labels       map[int]string
	ExternalHubs map[int]*ExternalHubData
}

type Devices struct {
	ChannelId   int    `json:"channelId"`
	Type        byte   `json:"type"`
	Model       byte   `json:"-"`
	DeviceId    string `json:"deviceId"`
	Name        string `json:"name"`
	LedChannels uint8  `json:"-"`
	Description string `json:"description"`
	HubId       string `json:"-"`
	Profile     string `json:"profile"`
	RGB         string `json:"rgb"`
	Label       string `json:"label"`
	PortId      byte   `json:"-"`
	CellSize    uint8
}

type Device struct {
	dev                     *hid.Device
	Manufacturer            string           `json:"manufacturer"`
	Product                 string           `json:"product"`
	Serial                  string           `json:"serial"`
	Firmware                string           `json:"firmware"`
	Devices                 map[int]*Devices `json:"devices"`
	DeviceProfile           *DeviceProfile
	profileConfig           string
	ExternalLedDeviceAmount map[int]string
	ExternalLedDevice       []ExternalLedDevice
	activeRgb               map[int]*rgb.ActiveRGB
	Template                string
}

var (
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
	maximumLedAmount        = 204
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
		Template:          "lnpro.html",
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
	}

	// Bootstrap
	d.getManufacturer()   // Manufacturer
	d.getProduct()        // Product
	d.getSerial()         // Serial
	d.setProfileConfig()  // Device profile
	d.getDeviceProfile()  // Get device profile if any
	d.getDeviceFirmware() // Firmware
	d.getDevices()        // Get devices connected to a hub
	d.setAutoRefresh()    // Set auto device refresh
	d.saveDeviceProfile() // Create device profile
	d.setColorEndpoint()  // Setup lightning
	d.setDeviceColor()    // Activate device RGB
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
		config := []byte{externalHub.PortId, 0x00, byte(lightChannels), 0x04, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff}
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
		_, err = d.transfer(cmdWriteLedConfig, config)
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
	authRefreshChan <- true

	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
		}
	}
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
	)

	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to write to a device")
		return
	}

	v1, v2, v3 := int(fw[1]), int(fw[2]), int(fw[3])
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
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
	rgbProfiles := make(map[int]string, len(d.Devices))
	labels := make(map[int]string, len(d.Devices))

	for _, device := range d.Devices {
		if device.LedChannels > 0 {
			rgbProfiles[device.ChannelId] = device.RGB
		}
		labels[device.ChannelId] = device.Label
	}

	deviceProfile := &DeviceProfile{
		Product:      d.Product,
		Serial:       d.Serial,
		RGBProfiles:  rgbProfiles,
		ExternalHubs: make(map[int]*ExternalHubData, 2),
		Labels:       labels,
	}

	// First save, assign saved profile to a device
	if d.DeviceProfile == nil {
		for _, device := range d.Devices {
			if device.LedChannels > 0 {
				rgbProfiles[device.ChannelId] = "static"
			}
			labels[device.ChannelId] = "Not Set"
		}
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
		logger.Log(logger.Fields{"error": err, "location": d.profileConfig}).Error("Unable to close file handle")
	}
}

// getDevices will fetch all devices connected to a hub
func (d *Device) getDevices() int {
	var devices = make(map[int]*Devices, 0)
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
					label := "Not Set"

					if rp, ok := d.DeviceProfile.RGBProfiles[m]; ok {
						if rgb.GetRgbProfile(rp) != nil { // Speed profile exists in configuration
							// Speed profile exists in configuration
							rgbProfile = rp
						}
					}

					// Device label
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
						CellSize:    2,
						PortId:      externalHub.PortId,
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
				if d.Devices[k].PortId == externalHub.PortId {
					rgbCustomColor := true
					lc += int(d.Devices[k].LedChannels)
					keys = append(keys, k)
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
						r.RGBStartColor = d.activeRgb[i].RGBStartColor
						r.RGBEndColor = d.activeRgb[i].RGBEndColor
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
			config := []byte{externalHub.PortId, 0x00, byte(lightChannels), 0x04, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff}
			_, err := d.transfer(cmdLedReset, []byte{externalHub.PortId})
			if err != nil {
				return
			}

			_, err = d.transfer(cmdWriteLedConfig, config)
			if err != nil {
				return
			}
		}
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint byte, buffer []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	mutex.Lock()
	defer mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[0] = endpoint

	if buffer != nil && len(buffer) > 0 {
		if len(buffer) > bufferSize-1 {
			buffer = buffer[:bufferSize-1]
		}
		copy(bufferW[1:], buffer)
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
