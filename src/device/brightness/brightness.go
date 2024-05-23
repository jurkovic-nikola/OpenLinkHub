package brightness

import (
	"OpenICUELinkHub/src/structs"
	"math"
)

type RGB struct {
	R, G, B float64
}

func ToHSL(c structs.Color) HSL {
	var h, s, l float64

	r := c.Red
	g := c.Green
	b := c.Blue

	maxValue := math.Max(math.Max(r, g), b)
	minValue := math.Min(math.Min(r, g), b)

	// Luminosity is the average of the maxValue and minValue rgb color intensities.
	l = (maxValue + minValue) / 2

	// saturation
	delta := maxValue - minValue
	if delta == 0 {
		// it's gray
		return HSL{0, 0, l}
	}

	// it's not gray
	if l < 0.5 {
		s = delta / (maxValue + minValue)
	} else {
		s = delta / (2 - maxValue - minValue)
	}

	// hue
	r2 := (((maxValue - r) / 6) + (delta / 2)) / delta
	g2 := (((maxValue - g) / 6) + (delta / 2)) / delta
	b2 := (((maxValue - b) / 6) + (delta / 2)) / delta
	switch {
	case r == maxValue:
		h = b2 - g2
	case g == maxValue:
		h = (1.0 / 3.0) + r2 - b2
	case b == maxValue:
		h = (2.0 / 3.0) + g2 - r2
	}

	// fix wraparounds
	switch {
	case h < 0:
		h += 1
	case h > 1:
		h -= 1
	}

	return HSL{h, s, l}
}

type HSL struct {
	H, S, L float64
}

func hueToRGB(v1, v2, h float64) float64 {
	if h < 0 {
		h += 1
	}
	if h > 1 {
		h -= 1
	}
	switch {
	case 6*h < 1:
		return v1 + (v2-v1)*6*h
	case 2*h < 1:
		return v2
	case 3*h < 2:
		return v1 + (v2-v1)*((2.0/3.0)-h)*6
	}
	return v1
}

func ToRGB(c HSL) *structs.Color {
	h := c.H
	s := c.S
	l := c.L

	if s == 0 {
		return &structs.Color{
			Red:   math.Round(l),
			Green: math.Round(l),
			Blue:  math.Round(l),
		}
	}

	var v1, v2 float64
	if l < 0.5 {
		v2 = l * (1 + s)
	} else {
		v2 = (l + s) - (s * l)
	}

	v1 = 2*l - v2

	r := hueToRGB(v1, v2, h+(1.0/3.0))
	g := hueToRGB(v1, v2, h)
	b := hueToRGB(v1, v2, h-(1.0/3.0))

	return &structs.Color{
		Red:   math.Round(r),
		Green: math.Round(g),
		Blue:  math.Round(b),
	}
}

func ModifyBrightness(c structs.Color) *structs.Color {
	if c.Brightness > 1 {
		c.Brightness = 1
	} else if c.Brightness < 0 {
		c.Brightness = 0
	}
	hsl := ToHSL(c)
	hsl.L *= c.Brightness
	return ToRGB(hsl)
}
