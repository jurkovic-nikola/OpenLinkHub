package rgb

import (
	"OpenICUELinkHub/src/device/brightness"
	"OpenICUELinkHub/src/device/comm"
	"OpenICUELinkHub/src/device/common"
	"OpenICUELinkHub/src/device/opcodes"
	"OpenICUELinkHub/src/structs"
	"math/rand"
	"time"
)

// generateFlickeringColors will generate color based on start and end color
func generateFlickeringColors(
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

// Flickering will run RGB function
func (r *ActiveRGB) Flickering() {
	for {
		select {
		default:
			buf := map[int][]byte{}
			if !r.rgbCustomColor {
				r.rgbStartColor = common.GenerateRandomColor(r.bts)
				r.rgbEndColor = common.GenerateRandomColor(r.bts)
			}

			for i := 0; i < r.lightChannels; i++ {
				t := float64(i) / float64(r.lightChannels) // Calculate interpolation factor
				colors := generateFlickeringColors(r.lightChannels, r.rgbStartColor, r.rgbEndColor, t, r.bts)
				for j, color := range colors {
					if rand.Intn(2) == 1 {
						buf[j] = []byte{0, 0, 0}
					} else {
						buf[j] = []byte{
							byte(color.R),
							byte(color.G),
							byte(color.B),
						}
					}
				}
				select {
				case <-r.exit:
					return
				case <-time.After(40 * time.Millisecond):
					data := common.SetColor(buf)
					comm.WriteColor(opcodes.DataTypeSetColor, data)
					time.Sleep(r.rgbLoopDuration)
				}
			}
		}
	}
}
