package rainbow

import (
	"OpenICUELinkHub/src/device/brightness"
	"OpenICUELinkHub/src/device/comm"
	"OpenICUELinkHub/src/device/common"
	"OpenICUELinkHub/src/device/opcodes"
	"OpenICUELinkHub/src/structs"
	"math"
)

// rainbowColor function returns an RGB color corresponding to a given position in the rainbow
func rainbowColor(position float64) (int, int, int) {
	// Normalize position to be between 0 and 1
	position = math.Mod(position, 1.0)

	// Calculate the color based on the position
	if position < 0.2 {
		// Red to Yellow
		return interpolate(1, 0, 0, 1, 1, 0, position/0.2)
	} else if position < 0.4 {
		// Yellow to Green
		return interpolate(1, 1, 0, 0, 1, 0, (position-0.2)/0.2)
	} else if position < 0.6 {
		// Green to Cyan
		return interpolate(0, 1, 0, 0, 1, 1, (position-0.4)/0.2)
	} else if position < 0.8 {
		// Cyan to Blue
		return interpolate(0, 1, 1, 0, 0, 1, (position-0.6)/0.2)
	} else {
		// Blue to Red
		return interpolate(0, 0, 1, 1, 0, 0, (position-0.8)/0.2)
	}
}

// Interpolate function to calculate the intermediate color
func interpolate(r1, g1, b1, r2, g2, b2 float64, fraction float64) (int, int, int) {
	r := r1 + fraction*(r2-r1)
	g := g1 + fraction*(g2-g1)
	b := b1 + fraction*(b2-b1)
	return int(r * 255), int(g * 255), int(b * 255)
}

// generateColors will generate color based on start and end color
func generateColors(lc int, elapsedTime, bts float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lc)
	for i := 0; i < lc; i++ {
		position := (float64(i) / float64(lc)) + (elapsedTime / 4.0)
		position = math.Mod(position, 1.0) // Keep position within the 0-1 range
		r, g, b := rainbowColor(position)

		color := &structs.Color{
			Red:        float64(r),
			Green:      float64(g),
			Blue:       float64(b),
			Brightness: bts,
		}

		modify := brightness.ModifyBrightness(*color)
		colors[i] = struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
	}
	return colors
}

// Init will run RGB function
func Init(lc int, elapsed, bts float64) {
	buf := map[int][]byte{}
	colors := generateColors(lc, elapsed, bts)
	for i, color := range colors {
		buf[i] = []byte{
			byte(color.R),
			byte(color.G),
			byte(color.B),
		}
	}
	data := common.SetColor(buf)
	comm.WriteColor(opcodes.DataTypeSetColor, data)
}
