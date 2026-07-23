package openrgbimport

import (
	"LumenForge/src/cluster"
	"LumenForge/src/common"
	"LumenForge/src/config"
	"LumenForge/src/dashboard"
	"LumenForge/src/logger"
	"LumenForge/src/openrgb"
	"LumenForge/src/rgb"
	"LumenForge/src/temperatures"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

var rgbModes = []string{
	"circle",
	"circleshift",
	"colorpulse",
	"colorshift",
	"colorwarp",
	"cpu-temperature",
	"flickering",
	"flame",
	"aurora", "cyberpunkglitch", "gpu-temperature",
	"gradient",
	"off",
	"rainbow",
	"pastelrainbow",
	"rotator",
	"spinner",
	"static",
	"storm",
	"watercolor",
	"wave",
}

const hardwareBufferDrainDelay = 75 * time.Millisecond

var (
	configStoreMutex sync.Mutex
	configStorePath  = func() string {
		return filepath.Join(config.GetConfig().ConfigPath, "database", "openrgbimport-zones.json")
	}
	renameConfigStore = os.Rename
	sendConfigFrame   = openrgb.SendFrame
	checkConfigHealth = openrgb.HealthCheck
	getConfigCluster  = cluster.Get
)

type ConfigStore struct {
	Devices map[string]DeviceConfig `json:"devices"`
}

type ZoneConfig struct {
	Name     string `json:"name"`
	LedCount int    `json:"ledCount"`
}

type DeviceConfig struct {
	Serial         string       `json:"serial"`
	Product        string       `json:"product,omitempty"`
	ExternalSerial string       `json:"externalSerial,omitempty"`
	Location       string       `json:"location,omitempty"`
	Vendor         string       `json:"vendor,omitempty"`
	Zones          []ZoneConfig `json:"zones"`
}

type ZoneColors struct {
	Color      *rgb.Color
	ColorIndex []int
	Name       string
}

type RGBOverride struct {
	Enabled        bool
	RGBStartColor  rgb.Color
	RGBEndColor    rgb.Color
	RGBMiddleColor rgb.Color
	RgbModeSpeed   float64
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
	RGBOverride      *RGBOverride       `json:"RGBOverride"`
}

type Device struct {
	Product            string
	Serial             string
	IsOpenRGB          bool
	DisplaySerial      string
	DisplaySerialLabel string
	instance           *common.Device
	controllerId       int
	colorCount         int
	LEDCount           int
	ZoneAmount         int
	Version            string
	Description        string
	Config             *DeviceConfig
	DeviceProfile      *DeviceProfile
	UserProfiles       map[string]*DeviceProfile
	Rgb                *rgb.RGB
	rgbMutex           sync.RWMutex
	RGBModes           []string

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

// DeviceSnapshot is an immutable presentation/configuration view of an imported device.
// It intentionally excludes live connections, workers, channels, mutexes, and callbacks.
type DeviceSnapshot struct {
	Product            string
	Serial             string
	IsOpenRGB          bool
	DisplaySerial      string
	DisplaySerialLabel string
	LEDCount           int
	ZoneAmount         int
	Version            string
	Description        string
	Config             *DeviceConfig
	DeviceProfile      *DeviceProfile
	UserProfiles       map[string]*DeviceProfile
	Rgb                *rgb.RGB
	RGBModes           []string
	Effect             string `json:"-"`
	Speed              string `json:"-"`
	Brightness         uint8  `json:"-"`
	RGBCluster         bool   `json:"-"`
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

func usableExternalSerial(value string) string {
	serial := sanitizeDisplaySerial(value)
	if !isUsableDisplaySerial(serial) {
		return ""
	}
	return serial
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

	hashInput := fmt.Sprintf("%s|%s|%s|%s|%d", dc.Name, dc.Vendor, dc.Version, dc.Description, len(dc.Zones))
	hash := sha256.Sum256([]byte(hashInput))
	fallback := fmt.Sprintf("ORGB-Import-%x", hash[:6])
	return fallback, "FALLBACK"
}

func isLegacyASUSMotherboardImport(name, vendor string) bool {
	nameLower := strings.ToLower(name)
	vendorLower := strings.ToLower(vendor)
	isGPU := strings.Contains(nameLower, "geforce") || strings.Contains(nameLower, "rtx") || strings.Contains(nameLower, "gtx") || strings.Contains(nameLower, "radeon") || strings.Contains(nameLower, "gpu") || strings.Contains(nameLower, "graphics") || strings.Contains(nameLower, "vga")
	if isGPU {
		return false
	}
	isAsus := strings.Contains(nameLower, "asus") || strings.Contains(vendorLower, "asus")
	isMobo := strings.Contains(nameLower, "motherboard") || strings.Contains(nameLower, "mainboard") || strings.Contains(nameLower, "rog strix") || strings.Contains(nameLower, "tuf") || strings.Contains(nameLower, "aura") || strings.Contains(vendorLower, "aura") || strings.Contains(nameLower, "prime")
	return isAsus && isMobo
}

func getConfigPath() string {
	return configStorePath()
}

func emptyConfigStore() *ConfigStore {
	return &ConfigStore{Devices: make(map[string]DeviceConfig)}
}

func loadConfigStoreUnlocked(configPath string) (*ConfigStore, error) {
	store := &ConfigStore{
		Devices: make(map[string]DeviceConfig),
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, fmt.Errorf("read OpenRGB import store: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("OpenRGB import store is empty")
	}

	if err = json.Unmarshal(data, store); err != nil {
		return nil, fmt.Errorf("decode OpenRGB import store: %w", err)
	}

	if store.Devices == nil {
		store.Devices = make(map[string]DeviceConfig)
	}

	return store, nil
}

func loadConfigStore() (*ConfigStore, error) {
	configStoreMutex.Lock()
	defer configStoreMutex.Unlock()
	return loadConfigStoreUnlocked(getConfigPath())
}

func saveConfigStoreUnlocked(configPath string, store *ConfigStore) error {
	if store == nil {
		store = emptyConfigStore()
	}
	if store.Devices == nil {
		store.Devices = make(map[string]DeviceConfig)
	}
	for serial, device := range store.Devices {
		device.ExternalSerial = usableExternalSerial(device.ExternalSerial)
		store.Devices[serial] = device
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	temporary, err := os.CreateTemp(filepath.Dir(configPath), ".openrgbimport-zones-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err = temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err = temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err = temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	if err = renameConfigStore(temporaryPath, configPath); err != nil {
		return err
	}

	return nil
}

func saveConfigStore(store *ConfigStore) error {
	configStoreMutex.Lock()
	defer configStoreMutex.Unlock()
	return saveConfigStoreUnlocked(getConfigPath(), store)
}

func updateConfigStore(update func(*ConfigStore) error) error {
	return updateConfigStoreIfChanged(func(store *ConfigStore) (bool, error) {
		return true, update(store)
	})
}

func updateConfigStoreIfChanged(update func(*ConfigStore) (bool, error)) error {
	configStoreMutex.Lock()
	defer configStoreMutex.Unlock()

	store, err := loadConfigStoreUnlocked(getConfigPath())
	if err != nil {
		return err
	}
	changed, err := update(store)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	return saveConfigStoreUnlocked(getConfigPath(), store)
}

func getDeviceConfig(serial string) (*DeviceConfig, error) {
	store, err := loadConfigStore()
	if err != nil {
		return nil, err
	}
	if cfg, ok := store.Devices[serial]; ok {
		deviceCfg := cfg
		return &deviceCfg, nil
	}
	return nil, nil
}

func sanitizeZoneName(name string) string {
	v := strings.Map(func(r rune) rune {
		if r == '\uFFFD' {
			return -1
		}
		if unicode.IsControl(r) || !unicode.IsPrint(r) {
			return -1
		}
		return r
	}, name)
	return strings.TrimSpace(v)
}

func buildDefaultDeviceConfig(serial string, dc openrgb.DiscoveredController) *DeviceConfig {
	nameLower := strings.ToLower(dc.Name)
	cfg := &DeviceConfig{Serial: serial, Product: dc.Name}

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
		// Compatibility fallback for current OpenRGB-reported zone lengths.
		// These defaults ensure stable initialization, though they remain user-editable.
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
		zoneLimit := len(dc.Zones)
		if zoneLimit > 128 {
			zoneLimit = 128
		}
		cfg.Zones = make([]ZoneConfig, zoneLimit)
		totalLeds := 0
		for i := 0; i < zoneLimit; i++ {
			z := dc.Zones[i]
			name := sanitizeZoneName(z.Name)
			if name == "" {
				name = fmt.Sprintf("Zone %d", i+1)
			}
			ledCount := z.LEDCount
			if ledCount <= 0 {
				ledCount = 1
			} else if ledCount > 1024 {
				ledCount = 1024
			}
			if totalLeds+ledCount > 4096 {
				ledCount = 4096 - totalLeds
				if ledCount <= 0 {
					ledCount = 1
				}
			}
			totalLeds += ledCount
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

func validateDeviceConfig(targetSerial string, input DeviceConfig, allowLegacyEmptySerial bool) (DeviceConfig, error) {
	if !common.AlphanumericDashRegex.MatchString(targetSerial) {
		return DeviceConfig{}, fmt.Errorf("OpenRGB import %q has an unusable internal serial; expected only letters, numbers, and dashes", targetSerial)
	}

	validated := input
	validated.Zones = append([]ZoneConfig(nil), input.Zones...)

	if validated.Serial == "" {
		if !allowLegacyEmptySerial {
			return DeviceConfig{}, fmt.Errorf("OpenRGB import %q has an empty internal serial; expected %q", targetSerial, targetSerial)
		}
		validated.Serial = targetSerial
	} else if validated.Serial != targetSerial {
		return DeviceConfig{}, fmt.Errorf("OpenRGB import %q stores conflicting internal serial %q; expected %q", targetSerial, validated.Serial, targetSerial)
	}

	if len(validated.Zones) < 1 || len(validated.Zones) > 128 {
		return DeviceConfig{}, fmt.Errorf("OpenRGB import %q has %d zones; expected 1 through 128", targetSerial, len(validated.Zones))
	}

	total := 0
	for index, zone := range validated.Zones {
		if zone.LedCount < 1 || zone.LedCount > 1024 {
			return DeviceConfig{}, fmt.Errorf("OpenRGB import %q zone %d has %d LEDs; expected 1 through 1024", targetSerial, index+1, zone.LedCount)
		}
		if zone.LedCount > 4096-total {
			return DeviceConfig{}, fmt.Errorf("OpenRGB import %q zone %d has %d LEDs and would exceed the permitted total range of 1 through 4096", targetSerial, index+1, zone.LedCount)
		}
		total += zone.LedCount

		name := sanitizeZoneName(zone.Name)
		if name == "" {
			name = fmt.Sprintf("Zone %d", index+1)
		}
		validated.Zones[index].Name = name
	}
	if total < 1 {
		return DeviceConfig{}, fmt.Errorf("OpenRGB import %q has a total of %d LEDs; expected 1 through 4096", targetSerial, total)
	}

	return validated, nil
}

func validateStoredDeviceConfig(mapSerial string, cfg DeviceConfig) (DeviceConfig, error) {
	return validateDeviceConfig(mapSerial, cfg, true)
}

func validateConfiguredStore(store *ConfigStore) error {
	for serial, cfg := range store.Devices {
		validated, err := validateStoredDeviceConfig(serial, cfg)
		if err != nil {
			return err
		}
		store.Devices[serial] = validated
	}
	return nil
}

func validatedConfigForController(cfg *DeviceConfig, _ openrgb.DiscoveredController) (*DeviceConfig, bool) {
	if cfg == nil {
		return nil, false
	}
	validated, err := validateDeviceConfig(cfg.Serial, *cfg, false)
	if err != nil {
		return nil, false
	}
	return &validated, true
}

func isConfigValidForController(cfg *DeviceConfig, dc openrgb.DiscoveredController) bool {
	_, valid := validatedConfigForController(cfg, dc)
	return valid
}

func resolveDeviceConfig(serial string, dc openrgb.DiscoveredController) *DeviceConfig {
	cfg, err := getDeviceConfig(serial)
	if err != nil {
		logger.Log(logger.Fields{"error": err, "serial": serial}).Error("Unable to load OpenRGB import configuration")
		return nil
	}
	if validated, valid := validatedConfigForController(cfg, dc); valid {
		cfg = validated
		if cfg.Product != dc.Name {
			cfg.Product = dc.Name
			if err := updateConfigStore(func(store *ConfigStore) error {
				stored := store.Devices[serial]
				stored.Product = dc.Name
				store.Devices[serial] = stored
				return nil
			}); err != nil {
				logger.Log(logger.Fields{"error": err, "serial": serial}).Error("Unable to update OpenRGB import configuration")
			}
		}
		return cfg
	}

	cfg = buildDefaultDeviceConfig(serial, dc)
	if validated, valid := validatedConfigForController(cfg, dc); valid {
		cfg = validated
	} else {
		logger.Log(logger.Fields{
			"serial":  serial,
			"product": dc.Name,
		}).Warn("OpenRGB controller returned invalid layout config. Enforcing fallback layout: 1 zone, 1 LED.")
		cfg = &DeviceConfig{
			Serial:  serial,
			Product: dc.Name,
			Zones: []ZoneConfig{
				{Name: "Zone 1", LedCount: 1},
			},
		}
	}

	if err := updateConfigStore(func(store *ConfigStore) error {
		store.Devices[serial] = *cloneDeviceConfig(cfg)
		return nil
	}); err != nil {
		logger.Log(logger.Fields{"error": err, "serial": serial}).Error("Unable to save OpenRGB import configuration")
	}

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
		Serial:         cfg.Serial,
		Product:        cfg.Product,
		ExternalSerial: cfg.ExternalSerial,
		Location:       cfg.Location,
		Vendor:         cfg.Vendor,
		Zones:          append([]ZoneConfig(nil), cfg.Zones...),
	}
	return cloned
}

func cloneDeviceProfile(profile *DeviceProfile) *DeviceProfile {
	if profile == nil {
		return nil
	}
	cloned := *profile
	if profile.BrightnessSlider != nil {
		brightness := *profile.BrightnessSlider
		cloned.BrightnessSlider = &brightness
	}
	if profile.RGBOverride != nil {
		override := *profile.RGBOverride
		cloned.RGBOverride = &override
	}
	if profile.ZoneColors != nil {
		cloned.ZoneColors = make(map[int]ZoneColors, len(profile.ZoneColors))
		for index, zone := range profile.ZoneColors {
			zoneCopy := zone
			if zone.Color != nil {
				colorCopy := *zone.Color
				zoneCopy.Color = &colorCopy
			}
			zoneCopy.ColorIndex = append([]int(nil), zone.ColorIndex...)
			cloned.ZoneColors[index] = zoneCopy
		}
	}
	return &cloned
}

func cloneRGBState(state *rgb.RGB) *rgb.RGB {
	if state == nil {
		return nil
	}
	cloned := *state
	if state.Profiles != nil {
		cloned.Profiles = make(map[string]rgb.Profile, len(state.Profiles))
		for name, profile := range state.Profiles {
			profileCopy := profile
			if profile.Gradients != nil {
				profileCopy.Gradients = make(map[int]rgb.Color, len(profile.Gradients))
				for index, color := range profile.Gradients {
					profileCopy.Gradients[index] = color
				}
			}
			cloned.Profiles[name] = profileCopy
		}
	}
	return &cloned
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
		if err := checkConfigHealth(); err != nil {
			return err
		}
	}

	return nil
}

func (d *Device) resolveControllerId() {
	if d.controllerId < 0 {
		requestReconciliation()
	}
}

func (d *Device) SaveDeviceConfig(cfg *DeviceConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}

	validated, err := validateDeviceConfig(d.Serial, *cfg, false)
	if err != nil {
		return err
	}

	total := configLedCount(&validated)
	if total <= 0 {
		return fmt.Errorf("OpenRGB import %q has a total of %d LEDs; expected 1 through 4096", d.Serial, total)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	savedCfg, err := getDeviceConfig(d.Serial)
	if err != nil {
		return err
	}
	if savedCfg != nil {
		validated.Product = savedCfg.Product
		validated.ExternalSerial = savedCfg.ExternalSerial
		validated.Location = savedCfg.Location
		validated.Vendor = savedCfg.Vendor
	} else if validated.Product == "" {
		validated.Product = d.Product
	}
	riskyIncrease := hasLEDCountIncrease(savedCfg, &validated)

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
	d.applyConfigLocked(&validated, brightness)

	d.resolveControllerId()

	if d.controllerId >= 0 {
		time.Sleep(hardwareBufferDrainDelay)
		if err := sendConfigFrame(uint32(d.controllerId), d.buildZoneFrame()); err != nil {
			d.applyConfigLocked(previousCfg, previousBrightness)
			d.recordOutputFailureLocked(err)
			return err
		}
		if riskyIncrease {
			if err := checkOpenRGBStable(4, 500*time.Millisecond); err != nil {
				d.applyConfigLocked(previousCfg, previousBrightness)
				return fmt.Errorf("OpenRGB became unavailable after applying increased LED counts; config was not saved. Confirm zone and LED counts in OpenRGB and try again")
			}
		}
	}

	if err := updateConfigStore(func(store *ConfigStore) error {
		if current, ok := store.Devices[d.Serial]; ok {
			validated.ExternalSerial = current.ExternalSerial
			validated.Location = current.Location
			validated.Vendor = current.Vendor
			if validated.Product == "" {
				validated.Product = current.Product
			}
		}
		store.Devices[d.Serial] = *cloneDeviceConfig(&validated)
		return nil
	}); err != nil {
		return err
	}

	if d.DeviceProfile != nil {
		d.saveDeviceProfile()
		if d.DeviceProfile.RGBCluster {
			serial := d.Serial
			d.mu.Unlock()
			if clusterDevice := getConfigCluster(); clusterDevice != nil {
				clusterDevice.RemoveDeviceControllerBySerial(serial)
			}
			d.setupClusterController()
			d.mu.Lock()
		}
	}

	return nil
}

func Init() *common.Device {
	d := &Device{
		Product:       "Imported ASUS Motherboard",
		Serial:        "openrgb-mobo-1",
		IsOpenRGB:     true,
		DisplaySerial: "",
		colorCount:    4,
		brightness:    100,
		lastColor:     []byte{99, 213, 255}, // default #63d5ff
		effect:        "static",
		speed:         2.0,
		stopChan:      nil,
		doneChan:      nil,
		running:       false,
		LEDCount:      4,
	}

	controllerId, err := openrgb.FindControllerIDByNameOrVendor(
		"asus rog strix z890-e gaming wifi",
		"asus aura",
	)
	if err != nil {
		d.controllerId = -1
	} else {
		d.controllerId = controllerId
	}

	cfg := resolveDeviceConfig(d.Serial, openrgb.DiscoveredController{
		Name:   d.Product,
		Vendor: "asus aura",
	})
	d.RGBModes = rgbModes
	d.loadRgb()
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
	productName := strings.TrimSpace(cfg.Product)
	if productName == "" {
		productName = "Imported OpenRGB Device"
	}
	if serial == "openrgb-mobo-1" {
		if strings.TrimSpace(cfg.Product) == "" {
			productName = "Imported ASUS Motherboard"
		}
	}

	d := &Device{
		Product:            productName,
		Serial:             serial,
		IsOpenRGB:          true,
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
		Config:             cloneDeviceConfig(&cfg),
		ZoneAmount:         len(cfg.Zones),
		LEDCount:           colorCount,
	}

	d.RGBModes = rgbModes
	d.loadRgb()

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
	store, err := loadConfigStore()
	if err != nil {
		openrgb.SetDisconnected(err)
		logger.Log(logger.Fields{"error": err, "location": getConfigPath()}).Error("Unable to load OpenRGB import store")
		setConfiguredDevices(nil)
		return nil
	}

	if len(store.Devices) == 0 {
		openrgb.SetNotConfigured()
		setConfiguredDevices(nil)
		return nil
	}
	if err = validateConfiguredStore(store); err != nil {
		openrgb.SetDisconnected(err)
		logger.Log(logger.Fields{"error": err, "location": getConfigPath()}).Error("Invalid OpenRGB import store")
		setConfiguredDevices(nil)
		return nil
	}
	openrgb.SetDisconnected(nil)

	serials := make([]string, 0, len(store.Devices))
	for serial := range store.Devices {
		serials = append(serials, serial)
	}
	sort.Strings(serials)

	configured := make(map[string]*Device, len(serials))
	result := make([]*common.Device, 0, len(serials))
	for _, serial := range serials {
		d := newOfflineDevice(serial, store.Devices[serial])
		d.createDevice()
		d.instance.Unavailable = true
		configured[serial] = d
		result = append(result, d.instance)
	}
	setConfiguredDevices(configured)
	return result
}

func migrateDeviceData(dc openrgb.DiscoveredController, newSerial string) {
	var candidateSerial string
	err := updateConfigStoreIfChanged(func(store *ConfigStore) (bool, error) {
		// Search order 1: Look for any older hash-based serial (openrgb-hash-*) with same product name
		for s, cfg := range store.Devices {
			if strings.HasPrefix(s, "openrgb-hash-") && cfg.Product == dc.Name && s != newSerial {
				candidateSerial = s
				break
			}
		}

		// Search order 2: Look for the specific ID-based serial (openrgb-import-ID)
		if candidateSerial == "" {
			oldImportSerial := fmt.Sprintf("openrgb-import-%d", dc.ID)
			if _, exists := store.Devices[oldImportSerial]; exists {
				candidateSerial = oldImportSerial
			}
		}

		// Search order 3: Look for any other entry with same product name
		if candidateSerial == "" {
			for s, cfg := range store.Devices {
				if cfg.Product == dc.Name && s != newSerial {
					candidateSerial = s
					break
				}
			}
		}

		if candidateSerial == "" {
			return false, nil
		}

		oldCfg := store.Devices[candidateSerial]
		oldCfg.Serial = newSerial
		store.Devices[newSerial] = oldCfg
		delete(store.Devices, candidateSerial)
		return true, nil
	})
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Unable to save migrated OpenRGB import store")
		return
	}
	if candidateSerial == "" {
		return
	}

	logger.Log(logger.Fields{
		"oldSerial": candidateSerial,
		"newSerial": newSerial,
		"product":   dc.Name,
	}).Info("Migrating OpenRGB device config and profiles to new persistent serial")

	// 2. Migrate dashboard settings
	dashboard.MigrateDeviceSerial(candidateSerial, newSerial)
	if cluster.Get() != nil {
		cluster.Get().MigrateDeviceOrderSerial(candidateSerial, newSerial)
	}

	// 3. Migrate profile files in database/profiles/
	profileDir := filepath.Join(config.GetConfig().ConfigPath, "database", "profiles")
	files, err := os.ReadDir(profileDir)
	if err != nil {
		return
	}

	for _, fi := range files {
		if fi.IsDir() {
			continue
		}
		if !common.IsValidExtension(fi.Name(), ".json") {
			continue
		}

		fileName := strings.TrimSuffix(fi.Name(), ".json")
		var newFileName string
		if fileName == candidateSerial {
			newFileName = newSerial + ".json"
		} else if strings.HasPrefix(fileName, candidateSerial+"-") {
			newFileName = newSerial + "-" + strings.TrimPrefix(fileName, candidateSerial+"-") + ".json"
		} else {
			continue
		}

		oldPath := filepath.Join(profileDir, fi.Name())
		newPath := filepath.Join(profileDir, newFileName)

		data, err := os.ReadFile(oldPath)
		if err != nil {
			continue
		}

		var pf DeviceProfile
		if err := json.Unmarshal(data, &pf); err != nil {
			continue
		}

		pf.Serial = newSerial
		pf.Path = newPath

		newData, err := json.MarshalIndent(pf, "", "  ")
		if err != nil {
			continue
		}

		if err := os.WriteFile(newPath, newData, 0o644); err == nil {
			_ = os.Remove(oldPath)
		}
	}
}

func newDeviceFromController(dc openrgb.DiscoveredController) *Device {
	isLegacyASUS := isLegacyASUSMotherboardImport(dc.Name, dc.Vendor)

	var serial string
	if isLegacyASUS {
		serial = "openrgb-mobo-1"
	} else {
		hashInput := fmt.Sprintf("%s|%s|%s|%s|%d", dc.Name, dc.Vendor, dc.Version, dc.Description, len(dc.Zones))
		hash := sha256.Sum256([]byte(hashInput))
		serial = fmt.Sprintf("openrgb-hash-%x", hash[:16])
	}

	if !isLegacyASUS {
		migrateDeviceData(dc, serial)
	}

	product := dc.Name
	displaySerial := ""
	displaySerialLabel := ""
	colorCount := dc.LEDCount

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
		IsOpenRGB:          true,
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
		LEDCount:           colorCount,
		Version:            dc.Version,
		Description:        dc.Description,
	}

	d.RGBModes = rgbModes
	d.loadRgb()

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

		// Apply initial state so the device lights up on boot
		if !d.DeviceProfile.RGBCluster {
			if d.effect == "static" || d.effect == "off" {
				d.mu.Lock()
				frame := d.buildZoneFrame()
				d.mu.Unlock()
				if len(frame) > 0 {
					conn, _ := openrgb.SendFramePersistent(d.openrgbConn, uint32(d.controllerId), frame)
					d.openrgbConn = conn
				}
			} else {
				go func() {
					_ = d.SetEffect(d.effect)
				}()
			}
		}
	}

	return d
}

func (d *Device) resolveDeviceIcon() string {
	nameLower := strings.ToLower(d.Product)
	descLower := strings.ToLower(d.Description)

	if strings.Contains(descLower, "motherboard") || strings.Contains(nameLower, "motherboard") || strings.Contains(nameLower, "z690") || strings.Contains(nameLower, "x570") || strings.Contains(nameLower, "z790") || strings.Contains(nameLower, "b650") {
		return "icon-motherboard.svg"
	}
	if strings.Contains(descLower, "gpu") || strings.Contains(descLower, "vga") || strings.Contains(nameLower, "geforce") || strings.Contains(nameLower, "radeon") {
		return "icon-device.svg"
	}
	if strings.Contains(descLower, "dram") || strings.Contains(nameLower, "ram") || strings.Contains(nameLower, "memory") || strings.Contains(nameLower, "ddr4") || strings.Contains(nameLower, "ddr5") {
		return "icon-ram.svg"
	}
	if strings.Contains(descLower, "keyboard") || strings.Contains(nameLower, "keyboard") {
		return "icon-keyboard.svg"
	}
	if strings.Contains(descLower, "mouse") || strings.Contains(nameLower, "mouse") {
		return "icon-mouse.svg"
	}
	if strings.Contains(nameLower, "strimer") || strings.Contains(nameLower, "controller") || strings.Contains(nameLower, "hub") || strings.Contains(nameLower, "node") || strings.Contains(nameLower, "commander") {
		return "icon-controller.svg"
	}

	return "icon-rgb.svg"
}

func (d *Device) createDevice() {
	d.instance = &common.Device{
		ProductType: common.ProductTypeMotherboard,
		Product:     d.Product,
		Serial:      d.Serial,
		Firmware:    "",
		Image:       d.resolveDeviceIcon(),
		Instance:    d,
		GetDevice:   d,
	}
}

// Snapshot returns a race-safe immutable copy for WebUI and JSON presentation.
func (d *Device) Snapshot() DeviceSnapshot {
	d.mu.Lock()
	defer d.mu.Unlock()

	effect := d.effect
	if effect == "" {
		effect = "static"
	}
	speed := "normal"
	switch d.speed {
	case 4.0:
		speed = "slow"
	case 0.8:
		speed = "fast"
	}

	var userProfiles map[string]*DeviceProfile
	if d.UserProfiles != nil {
		userProfiles = make(map[string]*DeviceProfile, len(d.UserProfiles))
		for name, profile := range d.UserProfiles {
			userProfiles[name] = cloneDeviceProfile(profile)
		}
	}
	rgbCluster := d.DeviceProfile != nil && d.DeviceProfile.RGBCluster

	d.rgbMutex.RLock()
	rgbState := cloneRGBState(d.Rgb)
	d.rgbMutex.RUnlock()

	return DeviceSnapshot{
		Product:            d.Product,
		Serial:             d.Serial,
		IsOpenRGB:          d.IsOpenRGB,
		DisplaySerial:      d.DisplaySerial,
		DisplaySerialLabel: d.DisplaySerialLabel,
		LEDCount:           d.LEDCount,
		ZoneAmount:         d.ZoneAmount,
		Version:            d.Version,
		Description:        d.Description,
		Config:             cloneDeviceConfig(d.Config),
		DeviceProfile:      cloneDeviceProfile(d.DeviceProfile),
		UserProfiles:       userProfiles,
		Rgb:                rgbState,
		RGBModes:           append([]string(nil), d.RGBModes...),
		Effect:             effect,
		Speed:              speed,
		Brightness:         d.brightness,
		RGBCluster:         rgbCluster,
	}
}

func (d *Device) GetDeviceTemplate() string {
	return "openrgb.html"
}

func (d *Device) ControllerID() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.controllerId
}

func (d *Device) bindController(dc openrgb.DiscoveredController) bool {
	d.mu.Lock()
	changed := d.controllerId != dc.ID
	if changed {
		d.stopEffectLoopLocked()
		if d.openrgbConn != nil {
			_ = d.openrgbConn.Close()
			d.openrgbConn = nil
		}
	}

	d.controllerId = dc.ID
	d.LEDCount = dc.LEDCount
	d.Version = dc.Version
	d.Description = dc.Description
	if strings.TrimSpace(dc.Name) != "" {
		d.Product = dc.Name
	}
	d.DisplaySerial, d.DisplaySerialLabel = pickDisplaySerialAndLabel(dc)
	d.updateIdentityMetadataLocked(dc)
	clusterController := d.clusterControllerLocked()
	d.mu.Unlock()

	addClusterController(clusterController)
	return changed
}

func (d *Device) wrapperPresentation() (product, firmware, image string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.Product, d.Version, d.resolveDeviceIcon()
}

func (d *Device) markUnavailable() bool {
	d.mu.Lock()
	changed := d.controllerId >= 0 || d.openrgbConn != nil || d.running
	d.stopEffectLoopLocked()
	d.controllerId = -1
	if d.openrgbConn != nil {
		_ = d.openrgbConn.Close()
		d.openrgbConn = nil
	}
	d.mu.Unlock()
	return changed
}

func (d *Device) updateIdentityMetadata(dc openrgb.DiscoveredController) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.updateIdentityMetadataLocked(dc)
}

func (d *Device) updateIdentityMetadataLocked(dc openrgb.DiscoveredController) {
	if d.Config == nil {
		return
	}
	d.Config.ExternalSerial = usableExternalSerial(d.Config.ExternalSerial)
	if externalSerial := usableExternalSerial(dc.Serial); externalSerial != "" {
		d.Config.ExternalSerial = externalSerial
	}
	if location := strings.TrimSpace(dc.Location); location != "" {
		d.Config.Location = location
	}
	if vendor := strings.TrimSpace(dc.Vendor); vendor != "" {
		d.Config.Vendor = vendor
	}
	if strings.TrimSpace(dc.Name) != "" {
		d.Config.Product = dc.Name
	}
}

func (d *Device) resumeDesiredState(ctx context.Context) error {
	d.mu.Lock()
	if d.controllerId < 0 || d.DeviceProfile == nil || d.DeviceProfile.RGBCluster {
		d.mu.Unlock()
		return nil
	}
	effect := d.effect
	if effect == "" {
		effect = d.DeviceProfile.RGBProfile
	}
	if effect == "" {
		effect = "static"
	}
	d.mu.Unlock()

	return d.setEffectContext(ctx, effect, false)
}

func (d *Device) recordOutputFailureLocked(err error) {
	if d.openrgbConn != nil {
		_ = d.openrgbConn.Close()
		d.openrgbConn = nil
	}
	d.controllerId = -1
	openrgb.SetDisconnected(err)
	reportOutputFailure(d, err)
}

func (d *Device) handleOutputFailure(err error) {
	if err == nil {
		return
	}
	d.mu.Lock()
	d.stopEffectLoopLocked()
	d.recordOutputFailureLocked(err)
	d.mu.Unlock()
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

	d.resolveControllerId()
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

		time.Sleep(hardwareBufferDrainDelay)
		err := openrgb.SendFrame(uint32(d.controllerId), d.buildZoneFrame())
		if err != nil {
			d.recordOutputFailureLocked(err)
		}
		return err
	}

	if d.DeviceProfile != nil {
		d.saveDeviceProfile()
	}

	scaled := d.applyBrightness(d.lastColor)
	err := openrgb.SendColor(uint32(d.controllerId), d.colorCount, scaled)
	if err != nil {
		d.recordOutputFailureLocked(err)
	}
	return err
}

func (d *Device) SetBrightness(brightness uint8) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.resolveControllerId()
	if d.controllerId < 0 {
		return fmt.Errorf("controllerId not set")
	}

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
		err := openrgb.SendFrame(uint32(d.controllerId), d.buildZoneFrame())
		if err != nil {
			d.recordOutputFailureLocked(err)
		}
		return err
	}

	scaled := d.applyBrightness(d.lastColor)

	err := openrgb.SendColor(uint32(d.controllerId), d.colorCount, scaled)
	if err != nil {
		d.recordOutputFailureLocked(err)
	}
	return err
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
	return d.setEffectContext(context.Background(), effect, true)
}

func (d *Device) setEffectContext(ctx context.Context, effect string, reportFailure bool) error {
	if ctx == nil {
		ctx = context.Background()
	}
	d.mu.Lock()

	d.resolveControllerId()

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

	// off just sets black and exits
	if effect == "off" {
		if d.openrgbConn != nil {
			d.openrgbConn.Close()
			d.openrgbConn = nil
		}

		controllerID := d.controllerId
		colorCount := d.colorCount
		d.mu.Unlock()

		// Wait for hardware buffer to drain, matching the static color sequence
		if err := waitForContext(ctx, hardwareBufferDrainDelay); err != nil {
			return err
		}
		err := openrgb.SendColorContext(ctx, uint32(controllerID), colorCount, []byte{0, 0, 0})
		if err != nil && reportFailure && ctx.Err() == nil {
			d.handleOutputFailure(err)
		}
		return err
	}

	// Static just reapplies current color once
	if effect == "static" {
		if d.openrgbConn != nil {
			d.openrgbConn.Close()
			d.openrgbConn = nil
		}

		if d.Config != nil && d.ZoneAmount > 0 {
			if err := waitForContext(ctx, hardwareBufferDrainDelay); err != nil {
				d.mu.Unlock()
				return err
			}
			frame := d.buildZoneFrame()
			controllerID := d.controllerId
			d.mu.Unlock()
			err := openrgb.SendFrameContext(ctx, uint32(controllerID), frame)
			if err != nil && reportFailure && ctx.Err() == nil {
				d.handleOutputFailure(err)
			}
			return err
		}

		scaled := d.applyBrightness(d.lastColor)
		controllerID := d.controllerId
		colorCount := d.colorCount
		d.mu.Unlock()
		err := openrgb.SendColorContext(ctx, uint32(controllerID), colorCount, scaled)
		if err != nil && reportFailure && ctx.Err() == nil {
			d.handleOutputFailure(err)
		}
		return err
	}
	if err := ctx.Err(); err != nil {
		d.mu.Unlock()
		return err
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	d.stopChan = stop
	d.doneChan = done
	d.running = true

	var initialStartColor *rgb.Color
	var initialEndColor *rgb.Color
	var initialSpeed float64 = d.speed

	if d.DeviceProfile != nil && d.DeviceProfile.RGBOverride != nil && d.DeviceProfile.RGBOverride.Enabled {
		initialStartColor = &d.DeviceProfile.RGBOverride.RGBStartColor
		initialEndColor = &d.DeviceProfile.RGBOverride.RGBEndColor
		initialSpeed = d.DeviceProfile.RGBOverride.RgbModeSpeed
	} else {
		profile := d.GetRgbProfile(effect)
		if profile != nil {
			initialStartColor = &profile.StartColor
			initialEndColor = &profile.EndColor
			initialSpeed = profile.Speed
		} else {
			initialStartColor = &rgb.Color{
				Red:        float64(d.lastColor[0]),
				Green:      float64(d.lastColor[1]),
				Blue:       float64(d.lastColor[2]),
				Brightness: rgb.GetBrightnessValueFloat(d.brightness),
			}
			initialEndColor = &rgb.Color{
				Red:        255,
				Green:      0,
				Blue:       255,
				Brightness: rgb.GetBrightnessValueFloat(d.brightness),
			}
		}
	}

	runner := rgb.New(
		d.colorCount,
		initialSpeed,
		initialStartColor,
		initialEndColor,
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

				// Check if effect changed or stopped
				if !d.running || d.effect == "static" || d.effect == "off" {
					d.mu.Unlock()
					return
				}

				// refresh dynamic values from saved profile if exists, else from device defaults
				pf := d.GetRgbProfile(d.effect)
				if pf != nil {
					runner.RgbModeSpeed = pf.Speed
					runner.RGBBrightness = rgb.GetBrightnessValueFloat(d.brightness)
					runner.RGBStartColor = &pf.StartColor
					runner.RGBEndColor = &pf.EndColor
					runner.MinTemp = pf.MinTemp
					runner.MaxTemp = pf.MaxTemp
				} else {
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
				}

				if runner.RGBMiddleColor == nil {
					runner.RGBMiddleColor = &rgb.Color{}
				}

				if d.DeviceProfile != nil && d.DeviceProfile.RGBOverride != nil && d.DeviceProfile.RGBOverride.Enabled {
					runner.RGBStartColor = &d.DeviceProfile.RGBOverride.RGBStartColor
					runner.RGBEndColor = &d.DeviceProfile.RGBOverride.RGBEndColor
					runner.RGBMiddleColor = &d.DeviceProfile.RGBOverride.RGBMiddleColor
					runner.RgbModeSpeed = common.FClamp(d.DeviceProfile.RGBOverride.RgbModeSpeed, 0.1, 10)
				}

				switch d.effect {
				case "rainbow":
					runner.Rainbow(startTime)
				case "pastelrainbow":
					runner.PastelRainbow(startTime)
				case "spiralrainbow":
					runner.SpiralRainbow(startTime)
				case "pastelspiralrainbow":
					runner.PastelSpiralRainbow(startTime)
				case "watercolor":
					runner.Watercolor(startTime)
				case "gradient":
					var gradients map[int]rgb.Color
					var speed float64 = 2.0
					if pf != nil {
						gradients = pf.Gradients
						speed = pf.Speed
					}
					runner.ColorshiftGradient(startTime, gradients, speed)
				case "cpu-temperature":
					runner.Temperature(float64(temperatures.GetCpuTemperature()))
				case "gpu-temperature":
					runner.Temperature(float64(temperatures.GetGpuTemperature()))
				case "colorpulse":
					runner.Colorpulse(&startTime)
				case "rotator":
					runner.Rotator(&startTime)
				case "wave":
					runner.Wave(&startTime)
				case "storm":
					runner.Storm()
				case "flickering":
					runner.Flickering(&startTime)
				case "flame":
					runner.Flame(&startTime)
				case "aurora":
					runner.Aurora(&startTime)
				case "cyberpunkglitch":
					runner.CyberpunkGlitch(&startTime)
				case "colorshift":
					runner.Colorshift(&startTime, runner)
				case "circleshift":
					runner.CircleShift(&startTime)
				case "circle":
					runner.Circle(&startTime)
				case "spinner":
					runner.Spinner(&startTime)
				case "colorwarp":
					runner.Colorwarp(&startTime, runner)
				default:
					runner.Static()
				}

				frame := make([]byte, len(runner.Output))
				copy(frame, runner.Output)

				conn, err := openrgb.SendFramePersistent(d.openrgbConn, uint32(controllerId), frame)
				if err != nil {
					d.running = false
					d.stopChan = nil
					d.doneChan = nil
					d.recordOutputFailureLocked(err)
					d.mu.Unlock()
					return
				} else {
					d.openrgbConn = conn
				}
				d.mu.Unlock()
			}
		}
	}()

	return nil
}

func waitForContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
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
	d.loadDeviceProfiles()
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

		var profileName string
		if fileName == d.Serial {
			profileName = "default"
		} else if strings.HasPrefix(fileName, d.Serial+"-") {
			profileName = strings.TrimPrefix(fileName, d.Serial+"-")
		} else {
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

		profileList[profileName] = pf
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
		profilePath := filepath.Join(profileDir, d.Serial+"-"+profileName+".json")

		// Deep copy ZoneColors map
		copiedZoneColors := make(map[int]ZoneColors)
		for k, v := range d.DeviceProfile.ZoneColors {
			var copiedColor *rgb.Color
			if v.Color != nil {
				copiedColor = &rgb.Color{
					Red:        v.Color.Red,
					Green:      v.Color.Green,
					Blue:       v.Color.Blue,
					Brightness: v.Color.Brightness,
					Hex:        v.Color.Hex,
				}
			}

			var copiedColorIndex []int
			if v.ColorIndex != nil {
				copiedColorIndex = make([]int, len(v.ColorIndex))
				copy(copiedColorIndex, v.ColorIndex)
			}

			copiedZoneColors[k] = ZoneColors{
				Color:      copiedColor,
				ColorIndex: copiedColorIndex,
				Name:       v.Name,
			}
		}

		// Deep copy BrightnessSlider pointer
		var copiedBrightness *uint8
		if d.DeviceProfile.BrightnessSlider != nil {
			val := *d.DeviceProfile.BrightnessSlider
			copiedBrightness = &val
		}

		newProfile := &DeviceProfile{
			Active:           false,
			Path:             profilePath,
			Product:          d.Product,
			Serial:           d.Serial,
			RGBProfile:       d.DeviceProfile.RGBProfile,
			BrightnessSlider: copiedBrightness,
			ZoneColors:       copiedZoneColors,
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
	d.mu.Lock()
	clusterController := d.clusterControllerLocked()
	d.mu.Unlock()
	addClusterController(clusterController)
}

func (d *Device) clusterControllerLocked() *common.ClusterController {
	if d.DeviceProfile == nil || !d.DeviceProfile.RGBCluster {
		return nil
	}
	return &common.ClusterController{
		Product:      d.Product,
		Serial:       d.Serial,
		LedChannels:  uint32(d.colorCount),
		WriteColorEx: d.writeColorCluster,
	}
}

func addClusterController(controller *common.ClusterController) {
	if controller == nil {
		return
	}
	if clusterDevice := cluster.Get(); clusterDevice != nil {
		clusterDevice.AddDeviceController(controller)
	}
}

// writeColorCluster will write data to the device from cluster client
func (d *Device) writeColorCluster(data []byte, _ int) {
	d.mu.Lock()

	if d.controllerId < 0 || d.DeviceProfile == nil || !d.DeviceProfile.RGBCluster {
		d.mu.Unlock()
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
		d.recordOutputFailureLocked(err)
	} else {
		d.openrgbConn = conn
	}
	d.mu.Unlock()
}

// ProcessSetRgbCluster will update OpenRGB integration status for cluster
func (d *Device) ProcessSetRgbCluster(enabled bool) uint8 {
	d.mu.Lock()
	if d.DeviceProfile == nil {
		d.mu.Unlock()
		return 0
	}

	d.DeviceProfile.RGBCluster = enabled
	d.saveDeviceProfile()

	if enabled {
		d.stopEffectLoopLocked()
		clusterController := d.clusterControllerLocked()
		serial := d.Serial
		d.mu.Unlock()
		if clusterDevice := cluster.Get(); clusterDevice != nil {
			clusterDevice.RemoveDeviceControllerBySerial(serial)
		}
		addClusterController(clusterController)
	} else {
		serial := d.Serial
		if d.openrgbConn != nil {
			d.openrgbConn.Close()
			d.openrgbConn = nil
		}
		effect := d.effect
		d.mu.Unlock()
		if clusterDevice := cluster.Get(); clusterDevice != nil {
			clusterDevice.RemoveDeviceControllerBySerial(serial)
		}
		if effect != "" {
			go func() {
				_ = d.SetEffect(effect)
			}()
		}
	}

	return 1
}

func (d *Device) Stop() {
	d.mu.Lock()
	d.stopEffectLoopLocked()
	if d.openrgbConn != nil {
		d.openrgbConn.Close()
		d.openrgbConn = nil
	}
	d.mu.Unlock()
}

func (d *Device) StopDirty() uint8 {
	d.mu.Lock()
	d.stopEffectLoopLocked()
	if d.openrgbConn != nil {
		d.openrgbConn.Close()
		d.openrgbConn = nil
	}
	d.mu.Unlock()
	return 2
}

func (d *Device) GetRgbProfiles() interface{} {
	d.rgbMutex.RLock()
	defer d.rgbMutex.RUnlock()

	if d.Rgb == nil {
		return nil
	}

	tmp := *d.Rgb

	// Filter unsupported modes out
	profiles := make(map[string]rgb.Profile, len(tmp.Profiles))
	for key, value := range tmp.Profiles {
		if slices.Contains(d.RGBModes, key) {
			profiles[key] = value
		}
	}
	tmp.Profiles = profiles
	return tmp
}

func (d *Device) GetRgbProfile(profile string) *rgb.Profile {
	d.rgbMutex.RLock()
	defer d.rgbMutex.RUnlock()

	if d.Rgb == nil {
		return nil
	}

	if val, ok := d.Rgb.Profiles[profile]; ok {
		return &val
	}
	return nil
}

func (d *Device) loadRgb() {
	d.rgbMutex.Lock()
	defer d.rgbMutex.Unlock()

	pwd := config.GetConfig().ConfigPath
	rgbDirectory := filepath.Join(pwd, "database", "rgb")
	rgbFilename := filepath.Join(rgbDirectory, d.Serial+".json")

	// Ensure directory exists
	_ = os.MkdirAll(rgbDirectory, 0o755)

	if !common.FileExists(rgbFilename) {
		profile := rgb.GetRGB()
		profile.Device = d.Product
		if profile.Profiles == nil {
			profile.Profiles = make(map[string]rgb.Profile)
		}

		if err := common.SaveJsonData(rgbFilename, profile); err != nil {
			fmt.Printf("Unable to write rgb profile data for %s: %v\n", d.Serial, err)
			return
		}
	}

	file, err := os.Open(rgbFilename)
	if err != nil {
		fmt.Printf("Unable to load RGB for %s: %v\n", d.Serial, err)
		return
	}
	defer file.Close()

	if err = json.NewDecoder(file).Decode(&d.Rgb); err != nil {
		fmt.Printf("Unable to decode profile for %s: %v\n", d.Serial, err)
		return
	}
	if d.Rgb.Profiles == nil {
		d.Rgb.Profiles = make(map[string]rgb.Profile)
	}

	// Upgrade profiles
	d.upgradeRgbProfileLocked(rgbFilename, []string{"gradient", "pastelrainbow", "pastelspiralrainbow", "flame", "aurora", "cyberpunkglitch"})
}

func (d *Device) upgradeRgbProfileLocked(path string, profiles []string) {
	if d.Rgb == nil {
		return
	}
	save := false
	for _, profile := range profiles {
		if _, ok := d.Rgb.Profiles[profile]; !ok {
			save = true
			template := rgb.GetRgbProfile(profile)
			if template == nil {
				d.Rgb.Profiles[profile] = rgb.Profile{}
			} else {
				d.Rgb.Profiles[profile] = *template
			}
		}
	}

	if save {
		if err := common.SaveJsonData(path, d.Rgb); err != nil {
			fmt.Printf("Unable to upgrade rgb profile data for %s: %v\n", d.Serial, err)
			return
		}
	}
}

func (d *Device) saveRgbProfile() {
	d.rgbMutex.Lock()
	defer d.rgbMutex.Unlock()

	if d.Rgb == nil {
		return
	}

	pwd := config.GetConfig().ConfigPath
	rgbFilename := filepath.Join(pwd, "database", "rgb", d.Serial+".json")

	if err := common.SaveJsonData(rgbFilename, d.Rgb); err != nil {
		fmt.Printf("Unable to write device rgb profile data for %s: %v\n", d.Serial, err)
		return
	}
}

func (d *Device) UpdateRgbProfileData(profileName string, profile rgb.Profile) uint8 {
	pf := d.GetRgbProfile(profileName)
	if pf == nil {
		return 0
	}

	d.rgbMutex.Lock()
	profile.StartColor.Brightness = pf.StartColor.Brightness
	profile.EndColor.Brightness = pf.EndColor.Brightness
	pf.StartColor = profile.StartColor
	pf.EndColor = profile.EndColor
	pf.Speed = profile.Speed
	pf.Gradients = profile.Gradients
	d.Rgb.Profiles[profileName] = *pf
	d.rgbMutex.Unlock()

	d.saveRgbProfile()

	// If we are currently running this effect, we want to restart/reapply it to pick up changes!
	if d.GetEffect() == profileName {
		_ = d.SetEffect(profileName)
	}

	return 1
}

func (d *Device) UpdateRgbProfile(_ int, profile string) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if d.GetRgbProfile(profile) == nil {
		return 0
	}

	if d.DeviceProfile.RGBCluster {
		return 5
	}

	err := d.SetEffect(profile)
	if err != nil {
		return 0
	}

	return 1
}

func (d *Device) ProcessGetRgbOverride(channelId, subDeviceId int) interface{} {
	defaultOverride := &RGBOverride{
		Enabled:        false,
		RGBStartColor:  rgb.Color{Red: 255, Green: 255, Blue: 255},
		RGBMiddleColor: rgb.Color{Red: 255, Green: 255, Blue: 255},
		RGBEndColor:    rgb.Color{Red: 255, Green: 255, Blue: 255},
		RgbModeSpeed:   5.0,
	}

	if d.DeviceProfile == nil {
		return defaultOverride
	}

	if d.DeviceProfile.RGBOverride == nil {
		d.DeviceProfile.RGBOverride = defaultOverride
	}

	return d.DeviceProfile.RGBOverride
}

func (d *Device) ProcessSetRgbOverride(channelId, subDeviceId int, enabled bool, startColor, endColor, middleColor rgb.Color, speed float64) uint8 {
	if d.DeviceProfile == nil {
		return 0
	}

	if d.DeviceProfile.RGBOverride == nil {
		d.DeviceProfile.RGBOverride = &RGBOverride{}
	}

	if speed < 0 || speed > 10 {
		return 0
	}

	d.DeviceProfile.RGBOverride.Enabled = enabled
	d.DeviceProfile.RGBOverride.RGBStartColor = startColor
	d.DeviceProfile.RGBOverride.RGBEndColor = endColor
	d.DeviceProfile.RGBOverride.RGBMiddleColor = middleColor
	d.DeviceProfile.RGBOverride.RgbModeSpeed = speed
	d.DeviceProfile.RGBOverride.RGBStartColor.Brightness = 1
	d.DeviceProfile.RGBOverride.RGBEndColor.Brightness = 1
	d.DeviceProfile.RGBOverride.RGBMiddleColor.Brightness = 1

	d.saveDeviceProfile()

	_ = d.SetEffect(d.DeviceProfile.RGBProfile)

	return 1
}
