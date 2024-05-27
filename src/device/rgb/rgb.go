package rgb

import (
	"OpenICUELinkHub/src/config"
	"OpenICUELinkHub/src/structs"
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
