package rgb

import (
	"math"
	"time"
)

// Sequential will run RGB function
func (r *ActiveRGB) Sequential(startTime *time.Time) {
	elapsed := time.Since(*startTime).Milliseconds()
	cycleDuration := r.RgbModeSpeed * 1000

	totalCycles := int(float64(elapsed) / cycleDuration)
	progress := math.Mod(float64(elapsed)/cycleDuration, 1.0)

	ledCount := r.LightChannels
	step := int(progress * float64(ledCount))
	if step >= ledCount {
		step = ledCount - 1
	}

	currentColor := GenerateRandomColorSeeded(int64(totalCycles), r.RGBBrightness)

	colors := make([]struct{ R, G, B float64 }, ledCount)

	for i := 0; i < ledCount; i++ {
		if i <= step {
			colors[i] = struct{ R, G, B float64 }{currentColor.Red, currentColor.Green, currentColor.Blue}
		} else {
			if totalCycles > 0 && i < ledCount {
				prevColor := GenerateRandomColorSeeded(int64(totalCycles-1), r.RGBBrightness)
				colors[i] = struct{ R, G, B float64 }{prevColor.Red, prevColor.Green, prevColor.Blue}
			} else {
				colors[i] = struct{ R, G, B float64 }{0, 0, 0}
			}
		}
	}

	for j, c := range colors {
		if len(r.Buffer) > 0 {
			r.Buffer[j] = byte(c.R)
			r.Buffer[j+r.ColorOffset] = byte(c.G)
			r.Buffer[j+(r.ColorOffset*2)] = byte(c.B)
		}
	}

	raw := map[int][]byte{}
	for j, c := range colors {
		raw[j] = []byte{byte(c.R), byte(c.G), byte(c.B)}
	}
	r.Raw = raw

	if r.Inverted {
		r.Output = SetColorInverted(raw)
	} else {
		r.Output = SetColor(raw)
	}
	
	if progress >= 1.0 {
		*startTime = time.Now()
	}
}
