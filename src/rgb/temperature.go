package rgb

import (
	"math"
)

func clamp(v, min, max float64) float64 {
	return math.Max(min, math.Min(max, v))
}

func smoothColor(old, new *Color, smoothing float64) *Color {
	return &Color{
		Red:   old.Red + smoothing*(new.Red-old.Red),
		Green: old.Green + smoothing*(new.Green-old.Green),
		Blue:  old.Blue + smoothing*(new.Blue-old.Blue),
	}
}

func interpolateTempColor(start, end *Color, t float64, brightness float64) *Color {
	t = clamp(t, 0, 1)
	return &Color{
		Red:   (start.Red + (end.Red-start.Red)*t) * brightness,
		Green: (start.Green + (end.Green-start.Green)*t) * brightness,
		Blue:  (start.Blue + (end.Blue-start.Blue)*t) * brightness,
	}
}

func mapTemperatureInRange(temp, minTemp, maxTemp float64) float64 {
	if maxTemp == minTemp {
		return 0.5
	}
	clampedTemp := clamp(temp, math.Min(minTemp, maxTemp), math.Max(minTemp, maxTemp))
	return (clampedTemp - minTemp) / (maxTemp - minTemp)
}

func interpolateTemperatureColor(start, middle, end *Color, currentTemp, brightness float64) *Color {
	if middle == nil {
		t := mapTemperatureInRange(currentTemp, start.Temperature, end.Temperature)
		return interpolateTempColor(start, end, t, brightness)
	}

	c1, c2, c3 := start, middle, end
	if c1.Temperature > c2.Temperature {
		c1, c2 = c2, c1
	}
	if c2.Temperature > c3.Temperature {
		c2, c3 = c3, c2
	}
	if c1.Temperature > c2.Temperature {
		c1, c2 = c2, c1
	}

	if currentTemp <= c1.Temperature {
		return interpolateColor(c1, c1, 0, brightness)
	}

	if currentTemp <= c2.Temperature {
		t := mapTemperatureInRange(currentTemp, c1.Temperature, c2.Temperature)
		return interpolateColor(c1, c2, t, brightness)
	}

	if currentTemp <= c3.Temperature {
		t := mapTemperatureInRange(currentTemp, c2.Temperature, c3.Temperature)
		return interpolateColor(c2, c3, t, brightness)
	}

	return interpolateColor(c3, c3, 0, brightness)
}

func (r *ActiveRGB) SmoothTemperature(currentTemp float64) float64 {
	if r.TempAlpha == 0 {
		r.TempAlpha = 0.1
	}
	if r.PreviousTemp == 0 {
		r.PreviousTemp = currentTemp
	}
	r.PreviousTemp = r.PreviousTemp + r.TempAlpha*(currentTemp-r.PreviousTemp)
	return r.PreviousTemp
}

func (r *ActiveRGB) Temperature(currentTemp float64) {
	smoothedTemp := r.SmoothTemperature(currentTemp)
	targetColor := interpolateTemperatureColor(
		r.RGBStartColor,
		r.RGBMiddleColor,
		r.RGBEndColor,
		smoothedTemp,
		r.RGBBrightness,
	)

	if r.PreviousColor == nil {
		r.PreviousColor = &Color{
			Red:   targetColor.Red,
			Green: targetColor.Green,
			Blue:  targetColor.Blue,
		}
	}

	smoothing := 0.1 // or whatever you wish
	smoothed := smoothColor(r.PreviousColor, targetColor, smoothing)
	r.PreviousColor = smoothed

	buf := map[int][]byte{}

	for j := 0; j < r.LightChannels; j++ {
		if len(r.Buffer) > 0 {
			r.Buffer[j] = byte(smoothed.Red)
			r.Buffer[j+r.ColorOffset] = byte(smoothed.Green)
			r.Buffer[j+(r.ColorOffset*2)] = byte(smoothed.Blue)
		} else {
			buf[j] = []byte{
				byte(smoothed.Red),
				byte(smoothed.Green),
				byte(smoothed.Blue),
			}
			if r.IsAIO && r.HasLCD {
				if j > 15 && j < 20 {
					buf[j] = []byte{0, 0, 0}
				}
			}
		}
	}
	// Raw colors
	r.Raw = buf

	if r.Inverted {
		r.Output = SetColorInverted(buf)
	} else {
		r.Output = SetColor(buf)
	}
}
