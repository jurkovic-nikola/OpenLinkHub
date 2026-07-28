package openrgbimport

import (
	"LumenForge/src/cluster"
	"LumenForge/src/common"
	"LumenForge/src/config"
	"LumenForge/src/openrgb"
	"LumenForge/src/rgb"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	maxLifecycleBatch  = 64
	selectionKeyPrefix = "orgb-v1-"
	internalKeyPrefix  = "openrgb-hash-"
)

type PreviewZone struct {
	Name           string `json:"name"`
	LEDCount       int    `json:"ledCount"`
	Classification string `json:"classification,omitempty"`
}

type ControllerPreview struct {
	Key                string        `json:"key,omitempty"`
	IdentityKind       string        `json:"identityKind,omitempty"`
	Product            string        `json:"product"`
	Vendor             string        `json:"vendor,omitempty"`
	Version            string        `json:"version,omitempty"`
	Description        string        `json:"description,omitempty"`
	DisplaySerial      string        `json:"displaySerial"`
	DisplaySerialLabel string        `json:"displaySerialLabel"`
	ZoneCount          int           `json:"zoneCount"`
	LEDCount           int           `json:"ledCount"`
	Zones              []PreviewZone `json:"zones"`
	State              string        `json:"state"`
	ReasonCode         string        `json:"reasonCode,omitempty"`
	Reason             string        `json:"reason,omitempty"`
	ConfiguredSerial   string        `json:"configuredSerial,omitempty"`
}

type ConfiguredImportSummary struct {
	Serial   string `json:"serial"`
	Product  string `json:"product"`
	Disabled bool   `json:"disabled,omitempty"`
}

type DiscoveryPreview struct {
	DiscoveryState string                    `json:"discoveryState"`
	Error          string                    `json:"error,omitempty"`
	Configured     []ConfiguredImportSummary `json:"configured"`
	Controllers    []ControllerPreview       `json:"controllers"`
}

type ImportResult struct {
	ConfiguredSerials []string                    `json:"configuredSerials"`
	Configured        []ConfiguredImportSummary   `json:"configured"`
	Controllers       []ImportedControllerSummary `json:"controllers"`
}

type ImportedControllerSummary struct {
	Serial  string `json:"serial"`
	Product string `json:"product"`
}

type RemoveResult struct {
	RemovedSerials []string `json:"removedSerials"`
}

// RegistryHooks restrict lifecycle transactions to importer-specific registry
// operations and keep the openrgbimport package independent from src/devices.
type RegistryHooks struct {
	Register func(*common.Device, *Device) error
	Remove   func(string, *Device) (*common.Device, bool)
	Lookup   func(string) (*common.Device, *Device, bool)
}

type controllerIdentity struct {
	kind   string
	fields []string
	digest string
}

type discoveryCandidate struct {
	controller openrgb.DiscoveredController
	preview    ControllerPreview
	identity   controllerIdentity
	config     DeviceConfig
	prior      string
}

type artifact struct {
	path    string
	data    []byte
	created bool
}

type preparedImport struct {
	device    *Device
	wrapper   *common.Device
	artifacts []*artifact
	config    DeviceConfig
}

type priorStoreEntry struct {
	config  DeviceConfig
	present bool
}

var (
	deliberateDiscoveryGate    = make(chan struct{}, 1)
	lifecycleMutationGate      = make(chan struct{}, 1)
	deliberateDiscoveryTimeout = 10 * time.Second

	statusNeutralDiscover = openrgb.DiscoverControllersStatusNeutralContext
	lifecycleConfigRoot   = func() string { return config.GetPaths().MutableDataRoot }
	lifecycleRGBTemplate  = rgb.GetRGB
	createArtifactFile    = createArtifactExclusive
	removeArtifactFile    = os.Remove
	addLifecycleCluster   = func(controller *common.ClusterController) error {
		if controller == nil {
			return nil
		}
		if device := cluster.Get(); device != nil {
			device.AddDeviceController(controller)
		}
		return nil
	}
	removeLifecycleCluster = func(serial string) error {
		if device := cluster.Get(); device != nil {
			device.RemoveDeviceControllerBySerial(serial)
		}
		return nil
	}
	addLifecycleManager    = addManagerDevices
	removeLifecycleManager = removeManagerDevices
	stopLifecycleManager   = stopManagerIfNoConfigured
)

func acquireGate(ctx context.Context, gate chan struct{}) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case gate <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseGate(gate chan struct{}) {
	<-gate
}

func safePresentationString(value string, limit int) string {
	value = strings.Map(func(r rune) rune {
		if r == unicode.ReplacementChar || unicode.IsControl(r) || !unicode.IsPrint(r) {
			return -1
		}
		return r
	}, value)
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > limit {
		value = string(runes[:limit])
	}
	return value
}

func normalizeIdentityField(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(safePresentationString(value, 512)), " "))
}

func usableIdentityLocation(value string) bool {
	switch normalizeIdentityField(value) {
	case "",
		"dir",
		"dire",
		"direct",
		"direct mode",
		"off",
		"on",
		"none",
		"n/a",
		"na",
		"null",
		"unknown",
		"default",
		"undefined",
		"unavailable",
		"not available",
		"not applicable":
		return false
	default:
		return true
	}
}

func hashIdentity(kind string, fields ...string) string {
	hash := sha256.New()
	writeIdentityPart(hash, "openrgb-import-identity-v1")
	writeIdentityPart(hash, kind)
	for _, field := range fields {
		writeIdentityPart(hash, normalizeIdentityField(field))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

type identityWriter interface {
	Write([]byte) (int, error)
}

func writeIdentityPart(writer identityWriter, value string) {
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(value)))
	_, _ = writer.Write(length[:])
	_, _ = writer.Write([]byte(value))
}

func identityTuple(kind string, fields ...string) controllerIdentity {
	normalized := make([]string, len(fields))
	for index, field := range fields {
		normalized[index] = normalizeIdentityField(field)
	}
	return controllerIdentity{
		kind:   kind,
		fields: normalized,
		digest: hashIdentity(kind, normalized...),
	}
}

func chooseControllerIdentities(discovered []openrgb.DiscoveredController) []controllerIdentity {
	external := make([]controllerIdentity, len(discovered))
	location := make([]controllerIdentity, len(discovered))
	metadata := make([]controllerIdentity, len(discovered))
	externalCount := make(map[string]int)
	locationCount := make(map[string]int)
	metadataCount := make(map[string]int)

	for index, controller := range discovered {
		if serial := usableExternalSerial(controller.Serial); serial != "" {
			external[index] = identityTuple("external-serial", serial)
			externalCount[external[index].digest]++
		}
		if usableIdentityLocation(controller.Location) &&
			normalizeIdentityField(controller.Name) != "" {
			location[index] = identityTuple("location-product-vendor", controller.Location, controller.Name, controller.Vendor)
			locationCount[location[index].digest]++
		}
		if normalizeIdentityField(controller.Name) != "" {
			metadata[index] = identityTuple("product-vendor-name", controller.Name, controller.Vendor)
			metadataCount[metadata[index].digest]++
		}
	}

	result := make([]controllerIdentity, len(discovered))
	for index := range discovered {
		switch {
		case external[index].digest != "" && externalCount[external[index].digest] == 1:
			result[index] = external[index]
		case location[index].digest != "" && locationCount[location[index].digest] == 1:
			result[index] = location[index]
		case metadata[index].digest != "" && metadataCount[metadata[index].digest] == 1:
			result[index] = metadata[index]
		}
	}
	return result
}

func configuredSummaries(store *ConfigStore) []ConfiguredImportSummary {
	serials := make([]string, 0, len(store.Devices))
	for serial := range store.Devices {
		serials = append(serials, serial)
	}
	sort.Strings(serials)
	result := make([]ConfiguredImportSummary, 0, len(serials))
	for _, serial := range serials {
		cfg := store.Devices[serial]
		product := safePresentationString(cfg.Product, 160)
		if product == "" {
			product = "Imported OpenRGB Device"
		}
		result = append(result, ConfiguredImportSummary{
			Serial:   serial,
			Product:  product,
			Disabled: cfg.Disabled,
		})
	}
	return result
}

func defaultConfigForController(serial string, controller openrgb.DiscoveredController) (DeviceConfig, error) {
	cfg := DeviceConfig{
		Serial:         serial,
		Product:        safePresentationString(controller.Name, 512),
		ExternalSerial: safePresentationString(usableExternalSerial(controller.Serial), 512),
		Location:       safePresentationString(controller.Location, 512),
		Vendor:         safePresentationString(controller.Vendor, 512),
		Zones:          make([]ZoneConfig, len(controller.Zones)),
	}
	if cfg.Product == "" {
		cfg.Product = "Imported OpenRGB Device"
	}
	for index, zone := range controller.Zones {
		cfg.Zones[index] = ZoneConfig{Name: zone.Name, LedCount: zone.LEDCount}
	}
	if controller.LEDCount < 1 || controller.LEDCount > 4096 {
		return DeviceConfig{}, fmt.Errorf("controller reports %d LEDs; expected 1 through 4096", controller.LEDCount)
	}
	validated, err := validateDeviceConfig(serial, cfg, false)
	if err != nil {
		return DeviceConfig{}, err
	}
	return validated, nil
}

func hasAllZeroLEDMetadata(controller openrgb.DiscoveredController) bool {
	if controller.LEDCount != 0 || len(controller.Zones) > 128 {
		return false
	}
	for _, zone := range controller.Zones {
		if zone.LEDCount != 0 {
			return false
		}
	}
	return true
}

func fallbackConfigForController(serial string, controller openrgb.DiscoveredController) (DeviceConfig, error) {
	cfg := buildDefaultDeviceConfig(serial, controller)
	cfg.Serial = serial
	cfg.Product = safePresentationString(controller.Name, 512)
	if cfg.Product == "" {
		cfg.Product = "Imported OpenRGB Device"
	}
	cfg.ExternalSerial = safePresentationString(usableExternalSerial(controller.Serial), 512)
	cfg.Location = safePresentationString(controller.Location, 512)
	cfg.Vendor = safePresentationString(controller.Vendor, 512)
	return validateDeviceConfig(serial, *cfg, false)
}

func storeIdentityMatches(cfg DeviceConfig, serial string, identity controllerIdentity, controller openrgb.DiscoveredController) bool {
	if selectedIdentityMatches(cfg, identity) {
		return true
	}
	return serial == internalSerialForController(controller) && !legacyIdentityMetadataConflicts(cfg, controller)
}

func selectedIdentityMatches(cfg DeviceConfig, identity controllerIdentity) bool {
	switch identity.kind {
	case "external-serial":
		stored := normalizeIdentityField(usableExternalSerial(cfg.ExternalSerial))
		return stored != "" && len(identity.fields) == 1 && stored == identity.fields[0]
	case "location-product-vendor":
		return len(identity.fields) == 3 &&
			normalizeIdentityField(cfg.Location) != "" &&
			normalizeIdentityField(cfg.Product) != "" &&
			normalizeIdentityField(cfg.Location) == identity.fields[0] &&
			normalizeIdentityField(cfg.Product) == identity.fields[1] &&
			normalizeIdentityField(cfg.Vendor) == identity.fields[2]
	case "product-vendor-name":
		return len(identity.fields) == 2 &&
			normalizeIdentityField(cfg.Product) != "" &&
			normalizeIdentityField(cfg.Product) == identity.fields[0] &&
			normalizeIdentityField(cfg.Vendor) == identity.fields[1]
	default:
		return false
	}
}

func legacyIdentityMetadataConflicts(cfg DeviceConfig, controller openrgb.DiscoveredController) bool {
	storedExternal := normalizeIdentityField(usableExternalSerial(cfg.ExternalSerial))
	discoveredExternal := normalizeIdentityField(usableExternalSerial(controller.Serial))
	if storedExternal != "" && discoveredExternal != "" {
		return storedExternal != discoveredExternal
	}

	if nonemptyIdentityFieldsDiffer(cfg.Location, controller.Location) {
		return true
	}
	return nonemptyIdentityFieldsDiffer(cfg.Vendor, controller.Vendor)
}

func nonemptyIdentityFieldsDiffer(stored, discovered string) bool {
	stored = normalizeIdentityField(stored)
	discovered = normalizeIdentityField(discovered)
	return stored != "" && discovered != "" && stored != discovered
}

func selectedIdentityConflicts(cfg DeviceConfig, identity controllerIdentity) bool {
	var stored []string
	switch identity.kind {
	case "external-serial":
		stored = []string{normalizeIdentityField(usableExternalSerial(cfg.ExternalSerial))}
	case "location-product-vendor":
		stored = []string{
			normalizeIdentityField(cfg.Location),
			normalizeIdentityField(cfg.Product),
			normalizeIdentityField(cfg.Vendor),
		}
	case "product-vendor-name":
		stored = []string{
			normalizeIdentityField(cfg.Product),
			normalizeIdentityField(cfg.Vendor),
		}
	default:
		return false
	}
	if len(stored) != len(identity.fields) {
		return false
	}
	for index, value := range stored {
		if value != "" && identity.fields[index] != "" && value != identity.fields[index] {
			return true
		}
	}
	return false
}

func buildDiscoveryCandidates(store *ConfigStore, discovered []openrgb.DiscoveredController) []discoveryCandidate {
	identities := chooseControllerIdentities(discovered)
	result := make([]discoveryCandidate, len(discovered))
	for index, controller := range discovered {
		product := safePresentationString(controller.Name, 160)
		if product == "" {
			product = "Imported OpenRGB Device"
		}
		displaySerial, displayLabel := pickDisplaySerialAndLabel(controller)
		preview := ControllerPreview{
			Product:            product,
			Vendor:             safePresentationString(controller.Vendor, 160),
			Version:            safePresentationString(controller.Version, 160),
			Description:        safePresentationString(controller.Description, 512),
			DisplaySerial:      safePresentationString(displaySerial, 160),
			DisplaySerialLabel: safePresentationString(displayLabel, 32),
			ZoneCount:          len(controller.Zones),
			LEDCount:           controller.LEDCount,
			Zones:              make([]PreviewZone, len(controller.Zones)),
			State:              "selectable",
		}
		for zoneIndex, zone := range controller.Zones {
			preview.Zones[zoneIndex] = PreviewZone{
				Name:           safePresentationString(zone.Name, 160),
				LEDCount:       zone.LEDCount,
				Classification: safePresentationString(zone.Classification, 80),
			}
		}

		identity := identities[index]
		candidate := discoveryCandidate{controller: controller, identity: identity, preview: preview}
		if identity.digest == "" {
			candidate.preview.State = "ambiguous"
			candidate.preview.ReasonCode = "ambiguous_identity"
			candidate.preview.Reason = "This controller does not have a unique stable identity."
			result[index] = candidate
			continue
		}
		candidate.preview.IdentityKind = identity.kind
		candidate.preview.Key = selectionKeyPrefix + identity.digest

		proposedSerial := internalKeyPrefix + identity.digest
		matches := make([]string, 0, 1)
		for serial, stored := range store.Devices {
			if storeIdentityMatches(stored, serial, identity, controller) {
				matches = append(matches, serial)
			}
		}
		sort.Strings(matches)

		if stored, exists := store.Devices[proposedSerial]; exists &&
			!storeIdentityMatches(stored, proposedSerial, identity, controller) {
			candidate.preview.Key = ""
			candidate.preview.State = "invalid"
			candidate.preview.ReasonCode = "internal_serial_collision"
			if selectedIdentityConflicts(stored, identity) {
				candidate.preview.Reason = "The generated internal serial is already used by a configured import with conflicting identity metadata."
			} else {
				candidate.preview.Reason = "The generated internal serial is already used by a configured import whose identity cannot be verified."
			}
			result[index] = candidate
			continue
		}

		if len(matches) > 1 {
			candidate.preview.Key = ""
			candidate.preview.State = "ambiguous"
			candidate.preview.ReasonCode = "ambiguous_configured_identity"
			candidate.preview.Reason = "Multiple configured imports claim this controller identity."
			result[index] = candidate
			continue
		}

		cfg, layoutErr := defaultConfigForController(proposedSerial, controller)
		if len(matches) == 1 {
			candidate.prior = matches[0]
			candidate.preview.ConfiguredSerial = matches[0]
			stored := store.Devices[matches[0]]
			if layoutErr != nil {
				// OpenRGB can report unusable fresh layout metadata, including zero
				// LEDs, while this exact stable identity is operating from a
				// previously validated saved layout. Preserve that authoritative layout.
				candidate.config = *cloneDeviceConfig(&stored)
			} else {
				candidate.config = cfg
			}
			if !stored.Disabled {
				candidate.preview.State = "imported"
				candidate.preview.ReasonCode = "already_imported"
				if layoutErr != nil {
					candidate.preview.Reason = "This controller is already imported; its active saved layout is retained because the fresh OpenRGB layout is unusable."
				} else {
					candidate.preview.Reason = "This controller is already imported."
				}
			}
			result[index] = candidate
			continue
		}

		if layoutErr != nil && hasAllZeroLEDMetadata(controller) {
			fallback, fallbackErr := fallbackConfigForController(proposedSerial, controller)
			if fallbackErr == nil {
				candidate.config = fallback
				candidate.preview.ReasonCode = "fallback_layout"
				candidate.preview.Reason = "OpenRGB did not report LED counts. LumenForge will import a safe starting layout that can be edited after import."
				result[index] = candidate
				continue
			}
			layoutErr = fallbackErr
		}

		if layoutErr != nil {
			candidate.preview.Key = ""
			candidate.preview.State = "invalid"
			candidate.preview.ReasonCode = "invalid_layout"
			candidate.preview.Reason = safePresentationString(layoutErr.Error(), 512)
			result[index] = candidate
			continue
		}
		candidate.config = cfg
		result[index] = candidate
	}
	return result
}

func discoverWithStore(ctx context.Context, store *ConfigStore) ([]discoveryCandidate, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, deliberateDiscoveryTimeout)
	defer cancel()
	if localTargetServerEnabled() {
		return nil, targetServerConflictError()
	}
	if err := acquireGate(ctx, deliberateDiscoveryGate); err != nil {
		return nil, err
	}
	defer releaseGate(deliberateDiscoveryGate)

	discovered, err := statusNeutralDiscover(ctx)
	if err != nil {
		return nil, err
	}
	return buildDiscoveryCandidates(store, discovered), nil
}

// DiscoverPreview performs deliberate status-neutral discovery and always
// includes configured summaries when the store itself is valid.
func DiscoverPreview(ctx context.Context) (DiscoveryPreview, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result := DiscoveryPreview{
		DiscoveryState: "offline",
		Configured:     []ConfiguredImportSummary{},
		Controllers:    []ControllerPreview{},
	}
	store, err := loadConfigStore()
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	if err = validateConfiguredStore(store); err != nil {
		result.Error = err.Error()
		return result, err
	}
	result.Configured = configuredSummaries(store)

	candidates, err := discoverWithStore(ctx, store)
	if err != nil {
		result.Error = err.Error()
		if localTargetServerEnabled() {
			result.DiscoveryState = "conflict"
		}
		return result, err
	}
	result.DiscoveryState = "available"
	result.Controllers = make([]ControllerPreview, len(candidates))
	for index, candidate := range candidates {
		result.Controllers[index] = candidate.preview
	}
	return result, nil
}

func validateRegistryHooks(hooks RegistryHooks) error {
	if hooks.Register == nil || hooks.Remove == nil || hooks.Lookup == nil {
		return fmt.Errorf("OpenRGB importer registry hooks are unavailable")
	}
	return nil
}

func deduplicateValues(values []string, kind string) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("at least one %s is required", kind)
	}
	if len(values) > maxLifecycleBatch {
		return nil, fmt.Errorf("too many %s values; maximum batch size is %d", kind, maxLifecycleBatch)
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("%s values must not be empty", kind)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func prepareImport(serial string, cfg DeviceConfig) (*preparedImport, error) {
	colorCount := configLedCount(&cfg)
	product := safePresentationString(cfg.Product, 160)
	if product == "" {
		product = "Imported OpenRGB Device"
	}
	device := &Device{
		Product:             product,
		Serial:              serial,
		IsOpenRGB:           true,
		controllerId:        -1,
		colorCount:          colorCount,
		brightness:          100,
		lastColor:           []byte{99, 213, 255},
		effect:              "static",
		speed:               2,
		Config:              cloneDeviceConfig(&cfg),
		ZoneAmount:          len(cfg.Zones),
		LEDCount:            colorCount,
		RGBModes:            append([]string(nil), rgbModes...),
		UserProfiles:        make(map[string]*DeviceProfile),
		lifecycleActivating: true,
	}

	configRoot := lifecycleConfigRoot()
	rgbPath := filepath.Join(configRoot, "database", "rgb", serial+".json")
	profilePath := filepath.Join(configRoot, "database", "profiles", serial+".json")
	artifacts := make([]*artifact, 0, 2)

	if data, err := os.ReadFile(rgbPath); err == nil {
		var state rgb.RGB
		if err = json.Unmarshal(data, &state); err != nil {
			return nil, fmt.Errorf("decode preserved RGB file for %q: %w", serial, err)
		}
		if state.Profiles == nil {
			state.Profiles = make(map[string]rgb.Profile)
		}
		device.Rgb = &state
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read preserved RGB file for %q: %w", serial, err)
	} else {
		template := lifecycleRGBTemplate()
		state := cloneRGBState(&template)
		state.Device = product
		if state.Profiles == nil {
			state.Profiles = make(map[string]rgb.Profile)
		}
		device.Rgb = state
		data, marshalErr := json.MarshalIndent(state, "", "  ")
		if marshalErr != nil {
			return nil, marshalErr
		}
		artifacts = append(artifacts, &artifact{path: rgbPath, data: data})
	}

	defaultBrightness := uint8(100)
	defaultProfile := &DeviceProfile{
		Active:           true,
		Path:             profilePath,
		Product:          product,
		Serial:           serial,
		RGBProfile:       "static",
		BrightnessSlider: &defaultBrightness,
		ZoneColors:       buildZoneColorsFromConfig(&cfg, device.lastColor),
	}
	if err := loadPreservedProfiles(device, defaultProfile); err != nil {
		return nil, err
	}
	if _, err := os.Stat(profilePath); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("inspect preserved profile for %q: %w", serial, err)
		}
		if device.DeviceProfile == nil {
			device.DeviceProfile = defaultProfile
		} else {
			defaultProfile.Active = false
		}
		device.UserProfiles["default"] = defaultProfile
		data, marshalErr := json.MarshalIndent(defaultProfile, "", "  ")
		if marshalErr != nil {
			return nil, marshalErr
		}
		artifacts = append(artifacts, &artifact{path: profilePath, data: data})
	}
	if device.DeviceProfile == nil {
		device.DeviceProfile = defaultProfile
		device.UserProfiles["default"] = defaultProfile
	}

	device.createDevice()
	device.instance.Unavailable = true
	return &preparedImport{
		device:    device,
		wrapper:   device.instance,
		artifacts: artifacts,
		config:    cfg,
	}, nil
}

func loadPreservedProfiles(device *Device, defaultProfile *DeviceProfile) error {
	profileDir := filepath.Dir(defaultProfile.Path)
	files, err := os.ReadDir(profileDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read preserved profiles for %q: %w", device.Serial, err)
	}
	for _, file := range files {
		if file.IsDir() || !common.IsValidExtension(file.Name(), ".json") {
			continue
		}
		base := strings.TrimSuffix(file.Name(), ".json")
		name := ""
		switch {
		case base == device.Serial:
			name = "default"
		case strings.HasPrefix(base, device.Serial+"-"):
			name = strings.TrimPrefix(base, device.Serial+"-")
		default:
			continue
		}
		if !common.AlphanumericDashRegex.MatchString(base) {
			continue
		}
		path := filepath.Join(profileDir, file.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read preserved profile %q: %w", path, readErr)
		}
		profile := &DeviceProfile{ZoneColors: buildZoneColorsFromConfig(device.Config, device.lastColor)}
		if err = json.Unmarshal(data, profile); err != nil {
			return fmt.Errorf("decode preserved profile %q: %w", path, err)
		}
		profile.Path = path
		profile.Serial = device.Serial
		profile.Product = device.Product
		device.UserProfiles[name] = profile
		if profile.Active || (device.DeviceProfile == nil && name == "default") {
			device.DeviceProfile = profile
			if profile.BrightnessSlider != nil {
				device.brightness = *profile.BrightnessSlider
			}
			if profile.RGBProfile != "" {
				device.effect = profile.RGBProfile
			}
		}
	}
	return nil
}

func createArtifactExclusive(path string, data []byte) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return false, fmt.Errorf("artifact appeared after preparation")
		}
		return false, err
	}
	if _, err = file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return false, err
	}
	if err = file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return false, err
	}
	if err = file.Close(); err != nil {
		_ = os.Remove(path)
		return false, err
	}
	return true, nil
}

func activateArtifacts(prepared []*preparedImport) error {
	for _, item := range prepared {
		for _, artifact := range item.artifacts {
			created, err := createArtifactFile(artifact.path, artifact.data)
			if err != nil {
				return fmt.Errorf("create OpenRGB import artifact %q: %w", artifact.path, err)
			}
			artifact.created = created
		}
	}
	return nil
}

func rollbackArtifacts(prepared []*preparedImport) error {
	var result error
	for itemIndex := len(prepared) - 1; itemIndex >= 0; itemIndex-- {
		for artifactIndex := len(prepared[itemIndex].artifacts) - 1; artifactIndex >= 0; artifactIndex-- {
			artifact := prepared[itemIndex].artifacts[artifactIndex]
			if !artifact.created {
				continue
			}
			if err := removeArtifactFile(artifact.path); err != nil && !os.IsNotExist(err) {
				result = errors.Join(result, fmt.Errorf("remove created artifact %q: %w", artifact.path, err))
			}
		}
	}
	return result
}

func commitStoreChanges(expected *ConfigStore, changes map[string]DeviceConfig) (map[string]priorStoreEntry, error) {
	for serial, cfg := range changes {
		changes[serial] = canonicalDeviceConfigForPersistence(cfg)
	}
	prior := make(map[string]priorStoreEntry, len(changes))
	err := updateConfigStoreIfChanged(func(current *ConfigStore) (bool, error) {
		for serial := range changes {
			currentConfig, currentPresent := current.Devices[serial]
			prior[serial] = priorStoreEntry{
				config:  canonicalDeviceConfigForPersistence(currentConfig),
				present: currentPresent,
			}
		}
		if err := validateConfiguredStore(current); err != nil {
			return false, err
		}
		for serial := range changes {
			expectedConfig, expectedPresent := expected.Devices[serial]
			currentConfig, currentPresent := current.Devices[serial]
			expectedConfig = canonicalDeviceConfigForPersistence(expectedConfig)
			currentConfig = canonicalDeviceConfigForPersistence(currentConfig)
			if expectedPresent != currentPresent || (expectedPresent && !reflect.DeepEqual(expectedConfig, currentConfig)) {
				return false, fmt.Errorf("OpenRGB import store changed concurrently for serial %q", serial)
			}
		}
		for serial, cfg := range changes {
			current.Devices[serial] = canonicalDeviceConfigForPersistence(cfg)
		}
		return true, nil
	})
	return prior, err
}

func restoreStoreChanges(committed map[string]DeviceConfig, prior map[string]priorStoreEntry) error {
	return updateConfigStoreIfChanged(func(current *ConfigStore) (bool, error) {
		for serial, expected := range committed {
			actual, ok := current.Devices[serial]
			actual = canonicalDeviceConfigForPersistence(actual)
			expected = canonicalDeviceConfigForPersistence(expected)
			if !ok || !reflect.DeepEqual(actual, expected) {
				return false, fmt.Errorf("cannot roll back OpenRGB import %q because its store entry changed", serial)
			}
		}
		for serial, entry := range prior {
			if entry.present {
				current.Devices[serial] = canonicalDeviceConfigForPersistence(entry.config)
			} else {
				delete(current.Devices, serial)
			}
		}
		return true, nil
	})
}

func configuredSummaryFor(serial string, cfg DeviceConfig) ConfiguredImportSummary {
	product := safePresentationString(cfg.Product, 160)
	if product == "" {
		product = "Imported OpenRGB Device"
	}
	return ConfiguredImportSummary{Serial: serial, Product: product, Disabled: cfg.Disabled}
}

// ImportControllers validates a fresh status-neutral rediscovery and enrolls
// only the explicitly selected stable keys.
func ImportControllers(ctx context.Context, keys []string, registry RegistryHooks) (ImportResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result := ImportResult{
		ConfiguredSerials: []string{},
		Configured:        []ConfiguredImportSummary{},
		Controllers:       []ImportedControllerSummary{},
	}
	if err := validateRegistryHooks(registry); err != nil {
		return result, err
	}
	keys, err := deduplicateValues(keys, "selection key")
	if err != nil {
		return result, err
	}
	for _, key := range keys {
		if len(key) != len(selectionKeyPrefix)+sha256.Size*2 ||
			!strings.HasPrefix(key, selectionKeyPrefix) {
			return result, fmt.Errorf("invalid OpenRGB selection key %q", key)
		}
		if _, err = hex.DecodeString(strings.TrimPrefix(key, selectionKeyPrefix)); err != nil {
			return result, fmt.Errorf("invalid OpenRGB selection key %q", key)
		}
	}
	if localTargetServerEnabled() {
		return result, targetServerConflictError()
	}
	if err = acquireGate(ctx, lifecycleMutationGate); err != nil {
		return result, err
	}
	defer releaseGate(lifecycleMutationGate)
	if err = ctx.Err(); err != nil {
		return result, err
	}

	store, err := loadConfigStore()
	if err != nil {
		return result, err
	}
	if err = validateConfiguredStore(store); err != nil {
		return result, err
	}
	candidates, err := discoverWithStore(ctx, store)
	if err != nil {
		return result, err
	}
	byKey := make(map[string]discoveryCandidate, len(candidates))
	for _, candidate := range candidates {
		if candidate.preview.Key != "" {
			byKey[candidate.preview.Key] = candidate
		}
	}

	selected := make([]discoveryCandidate, 0, len(keys))
	changes := make(map[string]DeviceConfig)
	idempotent := make(map[string]DeviceConfig)
	for _, key := range keys {
		candidate, ok := byKey[key]
		if !ok {
			return result, fmt.Errorf("OpenRGB selection key %q is stale, missing, ambiguous, or invalid", key)
		}
		if candidate.preview.State == "imported" {
			serial := candidate.preview.ConfiguredSerial
			device, configured := configuredDevice(serial)
			_, registryDevice, registered := registry.Lookup(serial)
			if !configured || device == nil || !registered || registryDevice != device {
				return result, fmt.Errorf("configured OpenRGB import %q has inconsistent live membership", serial)
			}
			idempotent[serial] = store.Devices[serial]
			continue
		}
		if candidate.preview.State != "selectable" {
			return result, fmt.Errorf("OpenRGB selection key %q is not selectable", key)
		}

		serial := internalKeyPrefix + candidate.identity.digest
		cfg := candidate.config
		if candidate.prior != "" {
			serial = candidate.prior
			cfg = store.Devices[serial]
			cfg.Disabled = false
			if external := usableExternalSerial(candidate.controller.Serial); external != "" {
				cfg.ExternalSerial = safePresentationString(external, 512)
			}
			if location := safePresentationString(candidate.controller.Location, 512); location != "" {
				cfg.Location = location
			}
			if vendor := safePresentationString(candidate.controller.Vendor, 512); vendor != "" {
				cfg.Vendor = vendor
			}
			if product := safePresentationString(candidate.controller.Name, 512); product != "" {
				cfg.Product = product
			}
		}
		cfg.Serial = serial
		validated, validateErr := validateDeviceConfig(serial, cfg, false)
		if validateErr != nil {
			return result, validateErr
		}
		cfg = validated
		if existing, ok := store.Devices[serial]; ok && candidate.prior != serial {
			if !storeIdentityMatches(existing, serial, candidate.identity, candidate.controller) {
				return result, fmt.Errorf("OpenRGB internal serial collision for %q", serial)
			}
		}
		if _, ok := changes[serial]; ok {
			return result, fmt.Errorf("multiple selections resolve to OpenRGB import %q", serial)
		}
		if wrapper, _, ok := registry.Lookup(serial); ok || wrapper != nil {
			return result, fmt.Errorf("device registry already contains serial %q", serial)
		}
		candidate.config = cfg
		candidate.prior = serial
		selected = append(selected, candidate)
		changes[serial] = cfg
	}

	prepared := make([]*preparedImport, 0, len(selected))
	for _, candidate := range selected {
		if err = ctx.Err(); err != nil {
			for _, prior := range prepared {
				prior.device.Stop()
			}
			return result, err
		}
		item, prepareErr := prepareImport(candidate.prior, candidate.config)
		if prepareErr != nil {
			for _, prior := range prepared {
				prior.device.Stop()
			}
			return result, prepareErr
		}
		prepared = append(prepared, item)
	}
	if len(prepared) == 0 {
		for serial, cfg := range idempotent {
			result.ConfiguredSerials = append(result.ConfiguredSerials, serial)
			result.Configured = append(result.Configured, configuredSummaryFor(serial, cfg))
			result.Controllers = append(result.Controllers, ImportedControllerSummary{Serial: serial, Product: safePresentationString(cfg.Product, 160)})
		}
		sort.Strings(result.ConfiguredSerials)
		sort.Slice(result.Configured, func(i, j int) bool { return result.Configured[i].Serial < result.Configured[j].Serial })
		sort.Slice(result.Controllers, func(i, j int) bool { return result.Controllers[i].Serial < result.Controllers[j].Serial })
		return result, nil
	}
	if err = ctx.Err(); err != nil {
		for _, item := range prepared {
			item.device.Stop()
		}
		return result, err
	}

	prior, err := commitStoreChanges(store, changes)
	if err != nil {
		for _, item := range prepared {
			item.device.Stop()
		}
		return result, err
	}

	registered := make([]*preparedImport, 0, len(prepared))
	clustered := make([]*preparedImport, 0, len(prepared))
	configuredAdded := false
	managerAdded := false
	rollback := func(cause error) error {
		var rollbackErr error
		devicesMap := make(map[string]*Device, len(prepared))
		for _, item := range prepared {
			devicesMap[item.device.Serial] = item.device
		}
		if managerAdded {
			rollbackErr = errors.Join(rollbackErr, removeLifecycleManager(context.Background(), devicesMap))
		}
		if configuredAdded {
			rollbackErr = errors.Join(rollbackErr, removeConfiguredDevices(devicesMap))
		}
		for index := len(clustered) - 1; index >= 0; index-- {
			rollbackErr = errors.Join(
				rollbackErr,
				clustered[index].device.removeClusterControllerWith(removeLifecycleCluster),
			)
		}
		for index := len(registered) - 1; index >= 0; index-- {
			if _, removed := registry.Remove(registered[index].device.Serial, registered[index].device); !removed {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("could not remove transaction registry wrapper %q", registered[index].device.Serial))
			}
		}
		for _, item := range prepared {
			item.device.Stop()
		}
		rollbackErr = errors.Join(rollbackErr, rollbackArtifacts(prepared))
		rollbackErr = errors.Join(rollbackErr, restoreStoreChanges(changes, prior))
		stopLifecycleManager()
		if rollbackErr != nil {
			return errors.Join(cause, fmt.Errorf("OpenRGB import rollback failed: %w", rollbackErr))
		}
		return cause
	}

	if err = activateArtifacts(prepared); err != nil {
		return result, rollback(err)
	}
	for _, item := range prepared {
		if err = registry.Register(item.wrapper, item.device); err != nil {
			return result, rollback(err)
		}
		registered = append(registered, item)
	}
	for _, item := range prepared {
		item.device.mu.Lock()
		controller := item.device.clusterControllerLocked()
		item.device.mu.Unlock()
		if controller != nil {
			var added bool
			if added, err = item.device.registerLifecycleClusterControllerWith(controller, addLifecycleCluster); err != nil {
				return result, rollback(err)
			}
			if added {
				clustered = append(clustered, item)
			}
		}
	}
	devicesMap := make(map[string]*Device, len(prepared))
	for _, item := range prepared {
		devicesMap[item.device.Serial] = item.device
	}
	if err = addConfiguredDevices(devicesMap); err != nil {
		return result, rollback(err)
	}
	configuredAdded = true
	if _, err = addLifecycleManager(ctx, devicesMap); err != nil {
		return result, rollback(err)
	}
	managerAdded = true
	for _, item := range prepared {
		item.device.finishLifecycleActivation()
	}
	requestReconciliation()

	for serial, cfg := range idempotent {
		result.ConfiguredSerials = append(result.ConfiguredSerials, serial)
		result.Configured = append(result.Configured, configuredSummaryFor(serial, cfg))
		result.Controllers = append(result.Controllers, ImportedControllerSummary{Serial: serial, Product: safePresentationString(cfg.Product, 160)})
	}
	for _, item := range prepared {
		result.ConfiguredSerials = append(result.ConfiguredSerials, item.device.Serial)
		result.Configured = append(result.Configured, configuredSummaryFor(item.device.Serial, item.config))
		result.Controllers = append(result.Controllers, ImportedControllerSummary{
			Serial:  item.device.Serial,
			Product: safePresentationString(item.config.Product, 160),
		})
	}
	sort.Strings(result.ConfiguredSerials)
	sort.Slice(result.Configured, func(i, j int) bool { return result.Configured[i].Serial < result.Configured[j].Serial })
	sort.Slice(result.Controllers, func(i, j int) bool { return result.Controllers[i].Serial < result.Controllers[j].Serial })
	return result, nil
}

func managerIsActive() bool {
	activeManagerMutex.RLock()
	defer activeManagerMutex.RUnlock()
	return activeManager != nil
}

// RemoveConfiguredImports disables exact importer objects while preserving
// every profile, RGB file, dashboard entry, and saved cluster order.
func RemoveConfiguredImports(ctx context.Context, serials []string, registry RegistryHooks) (RemoveResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result := RemoveResult{RemovedSerials: []string{}}
	if err := validateRegistryHooks(registry); err != nil {
		return result, err
	}
	serials, err := deduplicateValues(serials, "serial")
	if err != nil {
		return result, err
	}
	for _, serial := range serials {
		if !common.AlphanumericDashRegex.MatchString(serial) {
			return result, fmt.Errorf("invalid OpenRGB import serial %q", serial)
		}
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if err = acquireGate(ctx, lifecycleMutationGate); err != nil {
		return result, err
	}
	defer releaseGate(lifecycleMutationGate)
	if err = ctx.Err(); err != nil {
		return result, err
	}

	store, err := loadConfigStore()
	if err != nil {
		return result, err
	}
	if err = validateConfiguredStore(store); err != nil {
		return result, err
	}

	targets := make(map[string]*Device, len(serials))
	changes := make(map[string]DeviceConfig, len(serials))
	for _, serial := range serials {
		cfg, ok := store.Devices[serial]
		if !ok || cfg.Disabled {
			return result, fmt.Errorf("OpenRGB import %q is missing or already disabled", serial)
		}
		configured, ok := configuredDevice(serial)
		if !ok || configured == nil {
			return result, fmt.Errorf("OpenRGB import %q is not an enabled configured importer", serial)
		}
		wrapper, instance, ok := registry.Lookup(serial)
		if !ok || wrapper == nil || instance != configured {
			return result, fmt.Errorf("OpenRGB import %q does not have the expected registry wrapper", serial)
		}
		targets[serial] = configured
		cfg.Disabled = true
		changes[serial] = cfg
	}

	wasManagerActive := managerIsActive()
	managerRemoved := false
	clusterRemoved := make(map[string]*common.ClusterController)
	storeCommitted := false
	configuredRemoved := false
	registryRemoved := make(map[string]bool)
	removedWrappers := make(map[string]*common.Device)
	var prior map[string]priorStoreEntry

	rollback := func(cause error) error {
		var rollbackErr error
		if storeCommitted {
			rollbackErr = errors.Join(rollbackErr, restoreStoreChanges(changes, prior))
		}
		if configuredRemoved {
			rollbackErr = errors.Join(rollbackErr, addConfiguredDevices(targets))
		}
		for _, serial := range serials {
			if registryRemoved[serial] {
				rollbackErr = errors.Join(rollbackErr, registry.Register(removedWrappers[serial], targets[serial]))
			}
		}
		for _, serial := range serials {
			if controller := clusterRemoved[serial]; controller != nil {
				rollbackErr = errors.Join(
					rollbackErr,
					targets[serial].restoreAfterRemoval(controller, addLifecycleCluster),
				)
			} else if _, detached := clusterRemoved[serial]; detached {
				rollbackErr = errors.Join(
					rollbackErr,
					targets[serial].restoreAfterRemoval(nil, addLifecycleCluster),
				)
			}
		}
		if managerRemoved && wasManagerActive {
			_, managerErr := addLifecycleManager(context.Background(), targets)
			rollbackErr = errors.Join(rollbackErr, managerErr)
		}
		if rollbackErr != nil {
			return errors.Join(cause, fmt.Errorf("OpenRGB removal rollback failed: %w", rollbackErr))
		}
		return cause
	}

	if err = removeLifecycleManager(ctx, targets); err != nil {
		return result, err
	}
	managerRemoved = wasManagerActive
	for _, serial := range serials {
		device := targets[serial]
		var controller *common.ClusterController
		controller, err = device.detachForRemoval(removeLifecycleCluster)
		clusterRemoved[serial] = controller
		if err != nil {
			return result, rollback(err)
		}
	}
	prior, err = commitStoreChanges(store, changes)
	if err != nil {
		return result, rollback(err)
	}
	storeCommitted = true
	if err = removeConfiguredDevices(targets); err != nil {
		return result, rollback(err)
	}
	configuredRemoved = true
	for _, serial := range serials {
		removedWrapper, removed := registry.Remove(serial, targets[serial])
		if !removed || removedWrapper == nil {
			return result, rollback(fmt.Errorf("registry wrapper for OpenRGB import %q changed during removal", serial))
		}
		registryRemoved[serial] = true
		removedWrappers[serial] = removedWrapper
	}
	stopLifecycleManager()
	result.RemovedSerials = append(result.RemovedSerials, serials...)
	return result, nil
}
