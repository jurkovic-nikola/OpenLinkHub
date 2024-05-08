package server

import (
	"OpenICUELinkHub/src/config"
	"OpenICUELinkHub/src/device"
	"OpenICUELinkHub/src/logger"
	"OpenICUELinkHub/src/server/requests"
	"OpenICUELinkHub/src/structs"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
)

// Send will process response and send it back to a client
func Send(r *structs.Response, w http.ResponseWriter) {
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

// HomePage returns response on /
func HomePage(w http.ResponseWriter, _ *http.Request) {
	resp := &structs.Response{
		Code:   http.StatusOK,
		Device: device.GetDevice(),
	}
	Send(resp, w)
}

// Devices returns response on /devices
func Devices(w http.ResponseWriter, _ *http.Request) {
	resp := &structs.Response{
		Code:    http.StatusOK,
		Devices: device.GetDevice().Devices,
	}
	Send(resp, w)
}

// SetDeviceSpeed handles device speed changes
func SetDeviceSpeed(w http.ResponseWriter, r *http.Request) {
	request := requests.ProcessChangeSpeed(r)
	resp := &structs.Response{
		Code:    request.Code,
		Message: request.Message,
	}
	Send(resp, w)
}

func Routes() *mux.Router {
	r := mux.NewRouter().StrictSlash(true)
	r.Methods(http.MethodGet).Path("/").HandlerFunc(HomePage)
	r.Methods(http.MethodGet).Path("/devices").HandlerFunc(Devices)
	r.Methods(http.MethodPost).Path("/speed").HandlerFunc(SetDeviceSpeed)

	return r
}

// Init will start a new web server used for metrics and fan control
func Init() {
	server := &http.Server{
		Addr: fmt.Sprintf(
			"%s:%v",
			config.GetConfig().ListenAddress,
			config.GetConfig().ListenPort,
		),
		Handler: Routes(),
	}

	err := server.ListenAndServe()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Fatal("Unable to start REST server")
	}
}
