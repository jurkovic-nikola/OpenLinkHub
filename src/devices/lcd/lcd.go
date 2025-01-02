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
	"golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	_ "golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/webp"
	_ "golang.org/x/image/webp"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"os"
	"path/filepath"
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
	DisplayImage          uint8 = 10
)

const (
	ImageFormatJpg  = 0
	ImageFormatBmp  = 1
	ImageFormatWebp = 2
	ImageFormatGif  = 3
)

var (
	pwd, _       = os.Getwd()
	location     = pwd + "/database/lcd/background.jpg"
	images       = pwd + "/database/lcd/images/"
	fontLocation = pwd + "/static/fonts/teko.ttf"
	mutex        sync.Mutex
	imgWidth            = 480
	imgHeight           = 480
	lcdDevices          = map[string]uint16{}
	vendorId     uint16 = 6940 // Corsair
)

type ImageData struct {
	Name   string
	Frames int
	Buffer []Frames
}

type Frames struct {
	Buffer []byte
	Delay  float64
}
type LCD struct {
	image     image.Image
	font      *truetype.Font
	fontBytes []byte
	sfntFont  *opentype.Font
	Devices   []Device
	ImageData []ImageData
}

type Device struct {
	Lcd       *hid.Device
	ProductId uint16
	VendorId  uint16
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
	loadLcdImages()
	loadLcdDevices()
}

// Reconnect will reconnect to all available LCD devices
func Reconnect() {
	for key, device := range lcd.Devices {
		lcdPanel, e := hid.Open(vendorId, device.ProductId, device.Serial)
		if e != nil {
			logger.Log(logger.Fields{"error": e, "vendorId": vendorId, "productId": device.ProductId}).Error("Unable to reconnect LCD HID device")
			continue
		}
		lcd.Devices[key].Lcd = lcdPanel
	}
}

// GetLcdImages will return all lcd images
func GetLcdImages() []ImageData {
	return lcd.ImageData
}

// GetLcdImage will return image data based on image name
func GetLcdImage(image string) *ImageData {
	for _, value := range lcd.ImageData {
		if value.Name == image {
			return &value
		}
	}
	return nil
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

// GetAioLCDData will return data of AIO LCD pumps
func GetAioLCDData() *Device {
	for _, device := range lcd.Devices {
		if device.AIO {
			return &device
		}
	}
	return nil
}

// GetNonAioLCDData will return data of XD5 LCD pumps
func GetNonAioLCDData() []Device {
	var devices []Device
	for _, device := range lcd.Devices {
		if !device.AIO {
			devices = append(devices, device)
		}
	}
	return devices
}

// GetLcdAmount will return amount of available LCDs
func GetLcdAmount() int {
	return len(lcd.Devices)
}

// GetLcdDevices will return all LCD devices
func GetLcdDevices() []Device {
	return lcd.Devices
}

func calculateIntXY(fontSize float64, value int) (int, int) {
	opts := opentype.FaceOptions{Size: fontSize, DPI: 72, Hinting: 0}
	fontFace, err := opentype.NewFace(lcd.sfntFont, &opts)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
	}

	bounds, _ := font.BoundString(fontFace, strconv.Itoa(value))
	textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
	textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

	x := (imgWidth - textWidth) / 2
	y := (imgHeight+textHeight)/2 - 10
	return x, y
}

func calculateStringXY(fontSize float64, value string) (int, int) {
	opts := opentype.FaceOptions{Size: fontSize, DPI: 72, Hinting: 0}
	fontFace, err := opentype.NewFace(lcd.sfntFont, &opts)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
	}

	bounds, _ := font.BoundString(fontFace, value)
	textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
	textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

	x := (imgWidth - textWidth) / 2
	y := (imgHeight+textHeight)/2 - 10
	return x, y
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
			x, y := calculateStringXY(40, "LIQUID TEMP")
			c = drawString(x, y-120, 40, c, "LIQUID TEMP")

			x, y = calculateStringXY(40, "[ °C ]")
			c = drawString(x, y+120, 40, c, "[ °C ]")

			x, y = calculateStringXY(240, strconv.Itoa(value))
			c = drawString(x, y, 240, c, strconv.Itoa(value))
		}
	case DisplayGPU:
		{
			x, y := calculateStringXY(40, "GPU TEMP")
			c = drawString(x, y-120, 40, c, "GPU TEMP")

			x, y = calculateStringXY(40, "[ °C ]")
			c = drawString(x, y+120, 40, c, "[ °C ]")

			x, y = calculateStringXY(240, strconv.Itoa(value))
			c = drawString(x, y, 240, c, strconv.Itoa(value))
		}
	case DisplayCPU:
		{
			x, y := calculateStringXY(40, "CPU TEMP")
			c = drawString(x, y-120, 40, c, "CPU TEMP")

			x, y = calculateStringXY(40, "[ °C ]")
			c = drawString(x, y+120, 40, c, "[ °C ]")

			x, y = calculateStringXY(240, strconv.Itoa(value))
			c = drawString(x, y, 240, c, strconv.Itoa(value))
		}
	case DisplayPump:
		{
			x, y := calculateStringXY(40, "PUMP TEMP")
			c = drawString(x, y-120, 40, c, "PUMP TEMP")

			x, y = calculateStringXY(40, "[ °C ]")
			c = drawString(x, y+120, 40, c, "[ RPM ]")

			x, y = calculateStringXY(200, strconv.Itoa(value))
			c = drawString(x, y, 200, c, strconv.Itoa(value))
		}
	case DisplayAllInOne:
		{
			x, y := calculateStringXY(40, "LIQUID")
			c = drawString(x-80, y-110, 40, c, "LIQUID")

			x, y = calculateStringXY(40, "CPU")
			c = drawString(x+80, y-110, 40, c, "CPU")

			x, y = calculateStringXY(40, "PUMP")
			c = drawString(x, y+130, 40, c, "PUMP")

			x, y = calculateStringXY(100, strconv.Itoa(value))
			c = drawString(x-80, y-40, 100, c, strconv.Itoa(value))

			x, y = calculateStringXY(100, strconv.Itoa(value1))
			c = drawString(x+80, y-40, 100, c, strconv.Itoa(value1))

			x, y = calculateStringXY(100, strconv.Itoa(value2))
			c = drawString(x, y+60, 100, c, strconv.Itoa(value2))
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
			x, y := calculateStringXY(70, common.GetDate())
			c = drawString(x, y-50, 70, c, common.GetDate())

			x, y = calculateStringXY(130, common.GetTime())
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

func loadImage(imagePath string, format uint8) {
	file, err := os.Open(imagePath)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": images, "image": imagePath}).Warn("Unable to open image")
		return
	}

	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": images, "image": imagePath}).Warn("Unable to close image")
		}
	}(file)

	filename := filepath.Base(imagePath)
	fileName := strings.TrimSuffix(filename, filepath.Ext(filename))
	imageBuffer := make([]Frames, 1)

	switch format {
	case ImageFormatJpg: // JPG, JPEG
		{
			var src image.Image
			var buffer bytes.Buffer

			src, err = jpeg.Decode(file)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "location": images, "image": imagePath}).Warn("Unable to decode image")
				return
			}

			resized := common.ResizeImage(src, imgWidth, imgHeight)
			err = jpeg.Encode(&buffer, resized, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "location": images, "image": imagePath}).Warn("Failed to encode image frame")
				return
			}
			imageBuffer[0] = Frames{
				Buffer: buffer.Bytes(),
				Delay:  0,
			}
		}
		break
	case ImageFormatBmp: // BMP
		{
			var src image.Image
			var buffer bytes.Buffer

			src, err = bmp.Decode(file)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "location": images, "image": imagePath}).Warn("Unable to decode image")
				return
			}

			resized := common.ResizeImage(src, imgWidth, imgHeight)
			err = jpeg.Encode(&buffer, resized, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "location": images, "image": imagePath}).Warn("Failed to encode image frame")
				return
			}
			imageBuffer[0] = Frames{
				Buffer: buffer.Bytes(),
				Delay:  0,
			}
		}
		break
	case ImageFormatWebp: // WEBP static
		{
			var src image.Image
			var buffer bytes.Buffer

			src, err = webp.Decode(file)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "location": images, "image": imagePath}).Warn("Unable to decode image")
				return
			}

			resized := common.ResizeImage(src, imgWidth, imgHeight)
			err = jpeg.Encode(&buffer, resized, nil)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "location": images, "image": imagePath}).Warn("Failed to encode image frame")
				return
			}
			imageBuffer[0] = Frames{
				Buffer: buffer.Bytes(),
				Delay:  0,
			}
		}
		break
	case ImageFormatGif: // Gif
		{
			var src *gif.GIF
			src, err = gif.DecodeAll(file)
			if err != nil {
				logger.Log(logger.Fields{"error": err, "location": images, "image": imagePath}).Warn("Error decoding gif animation")
				return
			}
			imageBuffer = make([]Frames, len(src.Image))

			for i, frame := range src.Image {
				var buffer bytes.Buffer
				resized := common.ResizeImage(frame, imgWidth, imgHeight)
				err = jpeg.Encode(&buffer, resized, nil)
				if err != nil {
					logger.Log(logger.Fields{"error": err, "location": images, "image": imagePath, "frame": i}).Warn("Failed to encode image frame")
					continue
				}
				imageBuffer[i] = Frames{
					Buffer: buffer.Bytes(),
					Delay:  float64(src.Delay[i]) * 10, // Multiply by 10 to get frame delay in milliseconds
				}
			}
		}
		break
	}

	imageList := &ImageData{
		Name:   fileName,
		Frames: len(imageBuffer),
		Buffer: imageBuffer,
	}
	lcd.ImageData = append(lcd.ImageData, *imageList)
}

// loadLcdImages will load all LCD images
func loadLcdImages() {
	files, err := os.ReadDir(images)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": images}).Error("Unable to read content of a folder")
		return
	}
	for _, fi := range files {
		imagePath := images + fi.Name()

		// Process filename
		filename := filepath.Base(imagePath)
		fileName := strings.TrimSuffix(filename, filepath.Ext(filename))
		if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", fileName); !m {
			logger.Log(logger.Fields{"error": err, "location": images, "image": imagePath}).Warn("Image name can only have letters, numbers, - and _. Please rename your image")
			continue
		}

		switch strings.ToLower(filepath.Ext(imagePath)) {
		case ".jpg":
			{
				loadImage(imagePath, ImageFormatJpg)
			}
			break
		case ".jpeg":
			{
				loadImage(imagePath, ImageFormatJpg)
			}
			break
		case ".bmp":
			{
				loadImage(imagePath, ImageFormatBmp)
			}
			break
		case ".webp":
			{
				loadImage(imagePath, ImageFormatWebp)
			}
			break
		case ".gif":
			{
				loadImage(imagePath, ImageFormatGif)
			}
			break
		default:
			logger.Log(logger.Fields{"error": err, "location": images, "image": imagePath}).Warn("Invalid image extension")
			continue
		}
	}
}

// loadLcdDevices will load all available LCD devices
func loadLcdDevices() {
	lcdProductIds := []uint16{3150, 3139}

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

	for serial, productId := range lcdDevices {
		lcdPanel, e := hid.Open(vendorId, productId, serial)
		if e != nil {
			logger.Log(logger.Fields{"error": e, "vendorId": vendorId, "productId": productId}).Error("Unable to open LCD HID device")
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
			VendorId:  vendorId,
		}
		lcd.Devices = append(lcd.Devices, *device)
	}
}
