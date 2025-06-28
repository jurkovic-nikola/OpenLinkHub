package rgb

import (
	"math"
	"time"
)

// Circle will run RGB function
func (r *ActiveRGB) Circle(startTime *time.Time) {
	elapsed := time.Since(*startTime).Milliseconds()

	progress := float64(elapsed) / (r.RgbModeSpeed * 1000)
	if progress >= 1.0 {
		*startTime = time.Now()
		elapsed = 0
		progress = 0
	}

	progress = math.Mod(progress, 1.0)

	buf := map[int][]byte{}
	fadeFactor := 6.0 // Adjust for sharpness of the glowing point

	for j := 0; j < r.LightChannels; j++ {
		pos := float64(j) / float64(r.LightChannels)
		distance := math.Abs(progress - pos)

		// Wrap around for circular motion
		if distance > 0.5 {
			distance = 1.0 - distance
		}

		// Cosine falloff for natural fade
		fade := math.Cos(distance * math.Pi * fadeFactor)
		if fade < 0 {
			fade = 0
		}

		// Single-color pulse, you can optionally use interpolateColors to create gradients if needed
		colors := interpolateColors(r.RGBStartColor, r.RGBStartColor, 0, r.RGBBrightness)

		red := byte(colors.Red * fade)
		green := byte(colors.Green * fade)
		blue := byte(colors.Blue * fade)

		if len(r.Buffer) > 0 {
			r.Buffer[j] = red
			r.Buffer[j+r.ColorOffset] = green
			r.Buffer[j+(r.ColorOffset*2)] = blue
		} else {
			buf[j] = []byte{red, green, blue}

			if r.IsAIO && r.HasLCD {
				if j > 15 && j < 20 {
					buf[j] = []byte{0, 0, 0}
				}
			}
		}
	}

	if len(r.Buffer) == 0 {
		for j := len(buf); j < r.LightChannels; j++ {
			buf[j] = []byte{0, 0, 0}
		}
	}

	r.Raw = buf

	if r.Inverted {
		r.Output = SetColorInverted(buf)
	} else {
		r.Output = SetColor(buf)
	}
}

// CircleShift will run RGB function
func (r *ActiveRGB) CircleShift(startTime *time.Time) {
	elapsed := time.Since(*startTime).Milliseconds()

	progress := float64(elapsed) / (r.RgbModeSpeed * 1000)
	if progress >= 1.0 {
		*startTime = time.Now()
		elapsed = 0
		progress = 0
	}

	// Normalize progress to loop around
	progress = math.Mod(progress, 1.0)

	buf := map[int][]byte{}
	fadeFactor := 6.0 // Controls how soft or sharp the gradient is

	for j := 0; j < r.LightChannels; j++ {
		pos := float64(j) / float64(r.LightChannels)
		distance := math.Abs(progress - pos)

		// Wrap distance for circular effect
		if distance > 0.5 {
			distance = 1.0 - distance
		}

		// Use cosine-based fade for smoother transition
		fade := math.Cos(distance * math.Pi * fadeFactor)
		if fade < 0 {
			fade = 0
		}

		colors := interpolateColors(r.RGBStartColor, r.RGBEndColor, pos, r.RGBBrightness)

		red := byte(colors.Red * fade)
		green := byte(colors.Green * fade)
		blue := byte(colors.Blue * fade)

		if len(r.Buffer) > 0 {
			r.Buffer[j] = red
			r.Buffer[j+r.ColorOffset] = green
			r.Buffer[j+(r.ColorOffset*2)] = blue
		} else {
			buf[j] = []byte{red, green, blue}

			if r.IsAIO && r.HasLCD {
				if j > 15 && j < 20 {
					buf[j] = []byte{0, 0, 0}
				}
			}
		}
	}

	// Fill unused buffer if needed
	if len(r.Buffer) == 0 {
		for j := len(buf); j < r.LightChannels; j++ {
			buf[j] = []byte{0, 0, 0}
		}
	}

	r.Raw = buf

	if r.Inverted {
		r.Output = SetColorInverted(buf)
	} else {
		r.Output = SetColor(buf)
	}
}
