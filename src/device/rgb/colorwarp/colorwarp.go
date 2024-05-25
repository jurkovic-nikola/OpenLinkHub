package colorwarp

import (
	"OpenICUELinkHub/src/device/brightness"
	"OpenICUELinkHub/src/device/comm"
	"OpenICUELinkHub/src/device/common"
	"OpenICUELinkHub/src/device/opcodes"
	"OpenICUELinkHub/src/structs"
	"time"
)

// interpolateColor performs linear interpolation between two colors
func interpolateColor(c1, c2 *structs.Color, t float64) *structs.Color {
	return &structs.Color{
		Red:   common.Lerp(c1.Red, c2.Red, t),
		Green: common.Lerp(c1.Green, c2.Green, t),
		Blue:  common.Lerp(c1.Blue, c2.Blue, t),
	}
}

// generateColors will generate color based on start and end color
func generateColors(
	lightChannels int,
	c1,
	c2 *structs.Color,
	factor,
	bts float64,
) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lightChannels)
	for i := 0; i < lightChannels; i++ {
		color := interpolateColor(c1, c2, factor)
		color.Brightness = bts
		modify := brightness.ModifyBrightness(*color)
		colors[i] = struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
	}
	return colors
}

// Init will run RGB function
func Init(lightChannels, smoothness int, bts float64) {
	buf := map[int][]byte{}
	c1 := common.GenerateRandomColor(bts)
	c2 := common.GenerateRandomColor(bts)
	for {
		for i := 0; i <= smoothness; i++ {
			t := float64(i) / float64(smoothness) // Calculate interpolation factor
			for j := 0; j < lightChannels; j++ {
				colors := generateColors(lightChannels, c1, c2, t, bts)
				buf[j] = []byte{
					byte(colors[j].R),
					byte(colors[j].G),
					byte(colors[j].B),
				}
			}
			data := common.SetColor(buf)
			comm.WriteColor(opcodes.DataTypeSetColor, data)
			time.Sleep(40 * time.Millisecond) // Adjust sleep time for smoother animation
		}
		c1 = c2
		c2 = common.GenerateRandomColor(bts)
	}
}
