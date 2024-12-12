package rgb

// Rotator will run RGB function
func (r *ActiveRGB) Rotator(hue int) {
	hue = hue * int(r.RgbModeSpeed)
	buf := map[int][]byte{}

	for j := 0; j < r.LightChannels; j++ {
		color := &Color{
			Red:        float64(byte(HsvToRgb(hue+j*5, 255, int(r.RGBStartColor.Red)))),
			Green:      float64(byte(HsvToRgb(hue+j*5, 255, int(r.RGBStartColor.Green)))),
			Blue:       float64(byte(HsvToRgb(hue+j*5, 255, int(r.RGBStartColor.Blue)))),
			Brightness: r.RGBBrightness,
		}

		modify := ModifyBrightness(*color)
		buf[j] = []byte{byte(modify.Red), byte(modify.Green), byte(modify.Blue)}
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

// HsvToRgb converts an HSV color to RGB color
func HsvToRgb(hue, saturation, value int) uint32 {

	h := hue % 360
	s := float64(saturation) / 255.0
	v := float64(value) / 255.0

	c := v * s
	x := c * (1 - absFloat64(float64(h)/60.0-float64((h/60)*2)))
	m := v - c

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	// Convert to 24-bit color and return
	return uint32((int((r+m)*255) << 16) | (int((g+m)*255) << 8) | int((b+m)*255))
}

// Helper function to get absolute float64
func absFloat64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
