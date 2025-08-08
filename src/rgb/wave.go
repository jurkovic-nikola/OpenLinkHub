package rgb

import (
	"math"
	"time"
)

// Wave will run RGB function
func (r *ActiveRGB) Wave(startTime *time.Time) {
	buf := map[int][]byte{}
	elapsed := time.Since(*startTime).Milliseconds()
	progress := float64(elapsed) / (r.RgbModeSpeed * 100)

	for i := 0; i < r.LightChannels; i++ {
		wavePos := (progress + float64(i)) / r.RgbModeSpeed
		intensity := 0.5 * (1 + math.Sin(2*math.Pi*wavePos))
		modify := interpolateColor(r.RGBStartColor, r.RGBEndColor, intensity, r.RGBBrightness)

		red := modify.Red * intensity
		green := modify.Green * intensity
		blue := modify.Blue * intensity

		if len(r.Buffer) > 0 {
			r.Buffer[i] = byte(red)
			r.Buffer[i+r.ColorOffset] = byte(green)
			r.Buffer[i+(r.ColorOffset*2)] = byte(blue)
		} else {
			buf[i] = []byte{byte(red), byte(green), byte(blue)}
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
