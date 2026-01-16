package audio

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"encoding/json"
	"os"
)

type Sink struct {
	Index  int    `json:"index"`
	Serial uint32 `json:"serial"`
	Name   string `json:"name"`
	Desc   string `json:"desc"`
}

type Audio struct {
	Enabled     bool   `json:"enabled"`
	SinkSerial  int    `json:"sinkSerial"`
	SinkName    string `json:"sinkName"`
	SinkDesc    string `json:"sinkDesc"`
	Latency     string `json:"latency"`
	MaxLatency  string `json:"maxLatency"`
	Rate        uint32 `json:"rate"`
	Channels    uint32 `json:"channels"`
	PollingRate uint32 `json:"pollingRate"`
	Debug       uint32 `json:"debug"`
	Frames      uint32 `json:"frames"`
}

var (
	location = ""
	audio    Audio
)

func Init() {
	location = config.GetConfig().ConfigPath + "/database/audio.json"
	if !common.FileExists(location) {
		logger.Log(logger.Fields{"file": location}).Info("Audio file is missing, creating initial one.")
		data := &Audio{
			Latency:     "128/48000",
			MaxLatency:  "256/48000",
			Rate:        48000,
			Channels:    2,
			PollingRate: 10,
			Debug:       0,
			Frames:      512,
		}
		if err := common.SaveJsonData(location, data); err != nil {
			logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to save audio data")
			return
		}
	}

	file, err := os.Open(location)
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err, "file": location}).Error("Failed to close file.")
		}
	}(file)

	if err != nil {
		logger.Log(logger.Fields{"error": err, "file": location}).Error("Failed to decode json")
	}
	if err = json.NewDecoder(file).Decode(&audio); err != nil {
		logger.Log(logger.Fields{"error": err, "file": location}).Error("Failed to decode json")
	}

	if audio.Enabled {
		//
	}
}

// StopAudio will stop audio server
func StopAudio() {

}

// StartAudio will start audio server
func StartAudio() {

}

// SetSink will set selected sink as physical output device
func SetSink() {

}

// SetBand will set band value
func SetBand(band int, value float32) {

}

// GetSinks will return list of available sinks
func GetSinks() []Sink {
	return []Sink{}
}

// GetCurrentSink will return data of currently selected Sink
func GetCurrentSink() interface{} {
	return nil
}

// UpdateAudio will update Audio settings
func UpdateAudio(settings *Audio) {
	audio = *settings
	if err := common.SaveJsonData(location, audio); err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to save audio data")
		return
	}
}

// GetAudio will return Audio
func GetAudio() *Audio {
	return &audio
}
