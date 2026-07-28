package led

// Package: led
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"LumenForge/src/common"
	"LumenForge/src/config"
	"LumenForge/src/logger"
	"LumenForge/src/rgb"
	"encoding/json"
	"os"
	"path/filepath"
)

type Device struct {
	Serial     string             `json:"serial"`
	DeviceName string             `json:"deviceName"`
	Devices    map[int]DeviceData `json:"devices"`
}

type DeviceData struct {
	LedChannels uint8             `json:"ledChannels"`
	Pump        bool              `json:"pump"`
	AIO         bool              `json:"aio"`
	Fan         bool              `json:"fan"`
	Stand       bool              `json:"stand"`
	Tower       bool              `json:"tower"`
	Channels    map[int]rgb.Color `json:"channels"`
}

// LoadProfile loads device LED profile
func LoadProfile(serial string) *Device {
	profile := filepath.Join(config.GetPaths().MutableLEDRoot, serial+".json")
	file, err := os.Open(profile)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to load led profile")
		return nil
	}

	device := new(Device)
	if err = json.NewDecoder(file).Decode(&device); err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to decode led profile")
		return nil
	}
	return device
}

// SaveProfile saves device LED profile
func SaveProfile(serial string, data Device) {
	profile := filepath.Join(config.GetPaths().MutableLEDRoot, serial+".json")

	if err := common.SaveJsonData(profile, data); err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to save LED profile")
		return
	}
}
