package spinner

import (
	"OpenICUELinkHub/src/device/brightness"
	"OpenICUELinkHub/src/device/comm"
	"OpenICUELinkHub/src/device/common"
	"OpenICUELinkHub/src/device/opcodes"
	"OpenICUELinkHub/src/structs"
	"time"
)

var exit = make(chan bool)

// interpolateColor performs linear interpolation between two colors
func interpolateColor(c1, c2 *structs.Color, t float64) *structs.Color {
	return &structs.Color{
		Red:   common.Lerp(c1.Red, c2.Red, t),
		Green: common.Lerp(c1.Green, c2.Green, t),
		Blue:  common.Lerp(c1.Blue, c2.Blue, t),
	}
}

// generateColors will generate color based on start and end color
func generateColor(
	c1,
	c2 *structs.Color,
	factor,
	bts float64,
) struct{ R, G, B float64 } {
	color := interpolateColor(c1, c2, factor)
	color.Brightness = bts
	modify := brightness.ModifyBrightness(*color)
	return struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
}

func Stop() {
	exit <- true
}

// Init will run RGB function
func Init(
	lightChannels int,
	rgbStartColor,
	rgbEndColor *structs.Color,
	bts float64,
) {
	buf := make(map[int][]byte, lightChannels)
	for {
		// Set the current LED to red and the rest to off
		for i := 0; i < lightChannels; i++ {
			t := float64(i) / float64(lightChannels) // Calculate interpolation factor
			color := generateColor(rgbStartColor, rgbEndColor, t, bts)

			// Turn all LEDs off
			for j := range buf {
				buf[j] = []byte{0, 0, 0}
			}
			// Set the current LED to red
			buf[i] = []byte{
				byte(color.R),
				byte(color.G),
				byte(color.B),
			}

			select {
			case <-exit:
				return
			case <-time.After(40 * time.Millisecond):
				data := common.SetColor(buf)
				comm.WriteColor(opcodes.DataTypeSetColor, data)
			}
		}
	}
}
