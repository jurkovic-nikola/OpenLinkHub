package rgb

import (
	"math"
	"time"
)

// pastelRainbowColor function returns an RGB color corresponding to a given position in the rainbow
func pastelRainbowColor(position float64) (int, int, int) {
	// Normalize position to be between 0 and 1
	position = math.Mod(position, 1.0)

	colors := []struct {
		R uint8
		G uint8
		B uint8
	}{
		{R: 244, G: 253, B: 255},
		{R: 255, G: 218, B: 247},
		{R: 255, G: 140, B: 230},
		{R: 254, G: 96,  B: 219},
		{R: 255, G: 17,  B: 206},
		{R: 254, G: 28,  B: 206},
		{R: 255, G: 106, B: 225},
		{R: 255, G: 184, B: 241},
		{R: 255, G: 230, B: 248},
		{R: 255, G: 255, B: 203},
		{R: 255, G: 255, B: 155},
		{R: 255, G: 255, B: 80},
		{R: 255, G: 255, B: 33},
		{R: 216, G: 250, B: 52},
		{R: 161, G: 244, B: 119},
		{R: 105, G: 237, B: 185},
		{R: 14,  G: 208, B: 250},
		{R: 37,  G: 210, B: 255},
		{R: 114, G: 227, B: 255},
		{R: 159, G: 235, B: 253},
	}

	position = position * float64(len(colors))

	// Calculate the color based on the position
	currentColor := colors[int(math.Floor(position))]
	nextColorId := int(math.Floor(position))  + 1
	if nextColorId == len(colors) {
		nextColorId = 0
	}
	nextColor := colors[nextColorId]

	r1, g1, b1 := normalize(currentColor.R, currentColor.G, currentColor.B)
	r2, g2, b2 := normalize(nextColor.R, nextColor.G, nextColor.B)
	return interpolate(r1, g1, b1, r2, g2, b2, position - math.Floor(position))
}

// generatePastelRainbowColors will generate color based on start and end color
func generatePastelRainbowColors(lightChannels int, elapsedTime, bts float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lightChannels)
	for i := 0; i < lightChannels; i++ {
		position := (float64(i) / float64(lightChannels)) + (elapsedTime / 4.0)
		position = math.Mod(position, 1.0) // Keep position within the 0-1 range
		r, g, b := pastelRainbowColor(position)

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
func (r *ActiveRGB) PastelRainbow(startTime time.Time) {
	elapsed := time.Since(startTime).Seconds()
	speedFactor := 4.0
	if r.RgbModeSpeed > 0 {
		speedFactor = 4.0 / r.RgbModeSpeed
	}

	buf := map[int][]byte{}
	colors := generatePastelRainbowColors(r.LightChannels, elapsed*speedFactor, r.RGBBrightness)
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

// generatePastelSpiralRainbow generates rainbow colors with spiral offset
func generatePastelSpiralRainbow(lightChannels int, elapsedTime, brightness, spiralDensity float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lightChannels)

	for i := 0; i < lightChannels; i++ {
		// Spiral effect: add offset based on index and density
		position := (elapsedTime / 4.0) + (float64(i) * spiralDensity / float64(lightChannels))
		position = math.Mod(position, 1.0)

		r, g, b := pastelRainbowColor(position)

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
func (r *ActiveRGB) PastelSpiralRainbow(startTime time.Time) {
	elapsed := time.Since(startTime).Seconds()

	// Speed control
	speedFactor := 2.0
	if r.RgbModeSpeed > 0 {
		speedFactor = 2.0 / r.RgbModeSpeed
	}

	// Spiral density (higher = more twists)
	spiralDensity := 5.0 // tweakable (1 = smooth sweep, 3–5 = tight spiral)

	buf := map[int][]byte{}
	colors := generatePastelSpiralRainbow(r.LightChannels, elapsed*speedFactor, r.RGBBrightness, spiralDensity)

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
