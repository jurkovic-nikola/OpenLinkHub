package rgb

import (
	"math/rand"
	"time"
)

// Flickering will run RGB function
func (r *ActiveRGB) Flickering(startTime *time.Time) {
	elapsed := time.Since(*startTime).Milliseconds()

	// Calculate progress and reset when it exceeds 1.0
	progress := float64(elapsed) / (r.RgbModeSpeed * 1000)
	if progress >= 1.0 {
		*startTime = time.Now() // Reset startTime to the current time
		elapsed = 0             // Reset elapsed time
		progress = 0            // Reset progress
	}

	buf := map[int][]byte{}
	for j := 0; j < r.LightChannels; j++ {
		t := float64(j) / float64(r.LightChannels) // Calculate interpolation factor
		colors := interpolateColors(r.RGBStartColor, r.RGBEndColor, t, r.RGBBrightness)
		if rand.Intn(r.LightChannels*int(r.RgbModeSpeed)) == 1 {
			buf[j] = []byte{0, 0, 0}
		} else {
			buf[j] = []byte{
				byte(colors.Red),
				byte(colors.Green),
				byte(colors.Blue),
			}
		}

		if r.IsAIO && r.HasLCD {
			if j > 15 && j < 20 {
				buf[j] = []byte{0, 0, 0}
			}
		}
	}

	if r.Inverted {
		r.Output = SetColorInverted(buf)
	} else {
		r.Output = SetColor(buf)
	}

}
