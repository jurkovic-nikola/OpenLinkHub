package rgb

import (
	"OpenICUELinkHub/src/config"
	"OpenICUELinkHub/src/device/rgb/circle"
	"OpenICUELinkHub/src/device/rgb/colorpulse"
	"OpenICUELinkHub/src/device/rgb/colorshift"
	"OpenICUELinkHub/src/device/rgb/colorwarp"
	"OpenICUELinkHub/src/device/rgb/flickering"
	"OpenICUELinkHub/src/device/rgb/heartbeat"
	"OpenICUELinkHub/src/device/rgb/rainbow"
	"OpenICUELinkHub/src/device/rgb/spinner"
	"OpenICUELinkHub/src/device/rgb/watercolor"
	"OpenICUELinkHub/src/structs"
)

var ActiveRGBMode = ""

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

// Stop will terminate any running rgb effect after a device wakes up
func Stop() {
	switch ActiveRGBMode {
	case "rainbow":
		rainbow.Stop()
	case "watercolor":
		watercolor.Stop()
	case "colorshift":
		colorshift.Stop()
	case "colorpulse":
		colorpulse.Stop()
	case "circle", "circleshift":
		circle.Stop()
	case "flickering":
		flickering.Stop()
	case "colorwarp":
		colorwarp.Stop()
	case "snipper":
		spinner.Stop()
	case "heartbeat":
		heartbeat.Stop()
	}
}
