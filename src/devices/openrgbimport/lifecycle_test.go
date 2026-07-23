package openrgbimport

import (
	"LumenForge/src/common"
	"LumenForge/src/openrgb"
	"LumenForge/src/rgb"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeImportRegistry struct {
	mutex            sync.Mutex
	wrappers         map[string]*common.Device
	failRegister     bool
	failRemove       bool
	failRemoveSerial string
}

func newFakeImportRegistry() *fakeImportRegistry {
	return &fakeImportRegistry{wrappers: make(map[string]*common.Device)}
}

func (registry *fakeImportRegistry) hooks() RegistryHooks {
	return RegistryHooks{
		Register: registry.register,
		Remove:   registry.remove,
		Lookup:   registry.lookup,
	}
}

func (registry *fakeImportRegistry) register(wrapper *common.Device, expected *Device) error {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()
	if registry.failRegister {
		return errors.New("injected registry failure")
	}
	if existing, ok := registry.wrappers[wrapper.Serial]; ok {
		if existing == wrapper && existing.Instance == expected {
			return nil
		}
		return errors.New("registry collision")
	}
	registry.wrappers[wrapper.Serial] = wrapper
	return nil
}

func (registry *fakeImportRegistry) remove(serial string, expected *Device) (*common.Device, bool) {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()
	if registry.failRemove || registry.failRemoveSerial == serial {
		return nil, false
	}
	wrapper, ok := registry.wrappers[serial]
	if !ok || wrapper.Instance != expected {
		return nil, false
	}
	delete(registry.wrappers, serial)
	return wrapper, true
}

func (registry *fakeImportRegistry) wrapper(serial string) *common.Device {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()
	return registry.wrappers[serial]
}

func (registry *fakeImportRegistry) lookup(serial string) (*common.Device, *Device, bool) {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()
	wrapper, ok := registry.wrappers[serial]
	if !ok {
		return nil, nil, false
	}
	instance, ok := wrapper.Instance.(*Device)
	if !ok {
		return nil, nil, false
	}
	copy := *wrapper
	return &copy, instance, true
}

func (registry *fakeImportRegistry) count() int {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()
	return len(registry.wrappers)
}

type fakeLifecycleCluster struct {
	mutex       sync.Mutex
	controllers map[string]*common.ClusterController
	adds        int
	removes     int
}

func newFakeLifecycleCluster() *fakeLifecycleCluster {
	return &fakeLifecycleCluster{controllers: make(map[string]*common.ClusterController)}
}

func (state *fakeLifecycleCluster) add(controller *common.ClusterController) error {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	if _, exists := state.controllers[controller.Serial]; exists {
		return fmt.Errorf("duplicate cluster controller for %q", controller.Serial)
	}
	state.controllers[controller.Serial] = controller
	state.adds++
	return nil
}

func (state *fakeLifecycleCluster) remove(serial string) error {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	delete(state.controllers, serial)
	state.removes++
	return nil
}

func (state *fakeLifecycleCluster) counts() (controllers, adds, removes int) {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	return len(state.controllers), state.adds, state.removes
}

func (state *fakeLifecycleCluster) controller(serial string) *common.ClusterController {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	return state.controllers[serial]
}

func lifecycleController(name, serial, location string, leds int) openrgb.DiscoveredController {
	return openrgb.DiscoveredController{
		Name:        name,
		Vendor:      "Test Vendor",
		Description: "Test Description",
		Version:     "1.0",
		Serial:      serial,
		Location:    location,
		LEDCount:    leds,
		Zones: []openrgb.DiscoveredZone{
			{Name: "Main Zone", LEDCount: leds, Classification: "linear"},
		},
	}
}

func setupLifecycleTest(t *testing.T) (string, string) {
	t.Helper()
	StopManager()
	setConfiguredDevices(nil)
	storePath := useStorePath(t)
	root := t.TempDir()

	previousRoot := lifecycleConfigRoot
	previousDiscovery := statusNeutralDiscover
	previousRGBTemplate := lifecycleRGBTemplate
	previousTarget := localTargetServerEnabled
	previousCreate := createArtifactFile
	previousRemove := removeArtifactFile
	previousClusterAdd := addLifecycleCluster
	previousClusterRemove := removeLifecycleCluster
	previousManagerAdd := addLifecycleManager
	previousManagerRemove := removeLifecycleManager
	previousManagerStop := stopLifecycleManager
	previousDiscoveryTimeout := deliberateDiscoveryTimeout

	lifecycleConfigRoot = func() string { return root }
	localTargetServerEnabled = func() bool { return false }
	createArtifactFile = createArtifactExclusive
	removeArtifactFile = os.Remove
	addLifecycleCluster = func(*common.ClusterController) error { return nil }
	removeLifecycleCluster = func(string) error { return nil }
	addLifecycleManager = func(context.Context, map[string]*Device) (bool, error) { return false, nil }
	removeLifecycleManager = func(context.Context, map[string]*Device) error { return nil }
	stopLifecycleManager = func() bool { return false }
	deliberateDiscoveryTimeout = 10 * time.Second

	if err := saveConfigStore(emptyConfigStore()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		StopManager()
		setConfiguredDevices(nil)
		lifecycleConfigRoot = previousRoot
		statusNeutralDiscover = previousDiscovery
		lifecycleRGBTemplate = previousRGBTemplate
		localTargetServerEnabled = previousTarget
		createArtifactFile = previousCreate
		removeArtifactFile = previousRemove
		addLifecycleCluster = previousClusterAdd
		removeLifecycleCluster = previousClusterRemove
		addLifecycleManager = previousManagerAdd
		removeLifecycleManager = previousManagerRemove
		stopLifecycleManager = previousManagerStop
		deliberateDiscoveryTimeout = previousDiscoveryTimeout
	})
	return storePath, root
}

func assertLifecycleGateAvailable(t *testing.T, gate chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := acquireGate(ctx, gate); err != nil {
		t.Fatalf("lifecycle gate remained blocked: %v", err)
	}
	releaseGate(gate)
}

func TestPrepareImportDeepCopiesRGBAndCompletesLiveDefaultProfile(t *testing.T) {
	_, _ = setupLifecycleTest(t)
	globalBeforeValue := rgb.GetRGB()
	globalBefore := cloneRGBState(&globalBeforeValue)
	globalConfig := testConfig("openrgb-prepare-global", "Global")
	globalPrepared, err := prepareImport(globalConfig.Serial, globalConfig)
	if err != nil {
		t.Fatal(err)
	}
	globalPrepared.device.Rgb.Profiles["mutation-probe"] = rgb.Profile{ProfileName: "mutation-probe"}
	globalAfterValue := rgb.GetRGB()
	globalAfter := cloneRGBState(&globalAfterValue)
	if !reflect.DeepEqual(globalBefore, globalAfter) {
		t.Fatal("mutating a prepared import changed rgb.GetRGB global state")
	}
	t.Cleanup(globalPrepared.device.Stop)

	template := rgb.RGB{
		Device: "Global Template",
		Profiles: map[string]rgb.Profile{
			"gradient": {
				ProfileName: "gradient",
				Gradients: map[int]rgb.Color{
					0: {Red: 10, Green: 20, Blue: 30},
				},
			},
		},
	}
	lifecycleRGBTemplate = func() rgb.RGB { return template }

	firstConfig := testConfig("openrgb-prepare-first", "First")
	secondConfig := testConfig("openrgb-prepare-second", "Second")
	first, err := prepareImport(firstConfig.Serial, firstConfig)
	if err != nil {
		t.Fatal(err)
	}
	second, err := prepareImport(secondConfig.Serial, secondConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		first.device.Stop()
		second.device.Stop()
	})

	firstDefault := first.device.UserProfiles["default"]
	if firstDefault == nil || first.device.DeviceProfile != firstDefault || !firstDefault.Active {
		t.Fatalf("fresh default profile membership = %#v, active profile %p", first.device.UserProfiles, first.device.DeviceProfile)
	}
	if secondDefault := second.device.UserProfiles["default"]; secondDefault == nil ||
		second.device.DeviceProfile != secondDefault || !secondDefault.Active {
		t.Fatalf("second fresh default profile membership = %#v, active profile %p", second.device.UserProfiles, second.device.DeviceProfile)
	}

	firstGradient := first.device.Rgb.Profiles["gradient"]
	firstGradient.Gradients[0] = rgb.Color{Red: 200}
	firstGradient.Gradients[1] = rgb.Color{Green: 201}
	first.device.Rgb.Profiles["gradient"] = firstGradient
	first.device.Rgb.Profiles["new-profile"] = rgb.Profile{ProfileName: "new-profile"}

	if _, ok := template.Profiles["new-profile"]; ok {
		t.Fatal("prepared import mutated the global RGB template profile map")
	}
	if got := template.Profiles["gradient"].Gradients[0].Red; got != 10 {
		t.Fatalf("prepared import mutated global template gradients: %v", got)
	}
	if _, ok := second.device.Rgb.Profiles["new-profile"]; ok {
		t.Fatal("fresh imports share their RGB profile map")
	}
	if got := second.device.Rgb.Profiles["gradient"].Gradients[0].Red; got != 10 {
		t.Fatalf("fresh imports share gradient maps: %v", got)
	}
}

func TestPrepareImportAddsInactiveDefaultBesidePreservedActiveProfile(t *testing.T) {
	_, root := setupLifecycleTest(t)
	cfg := testConfig("openrgb-prepare-preserved", "Preserved")
	profilePath := filepath.Join(root, "database", "profiles", cfg.Serial+"-custom.json")
	if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
		t.Fatal(err)
	}
	custom := DeviceProfile{Active: true, RGBProfile: "rainbow", Serial: cfg.Serial}
	data, err := json.Marshal(custom)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(profilePath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	prepared, err := prepareImport(cfg.Serial, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(prepared.device.Stop)
	defaultProfile := prepared.device.UserProfiles["default"]
	customProfile := prepared.device.UserProfiles["custom"]
	if defaultProfile == nil || defaultProfile.Active {
		t.Fatalf("default profile = %#v, want present and inactive", defaultProfile)
	}
	if customProfile == nil || !customProfile.Active || prepared.device.DeviceProfile != customProfile {
		t.Fatalf("active custom profile = %#v, DeviceProfile=%p", customProfile, prepared.device.DeviceProfile)
	}
	if prepared.device.DeviceProfile == defaultProfile {
		t.Fatal("inactive default incorrectly replaced the preserved active profile")
	}
}

func TestDiscoveryPreviewIdentitySafetyAndStability(t *testing.T) {
	storePath, _ := setupLifecycleTest(t)
	controllers := []openrgb.DiscoveredController{
		lifecycleController("External\x00 Product", "external-123", "", 4),
		lifecycleController("Location Product", "unknown", "usb:1", 3),
		lifecycleController("Metadata Product", "none", "", 2),
		lifecycleController("Duplicate", "unknown", "", 1),
		lifecycleController("Duplicate", "unknown", "", 1),
		lifecycleController("Invalid", "invalid-layout", "", 1025),
	}
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return append([]openrgb.DiscoveredController(nil), controllers...), nil
	}
	before, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatal(err)
	}
	openrgb.SetDisconnected(errors.New("manager status"))

	preview, err := DiscoverPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("discovery changed the importer store")
	}
	state, statusErr := openrgb.GetStatus()
	if state != openrgb.StateOffline || statusErr == nil || statusErr.Error() != "manager status" {
		t.Fatalf("discovery changed SDK status: %q, %v", state, statusErr)
	}
	if preview.DiscoveryState != "available" || len(preview.Controllers) != len(controllers) {
		t.Fatalf("preview = %#v", preview)
	}

	byProduct := make(map[string]ControllerPreview)
	for _, controller := range preview.Controllers {
		byProduct[controller.Product] = controller
		if strings.ContainsRune(controller.Product, '\x00') {
			t.Fatalf("unsafe product was not sanitized: %q", controller.Product)
		}
	}
	for _, product := range []string{"External Product", "Location Product", "Metadata Product"} {
		item := byProduct[product]
		if len(item.Key) != len(selectionKeyPrefix)+64 || !strings.HasPrefix(item.Key, selectionKeyPrefix) || item.State != "selectable" {
			t.Fatalf("%s preview = %#v", product, item)
		}
	}
	if byProduct["External Product"].IdentityKind != "external-serial" {
		t.Fatalf("external identity = %#v", byProduct["External Product"])
	}
	if byProduct["Location Product"].IdentityKind != "location-product-vendor" {
		t.Fatalf("location identity = %#v", byProduct["Location Product"])
	}
	if byProduct["Metadata Product"].IdentityKind != "product-vendor-name" {
		t.Fatalf("metadata identity = %#v", byProduct["Metadata Product"])
	}
	if item := byProduct["Duplicate"]; item.State != "ambiguous" || item.Key != "" {
		t.Fatalf("duplicate preview = %#v", item)
	}
	if item := byProduct["Invalid"]; item.State != "invalid" || item.Key != "" {
		t.Fatalf("invalid preview = %#v", item)
	}

	firstKeys := make(map[string]string)
	for _, item := range preview.Controllers {
		firstKeys[item.Product] = item.Key
	}
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		reordered := append([]openrgb.DiscoveredController(nil), controllers...)
		for left, right := 0, len(reordered)-1; left < right; left, right = left+1, right-1 {
			reordered[left], reordered[right] = reordered[right], reordered[left]
		}
		return reordered, nil
	}
	reordered, err := DiscoverPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range reordered.Controllers {
		if firstKeys[item.Product] != item.Key {
			t.Fatalf("key changed after reorder for %q: %q != %q", item.Product, firstKeys[item.Product], item.Key)
		}
	}
	if hashIdentity("test", "ab", "c") == hashIdentity("test", "a", "bc") {
		t.Fatal("length-delimited identity tuples collided")
	}
}

func TestDeliberateDiscoveryUsesOneDeadlineAndReleasesLifecycleGates(t *testing.T) {
	t.Run("preview uses default total bound", func(t *testing.T) {
		_, _ = setupLifecycleTest(t)
		deliberateDiscoveryTimeout = 20 * time.Millisecond
		sentinel := errors.New("preserved preview status")
		openrgb.SetDisconnected(sentinel)
		deadlineSeen := make(chan time.Time, 1)
		statusNeutralDiscover = func(ctx context.Context) ([]openrgb.DiscoveredController, error) {
			deadline, ok := ctx.Deadline()
			if !ok {
				return nil, errors.New("deliberate discovery context has no deadline")
			}
			deadlineSeen <- deadline
			<-ctx.Done()
			return nil, ctx.Err()
		}

		_, err := DiscoverPreview(context.Background())
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("preview timeout error = %v", err)
		}
		select {
		case deadline := <-deadlineSeen:
			if remaining := time.Until(deadline); remaining > 50*time.Millisecond {
				t.Fatalf("preview deadline was not bounded: %v remaining", remaining)
			}
		default:
			t.Fatal("preview discovery did not observe a deadline")
		}
		state, statusErr := openrgb.GetStatus()
		if state != openrgb.StateOffline || statusErr != sentinel {
			t.Fatalf("preview timeout changed SDK status to %q, %v", state, statusErr)
		}
		assertLifecycleGateAvailable(t, deliberateDiscoveryGate)
		assertLifecycleGateAvailable(t, lifecycleMutationGate)
	})

	t.Run("import preserves earlier caller deadline", func(t *testing.T) {
		_, _ = setupLifecycleTest(t)
		deliberateDiscoveryTimeout = time.Second
		controller := lifecycleController("Bounded Import", "bounded-import-external", "", 2)
		identity := chooseControllerIdentities([]openrgb.DiscoveredController{controller})[0]
		key := selectionKeyPrefix + identity.digest
		sentinel := errors.New("preserved import status")
		openrgb.SetDisconnected(sentinel)
		deadlineSeen := make(chan time.Time, 1)
		statusNeutralDiscover = func(ctx context.Context) ([]openrgb.DiscoveredController, error) {
			deadline, ok := ctx.Deadline()
			if !ok {
				return nil, errors.New("deliberate import context has no deadline")
			}
			deadlineSeen <- deadline
			<-ctx.Done()
			return nil, ctx.Err()
		}
		callerCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		callerDeadline, _ := callerCtx.Deadline()
		defer cancel()

		_, err := ImportControllers(callerCtx, []string{key}, newFakeImportRegistry().hooks())
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("import timeout error = %v", err)
		}
		select {
		case deadline := <-deadlineSeen:
			if !deadline.Equal(callerDeadline) {
				t.Fatalf("import deadline = %v, want earlier caller deadline %v", deadline, callerDeadline)
			}
		default:
			t.Fatal("import discovery did not observe a deadline")
		}
		state, statusErr := openrgb.GetStatus()
		if state != openrgb.StateOffline || statusErr != sentinel {
			t.Fatalf("import timeout changed SDK status to %q, %v", state, statusErr)
		}
		assertLifecycleGateAvailable(t, deliberateDiscoveryGate)
		assertLifecycleGateAvailable(t, lifecycleMutationGate)
	})
}

func TestDiscoveryPreviewConfiguredStatesFailureAndTargetConflict(t *testing.T) {
	_, _ = setupLifecycleTest(t)
	enabled := testConfig("openrgb-mobo-1", "Enabled")
	enabled.ExternalSerial = "enabled-external"
	disabled := testConfig("openrgb-hash-legacy-short", "Disabled")
	disabled.ExternalSerial = "disabled-external"
	disabled.Disabled = true
	if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{
		enabled.Serial:  enabled,
		disabled.Serial: disabled,
	}}); err != nil {
		t.Fatal(err)
	}
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return []openrgb.DiscoveredController{
			lifecycleController("Enabled", enabled.ExternalSerial, "", 1),
			lifecycleController("Disabled", disabled.ExternalSerial, "", 1),
		}, nil
	}
	preview, err := DiscoverPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	states := make(map[string]ControllerPreview)
	for _, item := range preview.Controllers {
		states[item.Product] = item
	}
	if states["Enabled"].State != "imported" || states["Enabled"].ConfiguredSerial != enabled.Serial {
		t.Fatalf("enabled preview = %#v", states["Enabled"])
	}
	if states["Disabled"].State != "selectable" || states["Disabled"].ConfiguredSerial != disabled.Serial {
		t.Fatalf("disabled preview = %#v", states["Disabled"])
	}

	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return nil, errors.New("SDK unavailable")
	}
	offline, err := DiscoverPreview(context.Background())
	if err == nil || len(offline.Configured) != 2 || offline.Error == "" {
		t.Fatalf("offline preview = %#v, %v", offline, err)
	}

	var dialed atomic.Bool
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		dialed.Store(true)
		return nil, nil
	}
	localTargetServerEnabled = func() bool { return true }
	conflict, err := DiscoverPreview(context.Background())
	if err == nil || conflict.DiscoveryState != "conflict" || dialed.Load() {
		t.Fatalf("target conflict = %#v, %v, dialed=%v", conflict, err, dialed.Load())
	}
	registry := newFakeImportRegistry()
	if _, err = ImportControllers(
		context.Background(),
		[]string{selectionKeyPrefix + strings.Repeat("a", 64)},
		registry.hooks(),
	); err == nil || dialed.Load() {
		t.Fatalf("target-conflicting import error=%v, dialed=%v", err, dialed.Load())
	}
	if err = RefreshManager(context.Background()); err == nil {
		t.Fatal("target-conflicting refresh unexpectedly succeeded")
	}
	activeManagerMutex.RLock()
	manager := activeManager
	activeManagerMutex.RUnlock()
	if manager != nil {
		t.Fatal("target-conflicting refresh started a manager")
	}
}

func TestStoreIdentityMatchesSelectedKindAndLegacyFallback(t *testing.T) {
	externalController := lifecycleController("Current Product", " External-123 ", "usb:new", 2)
	externalIdentity := chooseControllerIdentities([]openrgb.DiscoveredController{externalController})[0]
	externalConfig := testConfig("openrgb-external", "Old Product")
	externalConfig.ExternalSerial = "external-123"
	externalConfig.Location = "usb:old"
	externalConfig.Vendor = "Old Vendor"
	if !storeIdentityMatches(externalConfig, externalConfig.Serial, externalIdentity, externalController) {
		t.Fatal("matching external serial was overridden by presentation drift")
	}
	externalConfig.ExternalSerial = "different-external"
	if storeIdentityMatches(externalConfig, externalConfig.Serial, externalIdentity, externalController) {
		t.Fatal("different usable external serial matched")
	}

	locationController := lifecycleController("Location Product", "unknown", "usb:location", 2)
	locationIdentity := chooseControllerIdentities([]openrgb.DiscoveredController{locationController})[0]
	locationConfig := testConfig("openrgb-location", locationController.Name)
	locationConfig.ExternalSerial = "stale-optional-serial"
	locationConfig.Location = locationController.Location
	locationConfig.Vendor = locationController.Vendor
	if !storeIdentityMatches(locationConfig, locationConfig.Serial, locationIdentity, locationController) {
		t.Fatal("stale optional serial overrode the selected location tuple")
	}

	metadataController := lifecycleController("Metadata Product", "none", "", 2)
	metadataIdentity := chooseControllerIdentities([]openrgb.DiscoveredController{metadataController})[0]
	metadataConfig := testConfig("openrgb-metadata", metadataController.Name)
	metadataConfig.ExternalSerial = "stale-optional-serial"
	metadataConfig.Location = "stale-location"
	metadataConfig.Vendor = metadataController.Vendor
	if !storeIdentityMatches(metadataConfig, metadataConfig.Serial, metadataIdentity, metadataController) {
		t.Fatal("stale optional fields overrode the selected metadata tuple")
	}

	legacyController := lifecycleController("Legacy Missing Optional", "unknown", "", 2)
	legacySerial := internalSerialForController(legacyController)
	legacyConfig := testConfig(legacySerial, "")
	legacyConfig.ExternalSerial = "stored-but-currently-missing"
	legacyConfig.Location = "stored-but-currently-missing"
	legacyConfig.Vendor = legacyController.Vendor
	if legacyIdentityMetadataConflicts(legacyConfig, legacyController) {
		t.Fatal("missing current optional metadata was treated as contradictory")
	}
	if !storeIdentityMatches(legacyConfig, legacySerial, controllerIdentity{}, legacyController) {
		t.Fatal("safe exact legacy serial fallback was not recognized")
	}

	differentExternal := legacyController
	differentExternal.Serial = "discovered-external"
	legacyConfig.ExternalSerial = "stored-external"
	if !legacyIdentityMetadataConflicts(legacyConfig, differentExternal) {
		t.Fatal("different usable external serials were not contradictory")
	}
	if storeIdentityMatches(legacyConfig, legacySerial, controllerIdentity{}, differentExternal) {
		t.Fatal("different usable external serials allowed unsafe legacy reuse")
	}

	motherboard := openrgb.DiscoveredController{
		Name:     "ASUS ROG Strix Z890-E Gaming WiFi",
		Vendor:   "ASUS Aura",
		Serial:   "none",
		LEDCount: 1,
		Zones:    []openrgb.DiscoveredZone{{Name: "Aura", LEDCount: 1}},
	}
	motherboardConfig := testConfig("openrgb-mobo-1", "Imported ASUS Motherboard")
	motherboardConfig.Vendor = motherboard.Vendor
	if !storeIdentityMatches(motherboardConfig, motherboardConfig.Serial, controllerIdentity{}, motherboard) {
		t.Fatal("legacy openrgb-mobo-1 was not recognized across product presentation drift")
	}
}

func TestFullExternalIdentityPresentationDriftKeepsConfiguredState(t *testing.T) {
	tests := []struct {
		name      string
		disabled  bool
		wantState string
	}{
		{name: "enabled", wantState: "imported"},
		{name: "disabled", disabled: true, wantState: "selectable"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _ = setupLifecycleTest(t)
			controller := lifecycleController("Current Product", "stable-external", "usb:current", 2)
			identity := chooseControllerIdentities([]openrgb.DiscoveredController{controller})[0]
			serial := internalKeyPrefix + identity.digest
			stored := testConfig(serial, "Old Product")
			stored.Disabled = test.disabled
			stored.ExternalSerial = " STABLE-EXTERNAL "
			stored.Location = "usb:old"
			stored.Vendor = "Old Vendor"
			if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{serial: stored}}); err != nil {
				t.Fatal(err)
			}
			statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
				return []openrgb.DiscoveredController{controller}, nil
			}

			preview, err := DiscoverPreview(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			item := preview.Controllers[0]
			if item.State != test.wantState || item.ConfiguredSerial != serial {
				t.Fatalf("presentation-drift preview = %#v, want state=%q serial=%q", item, test.wantState, serial)
			}
			if test.disabled && item.Key != selectionKeyPrefix+identity.digest {
				t.Fatalf("disabled full identity key = %q, want exact identity key", item.Key)
			}
		})
	}
}

func TestExternalIdentityReimportRefreshesPresentationWithoutDuplicates(t *testing.T) {
	_, root := setupLifecycleTest(t)
	registry := newFakeImportRegistry()
	controller := lifecycleController("Refreshed Product", "stable-reimport-external", "usb:refreshed", 3)
	controller.Vendor = "Refreshed Vendor"
	identity := chooseControllerIdentities([]openrgb.DiscoveredController{controller})[0]
	serial := internalKeyPrefix + identity.digest
	stored := testConfig(serial, "Old Product")
	stored.Disabled = true
	stored.ExternalSerial = "STABLE-REIMPORT-EXTERNAL"
	stored.Location = "usb:old"
	stored.Vendor = "Old Vendor"
	if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{serial: stored}}); err != nil {
		t.Fatal(err)
	}

	rgbPath := filepath.Join(root, "database", "rgb", serial+".json")
	profilePath := filepath.Join(root, "database", "profiles", serial+".json")
	if err := os.MkdirAll(filepath.Dir(rgbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
		t.Fatal(err)
	}
	rgbData := []byte(`{"device":"preserved","profiles":{}}`)
	profileData, err := json.Marshal(DeviceProfile{
		Active:     true,
		Serial:     serial,
		RGBProfile: "static",
		RGBCluster: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(rgbPath, rgbData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(profilePath, profileData, 0o644); err != nil {
		t.Fatal(err)
	}

	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return []openrgb.DiscoveredController{controller}, nil
	}
	clusterMembership := make(map[string]*common.ClusterController)
	managerMembership := make(map[string]*Device)
	var clusterAdds atomic.Int32
	var managerAdds atomic.Int32
	addLifecycleCluster = func(controller *common.ClusterController) error {
		clusterAdds.Add(1)
		clusterMembership[controller.Serial] = controller
		return nil
	}
	addLifecycleManager = func(_ context.Context, devices map[string]*Device) (bool, error) {
		managerAdds.Add(1)
		for deviceSerial, device := range devices {
			managerMembership[deviceSerial] = device
		}
		return true, nil
	}

	key := selectionKeyPrefix + identity.digest
	imported, err := ImportControllers(context.Background(), []string{key}, registry.hooks())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(imported.ConfiguredSerials, []string{serial}) {
		t.Fatalf("reimported serials = %#v, want exact preserved serial %q", imported.ConfiguredSerials, serial)
	}
	repeated, err := ImportControllers(context.Background(), []string{key}, registry.hooks())
	if err != nil || !reflect.DeepEqual(repeated.ConfiguredSerials, []string{serial}) {
		t.Fatalf("idempotent reimport = %#v, %v", repeated, err)
	}

	store, err := loadConfigStore()
	if err != nil {
		t.Fatal(err)
	}
	refreshed := store.Devices[serial]
	if len(store.Devices) != 1 || refreshed.Disabled ||
		refreshed.ExternalSerial != controller.Serial ||
		refreshed.Location != controller.Location ||
		refreshed.Product != controller.Name ||
		refreshed.Vendor != controller.Vendor {
		t.Fatalf("refreshed store entry = %#v; full store=%#v", refreshed, store)
	}
	rgbAfter, err := os.ReadFile(rgbPath)
	if err != nil {
		t.Fatal(err)
	}
	profileAfter, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(rgbAfter) != string(rgbData) || string(profileAfter) != string(profileData) {
		t.Fatal("reimport changed preserved RGB/profile artifacts")
	}
	if registry.count() != 1 || enabledConfiguredCount() != 1 ||
		len(clusterMembership) != 1 || clusterAdds.Load() != 1 ||
		len(managerMembership) != 1 || managerAdds.Load() != 1 {
		t.Fatalf(
			"duplicate membership registry=%d configured=%d cluster=%d/%d manager=%d/%d",
			registry.count(),
			enabledConfiguredCount(),
			len(clusterMembership),
			clusterAdds.Load(),
			len(managerMembership),
			managerAdds.Load(),
		)
	}
}

func TestConfiguredExternalIdentityClaimsRemainAmbiguous(t *testing.T) {
	_, _ = setupLifecycleTest(t)
	controller := lifecycleController("Ambiguous Current", "shared-external", "usb:current", 2)
	identity := chooseControllerIdentities([]openrgb.DiscoveredController{controller})[0]
	fullSerial := internalKeyPrefix + identity.digest
	first := testConfig(fullSerial, "First Presentation")
	first.ExternalSerial = controller.Serial
	first.Location = "usb:first"
	second := testConfig("openrgb-hash-legacy-claim", "Second Presentation")
	second.ExternalSerial = controller.Serial
	second.Location = "usb:second"
	if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{
		first.Serial:  first,
		second.Serial: second,
	}}); err != nil {
		t.Fatal(err)
	}
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return []openrgb.DiscoveredController{controller}, nil
	}

	preview, err := DiscoverPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	item := preview.Controllers[0]
	if item.State != "ambiguous" || item.Key != "" || item.ReasonCode != "ambiguous_configured_identity" {
		t.Fatalf("duplicate configured identity preview = %#v", item)
	}
}

func TestLegacyExternalSerialCanonicalizationSurvivesImportRollback(t *testing.T) {
	externalCases := []struct {
		name       string
		stored     string
		discovered string
		want       string
	}{
		{name: "unknown", stored: "unknown", discovered: "canonical-unknown", want: ""},
		{name: "n-a", stored: " n/a ", discovered: "canonical-na", want: ""},
		{name: "padded-usable", stored: " canonical-usable ", discovered: "canonical-usable", want: "canonical-usable"},
	}
	failures := []string{"artifact", "registry", "cluster", "manager"}

	for _, external := range externalCases {
		for _, failure := range failures {
			t.Run(external.name+"/"+failure, func(t *testing.T) {
				storePath, root := setupLifecycleTest(t)
				registry := newFakeImportRegistry()
				controller := lifecycleController("Canonical Rollback", external.discovered, "", 2)
				serial := internalSerialForController(controller)
				cfg := testConfig(serial, controller.Name)
				cfg.Disabled = true
				cfg.ExternalSerial = external.stored
				rawStore, err := json.MarshalIndent(ConfigStore{Devices: map[string]DeviceConfig{serial: cfg}}, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(storePath, rawStore, 0o644); err != nil {
					t.Fatal(err)
				}

				profilePath := filepath.Join(root, "database", "profiles", serial+".json")
				if err = os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
					t.Fatal(err)
				}
				profileData, err := json.Marshal(DeviceProfile{
					Active:     true,
					Serial:     serial,
					RGBProfile: "static",
					RGBCluster: true,
				})
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(profilePath, profileData, 0o644); err != nil {
					t.Fatal(err)
				}

				clusterState := newFakeLifecycleCluster()
				addLifecycleCluster = func(controller *common.ClusterController) error {
					if failure == "cluster" {
						return errors.New("injected cluster activation failure")
					}
					return clusterState.add(controller)
				}
				removeLifecycleCluster = clusterState.remove
				managerMembership := make(map[string]*Device)
				addLifecycleManager = func(_ context.Context, devices map[string]*Device) (bool, error) {
					if failure == "manager" {
						return false, errors.New("injected manager activation failure")
					}
					for deviceSerial, device := range devices {
						managerMembership[deviceSerial] = device
					}
					return false, nil
				}
				removeLifecycleManager = func(_ context.Context, devices map[string]*Device) error {
					for deviceSerial := range devices {
						delete(managerMembership, deviceSerial)
					}
					return nil
				}
				switch failure {
				case "artifact":
					createArtifactFile = func(string, []byte) (bool, error) {
						return false, errors.New("injected artifact activation failure")
					}
				case "registry":
					registry.failRegister = true
				}
				statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
					return []openrgb.DiscoveredController{controller}, nil
				}

				preview, err := DiscoverPreview(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				item := preview.Controllers[0]
				if item.State != "selectable" || item.ConfiguredSerial != serial {
					t.Fatalf("legacy canonical preview = %#v", item)
				}
				_, err = ImportControllers(context.Background(), []string{item.Key}, registry.hooks())
				if err == nil || strings.Contains(err.Error(), "rollback failed") {
					t.Fatalf("injected %s failure error = %v", failure, err)
				}

				store, loadErr := loadConfigStore()
				if loadErr != nil {
					t.Fatal(loadErr)
				}
				restored := store.Devices[serial]
				if len(store.Devices) != 1 || !restored.Disabled || restored.ExternalSerial != external.want {
					t.Fatalf("restored canonical store = %#v", store)
				}
				if registry.count() != 0 || enabledConfiguredCount() != 0 ||
					len(managerMembership) != 0 {
					t.Fatalf(
						"rollback membership registry=%d configured=%d manager=%d",
						registry.count(),
						enabledConfiguredCount(),
						len(managerMembership),
					)
				}
				if controllers, _, _ := clusterState.counts(); controllers != 0 {
					t.Fatalf("rollback left %d cluster controllers", controllers)
				}
				rgbPath := filepath.Join(root, "database", "rgb", serial+".json")
				if _, statErr := os.Stat(rgbPath); !os.IsNotExist(statErr) {
					t.Fatalf("rollback left newly created RGB artifact: %v", statErr)
				}
				profileAfter, readErr := os.ReadFile(profilePath)
				if readErr != nil || string(profileAfter) != string(profileData) {
					t.Fatalf("rollback changed preserved profile: %v", readErr)
				}
			})
		}
	}
}

func TestLegacyExternalSerialCanonicalizationSurvivesRemovalRollback(t *testing.T) {
	tests := []struct {
		name   string
		stored string
		want   string
	}{
		{name: "unknown", stored: "unknown", want: ""},
		{name: "n-a", stored: " n/a ", want: ""},
		{name: "padded-usable", stored: " removal-usable ", want: "removal-usable"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storePath, root := setupLifecycleTest(t)
			registry := newFakeImportRegistry()
			serial := "openrgb-hash-removal-canonical-" + test.name
			cfg := testConfig(serial, "Removal Canonical")
			cfg.ExternalSerial = test.stored
			rawStore, err := json.MarshalIndent(ConfigStore{Devices: map[string]DeviceConfig{serial: cfg}}, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(storePath, rawStore, 0o644); err != nil {
				t.Fatal(err)
			}

			device := testDevice(cfg)
			if err = addConfiguredDevices(map[string]*Device{serial: device}); err != nil {
				t.Fatal(err)
			}
			wrapper := &common.Device{Serial: serial, Product: cfg.Product, Instance: device}
			if err = registry.register(wrapper, device); err != nil {
				t.Fatal(err)
			}
			registry.failRemove = true

			rgbPath := filepath.Join(root, "database", "rgb", serial+".json")
			profilePath := filepath.Join(root, "database", "profiles", serial+".json")
			if err = os.MkdirAll(filepath.Dir(rgbPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err = os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
				t.Fatal(err)
			}
			rgbData := []byte(`{"device":"removal-preserved","profiles":{}}`)
			profileData := []byte(`{"Active":true,"RGBProfile":"static"}`)
			if err = os.WriteFile(rgbPath, rgbData, 0o644); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(profilePath, profileData, 0o644); err != nil {
				t.Fatal(err)
			}

			_, err = RemoveConfiguredImports(context.Background(), []string{serial}, registry.hooks())
			if err == nil || strings.Contains(err.Error(), "rollback failed") {
				t.Fatalf("removal rollback error = %v", err)
			}
			store, loadErr := loadConfigStore()
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			restored := store.Devices[serial]
			if restored.Disabled || restored.ExternalSerial != test.want {
				t.Fatalf("removal rollback store = %#v", restored)
			}
			if current, ok := configuredDevice(serial); !ok || current != device {
				t.Fatal("removal rollback did not restore configured membership")
			}
			if _, current, ok := registry.lookup(serial); !ok || current != device {
				t.Fatal("removal rollback changed registry membership")
			}
			device.SetSpeed("fast")
			if device.GetSpeed() != "fast" {
				t.Fatal("removal rollback did not reactivate exact device")
			}
			rgbAfter, _ := os.ReadFile(rgbPath)
			profileAfter, _ := os.ReadFile(profilePath)
			if string(rgbAfter) != string(rgbData) || string(profileAfter) != string(profileData) {
				t.Fatal("removal rollback changed preserved artifacts")
			}
		})
	}
}

func TestEnabledShortLegacyExternalIdentityIgnoresPresentationDrift(t *testing.T) {
	_, _ = setupLifecycleTest(t)
	controller := lifecycleController("Current Product", "stable-legacy-external", "usb:current", 2)
	legacySerial := "openrgb-hash-legacy-short"
	stored := testConfig(legacySerial, "Old Product")
	stored.ExternalSerial = controller.Serial
	stored.Location = "usb:old"
	stored.Vendor = "Old Vendor"
	if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{legacySerial: stored}}); err != nil {
		t.Fatal(err)
	}
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return []openrgb.DiscoveredController{controller}, nil
	}

	preview, err := DiscoverPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	item := preview.Controllers[0]
	if item.State != "imported" || item.ConfiguredSerial != legacySerial || item.Key == "" {
		t.Fatalf("legacy presentation-drift preview = %#v", item)
	}
}

func TestDisabledLegacySerialsAndArtifactsAreReused(t *testing.T) {
	tests := []struct {
		name       string
		controller openrgb.DiscoveredController
		wantSerial string
	}{
		{
			name: "motherboard",
			controller: openrgb.DiscoveredController{
				Name:        "ASUS ROG Strix Z890-E Gaming WiFi",
				Vendor:      "ASUS Aura",
				Description: "Motherboard",
				Version:     "1.0",
				Serial:      "none",
				LEDCount:    1,
				Zones:       []openrgb.DiscoveredZone{{Name: "Aura", LEDCount: 1}},
			},
			wantSerial: "openrgb-mobo-1",
		},
		{
			name:       "shortened hash",
			controller: lifecycleController("Legacy Short Hash", "unknown", "", 2),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, root := setupLifecycleTest(t)
			registry := newFakeImportRegistry()
			serial := test.wantSerial
			if serial == "" {
				serial = internalSerialForController(test.controller)
			}
			cfg := testConfig(serial, test.controller.Name)
			cfg.Disabled = true
			cfg.ExternalSerial = ""
			cfg.Location = ""
			cfg.Vendor = ""
			if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{serial: cfg}}); err != nil {
				t.Fatal(err)
			}

			rgbPath := filepath.Join(root, "database", "rgb", serial+".json")
			profilePath := filepath.Join(root, "database", "profiles", serial+".json")
			if err := os.MkdirAll(filepath.Dir(rgbPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
				t.Fatal(err)
			}
			rgbData := []byte(`{"device":"legacy-preserved","profiles":{}}`)
			profileData := []byte(`{"Active":true,"RGBProfile":"static"}`)
			if err := os.WriteFile(rgbPath, rgbData, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(profilePath, profileData, 0o644); err != nil {
				t.Fatal(err)
			}

			statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
				return []openrgb.DiscoveredController{test.controller}, nil
			}
			preview, err := DiscoverPreview(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			item := preview.Controllers[0]
			if item.State != "selectable" || item.ConfiguredSerial != serial || item.Key == "" {
				t.Fatalf("legacy preview = %#v", item)
			}
			imported, err := ImportControllers(context.Background(), []string{item.Key}, registry.hooks())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(imported.ConfiguredSerials, []string{serial}) {
				t.Fatalf("legacy serial was replaced: %#v", imported.ConfiguredSerials)
			}
			rgbAfter, _ := os.ReadFile(rgbPath)
			profileAfter, _ := os.ReadFile(profilePath)
			if string(rgbAfter) != string(rgbData) || string(profileAfter) != string(profileData) {
				t.Fatal("legacy RGB/profile artifacts were overwritten")
			}
			store, err := loadConfigStore()
			if err != nil {
				t.Fatal(err)
			}
			if store.Devices[serial].Disabled {
				t.Fatal("legacy entry was not re-enabled")
			}
		})
	}
}

func TestConflictingFullSerialIsInvalidWithoutMutation(t *testing.T) {
	storePath, root := setupLifecycleTest(t)
	registry := newFakeImportRegistry()
	controller := lifecycleController("Collision Candidate", "collision-external", "", 2)
	identity := chooseControllerIdentities([]openrgb.DiscoveredController{controller})[0]
	fullSerial := internalKeyPrefix + identity.digest
	conflicting := testConfig(fullSerial, "Different Product")
	conflicting.Disabled = true
	conflicting.ExternalSerial = "different-external"
	conflicting.Vendor = "Different Vendor"
	if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{fullSerial: conflicting}}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatal(err)
	}
	var managerCalls atomic.Int32
	var clusterCalls atomic.Int32
	addLifecycleManager = func(context.Context, map[string]*Device) (bool, error) {
		managerCalls.Add(1)
		return false, nil
	}
	addLifecycleCluster = func(*common.ClusterController) error {
		clusterCalls.Add(1)
		return nil
	}
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return []openrgb.DiscoveredController{controller}, nil
	}

	preview, err := DiscoverPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	item := preview.Controllers[0]
	if item.State != "invalid" || item.Key != "" || item.ReasonCode != "internal_serial_collision" {
		t.Fatalf("collision preview = %#v", item)
	}
	key := selectionKeyPrefix + identity.digest
	if _, err = ImportControllers(context.Background(), []string{key}, registry.hooks()); err == nil {
		t.Fatal("conflicting full serial was importable")
	}
	after, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) || registry.count() != 0 ||
		enabledConfiguredCount() != 0 || managerCalls.Load() != 0 || clusterCalls.Load() != 0 {
		t.Fatalf("collision mutated state: storeChanged=%v registry=%d configured=%d manager=%d cluster=%d",
			string(after) != string(before), registry.count(), enabledConfiguredCount(), managerCalls.Load(), clusterCalls.Load())
	}
	for _, path := range []string{
		filepath.Join(root, "database", "rgb", fullSerial+".json"),
		filepath.Join(root, "database", "profiles", fullSerial+".json"),
	} {
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("collision created artifact %q: %v", path, statErr)
		}
	}
}

func TestImportRemoveAndExactDisabledReimport(t *testing.T) {
	_, root := setupLifecycleTest(t)
	registry := newFakeImportRegistry()
	controllers := []openrgb.DiscoveredController{
		lifecycleController("Selected", "selected-external", "", 4),
		lifecycleController("Unselected", "unselected-external", "", 3),
	}
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return append([]openrgb.DiscoveredController(nil), controllers...), nil
	}
	preview, err := DiscoverPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	key := preview.Controllers[0].Key
	var managerAdds atomic.Int32
	addLifecycleManager = func(context.Context, map[string]*Device) (bool, error) {
		managerAdds.Add(1)
		return managerAdds.Load() == 1, nil
	}

	imported, err := ImportControllers(context.Background(), []string{key, key}, registry.hooks())
	if err != nil {
		t.Fatal(err)
	}
	if len(imported.ConfiguredSerials) != 1 || registry.count() != 1 || enabledConfiguredCount() != 1 {
		t.Fatalf("import result = %#v, registry=%d configured=%d", imported, registry.count(), enabledConfiguredCount())
	}
	serial := imported.ConfiguredSerials[0]
	if !strings.HasPrefix(serial, internalKeyPrefix) || len(serial) != len(internalKeyPrefix)+64 {
		t.Fatalf("internal serial = %q", serial)
	}
	store, err := loadConfigStore()
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Devices) != 1 || store.Devices[serial].Disabled {
		t.Fatalf("store after subset import = %#v", store)
	}
	rgbPath := filepath.Join(root, "database", "rgb", serial+".json")
	profilePath := filepath.Join(root, "database", "profiles", serial+".json")
	rgbBefore, err := os.ReadFile(rgbPath)
	if err != nil {
		t.Fatal(err)
	}
	profileBefore, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}

	repeated, err := ImportControllers(context.Background(), []string{key}, registry.hooks())
	if err != nil || !reflect.DeepEqual(repeated.ConfiguredSerials, []string{serial}) || managerAdds.Load() != 1 {
		t.Fatalf("idempotent import = %#v, %v, manager adds=%d", repeated, err, managerAdds.Load())
	}

	removed, err := RemoveConfiguredImports(context.Background(), []string{serial}, registry.hooks())
	if err != nil || !reflect.DeepEqual(removed.RemovedSerials, []string{serial}) {
		t.Fatalf("remove = %#v, %v", removed, err)
	}
	store, err = loadConfigStore()
	if err != nil {
		t.Fatal(err)
	}
	if !store.Devices[serial].Disabled || registry.count() != 0 || enabledConfiguredCount() != 0 {
		t.Fatalf("removed membership store=%#v registry=%d configured=%d", store, registry.count(), enabledConfiguredCount())
	}
	rgbAfter, _ := os.ReadFile(rgbPath)
	profileAfter, _ := os.ReadFile(profilePath)
	if string(rgbAfter) != string(rgbBefore) || string(profileAfter) != string(profileBefore) {
		t.Fatal("removal changed preserved RGB/profile artifacts")
	}

	reimported, err := ImportControllers(context.Background(), []string{key}, registry.hooks())
	if err != nil || !reflect.DeepEqual(reimported.ConfiguredSerials, []string{serial}) {
		t.Fatalf("reimport = %#v, %v", reimported, err)
	}
}

func TestImportRollbackFailureSeamsAndPreservedFiles(t *testing.T) {
	tests := []struct {
		name   string
		inject func(*testing.T, *fakeImportRegistry, string, string)
	}{
		{
			name: "store",
			inject: func(t *testing.T, _ *fakeImportRegistry, _, _ string) {
				previous := renameConfigStore
				renameConfigStore = func(string, string) error { return errors.New("injected store failure") }
				t.Cleanup(func() { renameConfigStore = previous })
			},
		},
		{
			name: "artifact",
			inject: func(t *testing.T, _ *fakeImportRegistry, _, _ string) {
				createArtifactFile = func(string, []byte) (bool, error) {
					return false, errors.New("injected artifact failure")
				}
			},
		},
		{
			name: "registry",
			inject: func(_ *testing.T, registry *fakeImportRegistry, _, _ string) {
				registry.failRegister = true
			},
		},
		{
			name: "cluster",
			inject: func(t *testing.T, _ *fakeImportRegistry, serial, root string) {
				profilePath := filepath.Join(root, "database", "profiles", serial+".json")
				if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
					t.Fatal(err)
				}
				profile := DeviceProfile{Active: true, Serial: serial, RGBProfile: "static", RGBCluster: true}
				data, _ := json.Marshal(profile)
				if err := os.WriteFile(profilePath, data, 0o644); err != nil {
					t.Fatal(err)
				}
				addLifecycleCluster = func(*common.ClusterController) error {
					return errors.New("injected cluster failure")
				}
			},
		},
		{
			name: "manager",
			inject: func(_ *testing.T, _ *fakeImportRegistry, serial, root string) {
				rgbPath := filepath.Join(root, "database", "rgb", serial+".json")
				profilePath := filepath.Join(root, "database", "profiles", serial+".json")
				_ = os.MkdirAll(filepath.Dir(rgbPath), 0o755)
				_ = os.MkdirAll(filepath.Dir(profilePath), 0o755)
				rgbState := `{"device":"preserved","profiles":{}}`
				profileState := `{"Active":true,"RGBProfile":"static"}`
				_ = os.WriteFile(rgbPath, []byte(rgbState), 0o644)
				_ = os.WriteFile(profilePath, []byte(profileState), 0o644)
				addLifecycleManager = func(context.Context, map[string]*Device) (bool, error) {
					return false, errors.New("injected manager failure")
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storePath, root := setupLifecycleTest(t)
			registry := newFakeImportRegistry()
			controller := lifecycleController("Rollback", "rollback-"+test.name, "", 2)
			statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
				return []openrgb.DiscoveredController{controller}, nil
			}
			preview, err := DiscoverPreview(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			key := preview.Controllers[0].Key
			serial := internalKeyPrefix + strings.TrimPrefix(key, selectionKeyPrefix)
			test.inject(t, registry, serial, root)
			before, err := os.ReadFile(storePath)
			if err != nil {
				t.Fatal(err)
			}
			_, err = ImportControllers(context.Background(), []string{key}, registry.hooks())
			if err == nil {
				t.Fatal("expected injected import failure")
			}
			after, readErr := os.ReadFile(storePath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(after) != string(before) {
				t.Fatalf("failed import did not restore exact store\nbefore=%s\nafter=%s", before, after)
			}
			if registry.count() != 0 || enabledConfiguredCount() != 0 {
				t.Fatalf("rollback membership registry=%d configured=%d", registry.count(), enabledConfiguredCount())
			}
			if test.name == "manager" {
				rgbData, _ := os.ReadFile(filepath.Join(root, "database", "rgb", serial+".json"))
				profileData, _ := os.ReadFile(filepath.Join(root, "database", "profiles", serial+".json"))
				if string(rgbData) != `{"device":"preserved","profiles":{}}` ||
					string(profileData) != `{"Active":true,"RGBProfile":"static"}` {
					t.Fatal("rollback changed or deleted pre-existing artifacts")
				}
			}
		})
	}
}

func TestImportRejectsFabricatedAmbiguousAndStaleKeys(t *testing.T) {
	_, _ = setupLifecycleTest(t)
	registry := newFakeImportRegistry()
	duplicate := lifecycleController("Duplicate", "unknown", "", 1)
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return []openrgb.DiscoveredController{duplicate, duplicate}, nil
	}
	fabricated := selectionKeyPrefix + strings.Repeat("a", 64)
	if _, err := ImportControllers(context.Background(), []string{fabricated}, registry.hooks()); err == nil {
		t.Fatal("fabricated ambiguous key was accepted")
	}

	controller := lifecycleController("Stale", "stale-external", "", 1)
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return []openrgb.DiscoveredController{controller}, nil
	}
	preview, err := DiscoverPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return nil, nil
	}
	if _, err = ImportControllers(context.Background(), []string{preview.Controllers[0].Key}, registry.hooks()); err == nil {
		t.Fatal("stale key was accepted")
	}
}

func TestRemovalRollbackRestoresExactMembershipAndTargetModeAllowsRemoval(t *testing.T) {
	_, _ = setupLifecycleTest(t)
	registry := newFakeImportRegistry()
	controller := lifecycleController("Removal", "removal-external", "", 2)
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return []openrgb.DiscoveredController{controller}, nil
	}
	preview, err := DiscoverPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	imported, err := ImportControllers(context.Background(), []string{preview.Controllers[0].Key}, registry.hooks())
	if err != nil {
		t.Fatal(err)
	}
	serial := imported.ConfiguredSerials[0]
	expected, ok := configuredDevice(serial)
	if !ok {
		t.Fatal("imported object is not configured")
	}
	expected.mu.Lock()
	expected.DeviceProfile.RGBCluster = true
	expected.controllerId = 7
	clusterCallback := expected.clusterControllerLocked()
	expected.mu.Unlock()
	var clusterHookCalls atomic.Int32
	var clusterSends atomic.Int32
	previousClusterSend := sendClusterFrame
	sendClusterFrame = func(net.Conn, uint32, []byte) (net.Conn, error) {
		clusterSends.Add(1)
		return nil, nil
	}
	t.Cleanup(func() { sendClusterFrame = previousClusterSend })
	removeLifecycleCluster = func(string) error {
		clusterHookCalls.Add(1)
		if expected.ControllerID() != -1 {
			t.Errorf("cluster removal observed controller ID %d, want -1", expected.ControllerID())
		}
		clusterCallback.WriteColorEx([]byte{255, 0, 0}, 0)
		return nil
	}

	registry.failRemove = true
	if _, err = RemoveConfiguredImports(context.Background(), []string{serial}, registry.hooks()); err == nil {
		t.Fatal("expected injected registry removal failure")
	}
	restored, ok := configuredDevice(serial)
	_, registryInstance, registered := registry.lookup(serial)
	store, loadErr := loadConfigStore()
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if !ok || restored != expected || !registered || registryInstance != expected || store.Devices[serial].Disabled {
		t.Fatalf("rollback did not restore exact object: configured=%p registry=%p store=%#v", restored, registryInstance, store.Devices[serial])
	}
	if clusterHookCalls.Load() != 1 || clusterSends.Load() != 0 {
		t.Fatalf("cluster detach hook calls=%d SDK sends=%d", clusterHookCalls.Load(), clusterSends.Load())
	}
	restored.SetSpeed("fast")
	if restored.GetSpeed() != "fast" {
		t.Fatal("rolled-back exact object remained immutable")
	}
	if restored.SaveDeviceProfile("", false) != 1 {
		t.Fatal("rolled-back exact object could not persist a normal profile mutation")
	}
	restored.bindController(openrgb.DiscoveredController{ID: 42, Name: restored.Product})
	if restored.ControllerID() != 42 {
		t.Fatal("rolled-back exact object could not be rebound for reconciliation")
	}

	registry.failRemove = false
	localTargetServerEnabled = func() bool { return true }
	removed, err := RemoveConfiguredImports(context.Background(), []string{serial}, registry.hooks())
	if err != nil || !reflect.DeepEqual(removed.RemovedSerials, []string{serial}) {
		t.Fatalf("target-mode removal = %#v, %v", removed, err)
	}
}

func TestCanceledRefreshAndRemovalDoNotStartOrMutate(t *testing.T) {
	t.Run("refresh", func(t *testing.T) {
		_, _ = setupLifecycleTest(t)
		cfg := testConfig("openrgb-canceled-refresh", "Canceled Refresh")
		if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{cfg.Serial: cfg}}); err != nil {
			t.Fatal(err)
		}
		device := testDevice(cfg)
		if err := addConfiguredDevices(map[string]*Device{cfg.Serial: device}); err != nil {
			t.Fatal(err)
		}
		previousFactory := managerFactory
		var factories atomic.Int32
		managerFactory = func(devices map[string]*Device, update availabilityUpdater) *Manager {
			factories.Add(1)
			return newManager(devices, update)
		}
		t.Cleanup(func() { managerFactory = previousFactory })
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if err := RefreshManager(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled refresh error = %v", err)
		}
		activeManagerMutex.RLock()
		manager := activeManager
		activeManagerMutex.RUnlock()
		if factories.Load() != 0 || manager != nil {
			t.Fatalf("canceled refresh created %d managers; active=%p", factories.Load(), manager)
		}
	})

	t.Run("removal", func(t *testing.T) {
		storePath, root := setupLifecycleTest(t)
		cfg := testConfig("openrgb-canceled-removal", "Canceled Removal")
		if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{cfg.Serial: cfg}}); err != nil {
			t.Fatal(err)
		}
		device := testDevice(cfg)
		if err := addConfiguredDevices(map[string]*Device{cfg.Serial: device}); err != nil {
			t.Fatal(err)
		}
		registry := newFakeImportRegistry()
		wrapper := &common.Device{Serial: cfg.Serial, Product: cfg.Product, Instance: device}
		if err := registry.register(wrapper, device); err != nil {
			t.Fatal(err)
		}

		rgbPath := filepath.Join(root, "database", "rgb", cfg.Serial+".json")
		profilePath := filepath.Join(root, "database", "profiles", cfg.Serial+".json")
		if err := os.MkdirAll(filepath.Dir(rgbPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
			t.Fatal(err)
		}
		rgbData := []byte(`{"device":"canceled","profiles":{}}`)
		profileData := []byte(`{"Active":true,"RGBProfile":"static"}`)
		if err := os.WriteFile(rgbPath, rgbData, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(profilePath, profileData, 0o644); err != nil {
			t.Fatal(err)
		}
		storeBefore, err := os.ReadFile(storePath)
		if err != nil {
			t.Fatal(err)
		}
		var clusterCalls atomic.Int32
		var managerCalls atomic.Int32
		removeLifecycleCluster = func(string) error {
			clusterCalls.Add(1)
			return nil
		}
		removeLifecycleManager = func(context.Context, map[string]*Device) error {
			managerCalls.Add(1)
			return nil
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if _, err = RemoveConfiguredImports(ctx, []string{cfg.Serial}, registry.hooks()); !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled removal error = %v", err)
		}
		storeAfter, err := os.ReadFile(storePath)
		if err != nil {
			t.Fatal(err)
		}
		rgbAfter, _ := os.ReadFile(rgbPath)
		profileAfter, _ := os.ReadFile(profilePath)
		if string(storeAfter) != string(storeBefore) ||
			string(rgbAfter) != string(rgbData) ||
			string(profileAfter) != string(profileData) {
			t.Fatal("canceled removal changed store or artifacts")
		}
		if registry.count() != 1 || enabledConfiguredCount() != 1 ||
			clusterCalls.Load() != 0 || managerCalls.Load() != 0 {
			t.Fatalf(
				"canceled removal membership registry=%d configured=%d cluster=%d manager=%d",
				registry.count(),
				enabledConfiguredCount(),
				clusterCalls.Load(),
				managerCalls.Load(),
			)
		}
		device.mu.Lock()
		detached := device.lifecycleDetached
		device.mu.Unlock()
		if detached {
			t.Fatal("canceled removal detached the live object")
		}
		assertLifecycleGateAvailable(t, lifecycleMutationGate)
	})
}

func TestDetachForRemovalDisablesOutputAndPreservesDesiredState(t *testing.T) {
	cfg := testConfig("openrgb-detach", "Detach")
	device := testDevice(cfg)
	brightness := uint8(73)
	profile := &DeviceProfile{
		Active:           true,
		RGBProfile:       "rainbow",
		BrightnessSlider: &brightness,
		RGBCluster:       true,
		ZoneColors:       buildZoneColorsFromConfig(&cfg, []byte{1, 2, 3}),
	}
	rgbState := &rgb.RGB{Device: "Detach", Profiles: map[string]rgb.Profile{"rainbow": {ProfileName: "rainbow"}}}
	userProfiles := map[string]*DeviceProfile{"default": profile}
	client, server := net.Pipe()
	defer server.Close()
	if err := server.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	device.mu.Lock()
	device.controllerId = 17
	device.openrgbConn = client
	device.DeviceProfile = profile
	device.Rgb = rgbState
	device.UserProfiles = userProfiles
	device.effect = "rainbow"
	device.brightness = brightness
	device.mu.Unlock()

	previousSend := sendClusterFrame
	var sends atomic.Int32
	sendClusterFrame = func(net.Conn, uint32, []byte) (net.Conn, error) {
		sends.Add(1)
		return nil, nil
	}
	t.Cleanup(func() { sendClusterFrame = previousSend })

	clusterController, err := device.detachForRemoval(func(string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if clusterController == nil {
		t.Fatal("detach did not capture cluster controller for rollback")
	}
	if device.ControllerID() != -1 {
		t.Fatalf("controller ID after detach = %d, want -1", device.ControllerID())
	}
	device.mu.Lock()
	if device.openrgbConn != nil {
		device.mu.Unlock()
		t.Fatal("persistent connection was not cleared")
	}
	if device.DeviceProfile != profile || device.Config == nil || device.effect != "rainbow" ||
		device.brightness != brightness || device.Rgb != rgbState ||
		!reflect.DeepEqual(device.UserProfiles, userProfiles) {
		device.mu.Unlock()
		t.Fatal("detach changed preserved desired state")
	}
	device.mu.Unlock()

	clusterController.WriteColorEx([]byte{255, 0, 0}, 0)
	if sends.Load() != 0 {
		t.Fatalf("cluster callback emitted %d SDK frames after detach", sends.Load())
	}
	buffer := make([]byte, 1)
	if _, err := server.Read(buffer); err == nil {
		t.Fatal("persistent connection remained open after detach")
	}
}

func TestLifecycleDetachWaitsForInFlightRegistrationAndFinallyUnregisters(t *testing.T) {
	cfg := testConfig("openrgb-cluster-race", "Cluster Race")
	device := testDevice(cfg)
	brightness := uint8(100)
	device.mu.Lock()
	device.controllerId = 23
	device.DeviceProfile = &DeviceProfile{
		Active:           true,
		RGBProfile:       "static",
		BrightnessSlider: &brightness,
		RGBCluster:       true,
		ZoneColors:       buildZoneColorsFromConfig(&cfg, []byte{1, 2, 3}),
	}
	controller := device.clusterControllerLocked()
	device.mu.Unlock()
	if controller == nil {
		t.Fatal("could not capture intended cluster controller")
	}

	previousSend := sendClusterFrame
	var sends atomic.Int32
	sendClusterFrame = func(net.Conn, uint32, []byte) (net.Conn, error) {
		sends.Add(1)
		return nil, nil
	}
	t.Cleanup(func() { sendClusterFrame = previousSend })

	state := newFakeLifecycleCluster()
	paused := make(chan struct{})
	resume := make(chan struct{})
	type registrationResult struct {
		added bool
		err   error
	}
	registrationDone := make(chan registrationResult, 1)
	go func() {
		added, err := device.registerClusterControllerWith(
			controller,
			func(controller *common.ClusterController) error {
				close(paused)
				<-resume
				return state.add(controller)
			},
		)
		registrationDone <- registrationResult{added: added, err: err}
	}()
	select {
	case <-paused:
	case <-time.After(time.Second):
		t.Fatal("cluster registration did not reach the controlled pause")
	}

	type detachResult struct {
		controller *common.ClusterController
		err        error
	}
	detachStarted := make(chan struct{})
	detachDone := make(chan detachResult, 1)
	go func() {
		close(detachStarted)
		captured, err := device.detachForRemoval(func(serial string) error {
			if device.ControllerID() != -1 {
				return fmt.Errorf("cluster unregister observed controller ID %d", device.ControllerID())
			}
			controller.WriteColorEx([]byte{255, 0, 0}, 0)
			return state.remove(serial)
		})
		detachDone <- detachResult{controller: captured, err: err}
	}()
	<-detachStarted
	close(resume)

	var registered registrationResult
	select {
	case registered = <-registrationDone:
	case <-time.After(time.Second):
		t.Fatal("paused cluster registration did not resume")
	}
	if registered.err != nil || !registered.added {
		t.Fatalf("registration result = %#v", registered)
	}
	var detached detachResult
	select {
	case detached = <-detachDone:
	case <-time.After(time.Second):
		t.Fatal("lifecycle detach did not complete after registration")
	}
	if detached.err != nil || detached.controller == nil || detached.controller.Serial != controller.Serial {
		t.Fatalf("detach result = %#v, want controller for %q", detached, controller.Serial)
	}
	if controllers, adds, removes := state.counts(); controllers != 0 || adds != 1 || removes != 1 {
		t.Fatalf("cluster state after final unregister = %d controllers, %d adds, %d removes", controllers, adds, removes)
	}
	if sends.Load() != 0 {
		t.Fatalf("detached cluster callback emitted %d SDK frames", sends.Load())
	}
	if device.ControllerID() != -1 {
		t.Fatalf("controller ID after coordinated detach = %d, want -1", device.ControllerID())
	}
	device.mu.Lock()
	detachedFlag := device.lifecycleDetached
	device.mu.Unlock()
	if !detachedFlag {
		t.Fatal("successful lifecycle detach did not mark the device detached")
	}

	added, err := device.registerClusterControllerWith(controller, state.add)
	if err != nil || added {
		t.Fatalf("post-detach registration = added %v, error %v; want skipped", added, err)
	}
	if controllers, adds, removes := state.counts(); controllers != 0 || adds != 1 || removes != 1 {
		t.Fatalf("post-detach registration changed cluster state: %d controllers, %d adds, %d removes", controllers, adds, removes)
	}
}

func TestLifecycleRemovalRollbackRestoresOneControllerAcrossRepeatedCycles(t *testing.T) {
	cfg := testConfig("openrgb-cluster-rollback", "Cluster Rollback")
	device := testDevice(cfg)
	device.mu.Lock()
	device.DeviceProfile = &DeviceProfile{
		Active:     true,
		RGBProfile: "static",
		RGBCluster: true,
		ZoneColors: buildZoneColorsFromConfig(&cfg, []byte{1, 2, 3}),
	}
	controller := device.clusterControllerLocked()
	device.mu.Unlock()
	if controller == nil {
		t.Fatal("could not capture rollback cluster controller")
	}

	state := newFakeLifecycleCluster()
	added, err := device.registerClusterControllerWith(controller, state.add)
	if err != nil || !added {
		t.Fatalf("initial cluster registration = added %v, error %v", added, err)
	}

	const cycles = 3
	for cycle := 0; cycle < cycles; cycle++ {
		captured, detachErr := device.detachForRemoval(state.remove)
		if detachErr != nil {
			t.Fatal(detachErr)
		}
		if captured == nil || captured.Serial != controller.Serial {
			t.Fatalf("cycle %d captured controller %#v, want serial %q", cycle, captured, controller.Serial)
		}
		if controllers, _, _ := state.counts(); controllers != 0 {
			t.Fatalf("cycle %d detach left %d cluster controllers", cycle, controllers)
		}

		if restoreErr := device.restoreAfterRemoval(captured, state.add); restoreErr != nil {
			t.Fatal(restoreErr)
		}
		device.mu.Lock()
		detached := device.lifecycleDetached
		device.mu.Unlock()
		if detached {
			t.Fatalf("cycle %d rollback did not clear lifecycle-detached state", cycle)
		}
		if active := state.controller(device.Serial); active != captured {
			t.Fatalf("cycle %d restored controller %p, want exact captured callback %p", cycle, active, captured)
		}
		if controllers, adds, removes := state.counts(); controllers != 1 ||
			adds != cycle+2 || removes != cycle+1 {
			t.Fatalf(
				"cycle %d cluster state = %d controllers, %d adds, %d removes",
				cycle,
				controllers,
				adds,
				removes,
			)
		}
	}
}

func TestRemovalRollbackRestoresExactRegistryWrapperPointer(t *testing.T) {
	_, _ = setupLifecycleTest(t)
	registry := newFakeImportRegistry()
	controllers := []openrgb.DiscoveredController{
		lifecycleController("Wrapper One", "wrapper-one", "", 1),
		lifecycleController("Wrapper Two", "wrapper-two", "", 1),
	}
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return append([]openrgb.DiscoveredController(nil), controllers...), nil
	}
	preview, err := DiscoverPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	keys := []string{preview.Controllers[0].Key, preview.Controllers[1].Key}
	imported, err := ImportControllers(context.Background(), keys, registry.hooks())
	if err != nil {
		t.Fatal(err)
	}
	serials := imported.ConfiguredSerials
	firstWrapper := registry.wrapper(serials[0])
	secondWrapper := registry.wrapper(serials[1])
	firstDevice, ok := configuredDevice(serials[0])
	if !ok {
		t.Fatal("first imported object is missing")
	}
	firstWrapper.Product = "Live Product"
	firstWrapper.Firmware = "9.9"
	firstWrapper.Image = "live-image.svg"
	firstWrapper.Unavailable = false
	firstWrapper.Instance = firstDevice
	registry.failRemoveSerial = serials[1]

	if _, err = RemoveConfiguredImports(context.Background(), serials, registry.hooks()); err == nil {
		t.Fatal("expected second registry removal failure")
	}
	restored := registry.wrapper(serials[0])
	if restored != firstWrapper {
		t.Fatalf("rollback wrapper pointer = %p, want exact removed pointer %p", restored, firstWrapper)
	}
	if restored.Product != "Live Product" || restored.Firmware != "9.9" ||
		restored.Image != "live-image.svg" || restored.Unavailable ||
		restored.Instance != firstDevice {
		t.Fatalf("rollback changed live wrapper fields: %#v", restored)
	}
	if registry.wrapper(serials[1]) != secondWrapper {
		t.Fatal("wrapper that failed compare-and-delete was replaced")
	}
}

func TestActivatingPublishedImportRejectsArtifactMutations(t *testing.T) {
	_, root := setupLifecycleTest(t)
	registry := newFakeImportRegistry()
	controller := lifecycleController("Activating", "activating-external", "", 2)
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return []openrgb.DiscoveredController{controller}, nil
	}
	preview, err := DiscoverPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	serial := internalKeyPrefix + strings.TrimPrefix(preview.Controllers[0].Key, selectionKeyPrefix)
	managerEntered := make(chan struct{})
	managerRelease := make(chan struct{})
	addLifecycleManager = func(context.Context, map[string]*Device) (bool, error) {
		close(managerEntered)
		<-managerRelease
		return false, errors.New("injected manager activation failure")
	}

	importDone := make(chan error, 1)
	go func() {
		_, importErr := ImportControllers(context.Background(), []string{preview.Controllers[0].Key}, registry.hooks())
		importDone <- importErr
	}()
	select {
	case <-managerEntered:
	case <-time.After(time.Second):
		t.Fatal("import did not reach paused manager activation")
	}

	_, exposed, registered := registry.lookup(serial)
	if !registered || exposed == nil {
		t.Fatal("prepared wrapper was not available at the controlled activation point")
	}
	exposed.mu.Lock()
	activating := exposed.lifecycleActivating
	exposed.mu.Unlock()
	if !activating {
		t.Fatal("prepared wrapper was publicly mutable before transaction commit")
	}

	rgbPath := filepath.Join(root, "database", "rgb", serial+".json")
	profilePath := filepath.Join(root, "database", "profiles", serial+".json")
	rgbBefore, err := os.ReadFile(rgbPath)
	if err != nil {
		t.Fatal(err)
	}
	profileBefore, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	cfg := testConfig(serial, controller.Name)
	if err = exposed.SaveDeviceConfig(&cfg); err == nil || !strings.Contains(err.Error(), "activating") {
		t.Fatalf("activating SaveDeviceConfig error = %v", err)
	}
	if exposed.SaveDeviceProfile("", false) != 0 ||
		exposed.SaveUserProfile("stale") != 0 ||
		exposed.ProcessSetRgbCluster(true) != 0 ||
		exposed.UpdateRgbProfileData("static", rgb.Profile{ProfileName: "static"}) != 0 {
		t.Fatal("activating device accepted a profile, RGB, or cluster mutation")
	}
	rgbDuring, err := os.ReadFile(rgbPath)
	if err != nil {
		t.Fatal(err)
	}
	profileDuring, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(rgbDuring) != string(rgbBefore) || string(profileDuring) != string(profileBefore) {
		t.Fatal("activating device changed transaction-created artifacts")
	}
	if _, statErr := os.Stat(filepath.Join(root, "database", "profiles", serial+"-stale.json")); !os.IsNotExist(statErr) {
		t.Fatalf("activating device created a user profile: %v", statErr)
	}

	close(managerRelease)
	select {
	case err = <-importDone:
	case <-time.After(time.Second):
		t.Fatal("failed import remained blocked after manager release")
	}
	if err == nil || strings.Contains(err.Error(), "rollback failed") {
		t.Fatalf("failed import error = %v", err)
	}
	if registry.count() != 0 || enabledConfiguredCount() != 0 {
		t.Fatalf("failed activating import left registry=%d configured=%d", registry.count(), enabledConfiguredCount())
	}
	if _, statErr := os.Stat(rgbPath); !os.IsNotExist(statErr) {
		t.Fatalf("failed import left RGB artifact: %v", statErr)
	}
	if _, statErr := os.Stat(profilePath); !os.IsNotExist(statErr) {
		t.Fatalf("failed import left profile artifact: %v", statErr)
	}
}

func TestDetachedStaleObjectCannotMutateDisabledOrReimportedState(t *testing.T) {
	_, root := setupLifecycleTest(t)
	registry := newFakeImportRegistry()
	controller := lifecycleController("Stale Pointer", "stale-pointer-external", "", 2)
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return []openrgb.DiscoveredController{controller}, nil
	}

	managerMembership := make(map[string]*Device)
	var managerAdds atomic.Int32
	var managerRemoves atomic.Int32
	addLifecycleManager = func(_ context.Context, devices map[string]*Device) (bool, error) {
		managerAdds.Add(1)
		for serial, device := range devices {
			managerMembership[serial] = device
		}
		return false, nil
	}
	removeLifecycleManager = func(_ context.Context, devices map[string]*Device) error {
		managerRemoves.Add(1)
		for serial := range devices {
			delete(managerMembership, serial)
		}
		return nil
	}
	clusterState := newFakeLifecycleCluster()
	addLifecycleCluster = clusterState.add
	removeLifecycleCluster = clusterState.remove

	preview, err := DiscoverPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	key := preview.Controllers[0].Key
	imported, err := ImportControllers(context.Background(), []string{key}, registry.hooks())
	if err != nil {
		t.Fatal(err)
	}
	serial := imported.ConfiguredSerials[0]
	oldDevice, ok := configuredDevice(serial)
	if !ok {
		t.Fatal("initial import is not configured")
	}

	if _, err = RemoveConfiguredImports(context.Background(), []string{serial}, registry.hooks()); err != nil {
		t.Fatal(err)
	}
	disabledStore, err := loadConfigStore()
	if err != nil {
		t.Fatal(err)
	}
	disabledConfig := disabledStore.Devices[serial]
	mutatedConfig := *cloneDeviceConfig(&disabledConfig)
	mutatedConfig.Zones[0].LedCount++
	if err = oldDevice.SaveDeviceConfig(&mutatedConfig); err == nil || !strings.Contains(err.Error(), "detached") {
		t.Fatalf("detached SaveDeviceConfig error = %v", err)
	}
	afterDetachedCall, err := loadConfigStore()
	if err != nil {
		t.Fatal(err)
	}
	if !afterDetachedCall.Devices[serial].Disabled || !reflect.DeepEqual(afterDetachedCall.Devices[serial], disabledConfig) {
		t.Fatalf("detached SaveDeviceConfig changed disabled entry: %#v", afterDetachedCall.Devices[serial])
	}
	if registry.count() != 0 || enabledConfiguredCount() != 0 || len(managerMembership) != 0 {
		t.Fatalf(
			"detached mutation restored membership registry=%d configured=%d manager=%d",
			registry.count(),
			enabledConfiguredCount(),
			len(managerMembership),
		)
	}

	reimported, err := ImportControllers(context.Background(), []string{key}, registry.hooks())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reimported.ConfiguredSerials, []string{serial}) {
		t.Fatalf("reimported serials = %#v, want %q", reimported.ConfiguredSerials, serial)
	}
	newDevice, ok := configuredDevice(serial)
	if !ok || newDevice == oldDevice {
		t.Fatalf("reimport device = %p, old detached device = %p", newDevice, oldDevice)
	}

	rgbPath := filepath.Join(root, "database", "rgb", serial+".json")
	profilePath := filepath.Join(root, "database", "profiles", serial+".json")
	rgbBefore, err := os.ReadFile(rgbPath)
	if err != nil {
		t.Fatal(err)
	}
	profileBefore, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	storeBefore, err := loadConfigStore()
	if err != nil {
		t.Fatal(err)
	}
	oldBefore := oldDevice.Snapshot()
	clusterControllers, clusterAdds, clusterRemoves := clusterState.counts()
	managerAddsBefore := managerAdds.Load()
	managerRemovesBefore := managerRemoves.Load()

	currentConfig := storeBefore.Devices[serial]
	staleConfig := *cloneDeviceConfig(&currentConfig)
	staleConfig.Zones[0].LedCount++
	if err = oldDevice.SaveDeviceConfig(&staleConfig); err == nil ||
		oldDevice.SaveUserProfile("stale") != 0 ||
		oldDevice.SaveDeviceProfile("", false) != 0 ||
		oldDevice.ChangeDeviceProfile("default") != 0 ||
		oldDevice.DeleteDeviceProfile("stale") != 0 ||
		oldDevice.ProcessSetRgbCluster(true) != 0 ||
		oldDevice.UpdateRgbProfileData("static", rgb.Profile{ProfileName: "static"}) != 0 ||
		oldDevice.UpdateRgbProfile(-1, "static") != 0 ||
		oldDevice.ProcessSetRgbOverride(0, 0, true, rgb.Color{}, rgb.Color{}, rgb.Color{}, 1) != 0 {
		t.Fatal("detached stale object accepted a persistent or cluster mutation")
	}
	oldDevice.SetSpeed("fast")
	if err = oldDevice.SetColor([]byte{1, 2, 3}); err == nil {
		t.Fatal("detached SetColor succeeded")
	}
	if err = oldDevice.SetBrightness(50); err == nil {
		t.Fatal("detached SetBrightness succeeded")
	}
	if err = oldDevice.SetEffect("off"); err == nil {
		t.Fatal("detached SetEffect succeeded")
	}

	storeAfter, err := loadConfigStore()
	if err != nil {
		t.Fatal(err)
	}
	rgbAfter, err := os.ReadFile(rgbPath)
	if err != nil {
		t.Fatal(err)
	}
	profileAfter, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(storeAfter, storeBefore) ||
		string(rgbAfter) != string(rgbBefore) ||
		string(profileAfter) != string(profileBefore) ||
		!reflect.DeepEqual(oldDevice.Snapshot(), oldBefore) {
		t.Fatal("detached stale calls changed store, artifacts, or old runtime state")
	}
	if _, statErr := os.Stat(filepath.Join(root, "database", "profiles", serial+"-stale.json")); !os.IsNotExist(statErr) {
		t.Fatalf("detached stale call created profile: %v", statErr)
	}
	if current, ok := configuredDevice(serial); !ok || current != newDevice {
		t.Fatal("stale calls replaced the new configured object")
	}
	if _, current, ok := registry.lookup(serial); !ok || current != newDevice {
		t.Fatal("stale calls replaced the new registry object")
	}
	if managerMembership[serial] != newDevice ||
		managerAdds.Load() != managerAddsBefore ||
		managerRemoves.Load() != managerRemovesBefore {
		t.Fatal("stale calls changed manager membership")
	}
	if controllers, adds, removes := clusterState.counts(); controllers != clusterControllers ||
		adds != clusterAdds || removes != clusterRemoves {
		t.Fatal("stale calls changed cluster membership")
	}
}

func TestSaveDeviceConfigPreservesPersistedDisabledMembership(t *testing.T) {
	_, _ = setupLifecycleTest(t)
	serial := "openrgb-save-disabled-membership"
	stored := testConfig(serial, "Disabled Membership")
	stored.Disabled = true
	stored.ExternalSerial = " persisted-external "
	if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{serial: stored}}); err != nil {
		t.Fatal(err)
	}
	device := testDevice(stored)
	brightness := uint8(100)
	device.DeviceProfile = &DeviceProfile{
		Active:           true,
		RGBProfile:       "static",
		BrightnessSlider: &brightness,
		ZoneColors:       buildZoneColorsFromConfig(&stored, device.lastColor),
	}
	input := testConfig(serial, "Caller Product")
	input.Disabled = false
	input.ExternalSerial = "caller-external"
	input.Zones[0].LedCount = 2

	if err := device.SaveDeviceConfig(&input); err != nil {
		t.Fatal(err)
	}
	store, err := loadConfigStore()
	if err != nil {
		t.Fatal(err)
	}
	saved := store.Devices[serial]
	if !saved.Disabled || saved.ExternalSerial != "persisted-external" || saved.Zones[0].LedCount != 2 {
		t.Fatalf("SaveDeviceConfig changed lifecycle membership or identity: %#v", saved)
	}
	if device.Config == nil || !device.Config.Disabled {
		t.Fatalf("runtime config lost disabled membership: %#v", device.Config)
	}
}

func TestIntegratedImportAndFinalRemovalUsesRealManagerPath(t *testing.T) {
	_, _ = setupLifecycleTest(t)
	registry := newFakeImportRegistry()
	controller := lifecycleController("Integrated", "integrated-external", "", 2)
	statusNeutralDiscover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return []openrgb.DiscoveredController{controller}, nil
	}

	addLifecycleManager = addManagerDevices
	removeLifecycleManager = removeManagerDevices
	stopLifecycleManager = stopManagerIfNoConfigured
	previousFactory := managerFactory
	var factories atomic.Int32
	var discoveries atomic.Int32
	var captured *Manager
	managerFactory = func(devices map[string]*Device, update availabilityUpdater) *Manager {
		factories.Add(1)
		manager := newManager(devices, update)
		configureTestManager(manager)
		manager.healthyInterval = time.Hour
		manager.discover = func(context.Context) ([]openrgb.DiscoveredController, error) {
			discoveries.Add(1)
			return []openrgb.DiscoveredController{controller}, nil
		}
		manager.resume = func(context.Context, *Device) error { return nil }
		captured = manager
		return manager
	}
	t.Cleanup(func() { managerFactory = previousFactory })

	preview, err := DiscoverPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	imported, err := ImportControllers(context.Background(), []string{preview.Controllers[0].Key}, registry.hooks())
	if err != nil {
		t.Fatal(err)
	}
	if factories.Load() != 1 || captured == nil {
		t.Fatalf("manager factories=%d captured=%p", factories.Load(), captured)
	}
	waitFor(t, time.Second, func() bool { return discoveries.Load() >= 1 })
	beforeRemoval := discoveries.Load()

	if _, err = RemoveConfiguredImports(context.Background(), imported.ConfiguredSerials, registry.hooks()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)
	if got := discoveries.Load(); got != beforeRemoval {
		t.Fatalf("final manager removal rediscovered: before=%d after=%d", beforeRemoval, got)
	}
	activeManagerMutex.RLock()
	manager := activeManager
	activeManagerMutex.RUnlock()
	if manager != nil || enabledConfiguredCount() != 0 || len(captured.commands) != 0 {
		t.Fatalf("manager/configured state after final removal: active=%p configured=%d pending=%d",
			manager, enabledConfiguredCount(), len(captured.commands))
	}
	select {
	case <-captured.ctx.Done():
	default:
		t.Fatal("integrated manager worker was not stopped")
	}
	state, statusErr := openrgb.GetStatus()
	if state != openrgb.StateNotConfigured || statusErr != nil {
		t.Fatalf("status = %q, %v; want Not Configured", state, statusErr)
	}
}
