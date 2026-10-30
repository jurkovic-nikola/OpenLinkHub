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
