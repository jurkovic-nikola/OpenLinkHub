package rgb

import (
	"OpenLinkHub/src/common"
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"sort"
	"time"
)

type HSL struct {
	H, S, L float64
}

type Color struct {
	Red        float64 `json:"red"`
	Green      float64 `json:"green"`
	Blue       float64 `json:"blue"`
	Brightness float64 `json:"brightness"`
	Hex        string
}

type RGB struct {
	Device       string             `json:"device"`
	DefaultColor Color              `json:"defaultColor"`
	Profiles     map[string]Profile `json:"profiles"`
}

type Profile struct {
	Speed       float64 `json:"speed"`
	Brightness  float64 `json:"brightness"`
	Smoothness  int     `json:"smoothness"`
	StartColor  Color   `json:"start"`
	MiddleColor Color   `json:"middle,omitempty"`
	EndColor    Color   `json:"end"`
	MinTemp     float64 `json:"minTemp"`
	MaxTemp     float64 `json:"maxTemp"`
}

type ActiveRGB struct {
	LightChannels          int
	Smoothness             int
	RgbModeSpeed           float64
	RGBEndColor            *Color
	RGBStartColor          *Color
	RGBBrightness          float64
	RgbLoopDuration        time.Duration
	RGBCustomColor         bool
	lightChannelsPerDevice map[int][]int
	Exit                   chan bool
	Output                 []byte
	Tracking               []int
	Phase                  int
	TempColor              *Color
	HasLCD                 bool
	IsAIO                  bool
	MinTemp                float64
	MaxTemp                float64
}

var (
	rgb        RGB
	profileOff = Profile{
		Speed:       0,
		Brightness:  0,
		Smoothness:  0,
		StartColor:  Color{Red: 0, Green: 0, Blue: 0, Brightness: 0},
		MiddleColor: Color{Red: 0, Green: 0, Blue: 0, Brightness: 0},
		EndColor:    Color{Red: 0, Green: 0, Blue: 0, Brightness: 0},
	}
)

// GetRGB will return RGB
func GetRGB() RGB {
	return rgb
}

// Init will initialize RGB configuration
func Init() {
	pwd, _ := os.Getwd()
	cfg := pwd + "/database/rgb.json"
	f, err := os.Open(cfg)
	if err != nil {
		panic(err.Error())
	}
	if err = json.NewDecoder(f).Decode(&rgb); err != nil {
		panic(err.Error())
	}

	// Off profile to disable RGB
	rgb.Profiles["off"] = profileOff
}

// GetRgbProfile will return Profile struct
func GetRgbProfile(profile string) *Profile {
	if val, ok := rgb.Profiles[profile]; ok {
		return &val
	}
	return nil
}

// GetRgbProfiles will return all RGB profiles
func GetRgbProfiles() map[string]Profile {
	return rgb.Profiles
}

// interpolateColor performs linear interpolation between two colors
func interpolateColor(c1, c2 *Color, t float64) *Color {
	return &Color{
		Red:   common.Lerp(c1.Red, c2.Red, t),
		Green: common.Lerp(c1.Green, c2.Green, t),
		Blue:  common.Lerp(c1.Blue, c2.Blue, t),
	}
}

// New will create new ActiveRGB struct for RGB control
func New(
	lightChannels int,
	rgbModeSpeed float64,
	RGBStartColor *Color,
	RGBEndColor *Color,
	RGBBrightness float64,
	smoothness int,
	rgbLoopDuration time.Duration,
	RGBCustomColor bool,
) *ActiveRGB {
	return &ActiveRGB{
		LightChannels:   lightChannels,
		Smoothness:      smoothness,
		RgbModeSpeed:    rgbModeSpeed,
		RGBStartColor:   RGBStartColor,
		RGBEndColor:     RGBEndColor,
		RGBBrightness:   RGBBrightness,
		RgbLoopDuration: rgbLoopDuration,
		RGBCustomColor:  RGBCustomColor,
		Exit:            make(chan bool),
	}
}

func Exit() *ActiveRGB {
	return &ActiveRGB{
		Exit: make(chan bool),
	}
}

// Stop will send command to exit RGB for {} loop
func (r *ActiveRGB) Stop() {
	r.Exit <- true
	close(r.Exit)
}

func toHSL(c Color) HSL {
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

func toRGB(c HSL) *Color {
	h := c.H
	s := c.S
	l := c.L

	if s == 0 {
		return &Color{
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

	return &Color{
		Red:   math.Round(r),
		Green: math.Round(g),
		Blue:  math.Round(b),
	}
}

// GenerateRandomColor will generate random color with provided bts as brightness
func GenerateRandomColor(bts float64) *Color {
	r := rand.Intn(256) // Random value between 0 and 255
	g := rand.Intn(256) // Random value between 0 and 255
	b := rand.Intn(256) // Random value between 0 and 255

	color := &Color{
		Red:        float64(r),
		Green:      float64(g),
		Blue:       float64(b),
		Brightness: bts,
	}
	return ModifyBrightness(*color)
}

// ModifyBrightness will modify color brightness
func ModifyBrightness(c Color) *Color {
	if c.Brightness > 1 {
		c.Brightness = 1
	} else if c.Brightness < 0 {
		c.Brightness = 0
	}
	hsl := toHSL(c)
	hsl.L *= c.Brightness
	return toRGB(hsl)
}

// SetColor will generate byte output for RGB data
func SetColor(data map[int][]byte) []byte {
	buffer := make([]byte, len(data)*3)
	i := 0

	// We need to sort array due to the nature of RGB.
	// R G B needs to be applied in the same way it was created.
	keys := make([]int, 0)
	for k := range data {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		buffer[i] = data[k][0]   // r
		buffer[i+1] = data[k][1] // g
		buffer[i+2] = data[k][2] // b
		i += 3                   // Move to the next place
	}

	return buffer
}

// GetBrightnessValue will return brightness value in float64 based on mode
func GetBrightnessValue(mode uint8) float64 {
	switch mode {
	case 1:
		return 0.3
	case 2:
		return 0.6
	case 3:
		return 1
	case 4:
		return 0
	}
	return 0
}
