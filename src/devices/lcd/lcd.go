package lcd

// Package: LCD Controller
// This is the primary package for LCD pump covers.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/logger"
	"bytes"
	"fmt"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/sstallion/go-hid"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	_ "golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
)

const (
	DisplayLiquid         uint8 = 0
	DisplayPump           uint8 = 1
	DisplayCPU            uint8 = 2
	DisplayGPU            uint8 = 3
	DisplayAllInOne       uint8 = 4
	DisplayLiquidCPU      uint8 = 5
	DisplayCpuGpuTemp     uint8 = 6
	DisplayCpuGpuLoad     uint8 = 7
	DisplayCpuGpuLoadTemp uint8 = 8
	DisplayTime           uint8 = 9
)

var (
	pwd, _         = os.Getwd()
	location       = pwd + "/static/img/lcd/background.jpg"
	animationsPath = pwd + "/static/img/lcd/animations/"
	fontLocation   = pwd + "/static/fonts/teko.ttf"
	mutex          sync.Mutex
	imgWidth              = 480
	imgHeight             = 480
	lcdDevices            = map[string]uint16{}
	vendorId       uint16 = 6940 // Corsair
)

type Animations struct {
	Name   string
	Frames int
	Buffer [][]byte
}

type LCD struct {
	image      image.Image
	font       *truetype.Font
	fontBytes  []byte
	sfntFont   *opentype.Font
	Devices    []Device
	Animations []Animations
}

type Device struct {
	Lcd       *hid.Device
	ProductId uint16
	Product   string
	Serial    string
	AIO       bool
}

var lcd LCD

// Init will initialize LCD data
func Init() {
	lcdDevices = make(map[string]uint16, 0)

	// Open image
	file, e := os.Open(location)
	if e != nil {
		logger.Log(logger.Fields{"error": e, "location": location}).Error("Unable to open LCD image")
		return
	}

	// Decode the image
	img, e := jpeg.Decode(file)
	if e != nil {
		logger.Log(logger.Fields{"error": e, "location": location}).Error("Unable to decode LCD image")
		e = file.Close()
		if e != nil {
			logger.Log(logger.Fields{"error": e, "location": location}).Error("Unable to close LCD image")
		}
		return
	}

	// Load LCD font
	fontBytes, e := os.ReadFile(fontLocation) // Provide the path to your .ttf font file
	if e != nil {
		logger.Log(logger.Fields{"error": e, "location": fontLocation}).Error("Unable to get LCD font")
		return
	}
	fontParsed, e := freetype.ParseFont(fontBytes)
	if e != nil {
		logger.Log(logger.Fields{"error": e, "location": fontLocation}).Error("Unable to parse LCD font")
	}

	sfntFont, e := opentype.Parse(fontBytes)
	if e != nil {
		logger.Log(logger.Fields{"error": e, "location": fontLocation}).Error("Unable to parse LCD font")
	}

	lcdData := &LCD{
		image:     img,
		font:      fontParsed,
		fontBytes: fontBytes,
		sfntFont:  sfntFont,
	}
	lcd = *lcdData
	loadAnimations()
	loadLcdDevices()
}

func loadAnimations() {
	m := 0
	logger.Log(logger.Fields{}).Info("Loading LCD animation images")
	files, err := os.ReadDir(animationsPath)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": animationsPath}).Error("Unable to read content of a folder")
		return
	}
	for _, fi := range files {
		// Define a full path of filename
		imagePath := animationsPath + fi.Name()

		// Check if filename has .json extension
		if !common.IsValidExtension(imagePath, ".gif") {
			logger.Log(logger.Fields{"error": err, "location": animationsPath, "image": imagePath}).Warn("Image needs to have .gif extension")
			continue
		}

		// Process filename
		fileName := strings.Split(fi.Name(), ".")[0]
		if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", fileName); !m {
			logger.Log(logger.Fields{"error": err, "location": animationsPath, "image": imagePath}).Warn("Image name can only have letters and numbers. Please rename your image")
			continue
		}

		// Open image
		file, err := os.Open(imagePath)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": animationsPath, "image": imagePath}).Warn("Unable to open gif animation")
			continue
		}

		// Decode image
		g, err := gif.DecodeAll(file)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": animationsPath, "image": imagePath}).Warn("Error decoding gif animation")
			return
		}

		frames := len(g.Image)
		imageBuffer := make([][]byte, frames) // Slice to store each frame's byte slice

		resizedGIF := &gif.GIF{
			LoopCount: g.LoopCount,
		}
		for i, frame := range g.Image {
			// Convert frame to RGBA to support resizing
			bounds := frame.Bounds()
			palettedImage := image.NewPaletted(image.Rect(0, 0, imgWidth, imgHeight), frame.Palette)
			draw.Draw(palettedImage, palettedImage.Rect, resizeImage(frame, imgWidth, imgHeight), bounds.Min, draw.Src)

			// Append the resized frame to the new GIF
			resizedGIF.Image = append(resizedGIF.Image, palettedImage)
			resizedGIF.Delay = append(resizedGIF.Delay, g.Delay[0])

			// Create a buffer to encode the frame
			var buffer bytes.Buffer
			// Encode the frame back into a GIF
			err = jpeg.Encode(&buffer, palettedImage, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "location": animationsPath, "image": imagePath, "frame": i}).Warn("Failed to encode image frame")
				continue
			}

			// Store the byte slice for the frame
			imageBuffer[i] = buffer.Bytes()
		}
		animation := &Animations{
			Name:   fileName,
			Frames: frames,
			Buffer: imageBuffer,
		}
		lcd.Animations = append(lcd.Animations, *animation)
		m++
	}
	imgText := "image"
	if m > 1 {
		imgText = "images"
	}
	logger.Log(logger.Fields{"amount": m}).Info(fmt.Sprintf("Loaded %v LCD animation %s", m, imgText))
}

// resizeImage will resize image with given width and height
func resizeImage(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	srcBounds := src.Bounds()

	// Simple nearest-neighbor resizing
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := srcBounds.Min.X + x*srcBounds.Dx()/width
			srcY := srcBounds.Min.Y + y*srcBounds.Dy()/height
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}

func loadLcdDevices() {
	lcdProductIds := []uint16{3150, 3139, 3129}

	enum := hid.EnumFunc(func(info *hid.DeviceInfo) error {
		if info.InterfaceNbr == 0 {
			if slices.Contains(lcdProductIds, info.ProductID) {
				lcdDevices[info.SerialNbr] = info.ProductID
			}
		}
		return nil
	})

	// Enumerate all Corsair devices
	err := hid.Enumerate(vendorId, hid.ProductIDAny, enum)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "vendorId": vendorId}).Fatal("Unable to enumerate LCD devices")
		return
	}

	i := 0
	for serial, productId := range lcdDevices {
		lcdPanel, e := hid.Open(vendorId, productId, serial)
		if e != nil {
			logger.Log(logger.Fields{"error": err, "vendorId": vendorId, "productId": productId}).Error("Unable to open LCD HID device")
			continue
		}
		product := ""
		switch productId {
		case 3150:
			product = "iCUE LINK AIO LCD"
		case 3139:
			product = "iCUE LINK XD5 LCD"
		}
		device := &Device{
			Lcd:       lcdPanel,
			ProductId: productId,
			Product:   product,
			Serial:    serial,
			AIO:       productId == 3150,
		}
		lcd.Devices = append(lcd.Devices, *device)
		i++
	}
}

// GetLcdBySerial will return HID device by serial number
func GetLcdBySerial(serial string) *hid.Device {
	for _, device := range lcd.Devices {
		if device.Serial == serial {
			return device.Lcd
		}
	}
	return nil
}

// GetLcdByProductId will return HID device by product id
func GetLcdByProductId(productId uint16) *hid.Device {
	for _, device := range lcd.Devices {
		if device.ProductId == productId {
			return device.Lcd
		}
	}
	return nil
}

// GetNonAIOLCDSerials will return serial numbers of XD5 pumps
func GetNonAIOLCDSerials() []string {
	var serials []string
	for _, device := range lcd.Devices {
		if device.AIO {
			continue
		}
		serials = append(serials, device.Serial)
	}
	return serials
}

// GetAioLCDSerial will return serial number of AIO LCD pumps
func GetAioLCDSerial() string {
	for _, device := range lcd.Devices {
		if device.AIO {
			return device.Serial
		}
	}
	return ""
}

// GetLcdAmount will return amount of available LCDs
func GetLcdAmount() int {
	return len(lcd.Devices)
}

// GetLcdDevices will return all LCD devices
func GetLcdDevices() []Device {
	return lcd.Devices
}

// Stop will stop all LCD devices
func Stop() {
	for _, device := range lcd.Devices {
		lcdReports := map[int][]byte{0: {0x03, 0x1e, 0x01, 0x01}, 1: {0x03, 0x1d, 0x00, 0x01}}
		for i := 0; i <= 1; i++ {
			_, e := device.Lcd.SendFeatureReport(lcdReports[i])
			if e != nil {
				logger.Log(logger.Fields{"error": e}).Fatal("Unable to send report to LCD HID device")
			}
		}
		err := device.Lcd.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Fatal("Unable to close LCD HID device")
		}
	}
}

// drawString will create a new string for image
func drawString(x, y int, fontSite float64, c *freetype.Context, text string) *freetype.Context {
	c.SetFontSize(fontSite)
	pt := freetype.Pt(x, y)
	_, err := c.DrawString(text, pt)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to generate LCD image")
		return nil
	}
	return c
}

// GenerateScreenImage will generate LCD screen image with given value
func GenerateScreenImage(imageType uint8, value, value1, value2, value3 int) []byte {
	mutex.Lock()
	defer mutex.Unlock()

	rgba := image.NewRGBA(lcd.image.Bounds())
	draw.Draw(rgba, rgba.Bounds(), lcd.image, image.Point{}, draw.Src)

	c := freetype.NewContext()
	c.SetFont(lcd.font)
	c.SetFontSize(220)
	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetSrc(image.NewUniform(color.RGBA{R: 255, G: 255, B: 253, A: 255}))

	switch imageType {
	case DisplayLiquid:
		{
			c = drawString(150+int(c.PointToFixed(24)>>6), 100+int(c.PointToFixed(24)>>6), 40, c, "LIQUID TEMP")
			c = drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, c, "[ °C ]")

			opts := opentype.FaceOptions{Size: 220, DPI: 72, Hinting: 0}
			fontFace, err := opentype.NewFace(lcd.sfntFont, &opts)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
			}

			bounds, _ := font.BoundString(fontFace, strconv.Itoa(value))
			textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
			textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

			x := (imgWidth - textWidth) / 2
			y := (imgHeight+textHeight)/2 - 10

			c = drawString(x, y, 220, c, strconv.Itoa(value))
		}
	case DisplayGPU:
		{
			c = drawString(170+int(c.PointToFixed(24)>>6), 100+int(c.PointToFixed(24)>>6), 40, c, "GPU TEMP")
			c = drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, c, "[ °C ]")

			opts := opentype.FaceOptions{Size: 220, DPI: 72, Hinting: 0}
			fontFace, err := opentype.NewFace(lcd.sfntFont, &opts)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
			}

			bounds, _ := font.BoundString(fontFace, strconv.Itoa(value))
			textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
			textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

			x := (imgWidth - textWidth) / 2
			y := (imgHeight+textHeight)/2 - 10

			c = drawString(x, y, 220, c, strconv.Itoa(value))
		}
	case DisplayCPU:
		{
			c = drawString(170+int(c.PointToFixed(24)>>6), 100+int(c.PointToFixed(24)>>6), 40, c, "CPU TEMP")
			c = drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, c, "[ °C ]")

			opts := opentype.FaceOptions{Size: 220, DPI: 72, Hinting: 0}
			fontFace, err := opentype.NewFace(lcd.sfntFont, &opts)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
			}

			bounds, _ := font.BoundString(fontFace, strconv.Itoa(value))
			textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
			textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

			x := (imgWidth - textWidth) / 2
			y := (imgHeight+textHeight)/2 - 10

			c = drawString(x, y, 220, c, strconv.Itoa(value))
		}
	case DisplayPump:
		{
			c = drawString(150+int(c.PointToFixed(24)>>6), 100+int(c.PointToFixed(24)>>6), 40, c, "PUMP SPEED")
			c = drawString(180+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, c, "[ RPM ]")

			opts := opentype.FaceOptions{Size: 200, DPI: 72, Hinting: 0}
			fontFace, err := opentype.NewFace(lcd.sfntFont, &opts)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
			}

			bounds, _ := font.BoundString(fontFace, strconv.Itoa(value))
			textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
			textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

			x := (imgWidth - textWidth) / 2
			y := (imgHeight+textHeight)/2 - 10

			c = drawString(x, y, 200, c, strconv.Itoa(value))
		}
	case DisplayAllInOne:
		{
			c = drawString(120+int(c.PointToFixed(24)>>6), 110+int(c.PointToFixed(24)>>6), 40, c, "LIQUID")
			c = drawString(280+int(c.PointToFixed(24)>>6), 110+int(c.PointToFixed(24)>>6), 40, c, "CPU")
			c = drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, c, "PUMP")
			c = drawString(120+int(c.PointToFixed(24)>>6), 200+int(c.PointToFixed(24)>>6), 100, c, strconv.Itoa(value))
			c = drawString(270+int(c.PointToFixed(24)>>6), 200+int(c.PointToFixed(24)>>6), 100, c, strconv.Itoa(value1))
			c = drawString(160+int(c.PointToFixed(24)>>6), 300+int(c.PointToFixed(24)>>6), 100, c, strconv.Itoa(value2))
		}
	case DisplayLiquidCPU:
		{
			c = drawString(120+int(c.PointToFixed(24)>>6), 110+int(c.PointToFixed(24)>>6), 40, c, "LIQUID")
			c = drawString(280+int(c.PointToFixed(24)>>6), 110+int(c.PointToFixed(24)>>6), 40, c, "CPU")
			c = drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, c, "[ °C ]")
			c = drawString(95+int(c.PointToFixed(24)>>6), 270+int(c.PointToFixed(24)>>6), 160, c, strconv.Itoa(value))
			c = drawString(250+int(c.PointToFixed(24)>>6), 270+int(c.PointToFixed(24)>>6), 160, c, strconv.Itoa(value1))
		}
	case DisplayCpuGpuTemp:
		{
			c = drawString(120+int(c.PointToFixed(24)>>6), 120+int(c.PointToFixed(24)>>6), 40, c, "CPU")
			c = drawString(270+int(c.PointToFixed(24)>>6), 120+int(c.PointToFixed(24)>>6), 40, c, "GPU")
			c = drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, c, "[ °C ]")
			c = drawString(90+int(c.PointToFixed(24)>>6), 270+int(c.PointToFixed(24)>>6), 160, c, strconv.Itoa(value))
			c = drawString(240+int(c.PointToFixed(24)>>6), 270+int(c.PointToFixed(24)>>6), 160, c, strconv.Itoa(value1))
		}
	case DisplayCpuGpuLoadTemp:
		{
			c = drawString(130+int(c.PointToFixed(24)>>6), 140+int(c.PointToFixed(24)>>6), 40, c, "CPU")
			c = drawString(270+int(c.PointToFixed(24)>>6), 140+int(c.PointToFixed(24)>>6), 40, c, "GPU")
			c = drawString(190+int(c.PointToFixed(24)>>6), 90+int(c.PointToFixed(24)>>6), 40, c, "[ °C ]")
			c = drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, c, "[ % ]")
			c = drawString(120+int(c.PointToFixed(24)>>6), 220+int(c.PointToFixed(24)>>6), 80, c, fmt.Sprintf("%02d", value))
			c = drawString(260+int(c.PointToFixed(24)>>6), 220+int(c.PointToFixed(24)>>6), 80, c, fmt.Sprintf("%02d", value1))
			c = drawString(120+int(c.PointToFixed(24)>>6), 290+int(c.PointToFixed(24)>>6), 80, c, fmt.Sprintf("%02d", value2))
			c = drawString(260+int(c.PointToFixed(24)>>6), 290+int(c.PointToFixed(24)>>6), 80, c, fmt.Sprintf("%02d", value3))
		}
	case DisplayCpuGpuLoad:
		{
			c = drawString(120+int(c.PointToFixed(24)>>6), 120+int(c.PointToFixed(24)>>6), 40, c, "CPU")
			c = drawString(270+int(c.PointToFixed(24)>>6), 120+int(c.PointToFixed(24)>>6), 40, c, "GPU")
			c = drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, c, "[ % ]")

			reduce := 0
			bounds, _ := font.BoundString(basicfont.Face7x13, strconv.Itoa(value))
			textWidth := (bounds.Max.X - bounds.Min.X).Floor()
			if value == 100 {
				reduce = 30
			}
			x := 100 + textWidth - reduce
			c = drawString(x, 270+int(c.PointToFixed(24)>>6), 160, c, fmt.Sprintf("%02d", value))

			bounds, _ = font.BoundString(basicfont.Face7x13, strconv.Itoa(value1))
			textWidth = (bounds.Max.X - bounds.Min.X).Floor()
			if value == 100 {
				reduce = 30
			}
			x = 240 + textWidth + 15 - reduce
			c = drawString(x, 270+int(c.PointToFixed(24)>>6), 160, c, fmt.Sprintf("%02d", value1))
		}
	case DisplayTime:
		{
			opts := opentype.FaceOptions{Size: 70, DPI: 72, Hinting: 0}
			fontFace, err := opentype.NewFace(lcd.sfntFont, &opts)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
			}

			bounds, _ := font.BoundString(fontFace, common.GetDate())
			textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
			textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

			x := (imgWidth - textWidth) / 2
			y := (imgHeight+textHeight)/2 - 10
			c = drawString(x, y-50, 70, c, common.GetDate())

			opts = opentype.FaceOptions{Size: 130, DPI: 72, Hinting: 0}
			fontFace, err = opentype.NewFace(lcd.sfntFont, &opts)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
			}

			bounds, _ = font.BoundString(fontFace, common.GetTime())
			textWidth = (bounds.Max.X - bounds.Min.X).Ceil()
			textHeight = (bounds.Max.Y - bounds.Min.Y).Ceil()

			x = (imgWidth - textWidth) / 2
			y = (imgHeight+textHeight)/2 - 10

			c = drawString(x, y+50, 130, c, common.GetTime())
		}
	}

	// Buff it and return
	buffer := new(bytes.Buffer)
	err := jpeg.Encode(buffer, rgba, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to encode LCD image")
		return nil
	}
	return buffer.Bytes()
}
