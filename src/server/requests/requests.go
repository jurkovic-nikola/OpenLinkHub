package requests

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/inputmanager"
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
	DeviceId            string            `json:"deviceId"`
	ChannelId           int               `json:"channelId"`
	Mode                uint8             `json:"mode"`
	Rotation            uint8             `json:"rotation"`
	Value               uint16            `json:"value"`
	Color               rgb.Color         `json:"color"`
	StartColor          rgb.Color         `json:"startColor"`
	EndColor            rgb.Color         `json:"endColor"`
	Speed               float64           `json:"speed"`
	Smoothness          int               `json:"smoothness"`
	Profile             string            `json:"profile"`
	Label               string            `json:"label"`
	Static              bool              `json:"static"`
	Sensor              uint8             `json:"sensor"`
	HardwareLight       int               `json:"hardwareLight"`
	ZeroRpm             bool              `json:"zeroRpm"`
	Linear              bool              `json:"linear"`
	HwmonDeviceId       string            `json:"hwmonDeviceId"`
	Enabled             bool              `json:"enabled"`
	DeviceType          int               `json:"deviceType"`
	KeyOption           int               `json:"keyOption"`
	AreaOption          int               `json:"areaOption"`
	KeyId               int               `json:"keyId"`
	AreaId              int               `json:"areaId"`
	DeviceAmount        int               `json:"deviceAmount"`
	PortId              int               `json:"portId"`
	UserProfileName     string            `json:"userProfileName"`
	LcdSerial           string            `json:"lcdSerial"`
	KeyboardProfileName string            `json:"keyboardProfileName"`
	KeyboardLayout      string            `json:"keyboardLayout"`
	KeyboardControlDial int               `json:"keyboardControlDial"`
	SleepMode           int               `json:"sleepMode"`
	PollingRate         int               `json:"pollingRate"`
	ButtonOptimization  int               `json:"buttonOptimization"`
	AngleSnapping       int               `json:"angleSnapping"`
	PressAndHold        bool              `json:"pressAndHold"`
	KeyIndex            int               `json:"keyIndex"`
	KeyAssignmentType   uint8             `json:"keyAssignmentType"`
	KeyAssignmentValue  uint8             `json:"keyAssignmentValue"`
	MuteIndicator       int               `json:"muteIndicator"`
	RgbControl          bool              `json:"rgbControl"`
	RgbOff              string            `json:"rgbOff"`
	RgbOn               string            `json:"rgbOn"`
	Brightness          uint8             `json:"brightness"`
	Position            int               `json:"position"`
	DeviceIdString      string            `json:"deviceIdString"`
	Direction           int               `json:"direction"`
	StripId             int               `json:"stripId"`
	FanMode             int               `json:"fanMode"`
	New                 bool              `json:"new"`
	Stages              map[int]uint16    `json:"stages"`
	ColorDpi            rgb.Color         `json:"colorDpi"`
	ColorZones          map[int]rgb.Color `json:"colorZones"`
	Image               string            `json:"image"`
	Status              int
	Code                int
	Message             string
}

// ProcessDeleteTemperatureProfile will process deletion of temperature profile
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

// ProcessUpdateTemperatureProfile will process update of temperature profile
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

// ProcessNewTemperatureProfile will process the creation of temperature profile
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
	linear := req.Linear

	if static && linear {
		return &Payload{
			Message: "Unable to validate your request. Choose either Static or Linear option",
			Code:    http.StatusOK,
			Status:  0,
		}
	}

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

	if sensor > 5 || sensor < 0 {
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

	if temperatures.AddTemperatureProfile(profile, deviceId, static, zeroRpm, linear, sensor, channelId) {
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
		return &Payload{Message: "Manual flag in config.json is set to true", Code: http.StatusOK, Status: 0}
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

// ProcessUpdateRgbProfile will process POST request from a client for RGB profile update
func ProcessUpdateRgbProfile(r *http.Request) *Payload {
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

	// Device Id
	deviceId := req.DeviceId
	if len(deviceId) < 1 {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}
	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}
	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Profile name
	profile := req.Profile
	if len(profile) < 1 {
		return &Payload{Message: "Non-existing profile", Code: http.StatusOK, Status: 0}
	}
	if rgb.GetRgbProfile(profile) == nil {
		return &Payload{Message: "Non-existing profile", Code: http.StatusOK, Status: 0}
	}

	// Start color
	if req.StartColor.Red > 255 || req.StartColor.Green > 255 || req.StartColor.Blue > 255 {
		return &Payload{Message: "Invalid color selected", Code: http.StatusOK, Status: 0}
	}

	if req.StartColor.Red < 0 || req.StartColor.Green < 0 || req.StartColor.Blue < 0 {
		return &Payload{Message: "Invalid color selected", Code: http.StatusOK, Status: 0}
	}

	// End color
	if req.EndColor.Red > 255 || req.EndColor.Green > 255 || req.EndColor.Blue > 255 {
		return &Payload{Message: "Invalid color selected", Code: http.StatusOK, Status: 0}
	}

	if req.EndColor.Red < 0 || req.EndColor.Green < 0 || req.EndColor.Blue < 0 {
		return &Payload{Message: "Invalid color selected", Code: http.StatusOK, Status: 0}
	}

	// Speed
	if req.Speed < 1 || req.Speed > 10 {
		return &Payload{Message: "Invalid speed", Code: http.StatusOK, Status: 0}
	}

	startColor := req.StartColor
	startColor.Brightness = 1

	endColor := req.EndColor
	endColor.Brightness = 1

	rgbProfile := rgb.Profile{
		Speed:       req.Speed,
		Brightness:  1,
		StartColor:  startColor,
		MiddleColor: rgb.Color{},
		EndColor:    endColor,
		MinTemp:     0,
		MaxTemp:     0,
	}

	// Run it
	status := devices.UpdateRgbProfileData(deviceId, profile, rgbProfile)
	switch status {
	case 0:
		return &Payload{Message: "Unable to update RGB profile", Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: "RGB profile is successfully updated", Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: "Unable to update RGB profile", Code: http.StatusOK, Status: 0}
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

// ProcessLcdProfileChange will process POST request from a client for LCD mode change
func ProcessLcdProfileChange(r *http.Request) *Payload {
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

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	if len(req.Profile) < 0 {
		return &Payload{Message: "Invalid LCD mode", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateDeviceLcdProfile(req.DeviceId, req.Profile)
	switch status {
	case 1:
		return &Payload{Message: "LCD profile successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "You are already using selected LCD profile", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change lcd profile", Code: http.StatusOK, Status: 0}
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

// ProcessLcdImageChange will process POST request from a client for LCD image change
func ProcessLcdImageChange(r *http.Request) *Payload {
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

	if len(req.Image) == 0 {
		return &Payload{Message: "Invalid LCD image", Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.Image); !m {
		return &Payload{Message: "Invalid LCD image", Code: http.StatusOK, Status: 0}
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
	status := devices.UpdateDeviceLcdImage(req.DeviceId, req.ChannelId, req.Image)
	switch status {
	case 1:
		return &Payload{Message: "LCD image successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change LCD image", Code: http.StatusOK, Status: 0}
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
	return &Payload{Message: "Unable to save user profile", Code: http.StatusOK, Status: 0}
}

// ProcessSaveDeviceProfile will process PUT request from a client for device profile save
func ProcessSaveDeviceProfile(r *http.Request) *Payload {
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
	status := devices.SaveDeviceProfile(req.DeviceId, req.KeyboardProfileName, req.New)
	switch status {
	case 1:
		return &Payload{Message: "Keyboard profile successfully saved", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to save keyboard profile. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to save keyboard profile", Code: http.StatusOK, Status: 0}
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
		return &Payload{Message: "Keyboard layout successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change keyboard layout. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change keyboard layout", Code: http.StatusOK, Status: 0}
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
		return &Payload{Message: "Keyboard control dial successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change keyboard control dial. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change keyboard control dial", Code: http.StatusOK, Status: 0}
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
	status := devices.ChangeDeviceSleepMode(req.DeviceId, req.SleepMode)
	switch status {
	case 1:
		return &Payload{Message: "Device sleep mode successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change device sleep mode. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change device sleep mode", Code: http.StatusOK, Status: 0}
}

// ProcessChangePollingRate will process POST request from a client for device polling rate change
func ProcessChangePollingRate(r *http.Request) *Payload {
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

	if req.PollingRate < 1 || req.PollingRate > 10 {
		return &Payload{Message: "Invalid polling rate option", Code: http.StatusOK, Status: 0}
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
	status := devices.ChangeDevicePollingRate(req.DeviceId, req.PollingRate)
	switch status {
	case 1:
		return &Payload{Message: "Device polling rate successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change device polling rate. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change device polling rate", Code: http.StatusOK, Status: 0}
}

// ProcessChangeAngleSnapping will process POST request from a client for angle-snapping mode change
func ProcessChangeAngleSnapping(r *http.Request) *Payload {
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

	if req.AngleSnapping < 0 || req.AngleSnapping > 1 {
		return &Payload{Message: "Invalid angle snapping option", Code: http.StatusOK, Status: 0}
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
	status := devices.ChangeDeviceAngleSnapping(req.DeviceId, req.AngleSnapping)
	switch status {
	case 1:
		return &Payload{Message: "Device angle snapping mode successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change angle snapping mode. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change angle snapping mode", Code: http.StatusOK, Status: 0}
}

// ProcessChangeButtonOptimization will process POST request from a client for button optimization mode change
func ProcessChangeButtonOptimization(r *http.Request) *Payload {
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

	if req.ButtonOptimization < 0 || req.ButtonOptimization > 1 {
		return &Payload{Message: "Invalid angle snapping option", Code: http.StatusOK, Status: 0}
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
	status := devices.ChangeDeviceButtonOptimization(req.DeviceId, req.ButtonOptimization)
	switch status {
	case 1:
		return &Payload{Message: "Device button optimization mode successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change button optimization mode. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change button optimization mode", Code: http.StatusOK, Status: 0}
}

// ProcessChangeKeyAssignment will process POST request from a client for key assignment change
func ProcessChangeKeyAssignment(r *http.Request) *Payload {
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
	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}
	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	var keyAssignment = inputmanager.KeyAssignment{
		Name:          "",
		Default:       req.Enabled,
		ActionType:    req.KeyAssignmentType,
		ActionCommand: req.KeyAssignmentValue,
		ActionHold:    req.PressAndHold,
	}

	// Run it
	status := devices.ChangeDeviceKeyAssignment(req.DeviceId, req.KeyIndex, keyAssignment)
	switch status {
	case 1:
		return &Payload{Message: "Device key assignment successfully updated", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to update key assignment. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to update key assignment", Code: http.StatusOK, Status: 0}
}

// ProcessChangeMuteIndicator will process POST request from a client for device mute indicator change
func ProcessChangeMuteIndicator(r *http.Request) *Payload {
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

	if req.MuteIndicator < 0 || req.SleepMode > 1 {
		return &Payload{Message: "Invalid indicator option", Code: http.StatusOK, Status: 0}
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
	status := devices.ChangeDeviceMuteIndicator(req.DeviceId, req.MuteIndicator)
	switch status {
	case 1:
		return &Payload{Message: "Device mute indicator mode successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change device mute indicator mode. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change device mute indicator mode", Code: http.StatusOK, Status: 0}
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
		return &Payload{Message: "Keyboard profile successfully deleted", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to delete keyboard profile. Please try again", Code: http.StatusOK, Status: 0}
	case 3:
		return &Payload{Message: "Default keyboard profile can not be deleted", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to save keyboard profile", Code: http.StatusOK, Status: 0}
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
		return &Payload{Message: "Keyboard profile successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change keyboard profile. Please try again", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change keyboard profile", Code: http.StatusOK, Status: 0}
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
	return &Payload{Message: "Unable to change user profile", Code: http.StatusOK, Status: 0}
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
	return &Payload{Message: "Unable to change device brightness", Code: http.StatusOK, Status: 0}
}

// ProcessBrightnessChangeGradual will process POST request from a client for device brightness change via defined number from 0-100
func ProcessBrightnessChangeGradual(r *http.Request) *Payload {
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

	if req.Brightness < 0 || req.Brightness > 100 {
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
	status := devices.ChangeDeviceBrightnessGradual(req.DeviceId, req.Brightness)
	switch status {
	case 1:
		return &Payload{Message: "Device brightness successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Unable to change device brightness. You have exceeded maximum amount of LED channels per physical port", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change device brightness", Code: http.StatusOK, Status: 0}
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
	return &Payload{Message: "Unable to change device brightness", Code: http.StatusOK, Status: 0}
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
		return &Payload{Message: "Manual flag in config.json is not set to true", Code: http.StatusMethodNotAllowed, Status: 0}
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

	return &Payload{Message: "Unable to update device speed. Device is either unavailable or device does not have speed control", Code: http.StatusOK, Status: 0}
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

// ProcessHardwareChangeColor will process POST request from a client for hardware color change
func ProcessHardwareChangeColor(r *http.Request) *Payload {
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

	if req.HardwareLight < 0 || req.HardwareLight > 8 {
		return &Payload{Message: "Non-existing hardware profile", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateHardwareRgbProfile(req.DeviceId, req.HardwareLight)

	switch status {
	case 0:
		return &Payload{Message: "Unable to change device RGB profile", Code: http.StatusOK, Status: 0}
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
		return &Payload{Message: "Unable to change device color", Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: "Device color is successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Non-existing device type", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change device color", Code: http.StatusOK, Status: 0}
}

// ProcessMiscColor will process a POST request from a client for misc device color change
func ProcessMiscColor(r *http.Request) *Payload {
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

	if req.AreaId < 1 {
		return &Payload{Message: "Invalid area selected", Code: http.StatusOK, Status: 0}
	}

	if req.AreaOption < 0 || req.AreaOption > 2 {
		return &Payload{Message: "Invalid area option selected", Code: http.StatusOK, Status: 0}
	}

	status := devices.UpdateMiscColor(req.DeviceId, req.AreaId, req.AreaOption, req.Color)
	switch status {
	case 0:
		return &Payload{Message: "Unable to change device color", Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: "Device color is successfully changed", Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: "Non-existing device type", Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: "Unable to change device color", Code: http.StatusOK, Status: 0}
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
		return &Payload{Message: "You have exceeded maximum amount of supported LED channels", Code: http.StatusOK, Status: 0}
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
			return &Payload{Message: "Dashboard settings updated", Code: http.StatusOK, Status: 1}
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
		return &Payload{Message: "RGB scheduler successfully updated", Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: "Unable to change keyboard sleep mode", Code: http.StatusOK, Status: 0}
}

// ProcessPsuFanModeChange will process a POST request from a client for PSU fan mode change
func ProcessPsuFanModeChange(r *http.Request) *Payload {
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

	if req.FanMode < 0 || req.FanMode > 10 {
		return &Payload{Message: "Invalid fan mode selected", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdatePsuFanMode(req.DeviceId, req.FanMode)
	switch status {
	case 0:
		return &Payload{Message: "Unable to change PSU fan mode", Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: "PSU fan mode is successfully updated", Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: "Unable to change external LED hub device amount", Code: http.StatusOK, Status: 0}
}

// ProcessMouseDpiSave will process a POST request from a client for mouse DPI save
func ProcessMouseDpiSave(r *http.Request) *Payload {
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

	if len(req.Stages) == 0 {
		return &Payload{Message: "Invalid stages", Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: "Non-existing device", Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.SaveMouseDPI(req.DeviceId, req.Stages)
	switch status {
	case 0:
		return &Payload{Message: "Unable to save mouse DPI values", Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: "Mouse DPI values are successfully updated", Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: "Unable to save mouse DPI values", Code: http.StatusOK, Status: 0}
}

// ProcessMouseZoneColorsSave will process a POST request from a client for mouse zone colors save
func ProcessMouseZoneColorsSave(r *http.Request) *Payload {
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

	// Run it
	status := devices.SaveMouseZoneColors(req.DeviceId, req.ColorDpi, req.ColorZones)
	switch status {
	case 0:
		return &Payload{Message: "Unable to save mouse zone colors", Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: "Mouse zone colors are successfully updated", Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: "Unable to save mouse zone colors", Code: http.StatusOK, Status: 0}
}

// ProcessMouseDpiColorsSave will process a POST request from a client for mouse dpi colors save
func ProcessMouseDpiColorsSave(r *http.Request) *Payload {
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

	// Run it
	status := devices.SaveMouseDpiColors(req.DeviceId, req.ColorDpi, req.ColorZones)
	switch status {
	case 0:
		return &Payload{Message: "Unable to save mouse DPI colors", Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: "Mouse DPI colors are successfully updated", Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: "Unable to save mouse DPI colors", Code: http.StatusOK, Status: 0}
}

// ProcessHeadsetZoneColorsSave will process a POST request from a client for headset zone colors save
func ProcessHeadsetZoneColorsSave(r *http.Request) *Payload {
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

	// Run it
	status := devices.SaveHeadsetZoneColors(req.DeviceId, req.ColorZones)
	switch status {
	case 0:
		return &Payload{Message: "Unable to save mouse zone colors", Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: "Mouse zone colors are successfully updated", Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: "Unable to save mouse zone colors", Code: http.StatusOK, Status: 0}
}
