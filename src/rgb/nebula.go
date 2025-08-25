package rgb

import (
	"math"
	"math/rand"
	"time"
)

// Nebula will run RGB function
func (r *ActiveRGB) Nebula(startTime *time.Time) {
	buf := map[int][]byte{}
	elapsed := time.Since(*startTime).Milliseconds()
	progress := math.Mod(float64(elapsed)/(r.RgbModeSpeed*5000), 1.0)
	// ↑ slowed down 5x compared to Colorpulse

	if progress >= 1.0 {
		*startTime = time.Now()
		elapsed = 0
		progress = 0
	}

	colors := generateInterstellarColors(r.LightChannels, progress, r.RGBBrightness)

	for j, c := range colors {
		if len(r.Buffer) > 0 {
			r.Buffer[j] = byte(c.R)
			r.Buffer[j+r.ColorOffset] = byte(c.G)
			r.Buffer[j+(r.ColorOffset*2)] = byte(c.B)
		} else {
			buf[j] = []byte{
				byte(c.R),
				byte(c.G),
				byte(c.B),
			}
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

// generateInterstellarColors maps LEDs with nebula + twinkle effect
func generateInterstellarColors(lightChannels int, progress, brightness float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lightChannels)

	// Nebula effect scrolls slowly with progress
	for i := 0; i < lightChannels; i++ {
		pos := float64(i)/float64(lightChannels) + progress
		t := (math.Sin(pos*math.Pi*2) + 1) * 0.5

		h, s, v := interstellarPalette(t)
		r, g, b := hsvToRGB(h, s, v*brightness*0.6)

		colors[i] = struct{ R, G, B float64 }{float64(r), float64(g), float64(b)}
	}

	// Occasional twinkle overlay
	for i := 0; i < lightChannels; i++ {
		if rand.Float64() < 0.001 { // very rare spark
			r, g, b := hsvToRGB(rand.Float64()*360, 0.8, 1.0*brightness)
			colors[i] = struct{ R, G, B float64 }{float64(r), float64(g), float64(b)}
		}
	}

	return colors
}

// interstellarPalette creates a flowing space-like gradient
func interstellarPalette(t float64) (h, s, v float64) {
	switch {
	case t < 0.45:
		return lerp(220, 300, t/0.45), 0.9, 0.4 + 0.5*t
	case t < 0.75:
		return lerp(300, 180, (t-0.45)/0.30), 0.9, 0.6
	default:
		return lerp(180, 220, (t-0.75)/0.25), 0.9, 0.5
	}
}

// Helper functions
func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func hsvToRGB(h, s, v float64) (r, g, b int) {
	h = math.Mod(h, 360)
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := v - c

	var rr, gg, bb float64
	switch {
	case 0 <= h && h < 60:
		rr, gg, bb = c, x, 0
	case 60 <= h && h < 120:
		rr, gg, bb = x, c, 0
	case 120 <= h && h < 180:
		rr, gg, bb = 0, c, x
	case 180 <= h && h < 240:
		rr, gg, bb = 0, x, c
	case 240 <= h && h < 300:
		rr, gg, bb = x, 0, c
	case 300 <= h && h < 360:
		rr, gg, bb = c, 0, x
	}

	rr = (rr + m) * 255
	gg = (gg + m) * 255
	bb = (bb + m) * 255

	return int(rr), int(gg), int(bb)
}
