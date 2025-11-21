package rgb

import (
	"math"
	"sort"
	"time"
)

// ColorshiftGradient runs a smooth gradient effect across multiple colors
func (r *ActiveRGB) ColorshiftGradient(startTime time.Time, gradients map[int]Color, durationSeconds float64) {
	if durationSeconds <= 0 {
		durationSeconds = 5
	}

	elapsed := time.Since(startTime).Seconds()
	globalProgress := math.Mod(elapsed/durationSeconds, 1.0) // normalized 0..1

	numColors := len(gradients)
	if numColors < 2 {
		return // Need at least 2 colors
	}

	// Convert map to sorted slice based on Position
	gradSlice := make([]Color, 0, numColors)
	for i := 0; i < numColors; i++ {
		gradSlice = append(gradSlice, gradients[i])
	}
	sort.Slice(gradSlice, func(i, j int) bool {
		return gradSlice[i].Position < gradSlice[j].Position
	})

	// Find the current segment based on position/time
	var colorA, colorB Color
	var segmentProgress float64

	for i := 0; i < len(gradSlice)-1; i++ {
		if globalProgress >= gradSlice[i].Position && globalProgress < gradSlice[i+1].Position {
			colorA = gradSlice[i]
			colorB = gradSlice[i+1]
			segmentRange := colorB.Position - colorA.Position
			segmentProgress = (globalProgress - colorA.Position) / segmentRange
			break
		}
	}

	// Handle wrap-around (last to first)
	if globalProgress >= gradSlice[len(gradSlice)-1].Position {
		colorA = gradSlice[len(gradSlice)-1]
		colorB = gradSlice[0]
		segmentRange := 1.0 - colorA.Position + colorB.Position
		segmentProgress = (globalProgress - colorA.Position) / segmentRange
	}

	// Interpolate colors with brightness
	color := interpolateColorsWithBrightness(colorA, colorB, segmentProgress)

	// Fill buffer
	buf := map[int][]byte{}
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

	// Handle inversion/output
	if r.Inverted {
		r.Output = SetColorInverted(buf)
	} else {
		r.Output = SetColor(buf)
	}
}
