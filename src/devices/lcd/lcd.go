package lcd

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/logger"
	"bytes"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	_ "golang.org/x/image/font"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Package: LCD Controller
// This is the primary package for LCD pump covers.
// All device actions are controlled from this package.
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

const (
	DisplayLiquid    uint8 = 0
	DisplayPump      uint8 = 1
	DisplayCPU       uint8 = 2
	DisplayGPU       uint8 = 3
	DisplayAllInOne  uint8 = 4
	DisplayLiquidCPU uint8 = 5
)

var (
	pwd, _       = os.Getwd()
	location     = pwd + "/static/img/lcd/"
	fontLocation = pwd + "/static/fonts/teko.ttf"
	mutex        sync.Mutex
)

type LCD struct {
	images map[uint8]image.Image
	font   *truetype.Font
}

var lcdConfiguration LCD

// Init will initialize LCD data
func Init() {
	lcdImages := make(map[uint8]image.Image, 0)

	// Read folder content
	files, err := os.ReadDir(location)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to read content of a folder")
	}

	// Load LCD images
	for _, fi := range files {
		if fi.IsDir() {
			continue // Exclude folders if any
		}

		// Define a full path of filename
		profileLocation := location + fi.Name()

		// Check if filename has .json extension
		if !common.IsValidExtension(profileLocation, ".jpg") {
			continue
		}

		// Open image
		file, e := os.Open(profileLocation)
		if e != nil {
			logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to open LCD image")
			continue
		}

		// Decode the image
		img, e := jpeg.Decode(file)
		if e != nil {
			logger.Log(logger.Fields{"error": e, "location": location}).Error("Unable to decode LCD image")
			e = file.Close()
			if e != nil {
				logger.Log(logger.Fields{"error": e, "location": location}).Error("Unable to close LCD image")
			}
			continue
		}

		fileInfo, e := file.Stat()
		if e != nil {
			logger.Log(logger.Fields{"error": e, "location": location}).Error("Unable to get LCD image fileinfo")
			e = file.Close()
			if e != nil {
				logger.Log(logger.Fields{"error": e, "location": location}).Error("Unable to close LCD image")
			}
			continue
		}

		imageType := strings.Split(fileInfo.Name(), ".")[0]
		switch imageType {
		case "cpu":
			lcdImages[DisplayCPU] = img
		case "gpu":
			lcdImages[DisplayGPU] = img
		case "liquid":
			lcdImages[DisplayLiquid] = img
		case "pump":
			lcdImages[DisplayPump] = img
		case "aio":
			lcdImages[DisplayAllInOne] = img
		case "liquid-cpu":
			lcdImages[DisplayLiquidCPU] = img
		}
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

	lcd := &LCD{
		images: lcdImages,
		font:   fontParsed,
	}
	lcdConfiguration = *lcd
}

// GenerateScreenImage will generate LCD screen image with given value
func GenerateScreenImage(imageType uint8, value, value1, value2 int) []byte {
	mutex.Lock()
	defer mutex.Unlock()
	if img, ok := lcdConfiguration.images[imageType]; ok {
		rgba := image.NewRGBA(img.Bounds())
		draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)

		c := freetype.NewContext()
		c.SetDPI(72)
		c.SetFont(lcdConfiguration.font)
		c.SetFontSize(220)
		c.SetClip(rgba.Bounds())
		c.SetDst(rgba)
		c.SetSrc(image.NewUniform(color.RGBA{R: 255, G: 255, B: 253, A: 255}))

		pt := freetype.Pt(135+int(c.PointToFixed(24)>>6), 280+int(c.PointToFixed(24)>>6))

		switch imageType {
		case DisplayLiquid, DisplayGPU, DisplayCPU:
			{
				pt = freetype.Pt(140+int(c.PointToFixed(24)>>6), 280+int(c.PointToFixed(24)>>6))
				_, err := c.DrawString(strconv.Itoa(value), pt)
				if err != nil {
					logger.Log(logger.Fields{"error": err}).Error("Unable to generate LCD image")
					return nil
				}
			}
		case DisplayPump:
			{
				pt = freetype.Pt(90+int(c.PointToFixed(24)>>6), 280+int(c.PointToFixed(24)>>6))
				_, err := c.DrawString(strconv.Itoa(value), pt)
				if err != nil {
					logger.Log(logger.Fields{"error": err}).Error("Unable to generate LCD image")
					return nil
				}
			}
		case DisplayAllInOne:
			{
				c.SetDPI(72)
				c.SetFont(lcdConfiguration.font)
				c.SetFontSize(100)
				c.SetClip(rgba.Bounds())
				c.SetDst(rgba)
				c.SetSrc(image.NewUniform(color.RGBA{R: 255, G: 255, B: 253, A: 255}))

				pt = freetype.Pt(120+int(c.PointToFixed(24)>>6), 200+int(c.PointToFixed(24)>>6))
				_, err := c.DrawString(strconv.Itoa(value), pt)
				if err != nil {
					logger.Log(logger.Fields{"error": err}).Error("Unable to generate LCD image")
					return nil
				}

				pt = freetype.Pt(270+int(c.PointToFixed(24)>>6), 200+int(c.PointToFixed(24)>>6))
				_, err = c.DrawString(strconv.Itoa(value1), pt)
				if err != nil {
					logger.Log(logger.Fields{"error": err}).Error("Unable to generate LCD image")
					return nil
				}

				pt = freetype.Pt(160+int(c.PointToFixed(24)>>6), 290+int(c.PointToFixed(24)>>6))
				_, err = c.DrawString(strconv.Itoa(value2), pt)
				if err != nil {
					logger.Log(logger.Fields{"error": err}).Error("Unable to generate LCD image")
					return nil
				}
			}
		case DisplayLiquidCPU:
			{
				c.SetDPI(100)
				c.SetFont(lcdConfiguration.font)
				c.SetFontSize(100)
				c.SetClip(rgba.Bounds())
				c.SetDst(rgba)
				c.SetSrc(image.NewUniform(color.RGBA{R: 255, G: 255, B: 253, A: 255}))

				pt = freetype.Pt(95+int(c.PointToFixed(24)>>6), 270+int(c.PointToFixed(24)>>6))
				_, err := c.DrawString(strconv.Itoa(value), pt)
				if err != nil {
					logger.Log(logger.Fields{"error": err}).Error("Unable to generate LCD image")
					return nil
				}

				pt = freetype.Pt(250+int(c.PointToFixed(24)>>6), 270+int(c.PointToFixed(24)>>6))
				_, err = c.DrawString(strconv.Itoa(value1), pt)
				if err != nil {
					logger.Log(logger.Fields{"error": err}).Error("Unable to generate LCD image")
					return nil
				}
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
	return nil
}
