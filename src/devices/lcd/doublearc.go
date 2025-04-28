package lcd

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"encoding/json"
	"os"
)

type DoubleArc struct {
	Id             int          `json:"id"`
	Name           string       `json:"name"`
	Thickness      float64      `json:"thickness"`
	Margin         float64      `json:"margin"`
	GapRadians     float64      `json:"gapRadians"`
	Background     rgb.Color    `json:"background"`
	BorderColor    rgb.Color    `json:"borderColor"`
	SeparatorColor rgb.Color    `json:"separatorColor"`
	Arcs           map[int]Arcs `json:"arcs"`
}

type Arcs struct {
	Name       string    `json:"name"`
	Sensor     uint8     `json:"sensor"`
	StartColor rgb.Color `json:"startColor"`
	EndColor   rgb.Color `json:"endColor"`
	TextColor  rgb.Color `json:"textColor"`
}

var (
	doubleRrc = new(DoubleArc)
)

// InitDoubleArc will init Double Arc LCD mode
func InitDoubleArc() {
	profile := config.GetConfig().ConfigPath + "/database/lcd/double-arc.json"
	if common.FileExists(profile) {
		file, err := os.Open(profile)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to load double arc profile")
			return
		}
		if err = json.NewDecoder(file).Decode(&doubleRrc); err != nil {
			logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to decode double arc profile")
			return
		}
	} else {
		// Initial setup
		data := &DoubleArc{
			Id:         101,
			Name:       "Double Arc",
			Thickness:  50,
			Margin:     20,
			GapRadians: 0.5,
			Background: rgb.Color{
				Red:        24,
				Green:      24,
				Blue:       24,
				Brightness: 0,
				Hex:        "#181818",
			},
			BorderColor: rgb.Color{
				Red:        64,
				Green:      64,
				Blue:       64,
				Brightness: 0,
				Hex:        "#404040",
			},
			SeparatorColor: rgb.Color{
				Red:        160,
				Green:      160,
				Blue:       160,
				Brightness: 0,
				Hex:        "#a0a0a0",
			},
			Arcs: map[int]Arcs{
				0: {
					Name:   "Left Arc",
					Sensor: 0,
					StartColor: rgb.Color{
						Red:        0,
						Green:      128,
						Blue:       255,
						Brightness: 0,
						Hex:        "#0080ff",
					},
					EndColor: rgb.Color{
						Red:        0,
						Green:      255,
						Blue:       255,
						Brightness: 0,
						Hex:        "#00ffff",
					},
					TextColor: rgb.Color{
						Red:        140,
						Green:      220,
						Blue:       255,
						Brightness: 0,
						Hex:        "#8cdcff",
					},
				},
				1: {
					Name:   "Right Arc",
					Sensor: 2,
					StartColor: rgb.Color{
						Red:        255,
						Green:      128,
						Blue:       0,
						Brightness: 0,
						Hex:        "#ff8000",
					},
					EndColor: rgb.Color{
						Red:        255,
						Green:      178,
						Blue:       102,
						Brightness: 0,
						Hex:        "#ffb266",
					},
					TextColor: rgb.Color{
						Red:        255,
						Green:      218,
						Blue:       84,
						Brightness: 0,
						Hex:        "#ffda54",
					},
				},
			},
		}
		doubleRrc = data
		if SaveDoubleArc(data) == 0 {
			logger.Log(logger.Fields{}).Warn("Unable to save double arc profile. LCD will have default values")
		}
	}

}

// GetDoubleArc will return DoubleArc object
func GetDoubleArc() *DoubleArc {
	return doubleRrc
}

// SaveDoubleArc will save double arc profile
func SaveDoubleArc(value *DoubleArc) uint8 {
	doubleRrc = value
	profile := config.GetConfig().ConfigPath + "/database/lcd/double-arc.json"

	buffer, err := json.MarshalIndent(doubleRrc, "", "    ")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
		return 0
	}

	file, fileErr := os.Create(profile)
	if fileErr != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to create update double arc profile")
		return 0
	}

	_, err = file.Write(buffer)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to write data")
		return 0
	}

	err = file.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to close file handle")
	}
	return 1
}
