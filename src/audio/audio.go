package audio

/*
#cgo pkg-config: libpipewire-0.3
#cgo LDFLAGS: -lm
#include <stdlib.h>
#include "audio.h"
*/
import "C"

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"unsafe"
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
	location    = ""
	audio       Audio
	mutex       sync.Mutex
	audioErrors = map[C.int]string{
		-1: "audio engine already running",
		-2: "invalid sample rate",
		-3: "invalid channel count",
		-4: "invalid polling rate (range: 1–50)",
		-5: "invalid latency range; please define latencies",
	}
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
		if config.IsSystemService() {
			audio.Enabled = false
			logger.Log(logger.Fields{}).Error("Virtual Audio is not available while service is running in system context")
			return
		}

		if err = setAudioConfig(audio.Frames, audio.Rate, audio.Channels, audio.PollingRate, audio.Debug); err != nil {
			audio.Enabled = false
			logger.Log(logger.Fields{"error": err}).Error("Unable to set audio config")
			return
		}

		go func() {
			rc := C.audio_engine_start()
			if rc != 0 {
				err := C.GoString(C.audio_engine_last_error())
				logger.Log(logger.Fields{"error": err}).Error("Unable to start virtual audio")
				return
			}
		}()
	}
}

// StopAudio will stop audio server
func StopAudio() {
	C.audio_engine_stop()
}

// StartAudio will start audio server
func StartAudio() {
	if config.IsSystemService() {
		audio.Enabled = false
		logger.Log(logger.Fields{}).Error("Virtual Audio is not available while service is running in system context")
		return
	}

	if err := setAudioConfig(audio.Frames, audio.Rate, audio.Channels, audio.PollingRate, audio.Debug); err != nil {
		audio.Enabled = false
		logger.Log(logger.Fields{"error": err}).Error("Unable to set audio config")
		return
	}

	go func() {
		rc := C.audio_engine_start()
		if rc != 0 {
			err := C.GoString(C.audio_engine_last_error())
			logger.Log(logger.Fields{"error": err}).Error("Unable to start virtual audio")
			return
		}
	}()
}

// SetBand will set band value
func SetBand(band int, value float64) {
	mutex.Lock()
	defer mutex.Unlock()

	if C.audio_engine_running() == 1 {
		C.audio_engine_band(C.int(band), C.float(value))
	}
}

// GetSinks will return list of available sinks
func GetSinks() []Sink {
	mutex.Lock()
	defer mutex.Unlock()

	sinks := make([]Sink, 0)
	if audio.Enabled {
		n := int(C.audio_engine_sink_count())
		selfName := C.GoString(C.audio_engine_self_sink_name())
		for i := 0; i < n; i++ {
			name := C.GoString(C.audio_engine_sink_name(C.int(i)))
			if name == selfName {
				continue
			}

			sinks = append(sinks, Sink{
				Index:  i,
				Serial: uint32(C.audio_engine_sink_serial(C.int(i))),
				Name:   name,
				Desc:   C.GoString(C.audio_engine_sink_desc(C.int(i))),
			})
		}
		return sinks
	}
	return sinks
}

// UpdateAudio will update Audio settings
func UpdateAudio(settings *Audio) uint8 {
	if config.IsSystemService() {
		return 0
	}
	mutex.Lock()
	audio = *settings
	if err := common.SaveJsonData(location, audio); err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to save audio data")
		return 0
	}
	mutex.Unlock()

	if !audio.Enabled {
		if C.audio_engine_running() == 1 {
			StopAudio()
		}
	} else {
		if C.audio_engine_running() == 0 {
			StartAudio()
		}
	}

	return 1
}

// UpdateTargetDevice will update Audio settings with target device
func UpdateTargetDevice(settings *Audio) uint8 {
	if config.IsSystemService() {
		return 0
	}

	if C.audio_engine_running() == 0 {
		return 0
	}

	mutex.Lock()
	audio = *settings
	if err := common.SaveJsonData(location, audio); err != nil {
		logger.Log(logger.Fields{"error": err, "location": location}).Error("Unable to save audio data")
		return 0
	}
	mutex.Unlock()

	if C.audio_engine_set_target_sink(C.uint(settings.SinkSerial)) != 0 {
		logger.Log(logger.Fields{}).Error("Target Device not available")
		return 0
	}
	return 1
}

// GetAudio will return Audio
func GetAudio() *Audio {
	return &audio
}

func cString(s string) *C.char {
	return C.CString(s)
}

// setAudioConfig sets default engine config
func setAudioConfig(frames, rate, channels, pollingRate, debug uint32) error {
	latency := cString(audio.Latency)
	maxLatency := cString(audio.MaxLatency)
	sinkName := cString(audio.SinkName)
	sinkDesc := cString(audio.SinkDesc)

	defer func() {
		C.free(unsafe.Pointer(latency))
		C.free(unsafe.Pointer(maxLatency))
		C.free(unsafe.Pointer(sinkName))
		C.free(unsafe.Pointer(sinkDesc))
	}()

	rc := C.audio_engine_config(
		C.uint(rate),
		C.uint(channels),
		C.uint(pollingRate),
		C.uint(debug),
		C.uint(frames),
		latency,
		maxLatency,
		sinkName,
		sinkDesc,
	)

	if rc == 0 {
		return nil
	}

	if msg, ok := audioErrors[rc]; ok {
		return fmt.Errorf(msg)
	}

	return fmt.Errorf("audio engine error (%d)", rc)
}
