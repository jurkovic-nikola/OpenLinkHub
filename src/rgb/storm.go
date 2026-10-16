package rgb

import (
	"math/rand"
)

const (
	stormChance = 0.0005
	stormDecay  = 0.85
)

func (r *ActiveRGB) stormColorEffect(st *EffectState, i int, c1, c2 *Color, bts float64) (uint8, uint8, uint8) {
	st.StormFlash[i] *= stormDecay
	if rand.Float32() < stormChance {
		st.StormFlash[i] = 1.0
	}
	a := st.StormFlash[i]
	if a < 0.02 {
		a = 0
	}

	base := ModifyBrightness(Color{Red: c1.Red, Green: c1.Green, Blue: c1.Blue, Brightness: bts})
	flash := ModifyBrightness(Color{Red: c2.Red, Green: c2.Green, Blue: c2.Blue, Brightness: bts})

	mix := func(x, y float64) uint8 { return uint8(x*(1-a) + y*a) }
	return mix(float64(base.Red), float64(flash.Red)),
		mix(float64(base.Green), float64(flash.Green)),
		mix(float64(base.Blue), float64(flash.Blue))
}

// Storm will run RGB function
func (r *ActiveRGB) Storm() {
	buf := map[int][]byte{}

	st := r.effectState()
	if len(st.StormFlash) != r.LightChannels {
		st.StormFlash = make([]float64, r.LightChannels)
	}

	for i := 0; i < r.LightChannels; i++ {
		red, green, blue := r.stormColorEffect(st, i, r.RGBStartColor, r.RGBEndColor, r.RGBStartColor.Brightness)
		if len(r.Buffer) > 0 {
			r.Buffer[i] = red
			r.Buffer[i+r.ColorOffset] = green
			r.Buffer[i+(r.ColorOffset*2)] = blue
		} else {
			buf[i] = []byte{red, green, blue}
			if r.IsAIO && r.HasLCD {
				if i > 15 && i < 20 {
					buf[i] = []byte{0, 0, 0}
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
