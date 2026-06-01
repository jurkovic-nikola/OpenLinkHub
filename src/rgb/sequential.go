package rgb

import (
	"math"
	"time"
)

// Sequential will run RGB function
func (r *ActiveRGB) Sequential(startTime *time.Time) {
	random := false
	if r.RGBStartColor != nil && r.RGBEndColor != nil && *r.RGBStartColor == *r.RGBEndColor {
		random = true
	}

	elapsed := time.Since(*startTime).Milliseconds()
	cycleDuration := r.RgbModeSpeed * 1000
	if cycleDuration <= 0 {
		cycleDuration = 1000
	}

	totalCycles := int(float64(elapsed) / cycleDuration)
	progress := math.Mod(float64(elapsed)/cycleDuration, 1.0)

	ledCount := r.LightChannels
	if ledCount <= 0 {
		return
	}

	step := int(progress * float64(ledCount))
	if step >= ledCount {
		step = ledCount - 1
	}

	type rgb struct {
		R, G, B float64
	}

	var currentColor rgb
	var prevColor rgb

	if random {
		curr := GenerateRandomColorSeeded(int64(totalCycles), r.RGBBrightness)
		currentColor = rgb{curr.Red, curr.Green, curr.Blue}

		if totalCycles > 0 {
			prev := GenerateRandomColorSeeded(int64(totalCycles-1), r.RGBBrightness)
			prevColor = rgb{prev.Red, prev.Green, prev.Blue}
		} else {
			prevColor = rgb{0, 0, 0}
		}
	} else {
		start := rgb{
			R: r.RGBStartColor.Red,
			G: r.RGBStartColor.Green,
			B: r.RGBStartColor.Blue,
		}
		end := rgb{
			R: r.RGBEndColor.Red,
			G: r.RGBEndColor.Green,
			B: r.RGBEndColor.Blue,
		}

		if totalCycles%2 == 0 {
			currentColor = start
			prevColor = end
		} else {
			currentColor = end
			prevColor = start
		}
	}

	colors := make([]rgb, ledCount)

	for i := 0; i < ledCount; i++ {
		if i <= step {
			colors[i] = currentColor
		} else {
			if totalCycles > 0 || !random {
				colors[i] = prevColor
			} else {
				colors[i] = rgb{0, 0, 0}
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

}
