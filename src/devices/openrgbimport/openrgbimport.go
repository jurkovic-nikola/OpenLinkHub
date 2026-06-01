package openrgbimport

import (
	"OpenLinkHub/src/cluster"
	"OpenLinkHub/src/common"
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/openrgb"
	"OpenLinkHub/src/rgb"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
)

type ConfigStore struct {
	Devices map[string]DeviceConfig `json:"devices"`
}

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
	Active           bool               `json:"Active"`
	Path             string             `json:"Path"`
	Product          string             `json:"Product"`
	Serial           string             `json:"Serial"`
	RGBProfile       string             `json:"RGBProfile"`
	BrightnessSlider *uint8             `json:"BrightnessSlider"`
	ZoneColors       map[int]ZoneColors `json:"ZoneColors"`
	RGBCluster       bool               `json:"RGBCluster"`
}

type Device struct {
	Product            string
	Serial             string
	DisplaySerial      string
	DisplaySerialLabel string
	instance           *common.Device
	controllerId       int
	colorCount         int
	ZoneAmount         int
	Config             *DeviceConfig
	DeviceProfile      *DeviceProfile
	UserProfiles       map[string]*DeviceProfile

	brightness uint8
	lastColor  []byte

	effect      string
	speed       float64
	rgbRunner   *rgb.ActiveRGB
	stopChan    chan struct{}
	doneChan    chan struct{}
	running     bool
	openrgbConn net.Conn
	mu          sync.Mutex
}

func isUsableDisplaySerial(value string) bool {
	v := sanitizeDisplaySerial(value)
	if v == "" {
		return false
	}

	lower := strings.ToLower(v)
	switch lower {
	case "dir", "dire", "off", "on", "none", "n/a", "na", "unknown", "default":
		return false
	}

	if strings.HasPrefix(lower, "hid:") || strings.Contains(lower, "/dev/hidraw") {
		return false
	}

	hasAlphaNum := false
	for _, r := range v {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			hasAlphaNum = true
			continue
		}
		switch r {
		case '-', '_', '.', ' ':
			continue
		default:
			return false
		}
	}

	if !hasAlphaNum {
		return false
	}

	return true
}

func sanitizeDisplaySerial(value string) string {
	v := strings.Map(func(r rune) rune {
		if r == '\uFFFD' {
			return -1
		}
		if unicode.IsControl(r) || !unicode.IsPrint(r) {
			return -1
		}
		return r
	}, value)

	return strings.TrimSpace(v)
}

func pickDisplaySerialAndLabel(dc openrgb.DiscoveredController) (string, string) {
	serial := sanitizeDisplaySerial(dc.Serial)
	if isUsableDisplaySerial(serial) {
		return serial, "SERIAL"
	}

	version := sanitizeDisplaySerial(dc.Version)
	if isUsableDisplaySerial(version) {
		return version, "VERSION"
	}

	return "", ""
}

func isLegacyASUSMotherboardImport(name, vendor string) bool {
	nameLower := strings.ToLower(name)
	vendorLower := strings.ToLower(vendor)
	return strings.Contains(nameLower, "asus rog strix z890-e gaming wifi") || strings.Contains(vendorLower, "asus aura")
}

func getConfigPath() string {
	return filepath.Join(config.GetConfig().ConfigPath, "database", "openrgbimport-zones.json")
}

func loadConfigStore() *ConfigStore {
	configPath := getConfigPath()

	store := &ConfigStore{
		Devices: make(map[string]DeviceConfig),
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			_ = os.MkdirAll(filepath.Dir(configPath), 0o755)
			_ = saveConfigStore(store)
			return store
		}
		return store
	}

	if len(data) == 0 {
		return store
	}

	if err = json.Unmarshal(data, store); err != nil {
		return &ConfigStore{Devices: make(map[string]DeviceConfig)}
	}

	if store.Devices == nil {
		store.Devices = make(map[string]DeviceConfig)
	}

	return store
}

func saveConfigStore(store *ConfigStore) error {
	if store == nil {
		store = &ConfigStore{Devices: make(map[string]DeviceConfig)}
	}
	if store.Devices == nil {
		store.Devices = make(map[string]DeviceConfig)
	}

	configPath := getConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0o644)
}

func getDeviceConfig(serial string) *DeviceConfig {
	store := loadConfigStore()
	if cfg, ok := store.Devices[serial]; ok {
		deviceCfg := cfg
		return &deviceCfg
	}
	return nil
}

func buildDefaultDeviceConfig(serial string, dc openrgb.DiscoveredController) *DeviceConfig {
	nameLower := strings.ToLower(dc.Name)
	cfg := &DeviceConfig{Serial: serial}

	if isLegacyASUSMotherboardImport(dc.Name, dc.Vendor) {
		cfg.Zones = []ZoneConfig{
			{Name: "Aura Mainboard", LedCount: 1},
			{Name: "RGB Header 1", LedCount: 1},
			{Name: "RGB Header 2", LedCount: 1},
			{Name: "RGB Header 3", LedCount: 1},
		}
		return cfg
	}

	if strings.Contains(nameLower, "strimer") {
		cfg.Zones = []ZoneConfig{
			{Name: "24 Pin ATX Strip 0", LedCount: 20},
			{Name: "24 Pin ATX Strip 1", LedCount: 20},
			{Name: "24 Pin ATX Strip 2", LedCount: 20},
			{Name: "24 Pin ATX Strip 3", LedCount: 20},
			{Name: "24 Pin ATX Strip 4", LedCount: 20},
			{Name: "24 Pin ATX Strip 5", LedCount: 20},
			{Name: "8 Pin GPU Strip 0", LedCount: 27},
			{Name: "8 Pin GPU Strip 1", LedCount: 27},
			{Name: "8 Pin GPU Strip 2", LedCount: 27},
			{Name: "8 Pin GPU Strip 3", LedCount: 27},
			{Name: "8 Pin GPU Strip 4", LedCount: 27},
			{Name: "8 Pin GPU Strip 5", LedCount: 27},
		}
		return cfg
	}

	if len(dc.Zones) > 0 {
		cfg.Zones = make([]ZoneConfig, len(dc.Zones))
		for i, z := range dc.Zones {
			name := strings.TrimSpace(z.Name)
			if name == "" {
				name = fmt.Sprintf("Zone %d", i+1)
			}
			ledCount := z.LEDCount
			if ledCount <= 0 {
				ledCount = 1
			}
			cfg.Zones[i] = ZoneConfig{
				Name:     name,
				LedCount: ledCount,
			}
		}
		return cfg
	}

	cfg.Zones = []ZoneConfig{
		{Name: "Zone 1", LedCount: 1},
	}

	return cfg
}

func configLedCount(cfg *DeviceConfig) int {
	if cfg == nil {
		return 0
	}

	total := 0
	for _, zone := range cfg.Zones {
		if zone.LedCount > 0 {
			total += zone.LedCount
		}
	}
	return total
}

func isConfigValidForController(cfg *DeviceConfig, dc openrgb.DiscoveredController) bool {
	if cfg == nil {
		return false
	}

	if strings.TrimSpace(cfg.Serial) == "" {
		return false
	}
	if len(cfg.Zones) == 0 {
		return false
	}

	total := 0
	for i, zone := range cfg.Zones {
		if zone.LedCount <= 0 {
			return false
		}
		if strings.TrimSpace(zone.Name) == "" {
			cfg.Zones[i].Name = fmt.Sprintf("Zone %d", i+1)
		}
		total += zone.LedCount
	}

	return total > 0
}

func resolveDeviceConfig(serial string, dc openrgb.DiscoveredController) *DeviceConfig {
	cfg := getDeviceConfig(serial)
	if isConfigValidForController(cfg, dc) {
		return cfg
	}

	cfg = buildDefaultDeviceConfig(serial, dc)
	if !isConfigValidForController(cfg, dc) {
		return nil
	}

	store := loadConfigStore()
	store.Devices[serial] = *cfg
	_ = saveConfigStore(store)

	return cfg
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

func cloneDeviceConfig(cfg *DeviceConfig) *DeviceConfig {
	if cfg == nil {
		return nil
	}

	cloned := &DeviceConfig{
		Serial: cfg.Serial,
		Zones:  append([]ZoneConfig(nil), cfg.Zones...),
	}
	return cloned
}

func hasLEDCountIncrease(savedCfg *DeviceConfig, newCfg *DeviceConfig) bool {
	if newCfg == nil {
		return false
	}

	for i, zone := range newCfg.Zones {
		savedLEDCount := 0
		if savedCfg != nil && i < len(savedCfg.Zones) {
			savedLEDCount = savedCfg.Zones[i].LedCount
		}
		if zone.LedCount > savedLEDCount {
			return true
		}
	}

	return false
}

func (d *Device) applyConfigLocked(cfg *DeviceConfig, brightness uint8) {
	if cfg == nil {
		d.Config = nil
		d.colorCount = 0
		d.ZoneAmount = 0
		d.DeviceProfile = nil
		d.effect = "static"
		return
	}

	var wasCluster bool
	if d.DeviceProfile != nil {
		wasCluster = d.DeviceProfile.RGBCluster
	}

	d.Config = cloneDeviceConfig(cfg)
	d.colorCount = configLedCount(cfg)
	d.ZoneAmount = len(cfg.Zones)
	d.DeviceProfile = &DeviceProfile{
		RGBProfile:       "static",
		BrightnessSlider: &brightness,
		ZoneColors:       buildZoneColorsFromConfig(cfg, d.lastColor),
		RGBCluster:       wasCluster,
	}
	d.effect = "static"
}

func checkOpenRGBStable(attempts int, delay time.Duration) error {
	for i := 0; i < attempts; i++ {
		if i > 0 {
			time.Sleep(delay)
		}
		if err := openrgb.HealthCheck(); err != nil {
			return err
		}
	}

	return nil
}

func (d *Device) SaveDeviceConfig(cfg *DeviceConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	total := configLedCount(cfg)
	if total <= 0 {
		return fmt.Errorf("config must contain at least one LED")
	}

	cfg.Serial = d.Serial
	savedCfg := getDeviceConfig(d.Serial)
	riskyIncrease := hasLEDCountIncrease(savedCfg, cfg)

	brightness := uint8(d.brightness)
	if d.DeviceProfile != nil && d.DeviceProfile.BrightnessSlider != nil {
		brightness = *d.DeviceProfile.BrightnessSlider
	}
	previousCfg := cloneDeviceConfig(d.Config)
	previousBrightness := brightness
	if d.DeviceProfile == nil || d.DeviceProfile.BrightnessSlider == nil {
		previousBrightness = d.brightness
	}

	d.stopEffectLoopLocked()
	d.applyConfigLocked(cfg, brightness)

	if d.controllerId >= 0 {
		time.Sleep(75 * time.Millisecond)
		if err := openrgb.SendFrame(uint32(d.controllerId), d.buildZoneFrame()); err != nil {
			d.applyConfigLocked(previousCfg, previousBrightness)
			return err
		}
		if riskyIncrease {
			if err := checkOpenRGBStable(4, 500*time.Millisecond); err != nil {
				d.applyConfigLocked(previousCfg, previousBrightness)
				return fmt.Errorf("OpenRGB became unavailable after applying increased LED counts; config was not saved. Confirm zone and LED counts in OpenRGB and try again")
			}
		}
	}

	store := loadConfigStore()
	store.Devices[d.Serial] = *cloneDeviceConfig(cfg)
	if err := saveConfigStore(store); err != nil {
		return err
	}

	if d.DeviceProfile != nil {
		d.saveDeviceProfile()
		if d.DeviceProfile.RGBCluster {
			cluster.Get().RemoveDeviceControllerBySerial(d.Serial)
			d.setupClusterController()
		}
	}

	return nil
}

func Init() *common.Device {
	d := &Device{
		Product:       "Imported ASUS Motherboard",
		Serial:        "openrgb-mobo-1",
		DisplaySerial: "",
		colorCount:    4,
		brightness:    100,
		lastColor:     []byte{99, 213, 255}, // default #63d5ff
		effect:        "static",
		speed:         2.0,
		stopChan:      nil,
		doneChan:      nil,
		running:       false,
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

	cfg := resolveDeviceConfig(d.Serial, openrgb.DiscoveredController{
		Name:   d.Product,
		Vendor: "asus aura",
	})
	if cfg != nil {
		defaultBrightness := uint8(100)
		d.Config = cfg
		d.ZoneAmount = len(cfg.Zones)
		d.colorCount = configLedCount(cfg)
		d.DeviceProfile = &DeviceProfile{
			Active:           true,
			RGBProfile:       "static",
			BrightnessSlider: &defaultBrightness,
			ZoneColors:       buildZoneColorsFromConfig(cfg, d.lastColor),
		}
		d.loadDeviceProfiles()
		d.saveDeviceProfile()
		d.setupClusterController()
	}

	d.createDevice()
	return d.instance
}

func newOfflineDevice(serial string, cfg DeviceConfig) *Device {
	colorCount := configLedCount(&cfg)
	productName := "Imported OpenRGB Device"
	if serial == "openrgb-mobo-1" {
		productName = "Imported ASUS Motherboard"
	}

	d := &Device{
		Product:            productName,
		Serial:             serial,
		DisplaySerial:      "",
		DisplaySerialLabel: "",
		controllerId:       -1,
		colorCount:         colorCount,
		brightness:         100,
		lastColor:          []byte{99, 213, 255},
		effect:             "static",
		speed:              2.0,
		stopChan:           nil,
		doneChan:           nil,
		running:            false,
		Config:             &cfg,
		ZoneAmount:         len(cfg.Zones),
	}

	defaultBrightness := uint8(100)
	d.DeviceProfile = &DeviceProfile{
		Active:           true,
		RGBProfile:       "static",
		BrightnessSlider: &defaultBrightness,
		ZoneColors:       buildZoneColorsFromConfig(&cfg, d.lastColor),
	}
	d.loadDeviceProfiles()
	d.saveDeviceProfile()
	d.setupClusterController()

	return d
}

func InitAll() []*common.Device {
	discovered, err := openrgb.DiscoverControllers()
	if err != nil {
		store := loadConfigStore()
		var result []*common.Device
		for serial, cfg := range store.Devices {
			d := newOfflineDevice(serial, cfg)
			d.createDevice()
			d.instance.Unavailable = true
			result = append(result, d.instance)
		}
		if len(result) == 0 {
			dev := Init()
			dev.Unavailable = true
			result = append(result, dev)
		}
		return result
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
	displaySerial := ""
	displaySerialLabel := ""
	colorCount := dc.LEDCount

	if isLegacyASUS {
		serial = "openrgb-mobo-1"
	}

	if product == "" {
		product = fmt.Sprintf("Imported OpenRGB Controller %d", dc.ID)
	}

	displaySerial, displaySerialLabel = pickDisplaySerialAndLabel(dc)

	cfg := resolveDeviceConfig(serial, dc)

	if isConfigValidForController(cfg, dc) {
		colorCount = configLedCount(cfg)
	}

	d := &Device{
		Product:            product,
		Serial:             serial,
		DisplaySerial:      displaySerial,
		DisplaySerialLabel: displaySerialLabel,
		controllerId:       dc.ID,
		colorCount:         colorCount,
		brightness:         100,
		lastColor:          []byte{99, 213, 255},
		effect:             "static",
		speed:              2.0,
		stopChan:           nil,
		doneChan:           nil,
		running:            false,
	}

	if isConfigValidForController(cfg, dc) {
		defaultBrightness := uint8(100)
		d.Config = cfg
		d.ZoneAmount = len(cfg.Zones)
		d.DeviceProfile = &DeviceProfile{
			Active:           true,
			RGBProfile:       "static",
			BrightnessSlider: &defaultBrightness,
			ZoneColors:       buildZoneColorsFromConfig(cfg, d.lastColor),
		}
		d.loadDeviceProfiles()
		d.saveDeviceProfile()
		d.setupClusterController()
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

// applyBrightness scales every LED in the frame by the device brightness (0-100).
// It accepts any frame length that is a multiple of 3.
func (d *Device) applyBrightness(rgbBytes []byte) []byte {
	if len(rgbBytes) < 3 {
		return []byte{0, 0, 0}
	}

	b := int(d.brightness)
	out := make([]byte, len(rgbBytes))
	for i := 0; i+2 < len(rgbBytes); i += 3 {
		out[i] = byte((int(rgbBytes[i]) * b) / 100)
		out[i+1] = byte((int(rgbBytes[i+1]) * b) / 100)
		out[i+2] = byte((int(rgbBytes[i+2]) * b) / 100)
	}
	return out
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

	if d.DeviceProfile != nil && d.DeviceProfile.RGBCluster {
		return fmt.Errorf("device is controlled by RGB cluster")
	}

	if len(rgbBytes) < 3 {
		return fmt.Errorf("invalid rgb value")
	}

	d.lastColor = []byte{rgbBytes[0], rgbBytes[1], rgbBytes[2]}

	// Static color should stop animation
	d.stopEffectLoopLocked()
	d.effect = "static"
	if d.DeviceProfile != nil {
		d.DeviceProfile.RGBProfile = "static"
	}

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
			d.saveDeviceProfile()
		}

		time.Sleep(75 * time.Millisecond)
		return openrgb.SendFrame(uint32(d.controllerId), d.buildZoneFrame())
	}

	if d.DeviceProfile != nil {
		d.saveDeviceProfile()
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
	if d.DeviceProfile != nil {
		d.DeviceProfile.BrightnessSlider = &brightness
		d.saveDeviceProfile()
	}

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

	if d.DeviceProfile != nil && d.DeviceProfile.RGBCluster {
		d.mu.Unlock()
		return fmt.Errorf("device is controlled by RGB cluster")
	}

	// stop previous loop if any
	d.stopEffectLoopLocked()

	d.effect = effect
	if d.DeviceProfile != nil {
		d.DeviceProfile.RGBProfile = effect
		d.saveDeviceProfile()
	}

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

	// For now only colorshift and rainbow are implemented as real OLH effects
	if effect != "colorshift" && effect != "rainbow" {
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
		ticker := time.NewTicker(50 * time.Millisecond)
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

				if d.effect == "rainbow" {
					runner.Rainbow(startTime)
				} else {
					runner.Colorshift(&startTime, runner)
				}
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

func (d *Device) GetEffect() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.effect == "" {
		return "static"
	}
	return d.effect
}

func (d *Device) GetSpeed() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch d.speed {
	case 4.0:
		return "slow"
	case 0.8:
		return "fast"
	default:
		return "normal"
	}
}

func (d *Device) GetBrightness() uint8 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.brightness
}

func (d *Device) GetRGBCluster() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.DeviceProfile == nil {
		return false
	}
	return d.DeviceProfile.RGBCluster
}

func (d *Device) saveDeviceProfile() {
	if d.DeviceProfile == nil {
		return
	}

	profileDir := filepath.Join(config.GetConfig().ConfigPath, "database", "profiles")
	_ = os.MkdirAll(profileDir, 0o755)
	
	profilePath := d.DeviceProfile.Path
	if len(profilePath) == 0 {
		profilePath = filepath.Join(profileDir, d.Serial+".json")
		d.DeviceProfile.Path = profilePath
	}
	d.DeviceProfile.Serial = d.Serial
	d.DeviceProfile.Product = d.Product

	data, err := json.MarshalIndent(d.DeviceProfile, "", "  ")
	if err != nil {
		return
	}

	_ = os.WriteFile(profilePath, data, 0o644)
}

func (d *Device) loadDeviceProfiles() {
	profileList := make(map[string]*DeviceProfile)
	profileDir := filepath.Join(config.GetConfig().ConfigPath, "database", "profiles")
	_ = os.MkdirAll(profileDir, 0o755)

	files, err := os.ReadDir(profileDir)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "location": profileDir, "serial": d.Serial}).Warn("Unable to read profiles directory")
		return
	}

	for _, fi := range files {
		if fi.IsDir() {
			continue
		}

		profileLocation := filepath.Join(profileDir, fi.Name())

		if !common.IsValidExtension(profileLocation, ".json") {
			continue
		}

		fileName := strings.Split(fi.Name(), ".")[0]
		if m, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", fileName); !m {
			continue
		}

		fileSerial := ""
		if strings.Contains(fileName, "-") {
			fileSerial = strings.Split(fileName, "-")[0]
		} else {
			fileSerial = fileName
		}

		if fileSerial != d.Serial {
			continue
		}

		file, err := os.Open(profileLocation)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": profileLocation}).Warn("Unable to load profile")
			continue
		}

		pf := &DeviceProfile{}
		if d.Config != nil {
			pf.ZoneColors = buildZoneColorsFromConfig(d.Config, d.lastColor)
		}
		if err = json.NewDecoder(file).Decode(pf); err != nil {
			logger.Log(logger.Fields{"error": err, "serial": d.Serial, "location": profileLocation}).Warn("Unable to decode profile")
			file.Close()
			continue
		}
		file.Close()

		pf.Path = profileLocation
		pf.Serial = d.Serial
		pf.Product = d.Product

		if fileName == d.Serial {
			profileList["default"] = pf
		} else {
			parts := strings.Split(fileName, "-")
			if len(parts) > 1 {
				name := parts[1]
				profileList[name] = pf
			}
		}
		logger.Log(logger.Fields{"location": profileLocation, "serial": d.Serial}).Info("Loaded custom user profile")
	}

	d.UserProfiles = profileList
	d.getDeviceProfile()
}

func (d *Device) getDeviceProfile() {
	if len(d.UserProfiles) == 0 {
		logger.Log(logger.Fields{"serial": d.Serial}).Warn("No profile found for device. Probably initial start")
	} else {
		foundActive := false
		for _, pf := range d.UserProfiles {
			if pf.Active {
				d.DeviceProfile = pf
				if pf.BrightnessSlider != nil {
					d.brightness = *pf.BrightnessSlider
				}
				d.effect = pf.RGBProfile
				foundActive = true
				break
			}
		}
		if !foundActive {
			if pf, ok := d.UserProfiles["default"]; ok {
				pf.Active = true
				d.DeviceProfile = pf
				if pf.BrightnessSlider != nil {
					d.brightness = *pf.BrightnessSlider
				}
				d.effect = pf.RGBProfile
			}
		}
	}
}

// SaveUserProfile will generate a new user profile configuration and save it to a file
func (d *Device) SaveUserProfile(profileName string) uint8 {
	if d.DeviceProfile != nil {
		profileDir := filepath.Join(config.GetConfig().ConfigPath, "database", "profiles")
		profilePath := filepath.Join(profileDir, d.Serial + "-" + profileName + ".json")

		newProfile := &DeviceProfile{
			Active:           false,
			Path:             profilePath,
			Product:          d.Product,
			Serial:           d.Serial,
			RGBProfile:       d.DeviceProfile.RGBProfile,
			BrightnessSlider: d.DeviceProfile.BrightnessSlider,
			ZoneColors:       d.DeviceProfile.ZoneColors,
			RGBCluster:       d.DeviceProfile.RGBCluster,
		}

		buffer, err := json.MarshalIndent(newProfile, "", "  ")
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Error("Unable to convert to json format")
			return 0
		}

		// Create profile filename
		file, err := os.Create(profilePath)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": profilePath}).Error("Unable to create new device profile")
			return 0
		}
		defer file.Close()

		_, err = file.Write(buffer)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "location": profilePath}).Error("Unable to write data")
			return 0
		}

		d.loadDeviceProfiles()
		return 1
	}
	return 0
}

// SaveDeviceProfile will save the current active device profile
func (d *Device) SaveDeviceProfile(_ string, _ bool) uint8 {
	d.saveDeviceProfile()
	return 1
}

// ChangeDeviceProfile will change the active device profile
func (d *Device) ChangeDeviceProfile(profileName string) uint8 {
	d.mu.Lock()
	profile, ok := d.UserProfiles[profileName]
	if !ok {
		d.mu.Unlock()
		return 0
	}

	currentProfile := d.DeviceProfile
	if currentProfile != nil {
		currentProfile.Active = false
		d.saveDeviceProfile()
	}

	// Stop any running effect loop
	d.stopEffectLoopLocked()

	newProfile := profile
	newProfile.Active = true
	d.DeviceProfile = newProfile
	d.saveDeviceProfile()

	if newProfile.BrightnessSlider != nil {
		d.brightness = *newProfile.BrightnessSlider
	}
	d.effect = newProfile.RGBProfile
	d.mu.Unlock()

	// Handle cluster registration changes if needed
	if newProfile.RGBCluster {
		cluster.Get().RemoveDeviceControllerBySerial(d.Serial)
		d.setupClusterController()
	} else {
		cluster.Get().RemoveDeviceControllerBySerial(d.Serial)
		_ = d.SetEffect(d.effect)
	}

	return 1
}

// DeleteDeviceProfile deletes a device profile and its JSON file
func (d *Device) DeleteDeviceProfile(profileName string) uint8 {
	d.mu.Lock()
	defer d.mu.Unlock()

	profile, ok := d.UserProfiles[profileName]
	if !ok {
		return 0
	}

	if !common.IsValidExtension(profile.Path, ".json") {
		return 0
	}

	if profile.Active {
		return 2
	}

	if err := os.Remove(profile.Path); err != nil {
		return 3
	}

	delete(d.UserProfiles, profileName)
	return 1
}

func (d *Device) setupClusterController() {
	if d.DeviceProfile == nil {
		return
	}

	if !d.DeviceProfile.RGBCluster {
		return
	}

	clusterController := &common.ClusterController{
		Product:      d.Product,
		Serial:       d.Serial,
		LedChannels:  uint32(d.colorCount),
		WriteColorEx: d.writeColorCluster,
	}

	cluster.Get().AddDeviceController(clusterController)
}

// writeColorCluster will write data to the device from cluster client
func (d *Device) writeColorCluster(data []byte, _ int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.controllerId < 0 || d.DeviceProfile == nil || !d.DeviceProfile.RGBCluster {
		return
	}

	// Clamp data to our LED count in case the cluster sends more bytes than we own
	expected := d.colorCount * 3
	frame := make([]byte, expected)
	if len(data) >= expected {
		copy(frame, data[:expected])
	} else {
		copy(frame, data)
	}

	// Scale brightness across the entire frame
	scaled := d.applyBrightness(frame)

	conn, err := openrgb.SendFramePersistent(d.openrgbConn, uint32(d.controllerId), scaled)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": d.Serial, "controllerId": d.controllerId}).Error("SendFramePersistent failed in writeColorCluster")
		d.openrgbConn = nil
	} else {
		d.openrgbConn = conn
	}
}

// ProcessSetRgbCluster will update OpenRGB integration status for cluster
func (d *Device) ProcessSetRgbCluster(enabled bool) uint8 {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.DeviceProfile == nil {
		return 0
	}

	d.DeviceProfile.RGBCluster = enabled
	d.saveDeviceProfile()

	if enabled {
		d.stopEffectLoopLocked()

		clusterController := &common.ClusterController{
			Product:      d.Product,
			Serial:       d.Serial,
			LedChannels:  uint32(d.colorCount),
			WriteColorEx: d.writeColorCluster,
		}
		cluster.Get().AddDeviceController(clusterController)
	} else {
		cluster.Get().RemoveDeviceControllerBySerial(d.Serial)
		if d.openrgbConn != nil {
			d.openrgbConn.Close()
			d.openrgbConn = nil
		}
		if d.effect != "" {
			go func() {
				_ = d.SetEffect(d.effect)
			}()
		}
	}

	return 1
}
