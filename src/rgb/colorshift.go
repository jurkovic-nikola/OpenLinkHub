package rgb

import (
	"math"
	"time"
)

// Colorshift will run RGB function
func (r *ActiveRGB) Colorshift(startTime *time.Time, activeRgb *ActiveRGB) {
	buf := map[int][]byte{}

	elapsed := time.Since(*startTime).Milliseconds()
	if r.RgbModeSpeed == 0 {
		r.RgbModeSpeed = 1.0
	}

	totalProgress := float64(elapsed) / (r.RgbModeSpeed * 1000)
	progress := math.Mod(totalProgress, 1.0)

	cycleCount := int(totalProgress)
	if cycleCount%2 == 0 {
		activeRgb.Phase = 0
	} else {
		activeRgb.Phase = 1
	}

	var color *Color
	if activeRgb.Phase == 0 {
		color = interpolateColor(r.RGBStartColor, r.RGBEndColor, progress, r.RGBBrightness)
	} else {
		color = interpolateColor(r.RGBEndColor, r.RGBStartColor, progress, r.RGBBrightness)
	}

	// Update LED channels
	for j := 0; j < r.LightChannels; j++ {
		if len(r.Buffer) > 0 {
			r.Buffer[j] = byte(color.Red)
			r.Buffer[j+r.ColorOffset] = byte(color.Green)
			r.Buffer[j+(r.ColorOffset*2)] = byte(color.Blue)
		} else {
			buf[j] = []byte{
				byte(color.Red),
				byte(color.Green),
				byte(color.Blue),
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
