package lcd

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"encoding/json"
	"os"
)

type Dots struct {
	Id         int       `json:"id"`
	Name       string    `json:"name"`
	Sensor     uint8     `json:"sensor"`
	Margin     float64   `json:"margin"`
	Background rgb.Color `json:"background"`
	StartColor rgb.Color `json:"startColor"`
	EndColor   rgb.Color `json:"endColor"`
	TextColor  rgb.Color `json:"textColor"`
	ChannelId  int       `json:"channelId"`
	DeviceId   string    `json:"deviceId"`
}

var (
	dots = new(Dots)
)

// InitDots will init Dots LCD mode
func InitDots() {
	dotsProfile := config.GetConfig().ConfigPath + "/database/lcd/dots.json"
	if common.FileExists(dotsProfile) {
		file, err := os.Open(dotsProfile)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": dotsProfile}).Error("Unable to load dots profile")
			return
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				logger.Log(logger.Fields{"error": err, "location": dotsProfile}).Error("Unable to close file")
			}
		}(file)

		if err = json.NewDecoder(file).Decode(&dots); err != nil {
			logger.Log(logger.Fields{"error": err, "location": dotsProfile}).Error("Unable to decode dots profile")
			return
		}
	} else {
		// Initial setup
		data := &Dots{
			Id:     103,
			Name:   "Dots",
			Sensor: 0,
			Margin: 10,
			Background: rgb.Color{
				Red:        12,
				Green:      16,
				Blue:       30,
				Brightness: 0,
				Hex:        "#0c101e",
			},
			StartColor: rgb.Color{
				Red:        0,
				Green:      160,
				Blue:       255,
				Brightness: 0,
				Hex:        "#00a0ff",
			},
			EndColor: rgb.Color{
				Red:        120,
				Green:      255,
				Blue:       200,
				Brightness: 0,
				Hex:        "#78ffc8",
			},
			TextColor: rgb.Color{
				Red:        240,
				Green:      250,
				Blue:       255,
				Brightness: 0,
				Hex:        "#f0faff",
			},
		}
		dots = data
		if SaveDots(data) == 0 {
			logger.Log(logger.Fields{}).Warn("Unable to save dots profile. LCD will have default values")
		}
	}
}

// GetDots will return Dots object
func GetDots() *Dots {
	return dots
}

// SaveDots will save dots profile
func SaveDots(value *Dots) uint8 {
	dots = value
	profile := config.GetConfig().ConfigPath + "/database/lcd/dots.json"

	if err := common.SaveJsonData(profile, dots); err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to write lcd profile data")
		return 0
	}
	return 1
}
