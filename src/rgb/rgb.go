package rgb

import (
	"OpenLinkHub/src/common"
	"encoding/json"
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
	Speed           float64       `json:"speed"`
	Brightness      float64       `json:"brightness"`
	Smoothness      int           `json:"smoothness"`
	StartColor      Color         `json:"start"`
	MiddleColor     Color         `json:"middle,omitempty"`
	EndColor        Color         `json:"end"`
	Gradients       map[int]Color `json:"gradients"`
	MinTemp         float64       `json:"minTemp"`
	MaxTemp         float64       `json:"maxTemp"`
	ProfileName     string        `json:"profileName"`
	AlternateColors bool          `json:"alternateColors"`
	RgbDirection    byte          `json:"rgbDirection"`
	PerLed          bool          `json:"perLed"`
}

type LastCycle struct {
	RGBStartColor *Color
	RGBEndColor   *Color
	LastCycle     int
}

type ActiveRGB struct {
	LightChannels          int
	Smoothness             int
	RgbModeSpeed           float64
	RGBEndColor            *Color
	RGBStartColor          *Color
	PreviousColor          *Color
	Gradients              []Color
	GradientList           map[int]Color
	RGBBrightness          float64
	RgbLoopDuration        time.Duration
	RGBCustomColor         bool
	lightChannelsPerDevice map[int][]int
	Exit                   chan bool
	Output                 []byte
	Raw                    map[int][]byte
	Phase                  int
	TempColor              *Color
	HasLCD                 bool
	IsAIO                  bool
	MinTemp                float64
	MaxTemp                float64
	Inverted               bool
	Buffer                 []byte
	ColorOffset            int
	PreviousTemp           float64
	TempAlpha              float64
	ChannelId              int
	LastCycle              map[int]*LastCycle
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
	globalRand = rand.New(rand.NewSource(time.Now().UnixNano()))
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
func interpolateColor(c1, c2 *Color, t, bts float64) *Color {
	color := Color{
		Red:   common.Lerp(c1.Red, c2.Red, t),
		Green: common.Lerp(c1.Green, c2.Green, t),
		Blue:  common.Lerp(c1.Blue, c2.Blue, t),
	}
	color.Brightness = bts
	modify := ModifyBrightness(color)
	return modify
}

// interpolateColor blends two colors based on a weight factor `t` (0.0 to 1.0)
func interpolateColors(c1, c2 *Color, t, bts float64) Color {
	r := uint8(c1.Red*(1-t) + c2.Red*t)
	g := uint8(c1.Green*(1-t) + c2.Green*t)
	b := uint8(c1.Blue*(1-t) + c2.Blue*t)

	color := Color{Red: float64(r), Green: float64(g), Blue: float64(b)}
	color.Brightness = bts
	modify := ModifyBrightness(color)
	return *modify
}

func cloneColor(c *Color) *Color {
	return &Color{
		Red:        c.Red,
		Green:      c.Green,
		Blue:       c.Blue,
		Brightness: c.Brightness,
	}
}

// Interpolate function to calculate the intermediate color
func interpolate(
	r1,
	g1,
	b1,
	r2,
	g2,
	b2 float64,
	fraction float64,
) (int, int, int) {
	r := r1 + fraction*(r2-r1)
	g := g1 + fraction*(g2-g1)
	b := b1 + fraction*(b2-b1)
	return int(r * 255), int(g * 255), int(b * 255)
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
	lastCycle := map[int]*LastCycle{}
	for i := 0; i < 64; i++ {
		lastCycle[i] = &LastCycle{}
	}
	return &ActiveRGB{
		Exit:      make(chan bool),
		LastCycle: lastCycle,
	}
}

// Stop will send command to exit RGB for {} loop
func (r *ActiveRGB) Stop() {
	r.Exit <- true
	close(r.Exit)
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

// GenerateRandomColorSeeded will generate a random color based on a seed
func GenerateRandomColorSeeded(seed int64, bts float64) *Color {
	rnd := rand.New(rand.NewSource(seed)) // Create a deterministic RNG with seed
	r := rnd.Intn(256)
	g := rnd.Intn(256)
	b := rnd.Intn(256)

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
	/*
		if c.Brightness > 1 {
			c.Brightness = 1
		} else if c.Brightness < 0 {
			c.Brightness = 0
		}

		hsl := toHSL(c)
		hsl.L *= c.Brightness
		return toRGB(hsl)
	*/
	if c.Brightness > 1 {
		c.Brightness = 1
	} else if c.Brightness < 0 {
		c.Brightness = 0
	}

	// Apply the brightness factor to each color component.
	newR := uint8(c.Red * c.Brightness)
	newG := uint8(c.Green * c.Brightness)
	newB := uint8(c.Blue * c.Brightness)

	return &Color{Red: float64(newR), Green: float64(newG), Blue: float64(newB)}
}

// ModifyBrightnessSlice will modify color brightness with given byte array
func ModifyBrightnessSlice(data []byte, factor float64) {
	for i := 0; i < len(data); i++ {
		v := int(float64(data[i]) * factor)
		if v > 255 {
			v = 255
		}
		data[i] = byte(v)
	}
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

// SetColorInverted will generate byte output for RGB data in inverted state
func SetColorInverted(data map[int][]byte) []byte {
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
		buffer[i] = data[k][2]   // r
		buffer[i+1] = data[k][1] // g
		buffer[i+2] = data[k][0] // b
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

// GetBrightnessValueFloat will return brightness value in float64
func GetBrightnessValueFloat(mode uint8) float64 {
	return float64(mode) / 100
}
