package nautilusLcd

// Package: CORSAIR Nautilus LCD Cap
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices/lcd"
	"OpenLinkHub/src/led"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/systeminfo"
	"OpenLinkHub/src/temperatures"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"os"
	"path/filepath"
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
	LCDImage    string
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
	ledProfile        *led.Device
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
	rgbMutex          sync.RWMutex
	LCDImage          *lcd.ImageData
	Exit              bool
	mutex             sync.Mutex
	lcdRefreshChan    chan struct{}
	lcdImageChan      chan struct{}
	timer             *time.Ticker
	lcdTimer          *time.Ticker
	RGBModes          []string
	instance          *common.Device
}

var (
	pwd                        = ""
	lcdRefreshInterval         = 1000
	lcdHeaderSize              = 8
	lcdBufferSize              = 1024
	firmwareReportId           = byte(5)
	featureReportSize          = 32
	maxLCDBufferSizePerRequest = lcdBufferSize - lcdHeaderSize
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
		dev:       dev,
		Template:  "nautilusLcd.html",
		VendorId:  vendorId,
		ProductId: productId,
		LCDModes: map[int]string{
			2:   "CPU Temperature",
			3:   "GPU Temperature",
			6:   "CPU / GPU Temp",
			7:   "CPU / GPU Load",
			8:   "CPU / GPU Load/Temp",
			9:   "Time",
			10:  "Image / GIF",
			100: "Arc",
			101: "Double Arc",
			102: "Animation",
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
		lcdRefreshChan: make(chan struct{}),
		lcdImageChan:   make(chan struct{}),
		timer:          &time.Ticker{},
		lcdTimer:       &time.Ticker{},
		HasLCD:         true,
	}

	// Bootstrap
	d.getManufacturer()    // Manufacturer
	d.getProduct()         // Product
	d.getSerial()          // Serial
	d.getDeviceFirmware()  // Firmware
	d.loadDeviceProfiles() // Load all device profiles
	d.saveDeviceProfile()  // Save profile
	d.setLcdRotation()     // LCD rotation
	if d.DeviceProfile.LCDMode == lcd.DisplayImage {
		if d.loadLcdImage() != 1 {
			logger.Log(logger.Fields{"serial": d.Serial}).Warn("Unable to load LCD image from profile")
		} else {
			d.setupLCDImage()
		}
	} else {
		d.setupLCD(false)
	}
	d.createDevice() // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeNautilusLcdCap,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-device.svg",
		Instance:    d,
		GetDevice:   d,
	}
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	d.Exit = true

	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device...")
	d.timer.Stop()
	d.lcdTimer.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.DeviceProfile.LCDMode == lcd.DisplayImage {
				if d.lcdImageChan != nil {
					close(d.lcdImageChan)
				}
			} else {
				if d.lcdRefreshChan != nil {
					close(d.lcdRefreshChan)
				}
			}
		})
	}()

	if d.dev != nil {
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
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
}

// StopDirty will stop device in a dirty way
func (d *Device) StopDirty() uint8 {
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device (dirty)...")

	d.timer.Stop()
	d.lcdTimer.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.DeviceProfile.LCDMode == lcd.DisplayImage {
				if d.lcdImageChan != nil {
					close(d.lcdImageChan)
				}
			} else {
				if d.lcdRefreshChan != nil {
					close(d.lcdRefreshChan)
				}
			}
		})
	}()
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 2
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
	product = strings.ToUpper(product)
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

	if d.DeviceProfile == nil {
		deviceProfile.Label = "LCD Cap"
		if d.HasLCD {
			deviceProfile.LCDMode = 2
			deviceProfile.LCDRotation = 0
		}
		deviceProfile.Active = true
		deviceProfile.LCDImage = ""
		d.DeviceProfile = deviceProfile
	} else {
		deviceProfile.Active = d.DeviceProfile.Active
		deviceProfile.Label = d.DeviceProfile.Label
		deviceProfile.LCDImage = d.DeviceProfile.LCDImage
		if len(d.DeviceProfile.Path) < 1 {
			deviceProfile.Path = profilePath
			d.DeviceProfile.Path = profilePath
		} else {
			deviceProfile.Path = d.DeviceProfile.Path
		}
		deviceProfile.LCDMode = d.DeviceProfile.LCDMode
		deviceProfile.LCDRotation = d.DeviceProfile.LCDRotation
	}

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

	v1, v2, v3, v4 := string(buffer[6]), string(buffer[8]), string(buffer[10]), string(buffer[12])
	d.Firmware = fmt.Sprintf("%s.%s.%s.%s", v1, v2, v3, v4)
}

// ChangeDeviceProfile will change device profile
func (d *Device) ChangeDeviceProfile(profileName string) uint8 {
	if profile, ok := d.UserProfiles[profileName]; ok {
		currentProfile := d.DeviceProfile
		currentProfile.Active = false
		d.DeviceProfile = currentProfile
		d.saveDeviceProfile()

		newProfile := profile
		newProfile.Active = true
		d.DeviceProfile = newProfile
		d.saveDeviceProfile()
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

// UpdateDeviceLcd will update device LCD
func (d *Device) UpdateDeviceLcd(_ int, mode uint8) uint8 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.HasLCD {
		value := d.DeviceProfile.LCDMode
		if mode == lcd.DisplayImage {
			if len(lcd.GetLcdImages()) == 0 {
				return 0
			}

			if len(d.DeviceProfile.LCDImage) == 0 {
				lcdImage := lcd.GetLcdImages()[0]
				d.DeviceProfile.LCDImage = lcdImage.Name
				d.LCDImage = &lcdImage
			}

			if lcd.GetLcdImage(d.DeviceProfile.LCDImage) == nil {
				lcdImage := lcd.GetLcdImages()[0]
				d.DeviceProfile.LCDImage = lcdImage.Name
				d.LCDImage = &lcdImage
			}

			// Stop lcd timer and switch to animation loop
			if d.lcdRefreshChan != nil {
				close(d.lcdRefreshChan)
			}
			d.lcdTimer.Stop()
			d.setupLCDImage()
		} else {
			// Reset if old value was Animation and new mode is not
			if value == lcd.DisplayImage && value != mode {
				d.setupLCD(true)
			}
		}

		d.DeviceProfile.LCDMode = mode
		d.saveDeviceProfile()
		return 1
	} else {
		return 2
	}
}

// UpdateDeviceLcdRotation will update device LCD rotation
func (d *Device) UpdateDeviceLcdRotation(_ int, rotation uint8) uint8 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.HasLCD {
		d.DeviceProfile.LCDRotation = rotation
		d.saveDeviceProfile()
		d.setLcdRotation()
		return 1
	} else {
		return 2
	}
}

// UpdateDeviceLcdImage will update device LCD image
func (d *Device) UpdateDeviceLcdImage(_ int, image string) uint8 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.HasLCD {
		if d.DeviceProfile.LCDMode != lcd.DisplayImage {
			return 0
		}

		if len(lcd.GetLcdImages()) == 0 {
			return 0
		}

		if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", d.DeviceProfile.LCDImage); !m {
			return 0
		}

		lcdImage := lcd.GetLcdImage(image)
		if lcdImage == nil {
			return 0
		}

		// Save and restart LCD
		d.DeviceProfile.LCDImage = image
		d.LCDImage = lcdImage
		d.saveDeviceProfile()
		return 1
	} else {
		return 0
	}
}

// UpdateDeviceLabel will set / update device label
func (d *Device) UpdateDeviceLabel(_ int, label string) uint8 {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.DeviceProfile.Label = label
	d.saveDeviceProfile()
	return 1
}

// saveRgbProfile will save rgb profile data
func (d *Device) saveRgbProfile() {
	rgbDirectory := pwd + "/database/rgb/"
	rgbFilename := rgbDirectory + d.Serial + ".json"
	if common.FileExists(rgbFilename) {
		if err := common.SaveJsonData(rgbFilename, d.Rgb); err != nil {
			logger.Log(logger.Fields{"error": err, "location": rgbFilename}).Error("Unable to write rgb profile data")
			return
		}
	}
}

// setLcdRotation will change LCD rotation
func (d *Device) setLcdRotation() {
	lcdReport := []byte{0x03, 0x0c, d.DeviceProfile.LCDRotation, 0x01}
	_, err := d.dev.SendFeatureReport(lcdReport)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId, "productId": d.ProductId, "serial": d.Serial}).Error("Unable to change LCD rotation")
	}
}

// setupLCD will activate and configure LCD
func (d *Device) setupLCD(reload bool) {
	if reload {
		close(d.lcdImageChan)
	}
	d.lcdTimer = time.NewTicker(time.Duration(lcdRefreshInterval) * time.Millisecond)
	d.lcdRefreshChan = make(chan struct{})
	go func() {
		for {
			select {
			case <-d.lcdTimer.C:
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
						)
						d.transfer(buffer)
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
						)
						d.transfer(buffer)
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
						)
						d.transfer(buffer)
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
						)
						d.transfer(buffer)
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
						)
						d.transfer(buffer)
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
						)
						d.transfer(buffer)
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
						)
						d.transfer(buffer)
					}
				case lcd.DisplayTime:
					{
						buffer := lcd.GenerateScreenImage(
							lcd.DisplayTime,
							0,
							0,
							0,
							0,
						)
						d.transfer(buffer)
					}
				case lcd.DisplayArc:
					{
						val := 0
						arcType := 0
						sensor := 0
						switch lcd.GetArc().Sensor {
						case 0: // CPU temperature
							val = int(temperatures.GetCpuTemperature())
							break
						case 1: // GPU temperature
							val = int(temperatures.GetGpuTemperature())
							arcType = 1
							break
						case 2: // Liquid temperature
							val = 0
							arcType = 2
							sensor = 2
							break
						case 3: // CPU utilization
							val = int(systeminfo.GetCpuUtilization())
							sensor = 3
							break
						case 4: // GPU utilization
							val = systeminfo.GetGPUUtilization()
							sensor = 4
						}
						image := lcd.GenerateArcScreenImage(arcType, sensor, val)
						if image == nil {
							break // Fail
						}
						d.transfer(image)
					}
				case lcd.DisplayDoubleArc:
					{
						values := []float32{
							temperatures.GetCpuTemperature(),
							temperatures.GetGpuTemperature(),
							0,
							float32(systeminfo.GetCpuUtilization()),
							float32(systeminfo.GetGPUUtilization()),
						}
						image := lcd.GenerateDoubleArcScreenImage(values)
						if image != nil {
							d.transfer(image)
						}
					}
				case lcd.DisplayAnimation:
					{
						values := []float32{
							temperatures.GetCpuTemperature(),
							temperatures.GetGpuTemperature(),
							0,
							float32(systeminfo.GetCpuUtilization()),
							float32(systeminfo.GetGPUUtilization()),
						}
						image := lcd.GenerateAnimationScreenImage(values)
						if image != nil {
							imageLen := len(image)
							for i := 0; i < imageLen; i++ {
								d.transfer(image[i].Buffer)
								if i != imageLen-1 {
									if image[i].Delay > 0 {
										time.Sleep(time.Duration(image[i].Delay) * time.Millisecond)
									}
								}
							}
						}
					}
				}
			case <-d.lcdRefreshChan:
				d.lcdTimer.Stop()
				return
			}
		}
	}()
}

// loadLcdImage will load current LCD image
func (d *Device) loadLcdImage() uint8 {
	if len(d.DeviceProfile.LCDImage) == 0 {
		return 0
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", d.DeviceProfile.LCDImage); !m {
		return 0
	}

	lcdImage := lcd.GetLcdImage(d.DeviceProfile.LCDImage)
	if lcdImage == nil {
		d.DeviceProfile.LCDMode = 0
		d.saveDeviceProfile()
		d.setupLCD(false)
		return 0
	}
	d.LCDImage = lcdImage
	return 1
}

// setupLCDImage will set up lcd image
func (d *Device) setupLCDImage() {
	d.lcdImageChan = make(chan struct{})
	lcdImage := d.DeviceProfile.LCDImage
	if len(lcdImage) == 0 {
		logger.Log(logger.Fields{"serial": d.Serial}).Warn("Invalid LCD image")
		return
	}

	if d.LCDImage == nil {
		d.loadLcdImage()
	}

	go func() {
		for {
			select {
			default:
				if d.LCDImage.Frames > 1 {
					for i := 0; i < d.LCDImage.Frames; i++ {
						if d.Exit {
							return
						}
						data := d.LCDImage.Buffer[i]
						buffer := data.Buffer
						delay := data.Delay

						d.transfer(buffer)
						if delay > 0 {
							time.Sleep(time.Duration(delay) * time.Millisecond)
						} else {
							// Single frame, static image, generate 100ms of delay
							time.Sleep(100 * time.Millisecond)
						}
					}
				} else {
					data := d.LCDImage.Buffer[0]
					buffer := data.Buffer
					delay := data.Delay
					d.transfer(buffer)
					if delay > 0 {
						time.Sleep(time.Duration(delay) * time.Millisecond)
					} else {
						// Single frame, static image, generate 100ms of delay
						time.Sleep(100 * time.Millisecond)
					}
				}
			case <-d.lcdImageChan:
				return
			}
		}
	}()
}

// transfer will transfer data to LCD panel
func (d *Device) transfer(buffer []byte) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	chunks := common.ProcessMultiChunkPacket(buffer, maxLCDBufferSizePerRequest)
	for i, chunk := range chunks {
		if d.Exit {
			break
		}
		bufferW := make([]byte, lcdBufferSize)
		bufferW[0] = 0x02
		bufferW[1] = 0x05

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
