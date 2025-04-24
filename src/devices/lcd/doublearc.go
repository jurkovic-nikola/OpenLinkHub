package lcd

import (
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
	file, err := os.Open(profile)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to load double arc profile")
		return
	}
	if err = json.NewDecoder(file).Decode(&doubleRrc); err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to decode double arc profile")
		return
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
