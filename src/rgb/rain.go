package rgb

import (
	"math"
	"time"
)

// randomRainColor will generate random rain colors
func randomRainColor(slot float64, channel float64) (float64, float64, float64) {
	h := random01(slot, channel, 1.0)
	s := 0.85 + 0.15*random01(slot, channel, 2.0)
	v := 0.90 + 0.10*random01(slot, channel, 3.0)

	h = math.Mod(h, 1.0)
	if h < 0 {
		h += 1.0
	}

	i := int(h * 6.0)
	f := h*6.0 - float64(i)

	p := v * (1.0 - s)
	q := v * (1.0 - f*s)
	t := v * (1.0 - (1.0-f)*s)

	switch i % 6 {
	case 0:
		return v * 255.0, t * 255.0, p * 255.0
	case 1:
		return q * 255.0, v * 255.0, p * 255.0
	case 2:
		return p * 255.0, v * 255.0, t * 255.0
	case 3:
		return p * 255.0, q * 255.0, v * 255.0
	case 4:
		return t * 255.0, p * 255.0, v * 255.0
	default:
		return v * 255.0, p * 255.0, q * 255.0
	}
}

// generateRainColors will generate random rain colors or static from config
func generateRainColors(
	lightChannels int,
	elapsedTime, brightness, speed float64,
	random bool,
	startColor, endColor *Color,
) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lightChannels)

	if lightChannels <= 0 {
		return colors
	}
	if speed <= 0 {
		speed = 1.0
	}

	stepDuration := 0.115 / speed
	dropDuration := 0.55 / math.Sqrt(speed)
	spawnChance := clampFloat01(0.18 + (speed-1.0)*0.035)

	currentSlot := int(math.Floor(elapsedTime / stepDuration))

	for i := 0; i < lightChannels; i++ {
		var bestR, bestG, bestB float64
		bestIntensity := 0.0

		for back := 0; back < 6; back++ {
			slot := currentSlot - back
			if slot < 0 {
				continue
			}

			slotF := float64(slot)
			channelF := float64(i + 1)

			if random01(slotF, channelF, 10.0) > spawnChance {
				continue
			}

			jitter := random01(slotF, channelF, 20.0) * stepDuration
			start := slotF*stepDuration + jitter
			age := elapsedTime - start
			if age < 0 || age > dropDuration {
				continue
			}

			norm := age / dropDuration
			intensity := math.Pow(1.0-norm, 1.9)
			intensity *= 0.78 + 0.22*random01(slotF, channelF, 30.0)

			var rr, gg, bb float64

			if random || startColor == nil || endColor == nil {
				rr, gg, bb = randomRainColor(slotF, channelF)
			} else {
				if norm < 0.5 {
					rr = startColor.Red
					gg = startColor.Green
					bb = startColor.Blue
				} else {
					rr = endColor.Red
					gg = endColor.Green
					bb = endColor.Blue
				}
			}

			rr *= intensity
			gg *= intensity
			bb *= intensity

			if intensity > bestIntensity {
				bestIntensity = intensity
				bestR = rr
				bestG = gg
				bestB = bb
			}
		}

		color := &Color{
			Red:        bestR,
			Green:      bestG,
			Blue:       bestB,
			Brightness: brightness,
		}

		modify := ModifyBrightness(*color)
		colors[i] = struct{ R, G, B float64 }{
			R: modify.Red,
			G: modify.Green,
			B: modify.Blue,
		}
	}

	return colors
}

// Rain will generate rain rgb effect
func (r *ActiveRGB) Rain(startTime time.Time) {
	elapsed := time.Since(startTime).Seconds()

	speedFactor := 1.0
	switch r.RgbModeSpeed {
	case 1:
		speedFactor = 0.5
	case 2:
		speedFactor = 0.3
	case 3:
		speedFactor = 0.1
	}

	random := false
	if r.RGBStartColor != nil && r.RGBEndColor != nil && *r.RGBStartColor == *r.RGBEndColor {
		random = true
	}

	buf := map[int][]byte{}
	colors := generateRainColors(
		r.LightChannels,
		elapsed,
		r.RGBBrightness,
		speedFactor,
		random,
		r.RGBStartColor,
		r.RGBEndColor,
	)

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
