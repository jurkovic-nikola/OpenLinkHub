package server

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/dashboard"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/server/requests"
	"OpenLinkHub/src/systeminfo"
	"OpenLinkHub/src/temperatures"
	"OpenLinkHub/src/templates"
	"OpenLinkHub/src/version"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
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
		w.WriteHeader(http.StatusInternalServerError)
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

// getDeviceMetrics will return a list device metrics in prometheus format
func getDeviceMetrics(w http.ResponseWriter, r *http.Request) {
	devices.UpdateDeviceMetrics()
	promhttp.Handler().ServeHTTP(w, r)
}

// getDevices returns response on /devices
func getDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceOd, valid := vars["deviceOd"]
	if !valid {
		resp := &Response{
			Code:    http.StatusOK,
			Devices: devices.GetDevices(),
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

// getColor returns response on /color
func getColor(w http.ResponseWriter, r *http.Request) {
	resp := &Response{}

	vars := mux.Vars(r)
	profile, valid := vars["profile"]
	if !valid {
		resp = &Response{
			Code:   http.StatusOK,
			Status: 0,
			Data:   rgb.GetRgbProfiles(),
		}
	} else {
		if rgbProfile := rgb.GetRgbProfile(profile); rgbProfile != nil {
			resp = &Response{
				Code:   http.StatusOK,
				Status: 1,
				Data:   rgbProfile,
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
func getDashboardSettings(w http.ResponseWriter, r *http.Request) {
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

// saveKeyboardProfile handles a new keyboard profile
func saveKeyboardProfile(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessSaveKeyboardProfile(r)
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

// uiDeviceOverview handles device overview
func uiDeviceOverview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceOd, valid := vars["deviceOd"]
	if !valid {
		resp := &Response{
			Code:    http.StatusInternalServerError,
			Status:  0,
			Message: "Unable to process temperature request. Please try again",
		}
		resp.Send(w)
	}

	web := templates.Web{}
	web.Title = "Device Dashboard"
	web.Devices = devices.GetDevices()
	web.Device = devices.GetDevice(deviceOd)
	web.Temperatures = temperatures.GetTemperatureProfiles()
	web.Rgb = rgb.GetRGB().Profiles
	web.BuildInfo = version.GetBuildInfo()
	web.SystemInfo = systeminfo.GetInfo()
	web.CpuTemp = dashboard.GetDashboard().TemperatureToString(temperatures.GetCpuTemperature())
	web.GpuTemp = dashboard.GetDashboard().TemperatureToString(temperatures.GetGpuTemperature())
	t := templates.GetTemplate()

	for header := range headers {
		w.Header().Set(headers[header].Key, headers[header].Value)
	}

	err := t.ExecuteTemplate(w, "devices.html", web)
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
	t := templates.GetTemplate()

	for header := range headers {
		w.Header().Set(headers[header].Key, headers[header].Value)
	}

	err := t.ExecuteTemplate(w, "temperature.html", web)
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
	r.Methods(http.MethodGet).Path("/api/devices").
		HandlerFunc(getDevice)
	r.Methods(http.MethodGet).Path("/api/devices/{deviceOd}").
		HandlerFunc(getDevice)
	r.Methods(http.MethodGet).Path("/api/color").
		HandlerFunc(getColor)
	r.Methods(http.MethodGet).Path("/api/color/{profile}").
		HandlerFunc(getColor)
	r.Methods(http.MethodGet).Path("/api/temperatures").
		HandlerFunc(getTemperature)
	r.Methods(http.MethodGet).Path("/api/temperatures/{profile}").
		HandlerFunc(getTemperature)
	r.Methods(http.MethodPost).Path("/api/temperatures").
		HandlerFunc(newTemperatureProfile)
	r.Methods(http.MethodPut).Path("/api/temperatures").
		HandlerFunc(updateTemperatureProfile)
	r.Methods(http.MethodDelete).Path("/api/temperatures").
		HandlerFunc(deleteTemperatureProfile)
	r.Methods(http.MethodPost).Path("/api/speed").
		HandlerFunc(setDeviceSpeed)
	r.Methods(http.MethodPost).Path("/api/speed/manual").
		HandlerFunc(setManualDeviceSpeed)
	r.Methods(http.MethodPost).Path("/api/color").
		HandlerFunc(setDeviceColor)
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
	r.Methods(http.MethodPut).Path("/api/userProfile").
		HandlerFunc(saveUserProfile)
	r.Methods(http.MethodPost).Path("/api/userProfile").
		HandlerFunc(changeUserProfile)
	r.Methods(http.MethodPost).Path("/api/brightness").
		HandlerFunc(changeBrightness)
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
	r.Methods(http.MethodPut).Path("/api/keyboard/profile/new").
		HandlerFunc(saveKeyboardProfile)
	r.Methods(http.MethodPost).Path("/api/keyboard/profile/change").
		HandlerFunc(changeKeyboardProfile)
	r.Methods(http.MethodPost).Path("/api/keyboard/profile/save").
		HandlerFunc(saveKeyboardProfile)
	r.Methods(http.MethodDelete).Path("/api/keyboard/profile/delete").
		HandlerFunc(deleteKeyboardProfile)
	r.Methods(http.MethodPost).Path("/api/keyboard/layout").
		HandlerFunc(changeKeyboardLayout)
	r.Methods(http.MethodPost).Path("/api/keyboard/dial").
		HandlerFunc(changeControlDial)

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
		r.Methods(http.MethodGet).Path("/docs").
			HandlerFunc(uiDocumentationOverview)
		r.Methods(http.MethodGet).Path("/color").
			HandlerFunc(uiColorOverview)
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
