package xeneonedge

// Package: CORSAIR XENEON EDGE
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"encoding/json"
	"github.com/sstallion/go-hid"
	"math"
	"os"
	"regexp"
	"strings"
	"sync"
)

// DeviceProfile struct contains all device profile
type DeviceProfile struct {
	Active         bool
	Path           string
	Product        string
	Serial         string
	WidgetAreas    map[int]WidgetArea
	RgbOff         bool
	Widgets        []Widget `json:"widgets"`
	Brightness     int
	Backlight      int
	Contrast       int
	ColorToneRed   int
	ColorToneGreen int
	ColorToneBlue  int
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
	mutex           sync.Mutex
	Template        string
	VendorId        uint16
	ProductId       uint16
	instance        *common.Device
	Exit            bool
}

type Widget struct {
	Id              int     `json:"id"`
	Name            string  `json:"name"`
	Template        string  `json:"template"`
	Columns         []int   `json:"columns"`
	City            string  `json:"city"`
	Country         string  `json:"country"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	Source          string  `json:"source"`
	AutoWeather     bool    `json:"autoWeather"`
	DataColor       string  `json:"dataColor"`
	Max             int     `json:"max"`
	HeaderText      string  `json:"headerText"`
	Unit            string  `json:"unit"`
	TextColor       string  `json:"textColor"`
	FontSize        int     `json:"fontSize"`
	SubTextColor    string  `json:"subTextColor"`
	SubFontSize     int     `json:"subFontSize"`
	Left            int     `json:"left"`
	Top             int     `json:"top"`
	SubLeft         int     `json:"subLeft"`
	SubTop          int     `json:"subTop"`
	HideForecast    bool    `json:"hideForecast"`
	RefreshInterval int     `json:"refreshInterval"`
	HideArtwork     bool    `json:"hideArtwork"`
	HideControls    bool    `json:"hideControls"`
	BatteryDevice   string  `json:"batteryDevice"`
	DeviceId        string  `json:"deviceId"`
	ChannelId       int     `json:"channelId"`
	WeekStart       int     `json:"weekStart"`
}

var (
	pwd             = ""
	bufferSize      = 64
	bufferSizeWrite = bufferSize + 1
	headerSize      = 7
	cmdBacklight    = []byte{0x02, 0x00}
	cmdContrast     = []byte{0x02, 0x01}
	cmdBrightness   = []byte{0x02, 0x02}
	cmdColorTones   = map[int][]byte{
		0: {0x03, 0x01},
		1: {0x03, 0x02},
		2: {0x03, 0x03},
	}
	dataTypeRed        = byte(150)
	dataTypeGreen      = byte(126)
	dataTypeBlue       = byte(140)
	dataTypeColorTones = map[int]byte{
		0: dataTypeRed,
		1: dataTypeGreen,
		2: dataTypeBlue,
	}
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

	d.getDebugMode()        // Debug mode
	d.getManufacturer()     // Manufacturer
	d.getSerial()           // Serial
	d.loadWidgets()         // Widgets
	d.loadDeviceProfiles()  // Load all device profiles
	d.saveDeviceProfile()   // Save profile
	d.setDeviceBrightness() // Brightness
	d.setDeviceContrast()   // Contrast
	d.setDeviceBacklight()  // Backlight
	d.setDeviceColortone()  // Color tone
	d.createDevice()        // Device register
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
	d.Exit = true
	d.setScreenToDefault()

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
	if d.dev != nil {
		err := d.dev.Close()
		if err != nil {
			return 2
		}
	}

	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 2
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
		d.setDeviceBacklight()
		d.setDeviceContrast()
		d.setDeviceBrightness()
		d.setDeviceColortone()
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

// ChangeDeviceBrightnessValue will change device brightness via slider
func (d *Device) ChangeDeviceBrightnessValue(value uint8) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if value < 0 || value > 100 {
		return 0
	}

	d.DeviceProfile.Brightness = int(value)
	d.saveDeviceProfile()
	d.setDeviceBrightness()
	return 1
}

// ChangeDeviceContrastValue will change device contrast via slider
func (d *Device) ChangeDeviceContrastValue(value uint8) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if value < 0 || value > 100 {
		return 0
	}

	d.DeviceProfile.Contrast = int(value)
	d.saveDeviceProfile()
	d.setDeviceContrast()
	return 1
}

// ChangeDeviceBacklightValue will change device backlight via slider
func (d *Device) ChangeDeviceBacklightValue(value uint8) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	d.DeviceProfile.Backlight = int(value)
	d.saveDeviceProfile()
	d.setDeviceBacklight()
	return 1
}

// ChangeDeviceColorToneValue will change device color tone via slider
func (d *Device) ChangeDeviceColorToneValue(value, valueType int) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if value < 0 || value > 255 {
		return 0
	}

	switch valueType {
	case 0:
		d.DeviceProfile.ColorToneRed = value
		break
	case 1:
		d.DeviceProfile.ColorToneGreen = value
		break
	case 2:
		d.DeviceProfile.ColorToneBlue = value
		break
	default:
		return 0
	}
	d.saveDeviceProfile()
	d.setDeviceColortone()
	return 1
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

// ProcessGetWidget will return widget data
func (d *Device) ProcessGetWidget(widgetId int) interface{} {
	if d.DeviceProfile == nil || d.getWidget(widgetId) == nil {
		return nil
	}

	for _, area := range d.DeviceProfile.WidgetAreas {
		if area.Widget != nil && area.Widget.Id == widgetId {
			widget := *area.Widget
			return &widget
		}
	}

	for _, widget := range d.DeviceProfile.Widgets {
		if widget.Id == widgetId {
			return &widget
		}
	}

	return d.getWidget(widgetId)
}

// ProcessSaveWidget will save widget configuration
func (d *Device) ProcessSaveWidget(value Widget) uint8 {
	current, ok := d.ProcessGetWidget(value.Id).(*Widget)
	if !ok || current == nil {
		return 0
	}

	widget := *current
	colorPattern := regexp.MustCompile("^#[0-9a-fA-F]{6}$")
	if !colorPattern.MatchString(value.TextColor) || !colorPattern.MatchString(value.SubTextColor) ||
		value.FontSize < 8 || value.FontSize > 200 || value.SubFontSize < 8 || value.SubFontSize > 100 {
		return 0
	}

	widget.TextColor = value.TextColor
	widget.SubTextColor = value.SubTextColor
	widget.FontSize = value.FontSize
	widget.SubFontSize = value.SubFontSize

	switch current.Template {
	case "xeneon-clock":
		{
			if value.Left < 0 || value.Left > 1000 || value.Top < 0 || value.Top > 500 ||
				value.SubLeft < 0 || value.SubLeft > 1000 || value.SubTop < 0 || value.SubTop > 500 {
				return 0
			}

			widget.Left, widget.Top = value.Left, value.Top
			widget.SubLeft, widget.SubTop = value.SubLeft, value.SubTop
		}
		break
	case "xeneon-weather":
		{
			if len(strings.TrimSpace(value.City)) == 0 || len(value.City) > 100 || len(value.Country) > 100 ||
				math.IsNaN(value.Latitude) || math.IsInf(value.Latitude, 0) || value.Latitude < -90 || value.Latitude > 90 ||
				math.IsNaN(value.Longitude) || math.IsInf(value.Longitude, 0) || value.Longitude < -180 || value.Longitude > 180 ||
				value.RefreshInterval < 5 || value.RefreshInterval > 120 || (value.Unit != "°C" && value.Unit != "°F") {
				return 0
			}

			widget.City, widget.Country = strings.TrimSpace(value.City), strings.TrimSpace(value.Country)
			widget.Latitude, widget.Longitude = value.Latitude, value.Longitude
			widget.AutoWeather, widget.HideForecast = value.AutoWeather, value.HideForecast
			widget.RefreshInterval, widget.Unit = value.RefreshInterval, value.Unit
		}
		break
	case "xeneon-media-player":
		{
			if !colorPattern.MatchString(value.DataColor) {
				return 0
			}

			widget.DataColor = value.DataColor
			widget.HideArtwork, widget.HideControls = value.HideArtwork, value.HideControls
		}
		break
	case "xeneon-battery-status":
		{
			if !colorPattern.MatchString(value.DataColor) || len(value.BatteryDevice) > 128 {
				return 0
			}

			widget.DataColor, widget.BatteryDevice = value.DataColor, value.BatteryDevice
		}
		break
	case "xeneon-calendar":
		{
			if !colorPattern.MatchString(value.DataColor) || value.WeekStart < 0 || value.WeekStart > 1 {
				return 0
			}

			widget.DataColor, widget.WeekStart = value.DataColor, value.WeekStart
		}
		break
	case "xeneon-device-temp", "xeneon-device-rpm":
		{
			if !colorPattern.MatchString(value.DataColor) || value.Max < 1 || value.Max > 50000 ||
				len(strings.TrimSpace(value.HeaderText)) == 0 || len(value.HeaderText) > 80 ||
				len(value.DeviceId) > 128 || value.ChannelId < 0 {
				return 0
			}

			if current.Template == "xeneon-device-temp" {
				if (value.Unit != "°C" && value.Unit != "°F") || value.Max > 1000 {
					return 0
				}
			} else if value.Unit != "RPM" {
				return 0
			}

			widget.DeviceId, widget.ChannelId = value.DeviceId, value.ChannelId
			widget.DataColor, widget.Max = value.DataColor, value.Max
			widget.HeaderText, widget.Unit = strings.TrimSpace(value.HeaderText), value.Unit
		}
		break
	case "xeneon-cpu-temp", "xeneon-gpu-temp", "xeneon-cpu-load", "xeneon-gpu-load":
		{
			if !colorPattern.MatchString(value.DataColor) || value.Max < 1 || value.Max > 1000 ||
				len(strings.TrimSpace(value.HeaderText)) == 0 || len(value.HeaderText) > 80 {
				return 0
			}

			if strings.HasSuffix(current.Template, "-temp") {
				if value.Unit != "°C" && value.Unit != "°F" {
					return 0
				}
			} else if value.Unit != "%" || value.Max > 100 {
				return 0
			}

			widget.DataColor, widget.Max = value.DataColor, value.Max
			widget.HeaderText, widget.Unit = strings.TrimSpace(value.HeaderText), value.Unit
		}
		break
	default:
		return 0
	}

	profile := *d.DeviceProfile
	profile.Widgets = append([]Widget(nil), profile.Widgets...)

	found := false
	for i := range profile.Widgets {
		if profile.Widgets[i].Id == widget.Id {
			profile.Widgets[i] = widget
			found = true
		}
	}
	if !found {
		profile.Widgets = append(profile.Widgets, widget)
	}

	profile.WidgetAreas = make(map[int]WidgetArea, len(d.DeviceProfile.WidgetAreas))
	for key, area := range d.DeviceProfile.WidgetAreas {
		if area.WidgetId == widget.Id {
			cp := widget
			area.Widget = &cp
		}
		profile.WidgetAreas[key] = area
	}

	if err := common.SaveJsonData(profile.Path, &profile); err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile.Path}).Error("Unable to save widget configuration")
		return 0
	}

	d.loadDeviceProfiles()
	return 1
}

// ProcessSetWidgetArea will assign a widget to an area
func (d *Device) ProcessSetWidgetArea(areaId, widgetId int) uint8 {
	if d.DeviceProfile == nil || areaId < 1 || areaId > 11 || widgetId < 0 {
		return 0
	}

	column := 2
	if areaId <= 2 {
		column = 1
	} else if areaId >= 9 {
		column = 3
	}

	area := WidgetArea{}
	if widgetId != 0 {
		definition := d.getWidget(widgetId)
		if definition == nil {
			return 0
		}

		allowed := false
		for _, value := range definition.Columns {
			if value == column {
				allowed = true
			}
		}
		if !allowed {
			return 0
		}

		widget, ok := d.ProcessGetWidget(widgetId).(*Widget)
		if !ok || widget == nil {
			return 0
		}

		cp := *widget
		area = WidgetArea{WidgetId: widgetId, Widget: &cp}
	}

	profile := *d.DeviceProfile
	profile.WidgetAreas = make(map[int]WidgetArea, len(d.DeviceProfile.WidgetAreas))
	for key, value := range d.DeviceProfile.WidgetAreas {
		profile.WidgetAreas[key] = value
	}

	previous := profile.WidgetAreas[areaId]
	if widgetId != 0 {
		for key, value := range profile.WidgetAreas {
			if key != areaId && value.WidgetId == widgetId {
				profile.WidgetAreas[key] = previous
			}
		}
	}

	profile.WidgetAreas[areaId] = area
	if err := common.SaveJsonData(profile.Path, &profile); err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile.Path}).Error("Unable to save widget area")
		return 0
	}

	d.loadDeviceProfiles()
	return 1
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
		deviceProfile.Brightness = 50
		deviceProfile.Backlight = 100
		deviceProfile.Contrast = 50
		deviceProfile.ColorToneRed = 150
		deviceProfile.ColorToneGreen = 126
		deviceProfile.ColorToneBlue = 140
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
		deviceProfile.Brightness = d.DeviceProfile.Brightness
		deviceProfile.Backlight = d.DeviceProfile.Backlight
		deviceProfile.Contrast = d.DeviceProfile.Contrast
		deviceProfile.ColorToneRed = d.DeviceProfile.ColorToneRed
		deviceProfile.ColorToneGreen = d.DeviceProfile.ColorToneGreen
		deviceProfile.ColorToneBlue = d.DeviceProfile.ColorToneBlue
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
			for _, widget := range d.Widgets {
				found := false
				for _, saved := range pf.Widgets {
					if saved.Id == widget.Id {
						found = true
						break
					}
				}
				if !found {
					pf.Widgets = append(pf.Widgets, widget)
				}
			}

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

// setDeviceBrightness will set device brightness
func (d *Device) setDeviceBrightness() {
	if d.DeviceProfile == nil {
		return
	}

	buf := make([]byte, 1)
	buf[0] = byte(d.DeviceProfile.Brightness)

	if d.Exit {
		buf[0] = byte(50)
	}

	_, err := d.transfer(cmdBrightness, buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to set device brightness")
	}
}

// setDeviceContrast will set device contrast
func (d *Device) setDeviceContrast() {
	if d.DeviceProfile == nil {
		return
	}

	buf := make([]byte, 1)
	buf[0] = byte(d.DeviceProfile.Contrast)

	if d.Exit {
		buf[0] = byte(100)
	}

	_, err := d.transfer(cmdContrast, buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to set device contrast")
	}
}

// setDeviceBacklight will set device backlight
func (d *Device) setDeviceBacklight() {
	if d.DeviceProfile == nil {
		return
	}

	buf := make([]byte, 1)
	buf[0] = byte(d.DeviceProfile.Backlight)

	if d.Exit {
		buf[0] = byte(50)
	}

	_, err := d.transfer(cmdBacklight, buf)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to set device backlight")
	}
}

// setDeviceColortone will set device color tone
func (d *Device) setDeviceColortone() {
	colorTones := 3
	if d.DeviceProfile == nil {
		return
	}

	buf := make([]byte, 1)

	for i := 0; i < colorTones; i++ {
		switch i {
		case 0:
			buf[0] = byte(d.DeviceProfile.ColorToneRed)
			break
		case 1:
			buf[0] = byte(d.DeviceProfile.ColorToneGreen)
			break
		case 2:
			buf[0] = byte(d.DeviceProfile.ColorToneBlue)
			break
		}

		if d.Exit {
			buf[0] = dataTypeColorTones[i]
		}

		_, err := d.transfer(cmdColorTones[i], buf)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to set device backlight")
		}
	}
}

// setScreenToDefault will set device to default upon exit
func (d *Device) setScreenToDefault() {
	d.setDeviceBrightness()
	d.setDeviceContrast()
	d.setDeviceBacklight()
	d.setDeviceColortone()
}

// transfer will send data to a device and retrieve device output
func (d *Device) transfer(endpoint, buffer []byte) ([]byte, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	bufferW := make([]byte, bufferSizeWrite)
	bufferW[1] = 0x01
	bufferW[2] = 0x0f

	endpointHeaderPosition := bufferW[headerSize : headerSize+len(endpoint)]
	copy(endpointHeaderPosition, endpoint)
	if len(buffer) > 0 {
		copy(bufferW[headerSize+len(endpoint):headerSize+len(endpoint)+len(buffer)], buffer)
	}

	bufferR := make([]byte, bufferSize)

	if _, err := d.dev.Write(bufferW); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
		return bufferR, err
	}

	if _, err := d.dev.Read(bufferR); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to read data from device")
		return bufferR, err
	}

	return bufferR, nil
}
