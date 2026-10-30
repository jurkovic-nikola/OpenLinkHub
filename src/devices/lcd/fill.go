package lcd

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"encoding/json"
	"os"
)

type Fill struct {
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
	fill = new(Fill)
)

// InitFill will init Fill LCD mode
func InitFill() {
	fillProfile := config.GetConfig().ConfigPath + "/database/lcd/fill.json"
	if common.FileExists(fillProfile) {
		file, err := os.Open(fillProfile)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": fillProfile}).Error("Unable to load fill profile")
			return
		}
		defer file.Close()

		if err = json.NewDecoder(file).Decode(&fill); err != nil {
			logger.Log(logger.Fields{"error": err, "location": fillProfile}).Error("Unable to decode fill profile")
			return
		}
	} else {
		// Initial setup
		data := &Fill{
			Id:     104,
			Name:   "Fill",
			Sensor: 0,
			Margin: 10,
			Background: rgb.Color{
				Red:        20,
				Green:      12,
				Blue:       140,
				Brightness: 0,
				Hex:        "#140c8c",
			},
			StartColor: rgb.Color{
				Red:        255,
				Green:      220,
				Blue:       0,
				Brightness: 0,
				Hex:        "#ffdc00",
			},
			EndColor: rgb.Color{
				Red:        255,
				Green:      150,
				Blue:       0,
				Brightness: 0,
				Hex:        "#ff9600",
			},
			TextColor: rgb.Color{
				Red:        255,
				Green:      255,
				Blue:       253,
				Brightness: 0,
				Hex:        "#fffffd",
			},
		}
		fill = data
		if SaveFill(data) == 0 {
			logger.Log(logger.Fields{}).Warn("Unable to save fill profile. LCD will have default values")
		}
	}
}

// GetFill will return Fill object
func GetFill() *Fill {
	return fill
}

// SaveFill will save fill profile
func SaveFill(value *Fill) uint8 {
	fill = value
	profile := config.GetConfig().ConfigPath + "/database/lcd/fill.json"

	if err := common.SaveJsonData(profile, fill); err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to write lcd profile data")
		return 0
	}
	return 1
}
