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
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	_ "golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"strconv"
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
	pwd, _       = os.Getwd()
	location     = pwd + "/static/img/lcd/background.jpg"
	fontLocation = pwd + "/static/fonts/teko.ttf"
	mutex        sync.Mutex
	imgWidth     = 480
	imgHeight    = 480
)

type LCD struct {
	image     image.Image
	font      *truetype.Font
	fontBytes []byte
	sfntFont  *opentype.Font
}

var lcd LCD

// Init will initialize LCD data
func Init() {
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
			opts := opentype.FaceOptions{Size: 130, DPI: 72, Hinting: 0}
			fontFace, err := opentype.NewFace(lcd.sfntFont, &opts)
			if err != nil {
				logger.Log(logger.Fields{"error": err}).Error("Unable to process font face")
			}

			bounds, _ := font.BoundString(fontFace, common.GetTime())
			textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
			textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

			x := (imgWidth - textWidth) / 2
			y := (imgHeight+textHeight)/2 - 10
			c = drawString(x, y, 130, c, common.GetTime())

			//c = drawString(85+int(c.PointToFixed(24)>>6), 250+int(c.PointToFixed(24)>>6), 130, c, common.GetTime())
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
