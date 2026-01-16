package led

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"encoding/json"
	"fmt"
	"os"
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
	profile := fmt.Sprintf("%s/database/led/%s.json", config.GetConfig().ConfigPath, serial)
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
	profile := fmt.Sprintf("%s/database/led/%s.json", config.GetConfig().ConfigPath, serial)

	if err := common.SaveJsonData(profile, data); err != nil {
		logger.Log(logger.Fields{"error": err, "location": profile}).Error("Unable to save LED profile")
		return
	}
}
