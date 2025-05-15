package rgb

import (
	"time"
)

// Circle will run RGB function
func (r *ActiveRGB) Circle(startTime *time.Time) {
	elapsed := time.Since(*startTime).Milliseconds()

	// Calculate progress and reset when it exceeds 1.0
	progress := float64(elapsed) / (r.RgbModeSpeed * 1000)
	if progress >= 1.0 {
		*startTime = time.Now() // Reset startTime to the current time
		elapsed = 0             // Reset elapsed time
		progress = 0            // Reset progress
	}

	activeLEDs := int(progress * float64(r.LightChannels))
	if r.LightChannels == 1 {
		activeLEDs = 1
	}
	buf := map[int][]byte{}
	for j := 0; j < activeLEDs; j++ {
		t := float64(j) / float64(40) // Calculate interpolation factor
		colors := interpolateColors(r.RGBStartColor, r.RGBStartColor, t, r.RGBBrightness)

		if len(r.Buffer) > 0 {
			r.Buffer[j] = byte(colors.Red)
			r.Buffer[j+r.ColorOffset] = byte(colors.Green)
			r.Buffer[j+(r.ColorOffset*2)] = byte(colors.Blue)
		} else {
			buf[j] = []byte{
				byte(colors.Red),
				byte(colors.Green),
				byte(colors.Blue),
			}

			if r.IsAIO && r.HasLCD {
				if j > 15 && j < 20 {
					buf[j] = []byte{0, 0, 0}
				}
			}
		}
	}

	for j := activeLEDs; j < r.LightChannels; j++ {
		if len(r.Buffer) > 0 {
			r.Buffer[j] = 0
			r.Buffer[j+r.ColorOffset] = 0
			r.Buffer[j+(r.ColorOffset*2)] = 0
		} else {
			buf[j] = []byte{0, 0, 0}
		}
	}

	if r.Inverted {
		r.Output = SetColorInverted(buf)
	} else {
		r.Output = SetColor(buf)
	}
}

// CircleShift will run RGB function
func (r *ActiveRGB) CircleShift(startTime *time.Time) {
	elapsed := time.Since(*startTime).Milliseconds()

	// Calculate progress and reset when it exceeds 1.0
	progress := float64(elapsed) / (r.RgbModeSpeed * 1000)
	if progress >= 1.0 {
		*startTime = time.Now() // Reset startTime to the current time
		elapsed = 0             // Reset elapsed time
		progress = 0            // Reset progress
	}

	activeLEDs := int(progress * float64(r.LightChannels))
	if r.LightChannels == 1 {
		activeLEDs = 1
	}
	buf := map[int][]byte{}
	for j := 0; j < activeLEDs; j++ {
		t := float64(j) / float64(r.LightChannels) // Calculate interpolation factor
		colors := interpolateColors(r.RGBStartColor, r.RGBEndColor, t, r.RGBBrightness)
		if len(r.Buffer) > 0 {
			r.Buffer[j] = byte(colors.Red)
			r.Buffer[j+r.ColorOffset] = byte(colors.Green)
			r.Buffer[j+(r.ColorOffset*2)] = byte(colors.Blue)
		} else {
			buf[j] = []byte{
				byte(colors.Red),
				byte(colors.Green),
				byte(colors.Blue),
			}

			if r.IsAIO && r.HasLCD {
				if j > 15 && j < 20 {
					buf[j] = []byte{0, 0, 0}
				}
			}
		}
	}

	for j := activeLEDs; j < r.LightChannels; j++ {
		if len(r.Buffer) > 0 {
			r.Buffer[j] = 0
			r.Buffer[j+r.ColorOffset] = 0
			r.Buffer[j+(r.ColorOffset*2)] = 0
		} else {
			buf[j] = []byte{0, 0, 0}
		}
	}

	if r.Inverted {
		r.Output = SetColorInverted(buf)
	} else {
		r.Output = SetColor(buf)
	}
}
