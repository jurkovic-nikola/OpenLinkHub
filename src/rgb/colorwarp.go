package rgb

// generateColorwarpColors will generate color based on start and end color
func generateColorwarpColors(
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

// Colorwarp will run RGB function
func (r *ActiveRGB) Colorwarp(i int, RGBStartColor *Color, RGBEndColor *Color) {
	buf := map[int][]byte{}

	t := float64(i) / float64(r.Smoothness) // Calculate interpolation factor
	for j := 0; j < r.LightChannels; j++ {
		colors := generateColorwarpColors(r.LightChannels, RGBStartColor, RGBEndColor, t, r.RGBBrightness)
		buf[j] = []byte{
			byte(colors[j].R),
			byte(colors[j].G),
			byte(colors[j].B),
		}
		if r.IsAIO && r.HasLCD {
			if j > 15 && j < 20 {
				buf[j] = []byte{0, 0, 0}
			}
		}
	}
	if r.Inverted {
		r.Output = SetColorInverted(buf)
	} else {
		r.Output = SetColor(buf)
	}
}
