package colorpulse

import (
	"OpenICUELinkHub/src/device/brightness"
	"OpenICUELinkHub/src/structs"
)

// Lerp performs linear interpolation between two values
func Lerp(a, b, t float64) float64 {
	return a + t*(b-a)
}

// InterpolateColor performs linear interpolation between two colors
func InterpolateColor(c1, c2 *structs.Color, t float64) *structs.Color {
	return &structs.Color{
		Red:   Lerp(c1.Red, c2.Red, t),
		Green: Lerp(c1.Green, c2.Green, t),
		Blue:  Lerp(c1.Blue, c2.Blue, t),
	}
}

func GenerateColorPulseColors(numLEDs int, c1, c2 *structs.Color, factor, bth float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, numLEDs)
	for i := 0; i < numLEDs; i++ {
		color := InterpolateColor(c1, c2, factor)
		color.Brightness = bth
		modify := brightness.ModifyBrightness(*color)
		colors[i] = struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
	}
	return colors
}
