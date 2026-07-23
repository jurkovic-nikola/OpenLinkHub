package openrgbimport

import (
	"LumenForge/src/common"
	"LumenForge/src/config"
	"LumenForge/src/logger"
	"LumenForge/src/openrgb"
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultReconcileTimeout = 10 * time.Second
	defaultHealthyInterval  = 15 * time.Second
	managerCommandCapacity  = 64
)

var defaultRetryBackoff = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	16 * time.Second,
	30 * time.Second,
}

type availabilityUpdater func(serial string, unavailable bool)
type presentationUpdater func(serial, product, firmware, image string)

type managerCommandKind uint8

const (
	managerCommandAdd managerCommandKind = iota
	managerCommandRemove
	managerCommandRefresh
)

type managerCommand struct {
	kind    managerCommandKind
	devices map[string]*Device
	result  chan error
	ctx     context.Context
	state   atomic.Uint32
}

type Manager struct {
	ctx    context.Context
	cancel context.CancelFunc

	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup
	trigger   chan struct{}
	commands  chan *managerCommand

	devices map[string]*Device

	discover           func(context.Context) ([]openrgb.DiscoveredController, error)
	updateAvailable    availabilityUpdater
	updatePresentation presentationUpdater
	resume             func(context.Context, *Device) error
	retryBackoff       []time.Duration
	healthyInterval    time.Duration
	reconcileTimeout   time.Duration

	logFailure    func(error)
	logRecovery   func()
	logDiagnostic func(string, error)

	diagnosticMessages  map[string]string
	commandClaimed      func(*managerCommand)
	commandBeforeResult func(*managerCommand)
}

func newManager(devices map[string]*Device, update availabilityUpdater) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	clonedDevices := make(map[string]*Device, len(devices))
	for serial, device := range devices {
		clonedDevices[serial] = device
	}
	return &Manager{
		ctx:                ctx,
		cancel:             cancel,
		trigger:            make(chan struct{}, 1),
		commands:           make(chan *managerCommand, managerCommandCapacity),
		devices:            clonedDevices,
		discover:           openrgb.DiscoverControllersContext,
		updateAvailable:    update,
		resume:             func(ctx context.Context, device *Device) error { return device.resumeDesiredState(ctx) },
		retryBackoff:       append([]time.Duration(nil), defaultRetryBackoff...),
		healthyInterval:    defaultHealthyInterval,
		reconcileTimeout:   defaultReconcileTimeout,
		diagnosticMessages: make(map[string]string),
		logFailure: func(err error) {
			logger.Log(logger.Fields{"error": err}).Warn("OpenRGB SDK server unavailable; configured imports will retry in the background")
		},
		logRecovery: func() {
			logger.Log(logger.Fields{}).Info("OpenRGB SDK server connection recovered")
		},
		logDiagnostic: func(serial string, err error) {
			logger.Log(logger.Fields{"serial": serial, "error": err}).Warn("Unable to reconcile configured OpenRGB import")
		},
	}
}

func (m *Manager) Start() {
	m.startOnce.Do(func() {
		m.wg.Add(1)
		go m.run()
	})
}

func (m *Manager) Stop() {
	m.stopOnce.Do(m.cancel)
	m.wg.Wait()
}

func (m *Manager) Trigger() {
	select {
	case m.trigger <- struct{}{}:
	default:
	}
}

func (m *Manager) sendCommand(ctx context.Context, kind managerCommandKind, devices map[string]*Device) error {
	if ctx == nil {
		ctx = context.Background()
	}
	command := &managerCommand{
		kind:    kind,
		devices: devices,
		result:  make(chan error, 1),
		ctx:     ctx,
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-m.ctx.Done():
		return fmt.Errorf("OpenRGB importer manager is stopped")
	case m.commands <- command:
	}

	select {
	case <-ctx.Done():
		return waitForDefinitiveCommandResult(command, ctx.Err())
	case <-m.ctx.Done():
		return waitForDefinitiveCommandResult(command, fmt.Errorf("OpenRGB importer manager is stopped"))
	case err := <-command.result:
		return err
	}
}

func waitForDefinitiveCommandResult(command *managerCommand, queuedError error) error {
	if command.state.CompareAndSwap(0, 2) {
		return queuedError
	}
	return <-command.result
}

func (m *Manager) Add(ctx context.Context, serial string, device *Device) error {
	return m.AddDevices(ctx, map[string]*Device{serial: device})
}

func (m *Manager) AddDevices(ctx context.Context, devices map[string]*Device) error {
	return m.sendCommand(ctx, managerCommandAdd, devices)
}

func (m *Manager) Remove(ctx context.Context, serial string, expected *Device) error {
	return m.RemoveDevices(ctx, map[string]*Device{serial: expected})
}

func (m *Manager) RemoveDevices(ctx context.Context, devices map[string]*Device) error {
	return m.sendCommand(ctx, managerCommandRemove, devices)
}

func (m *Manager) Refresh(ctx context.Context) error {
	return m.sendCommand(ctx, managerCommandRefresh, nil)
}

func (m *Manager) run() {
	defer m.wg.Done()

	failureLogged := false
	failureIndex := 0
	partialFailureIndex := 0
	for {
		partialFailure, err := m.reconcile()
		if err != nil {
			if m.ctx.Err() != nil {
				return
			}
			openrgb.SetDisconnected(err)
			m.markAllUnavailable()
			partialFailureIndex = 0
			if !failureLogged {
				m.logFailure(err)
				failureLogged = true
			}

			delay := m.retryBackoff[len(m.retryBackoff)-1]
			if failureIndex < len(m.retryBackoff) {
				delay = m.retryBackoff[failureIndex]
				failureIndex++
			}
			if !m.wait(delay) {
				return
			}
			continue
		}

		if failureLogged {
			m.logRecovery()
			failureLogged = false
		}
		failureIndex = 0
		if partialFailure {
			delay := m.retryBackoff[len(m.retryBackoff)-1]
			if partialFailureIndex < len(m.retryBackoff) {
				delay = m.retryBackoff[partialFailureIndex]
				partialFailureIndex++
			}
			if !m.wait(delay) {
				return
			}
			continue
		}
		partialFailureIndex = 0
		if !m.wait(m.healthyInterval) {
			return
		}
	}
}

func (m *Manager) wait(delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return false
		case command := <-m.commands:
			needsReconcile := m.handleCommand(command)
			draining := true
			for draining {
				select {
				case next := <-m.commands:
					if m.handleCommand(next) {
						needsReconcile = true
					}
				default:
					draining = false
				}
			}
			if needsReconcile {
				return true
			}
		case <-m.trigger:
			return true
		case <-timer.C:
			return true
		}
	}
}

func (m *Manager) handleCommand(command *managerCommand) bool {
	if !command.state.CompareAndSwap(0, 1) {
		command.result <- commandContextError(command)
		return false
	}
	if m.commandClaimed != nil {
		m.commandClaimed(command)
	}

	var err error
	needsReconcile := false
	switch command.kind {
	case managerCommandAdd:
		if len(command.devices) == 0 {
			err = fmt.Errorf("OpenRGB manager add command has no devices")
			break
		}
		for serial, device := range command.devices {
			if device == nil {
				err = fmt.Errorf("cannot add nil OpenRGB import %q to manager", serial)
				break
			}
			if !common.AlphanumericDashRegex.MatchString(serial) || device.Serial != serial {
				err = fmt.Errorf("OpenRGB manager add serial %q is invalid or does not match device %q", serial, device.Serial)
				break
			}
			if existing, ok := m.devices[serial]; ok && existing != device {
				err = fmt.Errorf("OpenRGB manager already contains a different device for serial %q", serial)
				break
			}
		}
		if err == nil {
			for serial, device := range command.devices {
				m.devices[serial] = device
			}
			needsReconcile = len(m.devices) > 0
		}
	case managerCommandRemove:
		if len(command.devices) == 0 {
			err = fmt.Errorf("OpenRGB manager remove command has no devices")
			break
		}
		for serial, expected := range command.devices {
			if expected == nil {
				err = fmt.Errorf("cannot remove nil OpenRGB import %q from manager", serial)
				break
			}
			existing, ok := m.devices[serial]
			if !ok {
				err = fmt.Errorf("OpenRGB manager does not contain serial %q", serial)
				break
			}
			if existing != expected {
				err = fmt.Errorf("OpenRGB manager device pointer mismatch for serial %q", serial)
				break
			}
		}
		if err == nil {
			for serial := range command.devices {
				delete(m.devices, serial)
				delete(m.diagnosticMessages, serial)
			}
			needsReconcile = len(m.devices) > 0
		}
	case managerCommandRefresh:
		needsReconcile = len(m.devices) > 0
	default:
		err = fmt.Errorf("unknown OpenRGB manager command")
	}
	if m.commandBeforeResult != nil {
		m.commandBeforeResult(command)
	}
	command.result <- err
	return err == nil && needsReconcile
}

func commandContextError(command *managerCommand) error {
	if command.ctx != nil && command.ctx.Err() != nil {
		return command.ctx.Err()
	}
	return fmt.Errorf("OpenRGB manager command was canceled")
}

func (m *Manager) reconcile() (bool, error) {
	if len(m.devices) == 0 {
		return false, nil
	}
	if err := m.ctx.Err(); err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(m.ctx, m.reconcileTimeout)
	defer cancel()

	discovered, err := m.discover(ctx)
	if err != nil {
		return false, err
	}
	if err = ctx.Err(); err != nil {
		return false, err
	}
	openrgb.SetConnected()

	matches, diagnostics := matchConfiguredDevices(m.devices, discovered)
	serials := make([]string, 0, len(m.devices))
	for serial := range m.devices {
		serials = append(serials, serial)
	}
	sort.Strings(serials)
	partialFailure := false
	for _, serial := range serials {
		device := m.devices[serial]
		if err = ctx.Err(); err != nil {
			return false, err
		}
		controller, matched := matches[serial]
		if !matched {
			device.markUnavailable()
			m.setAvailability(serial, true)
			if diagnostic, ok := diagnostics[serial]; ok {
				m.reportDiagnostic(serial, diagnostic)
			} else {
				m.reportDiagnostic(serial, fmt.Errorf("configured controller was not present in the OpenRGB SDK device list"))
			}
			continue
		}

		changed := device.bindController(controller)
		product, firmware, image := device.wrapperPresentation()
		m.setPresentation(serial, product, firmware, image)
		if changed {
			if err = m.resume(ctx, device); err != nil {
				if ctx.Err() != nil {
					return false, ctx.Err()
				}
				device.markUnavailable()
				m.setAvailability(serial, true)
				m.reportDiagnostic(serial, fmt.Errorf("restore desired state: %w", err))
				partialFailure = true
				continue
			}
		}
		delete(m.diagnosticMessages, serial)
		m.setAvailability(serial, false)
	}

	if err = ctx.Err(); err != nil {
		return false, err
	}
	if err = persistIdentityMetadata(matches); err != nil {
		m.reportDiagnostic("store", err)
	}
	return partialFailure, nil
}

func (m *Manager) markAllUnavailable() {
	for serial, device := range m.devices {
		device.markUnavailable()
		m.setAvailability(serial, true)
	}
}

func (m *Manager) setAvailability(serial string, unavailable bool) {
	if m.updateAvailable != nil {
		m.updateAvailable(serial, unavailable)
	}
}

func (m *Manager) setPresentation(serial, product, firmware, image string) {
	if m.updatePresentation != nil {
		m.updatePresentation(serial, product, firmware, image)
	}
}

func (m *Manager) reportDiagnostic(serial string, err error) {
	message := err.Error()
	if m.diagnosticMessages[serial] == message {
		return
	}
	m.diagnosticMessages[serial] = message
	m.logDiagnostic(serial, err)
}

type matchStage func(*DeviceConfig, openrgb.DiscoveredController) bool

func matchConfiguredDevices(devices map[string]*Device, discovered []openrgb.DiscoveredController) (map[string]openrgb.DiscoveredController, map[string]error) {
	configs := make(map[string]*DeviceConfig, len(devices))
	serials := make([]string, 0, len(devices))
	for serial, device := range devices {
		device.mu.Lock()
		configs[serial] = cloneDeviceConfig(device.Config)
		device.mu.Unlock()
		serials = append(serials, serial)
	}
	sort.Strings(serials)

	matches := make(map[string]openrgb.DiscoveredController, len(devices))
	usedControllers := make(map[int]bool)
	diagnostics := make(map[string]error)

	stages := []matchStage{
		func(cfg *DeviceConfig, dc openrgb.DiscoveredController) bool {
			if cfg == nil {
				return false
			}
			configuredSerial := usableExternalSerial(cfg.ExternalSerial)
			discoveredSerial := usableExternalSerial(dc.Serial)
			return configuredSerial != "" && discoveredSerial != "" &&
				normalizeIdentityField(configuredSerial) == normalizeIdentityField(discoveredSerial)
		},
		func(cfg *DeviceConfig, dc openrgb.DiscoveredController) bool {
			if cfg == nil || normalizeIdentityField(cfg.Location) == "" ||
				normalizeIdentityField(cfg.Location) != normalizeIdentityField(dc.Location) {
				return false
			}
			return identifyingMetadataMatches(cfg, dc)
		},
		func(cfg *DeviceConfig, dc openrgb.DiscoveredController) bool {
			return cfg != nil && cfg.Serial == internalSerialForController(dc)
		},
		func(cfg *DeviceConfig, dc openrgb.DiscoveredController) bool {
			return cfg != nil && strings.TrimSpace(cfg.Product) != "" && identifyingMetadataMatches(cfg, dc)
		},
	}

	for stageIndex, stage := range stages {
		claims := make(map[int][]string)
		candidateFor := make(map[string]int)
		for _, serial := range serials {
			if _, matched := matches[serial]; matched {
				continue
			}
			candidates := make([]int, 0, 1)
			for index, controller := range discovered {
				if usedControllers[index] || !stage(configs[serial], controller) {
					continue
				}
				candidates = append(candidates, index)
			}
			if len(candidates) == 1 {
				candidateFor[serial] = candidates[0]
				claims[candidates[0]] = append(claims[candidates[0]], serial)
			} else if stageIndex == len(stages)-1 && len(candidates) > 1 {
				diagnostics[serial] = fmt.Errorf("%d indistinguishable OpenRGB controllers match this configured import", len(candidates))
			}
		}

		for _, serial := range serials {
			controllerIndex, ok := candidateFor[serial]
			if !ok {
				continue
			}
			if len(claims[controllerIndex]) != 1 {
				if stageIndex == len(stages)-1 {
					diagnostics[serial] = fmt.Errorf("OpenRGB controller identity is shared by configured imports %s", strings.Join(claims[controllerIndex], ", "))
				}
				continue
			}
			matches[serial] = discovered[controllerIndex]
			usedControllers[controllerIndex] = true
			delete(diagnostics, serial)
		}
	}

	return matches, diagnostics
}

func identifyingMetadataMatches(cfg *DeviceConfig, controller openrgb.DiscoveredController) bool {
	if normalizeIdentityField(cfg.Product) != normalizeIdentityField(controller.Name) {
		return false
	}
	return normalizeIdentityField(cfg.Vendor) == "" ||
		normalizeIdentityField(cfg.Vendor) == normalizeIdentityField(controller.Vendor)
}

func internalSerialForController(controller openrgb.DiscoveredController) string {
	if isLegacyASUSMotherboardImport(controller.Name, controller.Vendor) {
		return "openrgb-mobo-1"
	}
	hashInput := fmt.Sprintf("%s|%s|%s|%s|%d", controller.Name, controller.Vendor, controller.Version, controller.Description, len(controller.Zones))
	hash := sha256.Sum256([]byte(hashInput))
	return fmt.Sprintf("openrgb-hash-%x", hash[:16])
}

func persistIdentityMetadata(matches map[string]openrgb.DiscoveredController) error {
	if len(matches) == 0 {
		return nil
	}
	return updateConfigStoreIfChanged(func(store *ConfigStore) (bool, error) {
		changed := false
		for serial, controller := range matches {
			cfg, ok := store.Devices[serial]
			if !ok {
				continue
			}
			sanitizedExternal := safePresentationString(usableExternalSerial(cfg.ExternalSerial), 512)
			if cfg.ExternalSerial != sanitizedExternal {
				cfg.ExternalSerial = sanitizedExternal
				changed = true
			}
			if externalSerial := usableExternalSerial(controller.Serial); externalSerial != "" {
				externalSerial = safePresentationString(externalSerial, 512)
				if cfg.ExternalSerial != externalSerial {
					cfg.ExternalSerial = externalSerial
					changed = true
				}
			}
			if location := safePresentationString(controller.Location, 512); location != "" {
				if cfg.Location != location {
					cfg.Location = location
					changed = true
				}
			}
			if vendor := safePresentationString(controller.Vendor, 512); vendor != "" {
				if cfg.Vendor != vendor {
					cfg.Vendor = vendor
					changed = true
				}
			}
			if product := safePresentationString(controller.Name, 512); product != "" {
				if cfg.Product != product {
					cfg.Product = product
					changed = true
				}
			}
			store.Devices[serial] = cfg
		}
		return changed, nil
	})
}

var (
	configuredDevicesMutex   sync.RWMutex
	configuredDevices        = make(map[string]*Device)
	activeManagerMutex       sync.RWMutex
	activeManager            *Manager
	managerAvailability      availabilityUpdater
	managerPresentation      presentationUpdater
	managerFactory           = newManager
	localTargetServerEnabled = func() bool {
		return config.GetConfig().EnableOpenRGBTargetServer
	}
)

func setConfiguredDevices(devices map[string]*Device) {
	configuredDevicesMutex.Lock()
	defer configuredDevicesMutex.Unlock()
	configuredDevices = make(map[string]*Device, len(devices))
	for serial, device := range devices {
		configuredDevices[serial] = device
	}
}

func configuredDevicesSnapshot() map[string]*Device {
	configuredDevicesMutex.RLock()
	defer configuredDevicesMutex.RUnlock()
	result := make(map[string]*Device, len(configuredDevices))
	for serial, device := range configuredDevices {
		result[serial] = device
	}
	return result
}

func addConfiguredDevices(devices map[string]*Device) error {
	configuredDevicesMutex.Lock()
	defer configuredDevicesMutex.Unlock()

	for serial, device := range devices {
		if device == nil || device.Serial != serial || !common.AlphanumericDashRegex.MatchString(serial) {
			return fmt.Errorf("invalid configured OpenRGB import %q", serial)
		}
		if existing, ok := configuredDevices[serial]; ok && existing != device {
			return fmt.Errorf("configured OpenRGB import %q already references a different object", serial)
		}
	}
	for serial, device := range devices {
		configuredDevices[serial] = device
	}
	return nil
}

func removeConfiguredDevices(devices map[string]*Device) error {
	configuredDevicesMutex.Lock()
	defer configuredDevicesMutex.Unlock()

	for serial, expected := range devices {
		if expected == nil {
			return fmt.Errorf("cannot remove nil configured OpenRGB import %q", serial)
		}
		existing, ok := configuredDevices[serial]
		if !ok {
			return fmt.Errorf("configured OpenRGB import %q is missing", serial)
		}
		if existing != expected {
			return fmt.Errorf("configured OpenRGB import %q references a different object", serial)
		}
	}
	for serial := range devices {
		delete(configuredDevices, serial)
	}
	return nil
}

func configuredDevice(serial string) (*Device, bool) {
	configuredDevicesMutex.RLock()
	defer configuredDevicesMutex.RUnlock()
	device, ok := configuredDevices[serial]
	return device, ok
}

func enabledConfiguredCount() int {
	configuredDevicesMutex.RLock()
	defer configuredDevicesMutex.RUnlock()
	return len(configuredDevices)
}

func targetServerConflictError() error {
	return fmt.Errorf("OpenRGB imports cannot use 127.0.0.1:%d while the LumenForge OpenRGB target server is enabled", config.GetConfig().OpenRGBPort)
}

func managerCallbacks() (availabilityUpdater, presentationUpdater) {
	activeManagerMutex.RLock()
	defer activeManagerMutex.RUnlock()
	return managerAvailability, managerPresentation
}

func installManager(devices map[string]*Device) (*Manager, bool) {
	update, presentation := managerCallbacks()
	candidate := managerFactory(devices, update)
	candidate.updatePresentation = presentation

	activeManagerMutex.Lock()
	if activeManager != nil {
		manager := activeManager
		activeManagerMutex.Unlock()
		candidate.Stop()
		return manager, false
	}
	activeManager = candidate
	candidate.Start()
	activeManagerMutex.Unlock()
	return candidate, true
}

// StartManager starts asynchronous reconciliation after devices.Init has registered placeholders.
func StartManager(update availabilityUpdater, updatePresentation presentationUpdater) {
	devices := configuredDevicesSnapshot()
	activeManagerMutex.Lock()
	managerAvailability = update
	managerPresentation = updatePresentation
	alreadyActive := activeManager != nil
	activeManagerMutex.Unlock()
	if alreadyActive {
		logger.Log(logger.Fields{}).Warn("OpenRGB importer manager is already running; ignoring duplicate start")
		return
	}
	if len(devices) == 0 {
		return
	}
	if localTargetServerEnabled() {
		err := targetServerConflictError()
		openrgb.SetDisconnected(err)
		for serial := range devices {
			if update != nil {
				update(serial, true)
			}
		}
		logger.Log(logger.Fields{"error": err}).Error("OpenRGB importer disabled to prevent a local target-server self-connection")
		return
	}

	_, installed := installManager(devices)
	if !installed {
		logger.Log(logger.Fields{}).Warn("OpenRGB importer manager is already running; ignoring duplicate start")
	}
}

// StopManager cancels and joins the importer worker. It is safe to call repeatedly.
func StopManager() {
	activeManagerMutex.Lock()
	manager := activeManager
	activeManager = nil
	activeManagerMutex.Unlock()
	if manager != nil {
		manager.Stop()
	}
}

func addManagerDevices(ctx context.Context, devices map[string]*Device) (bool, error) {
	if len(devices) == 0 {
		return false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if localTargetServerEnabled() {
		return false, targetServerConflictError()
	}

	activeManagerMutex.RLock()
	manager := activeManager
	activeManagerMutex.RUnlock()
	if manager != nil {
		return false, manager.AddDevices(ctx, devices)
	}

	allConfigured := configuredDevicesSnapshot()
	manager, installed := installManager(allConfigured)
	if installed {
		return true, nil
	}
	return false, manager.AddDevices(ctx, devices)
}

func removeManagerDevices(ctx context.Context, devices map[string]*Device) error {
	if len(devices) == 0 {
		return nil
	}
	activeManagerMutex.RLock()
	manager := activeManager
	activeManagerMutex.RUnlock()
	if manager == nil {
		return nil
	}
	return manager.RemoveDevices(ctx, devices)
}

func stopManagerIfNoConfigured() bool {
	if enabledConfiguredCount() != 0 {
		return false
	}
	activeManagerMutex.Lock()
	manager := activeManager
	activeManager = nil
	activeManagerMutex.Unlock()
	if manager != nil {
		manager.Stop()
	}
	openrgb.SetNotConfigured()
	return true
}

// RefreshManager requests one coalesced reconciliation without discovering or
// importing any new controller.
func RefreshManager(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if localTargetServerEnabled() {
		return targetServerConflictError()
	}
	devices := configuredDevicesSnapshot()
	if len(devices) == 0 {
		openrgb.SetNotConfigured()
		return fmt.Errorf("OpenRGB imports are not configured")
	}

	activeManagerMutex.RLock()
	manager := activeManager
	activeManagerMutex.RUnlock()
	if manager == nil {
		var installed bool
		manager, installed = installManager(devices)
		if installed {
			return nil
		}
	}
	return manager.Refresh(ctx)
}

func requestReconciliation() {
	activeManagerMutex.RLock()
	manager := activeManager
	activeManagerMutex.RUnlock()
	if manager != nil {
		manager.Trigger()
	}
}

func reportOutputFailure(device *Device, _ error) {
	activeManagerMutex.RLock()
	manager := activeManager
	activeManagerMutex.RUnlock()
	if manager == nil {
		return
	}
	manager.setAvailability(device.Serial, true)
	manager.Trigger()
}
