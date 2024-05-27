package rgb

import (
	"OpenLinkHub/src/device/brightness"
	"OpenLinkHub/src/device/comm"
	"OpenLinkHub/src/device/common"
	"OpenLinkHub/src/device/opcodes"
	"OpenLinkHub/src/structs"
	"time"
)

// generateCircleColors will generate color based on start and end color
func generateCircleColors(
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

// Circle will run RGB function
func (r *ActiveRGB) Circle() {
	for {
		select {
		case <-r.exit:
			return
		default:
			buf := map[int][]byte{}
			for i := 0; i < r.lightChannels; i++ {
				t := float64(i) / float64(r.lightChannels) // Calculate interpolation factor
				colors := generateCircleColors(r.lightChannels, r.rgbStartColor, r.rgbEndColor, t, r.bts)
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
}
