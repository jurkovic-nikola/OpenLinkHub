package rgb

import (
	"math"
	"time"
)

// pastelColors is static object containing all pastel colors. Expand if needed
var pastelColors = [20]Color{
	{Red: 244, Green: 253, Blue: 255},
	{Red: 255, Green: 218, Blue: 247},
	{Red: 255, Green: 140, Blue: 230},
	{Red: 254, Green: 96, Blue: 219},
	{Red: 255, Green: 17, Blue: 206},
	{Red: 254, Green: 28, Blue: 206},
	{Red: 255, Green: 106, Blue: 225},
	{Red: 255, Green: 184, Blue: 241},
	{Red: 255, Green: 230, Blue: 248},
	{Red: 255, Green: 255, Blue: 203},
	{Red: 255, Green: 255, Blue: 155},
	{Red: 255, Green: 255, Blue: 80},
	{Red: 255, Green: 255, Blue: 33},
	{Red: 216, Green: 250, Blue: 52},
	{Red: 161, Green: 244, Blue: 119},
	{Red: 105, Green: 237, Blue: 185},
	{Red: 14, Green: 208, Blue: 250},
	{Red: 37, Green: 210, Blue: 255},
	{Red: 114, Green: 227, Blue: 255},
	{Red: 159, Green: 235, Blue: 253},
}

// pastelRainbowColor function returns an RGB color corresponding to a given position in the rainbow spectrum
func pastelRainbowColor(position float64) (int, int, int) {
	position -= math.Floor(position)

	pos := position * float64(len(pastelColors))
	idx := int(pos)
	frac := pos - float64(idx)

	nextIdx := idx + 1
	if nextIdx == len(pastelColors) {
		nextIdx = 0
	}

	c1 := pastelColors[idx]
	c2 := pastelColors[nextIdx]

	r := c1.Red + (c2.Red-c1.Red)*frac
	g := c1.Green + (c2.Green-c1.Green)*frac
	b := c1.Blue + (c2.Blue-c1.Blue)*frac

	return int(r), int(g), int(b)
}

// generatePastelRainbowColors will generate color based on pastelRainbowColor
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

// PastelRainbow will run RGB function
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

// generatePastelSpiralRainbow generates pastel rainbow colors with spiral offset
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

// PastelSpiralRainbow runs RGB function with spiral effect
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
