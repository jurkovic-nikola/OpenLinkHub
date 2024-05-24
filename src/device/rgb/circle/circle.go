package circle

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

// GenerateCircleColors will generate color based on start and end color
func GenerateCircleColors(numLEDs int, c1, c2 *structs.Color, factor, bts float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, numLEDs)
	for i := 0; i < numLEDs; i++ {
		color := interpolateColor(c1, c2, factor)
		color.Brightness = bts
		modify := brightness.ModifyBrightness(*color)
		colors[i] = struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
	}
	return colors
}

func Init(lc int, rgbLoopDuration time.Duration, rgbStartColor, rgbEndColor *structs.Color, bts float64) {
	st := time.Now()
	for {
		buf := map[int][]byte{}
		currentTime := time.Since(st)
		if currentTime >= rgbLoopDuration {
			break
		}

		for i := 0; i < lc; i++ {
			t := float64(i) / float64(lc) // Calculate interpolation factor
			colors := GenerateCircleColors(lc, rgbStartColor, rgbEndColor, t, bts)
			for j, color := range colors {
				if i < j-2 {
					buf[j] = []byte{0, 0, 0}
				} else {
					buf[j] = []byte{
						byte(color.R),
						byte(color.G),
						byte(color.B),
					}
				}
			}

			data := common.SetColor(buf)
			comm.WriteColor(opcodes.DataTypeSetColor, data)
			time.Sleep(40 * time.Millisecond)
		}
	}
}
