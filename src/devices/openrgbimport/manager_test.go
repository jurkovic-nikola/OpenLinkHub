package openrgbimport

import (
	"LumenForge/src/common"
	"LumenForge/src/config"
	"LumenForge/src/logger"
	"LumenForge/src/openrgb"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	temporaryDirectory, err := os.MkdirTemp("", "lumenforge-openrgbimport-test-")
	if err != nil {
		panic(err)
	}
	if err = os.Chdir(temporaryDirectory); err != nil {
		panic(err)
	}
	config.Init()
	logger.Init()

	code := m.Run()
	_ = os.Chdir(originalWorkingDirectory)
	_ = os.RemoveAll(temporaryDirectory)
	os.Exit(code)
}

func useStorePath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "openrgbimport-zones.json")
	previous := configStorePath
	configStorePath = func() string { return path }
	t.Cleanup(func() {
		configStorePath = previous
	})
	return path
}

func testConfig(serial, product string) DeviceConfig {
	return DeviceConfig{
		Serial:  serial,
		Product: product,
		Zones: []ZoneConfig{
			{Name: "Zone 1", LedCount: 1},
		},
	}
}

func testDevice(cfg DeviceConfig) *Device {
	d := &Device{
		Product:      cfg.Product,
		Serial:       cfg.Serial,
		IsOpenRGB:    true,
		controllerId: -1,
		colorCount:   configLedCount(&cfg),
		LEDCount:     configLedCount(&cfg),
		ZoneAmount:   len(cfg.Zones),
		Config:       cloneDeviceConfig(&cfg),
		brightness:   100,
		lastColor:    []byte{99, 213, 255},
		effect:       "static",
		speed:        2,
	}
	d.createDevice()
	d.instance.Unavailable = true
	return d
}

func configureTestManager(manager *Manager) {
	manager.retryBackoff = []time.Duration{5 * time.Millisecond, 10 * time.Millisecond, 20 * time.Millisecond}
	manager.healthyInterval = time.Hour
	manager.reconcileTimeout = time.Second
	manager.logFailure = func(error) {}
	manager.logRecovery = func() {}
	manager.logDiagnostic = func(string, error) {}
	manager.resume = func(context.Context, *Device) error { return nil }
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition was not satisfied before timeout")
}

func TestInitAllAndStartManagerPreserveStoreStatus(t *testing.T) {
	tests := []struct {
		name       string
		prepare    func(*testing.T, string)
		wantState  openrgb.ConnectionState
		wantError  bool
		wantAbsent bool
	}{
		{name: "absent store", wantState: openrgb.StateNotConfigured, wantAbsent: true},
		{
			name: "valid empty store",
			prepare: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, []byte(`{"devices":{}}`), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantState: openrgb.StateNotConfigured,
		},
		{
			name: "empty file",
			prepare: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, nil, 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantState: openrgb.StateOffline,
			wantError: true,
		},
		{
			name: "malformed JSON",
			prepare: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantState: openrgb.StateOffline,
			wantError: true,
		},
		{
			name: "read error",
			prepare: func(t *testing.T, path string) {
				t.Helper()
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantState: openrgb.StateOffline,
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := useStorePath(t)
			StopManager()
			setConfiguredDevices(nil)
			if test.prepare != nil {
				test.prepare(t, path)
			}

			if imported := InitAll(); len(imported) != 0 {
				t.Fatalf("InitAll returned %d devices, want 0", len(imported))
			}
			StartManager(nil, nil)
			if len(configuredDevicesSnapshot()) != 0 {
				t.Fatal("invalid or empty store created configured devices")
			}
			if test.wantAbsent {
				if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
					t.Fatalf("absent bootstrap created a store: %v", statErr)
				}
			}
			state, err := openrgb.GetStatus()
			if state != test.wantState || (err != nil) != test.wantError {
				t.Fatalf("status = %q, %v; want %q, error=%v", state, err, test.wantState, test.wantError)
			}
			activeManagerMutex.RLock()
			manager := activeManager
			activeManagerMutex.RUnlock()
			if manager != nil {
				t.Fatal("zero-device store started a manager")
			}
		})
	}
}

func TestInitAllRejectsSemanticallyInvalidStoreBeforeAllocation(t *testing.T) {
	validSerial := "openrgb-import-invalid-layout"
	zones := func(count, ledCount int) []ZoneConfig {
		result := make([]ZoneConfig, count)
		for index := range result {
			result[index] = ZoneConfig{Name: "Zone", LedCount: ledCount}
		}
		return result
	}
	tests := []struct {
		name      string
		mapSerial string
		config    DeviceConfig
	}{
		{name: "excessive zone count", mapSerial: validSerial, config: DeviceConfig{Serial: validSerial, Zones: zones(129, 1)}},
		{name: "excessive per-zone LED count", mapSerial: validSerial, config: DeviceConfig{Serial: validSerial, Zones: zones(1, 1025)}},
		{name: "excessive total LED count", mapSerial: validSerial, config: DeviceConfig{Serial: validSerial, Zones: zones(5, 1024)}},
		{name: "zero LED count", mapSerial: validSerial, config: DeviceConfig{Serial: validSerial, Zones: zones(1, 0)}},
		{name: "negative LED count", mapSerial: validSerial, config: DeviceConfig{Serial: validSerial, Zones: zones(1, -1)}},
		{name: "conflicting stored serial", mapSerial: validSerial, config: DeviceConfig{Serial: "openrgb-import-other", Zones: zones(1, 1)}},
		{name: "unusable map serial", mapSerial: "../invalid", config: DeviceConfig{Serial: "../invalid", Zones: zones(1, 1)}},
		{name: "huge LED count cannot allocate", mapSerial: validSerial, config: DeviceConfig{Serial: validSerial, Zones: zones(1, int(^uint(0)>>1))}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := useStorePath(t)
			StopManager()
			setConfiguredDevices(nil)
			data, err := json.Marshal(ConfigStore{Devices: map[string]DeviceConfig{test.mapSerial: test.config}})
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}

			started := time.Now()
			if imported := InitAll(); len(imported) != 0 {
				t.Fatalf("InitAll returned %d placeholders for an invalid store", len(imported))
			}
			if elapsed := time.Since(started); elapsed > time.Second {
				t.Fatalf("semantic rejection took %v", elapsed)
			}
			StartManager(nil, nil)
			if len(configuredDevicesSnapshot()) != 0 {
				t.Fatal("invalid store configured importer devices")
			}
			state, statusErr := openrgb.GetStatus()
			if state != openrgb.StateOffline || statusErr == nil {
				t.Fatalf("status = %q, %v; want Offline with semantic error", state, statusErr)
			}
			unchanged, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(unchanged) != string(data) {
				t.Fatal("semantic validation modified the existing store")
			}
			activeManagerMutex.RLock()
			manager := activeManager
			activeManagerMutex.RUnlock()
			if manager != nil {
				t.Fatal("invalid store started a manager")
			}
		})
	}
}

func TestInitAllLoadsValidLegacyStoreAndNormalizesEmptySerialInMemory(t *testing.T) {
	tests := []struct {
		name       string
		mapSerial  string
		config     DeviceConfig
		wantSerial string
		wantName   string
	}{
		{
			name:       "empty stored serial uses valid map key",
			mapSerial:  "openrgb-import-map-key",
			config:     DeviceConfig{Product: "Map Key Device", Zones: []ZoneConfig{{Name: "\x00", LedCount: 1}}},
			wantSerial: "openrgb-import-map-key",
			wantName:   "Zone 1",
		},
		{
			name:       "valid legacy entry",
			mapSerial:  "openrgb-mobo-1",
			config:     DeviceConfig{Serial: "openrgb-mobo-1", Product: "Legacy Device", Zones: []ZoneConfig{{Name: "Legacy Zone", LedCount: 4}}},
			wantSerial: "openrgb-mobo-1",
			wantName:   "Legacy Zone",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := useStorePath(t)
			StopManager()
			setConfiguredDevices(nil)
			data, err := json.Marshal(ConfigStore{Devices: map[string]DeviceConfig{test.mapSerial: test.config}})
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}

			wrappers := InitAll()
			if len(wrappers) != 1 {
				t.Fatalf("InitAll returned %d placeholders, want 1", len(wrappers))
			}
			device := configuredDevicesSnapshot()[test.mapSerial]
			device.mu.Lock()
			gotSerial := device.Config.Serial
			gotName := device.Config.Zones[0].Name
			device.mu.Unlock()
			if gotSerial != test.wantSerial || gotName != test.wantName {
				t.Fatalf("normalized config = serial %q, zone %q", gotSerial, gotName)
			}
			unchanged, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(unchanged) != string(data) {
				t.Fatal("in-memory compatibility normalization modified the store")
			}
			setConfiguredDevices(nil)
		})
	}
}

func TestConfiguredBootstrapReturnsPlaceholderBeforeAsyncBind(t *testing.T) {
	useStorePath(t)
	StopManager()
	serial := "openrgb-import-7"
	cfg := testConfig(serial, "Configured Device")
	if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{serial: cfg}}); err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	wrappers := InitAll()
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("disk-only bootstrap took %v", elapsed)
	}
	if len(wrappers) != 1 || !wrappers[0].Unavailable {
		t.Fatalf("bootstrap wrappers = %#v, want one unavailable placeholder", wrappers)
	}
	device := configuredDevicesSnapshot()[serial]
	if device == nil || device.ControllerID() != -1 {
		t.Fatalf("placeholder controller ID = %v, want -1", device)
	}
	if device.Config == nil || device.DeviceProfile == nil || device.Rgb == nil {
		t.Fatal("placeholder did not load saved zone, profile, and RGB state")
	}

	availability := make(chan bool, 4)
	manager := newManager(configuredDevicesSnapshot(), func(_ string, unavailable bool) {
		availability <- unavailable
	})
	configureTestManager(manager)
	manager.discover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return []openrgb.DiscoveredController{{ID: 4, Name: cfg.Product, Serial: "external-4"}}, nil
	}
	manager.Start()
	t.Cleanup(manager.Stop)

	waitFor(t, time.Second, func() bool { return device.ControllerID() == 4 })
	if device != configuredDevicesSnapshot()[serial] {
		t.Fatal("reconciliation replaced the configured device object")
	}
	select {
	case unavailable := <-availability:
		if unavailable {
			t.Fatal("bound device remained unavailable")
		}
	case <-time.After(time.Second):
		t.Fatal("availability callback was not invoked")
	}
	state, _ := openrgb.GetStatus()
	if state != openrgb.StateConnected {
		t.Fatalf("status = %q, want Connected", state)
	}
}

func TestManagerRetriesUntilDelayedServerIsAvailable(t *testing.T) {
	useStorePath(t)
	serial := "openrgb-import-delayed"
	cfg := testConfig(serial, "Delayed Device")
	if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{serial: cfg}}); err != nil {
		t.Fatal(err)
	}
	device := testDevice(cfg)
	manager := newManager(map[string]*Device{serial: device}, nil)
	configureTestManager(manager)

	var attempts atomic.Int32
	manager.discover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		if attempts.Add(1) < 3 {
			return nil, errors.New("server not started")
		}
		return []openrgb.DiscoveredController{{ID: 12, Name: cfg.Product, Serial: "delayed-external"}}, nil
	}
	started := time.Now()
	manager.Start()
	if elapsed := time.Since(started); elapsed > 50*time.Millisecond {
		t.Fatalf("Manager.Start blocked for %v", elapsed)
	}
	t.Cleanup(manager.Stop)

	waitFor(t, time.Second, func() bool { return device.ControllerID() == 12 })
	if attempts.Load() < 3 {
		t.Fatalf("attempts = %d, want at least 3", attempts.Load())
	}
}

func TestManagerIsolatesRestorationFailureAndRecoversDevice(t *testing.T) {
	useStorePath(t)
	failedConfig := testConfig("openrgb-import-a-failed", "Failed Device")
	failedConfig.ExternalSerial = "failed-external"
	healthyConfig := testConfig("openrgb-import-b-healthy", "Healthy Device")
	healthyConfig.ExternalSerial = "healthy-external"
	if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{
		failedConfig.Serial:  failedConfig,
		healthyConfig.Serial: healthyConfig,
	}}); err != nil {
		t.Fatal(err)
	}
	failedDevice := testDevice(failedConfig)
	healthyDevice := testDevice(healthyConfig)
	availability := map[string]bool{failedConfig.Serial: true, healthyConfig.Serial: true}
	var availabilityMutex sync.Mutex
	manager := newManager(map[string]*Device{
		failedConfig.Serial:  failedDevice,
		healthyConfig.Serial: healthyDevice,
	}, func(serial string, unavailable bool) {
		availabilityMutex.Lock()
		availability[serial] = unavailable
		availabilityMutex.Unlock()
	})
	configureTestManager(manager)
	manager.retryBackoff = []time.Duration{50 * time.Millisecond}

	var discoveries atomic.Int32
	var activeDiscoveries atomic.Int32
	var maximumDiscoveries atomic.Int32
	manager.discover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		discoveries.Add(1)
		current := activeDiscoveries.Add(1)
		defer activeDiscoveries.Add(-1)
		for {
			observed := maximumDiscoveries.Load()
			if current <= observed || maximumDiscoveries.CompareAndSwap(observed, current) {
				break
			}
		}
		time.Sleep(time.Millisecond)
		return []openrgb.DiscoveredController{
			{ID: 31, Name: failedConfig.Product, Serial: failedConfig.ExternalSerial},
			{ID: 32, Name: healthyConfig.Product, Serial: healthyConfig.ExternalSerial},
		}, nil
	}
	var failedRestorations atomic.Int32
	var healthyRestorations atomic.Int32
	manager.resume = func(_ context.Context, device *Device) error {
		if device.Serial == failedConfig.Serial {
			if failedRestorations.Add(1) == 1 {
				return errors.New("invalid local frame")
			}
			return nil
		}
		healthyRestorations.Add(1)
		return nil
	}
	var globalFailureLogs atomic.Int32
	manager.logFailure = func(error) { globalFailureLogs.Add(1) }
	var diagnosticMutex sync.Mutex
	diagnosticSerials := make([]string, 0, 1)
	manager.logDiagnostic = func(serial string, _ error) {
		diagnosticMutex.Lock()
		diagnosticSerials = append(diagnosticSerials, serial)
		diagnosticMutex.Unlock()
	}

	manager.Start()
	t.Cleanup(manager.Stop)
	waitFor(t, time.Second, func() bool {
		availabilityMutex.Lock()
		defer availabilityMutex.Unlock()
		return failedRestorations.Load() == 1 && availability[failedConfig.Serial] && !availability[healthyConfig.Serial]
	})
	if failedDevice.ControllerID() != -1 {
		t.Fatalf("failed device controller ID = %d, want unavailable", failedDevice.ControllerID())
	}
	if healthyDevice.ControllerID() != 32 {
		t.Fatalf("healthy device controller ID = %d, want 32", healthyDevice.ControllerID())
	}
	waitFor(t, time.Second, func() bool {
		availabilityMutex.Lock()
		defer availabilityMutex.Unlock()
		return failedRestorations.Load() >= 2 && !availability[failedConfig.Serial]
	})
	if failedDevice.ControllerID() != 31 {
		t.Fatalf("recovered device controller ID = %d, want 31", failedDevice.ControllerID())
	}
	if healthyRestorations.Load() != 1 {
		t.Fatalf("healthy device restored %d times, want 1", healthyRestorations.Load())
	}
	if maximumDiscoveries.Load() != 1 || discoveries.Load() < 2 {
		t.Fatalf("discoveries = %d, maximum concurrent = %d", discoveries.Load(), maximumDiscoveries.Load())
	}
	if globalFailureLogs.Load() != 0 {
		t.Fatalf("local restoration produced %d global SDK failure logs", globalFailureLogs.Load())
	}
	diagnosticMutex.Lock()
	if len(diagnosticSerials) != 1 || diagnosticSerials[0] != failedConfig.Serial {
		t.Fatalf("diagnostic serials = %#v, want only failed device", diagnosticSerials)
	}
	diagnosticMutex.Unlock()
	state, _ := openrgb.GetStatus()
	if state != openrgb.StateConnected {
		t.Fatalf("OpenRGB status = %q after partial recovery, want Connected", state)
	}
}

func TestManagerStopInterruptsPartialFailureRetry(t *testing.T) {
	useStorePath(t)
	cfg := testConfig("openrgb-import-partial-stop", "Partial Stop Device")
	cfg.ExternalSerial = "partial-stop-external"
	if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{cfg.Serial: cfg}}); err != nil {
		t.Fatal(err)
	}
	device := testDevice(cfg)
	manager := newManager(map[string]*Device{cfg.Serial: device}, nil)
	configureTestManager(manager)
	manager.retryBackoff = []time.Duration{time.Hour}
	var discoveries atomic.Int32
	manager.discover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		discoveries.Add(1)
		return []openrgb.DiscoveredController{{ID: 61, Name: cfg.Product, Serial: cfg.ExternalSerial}}, nil
	}
	manager.resume = func(context.Context, *Device) error { return errors.New("local restoration failed") }
	diagnostic := make(chan struct{}, 1)
	manager.logDiagnostic = func(string, error) { diagnostic <- struct{}{} }
	var globalFailureLogs atomic.Int32
	manager.logFailure = func(error) { globalFailureLogs.Add(1) }

	manager.Start()
	select {
	case <-diagnostic:
	case <-time.After(time.Second):
		t.Fatal("partial restoration failure was not reported")
	}
	started := time.Now()
	manager.Stop()
	manager.Stop()
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Stop during partial retry took %v", elapsed)
	}
	time.Sleep(20 * time.Millisecond)
	if discoveries.Load() != 1 {
		t.Fatalf("discoveries after canceled partial retry = %d, want 1", discoveries.Load())
	}
	if globalFailureLogs.Load() != 0 {
		t.Fatalf("partial retry produced %d global SDK failure logs", globalFailureLogs.Load())
	}
}

func TestManagerDisconnectAndRestartRebindsExistingDevice(t *testing.T) {
	useStorePath(t)
	serial := "openrgb-import-restart"
	cfg := testConfig(serial, "Restart Device")
	if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{serial: cfg}}); err != nil {
		t.Fatal(err)
	}
	device := testDevice(cfg)
	manager := newManager(map[string]*Device{serial: device}, nil)
	configureTestManager(manager)
	var controllerID atomic.Int32
	controllerID.Store(3)
	manager.discover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return []openrgb.DiscoveredController{{ID: int(controllerID.Load()), Name: cfg.Product, Serial: "restart-external"}}, nil
	}
	var resumes atomic.Int32
	manager.resume = func(context.Context, *Device) error {
		resumes.Add(1)
		return nil
	}

	activeManagerMutex.Lock()
	activeManager = manager
	activeManagerMutex.Unlock()
	t.Cleanup(func() {
		activeManagerMutex.Lock()
		if activeManager == manager {
			activeManager = nil
		}
		activeManagerMutex.Unlock()
		manager.Stop()
	})
	manager.Start()
	waitFor(t, time.Second, func() bool { return device.ControllerID() == 3 })

	clientConn, serverConn := net.Pipe()
	device.mu.Lock()
	device.openrgbConn = clientConn
	device.mu.Unlock()
	controllerID.Store(9)
	device.handleOutputFailure(errors.New("connection lost"))
	if device.ControllerID() != -1 {
		t.Fatalf("controller ID = %d after failure, want -1", device.ControllerID())
	}
	_ = serverConn.SetReadDeadline(time.Now().Add(time.Second))
	buffer := make([]byte, 1)
	if _, err := serverConn.Read(buffer); err == nil {
		t.Fatal("persistent connection remained open after failure")
	}
	_ = serverConn.Close()

	waitFor(t, time.Second, func() bool { return device.ControllerID() == 9 })
	if resumes.Load() != 2 {
		t.Fatalf("desired state resumed %d times, want 2", resumes.Load())
	}
	if len(manager.devices) != 1 || manager.devices[serial] != device {
		t.Fatal("restart duplicated or replaced the importer device")
	}
}

func TestMarkUnavailableStopsActiveOutputAndClosesConnection(t *testing.T) {
	device := testDevice(testConfig("openrgb-import-active", "Active Device"))
	stop := make(chan struct{})
	done := make(chan struct{})
	clientConn, serverConn := net.Pipe()
	device.mu.Lock()
	device.controllerId = 5
	device.running = true
	device.stopChan = stop
	device.doneChan = done
	device.openrgbConn = clientConn
	device.mu.Unlock()
	go func() {
		<-stop
		close(done)
	}()

	if !device.markUnavailable() {
		t.Fatal("active device transition was not reported as changed")
	}
	device.mu.Lock()
	controllerID := device.controllerId
	running := device.running
	connection := device.openrgbConn
	device.mu.Unlock()
	if controllerID != -1 || running || connection != nil {
		t.Fatalf("offline state = id %d, running %v, conn %v", controllerID, running, connection)
	}
	_ = serverConn.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := serverConn.Read(make([]byte, 1)); err == nil {
		t.Fatal("persistent connection remained open")
	}
	_ = serverConn.Close()
}

func TestManagerCoalescesTriggersAndSerializesReconciliation(t *testing.T) {
	useStorePath(t)
	serial := "openrgb-import-coalesce"
	cfg := testConfig(serial, "Coalesce Device")
	if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{serial: cfg}}); err != nil {
		t.Fatal(err)
	}
	device := testDevice(cfg)
	manager := newManager(map[string]*Device{serial: device}, nil)
	configureTestManager(manager)

	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	var attempts atomic.Int32
	var active atomic.Int32
	var maximum atomic.Int32
	var resumes atomic.Int32
	manager.resume = func(context.Context, *Device) error {
		resumes.Add(1)
		return nil
	}
	manager.discover = func(ctx context.Context) ([]openrgb.DiscoveredController, error) {
		attempts.Add(1)
		current := active.Add(1)
		defer active.Add(-1)
		for {
			observed := maximum.Load()
			if current <= observed || maximum.CompareAndSwap(observed, current) {
				break
			}
		}
		select {
		case entered <- struct{}{}:
		default:
		}
		if attempts.Load() == 1 {
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		return []openrgb.DiscoveredController{{ID: 1, Name: cfg.Product, Serial: "coalesce-external"}}, nil
	}
	manager.Start()
	t.Cleanup(manager.Stop)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first reconciliation did not start")
	}
	for i := 0; i < 100; i++ {
		manager.Trigger()
	}
	close(release)
	waitFor(t, time.Second, func() bool { return attempts.Load() >= 2 })
	waitFor(t, time.Second, func() bool { return device.ControllerID() == 1 })
	if maximum.Load() != 1 {
		t.Fatalf("maximum concurrent reconciliations = %d, want 1", maximum.Load())
	}
	if len(manager.devices) != 1 || manager.devices[serial] != device {
		t.Fatal("coalesced retries changed registry identity")
	}
	if resumes.Load() != 1 {
		t.Fatalf("unchanged controller binding resumed output %d times, want 1", resumes.Load())
	}
}

func TestManagerStopInterruptsBackoffAndStalledDiscovery(t *testing.T) {
	t.Run("backoff", func(t *testing.T) {
		device := testDevice(testConfig("openrgb-import-stop-backoff", "Backoff Device"))
		manager := newManager(map[string]*Device{device.Serial: device}, nil)
		configureTestManager(manager)
		manager.retryBackoff = []time.Duration{time.Hour}
		attempted := make(chan struct{}, 1)
		manager.discover = func(context.Context) ([]openrgb.DiscoveredController, error) {
			attempted <- struct{}{}
			return nil, errors.New("offline")
		}
		manager.Start()
		<-attempted
		started := time.Now()
		manager.Stop()
		manager.Stop()
		if elapsed := time.Since(started); elapsed > time.Second {
			t.Fatalf("Stop during backoff took %v", elapsed)
		}
	})

	t.Run("stalled discovery", func(t *testing.T) {
		device := testDevice(testConfig("openrgb-import-stop-stalled", "Stalled Device"))
		manager := newManager(map[string]*Device{device.Serial: device}, nil)
		configureTestManager(manager)
		entered := make(chan struct{})
		manager.discover = func(ctx context.Context) ([]openrgb.DiscoveredController, error) {
			close(entered)
			<-ctx.Done()
			return nil, ctx.Err()
		}
		manager.Start()
		<-entered
		started := time.Now()
		manager.Stop()
		if elapsed := time.Since(started); elapsed > time.Second {
			t.Fatalf("Stop during discovery took %v", elapsed)
		}
	})
}

func TestManagerStopInterruptsStalledRestorationWithoutRetry(t *testing.T) {
	device := testDevice(testConfig("openrgb-import-stop-restore", "Restore Device"))
	manager := newManager(map[string]*Device{device.Serial: device}, nil)
	configureTestManager(manager)
	manager.retryBackoff = []time.Duration{time.Millisecond}

	var attempts atomic.Int32
	manager.discover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		attempts.Add(1)
		return []openrgb.DiscoveredController{{ID: 44, Name: device.Product, Serial: "restore-serial"}}, nil
	}
	restoring := make(chan struct{})
	manager.resume = func(ctx context.Context, _ *Device) error {
		close(restoring)
		<-ctx.Done()
		return ctx.Err()
	}
	var failureLogs atomic.Int32
	manager.logFailure = func(error) { failureLogs.Add(1) }

	manager.Start()
	select {
	case <-restoring:
	case <-time.After(time.Second):
		t.Fatal("desired-state restoration did not start")
	}
	started := time.Now()
	manager.Stop()
	manager.Stop()
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Stop during restoration took %v", elapsed)
	}
	time.Sleep(20 * time.Millisecond)
	if attempts.Load() != 1 {
		t.Fatalf("discovery attempts = %d after cancellation, want 1", attempts.Load())
	}
	if failureLogs.Load() != 0 {
		t.Fatalf("canceled restoration logged %d server failures", failureLogs.Load())
	}
}

func TestIdentityMatchingAndLegacyMetadataMigration(t *testing.T) {
	t.Run("placeholder external serials do not outrank metadata", func(t *testing.T) {
		first := testConfig("openrgb-hash-first-placeholder", "First Device")
		first.ExternalSerial = "unknown"
		first.Location = "usb:first"
		second := testConfig("openrgb-hash-second-placeholder", "Second Device")
		second.ExternalSerial = "unknown"
		second.Location = "usb:second"
		devices := map[string]*Device{first.Serial: testDevice(first), second.Serial: testDevice(second)}
		discovered := []openrgb.DiscoveredController{
			{ID: 20, Name: second.Product, Serial: "unknown", Location: second.Location},
			{ID: 10, Name: first.Product, Serial: "unknown", Location: first.Location},
		}
		matches, diagnostics := matchConfiguredDevices(devices, discovered)
		if len(diagnostics) != 0 || matches[first.Serial].ID != 10 || matches[second.Serial].ID != 20 {
			t.Fatalf("matches = %#v, diagnostics = %#v", matches, diagnostics)
		}
	})

	t.Run("legacy placeholder serial is ignored", func(t *testing.T) {
		path := useStorePath(t)
		cfg := testConfig("openrgb-import-legacy-placeholder", "Legacy Placeholder")
		cfg.ExternalSerial = "unknown"
		cfg.Location = "usb:right"
		data, err := json.Marshal(ConfigStore{Devices: map[string]DeviceConfig{cfg.Serial: cfg}})
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		loaded, err := loadConfigStore()
		if err != nil {
			t.Fatal(err)
		}
		device := testDevice(loaded.Devices[cfg.Serial])
		discovered := []openrgb.DiscoveredController{
			{ID: 1, Name: "Wrong Device", Serial: "unknown", Location: "usb:wrong"},
			{ID: 2, Name: cfg.Product, Serial: "safe-serial", Location: cfg.Location},
		}
		matches, diagnostics := matchConfiguredDevices(map[string]*Device{cfg.Serial: device}, discovered)
		if len(diagnostics) != 0 || matches[cfg.Serial].ID != 2 {
			t.Fatalf("matches = %#v, diagnostics = %#v", matches, diagnostics)
		}
	})

	t.Run("external serial survives reorder", func(t *testing.T) {
		first := testConfig("openrgb-hash-first", "First")
		first.ExternalSerial = "serial-first"
		second := testConfig("openrgb-hash-second", "Second")
		second.ExternalSerial = "serial-second"
		devices := map[string]*Device{first.Serial: testDevice(first), second.Serial: testDevice(second)}
		discovered := []openrgb.DiscoveredController{
			{ID: 0, Name: "Second", Serial: "serial-second"},
			{ID: 1, Name: "First", Serial: "serial-first"},
		}
		matches, diagnostics := matchConfiguredDevices(devices, discovered)
		if len(diagnostics) != 0 || matches[first.Serial].ID != 1 || matches[second.Serial].ID != 0 {
			t.Fatalf("matches = %#v, diagnostics = %#v", matches, diagnostics)
		}
	})

	t.Run("unusable serial metadata is not retained or persisted", func(t *testing.T) {
		useStorePath(t)
		cfg := testConfig("openrgb-import-unusable-serial", "Unsafe Serial Device")
		cfg.ExternalSerial = "unknown"
		if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{cfg.Serial: cfg}}); err != nil {
			t.Fatal(err)
		}
		device := testDevice(cfg)
		controller := openrgb.DiscoveredController{ID: 8, Name: cfg.Product, Serial: "n/a"}
		device.updateIdentityMetadata(controller)
		if device.Config.ExternalSerial != "" {
			t.Fatalf("in-memory external serial = %q, want empty", device.Config.ExternalSerial)
		}
		matches := map[string]openrgb.DiscoveredController{cfg.Serial: controller}
		if err := persistIdentityMetadata(matches); err != nil {
			t.Fatal(err)
		}
		store, err := loadConfigStore()
		if err != nil {
			t.Fatal(err)
		}
		if stored := store.Devices[cfg.Serial].ExternalSerial; stored != "" {
			t.Fatalf("persisted external serial = %q, want empty", stored)
		}
	})

	t.Run("legacy unique product learns metadata", func(t *testing.T) {
		useStorePath(t)
		cfg := testConfig("openrgb-import-2", "Legacy Device")
		if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{cfg.Serial: cfg}}); err != nil {
			t.Fatal(err)
		}
		controller := openrgb.DiscoveredController{ID: 17, Name: cfg.Product, Vendor: "Vendor", Serial: "stable-serial", Location: "usb:4"}
		matches, diagnostics := matchConfiguredDevices(map[string]*Device{cfg.Serial: testDevice(cfg)}, []openrgb.DiscoveredController{controller})
		if len(diagnostics) != 0 || matches[cfg.Serial].ID != 17 {
			t.Fatalf("matches = %#v, diagnostics = %#v", matches, diagnostics)
		}
		if err := persistIdentityMetadata(matches); err != nil {
			t.Fatal(err)
		}
		store, err := loadConfigStore()
		if err != nil {
			t.Fatal(err)
		}
		migrated := store.Devices[cfg.Serial]
		if migrated.ExternalSerial != controller.Serial || migrated.Location != controller.Location || migrated.Vendor != controller.Vendor {
			t.Fatalf("migrated metadata = %#v", migrated)
		}
	})

	t.Run("ambiguous identical controllers stay unavailable", func(t *testing.T) {
		controller := openrgb.DiscoveredController{Name: "Identical Device", Vendor: "Vendor", Version: "1"}
		serial := internalSerialForController(controller)
		cfg := testConfig(serial, controller.Name)
		cfg.Vendor = controller.Vendor
		matches, diagnostics := matchConfiguredDevices(
			map[string]*Device{serial: testDevice(cfg)},
			[]openrgb.DiscoveredController{controller, controller},
		)
		if len(matches) != 0 || diagnostics[serial] == nil {
			t.Fatalf("matches = %#v, diagnostics = %#v", matches, diagnostics)
		}
	})
}

func TestConfigStoreBackwardCompatibilityAndAtomicFailure(t *testing.T) {
	path := useStorePath(t)
	legacy := ConfigStore{Devices: map[string]DeviceConfig{
		"openrgb-import-1": testConfig("openrgb-import-1", "Legacy"),
	}}
	legacyData, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, legacyData, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadConfigStore()
	if err != nil {
		t.Fatal(err)
	}
	legacyConfig := loaded.Devices["openrgb-import-1"]
	if legacyConfig.ExternalSerial != "" || legacyConfig.Location != "" || legacyConfig.Vendor != "" {
		t.Fatalf("legacy optional fields = %#v", legacyConfig)
	}

	legacyConfig.ExternalSerial = "external"
	legacyConfig.Location = "usb:1"
	legacyConfig.Vendor = "Vendor"
	loaded.Devices[legacyConfig.Serial] = legacyConfig
	if err = saveConfigStore(loaded); err != nil {
		t.Fatal(err)
	}
	roundTrip, err := loadConfigStore()
	if err != nil {
		t.Fatal(err)
	}
	if roundTrip.Devices[legacyConfig.Serial].ExternalSerial != "external" {
		t.Fatalf("optional metadata did not round-trip: %#v", roundTrip.Devices[legacyConfig.Serial])
	}

	validBefore, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	previousRename := renameConfigStore
	renameConfigStore = func(string, string) error { return errors.New("injected rename failure") }
	t.Cleanup(func() { renameConfigStore = previousRename })
	roundTrip.Devices[legacyConfig.Serial] = testConfig(legacyConfig.Serial, "Replacement")
	if err = saveConfigStore(roundTrip); err == nil {
		t.Fatal("expected injected save failure")
	}
	validAfter, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(validAfter) != string(validBefore) {
		t.Fatal("failed atomic write changed the prior valid store")
	}
}

func TestStartManagerRejectsLocalTargetConflict(t *testing.T) {
	StopManager()
	serial := "openrgb-import-conflict"
	device := testDevice(testConfig(serial, "Conflict Device"))
	setConfiguredDevices(map[string]*Device{serial: device})
	t.Cleanup(func() {
		StopManager()
		setConfiguredDevices(nil)
	})

	previousTargetCheck := localTargetServerEnabled
	localTargetServerEnabled = func() bool { return true }
	t.Cleanup(func() { localTargetServerEnabled = previousTargetCheck })
	var callbackCount atomic.Int32
	StartManager(func(callbackSerial string, unavailable bool) {
		if callbackSerial == serial && unavailable {
			callbackCount.Add(1)
		}
	}, nil)

	activeManagerMutex.RLock()
	manager := activeManager
	activeManagerMutex.RUnlock()
	if manager != nil {
		t.Fatal("target conflict started an importer manager")
	}
	if callbackCount.Load() != 1 {
		t.Fatalf("availability callbacks = %d, want 1", callbackCount.Load())
	}
	state, err := openrgb.GetStatus()
	if state != openrgb.StateOffline || err == nil {
		t.Fatalf("status = %q, %v; want Offline with conflict error", state, err)
	}
}

func TestRepeatedStartManagerKeepsSingleWorker(t *testing.T) {
	StopManager()
	serial := "openrgb-import-repeated-start"
	device := testDevice(testConfig(serial, "Repeated Start Device"))
	setConfiguredDevices(map[string]*Device{serial: device})
	t.Cleanup(func() {
		StopManager()
		setConfiguredDevices(nil)
	})

	previousTargetCheck := localTargetServerEnabled
	localTargetServerEnabled = func() bool { return false }
	t.Cleanup(func() { localTargetServerEnabled = previousTargetCheck })
	previousFactory := managerFactory
	var factories atomic.Int32
	var activeWorkers atomic.Int32
	var maximumWorkers atomic.Int32
	started := make(chan struct{}, 1)
	managerFactory = func(devices map[string]*Device, update availabilityUpdater) *Manager {
		factories.Add(1)
		manager := newManager(devices, update)
		configureTestManager(manager)
		manager.discover = func(ctx context.Context) ([]openrgb.DiscoveredController, error) {
			current := activeWorkers.Add(1)
			defer activeWorkers.Add(-1)
			for {
				observed := maximumWorkers.Load()
				if current <= observed || maximumWorkers.CompareAndSwap(observed, current) {
					break
				}
			}
			select {
			case started <- struct{}{}:
			default:
			}
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return manager
	}
	t.Cleanup(func() { managerFactory = previousFactory })

	StartManager(nil, nil)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first manager did not start")
	}
	StartManager(nil, nil)
	if factories.Load() != 1 {
		t.Fatalf("manager factory calls = %d, want 1", factories.Load())
	}
	if maximumWorkers.Load() != 1 {
		t.Fatalf("maximum active workers = %d, want 1", maximumWorkers.Load())
	}
	StopManager()
	StopManager()
	if activeWorkers.Load() != 0 {
		t.Fatalf("active workers after StopManager = %d, want 0", activeWorkers.Load())
	}
}

func TestConfigStoreReadErrorIsReturned(t *testing.T) {
	path := useStorePath(t)
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := loadConfigStore()
	if err == nil || store != nil {
		t.Fatalf("loadConfigStore = %#v, %v; want nil, error", store, err)
	}
}

func TestRegistryWrapperIdentityIsStableInManager(t *testing.T) {
	cfg := testConfig("openrgb-import-wrapper", "Wrapper Device")
	device := testDevice(cfg)
	wrapper := device.instance
	manager := newManager(map[string]*Device{cfg.Serial: device}, nil)
	if manager.devices[cfg.Serial].instance != wrapper {
		t.Fatal("manager copied or replaced the registry wrapper")
	}
	if wrapper.Instance != device || wrapper.GetDevice != device {
		t.Fatalf("wrapper instance changed: %#v", &common.Device{Instance: wrapper.Instance, GetDevice: wrapper.GetDevice})
	}
}

func TestSnapshotRemainsConsistentDuringReconciliation(t *testing.T) {
	useStorePath(t)
	cfg := testConfig("openrgb-import-snapshot", "Initial Device")
	cfg.ExternalSerial = "snapshot-external"
	if err := saveConfigStore(&ConfigStore{Devices: map[string]DeviceConfig{cfg.Serial: cfg}}); err != nil {
		t.Fatal(err)
	}
	device := testDevice(cfg)
	device.mu.Lock()
	device.RGBModes = append([]string(nil), rgbModes...)
	device.mu.Unlock()
	manager := newManager(map[string]*Device{cfg.Serial: device}, nil)
	configureTestManager(manager)
	var version atomic.Int32
	manager.discover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		if version.Add(1)%2 == 0 {
			return []openrgb.DiscoveredController{{
				ID: 72, Name: "Beta Device", Version: "2", Description: "Beta Description",
				Serial: cfg.ExternalSerial, Location: "beta-location", Vendor: "Vendor",
			}}, nil
		}
		return []openrgb.DiscoveredController{{
			ID: 71, Name: "Alpha Device", Version: "1", Description: "Alpha Description",
			Serial: cfg.ExternalSerial, Location: "alpha-location", Vendor: "Vendor",
		}}, nil
	}

	done := make(chan struct{})
	errorsFound := make(chan error, 1)
	var updates sync.WaitGroup
	updates.Add(1)
	go func() {
		defer updates.Done()
		defer close(done)
		for index := 0; index < 100; index++ {
			if _, err := manager.reconcile(); err != nil {
				select {
				case errorsFound <- err:
				default:
				}
				return
			}
		}
	}()

	checkSnapshot := func(snapshot DeviceSnapshot) error {
		if snapshot.Config == nil || len(snapshot.Config.Zones) != 1 {
			return errors.New("snapshot lost its device configuration")
		}
		switch snapshot.Product {
		case "Initial Device":
			if snapshot.Version != "" || snapshot.Description != "" || snapshot.Config.Product != "Initial Device" {
				return errors.New("initial snapshot mixed reconciliation metadata")
			}
		case "Alpha Device":
			if snapshot.Version != "1" || snapshot.Description != "Alpha Description" || snapshot.Config.Product != "Alpha Device" || snapshot.Config.Location != "alpha-location" {
				return errors.New("alpha snapshot mixed reconciliation metadata")
			}
		case "Beta Device":
			if snapshot.Version != "2" || snapshot.Description != "Beta Description" || snapshot.Config.Product != "Beta Device" || snapshot.Config.Location != "beta-location" {
				return errors.New("beta snapshot mixed reconciliation metadata")
			}
		default:
			return errors.New("snapshot contained an unexpected product")
		}
		return nil
	}

	for {
		select {
		case <-done:
			updates.Wait()
			select {
			case err := <-errorsFound:
				t.Fatal(err)
			default:
			}
			snapshot := device.Snapshot()
			if err := checkSnapshot(snapshot); err != nil {
				t.Fatal(err)
			}
			snapshot.Config.Zones[0].Name = "mutated"
			snapshot.RGBModes[0] = "mutated"
			next := device.Snapshot()
			if next.Config.Zones[0].Name == "mutated" || next.RGBModes[0] == "mutated" {
				t.Fatal("snapshot exposed mutable device slices")
			}
			return
		default:
			if err := checkSnapshot(device.Snapshot()); err != nil {
				t.Fatal(err)
			}
		}
	}
}
