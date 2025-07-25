package rgb

import (
	"math"
	"time"
)

// Colorwarp will run RGB function
func (r *ActiveRGB) Colorwarp(startTime *time.Time, activeRgb *ActiveRGB) {
	buf := map[int][]byte{}

	elapsed := time.Since(*startTime).Milliseconds()

	if r.RgbModeSpeed == 0 {
		r.RgbModeSpeed = 1.0
	}

	totalProgress := float64(elapsed) / (r.RgbModeSpeed * 1000)
	progress := math.Mod(totalProgress, 1.0)
	currentCycle := int(totalProgress)

	if activeRgb.LastCycle[r.ChannelId].RGBStartColor == nil {
		activeRgb.LastCycle[r.ChannelId].RGBStartColor = GenerateRandomColor(r.RGBBrightness)
	}

	if activeRgb.LastCycle[r.ChannelId].RGBEndColor == nil {
		activeRgb.LastCycle[r.ChannelId].RGBEndColor = GenerateRandomColor(r.RGBBrightness)
	}

	if currentCycle != activeRgb.LastCycle[r.ChannelId].LastCycle {
		activeRgb.LastCycle[r.ChannelId].LastCycle = currentCycle
		activeRgb.LastCycle[r.ChannelId].RGBStartColor = activeRgb.LastCycle[r.ChannelId].RGBEndColor
		activeRgb.LastCycle[r.ChannelId].RGBEndColor = GenerateRandomColor(r.RGBBrightness)
	}
	
	color := interpolateColor(
		activeRgb.LastCycle[r.ChannelId].RGBStartColor,
		activeRgb.LastCycle[r.ChannelId].RGBEndColor,
		progress,
		r.RGBBrightness,
	)

	// Update LED channels
	for j := 0; j < r.LightChannels; j++ {
		if len(r.Buffer) > 0 {
			r.Buffer[j] = byte(color.Red)
			r.Buffer[j+r.ColorOffset] = byte(color.Green)
			r.Buffer[j+(r.ColorOffset*2)] = byte(color.Blue)
		} else {
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
	}
	// Raw colors
	r.Raw = buf

	if r.Inverted {
		r.Output = SetColorInverted(buf)
	} else {
		r.Output = SetColor(buf)
	}
}

/*
// Colorwarp will run RGB function
func (r *ActiveRGB) Colorwarp2(startTime *time.Time, activeRgb *ActiveRGB) {
	buf := map[int][]byte{}

	elapsed := time.Since(*startTime).Milliseconds()

	if r.RgbModeSpeed == 0 {
		r.RgbModeSpeed = 1.0
	}

	totalProgress := float64(elapsed) / (r.RgbModeSpeed * 1000)
	progress := math.Mod(totalProgress, 1.0)
	currentCycle := int(totalProgress)

	if currentCycle != activeRgb.Tracking[r.ChannelId].LastCycle {
		activeRgb.Tracking[r.ChannelId].LastCycle = currentCycle
		activeRgb.Tracking[r.ChannelId].RGBStartColor = activeRgb.Tracking[r.ChannelId].RGBEndColor
		activeRgb.Tracking[r.ChannelId].RGBEndColor = GenerateRandomColor(r.RGBBrightness)
	}

	color := interpolateColor(activeRgb.Tracking[r.ChannelId].RGBStartColor, activeRgb.Tracking[r.ChannelId].RGBEndColor, progress, r.RGBBrightness)

	// Update LED channels
	for j := 0; j < r.LightChannels; j++ {
		if len(r.Buffer) > 0 {
			r.Buffer[j] = byte(color.Red)
			r.Buffer[j+r.ColorOffset] = byte(color.Green)
			r.Buffer[j+(r.ColorOffset*2)] = byte(color.Blue)
		} else {
			buf[j] = []byte{
				byte(color.Red),
				byte(color.Green),
				byte(color.Blue),
			}
			if r.IsAIO && r.HasLCD && j > 15 && j < 20 {
				buf[j] = []byte{0, 0, 0}
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
*/
