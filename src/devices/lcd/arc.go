package lcd

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"encoding/json"
	"math"
	"os"
)

type Arc struct {
	Id          int       `json:"id"`
	Name        string    `json:"name"`
	Sensor      uint8     `json:"sensor"`
	Thickness   float64   `json:"thickness"`
	Margin      float64   `json:"margin"`
	MaxValue    int       `json:"maxValue"`
	GapRadians  float64   `json:"gapRadians"`
	Background  rgb.Color `json:"background"`
	BorderColor rgb.Color `json:"borderColor"`
	StartColor  rgb.Color `json:"startColor"`
	EndColor    rgb.Color `json:"endColor"`
	TextColor   rgb.Color `json:"textColor"`
}

type ArcModes struct {
	ArcType  int
	MaxValue int
}

var (
	arc        = new(Arc)
	startAngle = math.Pi / 2
)

// InitArc will init Arc LCD mode
func InitArc() {
	arcProfile := config.GetConfig().ConfigPath + "/database/lcd/arc.json"
	if common.FileExists(arcProfile) {
		file, err := os.Open(arcProfile)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": arcProfile}).Error("Unable to load arc profile")
			return
		}
		if err = json.NewDecoder(file).Decode(&arc); err != nil {
			logger.Log(logger.Fields{"error": err, "location": arcProfile}).Error("Unable to decode arc profile")
			return
		}
	} else {
		// Initial setup
		data := &Arc{
			Id:         100,
			Name:       "Arc",
			Sensor:     0,
			Thickness:  50,
			Margin:     20,
			MaxValue:   100,
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
		}
		arc = data
		if SaveArc(data) == 0 {
			logger.Log(logger.Fields{}).Warn("Unable to save arc profile. LCD will have default values")
		}
	}
	startAngle = (math.Pi / 2) + arc.GapRadians
}

// GetArc will return Arc object
func GetArc() *Arc {
	return arc
}

// SaveArc will save arc profile
func SaveArc(value *Arc) uint8 {
	arc = value
	profile := config.GetConfig().ConfigPath + "/database/lcd/arc.json"

	if err := common.SaveJsonData(profile, arc); err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to write lcd profile data")
		return 0
	}
	return 1
}
