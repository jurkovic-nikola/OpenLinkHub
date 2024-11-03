package rgb

import (
	"math/rand"
)

func stormColorEffect(c1 *Color, bts float64) (uint8, uint8, uint8) {
	r, g, b := c1.Red, c1.Green, c1.Blue
	if rand.Float32() < 0.001 {
		r, g, b = 255, 255, 255
	}
	color := &Color{Red: r, Green: g, Blue: b, Brightness: bts}
	modify := ModifyBrightness(*color)
	return uint8(modify.Red), uint8(modify.Green), uint8(modify.Blue)
}

// Storm will run RGB function
func (r *ActiveRGB) Storm() {
	buf := map[int][]byte{}
	for i := 0; i < r.LightChannels; i++ {
		red, green, blue := stormColorEffect(r.RGBStartColor, r.RGBStartColor.Brightness)
		buf[i] = []byte{red, green, blue}
		if r.IsAIO && r.HasLCD {
			if i > 15 && i < 20 {
				buf[i] = []byte{0, 0, 0}
			}
		}
	}
	r.Output = SetColor(buf)
}
