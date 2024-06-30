package rgb

// generateColors will generate color based on start and end color
func generateColor(
	c1,
	c2 *Color,
	factor,
	bts float64,
) struct{ R, G, B float64 } {
	color := interpolateColor(c1, c2, factor)
	color.Brightness = bts
	modify := ModifyBrightness(*color)
	return struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
}

// Spinner will run RGB function
func (r *ActiveRGB) Spinner(i int) {
	buf := map[int][]byte{}
	t := float64(i) / float64(r.LightChannels) // Calculate interpolation factor
	colors := generateCircleColors(r.LightChannels, r.RGBStartColor, r.RGBEndColor, t, r.RGBBrightness)
	for j, color := range colors {
		if j == i {
			buf[j] = []byte{
				byte(color.R),
				byte(color.G),
				byte(color.B),
			}
		} else {
			buf[j] = []byte{0, 0, 0}
		}
	}
	r.Output = SetColor(buf)
}
