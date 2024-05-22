package rgb

import (
	"OpenICUELinkHub/src/config"
	"OpenICUELinkHub/src/structs"
)

var (
	rgbMode  = ""
	rgbModes = map[string]structs.RGBModes{}
)

// GetRGBSpeed will return RGB speed
func GetRGBSpeed() uint8 {
	speed := rgbModes[rgbMode].Speed
	if speed > 10 {
		return 10
	} else if speed < 1 {
		return 3
	}
	return speed
}

// GetRGBBrightness will return brightness value
func GetRGBBrightness() float64 {
	brightness := rgbModes[rgbMode].Brightness
	if brightness > 1 {
		return 1
	} else if brightness < 0 {
		return 0.1
	}
	return brightness
}

// IsRGBEnabled will return true or false if RGB mode is enabled
func IsRGBEnabled() bool {
	rgbMode = config.GetConfig().RGBMode
	rgbModes = config.GetConfig().RGBModes

	if _, ok := rgbModes[rgbMode]; ok {
		return true
	}
	return false
}

// GetRGBModeName will return the current rgb mode name
func GetRGBModeName() string {
	return rgbMode
}
