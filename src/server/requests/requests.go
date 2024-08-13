package requests

import (
	"OpenLinkHub/src/config"
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
	Label        string    `json:"label"`
	Static       bool      `json:"static"`
	Sensor       uint8     `json:"sensor"`
	ZeroRpm      bool      `json:"zeroRpm"`
	Enabled      bool      `json:"enabled"`
	DeviceType   int       `json:"deviceType"`
	DeviceAmount int       `json:"deviceAmount"`
	PortId       int       `json:"portId"`
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

// ProcessChangeSpeed will process POST request from a client for fan/pump profile speed change
func ProcessChangeSpeed(r *http.Request) *Payload {
	req := &Payload{}
	if config.GetConfig().Manual {
		return &Payload{Message: "Manual flag in config.json is set to true.", Code: http.StatusOK, Status: 0}
	}

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
	status := devices.UpdateSpeedProfile(req.DeviceId, req.ChannelId, req.Profile)
	switch status {
	case 0:
		return &Payload{Message: "Unable to apply speed profile. Non-existing profile selected", Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: "Device speed profile is successfully applied", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Liquid temperature profile require pump device with temperature sensor", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to apply speed profile", Code: http.StatusOK, Status: 0}
}

// ProcessLabelChange will process POST request from a client for label change
func ProcessLabelChange(r *http.Request) *Payload {
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

	if len(req.Label) < 1 {
		return &Payload{Message: "Invalid label", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9#.:_ -]*$", req.Label); !m {
		return &Payload{Message: "Detected invalid characters in label", Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if req.ChannelId < -1 {
		return &Payload{Message: "Non-existing channelId", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateDeviceLabel(req.DeviceId, req.ChannelId, req.Label)
	switch status {
	case 0:
		return &Payload{Message: "Unable to apply new label. Please try again", Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: "Device label is successfully applied", Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: "Unable to apply speed profile", Code: http.StatusOK, Status: 0}
}

// ProcessManualChangeSpeed will process POST request from a client for fan/pump speed change
func ProcessManualChangeSpeed(r *http.Request) *Payload {
	req := &Payload{}
	if !config.GetConfig().Manual {
		return &Payload{Message: "Manual flag in config.json is not set to true.", Code: http.StatusMethodNotAllowed, Status: 0}
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: "Unable to validate your request. Please try again!",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.Value > 100 {
		req.Value = 100
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

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	if devices.UpdateManualSpeed(req.DeviceId, req.ChannelId, req.Value) == 1 {
		return &Payload{Message: "Device speed profile is successfully changed", Code: http.StatusOK, Status: 1}
	}

	return &Payload{Message: "Unable to update device speed. Device is either unavailable or device does not have speed control.", Code: http.StatusOK, Status: 0}
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

// ProcessExternalHubDeviceType will process POST request from a client for external-LED hub
func ProcessExternalHubDeviceType(r *http.Request) *Payload {
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

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}
	if req.PortId < 0 || req.PortId > 1 {
		return &Payload{Message: "Non-existing LED Port-Id", Code: http.StatusOK, Status: 0}
	}

	status := devices.UpdateExternalHubDeviceType(req.DeviceId, req.PortId, req.DeviceType)
	switch status {
	case 0:
		return &Payload{Message: "Unable to change external LED hub device", Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: "External LED hub device is successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Non-existing external device type", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change external LED hub device", Code: http.StatusOK, Status: 0}
}

// ProcessExternalHubDeviceAmount will process POST request from a client for external-LED hub
func ProcessExternalHubDeviceAmount(r *http.Request) *Payload {
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

	if req.DeviceAmount < 0 || req.DeviceAmount > 6 {
		return &Payload{Message: "Invalid amount of devices", Code: http.StatusOK, Status: 0}
	}
	if req.PortId < 0 || req.PortId > 1 {
		return &Payload{Message: "Non-existing LED Port-Id", Code: http.StatusOK, Status: 0}
	}
	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	status := devices.UpdateExternalHubDeviceAmount(req.DeviceId, req.PortId, req.DeviceAmount)
	switch status {
	case 0:
		return &Payload{Message: "Unable to change external LED hub device amount", Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: "External LED hub device amount is successfully updated", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "You have exceeded maximum amount of supported LED channels.", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change external LED hub device amount", Code: http.StatusOK, Status: 0}
}
