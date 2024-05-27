package rgb

import (
	"OpenLinkHub/src/device/brightness"
	"OpenLinkHub/src/device/comm"
	"OpenLinkHub/src/device/common"
	"OpenLinkHub/src/device/opcodes"
	"OpenLinkHub/src/structs"
	"time"
)

// generateColorPulseColors will generate color based on start and end color
func generateColorPulseColors(
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

// Colorpulse will run RGB function
func (r *ActiveRGB) Colorpulse() {
	for {
		select {
		case <-r.exit:
			return
		default:
			buf := map[int][]byte{}
			for i := 0; i <= r.smoothness; i++ {
				t := float64(i) / float64(r.smoothness) // Calculate interpolation factor
				colors := generateColorPulseColors(r.lightChannels, r.rgbStartColor, r.rgbEndColor, t, r.bts)

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
			time.Sleep(r.rgbLoopDuration) // Loop duration
		}
	}
}
