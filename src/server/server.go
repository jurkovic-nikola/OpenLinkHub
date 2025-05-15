package server

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/devices/lcd"
	"OpenLinkHub/src/inputmanager"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/macro"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/scheduler"
	"OpenLinkHub/src/server/requests"
	"OpenLinkHub/src/stats"
	"OpenLinkHub/src/systeminfo"
	"OpenLinkHub/src/temperatures"
	"OpenLinkHub/src/templates"
	"OpenLinkHub/src/version"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"strconv"
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
	promhttp.Handler().ServeHTTP(w, r)
}

// getDevices returns response on /devices
func getDevices(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceOd, valid := vars["deviceOd"]
	if !valid {
		resp := &Response{
			Code:    http.StatusOK,
			Devices: devices.GetDevicesEx(),
		}
		resp.Send(w)
	} else {
		resp := &Response{
			Code:   http.StatusOK,
			Device: devices.GetDevice(deviceOd),
		}
		resp.Send(w)
	}
}

// getDeviceLed returns response on /led
func getDeviceLed(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceOd, valid := vars["deviceOd"]
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
			Data:   devices.GetDeviceLedData(deviceOd),
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
	vars := mux.Vars(r)
	macroId, valid := vars["macroId"]
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
				Message: "Unable to parse macroId",
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
	vars := mux.Vars(r)
	keyIndex, valid := vars["keyIndex"]
	if !valid {
		resp := &Response{
			Code:    http.StatusOK,
			Status:  0,
			Message: "Unable to parse keyIndex",
		}
		resp.Send(w)
	} else {
		val, err := strconv.ParseUint(keyIndex, 10, 8) // base 10, 8-bit size
		if err != nil {
			resp := &Response{
				Code:    http.StatusOK,
				Status:  0,
				Message: "Unable to parse macroId",
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

	vars := mux.Vars(r)
	profile, valid := vars["profile"]
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
				Message: "No such temperature profile",
			}
		}
	}
	resp.Send(w)
}

// getTemperatureGraph returns response on for temperature graph
func getTemperatureGraph(w http.ResponseWriter, r *http.Request) {
	resp := &Response{}

	vars := mux.Vars(r)
	profile, valid := vars["profile"]
	if !valid {
		resp = &Response{
			Code:    http.StatusOK,
			Status:  0,
			Message: "No such temperature profile",
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
				Message: "No such temperature profile",
			}
		}
	}
	resp.Send(w)
}

// getColor returns response on /color
func getColor(w http.ResponseWriter, r *http.Request) {
	resp := &Response{}

	vars := mux.Vars(r)
	deviceId, valid := vars["deviceId"]
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
				Message: "No such RGB profile",
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

// uiDeviceOverview handles device overview
func uiDeviceOverview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceOd, valid := vars["deviceOd"]
	if !valid {
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Status:  0,
			Message: "Unable to process device request. Please try again",
		}
		resp.Send(w)
	}

	device := devices.GetDevice(deviceOd)
	template := devices.GetDeviceTemplate(device)
	if len(template) == 0 {
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Status:  0,
			Message: "Unable to process device request. Please try again",
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
			Message: "unable to serve web content",
		}
		resp.Send(w)
	}
}

// uiIndex handles index page
func uiIndex(w http.ResponseWriter, _ *http.Request) {
	web := templates.Web{}
	web.Title = "Device Dashboard"
	web.Devices = devices.GetDevices()
	web.BuildInfo = version.GetBuildInfo()
	web.SystemInfo = systeminfo.GetInfo()
	web.CpuTemp = dashboard.GetDashboard().TemperatureToString(temperatures.GetCpuTemperature())
	web.GpuTemp = dashboard.GetDashboard().TemperatureToString(temperatures.GetGpuTemperature())
	web.Dashboard = dashboard.GetDashboard()
	web.BatteryStats = stats.GetBatteryStats()
	web.Page = "index"

	t := templates.GetTemplate()

	for header := range headers {
		w.Header().Set(headers[header].Key, headers[header].Value)
	}

	err := t.ExecuteTemplate(w, "index.html", web)
	if err != nil {
		fmt.Println(err)
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Message: "unable to serve web content",
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
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Message: "unable to serve web content",
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
			Message: "unable to serve web content",
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
			Message: "unable to serve web content",
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
			Message: "unable to serve web content",
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
			Message: "unable to serve web content",
		}
		resp.Send(w)
	}
}

// uiDocumentationOverview handles overview of documentation
func uiDocumentationOverview(w http.ResponseWriter, _ *http.Request) {
	web := templates.Web{}
	web.Title = "Device Dashboard"
	web.Devices = devices.GetDevices()
	web.Configuration = config.GetConfig()
	web.BuildInfo = version.GetBuildInfo()
	web.SystemInfo = systeminfo.GetInfo()
	web.Page = "documentation"
	t := templates.GetTemplate()

	for header := range headers {
		w.Header().Set(headers[header].Key, headers[header].Value)
	}

	err := t.ExecuteTemplate(w, "docs.html", web)

	if err != nil {
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Message: "unable to serve web content",
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
			Message: "unable to serve web content",
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
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Message: "unable to serve web content",
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
	web.Page = "settings"

	t := templates.GetTemplate()

	for header := range headers {
		w.Header().Set(headers[header].Key, headers[header].Value)
	}

	err := t.ExecuteTemplate(w, "settings.html", web)
	if err != nil {
		fmt.Println(err)
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Message: "unable to serve web content",
		}
		resp.Send(w)
	}
}

// setRoutes will set up all routes
func setRoutes() *mux.Router {
	r := mux.NewRouter().StrictSlash(true)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	// API
	r.Methods(http.MethodGet).Path("/api/").
		HandlerFunc(homePage)
	r.Methods(http.MethodGet).Path("/api/cpuTemp").
		HandlerFunc(getCpuTemperature)
	r.Methods(http.MethodGet).Path("/api/cpuTemp/clean").
		HandlerFunc(getCpuTemperatureClean)
	r.Methods(http.MethodGet).Path("/api/gpuTemp").
		HandlerFunc(getGpuTemperature)
	r.Methods(http.MethodGet).Path("/api/gpuTemp/clean").
		HandlerFunc(getGpuTemperatureClean)
	r.Methods(http.MethodGet).Path("/api/storageTemp").
		HandlerFunc(getStorageTemperature)
	r.Methods(http.MethodGet).Path("/api/batteryStats").
		HandlerFunc(getBatteryStats)
	r.Methods(http.MethodGet).Path("/api/devices").
		HandlerFunc(getDevices)
	r.Methods(http.MethodGet).Path("/api/devices/{deviceOd}").
		HandlerFunc(getDevices)
	r.Methods(http.MethodGet).Path("/api/color").
		HandlerFunc(getColor)
	r.Methods(http.MethodGet).Path("/api/color/{deviceId}").
		HandlerFunc(getColor)
	r.Methods(http.MethodPut).Path("/api/color").
		HandlerFunc(updateRgbProfile)
	r.Methods(http.MethodGet).Path("/api/temperatures").
		HandlerFunc(getTemperature)
	r.Methods(http.MethodGet).Path("/api/temperatures/{profile}").
		HandlerFunc(getTemperature)
	r.Methods(http.MethodGet).Path("/api/temperatures/graph/{profile}").
		HandlerFunc(getTemperatureGraph)
	r.Methods(http.MethodGet).Path("/api/input/media").
		HandlerFunc(getMediaKeys)
	r.Methods(http.MethodGet).Path("/api/input/keyboard").
		HandlerFunc(getInputKeys)
	r.Methods(http.MethodGet).Path("/api/input/mouse").
		HandlerFunc(getMouseButtons)
	r.Methods(http.MethodPost).Path("/api/temperatures").
		HandlerFunc(newTemperatureProfile)
	r.Methods(http.MethodPut).Path("/api/temperatures").
		HandlerFunc(updateTemperatureProfile)
	r.Methods(http.MethodPut).Path("/api/temperatures/graph").
		HandlerFunc(updateTemperatureProfileGraph)
	r.Methods(http.MethodDelete).Path("/api/temperatures").
		HandlerFunc(deleteTemperatureProfile)
	r.Methods(http.MethodPost).Path("/api/speed").
		HandlerFunc(setDeviceSpeed)
	r.Methods(http.MethodPost).Path("/api/speed/manual").
		HandlerFunc(setManualDeviceSpeed)
	r.Methods(http.MethodPost).Path("/api/color").
		HandlerFunc(setDeviceColor)
	r.Methods(http.MethodPost).Path("/api/color/hardware").
		HandlerFunc(setDeviceHardwareColor)
	r.Methods(http.MethodPost).Path("/api/hub/strip").
		HandlerFunc(setDeviceStrip)
	r.Methods(http.MethodPost).Path("/api/hub/type").
		HandlerFunc(setExternalHubDeviceType)
	r.Methods(http.MethodPost).Path("/api/hub/amount").
		HandlerFunc(setExternalHubDeviceAmount)
	r.Methods(http.MethodPost).Path("/api/label").
		HandlerFunc(setDeviceLabel)
	r.Methods(http.MethodPost).Path("/api/lcd").
		HandlerFunc(setDeviceLcd)
	r.Methods(http.MethodPost).Path("/api/lcd/device").
		HandlerFunc(changeDeviceLcd)
	r.Methods(http.MethodPost).Path("/api/lcd/rotation").
		HandlerFunc(setDeviceLcdRotation)
	r.Methods(http.MethodPost).Path("/api/lcd/profile").
		HandlerFunc(setDeviceLcdProfile)
	r.Methods(http.MethodPost).Path("/api/lcd/image").
		HandlerFunc(setDeviceLcdImage)
	r.Methods(http.MethodPut).Path("/api/lcd/modes").
		HandlerFunc(updateLcdProfile)
	r.Methods(http.MethodPut).Path("/api/userProfile").
		HandlerFunc(saveUserProfile)
	r.Methods(http.MethodPost).Path("/api/userProfile").
		HandlerFunc(changeUserProfile)
	r.Methods(http.MethodPost).Path("/api/brightness").
		HandlerFunc(changeBrightness)
	r.Methods(http.MethodPost).Path("/api/brightness/gradual").
		HandlerFunc(changeBrightnessGradual)
	r.Methods(http.MethodPost).Path("/api/position").
		HandlerFunc(changePosition)
	r.Methods(http.MethodGet).Path("/api/dashboard").
		HandlerFunc(getDashboardSettings)
	r.Methods(http.MethodPost).Path("/api/dashboard").
		HandlerFunc(setDashboardSettings)
	r.Methods(http.MethodPost).Path("/api/argb").
		HandlerFunc(setARGBDevice)
	r.Methods(http.MethodPost).Path("/api/keyboard/color").
		HandlerFunc(setKeyboardColor)
	r.Methods(http.MethodPost).Path("/api/misc/color").
		HandlerFunc(setMiscColor)
	r.Methods(http.MethodPut).Path("/api/keyboard/profile/new").
		HandlerFunc(saveDeviceProfile)
	r.Methods(http.MethodPost).Path("/api/keyboard/profile/change").
		HandlerFunc(changeKeyboardProfile)
	r.Methods(http.MethodPost).Path("/api/keyboard/profile/save").
		HandlerFunc(saveDeviceProfile)
	r.Methods(http.MethodDelete).Path("/api/keyboard/profile/delete").
		HandlerFunc(deleteKeyboardProfile)
	r.Methods(http.MethodPost).Path("/api/keyboard/layout").
		HandlerFunc(changeKeyboardLayout)
	r.Methods(http.MethodPost).Path("/api/keyboard/dial").
		HandlerFunc(changeControlDial)
	r.Methods(http.MethodPost).Path("/api/keyboard/sleep").
		HandlerFunc(changeSleepMode)
	r.Methods(http.MethodPost).Path("/api/keyboard/pollingRate").
		HandlerFunc(changePollingRate)
	r.Methods(http.MethodPost).Path("/api/scheduler/rgb").
		HandlerFunc(changeRgbScheduler)
	r.Methods(http.MethodPost).Path("/api/psu/speed").
		HandlerFunc(changePsuFanMode)
	r.Methods(http.MethodPost).Path("/api/mouse/dpi").
		HandlerFunc(saveMouseDpi)
	r.Methods(http.MethodPost).Path("/api/mouse/zoneColors").
		HandlerFunc(saveMouseZoneColors)
	r.Methods(http.MethodPost).Path("/api/mouse/dpiColors").
		HandlerFunc(saveMouseDpiColors)
	r.Methods(http.MethodPost).Path("/api/mouse/sleep").
		HandlerFunc(changeSleepMode)
	r.Methods(http.MethodPost).Path("/api/mouse/pollingRate").
		HandlerFunc(changePollingRate)
	r.Methods(http.MethodPost).Path("/api/mouse/angleSnapping").
		HandlerFunc(changeAngleSnapping)
	r.Methods(http.MethodPost).Path("/api/mouse/buttonOptimization").
		HandlerFunc(changeButtonOptimization)
	r.Methods(http.MethodPost).Path("/api/mouse/updateKeyAssignment").
		HandlerFunc(changeKeyAssignment)
	r.Methods(http.MethodPost).Path("/api/headset/zoneColors").
		HandlerFunc(saveHeadsetZoneColors)
	r.Methods(http.MethodPost).Path("/api/headset/sleep").
		HandlerFunc(changeSleepMode)
	r.Methods(http.MethodPost).Path("/api/headset/muteIndicator").
		HandlerFunc(changeMuteIndicator)
	r.Methods(http.MethodGet).Path("/api/led/{deviceOd}").
		HandlerFunc(getDeviceLed)
	r.Methods(http.MethodGet).Path("/api/led/").
		HandlerFunc(getDeviceLed)
	r.Methods(http.MethodPost).Path("/api/led/").
		HandlerFunc(updateDeviceLed)
	r.Methods(http.MethodGet).Path("/api/macro/{macroId}").
		HandlerFunc(getMacro)
	r.Methods(http.MethodGet).Path("/api/macro/").
		HandlerFunc(getMacro)
	r.Methods(http.MethodGet).Path("/api/macro/keyInfo/{keyIndex}").
		HandlerFunc(getKeyName)
	r.Methods(http.MethodDelete).Path("/api/macro/value").
		HandlerFunc(deleteMacroValue)
	r.Methods(http.MethodDelete).Path("/api/macro/").
		HandlerFunc(deleteMacroProfile)
	r.Methods(http.MethodPut).Path("/api/macro/").
		HandlerFunc(newMacroProfile)
	r.Methods(http.MethodPost).Path("/api/macro/").
		HandlerFunc(newMacroProfileValue)

	// Prometheus metrics
	if config.GetConfig().Metrics {
		r.Methods(http.MethodGet).Path("/api/metrics").
			HandlerFunc(getDeviceMetrics)
	}

	if config.GetConfig().Frontend {
		// Frontend
		r.Methods(http.MethodGet).Path("/").
			HandlerFunc(uiIndex)
		r.Methods(http.MethodGet).Path("/device/{deviceOd}").
			HandlerFunc(uiDeviceOverview)
		r.Methods(http.MethodGet).Path("/temperature").
			HandlerFunc(uiTemperatureOverview)
		r.Methods(http.MethodGet).Path("/temperatureGraphs").
			HandlerFunc(uiTemperatureGraphOverview)
		r.Methods(http.MethodGet).Path("/docs").
			HandlerFunc(uiDocumentationOverview)
		r.Methods(http.MethodGet).Path("/color").
			HandlerFunc(uiColorOverview)
		r.Methods(http.MethodGet).Path("/scheduler").
			HandlerFunc(uiSchedulerOverview)
		r.Methods(http.MethodGet).Path("/rgb").
			HandlerFunc(uiRgbEditor)
		r.Methods(http.MethodGet).Path("/macros").
			HandlerFunc(uiMacrosOverview)
		r.Methods(http.MethodGet).Path("/lcd").
			HandlerFunc(uiLcdOverview)
		r.Methods(http.MethodGet).Path("/settings").
			HandlerFunc(uiSettings)
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
