package rgb

import (
	"OpenLinkHub/src/device/brightness"
	"OpenLinkHub/src/device/comm"
	"OpenLinkHub/src/device/common"
	"OpenLinkHub/src/device/opcodes"
	"OpenLinkHub/src/structs"
	"time"
)

// generateColorwarpColors will generate color based on start and end color
func generateColorwarpColors(
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

// Colorwarp will run RGB function
func (r *ActiveRGB) Colorwarp() {
	buf := map[int][]byte{}
	c1 := common.GenerateRandomColor(r.bts)
	c2 := common.GenerateRandomColor(r.bts)
	for {
		select {
		case <-r.exit:
			return
		default:
			for i := 0; i <= r.smoothness; i++ {
				t := float64(i) / float64(r.smoothness) // Calculate interpolation factor
				for j := 0; j < r.lightChannels; j++ {
					colors := generateColorwarpColors(r.lightChannels, c1, c2, t, r.bts)
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
			c2 = common.GenerateRandomColor(r.bts)
		}
	}
}
