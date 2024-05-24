package colorshift

import (
	"OpenICUELinkHub/src/device/brightness"
	"OpenICUELinkHub/src/device/comm"
	"OpenICUELinkHub/src/device/common"
	"OpenICUELinkHub/src/device/opcodes"
	"OpenICUELinkHub/src/structs"
	"time"
)

// InterpolateColor performs linear interpolation between two colors
func interpolateColor(c1, c2 *structs.Color, t float64) *structs.Color {
	return &structs.Color{
		Red:   common.Lerp(c1.Red, c2.Red, t),
		Green: common.Lerp(c1.Green, c2.Green, t),
		Blue:  common.Lerp(c1.Blue, c2.Blue, t),
	}
}

// generateColors will generate color based on start and end color
func generateColors(lc int, c1, c2 *structs.Color, factor, bts float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lc)
	for i := 0; i < lc; i++ {
		color := interpolateColor(c1, c2, factor)
		color.Brightness = bts
		modify := brightness.ModifyBrightness(*color)
		colors[i] = struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
	}
	return colors
}

// Init will run RGB function
func Init(lc, smoothness int, rgbCustomColor bool, rgbLoopDuration time.Duration, rgbStartColor, rgbEndColor *structs.Color, bts float64) {
	st := time.Now()
	buf := map[int][]byte{}

	if !rgbCustomColor {
		rgbStartColor = common.GenerateRandomColor(bts)
		rgbEndColor = common.GenerateRandomColor(bts)
	}

	for {
		currentTime := time.Since(st)
		if currentTime >= rgbLoopDuration {
			break
		}

		// Initial
		for i := 0; i <= smoothness; i++ {
			t := float64(i) / float64(smoothness) // Calculate interpolation factor
			colors := generateColors(lc, rgbStartColor, rgbEndColor, t, bts)

			// Update LED channels
			for j, color := range colors {
				buf[j] = []byte{
					byte(color.R),
					byte(color.G),
					byte(color.B),
				}
			}
			data := common.SetColor(buf)
			comm.WriteColor(opcodes.DataTypeSetColor, data)
			time.Sleep(40 * time.Millisecond) // Adjust sleep time for smoother animation
		}

		// Wait for reverse
		time.Sleep(rgbLoopDuration)

		// Reverse
		for i := 0; i <= smoothness; i++ {
			t := float64(i) / float64(smoothness) // Calculate interpolation factor
			colors := generateColors(lc, rgbEndColor, rgbStartColor, t, bts)

			// Update LED channels
			for j, color := range colors {
				buf[j] = []byte{
					byte(color.R),
					byte(color.G),
					byte(color.B),
				}
			}
			data := common.SetColor(buf)
			comm.WriteColor(opcodes.DataTypeSetColor, data)
			time.Sleep(40 * time.Millisecond) // Adjust sleep time for smoother animation
		}

		time.Sleep(rgbLoopDuration) // Loop duration
	}
}
