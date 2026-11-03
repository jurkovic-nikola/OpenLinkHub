package nexus

// Package: CORSAIR iCUE NEXUS
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/appmonitor"
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices/lcd"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/language"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/macro"
	"OpenLinkHub/src/media"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/stats"
	"OpenLinkHub/src/systeminfo"
	"OpenLinkHub/src/temperatures"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/sstallion/go-hid"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	_ "golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	_ "golang.org/x/image/webp"
)

type screenGesture struct {
	active    bool
	start     int
	direction int
	moved     bool
}

type Tile struct {
	Type   string `json:"type"`
	Label  string `json:"label"`
	Width  int    `json:"width"`
	Source string `json:"source,omitempty"`
	Action string `json:"action,omitempty"`
	Target string `json:"target,omitempty"`
}

type Screen struct {
	Name  string `json:"name"`
	Tiles []Tile `json:"tiles"`
}

type DeviceProfile struct {
	Screens     map[string]Screen `json:"screens,omitempty"`
	Active      bool
	Path        string
	Product     string
	Serial      string
	LCDMode     string
	DynamicMode bool
	LCDImage    string
	Label       string
	RgbOff      bool
}

type Button struct {
	ActionCode       uint16    `json:"actionCode"`
	Text             string    `json:"text"`
	TextSize         int       `json:"textSize"`
	Width            int       `json:"width"`
	Height           int       `json:"height"`
	Background       rgb.Color `json:"background"`
	Border           int       `json:"border"`
	BorderColor      rgb.Color `json:"borderColor"`
	ShowIcon         bool      `json:"showIcon"`
	Icon             string    `json:"icon"`
	IconWidth        int       `json:"iconWidth"`
	IconHeight       int       `json:"iconHeight"`
	IconOffsetX      int       `json:"iconOffsetX"`
	IconOffsetY      int       `json:"iconOffsetY"`
	OffsetX          int       `json:"offsetX"`
	OffsetY          int       `json:"offsetY"`
	TouchPositionMin uint16    `json:"touchPositionMin"`
	TouchPositionMax uint16    `json:"touchPositionMax"`
	TextColor        rgb.Color `json:"textColor"`
}

type Profile struct {
	Name       string         `json:"name"`
	Background string         `json:"background"`
	FontPath   string         `json:"fontPath"`
	TextColor  rgb.Color      `json:"textColor"`
	Buttons    map[int]Button `json:"buttons"`
	Keyboard   bool           `json:"keyboard"`
	Buffer     []byte         `json:"-"`
	Image      image.Image    `json:"-"`
	Font       *truetype.Font `json:"-"`
	FontBytes  []byte         `json:"-"`
	SfntFont   *opentype.Font `json:"-"`
}

type LCDProfiles struct {
	Profiles map[string]Profile `json:"profiles"`
}

type Device struct {
	macroDone         chan struct{}
	macroStop         sync.Once
	macroMutex        sync.Mutex
	screenMutex       sync.RWMutex
	Debug             bool
	dev               *hid.Device
	listener          *hid.Device
	Manufacturer      string                    `json:"manufacturer"`
	Product           string                    `json:"product"`
	Serial            string                    `json:"serial"`
	Firmware          string                    `json:"firmware"`
	AIO               bool                      `json:"aio"`
	UserProfiles      map[string]*DeviceProfile `json:"userProfiles"`
	Devices           map[int]string            `json:"devices"`
	DeviceProfile     *DeviceProfile
	OriginalProfile   *DeviceProfile
	Template          string
	VendorId          uint16
	ProductId         uint16
	HasLCD            bool
	LCDModes          map[string]string
	CpuTemp           float32
	GpuTemp           float32
	TemperatureString string
	CPUModel          string
	GPUModel          string
	LCDImage          *lcd.ImageData
	Exit              bool
	mutex             sync.Mutex
	autoRefreshChan   chan struct{}
	lcdRefreshChan    chan struct{}
	lcdImageChan      chan struct{}
	timer             *time.Ticker
	lcdTimer          *time.Ticker
	LCDProfiles       *LCDProfiles
	Keyboard          bool
	instance          *common.Device
}

var (
	pwd                        = ""
	lcdRefreshInterval         = 1000
	deviceRefreshInterval      = 1000
	lcdHeaderSize              = 8
	lcdBufferSize              = 1024
	firmwareReportId           = byte(5)
	featureReportSize          = 32
	maxLCDBufferSizePerRequest = lcdBufferSize - lcdHeaderSize
	imgWidth                   = 640
	imgHeight                  = 48
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
		macroDone: make(chan struct{}),
		dev:       dev,
		Template:  "nexus.html",
		VendorId:  vendorId,
		ProductId: productId,
		LCDModes: map[string]string{
			"cpu-info": "CPU Info",
			"gpu-info": "GPU Info",
		},
		autoRefreshChan: make(chan struct{}),
		lcdRefreshChan:  make(chan struct{}),
		lcdImageChan:    make(chan struct{}),
		timer:           &time.Ticker{},
		lcdTimer:        &time.Ticker{},
	}

	if systeminfo.GetInfo().CPU != nil {
		d.CPUModel = systeminfo.GetInfo().CPU.Model
	}

	if systeminfo.GetInfo().GPU != nil {
		d.GPUModel = systeminfo.GetInfo().GPU[0].Model
	}

	// Bootstrap
	d.getManufacturer()    // Manufacturer
	d.getProduct()         // Product
	d.getSerial()          // Serial
	d.getDeviceFirmware()  // Firmware
	d.loadDeviceProfiles() // Load all device profiles
	d.setAutoRefresh()     // Set auto device refresh
	d.saveDeviceProfile()  // Save profile
	d.loadLcdProfiles()    // LCD profiles
	d.loadLcdFonts()       // LCD fonts
	d.loadLcdBackground()  // LCD background
	d.setupLCD()           // LCD
	d.backendListener()    // Control listener
	d.createDevice()       // Device register
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")

	return d.instance
}

// createDevice will create new device register object
func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeNexus,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    d.Firmware,
		Image:       "icon-device.svg",
		Instance:    d,
	}
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
	d.macroStop.Do(func() { close(d.macroDone) })
	d.Exit = true

	d.timer.Stop()
	d.lcdTimer.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.autoRefreshChan != nil {
				close(d.autoRefreshChan)
			}
			if d.lcdRefreshChan != nil {
				close(d.lcdRefreshChan)
			}
		})
	}()

	if d.dev != nil {
		// Switch LCD back to hardware mode
		lcdReports := map[int][]byte{0: {0x03, 0x0d, 0x01, 0x01}, 1: {0x03, 0x01, 0x64, 0x01}}
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
	d.macroStop.Do(func() { close(d.macroDone) })
	d.Exit = true
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Stopping device (dirty)...")

	d.timer.Stop()
	d.lcdTimer.Stop()
	var once sync.Once
	go func() {
		once.Do(func() {
			if d.autoRefreshChan != nil {
				close(d.autoRefreshChan)
			}
			if d.lcdRefreshChan != nil {
				close(d.lcdRefreshChan)
			}
		})
	}()
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device stopped")
	return 2
}

// UpdateDeviceMetrics will update device metrics
func (d *Device) UpdateDeviceMetrics() {
	header := &metrics.Header{
		Product:  d.Product,
		Serial:   d.Serial,
		Firmware: d.Firmware,
	}
	metrics.Populate(header)
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

	v1, v2, v3, v4 := string(buffer[6]), string(buffer[8]), string(buffer[10]), string(buffer[12:14])
	d.Firmware = fmt.Sprintf("%s.%s.%s.%s", v1, v2, v3, v4)
}

// loadLcdFonts will load LCD font for each profile
func (d *Device) loadLcdFonts() {
	if d.LCDProfiles != nil {
		if len(d.LCDProfiles.Profiles) > 0 {
			for key, value := range d.LCDProfiles.Profiles {
				fontLocation := pwd + value.FontPath

				fontBytes, e := os.ReadFile(fontLocation)
				if e != nil {
					logger.Log(logger.Fields{"error": e, "location": fontLocation}).Error("Unable to get LCD font")
					continue
				}

				fontParsed, e := freetype.ParseFont(fontBytes)
				if e != nil {
					logger.Log(logger.Fields{"error": e, "location": fontLocation}).Error("Unable to parse LCD font")
					continue
				}

				sfntFont, e := opentype.Parse(fontBytes)
				if e != nil {
					logger.Log(logger.Fields{"error": e, "location": fontLocation}).Error("Unable to parse LCD font")
					continue
				}

				value.FontBytes = fontBytes
				value.SfntFont = sfntFont
				value.Font = fontParsed
				d.LCDProfiles.Profiles[key] = value
			}
		}
	}
}

// loadLcdBackground will load background for each profile
func (d *Device) loadLcdBackground() {
	if d.LCDProfiles != nil {
		if len(d.LCDProfiles.Profiles) > 0 {
			for key, value := range d.LCDProfiles.Profiles {
				x, y := 0, 0

				// Load image
				background := pwd + value.Background
				file, err := os.Open(background)
				if err != nil {
					logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": background}).Error("Unable to load LCD profile background")
					continue
				}

				// Decode and resize image
				var src image.Image
				src, err = jpeg.Decode(file)
				if err != nil {
					logger.Log(logger.Fields{"error": err, "image": background}).Warn("Unable to decode LCD background image")
					continue
				}
				resized := common.ResizeImage(src, imgWidth, imgHeight)

				// Convert the image to RGBA to get pixel data
				rgbaImg := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

				// Draw the image onto the RGBA object
				for iy := y; iy < y+imgHeight; iy++ {
					for ix := x; ix < x+imgWidth; ix++ {
						// Get the color of the pixel
						c := resized.At(ix, iy)
						r, g, b, a := c.RGBA()

						// Convert to 8-bit RGBA (8 bits for each channel)
						// Flip R and B values
						rgbaImg.Set(ix-x, iy-y, color.RGBA{
							B: uint8(r >> 8),
							G: uint8(g >> 8),
							R: uint8(b >> 8),
							A: uint8(a >> 8),
						})
					}
				}

				if len(value.Buttons) > 0 {
					for _, v := range value.Buttons {
						// Set background and border colors in ARGB format
						// Red and Blue are shifted
						backgroundColor := color.NRGBA{
							A: 255,
							R: uint8(v.Background.Blue),
							G: uint8(v.Background.Green),
							B: uint8(v.Background.Red),
						}

						borderColor := color.NRGBA{
							A: 255,
							R: uint8(v.BorderColor.Blue),
							G: uint8(v.BorderColor.Green),
							B: uint8(v.BorderColor.Red),
						}

						borderThickness := v.Border // Border thickness in pixels

						// Create a new RGBA image
						img := image.NewRGBA(image.Rect(0, 0, v.Width, v.Height))

						// Fill the entire image with the background color
						draw.Draw(img, img.Bounds(), &image.Uniform{C: backgroundColor}, image.Point{}, draw.Src)

						// Draw the border
						for i := 0; i < v.Width; i++ {
							for j := 0; j < v.Height; j++ {
								if i < borderThickness || i >= v.Width-borderThickness || j < borderThickness || j >= v.Height-borderThickness {
									img.Set(i, j, borderColor)
								}
							}
						}
						xPos, yPos, _, _ := calculateStringXYCustomSize(value.SfntFont, float64(v.TextSize), v.Text, v.Width, v.Height)

						if len(v.Icon) > 0 && v.IconWidth > 0 && v.ShowIcon {
							drawString(value.SfntFont, xPos+(v.IconWidth/2), yPos, float64(v.TextSize), v.Text, img, &v.TextColor)
						} else {
							drawString(value.SfntFont, xPos, yPos, float64(v.TextSize), v.Text, img, &v.TextColor)
						}

						// Position for overlay on background
						offset := image.Pt(v.OffsetX, v.OffsetY)
						overlayRect := image.Rectangle{Min: offset, Max: offset.Add(img.Bounds().Size())}

						// Icons
						if len(v.Icon) > 0 && v.ShowIcon {
							icon := pwd + v.Icon
							overlayFile, e := os.Open(icon)
							if e != nil {
								logger.Log(logger.Fields{"error": e, "serial": d.Serial, "location": icon}).Error("Unable to load LCD profile icon")
								continue
							}

							overlayImg, _, decodeError := image.Decode(overlayFile)
							if decodeError != nil {
								logger.Log(logger.Fields{"error": decodeError, "serial": d.Serial, "location": icon}).Error("Unable to decode LCD profile icon")
								continue
							}

							resizedIcon := common.ResizeImage(overlayImg, v.IconWidth, v.IconHeight)

							// Convert the image to RGBA to get pixel data
							iconImg := image.NewRGBA(image.Rect(0, 0, v.IconWidth, v.IconHeight))

							// Draw the image onto the RGBA object
							for iy := y; iy < y+v.IconHeight; iy++ {
								for ix := x; ix < x+v.IconWidth; ix++ {
									// Get the color of the pixel
									c := resizedIcon.At(ix, iy)
									r, g, b, a := c.RGBA()

									// Convert to 8-bit RGBA (8 bits for each channel)
									// Flip R and B values
									iconImg.Set(ix-x, iy-y, color.RGBA{
										B: uint8(r >> 8),
										G: uint8(g >> 8),
										R: uint8(b >> 8),
										A: uint8(a >> 8),
									})
								}
							}

							// Set overlay position
							offsetIcon := image.Pt(v.IconOffsetX, v.IconOffsetY)
							overlayIconRect := image.Rectangle{Min: offsetIcon, Max: offsetIcon.Add(iconImg.Bounds().Size())}

							// Draw overlay onto background with transparency (draw.Over handles alpha)
							draw.Draw(img, overlayIconRect, iconImg, image.Point{}, draw.Over)
						}

						// Draw overlay on background
						draw.Draw(rgbaImg, overlayRect, img, image.Point{}, draw.Over)
					}
				}

				value.Image = rgbaImg
				d.LCDProfiles.Profiles[key] = value
			}
		}
	}
}

// loadLcdProfiles will load LCD profiles
func (d *Device) loadLcdProfiles() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	lcdProfilesFile := pwd + "/database/nexus/nexus.json"
	file, err := os.Open(lcdProfilesFile)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": lcdProfilesFile}).Error("Unable to load LCD profile")
		return
	}
	if err = json.NewDecoder(file).Decode(&d.LCDProfiles); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": lcdProfilesFile}).Error("Unable to decode profile")
		return
	}

	if val, ok := d.LCDProfiles.Profiles[d.DeviceProfile.LCDMode]; ok {
		d.Keyboard = val.Keyboard
	}

	if strings.Contains(d.DeviceProfile.LCDMode, ";") {
		d.DeviceProfile.DynamicMode = true
		d.Keyboard = false
	}
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

// setAutoRefresh will refresh device data
func (d *Device) setAutoRefresh() {
	d.timer = time.NewTicker(time.Duration(deviceRefreshInterval) * time.Millisecond)
	go func() {
		for {
			select {
			case <-d.timer.C:
				d.setTemperatures()
			case <-d.autoRefreshChan:
				d.timer.Stop()
				return
			}
		}
	}()
}

// setTemperatures will store current CPU temperature
func (d *Device) setTemperatures() {
	cpu := temperatures.GetCpuTemperature()
	gpu := temperatures.GetGpuTemperature()

	d.screenMutex.Lock()
	d.CpuTemp = cpu
	d.GpuTemp = gpu
	d.screenMutex.Unlock()
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
		deviceProfile.Label = "Companion Touch Screen"
		deviceProfile.LCDMode = "cpu-info"
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
		deviceProfile.DynamicMode = d.DeviceProfile.DynamicMode
		deviceProfile.RgbOff = d.DeviceProfile.RgbOff
		deviceProfile.Screens = d.DeviceProfile.Screens
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

// calculateStringXY will calculate X,Y and center text
func calculateStringXY(fontData *opentype.Font, fontSize float64, value string) (int, int, int, int) {
	opts := opentype.FaceOptions{Size: fontSize, DPI: 72, Hinting: 0}
	fontFace, err := opentype.NewFace(fontData, &opts)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
	}

	bounds, _ := font.BoundString(fontFace, value)
	textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
	textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

	x := (imgWidth - textWidth) / 2
	y := (imgHeight + textHeight) / 2
	return x, y, textWidth, textHeight
}

// calculateStringXY will calculate X,Y and return width and height of the given text
func calculateStringXYCustomSize(fontData *opentype.Font, fontSize float64, value string, width, height int) (int, int, int, int) {
	opts := opentype.FaceOptions{Size: fontSize, DPI: 72, Hinting: 0}
	fontFace, err := opentype.NewFace(fontData, &opts)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
	}

	bounds, _ := font.BoundString(fontFace, value)
	textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
	textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

	x := (width - textWidth) / 2
	y := (height + textHeight) / 2
	return x, y, textWidth, textHeight
}

// textToRight will align text to the right
func textToRight(fontData *opentype.Font, fontSize float64, value string) (int, int, int, int) {
	opts := opentype.FaceOptions{Size: fontSize, DPI: 72, Hinting: 0}
	fontFace, err := opentype.NewFace(fontData, &opts)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
	}

	bounds, _ := font.BoundString(fontFace, value)
	textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
	textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

	x := imgWidth - textWidth
	y := (imgHeight + textHeight) / 2
	return x, y, textWidth, textHeight
}

// UpdateDeviceLcdProfile will change device LCD profile
func (d *Device) UpdateDeviceLcdProfile(profileName string) uint8 {
	d.screenMutex.Lock()
	defer d.screenMutex.Unlock()
	if d.DeviceProfile == nil {
		return 0
	}

	if d.DeviceProfile.LCDMode == profileName {
		return 2
	}

	_, custom := d.DeviceProfile.Screens[profileName]
	if val, ok := d.LCDProfiles.Profiles[profileName]; ok || custom || profileName == "constructor" {
		if profileName == "gpu-info" && systeminfo.GetInfo().GPU == nil {
			return 0
		}

		d.DeviceProfile.LCDMode = profileName
		d.DeviceProfile.DynamicMode = false
		d.Keyboard = val.Keyboard
		d.saveDeviceProfile()
		return 1
	} else {
		if strings.Contains(profileName, ";") {
			parts := strings.SplitN(profileName, ";", 2)
			serial := parts[0]
			channelId, err := strconv.Atoi(parts[1])
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to parse channel id")
				return 0
			}

			data := stats.GetAIOData(serial, channelId)
			if data == nil {
				return 0
			}

			d.DeviceProfile.LCDMode = profileName
			d.DeviceProfile.DynamicMode = true
			d.Keyboard = false
			d.saveDeviceProfile()
			return 1
		} else {
			return 0
		}
	}
}

// renderDeviceInfo will render CPU / GPU info
func (d *Device) renderDeviceInfo(model, val1, val2 string) []byte {
	if d.LCDProfiles == nil || d.DeviceProfile == nil {
		return nil
	}

	if d.DeviceProfile.DynamicMode {
		if profile, ok := d.LCDProfiles.Profiles["default"]; ok {
			if profile.Image != nil {
				rgba := image.NewRGBA(profile.Image.Bounds())
				draw.Draw(rgba, rgba.Bounds(), profile.Image, image.Point{}, draw.Src)

				c := freetype.NewContext()
				c.SetDPI(72)
				c.SetFont(profile.Font)
				c.SetClip(rgba.Bounds())
				c.SetDst(rgba)
				c.SetSrc(image.NewUniform(color.RGBA{R: 255, G: 255, B: 253, A: 255}))

				_, y, _, _ := calculateStringXY(profile.SfntFont, 30, model)
				drawString(profile.SfntFont, 8, y, 30, model, rgba, &profile.TextColor)

				x, y, w, _ := textToRight(profile.SfntFont, 30, val1)
				drawString(profile.SfntFont, x-10, y, 30, val1, rgba, &profile.TextColor)

				x, y, _, _ = textToRight(profile.SfntFont, 30, val2)

				if len(val1) == 0 {
					drawString(profile.SfntFont, (x-15)-w, y, 30, val2, rgba, &profile.TextColor)
				} else {
					drawString(profile.SfntFont, (x-50)-w, y, 30, val2, rgba, &profile.TextColor)
				}
				return renderImageToBytes(rgba)
			}
		}
	} else {
		if profile, ok := d.LCDProfiles.Profiles[d.DeviceProfile.LCDMode]; ok {
			if profile.Image == nil {
				return nil
			}

			rgba := image.NewRGBA(profile.Image.Bounds())
			draw.Draw(rgba, rgba.Bounds(), profile.Image, image.Point{}, draw.Src)

			c := freetype.NewContext()
			c.SetDPI(72)
			c.SetFont(profile.Font)
			c.SetClip(rgba.Bounds())
			c.SetDst(rgba)
			c.SetSrc(image.NewUniform(color.RGBA{R: 255, G: 255, B: 253, A: 255}))

			_, y, _, _ := calculateStringXY(profile.SfntFont, 30, model)
			drawString(profile.SfntFont, 8, y, 30, model, rgba, &profile.TextColor)

			x, y, w, _ := textToRight(profile.SfntFont, 30, val1)
			drawString(profile.SfntFont, x-10, y, 30, val1, rgba, &profile.TextColor)

			x, y, _, _ = textToRight(profile.SfntFont, 30, val2)
			drawString(profile.SfntFont, (x-50)-w, y, 30, val2, rgba, &profile.TextColor)

			return renderImageToBytes(rgba)
		}
	}
	return nil
}

// renderEmpty will render empty background
func (d *Device) renderEmpty() []byte {
	if d.LCDProfiles == nil || d.DeviceProfile == nil {
		return nil
	}

	if profile, ok := d.LCDProfiles.Profiles[d.DeviceProfile.LCDMode]; ok {
		if profile.Image == nil {
			return nil
		}

		rgba := image.NewRGBA(profile.Image.Bounds())
		draw.Draw(rgba, rgba.Bounds(), profile.Image, image.Point{}, draw.Src)

		c := freetype.NewContext()
		c.SetDPI(72)
		c.SetFont(profile.Font)
		c.SetClip(rgba.Bounds())
		c.SetDst(rgba)
		c.SetSrc(image.NewUniform(color.RGBA{R: 255, G: 255, B: 253, A: 255}))
		return renderImageToBytes(rgba)
	}
	return nil
}

// renderDeviceInfo will render date and time info
func (d *Device) renderTimeInfo(time string) []byte {
	if d.LCDProfiles == nil || d.DeviceProfile == nil {
		return nil
	}

	if profile, ok := d.LCDProfiles.Profiles[d.DeviceProfile.LCDMode]; ok {
		if profile.Image == nil {
			return nil
		}

		rgba := image.NewRGBA(profile.Image.Bounds())
		draw.Draw(rgba, rgba.Bounds(), profile.Image, image.Point{}, draw.Src)

		c := freetype.NewContext()
		c.SetDPI(72)
		c.SetFont(profile.Font)
		c.SetClip(rgba.Bounds())
		c.SetDst(rgba)
		c.SetSrc(image.NewUniform(color.RGBA{R: 255, G: 255, B: 253, A: 255}))

		x, y, _, _ := calculateStringXY(profile.SfntFont, 30, time)
		drawString(profile.SfntFont, x, y, 30, time, rgba, &profile.TextColor)

		return renderImageToBytes(rgba)
	}
	return nil
}

// renderImageToBytes renders RGBA image into byte array
func renderImageToBytes(img *image.RGBA) []byte {
	data := make([]byte, imgWidth*imgHeight*4)

	offset := 0
	for y := 0; y < imgHeight; y++ {
		for x := 0; x < imgWidth; x++ {
			c := img.At(x, y).(color.RGBA)
			data[offset] = c.R
			data[offset+1] = c.G
			data[offset+2] = c.B
			data[offset+3] = c.A
			offset += 4
		}
	}

	return data
}

// drawString will create a new string for image
func drawString(fontData *opentype.Font, x, y int, fontSite float64, text string, rgba *image.RGBA, textColor *rgb.Color) {
	pt := freetype.Pt(x, y)
	opts := opentype.FaceOptions{Size: fontSite, DPI: 72, Hinting: 0}
	fontFace, err := opentype.NewFace(fontData, &opts)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
		return
	}

	d := &font.Drawer{
		Dst:  rgba,
		Src:  image.NewUniform(color.RGBA{R: uint8(textColor.Blue), G: uint8(textColor.Green), B: uint8(textColor.Red), A: 255}),
		Face: fontFace, // Use the built-in font
		Dot:  pt,
	}

	d.DrawString(text)
}

// ScreenCatalog returns the editable layouts. Saved layouts are replaced, never mutated.
func (d *Device) ScreenCatalog() map[string]Screen {
	d.screenMutex.RLock()
	defer d.screenMutex.RUnlock()

	screens := make(map[string]Screen)
	screens["constructor"] = constructorScreen()

	if d.LCDProfiles != nil {
		if profile, ok := d.LCDProfiles.Profiles["media-control"]; ok {
			screen := Screen{Name: profile.Name}
			for _, id := range slices.Sorted(maps.Keys(profile.Buttons)) {
				button := profile.Buttons[id]
				screen.Tiles = append(screen.Tiles, Tile{
					Label:  button.Text,
					Type:   "button",
					Width:  1,
					Action: "media",
					Target: strconv.Itoa(int(button.ActionCode)),
				})
			}
			screens["media-control"] = screen
		}
	}

	if d.DeviceProfile != nil {
		for id, screen := range d.DeviceProfile.Screens {
			screens[id] = screen
		}
	}

	return screens
}

// constructorScreen returns the default Constructor layout with CPU temperature, GPU temperature and a clock.
func constructorScreen() Screen {
	return Screen{
		Name: language.GetValue("txtNexusConstructor"),
		Tiles: []Tile{
			{Type: "sensor", Label: "CPU", Source: "cpu", Width: 2},
			{Type: "sensor", Label: "GPU", Source: "gpu", Width: 2},
			{Type: "clock", Width: 1},
		},
	}
}

// ManualProfiles will get list of manual profiles
func (d *Device) ManualProfiles() interface{} {
	return appmonitor.GetManualApplicationProfiles()
}

// BatterySources returns the battery devices available to the tile editor.
func (d *Device) BatterySources() map[string]stats.BatteryStats {
	return stats.GetBatteryStats()
}

// TouchMacros lists macros that can complete from a single tap.
func (d *Device) TouchMacros() map[int]string {
	names := make(map[int]string)
	for id, profile := range macro.GetProfiles() {
		if tapMacro(profile) {
			names[id] = profile.Name
		}
	}

	return names
}

// tapMacro will check if macro is valid for execution
func tapMacro(profile macro.Macro) bool {
	if len(profile.Actions) == 0 {
		return false
	}
	for _, action := range profile.Actions {
		if action.ActionHold {
			return false
		}
		switch action.ActionType {
		case 1, 3, 5, 6, 9, 20:
		default:
			return false
		}
	}
	return true
}

// waitMacro will wait for macro
func (d *Device) waitMacro(delay time.Duration) bool {
	if delay <= 0 {
		select {
		case <-d.macroDone:
			return false
		default:
			return true
		}
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-d.macroDone:
		return false
	case <-timer.C:
		return true
	}
}

// runTouchMacro ignores overlapping taps while a macro is playing.
func (d *Device) runTouchMacro(target string) {
	id, err := strconv.Atoi(target)
	if err != nil {
		return
	}

	profile := macro.GetProfile(id)
	if profile == nil || !tapMacro(*profile) || !d.macroMutex.TryLock() {
		return
	}

	go func() {
		defer d.macroMutex.Unlock()
		for run := 0; run < max(1, profile.Repeat); run++ {
			if run > 0 {
				if !d.waitMacro(time.Duration(profile.RepeatDelay) * time.Millisecond) {
					return
				}
			}
			for _, key := range slices.Sorted(maps.Keys(profile.Actions)) {
				action := profile.Actions[key]
				for n := 0; n < max(1, int(action.ActionRepeat)); n++ {
					if n > 0 {
						if !d.waitMacro(time.Duration(action.ActionRepeatDelay) * time.Millisecond) {
							return
						}
					}
					if !d.waitMacro(0) {
						return
					}
					switch action.ActionType {
					case 1, 3:
						inputmanager.InputControlKeyboard(action.ActionCommand, false)
					case 5:
						if !d.waitMacro(time.Duration(action.ActionDelay) * time.Millisecond) {
							return
						}
					case 6:
						inputmanager.InputControlKeyboardText(action.ActionText)
					case 9:
						inputmanager.InputControlMouse(action.ActionCommand)
					case 20:
						if action.MousePositionAbsolute {
							inputmanager.InputControlMoveAbsolute(int32(action.MousePositionX), int32(action.MousePositionY))
						} else {
							inputmanager.InputControlMove(int32(action.MousePositionX), int32(action.MousePositionY))
						}
					}
				}
			}
		}
	}()
}

// SaveDeviceScreen will save and activate a custom screen. An empty id creates a new copy.
func (d *Device) SaveDeviceScreen(id string, screen Screen) uint8 {
	d.screenMutex.Lock()
	defer d.screenMutex.Unlock()

	if d.DeviceProfile == nil || len(screen.Tiles) == 0 || len(screen.Tiles) > 6 {
		return 0
	}

	screen.Name = strings.TrimSpace(screen.Name)
	if screen.Name == "" || utf8.RuneCountInString(screen.Name) > 48 {
		return 0
	}

	screens := maps.Clone(d.DeviceProfile.Screens)
	if screens == nil {
		screens = make(map[string]Screen)
	}

	if id == "" {
		if len(screens) >= 64 {
			return 0
		}
		id = "custom-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	} else if _, ok := screens[id]; !ok && id != "constructor" && id != "media-control" {
		return 0
	}

	for key, existing := range screens {
		if key != id && strings.EqualFold(existing.Name, screen.Name) {
			current, ok := screens[id]
			if (!ok || !strings.EqualFold(current.Name, screen.Name)) && id != "constructor" && id != "media-control" {
				return 0
			}
		}
	}

	for _, tile := range screen.Tiles {
		if tile.Width < 1 || tile.Width > 3 || utf8.RuneCountInString(tile.Label) > 32 || len(tile.Source) > 160 || len(tile.Target) > 160 {
			return 0
		}

		switch tile.Type {
		case "button", "clock", "track":
		case "battery":
			if tile.Source == "" {
				return 0
			}
		case "sensor":
			if tile.Source != "cpu" && tile.Source != "gpu" {
				if _, _, _, ok := sensorSource(tile.Source); !ok {
					return 0
				}
			}
		default:
			return 0
		}
		switch tile.Action {
		case "":
		case "media":
			code, err := strconv.Atoi(tile.Target)
			if err != nil || code < 1 || code > 7 {
				return 0
			}
		case "macro":
			id, err := strconv.Atoi(tile.Target)
			if err != nil || id < 0 {
				return 0
			}
		case "profile":
			if tile.Target == "" {
				return 0
			}
		case "screen":
			_, custom := screens[tile.Target]
			_, preset := d.LCDProfiles.Profiles[tile.Target]
			if !custom && !preset && tile.Target != "constructor" || tile.Target == "default" {
				return 0
			}
		default:
			return 0
		}
	}
	screens[id] = screen
	profile := *d.DeviceProfile
	profile.Screens = screens
	profile.LCDMode = id
	profile.DynamicMode = false

	if err := common.SaveJsonData(profile.Path, &profile); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to save NEXUS screen")
		return 0
	}

	d.DeviceProfile.Screens = screens
	d.DeviceProfile.LCDMode = id
	d.DeviceProfile.DynamicMode = false
	d.Keyboard = false
	return 1
}

// DeleteDeviceScreen removes a custom layout and clears touch actions that point to it.
func (d *Device) DeleteDeviceScreen(id string) uint8 {
	d.screenMutex.Lock()
	defer d.screenMutex.Unlock()
	if d.DeviceProfile == nil || !strings.HasPrefix(id, "custom-") {
		return 0
	}

	profile := *d.DeviceProfile
	if _, ok := profile.Screens[id]; !ok {
		return 0
	}
	profile.Screens = maps.Clone(profile.Screens)
	delete(profile.Screens, id)

	for key, screen := range profile.Screens {
		screen.Tiles = slices.Clone(screen.Tiles)
		for i, tile := range screen.Tiles {
			if tile.Action == "screen" && tile.Target == id {
				screen.Tiles[i].Action = ""
				screen.Tiles[i].Target = ""
			}
		}
		profile.Screens[key] = screen
	}

	active := profile.LCDMode == id
	if active {
		profile.LCDMode = "constructor"
		profile.DynamicMode = false
	}

	if err := common.SaveJsonData(profile.Path, &profile); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to delete NEXUS screen")
		return 0
	}

	*d.DeviceProfile = profile
	if active {
		d.Keyboard = false
	}

	return 1
}

// sensorSource will check sensor source group
func sensorSource(source string) (string, int, string, bool) {
	parts := strings.Split(source, ";")
	if len(parts) != 3 || parts[0] == "" || (parts[2] != "temp" && parts[2] != "rpm") {
		return "", 0, "", false
	}

	channel, err := strconv.Atoi(parts[1])
	return parts[0], channel, parts[2], err == nil && channel >= 0
}

// tileEdges returns the exclusive right edge of each tile in LCD pixels.
func tileEdges(tiles []Tile) []int {
	total := 0
	for _, tile := range tiles {
		total += tile.Width
	}

	if total <= 0 {
		return nil
	}

	edges := make([]int, len(tiles))
	used := 0
	for i, tile := range tiles {
		used += tile.Width
		edges[i] = used * imgWidth / total
	}

	return edges
}

// currentTrack will return currently playing track
func currentTrack() string {
	player, err := media.GetCurrentlyPlaying()
	if err != nil || player.Title == "" {
		return "—"
	}
	if len(player.Artists) == 0 {
		return player.Title
	}
	return player.Title + " · " + strings.Join(player.Artists, ", ")
}

// tileValue will return selected tile value
func (d *Device) tileValue(tile Tile) string {
	if tile.Type == "battery" {
		battery, ok := stats.GetBatteryStats()[tile.Source]
		if !ok {
			return "—"
		}
		return fmt.Sprintf("%d%%", battery.Level)
	}
	if tile.Type == "track" {
		return currentTrack()
	}
	if tile.Type == "clock" {
		return common.GetTime()
	}
	switch tile.Source {
	case "cpu":
		return dashboard.GetDashboard().TemperatureToString(d.CpuTemp)
	case "gpu":
		if systeminfo.GetInfo().GPU != nil {
			return dashboard.GetDashboard().TemperatureToString(d.GpuTemp)
		}
	default:
		serial, channel, metric, ok := sensorSource(tile.Source)
		if ok {
			data := stats.GetAIOData(serial, channel)
			if data != nil {
				if metric == "temp" && data.TemperatureString != "" {
					return data.TemperatureString
				}
				if metric == "rpm" && data.Speed != "" {
					return data.Speed
				}
			}
		}
	}
	return "—"
}

// renderScreen will render screen and return byte array
func (d *Device) renderScreen(screen Screen) []byte {
	profile := d.LCDProfiles.Profiles["default"]
	if profile.SfntFont == nil {
		return nil
	}
	frame := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	left := 0
	for i, right := range tileEdges(screen.Tiles) {
		tile := screen.Tiles[i]
		width := right - left
		if width <= 4 {
			return nil
		}
		cell := image.NewRGBA(image.Rect(0, 0, width-4, imgHeight-4))
		draw.Draw(cell, cell.Bounds(), image.NewUniform(color.RGBA{32, 32, 32, 255}), image.Point{}, draw.Src)
		label := tile.Label
		if tile.Type == "button" {
			x, y, _, _ := calculateStringXYCustomSize(profile.SfntFont, 24, label, width-4, imgHeight-4)
			drawString(profile.SfntFont, max(2, x), y, 24, label, cell, &profile.TextColor)
		} else {
			value := d.tileValue(tile)
			if label != "" {
				drawString(profile.SfntFont, 5, 15, 17, label, cell, &profile.TextColor)
				drawString(profile.SfntFont, 5, 39, 28, value, cell, &profile.TextColor)
			} else {
				x, y, _, _ := calculateStringXYCustomSize(profile.SfntFont, 29, value, width-4, imgHeight-4)
				drawString(profile.SfntFont, max(2, x), y, 29, value, cell, &profile.TextColor)
			}
		}
		offset := image.Pt(left+2, 2)
		draw.Draw(frame, cell.Bounds().Add(offset), cell, image.Point{}, draw.Src)
		left = right
	}
	return renderImageToBytes(frame)
}

// activeScreen will activate screen
func (d *Device) activeScreen() (Screen, bool) {
	if screen, ok := d.DeviceProfile.Screens[d.DeviceProfile.LCDMode]; ok {
		return screen, true
	}
	if d.DeviceProfile.LCDMode == "constructor" {
		return constructorScreen(), true
	}
	return Screen{}, false
}

// touchX interpolates between the boundaries of the shipped media touch zones.
func (d *Device) touchX(raw int) int {
	buttons := d.LCDProfiles.Profiles["media-control"].Buttons
	lastRaw := 0
	lastX := 0
	for i := 1; i <= 6; i++ {
		endRaw := imgWidth
		endX := imgWidth
		if i < 6 {
			prev, ok := buttons[i-1]
			next, found := buttons[i]
			if !ok || !found {
				return raw
			}
			endRaw = (int(prev.TouchPositionMax) + int(next.TouchPositionMin)) / 2
			endX = (prev.OffsetX + prev.Width + next.OffsetX) / 2
		}
		if endRaw <= lastRaw {
			return raw
		}
		if raw <= endRaw {
			return min(imgWidth-1, lastX+(raw-lastRaw)*(endX-lastX)/(endRaw-lastRaw))
		}
		lastRaw = endRaw
		lastX = endX
	}
	return imgWidth
}

// touchAction will run action on touch detection
func (d *Device) touchAction(raw uint16) (string, string) {
	d.screenMutex.RLock()
	defer d.screenMutex.RUnlock()
	if d.DeviceProfile == nil || d.LCDProfiles == nil {
		return "", ""
	}
	if screen, ok := d.activeScreen(); ok {
		x := d.touchX(int(raw))
		for i, right := range tileEdges(screen.Tiles) {
			if x < right {
				return screen.Tiles[i].Action, screen.Tiles[i].Target
			}
		}
	} else if d.Keyboard {
		code := d.getActionCodeByPosition(raw)
		if code != 0 {
			return "media", strconv.Itoa(int(code))
		}
	}
	return "", ""
}

// setupLCD will activate and configure LCD
func (d *Device) setupLCD() {
	d.lcdTimer = time.NewTicker(time.Duration(lcdRefreshInterval) * time.Millisecond)
	d.lcdRefreshChan = make(chan struct{})
	go func() {
		for {
			select {
			case <-d.lcdTimer.C:
				d.refreshScreen()
			case <-d.lcdRefreshChan:
				d.lcdTimer.Stop()
				return
			}
		}
	}()
}

// refreshScreen will refresh screen
func (d *Device) refreshScreen() {
	d.screenMutex.RLock()
	defer d.screenMutex.RUnlock()

	if d.DeviceProfile == nil || d.LCDProfiles == nil {
		return
	}

	if screen, ok := d.activeScreen(); ok {
		buffer := d.renderScreen(screen)
		if len(buffer) > 0 {
			d.transfer(buffer)
		}
		return
	}

	lcdMode := d.DeviceProfile.LCDMode
	if d.DeviceProfile.DynamicMode {
		if strings.Contains(lcdMode, ";") {
			parts := strings.SplitN(lcdMode, ";", 2)
			serial := parts[0]
			channelId, err := strconv.Atoi(parts[1])
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to parse channel id")
				return
			}
			data := stats.GetAIOData(serial, channelId)
			if data != nil {
				buf := d.renderDeviceInfo(data.Device, data.Speed, data.TemperatureString)
				d.transfer(buf)
			}
		}
	} else {
		switch lcdMode {
		case "cpu-info":
			{
				cpuTemp := dashboard.GetDashboard().TemperatureToString(d.CpuTemp)
				cpuUtil := fmt.Sprintf("%.2f %s", systeminfo.GetCpuUtilization(), "%")
				buf := d.renderDeviceInfo(d.CPUModel, cpuTemp, cpuUtil)
				d.transfer(buf)
			}
			return
		case "gpu-info":
			{
				gpuModel := d.GPUModel
				gpuTemp := dashboard.GetDashboard().TemperatureToString(d.GpuTemp)
				gpuUtil := fmt.Sprintf("%.2v %s", systeminfo.GetGPUUtilization(), "%")
				buf := d.renderDeviceInfo(gpuModel, gpuTemp, gpuUtil)
				d.transfer(buf)
			}
			return
		case "time-info":
			{
				dateTime := fmt.Sprintf("%s - %s", common.GetDate(), common.GetTime())
				buf := d.renderTimeInfo(dateTime)
				d.transfer(buf)
			}
			return
		default:
			{
				if d.LCDProfiles == nil {
					return
				}

				if _, ok := d.LCDProfiles.Profiles[lcdMode]; ok {
					buf := d.renderEmpty()
					d.transfer(buf)
				}
			}
			return
		}
	}
}

// getListenerData will listen for keyboard events and return data on success or nil on failure.
// ReadWithTimeout is mandatory due to the nature of listening for events
func (d *Device) getListenerData() []byte {
	if d.listener == nil {
		return nil
	}
	data := make([]byte, 512)
	n, err := d.listener.ReadWithTimeout(data, 100*time.Millisecond)
	if err != nil || n == 0 {
		return nil
	}
	return data[:n]
}

// update will track horizontal motion during contact
func (g *screenGesture) update(state byte, x int) (int, bool) {
	switch state {
	case 1:
		if !g.active {
			g.active = true
			g.start = x
			return 0, false
		}
		distance := x - g.start
		if distance > 20 || distance < -20 {
			g.moved = true
		}
		if g.direction == 0 {
			if distance <= -80 {
				g.direction = 1
			} else if distance >= 80 {
				g.direction = -1
			}
		}
	case 0:
		if g.active {
			direction := g.direction
			tap := !g.moved
			*g = screenGesture{}
			return direction, tap
		}
	}
	return 0, false
}

// swipeScreen will cycle through Constructor and saved layouts
func (d *Device) swipeScreen(direction int) {
	d.screenMutex.RLock()
	if d.DeviceProfile == nil {
		d.screenMutex.RUnlock()
		return
	}

	current := d.DeviceProfile.LCDMode
	ids := []string{"constructor"}
	for _, id := range slices.Sorted(maps.Keys(d.DeviceProfile.Screens)) {
		if strings.HasPrefix(id, "custom-") {
			ids = append(ids, id)
		}
	}

	if d.LCDProfiles != nil {
		for _, id := range slices.Sorted(maps.Keys(d.LCDProfiles.Profiles)) {
			if id == "default" || id == "constructor" || (id == "gpu-info" && systeminfo.GetInfo().GPU == nil) {
				continue
			}
			ids = append(ids, id)
		}
	}

	d.screenMutex.RUnlock()
	if len(ids) < 2 {
		return
	}

	index := slices.Index(ids, current)
	if index < 0 {
		if direction > 0 {
			index = len(ids) - 1
		} else {
			index = 0
		}
	}

	d.UpdateDeviceLcdProfile(ids[(index+direction+len(ids))%len(ids)])
}

// backendListener will listen for events from the device
func (d *Device) backendListener() {
	var gesture screenGesture
	go func() {
		var path string
		err := hid.Enumerate(d.VendorId, d.ProductId, func(info *hid.DeviceInfo) error {
			if info.InterfaceNbr == 0 && info.SerialNbr == d.Serial {
				path = info.Path
			}
			return nil
		})

		if err != nil || path == "" {
			logger.Log(logger.Fields{"serial": d.Serial, "error": err}).Error("Unable to find NEXUS touch interface")
			return
		}

		listener, err := hid.OpenPath(path)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to open backend listener HID device")
			return
		}

		d.listener = listener
		for {
			select {
			default:
				if d.Exit {
					err = d.listener.Close()
					if err != nil {
						logger.Log(logger.Fields{"error": err, "vendorId": d.VendorId}).Error("Failed to close listener")
						return
					}
					return
				}

				data := d.getListenerData()
				if len(data) == 0 || data == nil {
					time.Sleep(5 * time.Millisecond)
					continue
				}

				if len(data) < 8 || data[0] != 1 || data[1] != 2 || data[2] != 0x21 {
					continue
				}

				start := gesture.start
				x := binary.LittleEndian.Uint16(data[6:8])

				direction, tap := gesture.update(data[5], int(x))
				if direction != 0 {
					d.swipeScreen(direction)
				}

				if !tap {
					continue
				}

				action, target := d.touchAction(uint16(start))
				switch action {
				case "macro":
					d.runTouchMacro(target)
				case "media":
					code, err := strconv.ParseUint(target, 10, 16)
					if err == nil {
						inputmanager.InputControlKeyboard(uint16(code), false)
					}
				case "profile":
					appmonitor.ManualApplicationProfileChange(target)
				case "screen":
					d.UpdateDeviceLcdProfile(target)
				}
			}
		}
	}()
}

// getActionCodeByPixel will return keyboard action code based on pixel position
func (d *Device) getActionCodeByPosition(pixel uint16) uint16 {
	if d.DeviceProfile == nil {
		return 0
	}

	if d.LCDProfiles == nil {
		return 0
	}

	if value, ok := d.LCDProfiles.Profiles[d.DeviceProfile.LCDMode]; ok {
		for _, button := range value.Buttons {
			if pixel >= button.TouchPositionMin && pixel <= button.TouchPositionMax {
				return button.ActionCode
			}
		}
	}
	return 0
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
		bufferW[2] = 0x40

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
