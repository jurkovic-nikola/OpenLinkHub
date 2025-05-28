package requests

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/devices/lcd"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/language"
	"OpenLinkHub/src/led"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/macro"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/scheduler"
	"OpenLinkHub/src/temperatures"
	"encoding/json"
	"net/http"
	"regexp"
)

// Payload contains data from a client about device speed change
type Payload struct {
	DeviceId            string               `json:"deviceId"`
	ChannelId           int                  `json:"channelId"`
	ProfileId           uint8                `json:"profileId"`
	Mode                uint8                `json:"mode"`
	Rotation            uint8                `json:"rotation"`
	Value               uint16               `json:"value"`
	BackgroundColor     rgb.Color            `json:"backgroundColor"`
	BorderColor         rgb.Color            `json:"borderColor"`
	SeparatorColor      rgb.Color            `json:"separatorColor"`
	Color               rgb.Color            `json:"color"`
	StartColor          rgb.Color            `json:"startColor"`
	EndColor            rgb.Color            `json:"endColor"`
	TextColor           rgb.Color            `json:"textColor"`
	Arcs                map[uint8]lcd.Arcs   `json:"arcs"`
	Speed               float64              `json:"speed"`
	Thickness           float64              `json:"thickness"`
	GapRadians          float64              `json:"gapRadians"`
	Margin              float64              `json:"margin"`
	Smoothness          int                  `json:"smoothness"`
	Profile             string               `json:"profile"`
	Label               string               `json:"label"`
	Static              bool                 `json:"static"`
	Sensor              uint8                `json:"sensor"`
	HardwareLight       int                  `json:"hardwareLight"`
	ZeroRpm             bool                 `json:"zeroRpm"`
	Linear              bool                 `json:"linear"`
	HwmonDeviceId       string               `json:"hwmonDeviceId"`
	Enabled             bool                 `json:"enabled"`
	DeviceType          int                  `json:"deviceType"`
	KeyOption           int                  `json:"keyOption"`
	AreaOption          int                  `json:"areaOption"`
	KeyId               int                  `json:"keyId"`
	AreaId              int                  `json:"areaId"`
	DeviceAmount        int                  `json:"deviceAmount"`
	PortId              int                  `json:"portId"`
	UserProfileName     string               `json:"userProfileName"`
	LcdSerial           string               `json:"lcdSerial"`
	KeyboardProfileName string               `json:"keyboardProfileName"`
	KeyboardLayout      string               `json:"keyboardLayout"`
	KeyboardControlDial int                  `json:"keyboardControlDial"`
	SleepMode           int                  `json:"sleepMode"`
	PollingRate         int                  `json:"pollingRate"`
	ButtonOptimization  int                  `json:"buttonOptimization"`
	AngleSnapping       int                  `json:"angleSnapping"`
	PressAndHold        bool                 `json:"pressAndHold"`
	KeyIndex            int                  `json:"keyIndex"`
	KeyAssignmentType   uint8                `json:"keyAssignmentType"`
	KeyAssignmentValue  uint16               `json:"keyAssignmentValue"`
	MuteIndicator       int                  `json:"muteIndicator"`
	RgbControl          bool                 `json:"rgbControl"`
	RgbOff              string               `json:"rgbOff"`
	RgbOn               string               `json:"rgbOn"`
	Brightness          uint8                `json:"brightness"`
	Position            int                  `json:"position"`
	DeviceIdString      string               `json:"deviceIdString"`
	Direction           int                  `json:"direction"`
	StripId             int                  `json:"stripId"`
	FanMode             int                  `json:"fanMode"`
	New                 bool                 `json:"new"`
	Stages              map[int]uint16       `json:"stages"`
	ColorDpi            rgb.Color            `json:"colorDpi"`
	ColorZones          map[int]rgb.Color    `json:"colorZones"`
	Image               string               `json:"image"`
	MacroId             int                  `json:"macroId"`
	MacroIndex          int                  `json:"macroIndex"`
	MacroName           string               `json:"macroName"`
	MacroType           uint8                `json:"macroType"`
	MacroValue          uint16               `json:"macroValue"`
	MacroDelay          uint16               `json:"macroDelay"`
	LedProfile          led.Device           `json:"ledProfile"`
	Points              []temperatures.Point `json:"points"`
	UpdateType          uint8                `json:"updateType"`
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
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	profile := req.Profile
	if len(profile) < 3 {
		return &Payload{
			Message: language.GetValue("txtProfileNameTooShort"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", profile); !m {
		return &Payload{
			Message: language.GetValue("txtProfileInvalidName"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if pf := temperatures.GetTemperatureProfile(profile); pf == nil {
		return &Payload{
			Message: language.GetValue("txtNonExistingSpeedProfile"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	temperatures.DeleteTemperatureProfile(profile)
	devices.ResetSpeedProfiles(profile)
	return &Payload{
		Message: language.GetValue("txtSpeedProfileDeleted"),
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
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	profile := r.FormValue("profile")
	data := r.FormValue("data")

	if len(profile) < 3 {
		return &Payload{
			Message: language.GetValue("txtProfileNameTooShort"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", profile); !m {
		return &Payload{
			Message: language.GetValue("txtProfileInvalidName"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if update := temperatures.UpdateTemperatureProfile(profile, data); update > 0 {
		return &Payload{
			Message: language.GetValue("txtSpeedProfileUpdated"),
			Code:    http.StatusOK,
			Status:  1,
		}
	} else {
		return &Payload{
			Message: language.GetValue("txtSpeedProfileNoChange"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}
}

// ProcessUpdateTemperatureProfileGraph will process update of temperature profile graph
func ProcessUpdateTemperatureProfileGraph(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	profile := req.Profile
	updateType := req.UpdateType

	if updateType < 0 || updateType > 1 {
		return &Payload{
			Message: language.GetValue("txtInvalidUpdateType"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", profile); !m {
		return &Payload{
			Message: language.GetValue("txtProfileInvalidName"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	pf := temperatures.GetTemperatureProfile(profile)
	if pf == nil {
		return &Payload{Message: language.GetValue("txtNonExistingSpeedProfile"), Code: http.StatusOK, Status: 0}
	}

	pf.Points[updateType] = req.Points
	if temperatures.UpdateTemperatureProfileGraph(profile, *pf) == 1 {
		return &Payload{Message: language.GetValue("txtSpeedProfileUpdated"), Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: language.GetValue("txtSpeedProfileNotUpdated"), Code: http.StatusOK, Status: 0}
}

// ProcessNewTemperatureProfile will process the creation of temperature profile
func ProcessNewTemperatureProfile(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
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
			Message: language.GetValue("txtStaticOrLinear"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(profile) < 3 {
		return &Payload{
			Message: language.GetValue("txtProfileNameTooShort"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", profile); !m {
		return &Payload{
			Message: language.GetValue("txtProfileInvalidName"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if sensor > 6 || sensor < 0 {
		return &Payload{
			Message: language.GetValue("txtInvalidSensorValue"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}
	deviceId := ""
	channelId := 0
	if sensor == 3 {
		deviceId = req.HwmonDeviceId
	}

	if sensor == 4 || sensor == 6 {
		deviceId = req.DeviceId
		channelId = req.ChannelId

		if len(deviceId) < 1 {
			return &Payload{
				Message: language.GetValue("txtInvalidSensorValue"),
				Code:    http.StatusOK,
				Status:  0,
			}
		}

		if channelId < 1 {
			return &Payload{
				Message: language.GetValue("txtInvalidSensorValue"),
				Code:    http.StatusOK,
				Status:  0,
			}
		}
	}

	if temperatures.AddTemperatureProfile(profile, deviceId, static, zeroRpm, linear, sensor, channelId) {
		return &Payload{
			Message: language.GetValue("txtSpeedProfileSaved"),
			Code:    http.StatusOK,
			Status:  1,
		}
	} else {
		return &Payload{
			Message: language.GetValue("txtSpeedProfileExists"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}
}

// ProcessChangeSpeed will process POST request from a client for fan/pump profile speed change
func ProcessChangeSpeed(r *http.Request) *Payload {
	req := &Payload{}
	if config.GetConfig().Manual {
		return &Payload{Message: language.GetValue("txtManualFlag"), Code: http.StatusOK, Status: 0}
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.Profile) < 1 {
		return &Payload{Message: language.GetValue("txtNonExistingSpeedProfile"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.Profile); !m {
		return &Payload{Message: language.GetValue("txtNonExistingSpeedProfile"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 1 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if req.ChannelId < -1 {
		return &Payload{Message: language.GetValue("txtNonExistingChannelId"), Code: http.StatusOK, Status: 0}
	}

	if temperatures.GetTemperatureProfile(req.Profile) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingSpeedProfile"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateSpeedProfile(req.DeviceId, req.ChannelId, req.Profile)
	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtNonExistingSpeedProfileSelected"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtDeviceSpeedProfileUpdated"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtSpeedProfileNoPump"), Code: http.StatusOK, Status: 0}
	case 3:
		return &Payload{Message: language.GetValue("txtDeviceProfileMismatch"), Code: http.StatusOK, Status: 0}
	case 4:
		return &Payload{Message: language.GetValue("txtSpeedProfileNonExistingDevice"), Code: http.StatusOK, Status: 0}
	case 5:
		return &Payload{Message: language.GetValue("txtSpeedProfileNoTemperatureData"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToApplySpeedProfile"), Code: http.StatusOK, Status: 0}
}

// ProcessUpdateRgbProfile will process POST request from a client for RGB profile update
func ProcessUpdateRgbProfile(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	// Device Id
	deviceId := req.DeviceId
	if len(deviceId) < 1 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}
	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}
	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Profile name
	profile := req.Profile
	if len(profile) < 1 {
		return &Payload{Message: language.GetValue("txtNonExistingProfile"), Code: http.StatusOK, Status: 0}
	}
	if rgb.GetRgbProfile(profile) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingProfile"), Code: http.StatusOK, Status: 0}
	}

	// Start color
	if req.StartColor.Red > 255 || req.StartColor.Green > 255 || req.StartColor.Blue > 255 {
		return &Payload{Message: language.GetValue("txtInvalidColorSelected"), Code: http.StatusOK, Status: 0}
	}

	if req.StartColor.Red < 0 || req.StartColor.Green < 0 || req.StartColor.Blue < 0 {
		return &Payload{Message: language.GetValue("txtInvalidColorSelected"), Code: http.StatusOK, Status: 0}
	}

	// End color
	if req.EndColor.Red > 255 || req.EndColor.Green > 255 || req.EndColor.Blue > 255 {
		return &Payload{Message: language.GetValue("txtInvalidColorSelected"), Code: http.StatusOK, Status: 0}
	}

	if req.EndColor.Red < 0 || req.EndColor.Green < 0 || req.EndColor.Blue < 0 {
		return &Payload{Message: language.GetValue("txtInvalidColorSelected"), Code: http.StatusOK, Status: 0}
	}

	// Speed
	if req.Speed < 1 || req.Speed > 10 {
		return &Payload{Message: language.GetValue("txtInvalidSpeed"), Code: http.StatusOK, Status: 0}
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
		return &Payload{Message: language.GetValue("txtRgbProfileNotUpdated"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtRgbProfileUpdated"), Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: language.GetValue("txtRgbProfileNotUpdated"), Code: http.StatusOK, Status: 0}
}

// ProcessLcdChange will process POST request from a client for LCD mode change
func ProcessLcdChange(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.Mode < 0 {
		return &Payload{Message: language.GetValue("txtInvalidLcdMode"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if req.ChannelId < -1 {
		return &Payload{Message: language.GetValue("txtNonExistingChannelId"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateDeviceLcd(req.DeviceId, req.ChannelId, req.Mode)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtLcdModeChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtInvalidLcdModeDevice"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeLcdMode"), Code: http.StatusOK, Status: 0}
}

// ProcessLcdProfileChange will process POST request from a client for LCD mode change
func ProcessLcdProfileChange(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if len(req.Profile) < 0 {
		return &Payload{Message: language.GetValue("txtInvalidLcdMode"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateDeviceLcdProfile(req.DeviceId, req.Profile)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtLcdProfileChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtSameLcdProfile"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeLcdProfile"), Code: http.StatusOK, Status: 0}
}

// ProcessLcdDeviceChange will process POST request from a client for LCD device change
func ProcessLcdDeviceChange(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.LcdSerial); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if req.ChannelId < -1 {
		return &Payload{Message: language.GetValue("txtNonExistingChannelId"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeDeviceLcd(req.DeviceId, req.ChannelId, req.LcdSerial)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtLcdDeviceChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtInvalidLcdDevice"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeLcdDevice"), Code: http.StatusOK, Status: 0}
}

// ProcessLcdRotationChange will process POST request from a client for LCD rotation change
func ProcessLcdRotationChange(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.Rotation < 0 || req.Rotation > 3 {
		return &Payload{Message: language.GetValue("txtInvalidLcdRotation"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if req.ChannelId < -1 {
		return &Payload{Message: language.GetValue("txtNonExistingChannelId"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateDeviceLcdRotation(req.DeviceId, req.ChannelId, req.Rotation)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtLcdRotationChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtInvalidLcdRotationDevice"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeLcdRotation"), Code: http.StatusOK, Status: 0}
}

// ProcessLcdImageChange will process POST request from a client for LCD image change
func ProcessLcdImageChange(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.Image) == 0 {
		return &Payload{Message: language.GetValue("txtInvalidLcdImage"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.Image); !m {
		return &Payload{Message: language.GetValue("txtInvalidLcdImage"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if req.ChannelId < -1 {
		return &Payload{Message: language.GetValue("txtNonExistingChannelId"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateDeviceLcdImage(req.DeviceId, req.ChannelId, req.Image)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtLcdImageChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableToChangeLcdImage"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeLcdImage"), Code: http.StatusOK, Status: 0}
}

// ProcessLcdProfileUpdate will process POST request from a client for LCD profile update
func ProcessLcdProfileUpdate(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	profileId := req.ProfileId
	if profileId < 100 {
		return &Payload{Message: language.GetValue("txtNonExistingLcdProfile"), Code: http.StatusOK, Status: 0}
	}

	thickness := req.Thickness
	if thickness < 10 || thickness > 50 {
		return &Payload{Message: language.GetValue("txtInvalidThickness"), Code: http.StatusOK, Status: 0}
	}

	margin := req.Margin
	if margin < 10 || thickness > 50 {
		return &Payload{Message: language.GetValue("txtInvalidMarginValue"), Code: http.StatusOK, Status: 0}
	}

	var status uint8 = 0

	switch profileId {
	case lcd.DisplayArc:
		sensorId := req.Sensor
		if sensorId < 0 || sensorId > 5 {
			return &Payload{Message: language.GetValue("txtNonExistingSensorId"), Code: http.StatusOK, Status: 0}
		}

		// Profile
		mode := lcd.GetArc()
		mode.Sensor = sensorId
		mode.Margin = margin
		mode.Thickness = thickness
		mode.GapRadians = req.GapRadians
		mode.Background = req.BackgroundColor
		mode.BorderColor = req.BorderColor
		mode.StartColor = req.StartColor
		mode.EndColor = req.EndColor
		mode.TextColor = req.TextColor

		// Send it
		status = lcd.SaveArc(mode)
		break
	case lcd.DisplayDoubleArc:
		// Colors
		background := req.BackgroundColor
		borderColor := req.BorderColor
		separatorColor := req.SeparatorColor

		mode := lcd.GetDoubleArc()
		for i := 0; i < 2; i++ {
			arc := req.Arcs[uint8(i)]
			sensorId := arc.Sensor
			if sensorId < 0 || sensorId > 5 {
				return &Payload{Message: language.GetValue("txtNonExistingSensorId"), Code: http.StatusOK, Status: 0}
			}
			arcName := mode.Arcs[i].Name

			mode.Arcs[i] = lcd.Arcs{
				Name:       arcName,
				Sensor:     sensorId,
				StartColor: arc.StartColor,
				EndColor:   arc.EndColor,
				TextColor:  arc.TextColor,
			}
		}

		mode.Margin = margin
		mode.GapRadians = req.GapRadians
		mode.Thickness = thickness
		mode.Background = background
		mode.BorderColor = borderColor
		mode.SeparatorColor = separatorColor

		// Send it
		status = lcd.SaveDoubleArc(mode)
		break
	default:
		status = 0
		break
	}

	if status == 1 {
		return &Payload{Message: language.GetValue("txtLcdProfileUpdated"), Code: http.StatusOK, Status: 1}
	} else {
		return &Payload{Message: language.GetValue("txtUnableToUpdateLcdProfile"), Code: http.StatusOK, Status: 0}
	}
}

// ProcessSaveUserProfile will process PUT request from a client for device profile save
func ProcessSaveUserProfile(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.UserProfileName) < 0 {
		return &Payload{Message: language.GetValue("txtInvalidProfileName"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.UserProfileName); !m {
		return &Payload{Message: language.GetValue("txtProfileOnlyLettersNumbers"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.SaveUserProfile(req.DeviceId, req.UserProfileName)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtUserProfileSaved"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableToSaveUserProfile"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToSaveUserProfile"), Code: http.StatusOK, Status: 0}
}

// ProcessSaveDeviceProfile will process PUT request from a client for device profile save
func ProcessSaveDeviceProfile(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.KeyboardProfileName) < 0 {
		return &Payload{Message: language.GetValue("txtInvalidProfileName"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.KeyboardProfileName); !m {
		return &Payload{Message: language.GetValue("txtProfileOnlyLettersNumbers"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.SaveDeviceProfile(req.DeviceId, req.KeyboardProfileName, req.New)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtKeyboardProfileSaved"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableToSaveKeyboardProfile"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToSaveKeyboardProfile"), Code: http.StatusOK, Status: 0}
}

// ProcessChangeKeyboardLayout will process POST request from a client for device layout change
func ProcessChangeKeyboardLayout(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.KeyboardLayout) < 1 {
		return &Payload{Message: language.GetValue("txtInvalidProfileName"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.KeyboardLayout); !m {
		return &Payload{Message: language.GetValue("txtProfileOnlyLettersNumbers"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeKeyboardLayout(req.DeviceId, req.KeyboardLayout)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtKeyboardLayoutChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableToChangeKeyboardLayout"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeKeyboardLayout"), Code: http.StatusOK, Status: 0}
}

// ProcessChangeControlDial will process POST request from a client for device control dial change
func ProcessChangeControlDial(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.KeyboardControlDial < 1 {
		return &Payload{Message: language.GetValue("txtInvalidControlDial"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeKeyboardControlDial(req.DeviceId, req.KeyboardControlDial)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtControlDialChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableToChangeControlDial"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeControlDial"), Code: http.StatusOK, Status: 0}
}

// ProcessChangeSleepMode will process POST request from a client for device sleep change
func ProcessChangeSleepMode(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.SleepMode < 1 || req.SleepMode > 60 {
		return &Payload{Message: language.GetValue("txtInvalidSleepOption"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeDeviceSleepMode(req.DeviceId, req.SleepMode)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtSleepModeChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableToChangeSleepMode"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeSleepMode"), Code: http.StatusOK, Status: 0}
}

// ProcessChangePollingRate will process POST request from a client for device polling rate change
func ProcessChangePollingRate(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.PollingRate < 1 || req.PollingRate > 10 {
		return &Payload{Message: language.GetValue("txtInvalidPollingRate"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeDevicePollingRate(req.DeviceId, req.PollingRate)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtPollingRateChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableToChangePollingRate"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangePollingRate"), Code: http.StatusOK, Status: 0}
}

// ProcessChangeAngleSnapping will process POST request from a client for angle-snapping mode change
func ProcessChangeAngleSnapping(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.AngleSnapping < 0 || req.AngleSnapping > 1 {
		return &Payload{Message: language.GetValue("txtInvalidAngleSnapping"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeDeviceAngleSnapping(req.DeviceId, req.AngleSnapping)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtAngleSnappingChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableToChangeAngleSnapping"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeAngleSnapping"), Code: http.StatusOK, Status: 0}
}

// ProcessChangeButtonOptimization will process POST request from a client for button optimization mode change
func ProcessChangeButtonOptimization(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.ButtonOptimization < 0 || req.ButtonOptimization > 1 {
		return &Payload{Message: language.GetValue("txtInvalidButtonOptimization"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeDeviceButtonOptimization(req.DeviceId, req.ButtonOptimization)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtButtonOptimizationChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableToChangeButtonOptimization"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeButtonOptimization"), Code: http.StatusOK, Status: 0}
}

// ProcessChangeKeyAssignment will process POST request from a client for key assignment change
func ProcessChangeKeyAssignment(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}
	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}
	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	var keyAssignment = inputmanager.KeyAssignment{
		Name:          "",
		Default:       req.Enabled,
		ActionType:    req.KeyAssignmentType,
		ActionCommand: req.KeyAssignmentValue,
		ActionHold:    req.PressAndHold,
		IsMacro:       req.KeyAssignmentType == 10,
	}

	// Run it
	status := devices.ChangeDeviceKeyAssignment(req.DeviceId, req.KeyIndex, keyAssignment)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtKeyAssigmentUpdated"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableToApplyKeyAssigment"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToApplyKeyAssigment"), Code: http.StatusOK, Status: 0}
}

// ProcessChangeMuteIndicator will process POST request from a client for device mute indicator change
func ProcessChangeMuteIndicator(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.MuteIndicator < 0 || req.MuteIndicator > 1 {
		return &Payload{Message: language.GetValue("txtInvalidIndicatorOption"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeDeviceMuteIndicator(req.DeviceId, req.MuteIndicator)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtIndicatorChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableToChangeIndicator"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeIndicator"), Code: http.StatusOK, Status: 0}
}

// ProcessDeleteKeyboardProfile will process DELETE request from a client for device profile deletion
func ProcessDeleteKeyboardProfile(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.KeyboardProfileName) < 0 {
		return &Payload{Message: language.GetValue("txtInvalidProfileName"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.KeyboardProfileName); !m {
		return &Payload{Message: language.GetValue("txtProfileOnlyLettersNumbers"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.DeleteKeyboardProfile(req.DeviceId, req.KeyboardProfileName)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtKeyboardProfileDeleted"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableKeyboardProfile"), Code: http.StatusOK, Status: 0}
	case 3:
		return &Payload{Message: language.GetValue("txtDefaultProfileNoDelete"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableKeyboardProfile"), Code: http.StatusOK, Status: 0}
}

// ProcessChangeKeyboardProfile will process POST request from a client for keyboard profile change
func ProcessChangeKeyboardProfile(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.KeyboardProfileName) < 0 {
		return &Payload{Message: language.GetValue("txtInvalidProfileName"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.KeyboardProfileName); !m {
		return &Payload{Message: language.GetValue("txtProfileOnlyLettersNumbers"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeKeyboardProfile(req.DeviceId, req.KeyboardProfileName)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtKeyboardProfileChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableToChangeKeyboardProfile"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeKeyboardProfile"), Code: http.StatusOK, Status: 0}
}

// ProcessChangeUserProfile will process POST request from a client for device profile change
func ProcessChangeUserProfile(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.UserProfileName) < 0 {
		return &Payload{Message: language.GetValue("txtInvalidProfileName"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.UserProfileName); !m {
		return &Payload{Message: language.GetValue("txtProfileOnlyLettersNumbers"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeUserProfile(req.DeviceId, req.UserProfileName)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtUserProfileChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableToChangeUserProfile"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeUserProfile"), Code: http.StatusOK, Status: 0}
}

// ProcessBrightnessChange will process POST request from a client for device brightness change
func ProcessBrightnessChange(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.Brightness < 0 || req.Brightness > 4 {
		return &Payload{Message: language.GetValue("txtInvalidBrightness"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeDeviceBrightness(req.DeviceId, req.Brightness)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtBrightnessChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtBrightnessTooHigh"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeBrightness"), Code: http.StatusOK, Status: 0}
}

// ProcessBrightnessChangeGradual will process POST request from a client for device brightness change via defined number from 0-100
func ProcessBrightnessChangeGradual(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.Brightness < 0 || req.Brightness > 100 {
		return &Payload{Message: language.GetValue("txtInvalidBrightness"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.ChangeDeviceBrightnessGradual(req.DeviceId, req.Brightness)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtBrightnessChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtBrightnessTooHigh"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeBrightness"), Code: http.StatusOK, Status: 0}
}

// ProcessPositionChange will process POST request from a client for device position change
func ProcessPositionChange(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if req.Direction < 0 || req.Direction > 1 {
		return &Payload{Message: language.GetValue("txtNonExistingDirection"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateDevicePosition(req.DeviceId, req.Position, req.Direction)
	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtInvalidPosition"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtPositionChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtInvalidPosition"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangePosition"), Code: http.StatusOK, Status: 0}
}

// ProcessLabelChange will process POST request from a client for label change
func ProcessLabelChange(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.Label) < 1 {
		return &Payload{Message: language.GetValue("txtInvalidLabel"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9#.:_ -]*$", req.Label); !m {
		return &Payload{Message: language.GetValue("txtInvalidLabelCharacters"), Code: http.StatusOK, Status: 0}
	}

	if req.DeviceType < 0 || req.DeviceType > 1 {
		return &Payload{Message: language.GetValue("txtNonExistingDeviceType"), Code: http.StatusOK, Status: 0}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if req.ChannelId < -1 {
		return &Payload{Message: language.GetValue("txtNonExistingChannelId"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateDeviceLabel(req.DeviceId, req.ChannelId, req.Label, req.DeviceType)
	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtUnableToApplyLabel"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtDeviceLabelApplied"), Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: language.GetValue("txtUnableToApplyLabel"), Code: http.StatusOK, Status: 0}
}

// ProcessManualChangeSpeed will process POST request from a client for fan/pump speed change
func ProcessManualChangeSpeed(r *http.Request) *Payload {
	req := &Payload{}
	if !config.GetConfig().Manual {
		return &Payload{Message: language.GetValue("txtManualFlag"), Code: http.StatusMethodNotAllowed, Status: 0}
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.Value > 100 {
		req.Value = 100
	}

	if len(req.DeviceId) < 1 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if req.ChannelId < -1 {
		return &Payload{Message: language.GetValue("txtNonExistingChannelId"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	if devices.UpdateManualSpeed(req.DeviceId, req.ChannelId, req.Value) == 1 {
		return &Payload{Message: language.GetValue("txtDeviceSpeedProfileChanged"), Code: http.StatusOK, Status: 1}
	}

	return &Payload{Message: language.GetValue("txtNoDeviceForSpeedControl"), Code: http.StatusOK, Status: 0}
}

// ProcessChangeColor will process POST request from a client for RGB profile change
func ProcessChangeColor(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.Profile) < 1 {
		return &Payload{Message: language.GetValue("txtNonExistingSpeedProfile"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", req.Profile); !m {
		return &Payload{Message: language.GetValue("txtNonExistingRgbProfile"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateRgbProfile(req.DeviceId, req.ChannelId, req.Profile)

	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtUnableToChangeRgbProfile"), Code: http.StatusOK, Status: 0}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableToChangeRgbProfileNoPump"), Code: http.StatusOK, Status: 0}
	case 3:
		return &Payload{Message: language.GetValue("txtUnableToChangeRgbProfileNoKeyboard"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtDeviceRgbProfileChanged"), Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeRgbProfile"), Code: http.StatusOK, Status: 0}
}

// ProcessHardwareChangeColor will process POST request from a client for hardware color change
func ProcessHardwareChangeColor(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.HardwareLight < 0 || req.HardwareLight > 8 {
		return &Payload{Message: language.GetValue("txtNonExistingHwProfile"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateHardwareRgbProfile(req.DeviceId, req.HardwareLight)

	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtUnableToChangeRgbProfile"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtDeviceRgbProfileChanged"), Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeRgbProfile"), Code: http.StatusOK, Status: 0}
}

// ProcessChangeStrip will process POST request from a client for RGB strip change
func ProcessChangeStrip(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.StripId < 0 || req.StripId > 4 {
		return &Payload{Message: language.GetValue("txtNonExistingRgbStrip"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateRgbStrip(req.DeviceId, req.ChannelId, req.StripId)

	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtUnableToChangeRgbStrip"), Code: http.StatusOK, Status: 0}
	case 2:
		return &Payload{Message: language.GetValue("txtUnableToChangeRgbStripNoLink"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeRgbStrip"), Code: http.StatusOK, Status: 0}
}

// ProcessExternalHubDeviceType will process POST request from a client for external-LED hub
func ProcessExternalHubDeviceType(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}
	if req.PortId < 0 || req.PortId > 1 {
		return &Payload{Message: language.GetValue("txtNonExistingLedPort"), Code: http.StatusOK, Status: 0}
	}

	status := devices.UpdateExternalHubDeviceType(req.DeviceId, req.PortId, req.DeviceType)
	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtUnableToChangeExternalHub"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtExternalHubChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtNonExistingExternalDeviceType"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeExternalHub"), Code: http.StatusOK, Status: 0}
}

// ProcessARGBDevice will process POST request from a client for ARGB 3-pin devices
func ProcessARGBDevice(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}
	if req.PortId < 0 || req.PortId > 5 {
		return &Payload{Message: language.GetValue("txtNonExistingLedPort"), Code: http.StatusOK, Status: 0}
	}

	status := devices.UpdateARGBDevice(req.DeviceId, req.PortId, req.DeviceType)
	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtUnableToChangeExternalHub"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtExternalHubChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtNonExistingExternalDeviceType"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeExternalHub"), Code: http.StatusOK, Status: 0}
}

// ProcessKeyboardColor will process POST request from a client for keyboard color change
func ProcessKeyboardColor(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if req.Color.Red > 255 || req.Color.Green > 255 || req.Color.Blue > 255 {
		return &Payload{Message: language.GetValue("txtInvalidColorSelected"), Code: http.StatusOK, Status: 0}
	}

	if req.Color.Red < 0 || req.Color.Green < 0 || req.Color.Blue < 0 {
		return &Payload{Message: language.GetValue("txtInvalidColorSelected"), Code: http.StatusOK, Status: 0}
	}

	if req.KeyId < 1 {
		return &Payload{Message: language.GetValue("txtInvalidKeySelected"), Code: http.StatusOK, Status: 0}
	}

	if req.KeyOption < 0 || req.KeyOption > 2 {
		return &Payload{Message: language.GetValue("txtInvalidKeyOptionSelected"), Code: http.StatusOK, Status: 0}
	}

	status := devices.UpdateKeyboardColor(req.DeviceId, req.KeyId, req.KeyOption, req.Color)
	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtUnableToChangeDeviceColor"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtDeviceColorChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtNonExistingDeviceType"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeDeviceColor"), Code: http.StatusOK, Status: 0}
}

// ProcessMiscColor will process a POST request from a client for misc device color change
func ProcessMiscColor(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if req.Color.Red > 255 || req.Color.Green > 255 || req.Color.Blue > 255 {
		return &Payload{Message: language.GetValue("txtInvalidColorSelected"), Code: http.StatusOK, Status: 0}
	}

	if req.Color.Red < 0 || req.Color.Green < 0 || req.Color.Blue < 0 {
		return &Payload{Message: language.GetValue("txtInvalidColorSelected"), Code: http.StatusOK, Status: 0}
	}

	if req.AreaId < 1 {
		return &Payload{Message: language.GetValue("txtInvalidAreaSelected"), Code: http.StatusOK, Status: 0}
	}

	if req.AreaOption < 0 || req.AreaOption > 2 {
		return &Payload{Message: language.GetValue("txtInvalidAreaOptionSelected"), Code: http.StatusOK, Status: 0}
	}

	status := devices.UpdateMiscColor(req.DeviceId, req.AreaId, req.AreaOption, req.Color)
	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtUnableToChangeDeviceColor"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtDeviceColorChanged"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtNonExistingDeviceType"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeDeviceColor"), Code: http.StatusOK, Status: 0}
}

// ProcessExternalHubDeviceAmount will process POST request from a client for external-LED hub
func ProcessExternalHubDeviceAmount(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.DeviceAmount < 0 || req.DeviceAmount > 9 {
		return &Payload{Message: language.GetValue("txtInvalidDeviceAmount"), Code: http.StatusOK, Status: 0}
	}
	if req.PortId < 0 || req.PortId > 1 {
		return &Payload{Message: language.GetValue("txtNonExistingLedPort"), Code: http.StatusOK, Status: 0}
	}
	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	status := devices.UpdateExternalHubDeviceAmount(req.DeviceId, req.PortId, req.DeviceAmount)
	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtUnableToChangeDeviceAmount"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtExternalLedAmountUpdated"), Code: http.StatusOK, Status: 1}
	case 2:
		return &Payload{Message: language.GetValue("txtTooMuchLedChannels"), Code: http.StatusOK, Status: 0}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeDeviceAmount"), Code: http.StatusOK, Status: 0}
}

// ProcessDashboardSettingsChange will process POST request from a client for dashboard settings change
func ProcessDashboardSettingsChange(r *http.Request) *Payload {
	req := &dashboard.Dashboard{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if language.GetLanguage(req.LanguageCode) == nil {
		return &Payload{
			Message: language.GetValue("txtInvalidLanguage"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	status := dashboard.SaveDashboardSettings(req, true)
	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtUnableToSaveDashboardSettings"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtDashboardSettingsUpdated"), Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: language.GetValue("txtUnableToSaveDashboardSettings"), Code: http.StatusOK, Status: 0}
}

// ProcessChangeRgbScheduler will process a POST request from a client for RGB scheduler change
func ProcessChangeRgbScheduler(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	// Run it
	status := scheduler.UpdateRgbSettings(req.RgbControl, req.RgbOff, req.RgbOn)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtRgbSchedulerUpdated"), Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeRgbScheduler"), Code: http.StatusOK, Status: 0}
}

// ProcessPsuFanModeChange will process a POST request from a client for PSU fan mode change
func ProcessPsuFanModeChange(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.FanMode < 0 || req.FanMode > 10 {
		return &Payload{Message: language.GetValue("txtInvalidFanMode"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdatePsuFanMode(req.DeviceId, req.FanMode)
	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtUnableToChangeFanMode"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtFanModeChanged"), Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeFanMode"), Code: http.StatusOK, Status: 0}
}

// ProcessMouseDpiSave will process a POST request from a client for mouse DPI save
func ProcessMouseDpiSave(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.Stages) == 0 {
		return &Payload{Message: language.GetValue("txtInvalidStages"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.SaveMouseDPI(req.DeviceId, req.Stages)
	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtUnableToSaveDPI"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtMouseDPIUpdated"), Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: language.GetValue("txtUnableToSaveDPI"), Code: http.StatusOK, Status: 0}
}

// ProcessMouseZoneColorsSave will process a POST request from a client for mouse zone colors save
func ProcessMouseZoneColorsSave(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.SaveMouseZoneColors(req.DeviceId, req.ColorDpi, req.ColorZones)
	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtUnableToSaveMouseColors"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtMouseZoneColorsChanged"), Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: language.GetValue("txtUnableToSaveMouseColors"), Code: http.StatusOK, Status: 0}
}

// ProcessMouseDpiColorsSave will process a POST request from a client for mouse dpi colors save
func ProcessMouseDpiColorsSave(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.SaveMouseDpiColors(req.DeviceId, req.ColorDpi, req.ColorZones)
	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtUnableToSaveDPIColors"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtMouseDpiColorsChanged"), Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: language.GetValue("txtUnableToSaveDPIColors"), Code: http.StatusOK, Status: 0}
}

// ProcessHeadsetZoneColorsSave will process a POST request from a client for headset zone colors save
func ProcessHeadsetZoneColorsSave(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.SaveHeadsetZoneColors(req.DeviceId, req.ColorZones)
	switch status {
	case 0:
		return &Payload{Message: language.GetValue("txtUnableToSaveMouseColors"), Code: http.StatusOK, Status: 0}
	case 1:
		return &Payload{Message: language.GetValue("txtMouseZoneColorsChanged"), Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: language.GetValue("txtUnableToSaveMouseColors"), Code: http.StatusOK, Status: 0}
}

// ProcessDeleteMacroValue will process deletion of macro profile value
func ProcessDeleteMacroValue(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.MacroIndex < 0 || req.MacroId < 0 {
		return &Payload{
			Message: language.GetValue("txtInvalidMacroSelected"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	res := macro.DeleteMacroValue(req.MacroId, req.MacroIndex)
	if res == 1 {
		return &Payload{
			Message: language.GetValue("txtMacroProfileValueDeleted"),
			Code:    http.StatusOK,
			Status:  1,
		}
	}

	return &Payload{
		Message: language.GetValue("txtUnableToDeleteMacroProfileValue"),
		Code:    http.StatusOK,
		Status:  0,
	}
}

// ProcessDeleteMacroProfile will process deletion of macro profile
func ProcessDeleteMacroProfile(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if req.MacroId < 0 {
		return &Payload{
			Message: language.GetValue("txtInvalidMacroSelected"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	res := macro.DeleteMacroProfile(req.MacroId)
	if res == 1 {
		return &Payload{
			Message: language.GetValue("txtMacroProfileDeleted"),
			Code:    http.StatusOK,
			Status:  1,
		}
	}

	return &Payload{
		Message: language.GetValue("txtUnableToDeleteMacroProfile"),
		Code:    http.StatusOK,
		Status:  0,
	}
}

// ProcessNewMacroProfile will process creation of new macro profile
func ProcessNewMacroProfile(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}
	if len(req.MacroName) < 3 {
		return &Payload{
			Message: language.GetValue("txtUnableToValidateMacroName"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.MacroName); !m {
		return &Payload{
			Message: language.GetValue("txtProfileInvalidName"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	res := macro.NewMacroProfile(req.MacroName)
	if res == 1 {
		return &Payload{
			Message: language.GetValue("txtMacroProfileCreated"),
			Code:    http.StatusOK,
			Status:  1,
		}
	}

	return &Payload{
		Message: language.GetValue("txtUnableToCreateMacroProfile"),
		Code:    http.StatusOK,
		Status:  0,
	}
}

// ProcessNewMacroProfileValue will process creation of new macro profile value
func ProcessNewMacroProfileValue(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	macroId := req.MacroId
	macroType := req.MacroType
	macroValue := req.MacroValue
	macroDelay := req.MacroDelay

	if macroId < 0 {
		return &Payload{
			Message: language.GetValue("txtInvalidMacroSelected"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if macroType == 0 {
		return &Payload{
			Message: language.GetValue("txtInvalidMacroType"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if macroType < 3 || macroType > 5 {
		return &Payload{
			Message: language.GetValue("txtInvalidMacroType"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if macroType == 3 && macroValue == 0 {
		return &Payload{
			Message: language.GetValue("txtInvalidMacroValue"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if macroType == 5 && macroDelay < 1 {
		return &Payload{
			Message: language.GetValue("txtInvalidMacroDelay"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	res := macro.NewMacroProfileValue(macroId, macroType, macroValue, macroDelay)
	if res == 1 {
		return &Payload{
			Message: language.GetValue("txtMacroProfileValueSaved"),
			Code:    http.StatusOK,
			Status:  1,
		}
	}

	return &Payload{
		Message: language.GetValue("txtUnableToCreateMacroProfileValue"),
		Code:    http.StatusOK,
		Status:  0,
	}
}

// ProcessLedChange will process POST request from a client for LED change
func ProcessLedChange(r *http.Request) *Payload {
	req := &Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &Payload{
			Message: language.GetValue("txtUnableToValidateRequest"),
			Code:    http.StatusOK,
			Status:  0,
		}
	}

	if len(req.DeviceId) < 0 {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.DeviceId); !m {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	if devices.GetDevice(req.DeviceId) == nil {
		return &Payload{Message: language.GetValue("txtNonExistingDevice"), Code: http.StatusOK, Status: 0}
	}

	// Run it
	status := devices.UpdateDeviceLedData(req.DeviceId, req.LedProfile)
	switch status {
	case 1:
		return &Payload{Message: language.GetValue("txtLedDataChanged"), Code: http.StatusOK, Status: 1}
	}
	return &Payload{Message: language.GetValue("txtUnableToChangeLedData"), Code: http.StatusOK, Status: 0}
}
