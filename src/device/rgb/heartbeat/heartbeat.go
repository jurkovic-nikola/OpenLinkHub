package heartbeat

import (
	"OpenICUELinkHub/src/device/brightness"
	"OpenICUELinkHub/src/device/comm"
	"OpenICUELinkHub/src/device/common"
	"OpenICUELinkHub/src/device/opcodes"
	"OpenICUELinkHub/src/structs"
	"sort"
	"time"
)

var exit = make(chan bool)

// generateColors will generate color based on start and end color
func generateColors(lightChannels int, c1 *structs.Color, bts float64) []struct{ R, G, B float64 } {
	colors := make([]struct{ R, G, B float64 }, lightChannels)
	for i := 0; i < lightChannels; i++ {
		c1.Brightness = bts
		modify := brightness.ModifyBrightness(*c1)
		colors[i] = struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
	}
	return colors
}

// generateColor will generate color based on start and end color
func generateColor(c1 *structs.Color, bts float64) struct{ R, G, B float64 } {
	c1.Brightness = bts
	modify := brightness.ModifyBrightness(*c1)
	return struct{ R, G, B float64 }{modify.Red, modify.Green, modify.Blue}
}

func Stop() {
	exit <- true
}

// Init will run RGB function
func Init(
	lightChannels,
	smoothness int,
	rgbCustomColor bool,
	rgbStartColor,
	rgbEndColor *structs.Color,
	rgbLoopDuration time.Duration,
	lightChannelsPerDevice map[int][]int,
) {
	phase := 0                                 // Phase rotation
	c1 := &structs.Color{}                     // Color init
	buf := make(map[int][]byte, lightChannels) // Buffer

	for {
		select {
		case <-exit:
			return
		default:
			var tracking []int
			// Phase color rotation
			if phase == 0 {
				if !rgbCustomColor {
					// Random color
					c1 = common.GenerateRandomColor(1)
				} else {
					// Rotate color 1 and 2
					if c1 == rgbStartColor {
						c1 = rgbEndColor
					} else {
						c1 = rgbStartColor
					}
				}
			}

			// Inner LEDs
			for i := 0; i <= smoothness; i++ {
				m := 0
				var t float64

				if phase == 0 {
					t = float64(i) / float64(smoothness)
				} else {
					t = float64(phase) - float64(i)/float64(smoothness)
				}

				keys := make([]int, 0)
				for k := range lightChannelsPerDevice {
					keys = append(keys, k)
				}
				sort.Ints(keys)

				for _, k := range keys {
					outer := lightChannelsPerDevice[k][0] + lightChannelsPerDevice[k][1]
					inner := lightChannelsPerDevice[k][2] + lightChannelsPerDevice[k][3]

					colors := generateColors(outer, c1, t)
					for _, color := range colors {
						buf[m] = []byte{
							byte(color.R),
							byte(color.G),
							byte(color.B),
						}
						m++
					}

					// Disable inner leds if any
					if inner > 0 {
						for p := 0; p < inner; p++ {
							// Keep track of inner leds
							tracking = append(tracking, m)

							if phase == 0 {
								// Turn off inner leds only in the initial phase.
								// Don't modify color in the second phase
								buf[m] = []byte{0, 0, 0}
							}
							m++
						}
					}
				}

				data := common.SetColor(buf)
				comm.WriteColor(opcodes.DataTypeSetColor, data)
				time.Sleep(40 * time.Millisecond)
			}

			// Inner LEDs
			if len(tracking) > 0 {
				for i := 0; i <= smoothness; i++ {
					var t float64

					if phase == 0 {
						t = float64(i) / float64(smoothness)
					} else {
						t = float64(phase) - float64(i)/float64(smoothness)
					}
					color := generateColor(c1, t)
					for k := range tracking {
						buf[tracking[k]] = []byte{
							byte(color.R),
							byte(color.G),
							byte(color.B),
						}
					}
					data := common.SetColor(buf)
					comm.WriteColor(opcodes.DataTypeSetColor, data)
					time.Sleep(40 * time.Millisecond) // Adjust sleep time for smoother animation
				}
			}
			if phase == 0 {
				phase = 1
			} else {
				phase = 0
			}
			time.Sleep(rgbLoopDuration) // Loop duration
		}
	}
}
