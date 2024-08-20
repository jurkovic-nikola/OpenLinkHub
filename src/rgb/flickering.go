package rgb

import (
	"math/rand"
)

// generateFlickeringColors will generate color based on start and end color
func generateFlickeringColors(
	lightChannels int,
	c1,
	c2 *Color,
	factor,
	bts float64,
) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lightChannels)
	for i := 0; i < lightChannels; i++ {
		color := interpolateColor(c1, c2, factor)
		color.Brightness = bts
		modify := ModifyBrightness(*color)
		colors[i] = struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
	}
	return colors
}

// Flickering will run RGB function
func (r *ActiveRGB) Flickering(i int) {
	buf := map[int][]byte{}
	t := float64(i) / float64(r.LightChannels) // Calculate interpolation factor
	colors := generateFlickeringColors(r.LightChannels, r.RGBStartColor, r.RGBEndColor, t, r.RGBBrightness)
	for j, color := range colors {
		if rand.Intn(2) == 1 {
			buf[j] = []byte{0, 0, 0}
			if r.ContainsPump && r.HasLCD {
				if j > 15 && j < 20 {
					buf[j] = []byte{0, 0, 0}
				}
			}
		} else {
			buf[j] = []byte{
				byte(color.R),
				byte(color.G),
				byte(color.B),
			}
			if r.ContainsPump && r.HasLCD {
				if j > 15 && j < 20 {
					buf[j] = []byte{0, 0, 0}
				}
			}
		}
	}
	r.Output = SetColor(buf)
}
