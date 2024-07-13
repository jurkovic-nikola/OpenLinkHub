package server

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/devices"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/rgb"
	"OpenLinkHub/src/server/requests"
	"OpenLinkHub/src/temperatures"
	"OpenLinkHub/src/templates"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/zcalusic/sysinfo"
	"net/http"
	"sync"
)

// Response contains data what is sent back to a client
type Response struct {
	sync.Mutex
	Code    int         `json:"code"`
	Status  int         `json:"status"`
	Message string      `json:"message,omitempty"`
	Device  interface{} `json:"device,omitempty"`
	Devices interface{} `json:"devices,omitempty"`
	Data    interface{} `json:"data,omitempty"` // For dataTables
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

// getCpuTemperature will return current cpu temperature
func getCpuTemperature(w http.ResponseWriter, _ *http.Request) {
	resp := &Response{
		Code: http.StatusOK,
		Data: temperatures.GetCpuTemperature(),
	}
	resp.Send(w)
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

	t := templates.GetTemplate()
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

	// System info
	var si sysinfo.SysInfo
	si.GetSysInfo()
	web.SystemInfo = si
	web.CpuTemp = temperatures.GetCpuTemperature()

	t := templates.GetTemplate()
	err := t.ExecuteTemplate(w, "index.html", web)
	if err != nil {
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
	web.Temperatures = temperatures.GetTemperatureProfiles()

	t := templates.GetTemplate()
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

	t := templates.GetTemplate()
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

	t := templates.GetTemplate()
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
	r.Methods(http.MethodGet).Path("/api/cputemp").
		HandlerFunc(getCpuTemperature)
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
	header := &Header{
		Key:   "Content-Type",
		Value: "application/json",
	}
	headers = append(headers, *header)

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
