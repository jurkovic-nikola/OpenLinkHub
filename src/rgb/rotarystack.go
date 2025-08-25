package rgb

import (
	"math"
	"time"
)

// RotaryStack will run RGB function
func (r *ActiveRGB) RotaryStack(startTime *time.Time) {
	elapsed := time.Since(*startTime).Milliseconds()
	progress := math.Mod(float64(elapsed)/(r.RgbModeSpeed*1000), 1.0)

	if progress >= 1.0 {
		*startTime = time.Now() // Reset startTime
		elapsed = 0
		progress = 0
	}

	ledCount := r.LightChannels

	// Total number of completed rotor cycles (integer part of elapsed/cycle)
	totalCycles := int(float64(elapsed) / float64(r.RgbModeSpeed*1000))
	trailCount := totalCycles
	if trailCount > ledCount {
		trailCount = ledCount
	}

	// Current rotor progress (fractional part)
	rotorPos := int(progress * float64(ledCount-trailCount))
	if rotorPos >= ledCount-trailCount {
		rotorPos = ledCount - trailCount - 1
	}

	colors := make([]struct{ R, G, B float64 }, ledCount)
	rotorRGB := interpolateColors(r.RGBStartColor, r.RGBStartColor, 0, r.RGBBrightness)
	trailRGB := interpolateColors(r.RGBEndColor, r.RGBEndColor, 0, r.RGBBrightness)

	for i := 0; i < ledCount; i++ {
		switch {
		case i >= ledCount-trailCount:
			// Already marked trail LEDs
			colors[i] = struct{ R, G, B float64 }{trailRGB.Red, trailRGB.Green, trailRGB.Blue}
		case i <= rotorPos:
			// Current rotor LEDs (behind rotor)
			colors[i] = struct{ R, G, B float64 }{rotorRGB.Red, rotorRGB.Green, rotorRGB.Blue}
		default:
			// Not yet reached
			colors[i] = struct{ R, G, B float64 }{0, 0, 0}
		}
	}

	// Update Buffer
	for j, c := range colors {
		if len(r.Buffer) > 0 {
			r.Buffer[j] = byte(c.R)
			r.Buffer[j+r.ColorOffset] = byte(c.G)
			r.Buffer[j+(r.ColorOffset*2)] = byte(c.B)
		}
	}

	// Update Raw for controller
	raw := map[int][]byte{}
	for j, c := range colors {
		raw[j] = []byte{byte(c.R), byte(c.G), byte(c.B)}
	}
	r.Raw = raw

	// Output
	if r.Inverted {
		r.Output = SetColorInverted(raw)
	} else {
		r.Output = SetColor(raw)
	}

	// Reset startTime when all LEDs are marked
	if trailCount >= ledCount {
		*startTime = time.Now()
	}
}
