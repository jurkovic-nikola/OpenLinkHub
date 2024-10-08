package rgb

import "math"

// Wave will run RGB function
func (r *ActiveRGB) Wave(wavePosition float64) {
	buf := map[int][]byte{}
	color := r.RGBStartColor
	modify := ModifyBrightness(*color)

	for i := 0; i < r.LightChannels; i++ {
		wavePos := (wavePosition + float64(i)) / r.RgbModeSpeed
		intensity := 0.5 * (1 + math.Sin(2*math.Pi*wavePos))
		red := modify.Red * intensity
		green := modify.Green * intensity
		blue := modify.Blue * intensity
		buf[i] = []byte{byte(red), byte(green), byte(blue)}
	}
	r.Output = SetColor(buf)
}
