package server

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/devices/lcd"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/language"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/macro"
	"OpenLinkHub/src/metrics"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/scheduler"
	"OpenLinkHub/src/server/requests"
	"OpenLinkHub/src/stats"
	"OpenLinkHub/src/systeminfo"
	"OpenLinkHub/src/systray"
	"OpenLinkHub/src/temperatures"
	"OpenLinkHub/src/templates"
	"OpenLinkHub/src/version"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Response contains data what is sent back to a client
type Response struct {
	sync.Mutex
	Code      int         `json:"code"`
	Status    int         `json:"status"`
	Message   string      `json:"message,omitempty"`
	Device    interface{} `json:"device,omitempty"`
	Devices   interface{} `json:"devices,omitempty"`
	Dashboard interface{} `json:"dashboard,omitempty"`
	Data      interface{} `json:"data,omitempty"` // For dataTables
}

type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var headers []Header
var server = &http.Server{}

// Send will process response and send it back to a client
func (r *Response) Send(w http.ResponseWriter) {
	r.Lock()
	defer r.Unlock()

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(r.Code)

	data, err := json.Marshal(r)
	if err != nil {
		_, err := fmt.Println(w, err.Error())
		if err != nil {
			return
		}
		return
	}

	_, err = w.Write(data)
	if err != nil {
		return
	}
}

// homePage returns response on /
func homePage(w http.ResponseWriter, _ *http.Request) {
	resp := &Response{
		Code:   http.StatusOK,
		Device: devices.GetDevices(),
	}
	resp.Send(w)
}

// getCpuTemperature will return current cpu temperature in string format
func getCpuTemperature(w http.ResponseWriter, _ *http.Request) {
	resp := &Response{
		Code:   http.StatusOK,
		Status: 1,
		Data:   dashboard.GetDashboard().TemperatureToString(temperatures.GetCpuTemperature()),
	}
	resp.Send(w)
}

// getCpuTemperatureClean will return current cpu temperature in float value
func getCpuTemperatureClean(w http.ResponseWriter, _ *http.Request) {
	resp := &Response{
		Code:   http.StatusOK,
		Status: 1,
		Data:   temperatures.GetCpuTemperature(),
	}
	resp.Send(w)
}

// getGpuTemperature will return current gpu temperature in string format
func getGpuTemperature(w http.ResponseWriter, _ *http.Request) {
	resp := &Response{
		Code:   http.StatusOK,
		Status: 1,
		Data:   dashboard.GetDashboard().TemperatureToString(temperatures.GetGpuTemperature()),
	}
	resp.Send(w)
}

// getGpuTemperatures will return current gpu temperature in string format
func getGpuTemperatures(w http.ResponseWriter, _ *http.Request) {
	data := make(map[int]interface{})
	for key, val := range systeminfo.GetInfo().GPU {
		data[key] = dashboard.GetDashboard().TemperatureToString(temperatures.GetGpuTemperatureIndex(val.Index))
	}
	resp := &Response{
		Code:   http.StatusOK,
		Status: 1,
		Data:   data,
	}
	resp.Send(w)
}

// getGpuTemperatureClean will return current gpu temperature in float value
func getGpuTemperatureClean(w http.ResponseWriter, _ *http.Request) {
	resp := &Response{
		Code:   http.StatusOK,
		Status: 1,
		Data:   temperatures.GetGpuTemperature(),
	}
	resp.Send(w)
}

// getStorageTemperature will return current storage temperature
func getStorageTemperature(w http.ResponseWriter, _ *http.Request) {
	resp := &Response{
		Code:   http.StatusOK,
		Status: 1,
		Data:   temperatures.GetStorageTemperatures(),
	}
	resp.Send(w)
}

// getBatteryStats will return battery stats
func getBatteryStats(w http.ResponseWriter, _ *http.Request) {
	resp := &Response{
		Code:   http.StatusOK,
		Status: 1,
		Data:   stats.GetBatteryStats(),
	}
	resp.Send(w)
}

// getDeviceMetrics will return a list device metrics in prometheus format
func getDeviceMetrics(w http.ResponseWriter, r *http.Request) {
	devices.UpdateDeviceMetrics()
	metrics.Handler(w, r)
}

// getDevices returns response on /devices
func getDevices(w http.ResponseWriter, r *http.Request) {
	deviceId, valid := getVar("/api/devices/", r)
	if !valid {
		resp := &Response{
			Code:    http.StatusOK,
			Devices: devices.GetDevicesEx(),
		}
		resp.Send(w)
	} else {
		resp := &Response{
			Code:   http.StatusOK,
			Device: devices.GetDevice(deviceId),
		}
		resp.Send(w)
	}
}

// getDeviceLed returns response on /led
func getDeviceLed(w http.ResponseWriter, r *http.Request) {
	deviceId, valid := getVar("/api/led/", r)
	if !valid {
		resp := &Response{
			Code:   http.StatusOK,
			Status: 1,
			Data:   devices.GetDevicesLedData(),
		}
		resp.Send(w)
	} else {
		resp := &Response{
			Code:   http.StatusOK,
			Status: 1,
			Data:   devices.GetDeviceLedData(deviceId),
		}
		resp.Send(w)
	}
}

// updateDeviceLed handles device LED changes
func updateDeviceLed(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessLedChange(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// getMacro returns response on /macro
func getMacro(w http.ResponseWriter, r *http.Request) {
	macroId, valid := getVar("/api/macro/", r)
	if !valid {
		resp := &Response{
			Code:   http.StatusOK,
			Status: 1,
			Data:   macro.GetProfiles(),
		}
		resp.Send(w)
	} else {
		val, err := strconv.Atoi(macroId)
		if err != nil {
			resp := &Response{
				Code:    http.StatusOK,
				Status:  0,
				Message: language.GetValue("txtUnableToParseMacroId"),
			}
			resp.Send(w)
		} else {
			resp := &Response{
				Code:   http.StatusOK,
				Status: 1,
				Data:   macro.GetProfile(val),
			}
			resp.Send(w)
		}
	}
}

// getKeyName returns response on /api/macro/keyInfo/
func getKeyName(w http.ResponseWriter, r *http.Request) {
	keyIndex, valid := getVar("/api/macro/keyInfo/", r)
	if !valid {
		resp := &Response{
			Code:    http.StatusOK,
			Status:  0,
			Message: language.GetValue("txtUnableToParseKeyIndex"),
		}
		resp.Send(w)
	} else {
		val, err := strconv.ParseUint(keyIndex, 10, 8) // base 10, 8-bit size
		if err != nil {
			resp := &Response{
				Code:    http.StatusOK,
				Status:  0,
				Message: language.GetValue("txtUnableToParseMacroId"),
			}
			resp.Send(w)
		} else {
			resp := &Response{
				Code:   http.StatusOK,
				Status: 1,
				Data:   inputmanager.GetKeyName(uint16(val)),
			}
			resp.Send(w)
		}
	}
}

// getTemperatures returns response on /temperatures
func getTemperature(w http.ResponseWriter, r *http.Request) {
	resp := &Response{}
	profile, valid := getVar("/api/temperatures/", r)
	if !valid {
		resp = &Response{
			Code:   http.StatusOK,
			Status: 0,
			Data:   temperatures.GetTemperatureProfiles(),
		}
	} else {
		if temperatureProfile := temperatures.GetTemperatureProfile(profile); temperatureProfile != nil {
			resp = &Response{
				Code:   http.StatusOK,
				Status: 1,
				Data:   temperatureProfile,
			}
		} else {
			resp = &Response{
				Code:    http.StatusOK,
				Status:  0,
				Message: language.GetValue("txtNoSuchTemperatureProfile"),
			}
		}
	}
	resp.Send(w)
}

// getTemperatureGraph returns response on for temperature graph
func getTemperatureGraph(w http.ResponseWriter, r *http.Request) {
	resp := &Response{}
	profile, valid := getVar("/api/temperatures/graph/", r)
	if !valid {
		resp = &Response{
			Code:    http.StatusOK,
			Status:  0,
			Message: language.GetValue("txtNoSuchTemperatureProfile"),
		}
	} else {
		if temperatureProfile := temperatures.GetTemperatureGraph(profile); temperatureProfile != nil {
			resp = &Response{
				Code:   http.StatusOK,
				Status: 1,
				Data:   temperatureProfile,
			}
		} else {
			resp = &Response{
				Code:    http.StatusOK,
				Status:  0,
				Message: language.GetValue("txtNoSuchTemperatureProfile"),
			}
		}
	}
	resp.Send(w)
}

// getColor returns response on /color
func getColor(w http.ResponseWriter, r *http.Request) {
	resp := &Response{}
	deviceId, valid := getVar("/api/color/", r)
	if !valid {
		resp = &Response{
			Code:   http.StatusOK,
			Status: 0,
			Data:   devices.GetRgbProfiles(),
		}
	} else {
		if rgbProfile := devices.GetRgbProfile(deviceId); rgbProfile != nil {
			resp = &Response{
				Code:   http.StatusOK,
				Status: 1,
				Data:   rgbProfile,
			}
		} else {
			resp = &Response{
				Code:    http.StatusOK,
				Status:  0,
				Message: language.GetValue("txtNoSuchRGBProfile"),
			}
		}
	}
	resp.Send(w)
}

// getMediaKeys will return a map of media keys
func getMediaKeys(w http.ResponseWriter, _ *http.Request) {
	resp := &Response{
		Code:   http.StatusOK,
		Status: 1,
		Data:   inputmanager.GetMediaKeys(),
	}
	resp.Send(w)
}

// getMediaKeys will return a map of non-media keys
func getInputKeys(w http.ResponseWriter, _ *http.Request) {
	resp := &Response{
		Code:   http.StatusOK,
		Status: 1,
		Data:   inputmanager.GetInputKeys(),
	}
	resp.Send(w)
}

// getMouseButtons will return a map of mouse buttons
func getMouseButtons(w http.ResponseWriter, _ *http.Request) {
	resp := &Response{
		Code:   http.StatusOK,
		Status: 1,
		Data:   inputmanager.GetMouseButtons(),
	}
	resp.Send(w)
}

// getSystrayData will return systray data
func getSystrayData(w http.ResponseWriter, _ *http.Request) {
	resp := &Response{
		Code:   http.StatusOK,
		Status: 1,
		Data:   systray.Get(),
	}
	resp.Send(w)
}

// getKeyAssignmentTypes returns list of key assignment types for keyboard
func getKeyAssignmentTypes(w http.ResponseWriter, r *http.Request) {
	deviceId, valid := getVar("/api/keyboard/assignmentsTypes/", r)
	if !valid {
		resp := &Response{
			Code:    http.StatusOK,
			Status:  0,
			Message: language.GetValue("txtInvalidDeviceId"),
		}
		resp.Send(w)
	} else {
		val := devices.ProcessGetKeyAssignmentTypes(deviceId)
		if val == nil {
			resp := &Response{
				Code:    http.StatusOK,
				Status:  0,
				Message: language.GetValue("txtUnableToGetAssignmentsTypes"),
			}
			resp.Send(w)
		} else {
			resp := &Response{
				Code:   http.StatusOK,
				Status: 1,
				Data:   val,
			}
			resp.Send(w)
		}
	}
}

// getKeyAssignmentModifiers returns list of key assignment modifiers for keyboard
func getKeyAssignmentModifiers(w http.ResponseWriter, r *http.Request) {
	deviceId, valid := getVar("/api/keyboard/assignmentsModifiers/", r)
	if !valid {
		resp := &Response{
			Code:    http.StatusOK,
			Status:  0,
			Message: language.GetValue("txtInvalidDeviceId"),
		}
		resp.Send(w)
	} else {
		val := devices.ProcessGetKeyAssignmentModifiers(deviceId)
		if val == nil {
			resp := &Response{
				Code:    http.StatusOK,
				Status:  0,
				Message: language.GetValue("txtUnableToGetAssignmentsModifiers"),
			}
			resp.Send(w)
		} else {
			resp := &Response{
				Code:   http.StatusOK,
				Status: 1,
				Data:   val,
			}
			resp.Send(w)
		}
	}
}

// getKeyboardPerformance returns keyboard performance data
func getKeyboardPerformance(w http.ResponseWriter, r *http.Request) {
	deviceId, valid := getVar("/api/keyboard/getPerformance/", r)
	if !valid {
		resp := &Response{
			Code:    http.StatusOK,
			Status:  0,
			Message: language.GetValue("txtInvalidDeviceId"),
		}
		resp.Send(w)
	} else {
		val := devices.ProcessGetKeyboardPerformance(deviceId)
		if val == nil {
			resp := &Response{
				Code:    http.StatusOK,
				Status:  0,
				Message: language.GetValue("txtUnableToGetKeyboardPerformance"),
			}
			resp.Send(w)
		} else {
			resp := &Response{
				Code:   http.StatusOK,
				Status: 1,
				Data:   val,
			}
			resp.Send(w)
		}
	}
}

// updateRgbProfile handles device rgb profile update
func updateRgbProfile(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessUpdateRgbProfile(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// newTemperatureProfile handles creation of new temperature profile
func newTemperatureProfile(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessNewTemperatureProfile(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// deleteTemperatureProfile handles deletion of temperature profile
func deleteTemperatureProfile(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessDeleteTemperatureProfile(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// updateTemperatureProfile handles update of temperature profile
func updateTemperatureProfile(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessUpdateTemperatureProfile(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// updateTemperatureProfile handles update of temperature profile
func updateTemperatureProfileGraph(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessUpdateTemperatureProfileGraph(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setDeviceSpeed handles device speed changes
func setDeviceSpeed(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeSpeed(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setDeviceLabel handles device label changes
func setDeviceLabel(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessLabelChange(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setDeviceLcd handles device LCD changes
func setDeviceLcd(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessLcdChange(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setDeviceLcdProfile handles device LCD changes
func setDeviceLcdProfile(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessLcdProfileChange(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changeDeviceLcd handles device LCD updates
func changeDeviceLcd(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessLcdDeviceChange(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setDeviceLcdRotation handles device LCD rotation changes
func setDeviceLcdRotation(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessLcdRotationChange(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setDeviceLcdImage handles device LCD image changes
func setDeviceLcdImage(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessLcdImageChange(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// updateLcdProfile handles update of LCD profile
func updateLcdProfile(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessLcdProfileUpdate(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// saveUserProfile handles saving custom user profiles
func saveUserProfile(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessSaveUserProfile(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changeUserProfile handles user profile change
func changeUserProfile(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeUserProfile(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changeBrightness handles user brightness change
func changeBrightness(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessBrightnessChange(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changeBrightnessGradual handles user brightness change via defined number from 0-100
func changeBrightnessGradual(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessBrightnessChangeGradual(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changePosition handles device position change
func changePosition(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessPositionChange(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setManualDeviceSpeed handles manual device speed changes
func setManualDeviceSpeed(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessManualChangeSpeed(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setDeviceColor handles device color changes
func setDeviceColor(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeColor(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setGlobalDeviceColor handles global color changes
func setGlobalDeviceColor(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessGlobalChangeColor(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setLinkAdapterColor handles LINK adapter color changes
func setLinkAdapterColor(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeLinkAdapterColor(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setLinkAdapterBulkColor handles LINK adapter bulk color changes
func setLinkAdapterBulkColor(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeLinkAdapterColorBulk(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// getRgbOverride return RGB override for given device
func getRgbOverride(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessGetRgbOverride(r)
	resp := &Response{
		Code:   request.Code,
		Status: request.Status,
		Data:   request.Data,
	}
	resp.Send(w)
}

// getRgbOverride return RGB override for given device
func setRgbOverride(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessSetRgbOverride(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// getLedData return RGB LED data
func getLedData(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessGetLedData(r)
	resp := &Response{
		Code:   request.Code,
		Status: request.Status,
		Data:   request.Data,
	}
	resp.Send(w)
}

// setLedData saves RGB LED data
func setLedData(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessSetLedData(r)
	resp := &Response{
		Code:   request.Code,
		Status: request.Status,
		Data:   request.Data,
	}
	resp.Send(w)
}

// setOpenRgbIntegration saves OpenRGB integration state
func setOpenRgbIntegration(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessSetOpenRgbIntegration(r)
	resp := &Response{
		Code:   request.Code,
		Status: request.Status,
		Data:   request.Data,
	}
	resp.Send(w)
}

// setDeviceHardwareColor handles device hardware color changes
func setDeviceHardwareColor(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessHardwareChangeColor(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setDeviceStrip handles device RGB strip changes
func setDeviceStrip(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeStrip(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setDeviceLinkAdapter handles LINK adapter device change
func setDeviceLinkAdapter(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeLinkAdapter(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setExternalHubDeviceType handles device change of external-LED hub
func setExternalHubDeviceType(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessExternalHubDeviceType(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setARGBDevice handles device change of ARGB 3-pin devices
func setARGBDevice(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessARGBDevice(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setExternalHubDeviceAmount handles device amount change of external-LED hub
func setExternalHubDeviceAmount(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessExternalHubDeviceAmount(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// getDashboardSettings will get dashboard settings
func getDashboardSettings(w http.ResponseWriter, _ *http.Request) {
	resp := &Response{
		Code:      http.StatusOK,
		Status:    1,
		Dashboard: dashboard.GetDashboard(),
	}
	resp.Send(w)
}

// setDashboardSettings handles dashboard settings change
func setDashboardSettings(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessDashboardSettingsChange(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setDashboardDevicePosition handles dashboard device position change
func setDashboardDevicePosition(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessDashboardDevicePositionChange(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setKeyboardColor handles keyboard color change
func setKeyboardColor(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessKeyboardColor(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// setMiscColor handles misc device color change
func setMiscColor(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessMiscColor(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// saveDeviceProfile handles a new device profile
func saveDeviceProfile(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessSaveDeviceProfile(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changeKeyboardLayout handles keyboard layout change
func changeKeyboardLayout(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeKeyboardLayout(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changeControlDial handles keyboard control dial function change
func changeControlDial(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeControlDial(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changeSleepMode handles device sleep mode change
func changeSleepMode(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeSleepMode(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changePollingRate handles device USB polling rate
func changePollingRate(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangePollingRate(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changeAngleSnapping handles device angle snapping mode
func changeAngleSnapping(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeAngleSnapping(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changeButtonOptimization handles device button optimization mode
func changeButtonOptimization(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeButtonOptimization(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changeKeyAssignment handles device key assignment update
func changeKeyAssignment(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeKeyAssignment(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changeMuteIndicator handles device mute indicator change
func changeMuteIndicator(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeMuteIndicator(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changeRgbScheduler handles RGB scheduler change
func changeRgbScheduler(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeRgbScheduler(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// deleteKeyboardProfile handles deletion of keyboard profile
func deleteKeyboardProfile(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessDeleteKeyboardProfile(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changeKeyboardProfile handles keyboard profile change
func changeKeyboardProfile(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeKeyboardProfile(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// changePsuFanMode handles PSU fan mode change
func changePsuFanMode(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessPsuFanModeChange(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// saveMouseDpi handles mouse DPI save
func saveMouseDpi(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessMouseDpiSave(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// saveMouseZoneColors handles mouse zone colors save
func saveMouseZoneColors(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessMouseZoneColorsSave(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// saveMouseDpiColors handles mouse DPI colors save
func saveMouseDpiColors(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessMouseDpiColorsSave(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// saveHeadsetZoneColors handles mouse zone colors save
func saveHeadsetZoneColors(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessHeadsetZoneColorsSave(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// deleteMacroValue handles deletion of macro profile value
func deleteMacroValue(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessDeleteMacroValue(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// updateMacroValue handles update of macro profile value
func updateMacroValue(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessUpdateMacroValue(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// deleteMacroProfile handles deletion of macro profile
func deleteMacroProfile(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessDeleteMacroProfile(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// newMacroProfile handles creation of new macro profile
func newMacroProfile(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessNewMacroProfile(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// newMacroProfileValue handles creation of new macro profile value
func newMacroProfileValue(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessNewMacroProfileValue(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// getKeyboardKey handles information about keyboard get
func getGetKeyboardKey(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessGetKeyboardKey(r)
	resp := &Response{
		Code:   request.Code,
		Status: request.Status,
		Data:   request.Data,
	}
	resp.Send(w)
}

// setKeyboardPerformance handles setting keyboard performance
func setKeyboardPerformance(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessSetKeyboardPerformance(r)
	resp := &Response{
		Code:    request.Code,
		Status:  request.Status,
		Message: request.Message,
	}
	resp.Send(w)
}

// uiDeviceOverview handles device overview
func uiDeviceOverview(w http.ResponseWriter, r *http.Request) {
	deviceId, valid := getVar("/device/", r)
	if !valid {
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Status:  0,
			Message: language.GetValue("txtUnableToProcessDeviceRequest"),
		}
		resp.Send(w)
	}

	device := devices.GetDevice(deviceId)
	template := devices.GetDeviceTemplate(device)
	if len(template) == 0 {
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Status:  0,
			Message: language.GetValue("txtUnableToProcessDeviceRequest"),
		}
		resp.Send(w)
	}

	web := templates.Web{}
	web.Title = "Device Dashboard"
	web.Devices = devices.GetDevices()
	web.Device = device
	web.Lcd = lcd.GetLcdDevices()
	web.LCDImages = lcd.GetLcdImages()
	web.Temperatures = temperatures.GetTemperatureProfiles()
	web.Rgb = rgb.GetRGB().Profiles
	web.BuildInfo = version.GetBuildInfo()
	web.SystemInfo = systeminfo.GetInfo()
	web.Stats = stats.GetAIOStats()
	web.Macros = macro.GetProfiles()

	web.CpuTemp = dashboard.GetDashboard().TemperatureToString(temperatures.GetCpuTemperature())
	web.GpuTemp = dashboard.GetDashboard().TemperatureToString(temperatures.GetGpuTemperature())
	t := templates.GetTemplate()

	for header := range headers {
		w.Header().Set(headers[header].Key, headers[header].Value)
	}

	err := t.ExecuteTemplate(w, template, web)
	if err != nil {
		fmt.Println(err)
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Message: language.GetValue("txtUnableToServeWebContent"),
		}
		resp.Send(w)
	}
}

// uiIndex handles index page
func uiIndex(w http.ResponseWriter, _ *http.Request) {
	deviceList := devices.GetDevices()
	web := templates.Web{}
	web.Title = "Device Dashboard"
	web.Devices = deviceList
	web.BuildInfo = version.GetBuildInfo()
	web.SystemInfo = systeminfo.GetInfo()
	web.CpuTemp = dashboard.GetDashboard().TemperatureToString(temperatures.GetCpuTemperature())
	web.GpuTemp = dashboard.GetDashboard().TemperatureToString(temperatures.GetGpuTemperature())
	web.Dashboard = dashboard.GetDashboard()
	web.BatteryStats = stats.GetBatteryStats()
	web.RGBModes = []string{
		"circle",
		"circleshift",
		"colorpulse",
		"colorshift",
		"colorwarp",
		"cpu-temperature",
		"flickering",
		"gpu-temperature",
		"off",
		"rainbow",
		"rotator",
		"spinner",
		"static",
		"storm",
		"watercolor",
		"wave",
	}
	web.Page = "index"

	// Add all devices to the list
	dashboard.AddDeviceToOrderList(deviceList)

	t := templates.GetTemplate()
	for header := range headers {
		w.Header().Set(headers[header].Key, headers[header].Value)
	}

	err := t.ExecuteTemplate(w, "index.html", web)
	if err != nil {
		fmt.Println(err)
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Message: language.GetValue("txtUnableToServeWebContent"),
		}
		resp.Send(w)
	}
}

// uiTemperatureOverview handles overview of temperature profiles
func uiTemperatureOverview(w http.ResponseWriter, _ *http.Request) {
	web := templates.Web{}
	web.Title = "Device Dashboard"
	web.Devices = devices.GetDevices()
	web.TemperatureProbes = devices.GetTemperatureProbes()
	web.HwMonSensors = temperatures.GetExternalHwMonSensors()
	web.Temperatures = temperatures.GetTemperatureProfiles()
	web.BuildInfo = version.GetBuildInfo()
	web.SystemInfo = systeminfo.GetInfo()
	web.Page = "temperature"

	t := templates.GetTemplate()

	for header := range headers {
		w.Header().Set(headers[header].Key, headers[header].Value)
	}

	tpl := "temperature.html"
	if config.GetConfig().GraphProfiles {
		tpl = "temperatureGraph.html"
	}

	err := t.ExecuteTemplate(w, tpl, web)
	if err != nil {
		fmt.Println(err)
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Message: language.GetValue("txtUnableToServeWebContent"),
		}
		resp.Send(w)
	}
}

// uiTemperatureGraphOverview handles overview of graph temperature profiles
func uiTemperatureGraphOverview(w http.ResponseWriter, _ *http.Request) {
	web := templates.Web{}
	web.Title = "Device Dashboard"
	web.Devices = devices.GetDevices()
	web.TemperatureProbes = devices.GetTemperatureProbes()
	web.HwMonSensors = temperatures.GetExternalHwMonSensors()
	web.Temperatures = temperatures.GetTemperatureProfiles()
	web.BuildInfo = version.GetBuildInfo()
	web.SystemInfo = systeminfo.GetInfo()
	web.Page = "temperature"

	t := templates.GetTemplate()

	for header := range headers {
		w.Header().Set(headers[header].Key, headers[header].Value)
	}

	err := t.ExecuteTemplate(w, "temperatureGraph.html", web)
	if err != nil {
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Message: language.GetValue("txtUnableToServeWebContent"),
		}
		resp.Send(w)
	}
}

// uiSchedulerOverview handles overview of scheduler settings
func uiSchedulerOverview(w http.ResponseWriter, _ *http.Request) {
	web := templates.Web{}
	web.Title = "Device Dashboard"
	web.Devices = devices.GetDevices()
	web.Scheduler = scheduler.GetScheduler()
	web.BuildInfo = version.GetBuildInfo()
	web.SystemInfo = systeminfo.GetInfo()
	web.Page = "scheduler"
	t := templates.GetTemplate()

	for header := range headers {
		w.Header().Set(headers[header].Key, headers[header].Value)
	}

	err := t.ExecuteTemplate(w, "scheduler.html", web)
	if err != nil {
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Message: language.GetValue("txtUnableToServeWebContent"),
		}
		resp.Send(w)
	}
}

// uiRgbEditor handles overview of RGB profiles
func uiRgbEditor(w http.ResponseWriter, _ *http.Request) {
	web := templates.Web{}
	web.Title = "Device Dashboard"
	web.Devices = devices.GetDevices()
	web.RGBProfiles = devices.GetRgbProfiles()
	web.BuildInfo = version.GetBuildInfo()
	web.SystemInfo = systeminfo.GetInfo()
	web.Page = "rgb"

	t := templates.GetTemplate()

	for header := range headers {
		w.Header().Set(headers[header].Key, headers[header].Value)
	}

	err := t.ExecuteTemplate(w, "rgb.html", web)
	if err != nil {
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Message: language.GetValue("txtUnableToServeWebContent"),
		}
		resp.Send(w)
	}
}

// uiColorOverview handles overview or RGB profiles
func uiColorOverview(w http.ResponseWriter, _ *http.Request) {
	web := templates.Web{}
	web.Title = "Device Dashboard"
	web.Devices = devices.GetDevices()
	web.Rgb = rgb.GetRgbProfiles()
	web.BuildInfo = version.GetBuildInfo()
	web.SystemInfo = systeminfo.GetInfo()
	web.Page = "colors"
	t := templates.GetTemplate()

	for header := range headers {
		w.Header().Set(headers[header].Key, headers[header].Value)
	}

	err := t.ExecuteTemplate(w, "rgb.html", web)
	if err != nil {
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Message: language.GetValue("txtUnableToServeWebContent"),
		}
		resp.Send(w)
	}
}

// uiMacrosOverview handles overview of macro profiles
func uiMacrosOverview(w http.ResponseWriter, _ *http.Request) {
	web := templates.Web{}
	web.Title = "Device Dashboard"
	web.Devices = devices.GetDevices()
	web.TemperatureProbes = devices.GetTemperatureProbes()
	web.Macros = macro.GetProfiles()
	web.InputActions = inputmanager.GetInputActions()
	web.BuildInfo = version.GetBuildInfo()
	web.SystemInfo = systeminfo.GetInfo()
	web.Page = "macros"

	t := templates.GetTemplate()

	for header := range headers {
		w.Header().Set(headers[header].Key, headers[header].Value)
	}

	err := t.ExecuteTemplate(w, "macros.html", web)
	if err != nil {
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Message: language.GetValue("txtUnableToServeWebContent"),
		}
		resp.Send(w)
	}
}

// uiLcdOverview handles overview of LCD profiles
func uiLcdOverview(w http.ResponseWriter, _ *http.Request) {
	web := templates.Web{}
	web.Title = "Device Dashboard"
	web.Devices = devices.GetDevices()
	web.TemperatureProbes = devices.GetTemperatureProbes()
	web.LCDProfiles = lcd.GetCustomLcdProfiles()
	web.LCDSensors = lcd.GetLcdSensors()
	web.InputActions = inputmanager.GetInputActions()
	web.BuildInfo = version.GetBuildInfo()
	web.SystemInfo = systeminfo.GetInfo()
	web.Page = "lcd"

	t := templates.GetTemplate()

	for header := range headers {
		w.Header().Set(headers[header].Key, headers[header].Value)
	}

	err := t.ExecuteTemplate(w, "lcd.html", web)
	if err != nil {
		fmt.Println(err)
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Message: language.GetValue("txtUnableToServeWebContent"),
		}
		resp.Send(w)
	}
}

// uiSettings handles index page
func uiSettings(w http.ResponseWriter, _ *http.Request) {
	web := templates.Web{}
	web.Title = "Device Dashboard"
	web.Devices = devices.GetDevices()
	web.BuildInfo = version.GetBuildInfo()
	web.SystemInfo = systeminfo.GetInfo()
	web.Dashboard = dashboard.GetDashboard()
	web.Languages = language.GetLanguages()
	web.LanguageCode = dashboard.GetDashboard().LanguageCode
	web.Page = "settings"

	t := templates.GetTemplate()

	for header := range headers {
		w.Header().Set(headers[header].Key, headers[header].Value)
	}

	err := t.ExecuteTemplate(w, "settings.html", web)
	if err != nil {
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Message: language.GetValue("txtUnableToServeWebContent"),
		}
		resp.Send(w)
	}
}

// getVar will extract dynamic path from GET request
func getVar(path string, r *http.Request) (string, bool) {
	value := strings.TrimPrefix(r.URL.Path, path)
	if value == "" || strings.Contains(value, "/") {
		return "", false
	}

	if m, _ := regexp.MatchString("^[a-zA-Z0-9-;:]+$", value); !m {
		return "", false
	}

	return value, true
}

func handleFunc(mux *http.ServeMux, path, method string, handler func(w http.ResponseWriter, r *http.Request)) {
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == method {
			handler(w, r)
		} else {
			http.Error(w, language.GetValue("txtMethodNotAllowed"), http.StatusMethodNotAllowed)
		}
	})
}

// setRoutes will set up all routes
func setRoutes() http.Handler {
	r := http.NewServeMux()
	fs := http.FileServer(http.Dir("./static"))
	r.Handle("/static/", http.StripPrefix("/static/", fs))

	// GET
	handleFunc(r, "/api/", http.MethodGet, homePage)
	handleFunc(r, "/api/cpuTemp", http.MethodGet, getCpuTemperature)
	handleFunc(r, "/api/cpuTemp/clean", http.MethodGet, getCpuTemperatureClean)
	handleFunc(r, "/api/gpuTemp", http.MethodGet, getGpuTemperature)
	handleFunc(r, "/api/gpuTemps", http.MethodGet, getGpuTemperatures)
	handleFunc(r, "/api/gpuTemp/clean", http.MethodGet, getGpuTemperatureClean)
	handleFunc(r, "/api/storageTemp", http.MethodGet, getStorageTemperature)
	handleFunc(r, "/api/batteryStats", http.MethodGet, getBatteryStats)
	handleFunc(r, "/api/devices/", http.MethodGet, getDevices)
	handleFunc(r, "/api/color/", http.MethodGet, getColor)
	handleFunc(r, "/api/temperatures/", http.MethodGet, getTemperature)
	handleFunc(r, "/api/temperatures/graph/", http.MethodGet, getTemperatureGraph)
	handleFunc(r, "/api/input/media", http.MethodGet, getMediaKeys)
	handleFunc(r, "/api/input/keyboard", http.MethodGet, getInputKeys)
	handleFunc(r, "/api/input/mouse", http.MethodGet, getMouseButtons)
	handleFunc(r, "/api/led/", http.MethodGet, getDeviceLed)
	handleFunc(r, "/api/macro/", http.MethodGet, getMacro)
	handleFunc(r, "/api/macro/keyInfo/", http.MethodGet, getKeyName)
	handleFunc(r, "/api/dashboard", http.MethodGet, getDashboardSettings)
	handleFunc(r, "/api/keyboard/assignmentsTypes/", http.MethodGet, getKeyAssignmentTypes)
	handleFunc(r, "/api/keyboard/assignmentsModifiers/", http.MethodGet, getKeyAssignmentModifiers)
	handleFunc(r, "/api/keyboard/getPerformance/", http.MethodGet, getKeyboardPerformance)
	handleFunc(r, "/api/systray", http.MethodGet, getSystrayData)

	// POST
	handleFunc(r, "/api/temperatures/new", http.MethodPost, newTemperatureProfile)
	handleFunc(r, "/api/speed", http.MethodPost, setDeviceSpeed)
	handleFunc(r, "/api/speed/manual", http.MethodPost, setManualDeviceSpeed)
	handleFunc(r, "/api/color", http.MethodPost, setDeviceColor)
	handleFunc(r, "/api/color/global", http.MethodPost, setGlobalDeviceColor)
	handleFunc(r, "/api/color/linkAdapter", http.MethodPost, setLinkAdapterColor)
	handleFunc(r, "/api/color/linkAdapter/bulk", http.MethodPost, setLinkAdapterBulkColor)
	handleFunc(r, "/api/color/getOverride", http.MethodPost, getRgbOverride)
	handleFunc(r, "/api/color/setOverride", http.MethodPost, setRgbOverride)
	handleFunc(r, "/api/color/getLedData", http.MethodPost, getLedData)
	handleFunc(r, "/api/color/setLedData", http.MethodPost, setLedData)
	handleFunc(r, "/api/color/setOpenRgbIntegration", http.MethodPost, setOpenRgbIntegration)
	handleFunc(r, "/api/color/hardware", http.MethodPost, setDeviceHardwareColor)
	handleFunc(r, "/api/hub/strip", http.MethodPost, setDeviceStrip)
	handleFunc(r, "/api/hub/linkAdapter", http.MethodPost, setDeviceLinkAdapter)
	handleFunc(r, "/api/hub/type", http.MethodPost, setExternalHubDeviceType)
	handleFunc(r, "/api/hub/amount", http.MethodPost, setExternalHubDeviceAmount)
	handleFunc(r, "/api/label", http.MethodPost, setDeviceLabel)
	handleFunc(r, "/api/lcd", http.MethodPost, setDeviceLcd)
	handleFunc(r, "/api/lcd/device", http.MethodPost, changeDeviceLcd)
	handleFunc(r, "/api/lcd/rotation", http.MethodPost, setDeviceLcdRotation)
	handleFunc(r, "/api/lcd/profile", http.MethodPost, setDeviceLcdProfile)
	handleFunc(r, "/api/lcd/image", http.MethodPost, setDeviceLcdImage)
	handleFunc(r, "/api/brightness", http.MethodPost, changeBrightness)
	handleFunc(r, "/api/brightness/gradual", http.MethodPost, changeBrightnessGradual)
	handleFunc(r, "/api/position", http.MethodPost, changePosition)
	handleFunc(r, "/api/dashboard/update", http.MethodPost, setDashboardSettings)
	handleFunc(r, "/api/dashboard/position", http.MethodPost, setDashboardDevicePosition)
	handleFunc(r, "/api/argb", http.MethodPost, setARGBDevice)
	handleFunc(r, "/api/keyboard/color", http.MethodPost, setKeyboardColor)
	handleFunc(r, "/api/misc/color", http.MethodPost, setMiscColor)
	handleFunc(r, "/api/userProfile/change", http.MethodPost, changeUserProfile)
	handleFunc(r, "/api/keyboard/profile/change", http.MethodPost, changeKeyboardProfile)
	handleFunc(r, "/api/keyboard/profile/save", http.MethodPost, saveDeviceProfile)
	handleFunc(r, "/api/keyboard/layout", http.MethodPost, changeKeyboardLayout)
	handleFunc(r, "/api/keyboard/dial", http.MethodPost, changeControlDial)
	handleFunc(r, "/api/keyboard/sleep", http.MethodPost, changeSleepMode)
	handleFunc(r, "/api/keyboard/pollingRate", http.MethodPost, changePollingRate)
	handleFunc(r, "/api/scheduler/rgb", http.MethodPost, changeRgbScheduler)
	handleFunc(r, "/api/psu/speed", http.MethodPost, changePsuFanMode)
	handleFunc(r, "/api/mouse/dpi", http.MethodPost, saveMouseDpi)
	handleFunc(r, "/api/mouse/zoneColors", http.MethodPost, saveMouseZoneColors)
	handleFunc(r, "/api/mouse/dpiColors", http.MethodPost, saveMouseDpiColors)
	handleFunc(r, "/api/mouse/sleep", http.MethodPost, changeSleepMode)
	handleFunc(r, "/api/mouse/pollingRate", http.MethodPost, changePollingRate)
	handleFunc(r, "/api/mouse/angleSnapping", http.MethodPost, changeAngleSnapping)
	handleFunc(r, "/api/mouse/buttonOptimization", http.MethodPost, changeButtonOptimization)
	handleFunc(r, "/api/mouse/updateKeyAssignment", http.MethodPost, changeKeyAssignment)
	handleFunc(r, "/api/headset/zoneColors", http.MethodPost, saveHeadsetZoneColors)
	handleFunc(r, "/api/headset/sleep", http.MethodPost, changeSleepMode)
	handleFunc(r, "/api/headset/muteIndicator", http.MethodPost, changeMuteIndicator)
	handleFunc(r, "/api/led/update", http.MethodPost, updateDeviceLed)
	handleFunc(r, "/api/macro/newValue", http.MethodPost, newMacroProfileValue)
	handleFunc(r, "/api/keyboard/getKey/", http.MethodPost, getGetKeyboardKey)
	handleFunc(r, "/api/keyboard/updateKeyAssignment", http.MethodPost, changeKeyAssignment)
	handleFunc(r, "/api/keyboard/setPerformance", http.MethodPost, setKeyboardPerformance)
	handleFunc(r, "/api/macro/updateValue", http.MethodPost, updateMacroValue)

	// PUT
	handleFunc(r, "/api/temperatures/update", http.MethodPut, updateTemperatureProfile)
	handleFunc(r, "/api/temperatures/updateGraph", http.MethodPut, updateTemperatureProfileGraph)
	handleFunc(r, "/api/lcd/modes", http.MethodPut, updateLcdProfile)
	handleFunc(r, "/api/userProfile", http.MethodPut, saveUserProfile)
	handleFunc(r, "/api/keyboard/profile/new", http.MethodPut, saveDeviceProfile)
	handleFunc(r, "/api/macro/new", http.MethodPut, newMacroProfile)
	handleFunc(r, "/api/color/change", http.MethodPut, updateRgbProfile)

	// DELETE
	handleFunc(r, "/api/keyboard/profile/delete", http.MethodDelete, deleteKeyboardProfile)
	handleFunc(r, "/api/macro/value", http.MethodDelete, deleteMacroValue)
	handleFunc(r, "/api/temperatures/delete", http.MethodDelete, deleteTemperatureProfile)
	handleFunc(r, "/api/macro/profile", http.MethodDelete, deleteMacroProfile)

	// Prometheus metrics
	if config.GetConfig().Metrics {
		handleFunc(r, "/api/metrics", http.MethodGet, getDeviceMetrics)
	}

	if config.GetConfig().Frontend {
		handleFunc(r, "/", http.MethodGet, uiIndex)
		handleFunc(r, "/device/", http.MethodGet, uiDeviceOverview)
		handleFunc(r, "/temperature", http.MethodGet, uiTemperatureOverview)
		handleFunc(r, "/temperatureGraphs", http.MethodGet, uiTemperatureGraphOverview)
		handleFunc(r, "/color", http.MethodGet, uiColorOverview)
		handleFunc(r, "/scheduler", http.MethodGet, uiSchedulerOverview)
		handleFunc(r, "/rgb", http.MethodGet, uiRgbEditor)
		handleFunc(r, "/macros", http.MethodGet, uiMacrosOverview)
		handleFunc(r, "/lcd", http.MethodGet, uiLcdOverview)
		handleFunc(r, "/settings", http.MethodGet, uiSettings)
	}
	return r
}

// Init will start a new web server used for metrics and fan control
func Init() {
	headers = []Header{
		{
			Key:   "Cache-Control",
			Value: "no-cache, no-store, must-revalidate",
		},
		{
			Key:   "Pragma",
			Value: "no-cache",
		},
		{
			Key:   "Expires",
			Value: "0",
		},
	}

	if config.GetConfig().ListenPort > 0 {
		templates.Init()
		server = &http.Server{
			Addr: fmt.Sprintf(
				"%s:%v",
				config.GetConfig().ListenAddress,
				config.GetConfig().ListenPort,
			),
			Handler: setRoutes(),
		}

		fmt.Println(
			fmt.Sprintf("[Server] Running REST and WebUI on %s. WebUI is accessible via: http://%s",
				server.Addr,
				server.Addr,
			),
		)
		err := server.ListenAndServe()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Fatal("Unable to start REST server")
		}
	} else {
		logger.Log(logger.Fields{}).Info("REST server is disabled")
	}
}
