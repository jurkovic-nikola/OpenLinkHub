package rgb

import (
	"math"
	"time"
)

// Marquee will run RGB function
func (r *ActiveRGB) Marquee(startTime *time.Time) {
	buf := map[int][]byte{}
	elapsed := time.Since(*startTime).Milliseconds()
	progress := math.Mod(float64(elapsed)/(r.RgbModeSpeed*1000), 1.0)

	if progress >= 1.0 {
		*startTime = time.Now()
		elapsed = 0
		progress = 0
	}

	colors := generateMarqueeColors(r.LightChannels, progress, r.RGBStartColor, r.RGBEndColor, r.RGBBrightness)

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

// generateMarqueeColors builds the chasing block pattern
func generateMarqueeColors(lightChannels int, progress float64, start, end *Color, brightness float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lightChannels)

	blockSize := 2
	gapSize := 2
	patternSize := blockSize + gapSize
	offset := int(progress * float64(patternSize))

	var shift float64
	if progress < 0.5 {
		shift = progress * 2
	} else {
		shift = (1 - progress) * 2
	}
	base := interpolateColors(start, end, shift, brightness)

	for i := 0; i < lightChannels; i++ {
		ledId := (i + offset) % patternSize
		if ledId < blockSize {
			colors[i] = struct{ R, G, B float64 }{base.Red, base.Green, base.Blue}
		} else {
			colors[i] = struct{ R, G, B float64 }{0, 0, 0}
		}
	}

	return colors
}
