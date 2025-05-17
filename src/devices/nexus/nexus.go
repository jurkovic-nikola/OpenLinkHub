package nexus

// Package: CORSAIR iCUE NEXUS
// This is the primary package for CORSAIR iCUE NEXUS
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices/lcd"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/stats"
	"OpenLinkHub/src/systeminfo"
	"OpenLinkHub/src/temperatures"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/sstallion/go-hid"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	_ "golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	_ "golang.org/x/image/webp"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DeviceProfile struct {
	Active      bool
	Path        string
	Product     string
	Serial      string
	LCDMode     string
	DynamicMode bool
	LCDImage    string
	Label       string
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
func Init(vendorId, productId uint16, serial string) *Device {
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
		d.GPUModel = systeminfo.GetInfo().GPU.Model
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
	logger.Log(logger.Fields{"serial": d.Serial, "product": d.Product}).Info("Device successfully initialized")
	return d
}

// Stop will stop all device operations and switch a device back to hardware mode
func (d *Device) Stop() {
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
	if d.DeviceProfile == nil {
		return 0
	}

	if d.DeviceProfile.LCDMode == profileName {
		return 2
	}

	if val, ok := d.LCDProfiles.Profiles[profileName]; ok {
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

// setupLCD will activate and configure LCD
func (d *Device) setupLCD() {
	d.lcdTimer = time.NewTicker(time.Duration(lcdRefreshInterval) * time.Millisecond)
	d.lcdRefreshChan = make(chan struct{})
	go func() {
		for {
			select {
			case <-d.lcdTimer.C:
				lcdMode := d.DeviceProfile.LCDMode
				if d.DeviceProfile.DynamicMode {
					if strings.Contains(lcdMode, ";") {
						parts := strings.SplitN(lcdMode, ";", 2)
						serial := parts[0]
						channelId, err := strconv.Atoi(parts[1])
						if err != nil {
							logger.Log(logger.Fields{"error": err}).Error("Unable to parse channel id")
							break
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
						break
					case "gpu-info":
						{
							gpuModel := d.GPUModel
							gpuTemp := dashboard.GetDashboard().TemperatureToString(d.GpuTemp)
							gpuUtil := fmt.Sprintf("%.2v %s", systeminfo.GetGPUUtilization(), "%")
							buf := d.renderDeviceInfo(gpuModel, gpuTemp, gpuUtil)
							d.transfer(buf)
						}
						break
					case "time-info":
						{
							dateTime := fmt.Sprintf("%s - %s", common.GetDate(), common.GetTime())
							buf := d.renderTimeInfo(dateTime)
							d.transfer(buf)
						}
						break
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
						break
					}
				}
			case <-d.lcdRefreshChan:
				d.lcdTimer.Stop()
				return
			}
		}
	}()
}

// getListenerData will listen for keyboard events and return data on success or nil on failure.
// ReadWithTimeout is mandatory due to the nature of listening for events
func (d *Device) getListenerData() []byte {
	data := make([]byte, 32)
	n, err := d.listener.ReadWithTimeout(data, 100*time.Millisecond)
	if err != nil || n == 0 {
		return nil
	}
	return data
}

// backendListener will listen for events from the device
func (d *Device) backendListener() {
	blocked := false
	go func() {
		listener, err := hid.Open(d.VendorId, d.ProductId, d.Serial)
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
					continue
				}

				if d.DeviceProfile == nil {
					continue
				}

				if d.Keyboard {
					active := data[5] // Touch activation
					if active == 1 && blocked == false {
						blocked = true
						touchPosition := binary.LittleEndian.Uint16(data[6:8])
						actionCode := d.getActionCodeByPosition(touchPosition)
						if actionCode != 0 {
							inputmanager.InputControlKeyboard(actionCode, false)
						}
					} else if active == 0 && blocked == true {
						blocked = false
					}
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

		// Send it
		if _, err := d.dev.Write(bufferW); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial}).Error("Unable to write to a device")
			break
		}
	}
}
