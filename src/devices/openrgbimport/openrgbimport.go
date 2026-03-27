package openrgbimport

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/openrgb"
	"OpenLinkHub/src/rgb"
	"fmt"
	"strings"
	"sync"
	"time"
)

type ZoneConfig struct {
	Name     string `json:"name"`
	LedCount int    `json:"ledCount"`
}

type DeviceConfig struct {
	Serial string       `json:"serial"`
	Zones  []ZoneConfig `json:"zones"`
}

type ZoneColors struct {
	Color      *rgb.Color
	ColorIndex []int
	Name       string
}

type DeviceProfile struct {
	RGBProfile       string
	BrightnessSlider *uint8
	ZoneColors       map[int]ZoneColors
}

type Device struct {
	Product       string
	Serial        string
	instance      *common.Device
	controllerId  int
	colorCount    int
	ZoneAmount    int
	Config        *DeviceConfig
	DeviceProfile *DeviceProfile

	brightness uint8
	lastColor  []byte

	effect    string
	speed     float64
	rgbRunner *rgb.ActiveRGB
	stopChan  chan struct{}
	doneChan  chan struct{}
	running   bool
	mu        sync.Mutex
}

func buildZoneColorsFromConfig(cfg *DeviceConfig, defaultColor []byte) map[int]ZoneColors {
	zoneColors := make(map[int]ZoneColors)

	red := float64(99)
	green := float64(213)
	blue := float64(255)
	if len(defaultColor) >= 3 {
		red = float64(defaultColor[0])
		green = float64(defaultColor[1])
		blue = float64(defaultColor[2])
	}

	ledOffset := 0
	for zoneIndex, zoneCfg := range cfg.Zones {
		colorIndex := make([]int, 0, zoneCfg.LedCount*3)
		for led := 0; led < zoneCfg.LedCount; led++ {
			base := (ledOffset + led) * 3
			colorIndex = append(colorIndex, base, base+1, base+2)
		}

		zoneColors[zoneIndex] = ZoneColors{
			Color: &rgb.Color{
				Red:        red,
				Green:      green,
				Blue:       blue,
				Brightness: 1,
				Hex:        fmt.Sprintf("#%02x%02x%02x", int(red), int(green), int(blue)),
			},
			ColorIndex: colorIndex,
			Name:       zoneCfg.Name,
		}

		ledOffset += zoneCfg.LedCount
	}

	return zoneColors
}

func Init() *common.Device {
	d := &Device{
		Product:    "Imported ASUS Motherboard",
		Serial:     "openrgb-mobo-1",
		colorCount: 3, // motherboard exposes 3 writable OpenRGB entries/zones
		brightness: 100,
		lastColor:  []byte{99, 213, 255}, // default #63d5ff
		effect:     "static",
		speed:      2.0,
		stopChan:   nil,
		doneChan:   nil,
		running:    false,
	}

	controllerId, err := openrgb.FindControllerIDByNameOrVendor(
		"asus rog strix z890-e gaming wifi",
		"asus aura",
	)
	if err != nil {
		fmt.Println("OpenRGB controller lookup failed:", err)
		d.controllerId = -1
	} else {
		d.controllerId = controllerId
		fmt.Println("OpenRGB controller lookup succeeded, controllerId =", d.controllerId)
	}

	d.createDevice()
	return d.instance
}

func InitAll() []*common.Device {
	discovered, err := openrgb.DiscoverControllers()
	if err != nil {
		// Preserve legacy single-motherboard behavior on discovery failure.
		return []*common.Device{Init()}
	}

	result := make([]*common.Device, 0, len(discovered))
	for _, dc := range discovered {
		d := newDeviceFromController(dc)
		if d == nil {
			continue
		}
		d.createDevice()
		result = append(result, d.instance)
	}

	if len(result) == 0 {
		// Preserve legacy behavior if filter removes everything.
		return []*common.Device{Init()}
	}

	return result
}

func newDeviceFromController(dc openrgb.DiscoveredController) *Device {
	nameLower := strings.ToLower(dc.Name)
	vendorLower := strings.ToLower(dc.Vendor)

	isLegacyASUS := strings.Contains(nameLower, "asus rog strix z890-e gaming wifi") ||
		strings.Contains(vendorLower, "asus aura")

	serial := fmt.Sprintf("openrgb-import-%d", dc.ID)
	product := dc.Name
	colorCount := dc.LEDCount

	if isLegacyASUS {
		serial = "openrgb-mobo-1"
		product = "Imported ASUS Motherboard"
		colorCount = 3 // keep legacy fallback only for ASUS motherboard path
	}

	if strings.Contains(nameLower, "strimer") && colorCount <= 0 {
		colorCount = 120
	}

	// Non-legacy imports require parsed LED count.
	if !isLegacyASUS && colorCount <= 0 {
		return nil
	}

	if product == "" {
		product = fmt.Sprintf("Imported OpenRGB Controller %d", dc.ID)
	}

	var cfg *DeviceConfig
	if strings.Contains(nameLower, "strimer") {
		cfg = &DeviceConfig{
			Serial: serial,
			Zones: []ZoneConfig{
				{Name: "24 Pin ATX Strip 0", LedCount: 20},
				{Name: "24 Pin ATX Strip 1", LedCount: 20},
				{Name: "24 Pin ATX Strip 2", LedCount: 20},
				{Name: "24 Pin ATX Strip 3", LedCount: 20},
				{Name: "24 Pin ATX Strip 4", LedCount: 20},
				{Name: "24 Pin ATX Strip 5", LedCount: 20},
			},
		}

		colorCount = 0
		for _, zone := range cfg.Zones {
			colorCount += zone.LedCount
		}
	}

	fmt.Println("DEBUG IMPORT DEVICE:", product, "| serial:", serial, "| controllerId:", dc.ID, "| colorCount:", colorCount)
	d := &Device{
		Product:      product,
		Serial:       serial,
		controllerId: dc.ID,
		colorCount:   colorCount,
		brightness:   100,
		lastColor:    []byte{99, 213, 255},
		effect:       "static",
		speed:        2.0,
		stopChan:     nil,
		doneChan:     nil,
		running:      false,
	}

	if cfg != nil {
		defaultBrightness := uint8(100)
		d.Config = cfg
		d.ZoneAmount = len(cfg.Zones)
		d.DeviceProfile = &DeviceProfile{
			RGBProfile:       "static",
			BrightnessSlider: &defaultBrightness,
			ZoneColors:       buildZoneColorsFromConfig(cfg, d.lastColor),
		}
	}

	return d
}

func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeMotherboard,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    "",
		Image:       "icon-motherboard.svg",
		Instance:    d,
		GetDevice:   d, // keep existing motherboard template behavior
	}
}

func (d *Device) GetDeviceTemplate() string {
	return "motherboard.html"
}

func (d *Device) ControllerID() int {
	return d.controllerId
}

func (d *Device) applyBrightness(rgbBytes []byte) []byte {
	if len(rgbBytes) < 3 {
		return []byte{0, 0, 0}
	}

	b := int(d.brightness)
	return []byte{
		byte((int(rgbBytes[0]) * b) / 100),
		byte((int(rgbBytes[1]) * b) / 100),
		byte((int(rgbBytes[2]) * b) / 100),
	}
}

func (d *Device) stopEffectLoopLocked() {
	if d.running && d.stopChan != nil {
		stop := d.stopChan
		done := d.doneChan
		d.stopChan = nil
		d.doneChan = nil
		d.running = false

		close(stop)

		d.mu.Unlock()
		if done != nil {
			<-done
		}
		d.mu.Lock()
	}
}

func (d *Device) buildZoneFrame() []byte {
	buf := make([]byte, d.colorCount*3)
	if d.DeviceProfile == nil {
		return buf
	}

	for zoneIndex := 0; zoneIndex < d.ZoneAmount; zoneIndex++ {
		zone, ok := d.DeviceProfile.ZoneColors[zoneIndex]
		if !ok || zone.Color == nil {
			continue
		}

		color := zone.Color
		scaled := d.applyBrightness([]byte{
			byte(color.Red),
			byte(color.Green),
			byte(color.Blue),
		})

		for i, idx := range zone.ColorIndex {
			if idx < 0 || idx >= len(buf) {
				continue
			}

			switch i % 3 {
			case 0:
				buf[idx] = scaled[0]
			case 1:
				buf[idx] = scaled[1]
			case 2:
				buf[idx] = scaled[2]
			}
		}
	}

	return buf
}

func (d *Device) SetColor(rgbBytes []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.controllerId < 0 {
		return fmt.Errorf("controllerId not set")
	}

	if len(rgbBytes) < 3 {
		return fmt.Errorf("invalid rgb value")
	}

	d.lastColor = []byte{rgbBytes[0], rgbBytes[1], rgbBytes[2]}

	// Static color should stop animation
	d.stopEffectLoopLocked()
	d.effect = "static"

	if d.Config != nil && d.ZoneAmount > 0 {
		if d.DeviceProfile != nil {
			for zoneIndex := 0; zoneIndex < d.ZoneAmount; zoneIndex++ {
				zoneColor, ok := d.DeviceProfile.ZoneColors[zoneIndex]
				if !ok || zoneColor.Color == nil {
					continue
				}

				zoneColor.Color.Red = float64(rgbBytes[0])
				zoneColor.Color.Green = float64(rgbBytes[1])
				zoneColor.Color.Blue = float64(rgbBytes[2])
				zoneColor.Color.Hex = fmt.Sprintf("#%02x%02x%02x", int(rgbBytes[0]), int(rgbBytes[1]), int(rgbBytes[2]))
				d.DeviceProfile.ZoneColors[zoneIndex] = zoneColor
			}
		}

		time.Sleep(75 * time.Millisecond)
		return openrgb.SendFrame(uint32(d.controllerId), d.buildZoneFrame())
	}

	scaled := d.applyBrightness(d.lastColor)
	return openrgb.SendColor(uint32(d.controllerId), d.colorCount, scaled)
}

func (d *Device) SetBrightness(brightness uint8) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if brightness > 100 {
		brightness = 100
	}

	d.brightness = brightness

	// If an effect is running, let the effect loop pick up the new brightness.
	if d.running {
		return nil
	}

	if d.Config != nil && d.ZoneAmount > 0 {
		return openrgb.SendFrame(uint32(d.controllerId), d.buildZoneFrame())
	}

	scaled := d.applyBrightness(d.lastColor)

	if d.controllerId < 0 {
		return fmt.Errorf("controllerId not set")
	}

	return openrgb.SendColor(uint32(d.controllerId), d.colorCount, scaled)
}

func (d *Device) SetSpeed(speed string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch speed {
	case "slow":
		d.speed = 4.0
	case "fast":
		d.speed = 0.8
	default:
		d.speed = 2.0
	}
}

func (d *Device) SetEffect(effect string) error {
	d.mu.Lock()

	if d.controllerId < 0 {
		d.mu.Unlock()
		return fmt.Errorf("controllerId not set")
	}

	// stop previous loop if any
	d.stopEffectLoopLocked()

	d.effect = effect

	// Static just reapplies current color once
	if effect == "static" {
		if d.Config != nil && d.ZoneAmount > 0 {
			time.Sleep(75 * time.Millisecond)
			frame := d.buildZoneFrame()
			d.mu.Unlock()
			return openrgb.SendFrame(uint32(d.controllerId), frame)
		}

		scaled := d.applyBrightness(d.lastColor)
		d.mu.Unlock()
		return openrgb.SendColor(uint32(d.controllerId), d.colorCount, scaled)
	}

	// For now only colorshift is implemented as a real OLH effect
	if effect != "colorshift" {
		if d.Config != nil && d.ZoneAmount > 0 {
			time.Sleep(75 * time.Millisecond)
			frame := d.buildZoneFrame()
			d.mu.Unlock()
			return openrgb.SendFrame(uint32(d.controllerId), frame)
		}

		scaled := d.applyBrightness(d.lastColor)
		d.mu.Unlock()
		return openrgb.SendColor(uint32(d.controllerId), d.colorCount, scaled)
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	d.stopChan = stop
	d.doneChan = done
	d.running = true

	startColor := &rgb.Color{
		Red:        float64(d.lastColor[0]),
		Green:      float64(d.lastColor[1]),
		Blue:       float64(d.lastColor[2]),
		Brightness: rgb.GetBrightnessValueFloat(d.brightness),
	}

	endColor := &rgb.Color{
		Red:        255,
		Green:      0,
		Blue:       255,
		Brightness: rgb.GetBrightnessValueFloat(d.brightness),
	}

	runner := rgb.New(
		d.colorCount,
		d.speed,
		startColor,
		endColor,
		rgb.GetBrightnessValueFloat(d.brightness),
		0,
		0,
		true,
	)
	d.rgbRunner = runner

	controllerId := d.controllerId
	d.mu.Unlock()

	go func() {
		defer close(done)

		startTime := time.Now()
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				d.mu.Lock()

				// refresh dynamic values so live changes apply
				runner.RgbModeSpeed = d.speed
				runner.RGBBrightness = rgb.GetBrightnessValueFloat(d.brightness)
				runner.RGBStartColor = &rgb.Color{
					Red:        float64(d.lastColor[0]),
					Green:      float64(d.lastColor[1]),
					Blue:       float64(d.lastColor[2]),
					Brightness: rgb.GetBrightnessValueFloat(d.brightness),
				}
				runner.RGBEndColor = &rgb.Color{
					Red:        255,
					Green:      0,
					Blue:       255,
					Brightness: rgb.GetBrightnessValueFloat(d.brightness),
				}

				runner.Colorshift(&startTime, runner)
				frame := make([]byte, len(runner.Output))
				copy(frame, runner.Output)
				d.mu.Unlock()

				select {
				case <-stop:
					return
				default:
				}

				_ = openrgb.SendFrame(uint32(controllerId), frame)
			}
		}
	}()

	return nil
}

func (d *Device) SetRed() error {
	return d.SetColor([]byte{255, 0, 0})
}
