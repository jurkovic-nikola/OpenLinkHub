package mm700

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/temperatures"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Device struct {
	Debug         bool
	dev           *hid.Device
	Manufacturer  string `json:"manufacturer"`
	Product       string `json:"product"`
	Serial        string `json:"serial"`
	Firmware      string `json:"firmware"`
	activeRgb     *rgb.ActiveRGB
	DeviceProfile *DeviceProfile
	UserProfiles  map[string]*DeviceProfile `json:"userProfiles"`
	Brightness    map[int]string
	Template      string
	VendorId      uint16
	ProductId     uint16
	LEDChannels   int
	CpuTemp       float32
	GpuTemp       float32
}

type ZoneColor struct {
	Color       *rgb.Color
	PacketIndex int
}

// DeviceProfile struct contains all device profile
type DeviceProfile struct {
	Active     bool
	Path       string
	Product    string
	Serial     string
	Brightness uint8
	RGBProfile string
	Label      string
	Stand      *Stand
}

type Stand struct {
	Row map[int]Row `json:"row"`
}

type Row struct {
	Zones map[int]Zones `json:"zones"`
}

type Zones struct {
	Name        string    `json:"name"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Left        int       `json:"left"`
	Top         int       `json:"top"`
	PacketIndex []int     `json:"packetIndex"`
	Color       rgb.Color `json:"color"`
}

var (
	pwd                   = ""
	mutex                 sync.Mutex
	bufferSize            = 64
	bufferSizeWrite       = bufferSize + 1
	headerSize            = 2
	headerSizeWrite       = 4
	deviceRefreshInterval = 1000
	timer                 = &time.Ticker{}
	timerKeepAlive        = &time.Ticker{}
	authRefreshChan       = make(chan bool)
	keepAliveChan         = make(chan bool)
	deviceKeepAlive       = 20000
	cmdWrite              = byte(0x08)
	cmdSoftwareMode       = []byte{0x01, 0x03, 0x00, 0x02}
	cmdHardwareMode       = []byte{0x01, 0x03, 0x00, 0x01}
	cmdGetFirmware        = []byte{0x02, 0x13}
	cmdWriteColor         = []byte{0x06, 0x00}
	cmdActivateLed        = []byte{0x0d, 0x00, 0x01}
	cmdKeepAlive          = []byte{0x12}
	colorPacketLength     = 9
)

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	logger.Log(logger.Fields{"serial": d.Serial}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}
	timer.Stop()
	authRefreshChan <- true

	d.setHardwareMode()
	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to close HID device")
		}
	}
}

func Init(vendorId, productId uint16, key string) *Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	// Open device, return if failure
	dev, err := hid.OpenPath(key)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId, "serial": key}).Error("Unable to open HID device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:       dev,
		VendorId:  vendorId,
		ProductId: productId,
		Product:   "MM700 RGB",
		Template:  "mm700.html",
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
		LEDChannels: 9,
	}

	d.getDebugMode()       // Debug mode
	d.getManufacturer()    // Manufacturer
	d.getSerial()          // Serial
	d.setSoftwareMode()    // Activate software mode
	d.getDeviceFirmware()  // Firmware
	d.loadDeviceProfiles() // Load all device profiles
	d.saveDeviceProfile()  // Save profile
	d.setAutoRefresh()     // Set auto device refresh
	d.setKeepAlive()       // Keepalive
	d.initLeds()           // Init LED
	d.setDeviceColor()     // Device color
	return d
}

// getManufacturer will return device manufacturer
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
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

// setSoftwareMode will switch a device to software mode
func (d *Device) setSoftwareMode() {
	_, err := d.transfer(cmdWrite, cmdSoftwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// setHardwareMode will switch a device to hardware mode
func (d *Device) setHardwareMode() {
	_, err := d.transfer(cmdWrite, cmdHardwareMode, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// getDeviceFirmware will return a device firmware version out as string
func (d *Device) getDeviceFirmware() {
	fw, err := d.transfer(
		cmdWrite,
		cmdGetFirmware,
		nil,
	)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to write to a device")
	}
	v1, v2, v3 := int(fw[3]), int(fw[4]), int(fw[5])
	d.Firmware = fmt.Sprintf("%d.%d.%d", v1, v2, v3)
}

// initLeds will initialize LED ports
func (d *Device) initLeds() {
	_, err := d.transfer(cmdWrite, cmdActivateLed, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to change device mode")
	}
}

// keepAlive will keep a device alive
func (d *Device) keepAlive() {
	_, err := d.transfer(cmdWrite, cmdKeepAlive, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
	}
}

// setAutoRefresh will refresh device data
func (d *Device) setKeepAlive() {
	timerKeepAlive = time.NewTicker(time.Duration(deviceKeepAlive) * time.Millisecond)
	keepAliveChan = make(chan bool)
	go func() {
		for {
			select {
			case <-timerKeepAlive.C:
				d.keepAlive()
			case <-keepAliveChan:
				timerKeepAlive.Stop()
				return
			}
		}
	}()
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
			case <-authRefreshChan:
				timer.Stop()
				return
			}
		}
	}()
}

// setCpuTemperature will store current CPU temperature
func (d *Device) setTemperatures() {
	d.CpuTemp = temperatures.GetCpuTemperature()
	d.GpuTemp = temperatures.GetGpuTemperature()
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
		deviceProfile.RGBProfile = "mousepad"
		deviceProfile.Label = "Mousepad"
		deviceProfile.Active = true
		deviceProfile.Stand = &Stand{
			Row: map[int]Row{
				1: {
					Zones: map[int]Zones{
						1: {Name: "", Width: 150, Height: 150, Left: 0, Top: 0, PacketIndex: []int{0}, Color: rgb.Color{Red: 0, Green: 255, Blue: 255, Brightness: 1}},
						2: {Name: "LOGO", Width: 150, Height: 40, Left: 360, Top: 0, PacketIndex: []int{2}, Color: rgb.Color{Red: 255, Green: 255, Blue: 0, Brightness: 1}},
					},
				},
				2: {
					map[int]Zones{
						3: {Name: "", Width: 150, Height: 150, Left: 510, Top: 58, PacketIndex: []int{1}, Color: rgb.Color{Red: 0, Green: 255, Blue: 255, Brightness: 1}},
					},
				},
			},
		}
	} else {
		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.Brightness = d.DeviceProfile.Brightness
		deviceProfile.RGBProfile = d.DeviceProfile.RGBProfile
		deviceProfile.Label = d.DeviceProfile.Label
		deviceProfile.Stand = d.DeviceProfile.Stand
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

// SaveDeviceProfile will save a new device profile
func (d *Device) SaveDeviceProfile() uint8 {
	d.saveDeviceProfile()
	return 1
}

// UpdateRgbProfile will update device RGB profile
func (d *Device) UpdateRgbProfile(profile string) uint8 {
	if rgb.GetRgbProfile(profile) == nil {
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

// UpdateDeviceColor will update device color based on selected input
func (d *Device) UpdateDeviceColor(keyId, keyOption int, color rgb.Color) uint8 {
	switch keyOption {
	case 0:
		{
			for rowIndex, row := range d.DeviceProfile.Stand.Row {
				for keyIndex, key := range row.Zones {
					if keyIndex == keyId {
						key.Color = rgb.Color{
							Red:        color.Red,
							Green:      color.Green,
							Blue:       color.Blue,
							Brightness: 0,
						}
						d.DeviceProfile.Stand.Row[rowIndex].Zones[keyIndex] = key
						if d.activeRgb != nil {
							d.activeRgb.Exit <- true // Exit current RGB mode
							d.activeRgb = nil
						}
						d.setDeviceColor() // Restart RGB
						return 1
					}
				}
			}
		}
	case 1:
		{
			rowId := -1
			for rowIndex, row := range d.DeviceProfile.Stand.Row {
				for keyIndex := range row.Zones {
					if keyIndex == keyId {
						rowId = rowIndex
						break
					}
				}
			}

			if rowId < 0 {
				return 0
			}

			for keyIndex, key := range d.DeviceProfile.Stand.Row[rowId].Zones {
				key.Color = rgb.Color{
					Red:        color.Red,
					Green:      color.Green,
					Blue:       color.Blue,
					Brightness: 0,
				}
				d.DeviceProfile.Stand.Row[rowId].Zones[keyIndex] = key
			}
			if d.activeRgb != nil {
				d.activeRgb.Exit <- true // Exit current RGB mode
				d.activeRgb = nil
			}
			d.setDeviceColor() // Restart RGB
			return 1
		}
	case 2:
		{
			for rowIndex, row := range d.DeviceProfile.Stand.Row {
				for keyIndex, key := range row.Zones {
					key.Color = rgb.Color{
						Red:        color.Red,
						Green:      color.Green,
						Blue:       color.Blue,
						Brightness: 0,
					}
					d.DeviceProfile.Stand.Row[rowIndex].Zones[keyIndex] = key
				}
			}
			if d.activeRgb != nil {
				d.activeRgb.Exit <- true // Exit current RGB mode
				d.activeRgb = nil
			}
			d.setDeviceColor() // Restart RGB
			return 1
		}
	}
	return 0
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
	var buf = make([]byte, colorPacketLength)
	//var buffer []byte
	for i := 0; i < d.LEDChannels; i++ {
		buf[i] = 0x00
	}
	d.writeColor(buf)

	if d.DeviceProfile == nil {
		logger.Log(logger.Fields{"serial": d.Serial}).Error("Unable to set color. DeviceProfile is null!")
		return
	}

	if d.DeviceProfile.RGBProfile == "mousepad" {
		for _, rows := range d.DeviceProfile.Stand.Row {
			for _, keys := range rows.Zones {
				if d.DeviceProfile.Brightness != 0 {
					keys.Color.Brightness = rgb.GetBrightnessValue(d.DeviceProfile.Brightness)
				}
				profileColor := rgb.ModifyBrightness(keys.Color)
				for _, packetIndex := range keys.PacketIndex {
					buf[packetIndex] = byte(profileColor.Red)
					buf[packetIndex+3] = byte(profileColor.Green)
					buf[packetIndex+6] = byte(profileColor.Blue)
				}
			}
		}
		d.writeColor(buf)
		return
	}

	if d.DeviceProfile.RGBProfile == "static" {
		profile := rgb.GetRgbProfile("static")
		if profile == nil {
			return
		}

		if d.DeviceProfile.Brightness != 0 {
			profile.StartColor.Brightness = rgb.GetBrightnessValue(d.DeviceProfile.Brightness)
		}

		profileColor := rgb.ModifyBrightness(profile.StartColor)
		for _, rows := range d.DeviceProfile.Stand.Row {
			for _, keys := range rows.Zones {
				for _, packetIndex := range keys.PacketIndex {
					buf[packetIndex] = byte(profileColor.Red)
					buf[packetIndex+3] = byte(profileColor.Green)
					buf[packetIndex+6] = byte(profileColor.Blue)
				}
			}
		}
		d.writeColor(buf)
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
				profile := rgb.GetRgbProfile(d.DeviceProfile.RGBProfile)
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

				for _, rows := range d.DeviceProfile.Stand.Row {
					for _, keys := range rows.Zones {
						for _, packetIndex := range keys.PacketIndex {
							switch packetIndex {
							case 0:
								buf[packetIndex] = buff[packetIndex]
								buf[packetIndex+3] = buff[packetIndex+1]
								buf[packetIndex+6] = buff[packetIndex+2]
							case 1:
								buf[packetIndex] = buff[packetIndex+2]
								buf[packetIndex+3] = buff[packetIndex+3]
								buf[packetIndex+6] = buff[packetIndex+4]
							case 2:
								buf[packetIndex] = buff[packetIndex+4]
								buf[packetIndex+3] = buff[packetIndex+5]
								buf[packetIndex+6] = buff[packetIndex+6]
							}
						}
					}
				}
				d.writeColor(buf)
				time.Sleep(20 * time.Millisecond)
				hue++
				wavePosition += 0.2
			}
		}
	}(d.LEDChannels)
}

// writeColor will write data to the device with a specific endpoint.
// writeColor does not require endpoint closing and opening like normal Write requires.
// Endpoint is open only once. Once the endpoint is open, color can be sent continuously.
func (d *Device) writeColor(data []byte) {
	// Buffer
	buffer := make([]byte, len(data)+len(data)+headerSize)
	buffer[0] = byte(len(data))
	copy(buffer[headerSizeWrite:], data)

	// Process buffer and create a chunked array if needed
	writeColorEp := cmdWriteColor
	colorEp := make([]byte, len(writeColorEp))
	copy(colorEp, writeColorEp)

	_, err := d.transfer(cmdWrite, cmdWriteColor, buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to color endpoint")
	}
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(command byte, endpoint, buffer []byte) ([]byte, error) {
	// Packet control, mandatory for this device
	mutex.Lock()
	defer mutex.Unlock()

	// Create write buffer
	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = command
	endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
	}

	// Send command to a device
	if _, err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return nil, err
	}

	bufferR := make([]byte, bufferSize)
	if _, err := d.dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		return nil, err
	}

	return bufferR, nil
}
