package server

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/device"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/server/requests"
	"OpenLinkHub/src/structs"
	"OpenLinkHub/src/templates"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/zcalusic/sysinfo"
	"net/http"
)

// send will process response and send it back to a client
func send(r *structs.Response, w http.ResponseWriter) {
	r.Lock()
	defer r.Unlock()

	for i := range config.GetConfig().Headers {
		w.Header().Add(config.GetConfig().Headers[i].Key, config.GetConfig().Headers[i].Value)
	}

	w.WriteHeader(200)
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
	resp := &structs.Response{
		Code:   http.StatusOK,
		Device: device.GetDevice(),
	}
	send(resp, w)
}

// getDevices returns response on /devices
func getDevices(w http.ResponseWriter, _ *http.Request) {
	resp := &structs.Response{
		Code:    http.StatusOK,
		Devices: device.GetDevice().Devices,
	}
	send(resp, w)
}

// setDeviceSpeed handles device speed changes
func setDeviceSpeed(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeSpeed(r)
	resp := &structs.Response{
		Code:    request.Code,
		Message: request.Message,
	}
	send(resp, w)
}

// setDeviceColor handles device color changes
func setDeviceColor(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeColor(r)
	resp := &structs.Response{
		Code:    request.Code,
		Message: request.Message,
	}
	send(resp, w)
}

func uiDeviceOverview(w http.ResponseWriter, r *http.Request) {
	web := templates.Web{}
	web.Title = "Device Dashboard"
	web.Device = device.GetDevice()

	// System info
	var si sysinfo.SysInfo
	si.GetSysInfo()
	web.SystemInfo = si

	web.Tpl = templates.GetTemplate("overview")
	err := web.Tpl.Execute(w, web)
	if err != nil {
		resp := &structs.Response{
			Code:    http.StatusInternalServerError,
			Message: "unable to serve web content",
		}
		send(resp, w)
	}
}

func setRoutes() *mux.Router {
	r := mux.NewRouter().StrictSlash(true)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	r.Methods(http.MethodGet).Path("/").HandlerFunc(homePage)
	r.Methods(http.MethodGet).Path("/devices").HandlerFunc(getDevices)
	r.Methods(http.MethodPost).Path("/speed").HandlerFunc(setDeviceSpeed)
	r.Methods(http.MethodPost).Path("/color").HandlerFunc(setDeviceColor)
	r.Methods(http.MethodGet).Path("/ui").HandlerFunc(uiDeviceOverview)
	return r
}

// Init will start a new web server used for metrics and fan control
func Init() {
	if config.GetConfig().ListenPort > 0 {
		templates.Init()
		server := &http.Server{
			Addr: fmt.Sprintf(
				"%s:%v",
				config.GetConfig().ListenAddress,
				config.GetConfig().ListenPort,
			),
			Handler: setRoutes(),
		}

		fmt.Println(
			fmt.Sprintf("Running REST and WebUI on %s. WebUI is accessible via: http://%s",
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
