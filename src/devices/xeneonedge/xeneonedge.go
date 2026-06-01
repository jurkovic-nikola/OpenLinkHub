package xeneonedge

// Package: CORSAIR XENEON EDGE
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"encoding/json"
	"fmt"
	"github.com/sstallion/go-hid"
	"os"
	"strings"
)

// DeviceProfile struct contains all device profile
type DeviceProfile struct {
	Active      bool
	Path        string
	Product     string
	Serial      string
	WidgetAreas map[int]WidgetArea
	RgbOff      bool
	Widgets     []Widget `json:"widgets"`
}

type WidgetArea struct {
	WidgetId int     `json:"widgetId"`
	Widget   *Widget `json:"widget"`
}
type Device struct {
	dev             *hid.Device
	Debug           bool
	Manufacturer    string                    `json:"manufacturer"`
	Product         string                    `json:"product"`
	Serial          string                    `json:"serial"`
	Firmware        string                    `json:"firmware"`
	UserProfiles    map[string]*DeviceProfile `json:"userProfiles"`
	Devices         map[int]string            `json:"devices"`
	Widgets         []Widget                  `json:"widgets"`
	DeviceProfile   *DeviceProfile
	OriginalProfile *DeviceProfile
	Template        string
	VendorId        uint16
	ProductId       uint16
	instance        *common.Device
}

type Widget struct {
	Id          int     `json:"id"`
	Name        string  `json:"name"`
	Template    string  `json:"template"`
	Columns     []int   `json:"columns"`
	City        string  `json:"city"`
	Country     string  `json:"country"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Source      string  `json:"source"`
	AutoWeather bool    `json:"autoWeather"`
	DataColor   string  `json:"dataColor"`
	Max         int     `json:"max"`
	HeaderText  string  `json:"headerText"`
	Unit        string  `json:"unit"`
	TextColor   string  `json:"textColor"`
}

var (
	pwd = ""
)

func Init(vendorId, productId uint16, _, path string) *common.Device {
	// Set global working directory
	pwd = config.GetConfig().ConfigPath

	dev, err := hid.OpenPath(path)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId, "path": path}).Error("Unable to open HID device")
		return nil
	}

	// Init new struct with HID device
	d := &Device{
		dev:       dev,
		Template:  "xeneonedge.html",
		VendorId:  vendorId,
		ProductId: productId,
		Firmware:  "n/a",
		Product:   "XENEON EDGE",
	}

	d.getDebugMode()       // Debug mode
	d.getManufacturer()    // Manufacturer
	d.getSerial()          // Serial
	d.loadWidgets()        // Widgets
	d.loadDeviceProfiles() // Load all device profiles
	d.saveDeviceProfile()  // Save profile
	d.createDevice()       // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")
	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeXeneonEdge,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-lcd.svg",
		Instance:    d,
	}
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			return
		}
	}
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
}

// StopDirty will stop device in a dirty way
func (d *Device) StopDirty() uint8 {
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 1
}

// getManufacturer will return device manufacturer
func (d *Device) getManufacturer() {
	manufacturer, err := d.dev.GetMfrStr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to get manufacturer")
	}
	d.Manufacturer = manufacturer
}

// getSerial will return device serial number
func (d *Device) getSerial() {
	serial, err := d.dev.GetSerialNbr()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to get device serial number")
	}
	d.Serial = serial
}

// loadWidgets will load xeneon widgets
func (d *Device) loadWidgets() {
	location := pwd + "/database/xeneon/xeneon.json"

	file, fe := os.Open(location)
	if fe != nil {
		logger.Log(logger.Fields{"error": fe, "location": location}).Warn("Unable to open widgets file")
		return
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			//
		}
	}(file)

	reader := json.NewDecoder(file)
	if err := reader.Decode(&d); err != nil {
		fmt.Println(err)
		logger.Log(logger.Fields{"error": err, "location": location}).Warn("Unable to decode widgets file")
		return
	}
}

// GetDeviceTemplate will return device template name
func (d *Device) GetDeviceTemplate() string {
	return d.Template
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

// getManufacturer will return device manufacturer
func (d *Device) getDebugMode() {
	d.Debug = config.GetConfig().Debug
}

func (d *Device) getWidget(widgetId int) *Widget {
	for _, widget := range d.Widgets {
		if widget.Id == widgetId {
			return &widget
		}
	}
	return nil
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
		deviceProfile.Active = true
		deviceProfile.WidgetAreas = map[int]WidgetArea{
			1: {
				WidgetId: 1,
				Widget:   d.getWidget(1),
			},
			2: {
				WidgetId: 2,
				Widget:   d.getWidget(2),
			},
			3: {
				WidgetId: 6,
				Widget:   d.getWidget(6),
			},
			4: {
				WidgetId: 7,
				Widget:   d.getWidget(7),
			},
			5: {
				WidgetId: 8,
				Widget:   d.getWidget(8),
			},
			6: {
				WidgetId: 9,
				Widget:   d.getWidget(9),
			},
			7: {
				WidgetId: 0,
			},
			8: {
				WidgetId: 0,
			},
			9: {
				WidgetId: 3,
				Widget:   d.getWidget(3),
			},
			10: {
				WidgetId: 5,
				Widget:   d.getWidget(5),
			},
			11: {
				WidgetId: 4,
				Widget:   d.getWidget(4),
			},
		}
		deviceProfile.Widgets = d.Widgets
	} else {
		deviceProfile.Active = d.DeviceProfile.Active
		if len(d.DeviceProfile.Path) < 1 {
			deviceProfile.Path = profilePath
			d.DeviceProfile.Path = profilePath
		} else {
			deviceProfile.Path = d.DeviceProfile.Path
		}
		deviceProfile.WidgetAreas = d.DeviceProfile.WidgetAreas
		deviceProfile.Widgets = d.DeviceProfile.Widgets
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

// loadDeviceProfiles will load custom user profiles
func (d *Device) loadDeviceProfiles() {
	profileList := make(map[string]*DeviceProfile)
	userProfileDirectory := pwd + "/database/profiles/"

	files, err := os.ReadDir(userProfileDirectory)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": userProfileDirectory, "serial": d.Serial}).Error("Unable to read content of a folder")
		return
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
