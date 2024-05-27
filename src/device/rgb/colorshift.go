package rgb

import (
	"OpenLinkHub/src/device/brightness"
	"OpenLinkHub/src/device/comm"
	"OpenLinkHub/src/device/common"
	"OpenLinkHub/src/device/opcodes"
	"OpenLinkHub/src/structs"
	"time"
)

// generateColorshiftColors will generate color based on start and end color
func generateColorshiftColors(
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

// Colorshift will run RGB function
func (r *ActiveRGB) Colorshift() {
	buf := map[int][]byte{}

	if !r.rgbCustomColor {
		r.rgbStartColor = common.GenerateRandomColor(r.bts)
		r.rgbEndColor = common.GenerateRandomColor(r.bts)
	}

	for {
		select {
		case <-r.exit:
			return
		default:
			// Initial
			for i := 0; i <= r.smoothness; i++ {
				t := float64(i) / float64(r.smoothness) // Calculate interpolation factor
				colors := generateColorshiftColors(r.lightChannels, r.rgbStartColor, r.rgbEndColor, t, r.bts)

				// Update LED channels
				for j, color := range colors {
					buf[j] = []byte{
						byte(color.R),
						byte(color.G),
						byte(color.B),
					}
				}
				select {
				case <-r.exit:
					return
				case <-time.After(40 * time.Millisecond):
					data := common.SetColor(buf)
					comm.WriteColor(opcodes.DataTypeSetColor, data)
				}
			}

			select {
			case <-r.exit:
				return
			case <-time.After(r.rgbLoopDuration):
			}

			// Reverse
			for i := 0; i <= r.smoothness; i++ {
				t := float64(i) / float64(r.smoothness) // Calculate interpolation factor
				colors := generateColorshiftColors(r.lightChannels, r.rgbEndColor, r.rgbStartColor, t, r.bts)

				// Update LED channels
				for j, color := range colors {
					buf[j] = []byte{
						byte(color.R),
						byte(color.G),
						byte(color.B),
					}
				}
				select {
				case <-r.exit:
					return
				case <-time.After(40 * time.Millisecond):
					data := common.SetColor(buf)
					comm.WriteColor(opcodes.DataTypeSetColor, data)
				}
			}
		}
	}
}
