package rgb

import (
	"math"
	"time"
)

func wrap01(x float64) float64 {
	x = math.Mod(x, 1.0)
	if x < 0 {
		x += 1.0
	}
	return x
}

func arcColor(travelPos, elapsed float64, hueOffset float64) (int, int, int) {
	h := wrap01((elapsed / 40.0) + hueOffset + (travelPos * 0.30))

	s := 0.92
	v := 1.0

	i := int(h * 6.0)
	f := h*6.0 - float64(i)
	p := v * (1.0 - s)
	q := v * (1.0 - s*f)
	t := v * (1.0 - s*(1.0-f))

	var r, g, b float64
	switch i % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	default:
		r, g, b = v, p, q
	}

	return int(math.Round(r * 255)), int(math.Round(g * 255)), int(math.Round(b * 255))
}

func gradientArcColor(startColor, endColor Color, position, elapsed float64, phaseOffset float64) (int, int, int) {
	t := 0.5 + 0.5*math.Sin((2.0*math.Pi*elapsed/40.0)+(2.0*math.Pi*(position+phaseOffset)))
	return lerpColor(startColor, endColor, t)
}

func arcColors(lightChannels int, elapsedTime, bts float64, random bool, startColor, endColor Color) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lightChannels)
	if lightChannels <= 0 {
		return colors
	}

	outerCount := lightChannels
	innerCount := 0
	if lightChannels >= 16 {
		outerCount = 24
		innerCount = lightChannels - outerCount
	}

	outerCenter := math.Mod(elapsedTime/10, 1.0)
	innerCenter := math.Mod(0.18+(elapsedTime/10), 1.0)
	outerWidth := 0.23
	innerWidth := 0.36
	ambientFloor := 0.02
	pulse := 1.0

	for i := 0; i < lightChannels; i++ {
		position := float64(i) / float64(lightChannels)
		mask := 0.0
		hueOffset := 0.0
		phaseOffset := 0.0

		if i < outerCount {
			outerPos := float64(i) / float64(outerCount)
			mask = arcMask(outerPos, outerCenter, outerWidth)
			hueOffset = 0.00
			phaseOffset = 0.00
			position = outerPos
		} else if innerCount > 0 {
			innerIdx := i - outerCount
			innerPos := float64(innerIdx) / float64(innerCount)
			mask = arcMask(innerPos, innerCenter, innerWidth)
			hueOffset = 0.38
			phaseOffset = 0.18
			position = innerPos
		}

		intensity := ambientFloor + (mask * pulse)
		if intensity > 1.0 {
			intensity = 1.0
		}

		var rr, gg, bb int
		if random {
			rr, gg, bb = arcColor(position, elapsedTime, hueOffset)
		} else {
			rr, gg, bb = gradientArcColor(startColor, endColor, position, elapsedTime, phaseOffset)
		}

		color := &Color{
			Red:        float64(rr) * intensity,
			Green:      float64(gg) * intensity,
			Blue:       float64(bb) * intensity,
			Brightness: bts,
		}

		modify := ModifyBrightness(*color)
		colors[i] = struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
	}

	return colors
}

// Arc will generate arc based rgb effect
func (r *ActiveRGB) Arc(startTime time.Time) {
	elapsed := time.Since(startTime).Seconds()

	speedFactor := 10.0
	if r.RgbModeSpeed > 0 {
		speedFactor = 10.0 / r.RgbModeSpeed
	}

	random := true
	startColor := Color{Red: 255, Green: 64, Blue: 160}
	endColor := Color{Red: 64, Green: 180, Blue: 255}

	if r.RGBStartColor != nil && r.RGBEndColor != nil {
		startColor = *r.RGBStartColor
		endColor = *r.RGBEndColor
		random = r.RGBStartColor.Red == r.RGBEndColor.Red &&
			r.RGBStartColor.Green == r.RGBEndColor.Green &&
			r.RGBStartColor.Blue == r.RGBEndColor.Blue
	}

	buf := map[int][]byte{}
	colors := arcColors(r.LightChannels, elapsed*speedFactor, r.RGBBrightness, random, startColor, endColor)
	for i, color := range colors {
		if len(r.Buffer) > 0 {
			r.Buffer[i] = byte(color.R)
			r.Buffer[i+r.ColorOffset] = byte(color.G)
			r.Buffer[i+(r.ColorOffset*2)] = byte(color.B)
		} else {
			buf[i] = []byte{
				byte(color.R),
				byte(color.G),
				byte(color.B),
			}
			if r.IsAIO && r.HasLCD {
				if i > 15 && i < 20 {
					buf[i] = []byte{0, 0, 0}
				}
			}
		}
	}

	r.Raw = buf

	if r.Inverted {
		r.Output = SetColorInverted(buf)
	} else {
		r.Output = SetColor(buf)
	}
}
