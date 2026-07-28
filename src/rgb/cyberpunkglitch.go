package rgb

import (
	"math"
	"time"
)

// CyberpunkGlitch simulates a cybernetic glitch effect: dark breathing purple base, neon cyan/pink ripples, and static frame noise
func (r *ActiveRGB) CyberpunkGlitch(startTime *time.Time) {
	elapsed := time.Since(*startTime).Milliseconds()
	tSeconds := float64(elapsed) / 1000.0

	// Respect configured speed slider/value by multiplying time factor
	tScaled := tSeconds * r.RgbModeSpeed

	// We calculate active glitches deterministically based on time slots
	// Each slot is 0.4 seconds long
	slotDuration := 0.40
	currentSlot := math.Floor(tScaled / slotDuration)

	buf := map[int][]byte{}
	for j := 0; j < r.LightChannels; j++ {
		// Calculate position factor across channels
		pos := float64(j) / float64(r.LightChannels)

		// Base slow-breathing dark purple/indigo background
		breath := 0.4 + 0.3*math.Sin(tScaled*1.2+pos*4.0)
		red := 16.0 * breath
		green := 1.0 * breath
		blue := 38.0 * breath

		// Check active ripples from the current and previous slots to allow overlaps
		for slotOffset := -1; slotOffset <= 0; slotOffset++ {
			slot := currentSlot + float64(slotOffset)

			// Deterministic pseudo-random generation based on slot index
			if random01(slot, 11.0) > 0.4 { // 60% chance of a ripple in this slot
				// Ripple center position
				center := random01(slot, 12.0)
				// Ripple color (cyan vs pink)
				isPink := random01(slot, 13.0) > 0.5
				// Ripple start offset within the slot
				startOffset := random01(slot, 14.0) * 0.1
				// Ripple duration
				duration := 0.18 + random01(slot, 15.0)*0.18 // 0.18s to 0.36s
				// Speed of propagation
				speed := 1.8 + random01(slot, 16.0)*2.0 // 1.8 to 3.8 units per second

				// Calculate age of the ripple in real time
				rippleAge := (tScaled - (slot*slotDuration + startOffset))

				if rippleAge >= 0 && rippleAge < duration {
					progress := rippleAge / duration
					radius := rippleAge * speed
					dist := math.Abs(pos - center)

					// If the LED is within the expanding ripple wave front (thickness of 0.15)
					thickness := 0.15
					if dist < radius && dist > (radius-thickness) {
						// Calculate intensity with distance fading
						waveIntensity := (1.0 - progress) * (1.0 - (radius-dist)/thickness)
						waveIntensity = clampFloat01(waveIntensity)

						var rGlitch, gGlitch, bGlitch float64
						if isPink {
							rGlitch = 255
							gGlitch = 0
							bGlitch = 128
						} else {
							rGlitch = 0
							gGlitch = 255
							bGlitch = 255
						}

						// Blend wave front with existing color
						red = lerp(red, rGlitch, waveIntensity)
						green = lerp(green, gGlitch, waveIntensity)
						blue = lerp(blue, bGlitch, waveIntensity)
					}
				}
			}
		}

		// Single frame static noise glitch (1.2% chance per LED per frame)
		// We use both position and absolute time to get a unique seed per LED per frame
		frameNoise := random01(pos, tScaled)
		if frameNoise > 0.988 {
			if random01(pos, tScaled, 1.0) > 0.5 {
				red = 255
				green = 0
				blue = 128
			} else {
				red = 0
				green = 255
				blue = 255
			}
		}

		color := Color{
			Red:        red,
			Green:      green,
			Blue:       blue,
			Brightness: r.RGBBrightness,
		}
		modify := ModifyBrightness(color)

		if len(r.Buffer) > 0 {
			r.Buffer[j] = byte(modify.Red)
			r.Buffer[j+r.ColorOffset] = byte(modify.Green)
			r.Buffer[j+(r.ColorOffset*2)] = byte(modify.Blue)
		} else {
			buf[j] = []byte{
				byte(modify.Red),
				byte(modify.Green),
				byte(modify.Blue),
			}
			if r.IsAIO && r.HasLCD {
				if j > 15 && j < 20 {
					buf[j] = []byte{0, 0, 0}
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
