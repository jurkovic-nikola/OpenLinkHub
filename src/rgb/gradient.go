package rgb

import (
	"math"
	"time"
)

// ColorshiftGradient runs a smooth gradient effect across multiple colors
func (r *ActiveRGB) ColorshiftGradient(startTime time.Time, gradients map[int]Color, durationSeconds float64) {
	buf := map[int][]byte{}

	elapsed := time.Since(startTime).Seconds() // in seconds
	if durationSeconds <= 0 {
		durationSeconds = 5
	}

	progress := math.Mod(elapsed/durationSeconds, 1.0)
	numColors := len(gradients)
	if numColors < 2 {
		return // Need at least 2 colors
	}

	totalSegments := float64(numColors - 1)
	segment := progress * totalSegments
	segmentIndex := int(segment)
	segmentProgress := segment - float64(segmentIndex)

	colorA := gradients[segmentIndex]
	colorB := gradients[(segmentIndex+1)%numColors]
	color := interpolateColors(&colorA, &colorB, segmentProgress, r.RGBBrightness)

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

	r.Raw = buf
	if r.Inverted {
		r.Output = SetColorInverted(buf)
	} else {
		r.Output = SetColor(buf)
	}
}
