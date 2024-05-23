package config

import (
	"OpenICUELinkHub/src/structs"
	"encoding/json"
	"fmt"
	"os"
)

var configuration structs.Configuration

// Init will initialize a new config object
func Init() {
	pwd, _ := os.Getwd()

	configFile, err := os.Open(pwd + "/config.json")
	if err != nil {
		panic(err.Error())
	}

	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&configuration); err != nil {
		panic(err.Error())
	}

	if configuration.UseCustomChannelIdSpeed {
		fmt.Println("Ignoring standalone flag due to useCustomChannelIdSpeed is set to true")
		configuration.Standalone = false
	}

	if configuration.UseRgbEffects {
		rgbMode := configuration.RGBMode
		if _, ok := configuration.RGBModes[rgbMode]; !ok {
			fmt.Println("RGB mode not found in configuration. Setting to default (nothing)")
			configuration.RGBMode = ""
		} else {
			fmt.Println(fmt.Sprintf("RGB mode %s found in configuration. Setting UseCustomChannelIdColor to false", rgbMode))
			configuration.UseCustomChannelIdColor = false
		}
	}
}

// GetConfig will return Configuration struct
func GetConfig() structs.Configuration {
	return configuration
}
