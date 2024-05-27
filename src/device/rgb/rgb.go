package rgb

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/device/common"
	"OpenLinkHub/src/structs"
	"time"
)

type ActiveRGB struct {
	lightChannels          int
	smoothness             int
	rgbModeSpeed           float64
	rgbEndColor            *structs.Color
	rgbStartColor          *structs.Color
	bts                    float64
	rgbLoopDuration        time.Duration
	rgbCustomColor         bool
	lightChannelsPerDevice map[int][]int
	exit                   chan bool
}

// IsGRBEnabled will return true or false if RGB is enabled
func IsGRBEnabled() bool {
	return config.GetRGB().UseRgbEffects
}

// GetRGBModeName will return the current rgb mode name
func GetRGBModeName() string {
	return config.GetRGB().RGBMode
}

// GetRGBMode will return structs.RGBModes struct
func GetRGBMode() *structs.RGBModes {
	rgbMode := config.GetRGB().RGBMode
	if val, ok := config.GetRGB().RGBModes[rgbMode]; ok {
		return &val
	}
	return nil
}

// interpolateColor performs linear interpolation between two colors
func interpolateColor(c1, c2 *structs.Color, t float64) *structs.Color {
	return &structs.Color{
		Red:   common.Lerp(c1.Red, c2.Red, t),
		Green: common.Lerp(c1.Green, c2.Green, t),
		Blue:  common.Lerp(c1.Blue, c2.Blue, t),
	}
}

// New will create new ActiveRGB struct for RGB control
func New(
	lightChannels int,
	rgbModeSpeed float64,
	rgbStartColor *structs.Color,
	rgbEndColor *structs.Color,
	bts float64,
	smoothness int,
	rgbLoopDuration time.Duration,
	rgbCustomColor bool,
	lightChannelsPerDevice map[int][]int,
) *ActiveRGB {

	return &ActiveRGB{
		lightChannels:          lightChannels,
		smoothness:             smoothness,
		rgbModeSpeed:           rgbModeSpeed,
		rgbStartColor:          rgbStartColor,
		rgbEndColor:            rgbEndColor,
		bts:                    bts,
		rgbLoopDuration:        rgbLoopDuration,
		rgbCustomColor:         rgbCustomColor,
		lightChannelsPerDevice: lightChannelsPerDevice,
		exit:                   make(chan bool),
	}
}

// Stop will send command to exit RGB for {} loop
func (r *ActiveRGB) Stop() {
	r.exit <- true
}
