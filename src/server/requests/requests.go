package requests

import (
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/temperatures"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
)

// Payload contains data from a client about device speed change
type Payload struct {
	DeviceId     string    `json:"deviceId"`
	ChannelId    int       `json:"channelId"`
	Mode         uint8     `json:"mode"`
	Value        uint16    `json:"value"`
	Color        rgb.Color `json:"color"`
	Profile      string    `json:"profile"`
	Static       bool      `json:"static"`
	Sensor       uint8     `json:"sensor"`
	ZeroRpm      bool      `json:"zeroRpm"`
	Enabled      bool      `json:"enabled"`
	DeviceType   int       `json:"deviceType"`
	DeviceAmount int       `json:"deviceAmount"`
	Status       int
	Code         int
	Message      string
}

func ProcessDeleteTemperatureProfile(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: "Unable to validate your request. Please try again!",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	profile := req.Profile
	if len(profile) < 3 {
		return &Payload{
			Message: "Unable to validate your request. Profile name is less then 3 characters",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", profile); !m {
		return &Payload{
			Message: "Unable to validate your request. Profile name contains invalid characters",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if pf := temperatures.GetTemperatureProfile(profile); pf == nil {
		return &Payload{
			Message: "Non-existing speed profile",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	temperatures.DeleteTemperatureProfile(profile)
	devices.ResetSpeedProfiles(profile)
	return &Payload{
		Message: "Speed profile is successfully deleted",
		Code:    http.StatusOK,
		Status:  1,
	}
}

func ProcessUpdateTemperatureProfile(r *http.Request) *Payload {
	err := r.ParseForm()
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to parse form")
		return &Payload{
			Message: "Unable to validate your request. Please try again!",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	profile := r.FormValue("profile")
	data := r.FormValue("data")

	if len(profile) < 3 {
		return &Payload{
			Message: "Unable to validate your request. Profile name is less then 3 characters",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", profile); !m {
		return &Payload{
			Message: "Unable to validate your request. Profile name contains invalid characters",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if update := temperatures.UpdateTemperatureProfile(profile, data); update > 0 {
		st := ""
		if update == 1 {
			st = "profile"
		} else {
			st = "profiles"
		}
		return &Payload{
			Message: fmt.Sprintf("Succesfully updated %d %s", update, st),
			Code:    http.StatusOK,
			Status:  1,
		}
	} else {
		return &Payload{
			Message: "No profiles updated or there was no profile change",
			Code:    http.StatusOK,
			Status:  0,
		}
	}
}

func ProcessNewTemperatureProfile(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: "Unable to validate your request. Please try again!",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	profile := req.Profile
	static := req.Static
	sensor := req.Sensor
	zeroRpm := req.ZeroRpm

	if len(profile) < 3 {
		return &Payload{
			Message: "Unable to validate your request. Profile name is less then 3 characters",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", profile); !m {
		return &Payload{
			Message: "Unable to validate your request. Profile name contains invalid characters",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if sensor > 2 || sensor < 0 {
		return &Payload{
			Message: "Unable to validate your request. Invalid sensor value",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if temperatures.AddTemperatureProfile(profile, static, zeroRpm, sensor) {
		return &Payload{
			Message: "Speed profile is successfully saved",
			Code:    http.StatusOK,
			Status:  1,
		}
	} else {
		return &Payload{
			Message: "Unable to validate your request. Profile already exists",
			Code:    http.StatusOK,
			Status:  0,
		}
	}
}

// ProcessChangeSpeed will process POST request from a client for fan speed change
func ProcessChangeSpeed(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: "Unable to validate your request. Please try again!",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.Profile) < 1 {
		return &Payload{Message: "Non-existing speed profile", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.Profile); !m {
		return &Payload{Message: "Non-existing speed profile", Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 1 {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if req.ChannelId < -1 {
		return &Payload{Message: "Non-existing channelId", Code: http.StatusOK, Status: 0}
	}

	if temperatures.GetTemperatureProfile(req.Profile) == nil {
		return &Payload{Message: "Non-existing speed profile", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	devices.UpdateSpeedProfile(req.DeviceId, req.ChannelId, req.Profile)

	return &Payload{Message: "Device speed profile is successfully changed", Code: http.StatusOK, Status: 1}
}

// ProcessChangeColor will process POST request from a client for RGB profile change
func ProcessChangeColor(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: "Unable to validate your request. Please try again!",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.Profile) < 1 {
		return &Payload{Message: "Non-existing speed profile", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.Profile); !m {
		return &Payload{Message: "Non-existing RGB profile", Code: http.StatusOK, Status: 0}
	}

	if rgb.GetRgbProfile(req.Profile) == nil {
		return &Payload{Message: "Non-existing RGB profile", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	devices.UpdateRgbProfile(req.DeviceId, req.ChannelId, req.Profile)

	return &Payload{Message: "Device RGB profile is successfully changed", Code: http.StatusOK, Status: 1}
}
