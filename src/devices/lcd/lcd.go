package lcd

// Package: LCD Controller
// This is the primary package for LCD pump covers.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
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
	"math"
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
	DisplayArc            uint8 = 100
	DisplayDoubleArc      uint8 = 101
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
	lcdSensors          = map[uint8]string{
		0: "CPU Temp",
		1: "GPU Temp",
		2: "Liquid Temp",
		3: "CPU Load",
		4: "GPU Load",
	}
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
	lcdDevices = make(map[string]uint16)

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

	// Single Arc
	InitArc()

	// Double arc
	InitDoubleArc()
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

// GetLcdSensors will return list of LCD sensors
func GetLcdSensors() map[uint8]string {
	return lcdSensors
}

// generateColor will generate color.RGBA based on red, green and blue
func generateColor(c rgb.Color) color.RGBA {
	return color.RGBA{R: uint8(c.Red), G: uint8(c.Green), B: uint8(c.Blue), A: 255}
}

// drawCircle will draw filled circle
func drawCircle(img *image.RGBA, centerX, centerY, radius float64, col color.Color) {
	minX := int(centerX - radius)
	maxX := int(centerX + radius)
	minY := int(centerY - radius)
	maxY := int(centerY + radius)

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			if dx*dx+dy*dy <= radius*radius {
				if x >= 0 && y >= 0 && x < img.Bounds().Dx() && y < img.Bounds().Dy() {
					img.Set(x, y, col)
				}
			}
		}
	}
}

// drawSmoothArcGradient will draw arc with color gradient
func drawSmoothArcGradient(img *image.RGBA, centerX, centerY, innerR, outerR float64, startAngle, endAngle float64, cStart, cEnd color.RGBA) {
	step := 0.01
	angleRange := endAngle - startAngle

	for angle := startAngle; angle <= endAngle; angle += step {
		t := (angle - startAngle) / angleRange
		col := lerpColor(cStart, cEnd, t)

		radius := (innerR + outerR) / 2
		x := centerX + radius*math.Cos(angle)
		y := centerY + radius*math.Sin(angle)
		drawCircle(img, x, y, (outerR-innerR)/2, col)
	}
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

// sensorMaximumValue will return sensor maximum value
func sensorMaximumValue(sensor uint8) int {
	switch sensor {
	case 0:
		return 100
	case 1:
		return 90
	case 2:
		return 60
	case 3, 4:
		return 100
	default:
		return 100
	}
}

// isSensorTemperature will check if given sensor is temperature one
func isSensorTemperature(sensor uint8) bool {
	if sensor == 0 || sensor == 1 || sensor == 2 {
		return true
	}
	return false
}

// drawArcOutline will draw small outline
func drawArcOutline(img *image.RGBA, centerX, centerY, innerR, outerR float64, startAngle, endAngle float64, col color.RGBA) {
	step := 0.01
	radius := (innerR + outerR) / 2
	thickness := 3.0 // very thin

	for angle := startAngle; angle <= endAngle; angle += step {
		x := centerX + radius*math.Cos(angle)
		y := centerY + radius*math.Sin(angle)
		drawCircle(img, x, y, thickness, col)
	}
}

// lerpColor will interpolates between two colors
func lerpColor(c1, c2 color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c1.R)*(1-t) + float64(c2.R)*t),
		G: uint8(float64(c1.G)*(1-t) + float64(c2.G)*t),
		B: uint8(float64(c1.B)*(1-t) + float64(c2.B)*t),
		A: 255,
	}
}

// GetCustomLcdProfiles will return list of LCD profiles currently supported by the product. This list is defined
// manually and its updated how new modes are created.
func GetCustomLcdProfiles() map[uint8]interface{} {
	profiles := make(map[uint8]interface{})
	profiles[DisplayArc] = GetArc()
	profiles[DisplayDoubleArc] = GetDoubleArc()
	return profiles
}

// GenerateDoubleArcScreenImage handles generation or double arc screen image
func GenerateDoubleArcScreenImage(values []int) []byte {
	arcImage := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	bg := generateColor(doubleRrc.Background)
	draw.Draw(arcImage, arcImage.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(lcd.font)
	c.SetClip(arcImage.Bounds())
	c.SetDst(arcImage)
	c.SetSrc(image.NewUniform(color.RGBA{R: 255, G: 255, B: 253, A: 255}))

	// Common radius math
	outerRadius := float64(imgWidth)/2 - doubleRrc.Margin
	innerRadius := outerRadius - doubleRrc.Thickness
	centerY := float64(imgHeight) / 2

	// Border
	borderColor := generateColor(doubleRrc.BorderColor)

	// Left arc
	leftArc := doubleRrc.Arcs[0]
	leftColStart := generateColor(leftArc.StartColor)
	leftColEnd := generateColor(leftArc.EndColor)
	leftCenterX := doubleRrc.Margin + outerRadius
	leftMax := sensorMaximumValue(leftArc.Sensor)
	leftValue := values[leftArc.Sensor]
	if leftValue > leftMax {
		leftValue = leftMax
	}

	leftArcStart := math.Pi/2 + doubleRrc.GapRadians/2
	leftArcEnd := leftArcStart + float64(leftValue)/float64(leftMax)*(math.Pi-doubleRrc.GapRadians)
	leftBorderEnd := leftArcStart + float64(leftMax)/float64(leftMax)*(math.Pi-doubleRrc.GapRadians)

	// Border
	drawArcOutline(arcImage, leftCenterX, centerY, innerRadius, outerRadius, leftArcStart, leftBorderEnd, borderColor)

	// Arc
	drawSmoothArcGradient(arcImage, leftCenterX, centerY, innerRadius, outerRadius, leftArcStart, leftArcEnd, leftColStart, leftColEnd)

	if isSensorTemperature(leftArc.Sensor) {
		v := dashboard.GetDashboard().TemperatureToString(float32(leftValue))
		x, y := calculateStringXY(100, v)
		drawColorString(x, y-80, 100, v, arcImage, leftArc.TextColor)
	} else {
		v := fmt.Sprintf("%v %s", leftValue, "%")
		x, y := calculateStringXY(100, v)
		drawColorString(x, y-80, 100, v, arcImage, leftArc.TextColor)
	}

	separator := "-------------------------------------------"
	x, y := calculateStringXY(20, separator)
	drawColorString(x, y, 20, separator, arcImage, doubleRrc.SeparatorColor)

	// Right Arc
	rightArc := doubleRrc.Arcs[1]
	rightColStart := generateColor(rightArc.EndColor) // Reversed
	rightColEnd := generateColor(rightArc.StartColor) // Reversed
	rightCenterX := float64(imgWidth) - doubleRrc.Margin - outerRadius
	rightMax := sensorMaximumValue(rightArc.Sensor)
	rightValue := values[rightArc.Sensor]
	if rightValue > leftMax {
		rightValue = leftMax
	}
	rightArcEnd := math.Pi/2 - doubleRrc.GapRadians/2
	rightArcStart := rightArcEnd - float64(rightValue)/float64(rightMax)*(math.Pi-doubleRrc.GapRadians)
	rightBorderStart := rightArcEnd - float64(rightMax)/float64(rightMax)*(math.Pi-doubleRrc.GapRadians)

	// Border
	drawArcOutline(arcImage, rightCenterX, centerY, innerRadius, outerRadius, rightBorderStart, rightArcEnd, borderColor)

	// Arc
	drawSmoothArcGradient(arcImage, rightCenterX, centerY, innerRadius, outerRadius, rightArcStart, rightArcEnd, rightColStart, rightColEnd)

	// Text
	if isSensorTemperature(rightArc.Sensor) {
		v := dashboard.GetDashboard().TemperatureToString(float32(rightValue))
		x, y = calculateStringXY(100, v)
		drawColorString(x, y+80, 100, v, arcImage, rightArc.TextColor)
	} else {
		v := fmt.Sprintf("%v %s", rightValue, "%")
		x, y = calculateStringXY(100, v)
		drawColorString(x, y+80, 100, v, arcImage, rightArc.TextColor)
	}

	// Send it
	var b bytes.Buffer
	err := jpeg.Encode(&b, arcImage, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to encode LCD image")
		return nil
	}
	return b.Bytes()
}

// GenerateArcScreenImage handles generation or arc screen image
func GenerateArcScreenImage(arcType, sensor, value int) []byte {
	mutex.Lock()
	defer mutex.Unlock()

	// Prevent over 100
	if (arcType == 0 || arcType == 1) && value > 100 {
		value = 100
	}

	// Prevent over 60
	if arcType == 2 && value > 60 {
		value = 60
	}

	bg := generateColor(arc.Background)
	arcStartColor := generateColor(arc.StartColor)
	arcEndColor := generateColor(arc.EndColor)
	arcThickness := arc.Thickness

	maxValue := sensorMaximumValue(arc.Sensor)

	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	centerX, centerY := float64(imgWidth)/2, float64(imgHeight)/2
	outerRadius := float64(imgWidth)/2 - arc.Margin
	innerRadius := outerRadius - arcThickness
	angleFraction := float64(value) / float64(maxValue)
	angleFraction = angleFraction - (arc.GapRadians * 2 / (2 * math.Pi))
	endAngle := startAngle + angleFraction*2*math.Pi

	// Border
	borderAngle := startAngle + 1*2*math.Pi
	borderColor := generateColor(arc.BorderColor)

	// Draw the arc
	drawArcOutline(img, centerX, centerY, innerRadius, outerRadius, startAngle, borderAngle, borderColor)
	drawSmoothArcGradient(img, centerX, centerY, innerRadius, outerRadius, startAngle, endAngle, arcStartColor, arcEndColor)

	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(lcd.font)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.NewUniform(color.RGBA{R: 255, G: 255, B: 253, A: 255}))

	// Text
	if isSensorTemperature(uint8(sensor)) {
		// Value
		v := dashboard.GetDashboard().Temperature(float32(value))
		x, y := calculateStringXY(250, v[0])
		drawColorString(x, y, 250, v[0], img, arc.TextColor)

		// Unit
		unit := fmt.Sprintf("[ %s ]", v[1])
		x, y = calculateStringXY(40, unit)
		drawColorString(x, y+120, 40, unit, img, arc.TextColor)
	} else {
		// Value
		x, y := calculateStringXY(280, strconv.Itoa(value))
		drawColorString(x, y, 280, strconv.Itoa(value), img, arc.TextColor)

		// Unit
		x, y = calculateStringXY(40, "[ % ]")
		drawColorString(x, y+120, 40, "[ % ]", img, arc.TextColor)
	}

	// Send it
	var b bytes.Buffer
	err := jpeg.Encode(&b, img, nil)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to encode LCD image")
		return nil
	}
	return b.Bytes()
}

// GenerateScreenImage will generate LCD screen image with given value
func GenerateScreenImage(imageType uint8, value, value1, value2, value3 int) []byte {
	mutex.Lock()
	defer mutex.Unlock()

	rgba := image.NewRGBA(lcd.image.Bounds())
	draw.Draw(rgba, rgba.Bounds(), lcd.image, image.Point{}, draw.Src)

	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(lcd.font)
	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetSrc(image.NewUniform(color.RGBA{R: 255, G: 255, B: 253, A: 255}))

	switch imageType {
	case DisplayLiquid:
		{
			x, y := calculateStringXY(40, "LIQUID TEMP")
			drawString(x, y-120, 40, "LIQUID TEMP", rgba)

			x, y = calculateStringXY(40, "[ °C ]")
			drawString(x, y+120, 40, "[ °C ]", rgba)

			x, y = calculateStringXY(240, strconv.Itoa(value))
			drawString(x, y, 240, strconv.Itoa(value), rgba)
		}
	case DisplayGPU:
		{
			x, y := calculateStringXY(40, "GPU TEMP")
			drawString(x, y-120, 40, "GPU TEMP", rgba)

			x, y = calculateStringXY(40, "[ °C ]")
			drawString(x, y+120, 40, "[ °C ]", rgba)

			x, y = calculateStringXY(240, strconv.Itoa(value))
			drawString(x, y, 240, strconv.Itoa(value), rgba)
		}
	case DisplayCPU:
		{
			x, y := calculateStringXY(40, "CPU TEMP")
			drawString(x, y-120, 40, "CPU TEMP", rgba)

			x, y = calculateStringXY(40, "[ °C ]")
			drawString(x, y+120, 40, "[ °C ]", rgba)

			x, y = calculateStringXY(240, strconv.Itoa(value))
			drawString(x, y, 240, strconv.Itoa(value), rgba)
		}
	case DisplayPump:
		{
			x, y := calculateStringXY(40, "PUMP SPEED")
			drawString(x, y-120, 40, "PUMP SPEED", rgba)

			x, y = calculateStringXY(40, "[ RPM ]")
			drawString(x, y+120, 40, "[ RPM ]", rgba)

			x, y = calculateStringXY(200, strconv.Itoa(value))
			drawString(x, y, 200, strconv.Itoa(value), rgba)
		}
	case DisplayAllInOne:
		{
			x, y := calculateStringXY(40, "LIQUID")
			drawString(x-80, y-110, 40, "LIQUID", rgba)

			x, y = calculateStringXY(40, "CPU")
			drawString(x+80, y-110, 40, "CPU", rgba)

			x, y = calculateStringXY(40, "PUMP")
			drawString(x, y+130, 40, "PUMP", rgba)

			x, y = calculateStringXY(100, strconv.Itoa(value))
			drawString(x-80, y-40, 100, strconv.Itoa(value), rgba)

			x, y = calculateStringXY(100, strconv.Itoa(value1))
			drawString(x+80, y-40, 100, strconv.Itoa(value1), rgba)

			x, y = calculateStringXY(100, strconv.Itoa(value2))
			drawString(x, y+60, 100, strconv.Itoa(value2), rgba)
		}
	case DisplayLiquidCPU:
		{
			drawString(120+int(c.PointToFixed(24)>>6), 110+int(c.PointToFixed(24)>>6), 40, "LIQUID", rgba)
			drawString(280+int(c.PointToFixed(24)>>6), 110+int(c.PointToFixed(24)>>6), 40, "CPU", rgba)
			drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, "[ °C ]", rgba)
			drawString(95+int(c.PointToFixed(24)>>6), 270+int(c.PointToFixed(24)>>6), 160, strconv.Itoa(value), rgba)
			drawString(250+int(c.PointToFixed(24)>>6), 270+int(c.PointToFixed(24)>>6), 160, strconv.Itoa(value1), rgba)
		}
	case DisplayCpuGpuTemp:
		{
			drawString(120+int(c.PointToFixed(24)>>6), 120+int(c.PointToFixed(24)>>6), 40, "CPU", rgba)
			drawString(270+int(c.PointToFixed(24)>>6), 120+int(c.PointToFixed(24)>>6), 40, "GPU", rgba)
			drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, "[ °C ]", rgba)
			drawString(90+int(c.PointToFixed(24)>>6), 270+int(c.PointToFixed(24)>>6), 160, strconv.Itoa(value), rgba)
			drawString(240+int(c.PointToFixed(24)>>6), 270+int(c.PointToFixed(24)>>6), 160, strconv.Itoa(value1), rgba)
		}
	case DisplayCpuGpuLoadTemp:
		{
			drawString(130+int(c.PointToFixed(24)>>6), 140+int(c.PointToFixed(24)>>6), 40, "CPU", rgba)
			drawString(270+int(c.PointToFixed(24)>>6), 140+int(c.PointToFixed(24)>>6), 40, "GPU", rgba)
			drawString(190+int(c.PointToFixed(24)>>6), 90+int(c.PointToFixed(24)>>6), 40, "[ °C ]", rgba)
			drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, "[ % ]", rgba)
			drawString(120+int(c.PointToFixed(24)>>6), 220+int(c.PointToFixed(24)>>6), 80, fmt.Sprintf("%02d", value), rgba)
			drawString(260+int(c.PointToFixed(24)>>6), 220+int(c.PointToFixed(24)>>6), 80, fmt.Sprintf("%02d", value1), rgba)
			drawString(120+int(c.PointToFixed(24)>>6), 290+int(c.PointToFixed(24)>>6), 80, fmt.Sprintf("%02d", value2), rgba)
			drawString(260+int(c.PointToFixed(24)>>6), 290+int(c.PointToFixed(24)>>6), 80, fmt.Sprintf("%02d", value3), rgba)
		}
	case DisplayCpuGpuLoad:
		{
			drawString(120+int(c.PointToFixed(24)>>6), 120+int(c.PointToFixed(24)>>6), 40, "CPU", rgba)
			drawString(270+int(c.PointToFixed(24)>>6), 120+int(c.PointToFixed(24)>>6), 40, "GPU", rgba)
			drawString(190+int(c.PointToFixed(24)>>6), 350+int(c.PointToFixed(24)>>6), 40, "[ % ]", rgba)

			reduce := 0
			bounds, _ := font.BoundString(basicfont.Face7x13, strconv.Itoa(value))
			textWidth := (bounds.Max.X - bounds.Min.X).Floor()
			if value == 100 {
				reduce = 30
			}
			x := 100 + textWidth - reduce
			drawString(x, 270+int(c.PointToFixed(24)>>6), 160, fmt.Sprintf("%02d", value), rgba)

			bounds, _ = font.BoundString(basicfont.Face7x13, strconv.Itoa(value1))
			textWidth = (bounds.Max.X - bounds.Min.X).Floor()
			if value == 100 {
				reduce = 30
			}
			x = 240 + textWidth + 15 - reduce
			drawString(x, 270+int(c.PointToFixed(24)>>6), 160, fmt.Sprintf("%02d", value1), rgba)
		}
	case DisplayTime:
		{
			x, y := calculateStringXY(70, common.GetDate())
			drawString(x, y-50, 70, common.GetDate(), rgba)

			x, y = calculateStringXY(130, common.GetTime())
			drawString(x, y+50, 130, common.GetTime(), rgba)
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

/*
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
*/

// drawString will create a new string for image
func drawString(x, y int, fontSite float64, text string, rgba *image.RGBA) {
	pt := freetype.Pt(x, y)
	opts := opentype.FaceOptions{Size: fontSite, DPI: 72, Hinting: 0}
	fontFace, err := opentype.NewFace(lcd.sfntFont, &opts)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
		return
	}
	d := &font.Drawer{
		Dst:  rgba,
		Src:  image.NewUniform(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
		Face: fontFace, // Use the built-in font
		Dot:  pt,
	}
	d.DrawString(text)
}

// drawColorString will create a new string for image with color ability
func drawColorString(x, y int, fontSite float64, text string, rgba *image.RGBA, textColor rgb.Color) {
	pt := freetype.Pt(x, y)
	opts := opentype.FaceOptions{Size: fontSite, DPI: 72, Hinting: 0}
	fontFace, err := opentype.NewFace(lcd.sfntFont, &opts)
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
		return
	}
	d := &font.Drawer{
		Dst: rgba,
		Src: image.NewUniform(
			color.RGBA{
				R: uint8(textColor.Red),
				G: uint8(textColor.Green),
				B: uint8(textColor.Blue),
				A: 255,
			},
		),
		Face: fontFace, // Use the built-in font
		Dot:  pt,
	}
	d.DrawString(text)
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
