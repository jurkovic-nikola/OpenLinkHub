package lcd

import (
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
	file, err := os.Open(arcProfile)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": arcProfile}).Error("Unable to load arc profile")
		return
	}
	if err = json.NewDecoder(file).Decode(&arc); err != nil {
		logger.Log(logger.Fields{"error": err, "location": arcProfile}).Error("Unable to decode arc profile")
		return
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
	profile := config.GetConfig().ConfigPath + "/database/lcd/double-arc.json"

	buffer, err := json.MarshalIndent(arc, "", "    ")
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
		return 0
	}

	file, fileErr := os.Create(profile)
	if fileErr != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to create update arc profile")
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
