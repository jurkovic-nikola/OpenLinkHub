package rgb

import (
	"math"
	"time"
)

// Nebula will run RGB function
func (r *ActiveRGB) Nebula(startTime *time.Time) {
	buf := map[int][]byte{}
	elapsed := time.Since(*startTime).Milliseconds()
	progress := math.Mod(float64(elapsed)/(r.RgbModeSpeed*5000), 1.0)

	colors := generateInterstellarColors(r.LightChannels, progress, r.RGBBrightness, float64(elapsed))

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
func generateInterstellarColors(lightChannels int, progress, brightness, elapsed float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lightChannels)

	// Nebula effect scrolls slowly with progress
	for i := 0; i < lightChannels; i++ {
		pos := float64(i)/float64(lightChannels) + progress
		t := (math.Sin(pos*math.Pi*2) + 1) * 0.5

		h, s, v := interstellarPalette(t)
		r, g, b := hsvToRGB(h, s, v*brightness*0.6)

		colors[i] = struct{ R, G, B float64 }{float64(r), float64(g), float64(b)}
	}

	// Time-based twinkles fade smoothly without storing per-LED state.
	const (
		sparkWindow   = 2000.0 // Milliseconds between spark opportunities
		sparkDuration = 600.0  // Milliseconds per twinkle
		sparkChance   = 0.12   // Probability per LED per window
	)

	window := int64(math.Floor(elapsed / sparkWindow))
	for i := 0; i < lightChannels; i++ {
		// Check the current and previous windows so twinkles can cross window boundaries without being cut short.
		for w := window - 1; w <= window; w++ {
			if w < 0 {
				continue
			}

			seed := uint64(w)*0x9e3779b97f4a7c15 + uint64(i+1)
			if nebulaRandom(seed) >= sparkChance {
				continue
			}

			start := float64(w)*sparkWindow + nebulaRandom(seed+1)*sparkWindow
			age := elapsed - start
			if age < 0 || age >= sparkDuration {
				continue
			}

			// Quick fade-in followed by a longer, smooth decay.
			const vectorIn = 120.0
			var intensity float64
			if age < vectorIn {
				intensity = age / vectorIn
			} else {
				decay := (age - vectorIn) / (sparkDuration - vectorIn)
				intensity = (math.Exp(-4*decay) - math.Exp(-4)) / (1 - math.Exp(-4))
			}

			// Keep sparks within the nebula's cyan-to-magenta palette.
			hue := 180.0 + nebulaRandom(seed+2)*120.0
			r, g, b := hsvToRGB(hue, 0.8, brightness)
			colors[i].R = colors[i].R*(1-intensity) + float64(r)*intensity
			colors[i].G = colors[i].G*(1-intensity) + float64(g)*intensity
			colors[i].B = colors[i].B*(1-intensity) + float64(b)*intensity
		}
	}

	return colors
}

// nebulaRandom returns a reproducible value in [0, 1) for a given seed.
func nebulaRandom(seed uint64) float64 {
	seed ^= seed >> 30
	seed *= 0xbf58476d1ce4e5b9 // SplitMix64 mixing algorithm
	seed ^= seed >> 27
	seed *= 0x94d049bb133111eb // SplitMix64 mixing algorithm
	seed ^= seed >> 31

	return float64(seed>>11) / (1 << 53)
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
