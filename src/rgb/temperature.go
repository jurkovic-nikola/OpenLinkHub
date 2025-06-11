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
func (r *ActiveRGB) Temperature(currentTemp float64) {
	startColor := r.RGBStartColor
	buf := map[int][]byte{}
	t := MapTemperatureToPercent(currentTemp, r.MinTemp, r.MaxTemp)
	//result := GenerateTemperatureColor(r.RGBStartColor, r.RGBEndColor, t, r.RGBStartColor.Brightness)
	startColor = interpolateColor(r.RGBStartColor, r.RGBEndColor, t, r.RGBBrightness)

	for j := 0; j < r.LightChannels; j++ {
		if len(r.Buffer) > 0 {
			r.Buffer[j] = byte(startColor.Red)
			r.Buffer[j+r.ColorOffset] = byte(startColor.Green)
			r.Buffer[j+(r.ColorOffset*2)] = byte(startColor.Blue)
		} else {
			buf[j] = []byte{
				byte(startColor.Red),
				byte(startColor.Green),
				byte(startColor.Blue),
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
