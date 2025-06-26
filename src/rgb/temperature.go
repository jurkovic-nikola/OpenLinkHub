package rgb

import (
	"math"
)

// smoothColor smooths the color with smoothing factor
func smoothColor(old, new *Color, smoothing float64) *Color {
	return &Color{
		Red:   old.Red + smoothing*(new.Red-old.Red),
		Green: old.Green + smoothing*(new.Green-old.Green),
		Blue:  old.Blue + smoothing*(new.Blue-old.Blue),
	}
}

// interpolateTemperatureColor interpolates between 2 colors
func interpolateTemperatureColor(start, end *Color, t float64, brightness float64) *Color {
	return &Color{
		Red:   (start.Red + (end.Red-start.Red)*t) * brightness,
		Green: (start.Green + (end.Green-start.Green)*t) * brightness,
		Blue:  (start.Blue + (end.Blue-start.Blue)*t) * brightness,
	}
}

// MapTemperatureToPercent maps a temperature value within a range to a percentage between 0 and 1
func MapTemperatureToPercent(temp, minTemp, maxTemp float64) float64 {
	clampedTemp := math.Max(minTemp, math.Min(maxTemp, temp))
	if maxTemp == minTemp {
		return 0.5
	}
	return (clampedTemp - minTemp) / (maxTemp - minTemp)
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
	t := MapTemperatureToPercent(smoothedTemp, r.MinTemp, r.MaxTemp)
	targetColor := interpolateTemperatureColor(r.RGBStartColor, r.RGBEndColor, t, r.RGBBrightness)

	// Initialize PreviousColor if needed
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
