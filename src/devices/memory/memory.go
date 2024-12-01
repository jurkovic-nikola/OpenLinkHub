package memory

// Package: Memory
// This is the primary package for memory control.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/smbus"
	"OpenLinkHub/src/temperatures"
	"encoding/json"
	"fmt"
	"github.com/godbus/dbus/v5"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Devices struct {
	ChannelId          int     `json:"channelId"`
	DeviceId           int     `json:"deviceId"`
	Sku                string  `json:"sku"`
	Size               uint8   `json:"size"`
	MemoryType         int     `json:"memoryType"`
	Amount             int     `json:"amount"`
	Speed              int     `json:"speed"`
	Latency            int     `json:"latency"`
	LedChannels        uint8   `json:"ledChannels"`
	ColorRegister      uint8   `json:"colorRegister"`
	Name               string  `json:"name"`
	Temperature        float32 `json:"temperature"`
	TemperatureString  string  `json:"temperatureString"`
	Label              string  `json:"label"`
	RGB                string  `json:"rgb"`
	HasTemps           bool    `json:"-"`
	HasSpeed           bool
	ContainsPump       bool
	IsTemperatureProbe bool
}

// DeviceProfile struct contains all device profile
type DeviceProfile struct {
	Active      bool
	Path        string
	Product     string
	Serial      string
	Brightness  uint8
	RGBProfiles map[int]string
	Labels      map[int]string
}

type Device struct {
	Debug            bool
	Product          string                    `json:"product"`
	Serial           string                    `json:"serial"`
	AIO              bool                      `json:"aio"`
	UserProfiles     map[string]*DeviceProfile `json:"userProfiles"`
	Devices          map[int]*Devices          `json:"devices"`
	DeviceProfile    *DeviceProfile
	OriginalProfile  *DeviceProfile
	activeRgb        *rgb.ActiveRGB
	Template         string
	HasLCD           bool
	Brightness       map[int]string
	GlobalBrightness float64
	LEDChannels      int
	CpuTemp          float32
	GpuTemp          float32
	dev              *smbus.Connection
	Rgb              *rgb.RGB
}

// https://www.3dbrew.org/wiki/CRC-8-CCITT
var crcTable = [256]uint8{
	0x00, 0x07, 0x0E, 0x09, 0x1C, 0x1B, 0x12, 0x15, 0x38, 0x3F, 0x36, 0x31, 0x24, 0x23, 0x2A, 0x2D,
	0x70, 0x77, 0x7E, 0x79, 0x6C, 0x6B, 0x62, 0x65, 0x48, 0x4F, 0x46, 0x41, 0x54, 0x53, 0x5A, 0x5D,
	0xE0, 0xE7, 0xEE, 0xE9, 0xFC, 0xFB, 0xF2, 0xF5, 0xD8, 0xDF, 0xD6, 0xD1, 0xC4, 0xC3, 0xCA, 0xCD,
	0x90, 0x97, 0x9E, 0x99, 0x8C, 0x8B, 0x82, 0x85, 0xA8, 0xAF, 0xA6, 0xA1, 0xB4, 0xB3, 0xBA, 0xBD,
	0xC7, 0xC0, 0xC9, 0xCE, 0xDB, 0xDC, 0xD5, 0xD2, 0xFF, 0xF8, 0xF1, 0xF6, 0xE3, 0xE4, 0xED, 0xEA,
	0xB7, 0xB0, 0xB9, 0xBE, 0xAB, 0xAC, 0xA5, 0xA2, 0x8F, 0x88, 0x81, 0x86, 0x93, 0x94, 0x9D, 0x9A,
	0x27, 0x20, 0x29, 0x2E, 0x3B, 0x3C, 0x35, 0x32, 0x1F, 0x18, 0x11, 0x16, 0x03, 0x04, 0x0D, 0x0A,
	0x57, 0x50, 0x59, 0x5E, 0x4B, 0x4C, 0x45, 0x42, 0x6F, 0x68, 0x61, 0x66, 0x73, 0x74, 0x7D, 0x7A,
	0x89, 0x8E, 0x87, 0x80, 0x95, 0x92, 0x9B, 0x9C, 0xB1, 0xB6, 0xBF, 0xB8, 0xAD, 0xAA, 0xA3, 0xA4,
	0xF9, 0xFE, 0xF7, 0xF0, 0xE5, 0xE2, 0xEB, 0xEC, 0xC1, 0xC6, 0xCF, 0xC8, 0xDD, 0xDA, 0xD3, 0xD4,
	0x69, 0x6E, 0x67, 0x60, 0x75, 0x72, 0x7B, 0x7C, 0x51, 0x56, 0x5F, 0x58, 0x4D, 0x4A, 0x43, 0x44,
	0x19, 0x1E, 0x17, 0x10, 0x05, 0x02, 0x0B, 0x0C, 0x21, 0x26, 0x2F, 0x28, 0x3D, 0x3A, 0x33, 0x34,
	0x4E, 0x49, 0x40, 0x47, 0x52, 0x55, 0x5C, 0x5B, 0x76, 0x71, 0x78, 0x7F, 0x6A, 0x6D, 0x64, 0x63,
	0x3E, 0x39, 0x30, 0x37, 0x22, 0x25, 0x2C, 0x2B, 0x06, 0x01, 0x08, 0x0F, 0x1A, 0x1D, 0x14, 0x13,
	0xAE, 0xA9, 0xA0, 0xA7, 0xB2, 0xB5, 0xBC, 0xBB, 0x96, 0x91, 0x98, 0x9F, 0x8A, 0x8D, 0x84, 0x83,
	0xDE, 0xD9, 0xD0, 0xD7, 0xC2, 0xC5, 0xCC, 0xCB, 0xE6, 0xE1, 0xE8, 0xEF, 0xFA, 0xFD, 0xF4, 0xF3,
}

const (
	transferTypeColor       = 0
	transferTypeTemperature = 1
)

var (
	pwd                   = ""
	deviceWakeupDelay     = 5000
	timer                 = &time.Ticker{}
	mutex                 sync.Mutex
	authRefreshChan       = make(chan bool)
	deviceRefreshInterval = 1000
	cmdActivations        = []byte{0x36, 0x37} // SPA0 and SPA1
	maximumRegisters      = 8
	colorAddresses        = []byte{0x58, 0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f} // DDR4
	temperatureAddresses  = []byte{0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f} // DDR4
	dimmInfoAddresses     = []byte{0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57} // DDR4, DDR5
	temperatureRegister   = byte(0x05)
)

func Init(device, product string) *Device {
	if config.GetConfig().MemoryType == 5 {
		temperatureAddresses = dimmInfoAddresses                                // DDR5
		colorAddresses = []byte{0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f} // DDR5
		temperatureRegister = byte(0x31)                                        // DDR5 temperature register
	}

	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	dev, err := smbus.Open(device)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "device": device}).Error("Unable to open I2C device")
		return nil
	}

	serial := filepath.Base(device)
	serial = strings.Replace(serial, "-", "", -1)

	d := &Device{
		dev:      dev,
		Template: "memory.html",
		Brightness: map[int]string{
			0: "RGB Profile",
			1: "33 %",
			2: "66 %",
			3: "100 %",
		},
		Product:     product,
		Serial:      serial,
		LEDChannels: 0,
	}

	d.getDebugMode()       // Debug mode
	d.loadRgb()            // Load RGB
	d.loadDeviceProfiles() // Load all device profiles
	count := d.getDevices()
	if count == 0 {
		return nil // Nothing found
	}

	d.setAutoRefresh()    // Set auto device refresh
	d.saveDeviceProfile() // Save profile
	d.setDeviceColor()    // Device color
	if config.GetConfig().DbusMonitor {
		d.dbusDeviceMonitor()
	}
	return d
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	logger.Log(logger.Fields{"serial": d.Serial}).Info("Stopping device...")
	if d.activeRgb != nil {
		d.activeRgb.Stop()
	}
	authRefreshChan <- true
	d.disableLighting()
	err := d.dev.File.Close()
	if err != nil {
		return
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
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
}

// getDevices will get a list of DIMMs
func (d *Device) getDevices() int {
	var devices = make(map[int]*Devices, 0)
	activated := 0

	// DDR4
	skuRangeLow := byte(0x49)
	skuRangeHigh := byte(0x5b)
	if config.GetConfig().MemoryType == 5 {
		// DDR5
		skuRangeLow = byte(0x89)
		skuRangeHigh = byte(0x9b)
	}

	if d.Debug {
		logger.Log(logger.Fields{"skuRangeLow": skuRangeLow, "skuRangeHigh": skuRangeHigh}).Info("DEBUG skuRange")
	}
	for i := 0; i < maximumRegisters; i++ {
		if d.Debug {
			logger.Log(logger.Fields{"address": dimmInfoAddresses[i]}).Info("Probing address")
		}

		// Probe for register
		_, err := smbus.ReadRegister(d.dev.File, dimmInfoAddresses[i], 0x00)
		if err != nil {
			logger.Log(logger.Fields{"register": dimmInfoAddresses[i]}).Info("No such register found. Skipping...")
			continue
		}

		if d.Debug {
			logger.Log(logger.Fields{"memoryType": config.GetConfig().MemoryType}).Info("Probing address")
		}

		if config.GetConfig().MemoryType == 5 {
			if config.GetConfig().DecodeMemorySku {
				// DDR5 has no SPA0 and SPA1, it uses actual DIMM info addresses for different info
				// I2C Legacy Mode Device Configuration
				err = smbus.WriteRegister(d.dev.File, dimmInfoAddresses[i], 0x0b, 0x04)
				if err != nil {
					logger.Log(logger.Fields{"error": err, "address": dimmInfoAddresses[i]}).Error("Failed to activate DIMM info")
					continue
				}
			}
		} else {
			// We send 0x00 to 0x00 to SPA addresses
			for _, cmdActivation := range cmdActivations {
				err = smbus.WriteRegister(d.dev.File, cmdActivation, 0x00, 0x00)
				if err != nil {
					logger.Log(logger.Fields{"error": err}).Error("Failed to activate DIMM info")
					continue
				}
				activated++
			}
			if activated == 0 {
				continue
			}
		}
		var buf []byte

		if config.GetConfig().DecodeMemorySku {
			time.Sleep(1 * time.Millisecond)
			// Check SKU 1st letter, must match to C = Corsair
			check, err := smbus.ReadRegister(d.dev.File, dimmInfoAddresses[i], skuRangeLow)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "register": skuRangeLow}).Error("Failed to get first letter of SKU")
				continue
			}
			if string(check) != "C" {
				logger.Log(logger.Fields{"error": err, "register": skuRangeLow, "letter": string(check)}).Warn("First SKU letter does not match to letter C")
				continue
			}

			if d.Debug {
				logger.Log(logger.Fields{"skuLetter": string(check)}).Info("Memory SKU - First letter")
			}

			// Get SKU
			for addr := skuRangeLow; addr <= skuRangeHigh; addr++ {
				reg, err := smbus.ReadRegister(d.dev.File, dimmInfoAddresses[i], addr)
				if err != nil {
					break
				}
				if reg == 32 || reg == 0 {
					continue
				}
				buf = append(buf, reg)
			}

			if d.Debug {
				logger.Log(logger.Fields{"sku": buf, "skuString": string(buf), "skuLen": len(buf)}).Info("Memory SKU")
			}
		} else {
			// This is where memory SKU cannot be fetched
			memorySku := config.GetConfig().MemorySku
			if len(memorySku) < 1 {
				logger.Log(logger.Fields{}).Warn("decodeMemorySku set to false without memorySku value")
				continue
			}
			buf = []byte(memorySku)
		}

		if len(buf) > 15 {
			// https://help.corsair.com/hc/en-us/articles/8528259685901-RAM-How-to-Read-the-CORSAIR-memory-part-number
			// https://help.corsair.com/hc/en-us/articles/360051011331-RAM-DDR4-memory-module-dimensions
			dimmInfo := string(buf)

			if d.Debug {
				logger.Log(logger.Fields{"dimmInfo": dimmInfo}).Info("Memory DIMM Info")
			}

			skuLine := ""
			ledChannels := 0
			vendor := dimmInfo[0:2]
			colorRegister := 0

			if d.Debug {
				logger.Log(logger.Fields{"dimmInfoVendor": vendor}).Info("Memory DIMM Info - Vendor")
			}

			if vendor == "CM" { // Corsair Memory
				line := dimmInfo[2:3]
				size, e := strconv.Atoi(dimmInfo[3:5])
				if e != nil {
					continue
				}
				memoryType, e := strconv.Atoi(dimmInfo[7:8])
				if e != nil {
					continue
				}
				amount, e := strconv.Atoi(dimmInfo[9:10])
				if e != nil {
					continue
				}
				speed, e := strconv.Atoi(dimmInfo[11:15])
				if e != nil {
					continue
				}
				latency, e := strconv.Atoi(dimmInfo[16:18])
				if e != nil {
					continue
				}

				if d.Debug {
					logger.Log(logger.Fields{
						"dimmInfoLine": line,
						"memoryType":   memoryType,
						"amount":       amount,
						"speed":        speed,
						"latency":      latency,
					}).Info("Memory DIMM Info - Data")
				}

				if config.GetConfig().MemoryType == 4 {
					// DDR4
					switch line {
					case "U":
						skuLine = "VENGEANCE LED"
					case "W":
						skuLine = "VENGEANCE RGB PRO"
						ledChannels = 10
						colorRegister = 0x31
					case "H":
						skuLine = "VENGEANCE RGB PRO SL"
						ledChannels = 10
						colorRegister = 0x31
					case "N":
						skuLine = "VENGEANCE RGB RT"
						ledChannels = 10
						colorRegister = 0x31
					case "G":
						skuLine = "VENGEANCE RGB RS"
						ledChannels = 6
						colorRegister = 0x31
					case "D":
						skuLine = "DOMINATOR PLATINUM"
					case "T":
						skuLine = "DOMINATOR PLATINUM RGB"
						ledChannels = 12
						colorRegister = 0x31
					case "K":
						skuLine = "VENGEANCE LPX"
					case "P":
						skuLine = "DOMINATOR TITANIUM"
						//ledChannels = 11
					}
				} else {
					// DDR5
					switch line {
					case "K":
						skuLine = "VENGEANCE"
					case "H":
						skuLine = "VENGEANCE RGB"
						ledChannels = 10
						colorRegister = 0x31
					case "T":
						skuLine = "DOMINATOR PLATINUM RGB"
						ledChannels = 12
						colorRegister = 0x31
					case "P":
						skuLine = "DOMINATOR TITANIUM RGB"
						ledChannels = 11
						colorRegister = 0x31
					}
				}

				hasTemp := false
				temperature := 0.0
				temperatureString := ""

				// Temperature
				temp := d.transfer(nil, temperatureAddresses[i], 0, 0, transferTypeTemperature)
				if temp < 1 {
					// No sensor
				} else {
					temperature = d.calculateTemperature(temp)
					temperatureString = dashboard.GetDashboard().TemperatureToString(float32(temperature))
					hasTemp = true
				}

				label := "Set Label"
				if d.DeviceProfile != nil {
					// Device label
					if lb, ok := d.DeviceProfile.Labels[i]; ok {
						if len(lb) > 0 {
							label = lb
						}
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
						if d.GetRgbProfile(rp) != nil { // Speed profile exists in configuration
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

				if size > 0 && amount > 0 {
					device := &Devices{
						ChannelId:         i,
						DeviceId:          i,
						Sku:               dimmInfo,
						Size:              uint8(size / amount),
						MemoryType:        memoryType,
						Amount:            amount,
						Speed:             speed,
						Latency:           latency,
						LedChannels:       uint8(ledChannels),
						ColorRegister:     uint8(colorRegister),
						Name:              skuLine,
						Temperature:       float32(temperature),
						TemperatureString: temperatureString,
						Label:             label,
						RGB:               rgbProfile,
						HasTemps:          hasTemp,
					}
					if d.Debug {
						logger.Log(logger.Fields{"memoryDevice": device}).Info("Memory DIMM Info - Device")
					}
					devices[i] = device
					d.LEDChannels += ledChannels
				}
			}
		}
	}

	d.Devices = devices
	return len(devices)
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

// saveDeviceProfile will save device profile for persistent configuration
func (d *Device) saveDeviceProfile() {
	profilePath := pwd + "/database/profiles/" + d.Serial + ".json"

	rgbProfiles := make(map[int]string, len(d.Devices))
	labels := make(map[int]string, len(d.Devices))

	for _, device := range d.Devices {
		rgbProfiles[device.ChannelId] = device.RGB
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
			rgbProfiles[device.ChannelId] = "static"
			labels[device.ChannelId] = "Set Label"
		}
		deviceProfile.Active = true
		d.DeviceProfile = deviceProfile
	} else {
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
		logger.Log(logger.Fields{"error": err, "location": deviceProfile.Path}).Error("Unable to close file handle")
	}
	d.loadDeviceProfiles()
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
						if d.activeRgb != nil {
							d.activeRgb.Exit <- true
							d.activeRgb = nil
						}
						d.disableLighting()
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

// disableLighting will be called upon sleep. This is turn off lighting when sleeping
func (d *Device) disableLighting() {
	lightChannels := 0
	keys := make([]int, 0)
	for k := range d.Devices {
		lightChannels += int(d.Devices[k].LedChannels)
		if d.Devices[k].LedChannels > 0 {
			keys = append(keys, k)
		}
	}
	sort.Ints(keys)

	if lightChannels > 0 {
		for _, k := range keys {
			static := map[int][]byte{}
			for i := 0; i < int(d.Devices[k].LedChannels); i++ {
				static[i] = []byte{0, 0, 0}
			}
			buffer := rgb.SetColor(static)
			d.transfer(buffer, colorAddresses[k], d.Devices[k].LedChannels, d.Devices[k].ColorRegister, transferTypeColor)
		}
		return
	}
}

// setDeviceColor will activate and set device RGB
func (d *Device) setDeviceColor() {
	// Reset
	var buffer []byte
	lightChannels := 0

	keys := make([]int, 0)
	for k := range d.Devices {
		lightChannels += int(d.Devices[k].LedChannels)
		if d.Devices[k].LedChannels > 0 {
			keys = append(keys, k)
		}
	}
	sort.Ints(keys)

	// Do we have any RGB component in the system?
	if lightChannels == 0 {
		logger.Log(logger.Fields{}).Info("No RGB compatible devices found")
		return
	}

	// Are all devices under static mode?
	// In static mode, we only need to send color once;
	// there is no need for continuous packet sending.
	s, l := 0, 0
	for _, k := range keys {
		if d.Devices[k].LedChannels > 0 {
			l++ // device has LED
			if d.Devices[k].RGB == "static" {
				s++ // led profile is set to static
			}
		}
	}

	if s > 0 || l > 0 { // We have some values
		if s == l { // number of devices matches number of devices with static profile
			profile := d.GetRgbProfile("static")
			if d.DeviceProfile.Brightness != 0 {
				profile.StartColor.Brightness = rgb.GetBrightnessValue(d.DeviceProfile.Brightness)
			}

			// Global override
			if d.GlobalBrightness != 0 {
				profile.StartColor.Brightness = d.GlobalBrightness
			}

			profileColor := rgb.ModifyBrightness(profile.StartColor)
			for _, k := range keys {
				static := map[int][]byte{}
				for i := 0; i < int(d.Devices[k].LedChannels); i++ {
					static[i] = []byte{
						byte(profileColor.Red),
						byte(profileColor.Green),
						byte(profileColor.Blue),
					}
				}
				buffer = rgb.SetColor(static)
				d.transfer(buffer, colorAddresses[k], d.Devices[k].LedChannels, d.Devices[k].ColorRegister, transferTypeColor)
				time.Sleep(5 * time.Millisecond)
			}
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
		counterCpuTemp := map[int]int{}
		counterGpuTemp := map[int]int{}
		temperatureKeys := map[int]*rgb.Color{}
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
				for _, k := range keys {
					rgbCustomColor := true
					profile := d.GetRgbProfile(d.Devices[k].RGB)
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

					// Global override
					if d.GlobalBrightness != 0 {
						r.RGBBrightness = rgb.GetBrightnessValue(d.DeviceProfile.Brightness)
						r.RGBStartColor.Brightness = r.RGBBrightness
						r.RGBEndColor.Brightness = r.RGBBrightness
					}

					switch d.Devices[k].RGB {
					case "off":
						{
							off := map[int][]byte{}
							for i := 0; i < int(d.Devices[k].LedChannels); i++ {
								off[i] = []byte{0, 0, 0}
							}
							buff = rgb.SetColor(off)
						}
					case "rainbow":
						{
							r.Rainbow(startTime)
							buff = r.Output
						}
					case "watercolor":
						{
							r.Watercolor(startTime)
							buff = r.Output
						}
					case "cpu-temperature":
						{
							lock.Lock()
							counterCpuTemp[k]++
							if counterCpuTemp[k] >= r.Smoothness {
								counterCpuTemp[k] = 0
							}

							if _, ok := temperatureKeys[k]; !ok {
								temperatureKeys[k] = r.RGBStartColor
							}

							r.MinTemp = profile.MinTemp
							r.MaxTemp = profile.MaxTemp
							res := r.Temperature(float64(d.CpuTemp), counterCpuTemp[k], temperatureKeys[k])
							temperatureKeys[k] = res
							lock.Unlock()
							buff = r.Output
						}
					case "gpu-temperature":
						{
							lock.Lock()
							counterGpuTemp[k]++
							if counterGpuTemp[k] >= r.Smoothness {
								counterGpuTemp[k] = 0
							}

							if _, ok := temperatureKeys[k]; !ok {
								temperatureKeys[k] = r.RGBStartColor
							}

							r.MinTemp = profile.MinTemp
							r.MaxTemp = profile.MaxTemp
							res := r.Temperature(float64(d.GpuTemp), counterGpuTemp[k], temperatureKeys[k])
							temperatureKeys[k] = res
							lock.Unlock()
							buff = r.Output
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
							buff = r.Output
						}
					case "static":
						{
							r.Static()
							buff = r.Output
						}
					case "rotator":
						{
							r.Rotator(hue)
							buff = r.Output
						}
					case "wave":
						{
							r.Wave(wavePosition)
							buff = r.Output
						}
					case "storm":
						{
							r.Storm()
							buff = r.Output
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
							buff = r.Output
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
							buff = r.Output
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
							buff = r.Output
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
							buff = r.Output
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
							buff = r.Output
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
							buff = r.Output
						}
					}
					// Send it
					d.transfer(buff, colorAddresses[k], d.Devices[k].LedChannels, d.Devices[k].ColorRegister, transferTypeColor)
					time.Sleep(15 * time.Millisecond)
				}
				hue++
				wavePosition += 0.2
			}
		}
	}(lightChannels)
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

// UpdateRgbProfile will update device RGB profile
func (d *Device) UpdateRgbProfile(channelId int, profile string) uint8 {
	if d.GetRgbProfile(profile) == nil {
		logger.Log(logger.Fields{"serial": d.Serial, "profile": profile}).Warn("Non-existing RGB profile")
		return 0
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

	d.saveDeviceProfile() // Save profile
	if d.activeRgb != nil {
		d.activeRgb.Exit <- true // Exit current RGB mode
		d.activeRgb = nil
	}
	d.setDeviceColor() // Restart RGB
	return 1
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

// calculateTemperature will calculate temperature to readable value
func (d *Device) calculateTemperature(temp uint16) float64 {
	temperature := 0.0
	if config.GetConfig().MemoryType == 4 {
		// DDR4
		res := ((temp & 0xff) << 8) | (temp >> 8)
		res = res & 0x1fff
		if res > 0x0fff {
			res -= 0x2000
		}
		scale, bits := 0.25, 10.0
		multiplier := res >> (12 - int(bits))
		temperature = scale * float64(multiplier)
	} else {
		// DDR5 SPD5118 standard
		res := (temp >> 8) | temp
		shift := 21
		val := (int32(res>>2) << shift) >> shift
		val = val * 256
		temperature = math.Round((float64(val)/1000)*100) / 100
	}
	return temperature
}

// setTemperatures will fetch temperature values
func (d *Device) setTemperatures() {
	d.CpuTemp = temperatures.GetCpuTemperature()
	d.GpuTemp = temperatures.GetGpuTemperature()
	for _, device := range d.Devices {
		if device.HasTemps {
			// Temperature
			temp := d.transfer(
				nil,
				temperatureAddresses[device.ChannelId],
				0,
				0, transferTypeTemperature,
			)
			if temp < 1 {
				// No sensor
			} else {
				temperature := d.calculateTemperature(temp)
				temperatureString := dashboard.GetDashboard().TemperatureToString(float32(temperature))
				d.Devices[device.ChannelId].Temperature = float32(temperature)
				d.Devices[device.ChannelId].TemperatureString = temperatureString
			}
		}
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
				d.setTemperatures()
			case <-authRefreshChan:
				timer.Stop()
				return
			}
		}
	}()
}

// calculateChecksum will calculate CRC checksum
func (d *Device) calculateChecksum(data []byte) byte {
	var val uint8 = 0
	for _, b := range data {
		val = crcTable[val^b]
	}
	return val
}

// transfer will transfer data to a i2c device
func (d *Device) transfer(buffer []byte, address, ledDevices byte, colorRegister uint8, transferType int) uint16 {
	mutex.Lock()
	defer mutex.Unlock()

	switch transferType {
	case transferTypeColor:
		{
			// RGB
			var buf []byte
			buf = append(buf, ledDevices)
			for i := 0; i < len(buffer); i++ {
				buf = append(buf, buffer[i])
			}
			buf = append(buf, d.calculateChecksum(buf))
			if d.Debug {
				logger.Log(logger.Fields{"colorPacket": fmt.Sprint("% 2x", buf)}).Info("Memory Color")
			}
			if len(buf) > 32 {
				// We have more than 10 LEDs, we need to chunk packet and increment color register.
				// This is relevant for DOMINATOR PLATINUM RGB that has 12 LEDs.
				chunks := common.ProcessMultiChunkPacket(buf, 32)
				for _, chunk := range chunks {
					if d.Debug {
						logger.Log(logger.Fields{"colorPacket": fmt.Sprint("% 2x", chunk)}).Info("Memory Color - Chunk")
					}
					err := smbus.WriteBlockData(d.dev.File, address, colorRegister, chunk)
					if err != nil {
						logger.Log(logger.Fields{"error": err, "address": address}).Error("Unable to write to i2c register")
					}
					colorRegister += 1
					time.Sleep(1 * time.Millisecond)
				}
			} else {
				err := smbus.WriteBlockData(d.dev.File, address, colorRegister, buf)
				if err != nil {
					logger.Log(logger.Fields{"error": err, "address": address}).Error("Unable to write to i2c register")
				}
			}
		}
	case transferTypeTemperature:
		{
			// Temperature
			temp, err := smbus.ReadWord(d.dev.File, address, temperatureRegister)
			if err == nil {
				return temp
			}
		}
	}
	return 0
}
