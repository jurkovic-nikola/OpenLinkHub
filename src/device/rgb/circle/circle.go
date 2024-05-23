package circle

import (
	"OpenICUELinkHub/src/device/brightness"
	"OpenICUELinkHub/src/device/common"
	"OpenICUELinkHub/src/structs"
)

// InterpolateColor performs linear interpolation between two colors
func interpolateColor(c1, c2 *structs.Color, t float64) *structs.Color {
	return &structs.Color{
		Red:   common.Lerp(c1.Red, c2.Red, t),
		Green: common.Lerp(c1.Green, c2.Green, t),
		Blue:  common.Lerp(c1.Blue, c2.Blue, t),
	}
}

// GenerateCircleColors will generate color based on start and end color
func GenerateCircleColors(numLEDs int, c1, c2 *structs.Color, factor, bts float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, numLEDs)
	for i := 0; i < numLEDs; i++ {
		color := interpolateColor(c1, c2, factor)
		color.Brightness = bts
		modify := brightness.ModifyBrightness(*color)
		colors[i] = struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
	}
	return colors
}
