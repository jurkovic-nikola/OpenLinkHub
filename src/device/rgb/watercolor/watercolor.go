package watercolor

import (
	"OpenICUELinkHub/src/device/brightness"
	"OpenICUELinkHub/src/structs"
	"math"
)

// WatercolorColor function returns an RGB color corresponding to a given position in the watercolor spectrum
func WatercolorColor(position float64) (int, int, int) {
	// Normalize position to be between 0 and 1
	position = math.Mod(position, 1.0)

	// Adjust hue, saturation, and brightness to create pastel colors
	hue := position * 360 // Convert position to hue angle (0-360 degrees)
	saturation := 0.4     // Lower saturation for watercolor effect
	brightness := 1.0     // Full brightness for watercolor effect

	return HSBToRGB(hue, saturation, brightness)
}

// HSBToRGB function converts HSB/HSV color space to RGB color space
func HSBToRGB(h, s, v float64) (int, int, int) {
	h = math.Mod(h, 360)
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := v - c

	var r, g, b float64
	switch {
	case 0 <= h && h < 60:
		r, g, b = c, x, 0
	case 60 <= h && h < 120:
		r, g, b = x, c, 0
	case 120 <= h && h < 180:
		r, g, b = 0, c, x
	case 180 <= h && h < 240:
		r, g, b = 0, x, c
	case 240 <= h && h < 300:
		r, g, b = x, 0, c
	case 300 <= h && h < 360:
		r, g, b = c, 0, x
	}

	r = (r + m) * 255
	g = (g + m) * 255
	b = (b + m) * 255

	return int(r), int(g), int(b)
}

// GenerateWatercolorColors generates a list of RGB colors for the given number of LEDs at the current time
func GenerateWatercolorColors(numLEDs int, elapsedTime, brightnessValue float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, numLEDs)
	for i := 0; i < numLEDs; i++ {
		position := (float64(i) / float64(numLEDs)) + (elapsedTime / 4.0)
		position = math.Mod(position, 1.0) // Keep position within the 0-1 range
		r, g, b := WatercolorColor(position)

		color := &structs.Color{
			Red:        float64(r),
			Green:      float64(g),
			Blue:       float64(b),
			Brightness: brightnessValue,
		}
		modify := brightness.ModifyBrightness(*color, color.Brightness)
		colors[i] = struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
	}
	return colors
}
