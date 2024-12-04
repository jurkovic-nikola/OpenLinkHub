package rgb

import (
	"OpenLinkHub/src/common"
	"math"
)

// GenerateTemperatureColor will generate temperature color based on min and max temperature value and given t factor
func GenerateTemperatureColor(c1, c2 *Color, t, brightness float64) *Color {
	// Lerp
	r := uint8(common.Lerp(c1.Red, c2.Red, t))
	g := uint8(common.Lerp(c1.Green, c2.Green, t))
	b := uint8(common.Lerp(c1.Blue, c2.Blue, t))

	// Generate new color
	endColor := Color{
		Red:        float64(r),
		Green:      float64(g),
		Blue:       float64(b),
		Brightness: brightness,
	}

	// Brightness
	modify := ModifyBrightness(endColor)
	return modify
}

// MapTemperatureToPercent maps a temperature value within a range to a percentage between 0 and 1
func MapTemperatureToPercent(temp, minTemp, maxTemp float64) float64 {
	clampedTemp := math.Max(minTemp, math.Min(maxTemp, temp))
	return (clampedTemp - minTemp) / (maxTemp - minTemp)
}

// Temperature will return color based from min/max and current temperature factor
func (r *ActiveRGB) Temperature(currentTemp float64, i int, current *Color) *Color {
	s := float64(i) / float64(r.Smoothness)
	buf := map[int][]byte{}
	t := MapTemperatureToPercent(currentTemp, r.MinTemp, r.MaxTemp)
	result := GenerateTemperatureColor(r.RGBStartColor, r.RGBEndColor, t, r.RGBStartColor.Brightness)
	color := interpolateColor(current, result, s)

	for j := 0; j < r.LightChannels; j++ {
		buf[j] = []byte{
			byte(color.Red),
			byte(color.Green),
			byte(color.Blue),
		}
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
	return color
}
