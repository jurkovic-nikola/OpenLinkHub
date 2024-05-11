package requests

import (
	"OpenICUELinkHub/src/config"
	"OpenICUELinkHub/src/device"
	"OpenICUELinkHub/src/logger"
	"OpenICUELinkHub/src/structs"
	"encoding/json"
	"net/http"
)

// ProcessChangeSpeed will process POST request from a client for fan speed change
func ProcessChangeSpeed(r *http.Request) *structs.Payload {
	var value uint16
	req := &structs.Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &structs.Payload{
			Message: "Unable to validate your request. Please try again!",
			Code:    http.StatusBadRequest,
		}
	}

	// When a program runs in standalone, it measures temperature on its own and modifies the speed of devices based on
	// a user-defined curve in configuration file.
	if config.GetConfig().Standalone {
		return &structs.Payload{
			Message: "Standalone mode active, speed modifications are not possible",
			Code:    http.StatusMethodNotAllowed,
		}
	}

	if req.ChannelId < 1 {
		return &structs.Payload{
			Message: "Non-existing channelId",
			Code:    http.StatusBadRequest,
		}
	}

	if _, ok := device.GetDevice().Devices[req.ChannelId]; !ok {
		return &structs.Payload{
			Message: "Non-existing channelId",
			Code:    http.StatusBadRequest,
		}
	}

	if req.Mode < 0 || req.Mode > 1 {
		return &structs.Payload{
			Message: "Non-existing speed mode",
			Code:    http.StatusBadRequest,
		}
	}

	dev := device.GetDevice().Devices[req.ChannelId]
	if dev.Type == 0x07 && req.Mode == 1 { // Liquid cooler (AIO)
		return &structs.Payload{
			Message: "Pump speed can not be controlled via RPM",
			Code:    http.StatusMethodNotAllowed,
		}
	}

	value = req.Value
	if req.Mode == 0 && req.Value < 20 { // Percent mode
		value = 20
	}

	if req.Mode == 0 && req.Value > 100 { // Percent mode
		value = 90
	}

	if req.Mode == 1 && req.Value < 300 { // RPM mode
		value = 300
	}

	if req.Mode == 1 && req.Value > 3000 { // RPM mode
		value = 3000
	}

	if device.SetDeviceSpeed(req.ChannelId, value, req.Mode) == 1 {
		return &structs.Payload{
			Message: "Device speed successfully changed",
			Code:    http.StatusOK,
		}
	} else {
		return &structs.Payload{
			Message: "Unable to change device speed. Check stdout.log for more details",
			Code:    http.StatusOK,
		}
	}
}

// ProcessChangeColor will process POST request from a client for color change
func ProcessChangeColor(r *http.Request) *structs.Payload {
	req := &structs.Payload{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Log(map[string]interface{}{"error": err}).Error("Unable to decode JSON")
		return &structs.Payload{
			Message: "Unable to validate your request. Please try again!",
			Code:    http.StatusBadRequest,
		}
	}

	if config.GetConfig().Standalone {
		return &structs.Payload{
			Message: "Standalone mode active, color modifications are not possible",
			Code:    http.StatusMethodNotAllowed,
		}
	}

	if config.GetConfig().UseCustomChannelIdColor {
		return &structs.Payload{
			Message: "UseCustomChannelIdColor mode active, color modifications are not possible.",
			Code:    http.StatusMethodNotAllowed,
		}
	}

	if req.ChannelId < 0 {
		return &structs.Payload{
			Message: "Non-existing channelId",
			Code:    http.StatusBadRequest,
		}
	}

	if req.ChannelId != 0 {
		if _, ok := device.GetDevice().Devices[req.ChannelId]; !ok {
			return &structs.Payload{
				Message: "Non-existing channelId",
				Code:    http.StatusBadRequest,
			}
		}
	}

	if req.Mode < 0 || req.Mode > 1 {
		return &structs.Payload{
			Message: "Non-existing speed mode",
			Code:    http.StatusBadRequest,
		}
	}

	dev := device.GetDevice().Devices[req.ChannelId]
	if dev.Type == 0x07 && req.Mode == 1 { // Liquid cooler (AIO)
		return &structs.Payload{
			Message: "Pump speed can not be controlled via RPM",
			Code:    http.StatusBadRequest,
		}
	}

	color := &structs.Color{
		Red:        req.Color.Red,
		Green:      req.Color.Green,
		Blue:       req.Color.Blue,
		Brightness: req.Color.Brightness,
	}
	device.SetDeviceColor(req.ChannelId, color)

	return &structs.Payload{
		Message: "Device color successfully changed",
		Code:    http.StatusOK,
	}
}
