package rgb

import (
	"math"
	"time"
)

// Visor runs a left-to-right (and back) sweep at constant LED speed
func (r *ActiveRGB) Visor(startTime *time.Time) {
	buf := map[int][]byte{}
	elapsed := time.Since(*startTime).Milliseconds()

	ledsPerSecond := 10.0 / r.RgbModeSpeed // tweak base speed here

	// Total distance for one sweep (forth+back)
	cycleLength := float64((r.LightChannels - 1) * 2)

	// Beam position in "LED units" (0..cycleLength)
	pos := math.Mod(float64(elapsed)/1000.0*ledsPerSecond, cycleLength)

	// Mirror position when sweeping back
	var barCenter float64
	if pos <= float64(r.LightChannels-1) {
		barCenter = pos
	} else {
		barCenter = cycleLength - pos
	}

	colors := generateVisorColors(
		r.LightChannels,
		barCenter,
		r.RGBStartColor,
		r.RGBEndColor,
		r.RGBBrightness,
	)

	for j, c := range colors {
		if len(r.Buffer) > 0 {
			r.Buffer[j] = byte(c.R)
			r.Buffer[j+r.ColorOffset] = byte(c.G)
			r.Buffer[j+(r.ColorOffset*2)] = byte(c.B)
		} else {
			buf[j] = []byte{byte(c.R), byte(c.G), byte(c.B)}
			if r.IsAIO && r.HasLCD {
				if j > 15 && j < 20 {
					buf[j] = []byte{0, 0, 0}
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

// generateVisorColors creates the moving "visor" bar
func generateVisorColors(lightChannels int, barCenter float64, start, end *Color, brightness float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lightChannels)

	// Base color
	base := interpolateColors(start, end, 0.0, brightness)

	for i := 0; i < lightChannels; i++ {
		dist := math.Abs(float64(i) - barCenter)

		// Gaussian falloff (beam width ~3 LEDs)
		intensity := math.Exp(-0.5 * math.Pow(dist/3.0, 2))

		r := base.Red * intensity
		g := base.Green * intensity
		b := base.Blue * intensity

		colors[i] = struct{ R, G, B float64 }{r, g, b}
	}

	return colors
}
