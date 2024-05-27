package config

import (
	"OpenLinkHub/src/structs"
	"encoding/json"
	"fmt"
	"os"
)

var configuration structs.Configuration
var devices []structs.DeviceList
var rgb structs.RGB
var temperatures structs.Temperatures
var customChannels structs.CustomChannels

// Init will initialize a new config object
func Init() {
	jsonParser := loadJSON("config.json")
	if err := jsonParser.Decode(&configuration); err != nil {
		panic(err.Error())
	}

	// Load additional configuration files
	loadDevices()        // Supported devices
	loadRgb()            // RGB data
	loadTemperatures()   // Temperature data
	loadCustomChannels() // Custom channel data (advanced)

	if customChannels.UseCustomChannelIdSpeed {
		fmt.Println("Ignoring standalone flag due to useCustomChannelIdSpeed is set to true")
		configuration.Standalone = false
	}

	if rgb.UseRgbEffects {
		rgbMode := rgb.RGBMode
		if _, ok := rgb.RGBModes[rgbMode]; !ok {
			fmt.Println("RGB mode not found in configuration. Setting to default (nothing)")
			rgb.RGBMode = ""
		} else {
			fmt.Println(fmt.Sprintf("RGB mode %s found in configuration. Setting UseCustomChannelIdColor to false", rgbMode))
			customChannels.UseCustomChannelIdColor = false
		}
	}
}

// loadJSON will load given JSON file and return json.Decoder. In case of error, program will panic due to
// the nature of requirements for config files
func loadJSON(filename string) *json.Decoder {
	pwd, _ := os.Getwd()
	cfg := pwd + "/configs/" + filename
	f, err := os.Open(cfg)
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("Loading configuration file:", cfg)
	return json.NewDecoder(f)
}

// loadDevices will load a list of supported devices
func loadDevices() {
	jsonParser := loadJSON("devices.json")
	if err := jsonParser.Decode(&devices); err != nil {
		panic(err.Error())
	}
}

// loadDevices will load a list of rgb
func loadRgb() {
	jsonParser := loadJSON("rgb.json")
	if err := jsonParser.Decode(&rgb); err != nil {
		panic(err.Error())
	}
}

// loadTemperatures will load a temperature data
func loadTemperatures() {
	jsonParser := loadJSON("temperatures.json")
	if err := jsonParser.Decode(&temperatures); err != nil {
		panic(err.Error())
	}
}

// loadTemperatures will load a temperature data
func loadCustomChannels() {
	jsonParser := loadJSON("custom.json")
	if err := jsonParser.Decode(&customChannels); err != nil {
		panic(err.Error())
	}
}

// GetConfig will return structs.Configuration struct
func GetConfig() structs.Configuration {
	return configuration
}

// GetDevices will return structs.DeviceList array
func GetDevices() []structs.DeviceList {
	return devices
}

// GetRGB will return structs.RGB
func GetRGB() structs.RGB {
	return rgb
}

// GetTemperatures will return structs.Temperatures
func GetTemperatures() structs.Temperatures {
	return temperatures
}

// GetCustomChannels will return structs.CustomChannels
func GetCustomChannels() structs.CustomChannels {
	return customChannels
}
