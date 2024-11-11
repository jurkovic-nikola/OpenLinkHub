package requests

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/scheduler"
	"OpenLinkHub/src/temperatures"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
)

// Payload contains data from a client about device speed change
type Payload struct {
	DeviceId            string    `json:"deviceId"`
	ChannelId           int       `json:"channelId"`
	Mode                uint8     `json:"mode"`
	Rotation            uint8     `json:"rotation"`
	Value               uint16    `json:"value"`
	Color               rgb.Color `json:"color"`
	Profile             string    `json:"profile"`
	Label               string    `json:"label"`
	Static              bool      `json:"static"`
	Sensor              uint8     `json:"sensor"`
	ZeroRpm             bool      `json:"zeroRpm"`
	HwmonDeviceId       string    `json:"hwmonDeviceId"`
	Enabled             bool      `json:"enabled"`
	DeviceType          int       `json:"deviceType"`
	KeyOption           int       `json:"keyOption"`
	KeyId               int       `json:"keyId"`
	DeviceAmount        int       `json:"deviceAmount"`
	PortId              int       `json:"portId"`
	UserProfileName     string    `json:"userProfileName"`
	LcdSerial           string    `json:"lcdSerial"`
	KeyboardProfileName string    `json:"keyboardProfileName"`
	KeyboardLayout      string    `json:"keyboardLayout"`
	KeyboardControlDial int       `json:"keyboardControlDial"`
	SleepMode           int       `json:"sleepMode"`
	RgbControl          bool      `json:"rgbControl"`
	RgbOff              string    `json:"rgbOff"`
	RgbOn               string    `json:"rgbOn"`
	Brightness          uint8     `json:"brightness"`
	Position            int       `json:"position"`
	Direction           int       `json:"direction"`
	StripId             int       `json:"stripId"`
	New                 bool      `json:"new"`
	Status              int
	Code                int
	Message             string
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

	if sensor > 4 || sensor < 0 {
		return &Payload{
			Message: "Unable to validate your request. Invalid sensor value",
			Code:    http.StatusOK,
			Status:  0,
		}
	}
	deviceId := ""
	channelId := 0
	if sensor == 3 {
		deviceId = req.HwmonDeviceId
	}

	if sensor == 4 {
		deviceId = req.DeviceId
		channelId = req.ChannelId

		if len(deviceId) < 1 {
			return &Payload{
				Message: "Unable to validate your request. Invalid sensor value",
				Code:    http.StatusOK,
				Status:  0,
			}
		}

		if channelId < 1 {
			return &Payload{
				Message: "Unable to validate your request. Invalid sensor value",
				Code:    http.StatusOK,
				Status:  0,
			}
		}
	}

	if temperatures.AddTemperatureProfile(profile, deviceId, static, zeroRpm, sensor, channelId) {
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
	case 3:
		return &Payload{Message: "Profile and device mismatch. Please try again", Code: http.StatusOK, Status: 0}
	case 4:
		return &Payload{Message: "Non-existing device specified in the profile. Please re-create profile", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to apply speed profile", Code: http.StatusOK, Status: 0}
}

// ProcessLcdChange will process POST request from a client for LCD mode change
func ProcessLcdChange(r *http.Request) *Payload {
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

	if req.Mode < 0 || req.Mode > 10 {
		return &Payload{Message: "Invalid LCD mode", Code: http.StatusOK, Status: 0}
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
	status := devices.UpdateDeviceLcd(req.DeviceId, req.ChannelId, req.Mode)
	switch status {
	case 1:
		return &Payload{Message: "LCD mode successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change LCD mode. Either LCD is offline or you do not have LCD", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change lcd mode", Code: http.StatusOK, Status: 0}
}

// ProcessLcdDeviceChange will process POST request from a client for LCD device change
func ProcessLcdDeviceChange(r *http.Request) *Payload {
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

	if len(req.DeviceId) < 0 {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.LcdSerial); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if req.ChannelId < -1 {
		return &Payload{Message: "Non-existing channelId", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeDeviceLcd(req.DeviceId, req.ChannelId, req.LcdSerial)
	switch status {
	case 1:
		return &Payload{Message: "LCD device successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change LCD device. Either LCD is offline or you do not have LCD", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change lcd device", Code: http.StatusOK, Status: 0}
}

// ProcessLcdRotationChange will process POST request from a client for LCD rotation change
func ProcessLcdRotationChange(r *http.Request) *Payload {
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

	if req.Rotation < 0 || req.Rotation > 3 {
		return &Payload{Message: "Invalid LCD rotation value", Code: http.StatusOK, Status: 0}
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
	status := devices.UpdateDeviceLcdRotation(req.DeviceId, req.ChannelId, req.Rotation)
	switch status {
	case 1:
		return &Payload{Message: "LCD rotation successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change LCD rotation. Either LCD is offline or you do not have LCD", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change lcd rotation", Code: http.StatusOK, Status: 0}
}

// ProcessSaveUserProfile will process PUT request from a client for device profile save
func ProcessSaveUserProfile(r *http.Request) *Payload {
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

	if len(req.UserProfileName) < 0 {
		return &Payload{Message: "Invalid profile name", Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.UserProfileName); !m {
		return &Payload{Message: "Profile name can contain only letters and numbers", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.SaveUserProfile(req.DeviceId, req.UserProfileName)
	switch status {
	case 1:
		return &Payload{Message: "User profile successfully saved", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to save user profile. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to save user profile.", Code: http.StatusOK, Status: 0}
}

// ProcessSaveKeyboardProfile will process PUT request from a client for device profile save
func ProcessSaveKeyboardProfile(r *http.Request) *Payload {
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

	if len(req.KeyboardProfileName) < 0 {
		return &Payload{Message: "Invalid profile name", Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.KeyboardProfileName); !m {
		return &Payload{Message: "Profile name can contain only letters and numbers", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.SaveKeyboardProfile(req.DeviceId, req.KeyboardProfileName, req.New)
	switch status {
	case 1:
		return &Payload{Message: "Keyboard profile successfully saved.", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to save keyboard profile. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to save keyboard profile.", Code: http.StatusOK, Status: 0}
}

// ProcessChangeKeyboardLayout will process POST request from a client for device layout change
func ProcessChangeKeyboardLayout(r *http.Request) *Payload {
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

	if len(req.KeyboardLayout) < 1 {
		return &Payload{Message: "Invalid profile name", Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.KeyboardLayout); !m {
		return &Payload{Message: "Profile name can contain only letters and numbers", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeKeyboardLayout(req.DeviceId, req.KeyboardLayout)
	switch status {
	case 1:
		return &Payload{Message: "Keyboard layout successfully changed.", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change keyboard layout. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change keyboard layout.", Code: http.StatusOK, Status: 0}
}

// ProcessChangeControlDial will process POST request from a client for device control dial change
func ProcessChangeControlDial(r *http.Request) *Payload {
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

	if req.KeyboardControlDial < 1 {
		return &Payload{Message: "Invalid control dial option", Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeKeyboardControlDial(req.DeviceId, req.KeyboardControlDial)
	switch status {
	case 1:
		return &Payload{Message: "Keyboard control dial successfully changed.", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change keyboard control dial. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change keyboard control dial.", Code: http.StatusOK, Status: 0}
}

// ProcessChangeSleepMode will process POST request from a client for device sleep change
func ProcessChangeSleepMode(r *http.Request) *Payload {
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

	if req.SleepMode < 1 || req.SleepMode > 60 {
		return &Payload{Message: "Invalid sleep option", Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeKeyboardSleepMode(req.DeviceId, req.SleepMode)
	switch status {
	case 1:
		return &Payload{Message: "Keyboard sleep mode successfully changed.", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change keyboard sleep mode. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change keyboard sleep mode.", Code: http.StatusOK, Status: 0}
}

// ProcessDeleteKeyboardProfile will process DELETE request from a client for device profile deletion
func ProcessDeleteKeyboardProfile(r *http.Request) *Payload {
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

	if len(req.KeyboardProfileName) < 0 {
		return &Payload{Message: "Invalid profile name", Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.KeyboardProfileName); !m {
		return &Payload{Message: "Profile name can contain only letters and numbers", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.DeleteKeyboardProfile(req.DeviceId, req.KeyboardProfileName)
	switch status {
	case 1:
		return &Payload{Message: "Keyboard profile successfully deleted.", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to delete keyboard profile. Please try again", Code: http.StatusOK, Status: 0}
	case 3:
		return &Payload{Message: "Default keyboard profile can not be deleted", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to save keyboard profile.", Code: http.StatusOK, Status: 0}
}

// ProcessChangeKeyboardProfile will process POST request from a client for keyboard profile change
func ProcessChangeKeyboardProfile(r *http.Request) *Payload {
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

	if len(req.KeyboardProfileName) < 0 {
		return &Payload{Message: "Invalid profile name", Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.KeyboardProfileName); !m {
		return &Payload{Message: "Profile name can contain only letters and numbers", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeKeyboardProfile(req.DeviceId, req.KeyboardProfileName)
	switch status {
	case 1:
		return &Payload{Message: "Keyboard profile successfully changed.", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change keyboard profile. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change keyboard profile.", Code: http.StatusOK, Status: 0}
}

// ProcessChangeUserProfile will process POST request from a client for device profile change
func ProcessChangeUserProfile(r *http.Request) *Payload {
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

	if len(req.UserProfileName) < 0 {
		return &Payload{Message: "Invalid profile name", Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.UserProfileName); !m {
		return &Payload{Message: "Profile name can contain only letters and numbers", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeUserProfile(req.DeviceId, req.UserProfileName)
	switch status {
	case 1:
		return &Payload{Message: "User profile successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change user profile. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change user profile.", Code: http.StatusOK, Status: 0}
}

// ProcessBrightnessChange will process POST request from a client for device brightness change
func ProcessBrightnessChange(r *http.Request) *Payload {
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

	if req.Brightness < 0 || req.Brightness > 4 {
		return &Payload{Message: "Invalid brightness value", Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeDeviceBrightness(req.DeviceId, req.Brightness)
	switch status {
	case 1:
		return &Payload{Message: "Device brightness successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change device brightness. You have exceeded maximum amount of LED channels per physical port", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change device brightness.", Code: http.StatusOK, Status: 0}
}

// ProcessPositionChange will process POST request from a client for device position change
func ProcessPositionChange(r *http.Request) *Payload {
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

	if len(req.DeviceId) < 0 {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if req.Direction < 0 || req.Direction > 1 {
		return &Payload{Message: "Non-existing direction", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateDevicePosition(req.DeviceId, req.Position, req.Direction)
	switch status {
	case 0:
		return &Payload{Message: "Unable to change device position. Invalid position selected", Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: "Device position successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change device position. Invalid position selected", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change device brightness.", Code: http.StatusOK, Status: 0}
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

	if req.DeviceType < 0 || req.DeviceType > 1 {
		return &Payload{Message: "Non-existing device type", Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if req.ChannelId < -1 {
		return &Payload{Message: "Non-existing channelId", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateDeviceLabel(req.DeviceId, req.ChannelId, req.Label, req.DeviceType)
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

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateRgbProfile(req.DeviceId, req.ChannelId, req.Profile)

	switch status {
	case 0:
		return &Payload{Message: "Unable to change device RGB profile", Code: http.StatusOK, Status: 0}
	case 2:
		return &Payload{Message: "Unable to change device RGB profile. This profile requires a pump or AIO", Code: http.StatusOK, Status: 0}
	case 3:
		return &Payload{Message: "Unable to change device RGB profile. This profile requires a keyboard device", Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: "Device RGB profile is successfully changed", Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: "Unable to change device RGB profile", Code: http.StatusOK, Status: 0}
}

// ProcessChangeStrip will process POST request from a client for RGB strip change
func ProcessChangeStrip(r *http.Request) *Payload {
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

	if req.StripId < 0 || req.StripId > 4 {
		return &Payload{Message: "Non-existing RGB strip", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateRgbStrip(req.DeviceId, req.ChannelId, req.StripId)

	switch status {
	case 0:
		return &Payload{Message: "Unable to change device RGB strip", Code: http.StatusOK, Status: 0}
	case 2:
		return &Payload{Message: "Unable to change device RGB strip. You need iCUE Link Adapter", Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: "Device RGB strip is successfully changed", Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: "Unable to change device RGB strip", Code: http.StatusOK, Status: 0}
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

// ProcessARGBDevice will process POST request from a client for ARGB 3-pin devices
func ProcessARGBDevice(r *http.Request) *Payload {
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
	if req.PortId < 0 || req.PortId > 5 {
		return &Payload{Message: "Non-existing LED Port-Id", Code: http.StatusOK, Status: 0}
	}

	status := devices.UpdateARGBDevice(req.DeviceId, req.PortId, req.DeviceType)
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

// ProcessKeyboardColor will process POST request from a client for keyboard color change
func ProcessKeyboardColor(r *http.Request) *Payload {
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

	if req.Color.Red > 255 || req.Color.Green > 255 || req.Color.Blue > 255 {
		return &Payload{Message: "Invalid color selected", Code: http.StatusOK, Status: 0}
	}

	if req.Color.Red < 0 || req.Color.Green < 0 || req.Color.Blue < 0 {
		return &Payload{Message: "Invalid color selected", Code: http.StatusOK, Status: 0}
	}

	if req.KeyId < 1 {
		return &Payload{Message: "Invalid key selected", Code: http.StatusOK, Status: 0}
	}

	if req.KeyOption < 0 || req.KeyOption > 2 {
		return &Payload{Message: "Invalid key option selected", Code: http.StatusOK, Status: 0}
	}

	status := devices.UpdateKeyboardColor(req.DeviceId, req.KeyId, req.KeyOption, req.Color)
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

// ProcessDashboardSettingsChange will process POST request from a client for dashboard settings change
func ProcessDashboardSettingsChange(r *http.Request) *Payload {
	req := &dashboard.Dashboard{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: "Unable to validate your request. Please try again!",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	status := dashboard.SaveDashboardSettings(req, true)
	switch status {
	case 0:
		return &Payload{Message: "Unable to save dashboard settings", Code: http.StatusOK, Status: 0}
	case 1:
		{
			return &Payload{Message: "Dashboard settings updated.", Code: http.StatusOK, Status: 1}
		}
	}
	return &Payload{Message: "Unable to save dashboard settings", Code: http.StatusOK, Status: 0}
}

// ProcessChangeRgbScheduler will process a POST request from a client for RGB scheduler change
func ProcessChangeRgbScheduler(r *http.Request) *Payload {
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

	// Run it
	status := scheduler.UpdateRgbSettings(req.RgbControl, req.RgbOff, req.RgbOn)
	switch status {
	case 1:
		return &Payload{Message: "RGB scheduler successfully updated.", Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: "Unable to change keyboard sleep mode.", Code: http.StatusOK, Status: 0}
}
