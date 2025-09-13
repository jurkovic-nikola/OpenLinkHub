package rgb

import (
	"math"
	"time"
)

// rainbowColor function returns an RGB color corresponding to a given position in the rainbow
func rainbowColor(position float64) (int, int, int) {
	// Normalize position to be between 0 and 1
	position = math.Mod(position, 1.0)

	// Calculate the color based on the position
	if position < 0.2 {
		// Red to Yellow
		return interpolate(1, 0, 0, 1, 1, 0, position/0.2)
	} else if position < 0.4 {
		// Yellow to Green
		return interpolate(1, 1, 0, 0, 1, 0, (position-0.2)/0.2)
	} else if position < 0.6 {
		// Green to Cyan
		return interpolate(0, 1, 0, 0, 1, 1, (position-0.4)/0.2)
	} else if position < 0.8 {
		// Cyan to Blue
		return interpolate(0, 1, 1, 0, 0, 1, (position-0.6)/0.2)
	} else {
		// Blue to Red
		return interpolate(0, 0, 1, 1, 0, 0, (position-0.8)/0.2)
	}
}

// generateColors will generate color based on start and end color
func generateRainbowColors(lightChannels int, elapsedTime, bts float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lightChannels)
	for i := 0; i < lightChannels; i++ {
		position := (float64(i) / float64(lightChannels)) + (elapsedTime / 4.0)
		position = math.Mod(position, 1.0) // Keep position within the 0-1 range
		r, g, b := rainbowColor(position)

		color := &Color{
			Red:        float64(r),
			Green:      float64(g),
			Blue:       float64(b),
			Brightness: bts,
		}

		modify := ModifyBrightness(*color)
		colors[i] = struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
	}
	return colors
}

// Rainbow will run RGB function
func (r *ActiveRGB) Rainbow(startTime time.Time) {
	elapsed := time.Since(startTime).Seconds()
	speedFactor := 4.0
	if r.RgbModeSpeed > 0 {
		speedFactor = 4.0 / r.RgbModeSpeed
	}

	buf := map[int][]byte{}
	colors := generateRainbowColors(r.LightChannels, elapsed*speedFactor, r.RGBBrightness)
	for i, color := range colors {
		if len(r.Buffer) > 0 {
			r.Buffer[i] = byte(color.R)
			r.Buffer[i+r.ColorOffset] = byte(color.G)
			r.Buffer[i+(r.ColorOffset*2)] = byte(color.B)
		} else {
			buf[i] = []byte{
				byte(color.R),
				byte(color.G),
				byte(color.B),
			}
			if r.IsAIO && r.HasLCD {
				if i > 15 && i < 20 {
					buf[i] = []byte{0, 0, 0}
				}
			}
		}
	}
	// Raw colors
	r.Raw = buf

	if r.Inverted {
		r.Output = SetColorInverted(buf)
	} else {
		r.Output = SetColor(buf)
	}
}

// generateSpiralRainbow generates rainbow colors with spiral offset
func generateSpiralRainbow(lightChannels int, elapsedTime, brightness, spiralDensity float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lightChannels)

	for i := 0; i < lightChannels; i++ {
		// Spiral effect: add offset based on index and density
		position := (elapsedTime / 4.0) + (float64(i) * spiralDensity / float64(lightChannels))
		position = math.Mod(position, 1.0)

		r, g, b := rainbowColor(position)

		color := &Color{
			Red:        float64(r),
			Green:      float64(g),
			Blue:       float64(b),
			Brightness: brightness,
		}

		modify := ModifyBrightness(*color)
		colors[i] = struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
	}

	return colors
}

// SpiralRainbow runs RGB function with spiral effect
func (r *ActiveRGB) SpiralRainbow(startTime time.Time) {
	elapsed := time.Since(startTime).Seconds()

	// Speed control
	speedFactor := 4.0
	if r.RgbModeSpeed > 0 {
		speedFactor = 4.0 / r.RgbModeSpeed
	}

	// Spiral density (higher = more twists)
	spiralDensity := 3.0 // tweakable (1 = smooth sweep, 3–5 = tight spiral)

	buf := map[int][]byte{}
	colors := generateSpiralRainbow(r.LightChannels, elapsed*speedFactor, r.RGBBrightness, spiralDensity)

	for i, color := range colors {
		if len(r.Buffer) > 0 {
			r.Buffer[i] = byte(color.R)
			r.Buffer[i+r.ColorOffset] = byte(color.G)
			r.Buffer[i+(r.ColorOffset*2)] = byte(color.B)
		} else {
			buf[i] = []byte{
				byte(color.R),
				byte(color.G),
				byte(color.B),
			}
			if r.IsAIO && r.HasLCD {
				if i > 15 && i < 20 {
					buf[i] = []byte{0, 0, 0}
				}
			}
		}
	}

	r.Raw = buf

	if r.Inverted {
		r.Output = SetColorInverted(buf)
	} else {
		r.Output = SetColor(buf)
	}
}
