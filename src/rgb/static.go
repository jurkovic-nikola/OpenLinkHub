package rgb

// Static will run RGB function
func (r *ActiveRGB) Static() {
	buf := map[int][]byte{}
	r.RGBStartColor.Brightness = r.RGBBrightness
	modify := ModifyBrightness(*r.RGBStartColor)
	for j := 0; j < r.LightChannels; j++ {
		buf[j] = []byte{
			byte(modify.Red),
			byte(modify.Green),
			byte(modify.Blue),
		}
	}
	r.Output = SetColor(buf)
}
