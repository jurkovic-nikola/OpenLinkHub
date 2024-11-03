package rgb

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
			if r.IsAIO && r.HasLCD {
				if j > 15 && j < 20 {
					buf[j] = []byte{0, 0, 0}
				}
			}
		} else {
			buf[j] = []byte{0, 0, 0}
			if r.IsAIO && r.HasLCD {
				if j > 15 && j < 20 {
					buf[j] = []byte{0, 0, 0}
				}
			}
		}
	}
	r.Output = SetColor(buf)
}
