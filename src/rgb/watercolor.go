package rgb

import (
	"math"
	"time"
)

// watercolorColor function returns an RGB color corresponding to a given position in the watercolor spectrum
func watercolorColor(position float64) (int, int, int) {
	// Normalize position to be between 0 and 1
	position = math.Mod(position, 1.0)

	// Adjust hue, saturation, and brightness to create pastel colors
	hue := position * 360 // Convert position to hue angle (0-360 degrees)
	saturation := 0.4     // Lower saturation for watercolor effect
	bts := 1.0            // Full brightness for watercolor effect

	return HSBToRGB(hue, saturation, bts)
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

// generateWaterColors will generate color based on start and end color
func generateWaterColors(lightChannels int, elapsedTime, brightnessValue float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lightChannels)
	for i := 0; i < lightChannels; i++ {
		position := (float64(i) / float64(lightChannels)) + (elapsedTime / 4.0)
		position = math.Mod(position, 1.0) // Keep position within the 0-1 range
		r, g, b := watercolorColor(position)

		color := &Color{
			Red:        float64(r),
			Green:      float64(g),
			Blue:       float64(b),
			Brightness: brightnessValue,
		}
		modify := ModifyBrightness(*color)
		colors[i] = struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
	}
	return colors
}

// Watercolor will run RGB function
func (r *ActiveRGB) Watercolor(startTime time.Time) {
	elapsed := time.Since(startTime).Seconds() * r.RgbModeSpeed
	buf := map[int][]byte{}
	colors := generateWaterColors(r.LightChannels, elapsed, r.RGBBrightness)
	for i, color := range colors {
		buf[i] = []byte{
			byte(color.R),
			byte(color.G),
			byte(color.B),
		}
	}
	r.Output = SetColor(buf)
}
