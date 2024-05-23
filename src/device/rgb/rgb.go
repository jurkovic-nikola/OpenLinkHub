package rgb

import (
	"OpenICUELinkHub/src/config"
	"OpenICUELinkHub/src/structs"
)

func IsGRBEnabled() bool {
	return config.GetConfig().UseRgbEffects
}

// GetRGBModeName will return the current rgb mode name
func GetRGBModeName() string {
	return config.GetConfig().RGBMode
}

func GetRGBMode() *structs.RGBModes {
	rgbMode := config.GetConfig().RGBMode
	if val, ok := config.GetConfig().RGBModes[rgbMode]; ok {
		return &val
	}
	return nil
}
