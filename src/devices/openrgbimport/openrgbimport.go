package openrgbimport

import (
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/openrgb"
	"OpenLinkHub/src/rgb"
	"fmt"
	"sync"
	"time"
)

type Device struct {
	Product      string
	Serial       string
	instance     *common.Device
	controllerId int
	colorCount   int

	brightness uint8
	lastColor  []byte

	effect    string
	speed     float64
	rgbRunner *rgb.ActiveRGB
	stopChan  chan struct{}
	running   bool
	mu        sync.Mutex
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

func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeMotherboard,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    "",
		Image:       "icon-motherboard.svg",
		Instance:    d,
		GetDevice:   d,
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
		close(d.stopChan)
		d.stopChan = nil
		d.running = false
	}
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
		scaled := d.applyBrightness(d.lastColor)
		d.mu.Unlock()
		return openrgb.SendColor(uint32(d.controllerId), d.colorCount, scaled)
	}

	// For now only colorshift is implemented as a real OLH effect
	if effect != "colorshift" {
		scaled := d.applyBrightness(d.lastColor)
		d.mu.Unlock()
		return openrgb.SendColor(uint32(d.controllerId), d.colorCount, scaled)
	}

	stop := make(chan struct{})
	d.stopChan = stop
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
		startTime := time.Now()
		ticker := time.NewTicker(33 * time.Millisecond)
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

					_ = openrgb.SendFrame(uint32(controllerId), frame)
			}
		}
	}()

	return nil
}

func (d *Device) SetRed() error {
	return d.SetColor([]byte{255, 0, 0})
}
