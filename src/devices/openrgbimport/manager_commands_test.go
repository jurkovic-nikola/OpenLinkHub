package openrgbimport

import (
	"LumenForge/src/openrgb"
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func commandTestManager(initial map[string]*Device) *Manager {
	manager := newManager(initial, nil)
	configureTestManager(manager)
	manager.healthyInterval = time.Hour
	manager.discover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		return nil, nil
	}
	return manager
}

func TestManagerMembershipCommandsAcknowledgeAtomically(t *testing.T) {
	first := testDevice(testConfig("openrgb-command-first", "First"))
	second := testDevice(testConfig("openrgb-command-second", "Second"))
	manager := commandTestManager(map[string]*Device{first.Serial: first})
	manager.Start()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Add(ctx, first.Serial, first); err != nil {
		t.Fatalf("same-object add: %v", err)
	}
	if err := manager.Add(ctx, second.Serial, second); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := manager.Refresh(ctx); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if err := manager.Remove(ctx, second.Serial, second); err != nil {
		t.Fatalf("remove: %v", err)
	}
	manager.Stop()

	if len(manager.devices) != 1 || manager.devices[first.Serial] != first {
		t.Fatalf("manager membership = %#v", manager.devices)
	}
}

func TestManagerMembershipCommandsRejectPointerCollisions(t *testing.T) {
	first := testDevice(testConfig("openrgb-command-collision", "First"))
	replacement := testDevice(testConfig(first.Serial, "Replacement"))
	manager := commandTestManager(map[string]*Device{first.Serial: first})
	manager.Start()
	t.Cleanup(manager.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Add(ctx, first.Serial, replacement); err == nil || !strings.Contains(err.Error(), "different device") {
		t.Fatalf("different-object add error = %v", err)
	}
	if err := manager.Remove(ctx, first.Serial, replacement); err == nil || !strings.Contains(err.Error(), "pointer mismatch") {
		t.Fatalf("pointer-mismatch remove error = %v", err)
	}
}

func TestManagerCommandWaitsForBoundedReconciliation(t *testing.T) {
	first := testDevice(testConfig("openrgb-command-blocked", "First"))
	second := testDevice(testConfig("openrgb-command-after-block", "Second"))
	manager := commandTestManager(map[string]*Device{first.Serial: first})
	started := make(chan struct{})
	release := make(chan struct{})
	manager.discover = func(ctx context.Context) ([]openrgb.DiscoveredController, error) {
		select {
		case <-started:
		default:
			close(started)
		}
		select {
		case <-release:
			return nil, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	manager.Start()
	<-started

	result := make(chan error, 1)
	go func() {
		result <- manager.Add(context.Background(), second.Serial, second)
	}()
	select {
	case err := <-result:
		t.Fatalf("command returned before reconciliation completed: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("command after reconciliation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("command was not acknowledged")
	}
	manager.Stop()
}

func TestManagerStopUnblocksPendingCommand(t *testing.T) {
	first := testDevice(testConfig("openrgb-command-stop", "First"))
	second := testDevice(testConfig("openrgb-command-pending", "Second"))
	manager := commandTestManager(map[string]*Device{first.Serial: first})
	started := make(chan struct{})
	manager.discover = func(ctx context.Context) ([]openrgb.DiscoveredController, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	manager.Start()
	<-started

	var completed atomic.Bool
	result := make(chan error, 1)
	go func() {
		err := manager.Add(context.Background(), second.Serial, second)
		completed.Store(true)
		result <- err
	}()
	waitFor(t, time.Second, func() bool { return len(manager.commands) == 1 })
	manager.Stop()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("pending command unexpectedly succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("pending command did not unblock on Stop")
	}
	if !completed.Load() {
		t.Fatal("pending command goroutine did not complete")
	}
}

func TestCanceledManagerCommandCannotMutateLater(t *testing.T) {
	first := testDevice(testConfig("openrgb-command-cancel-first", "First"))
	second := testDevice(testConfig("openrgb-command-cancel-second", "Second"))
	manager := commandTestManager(map[string]*Device{first.Serial: first})
	started := make(chan struct{})
	release := make(chan struct{})
	manager.discover = func(ctx context.Context) ([]openrgb.DiscoveredController, error) {
		select {
		case <-started:
		default:
			close(started)
		}
		select {
		case <-release:
			return nil, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	manager.Start()
	<-started

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- manager.Add(ctx, second.Serial, second)
	}()
	waitFor(t, time.Second, func() bool { return len(manager.commands) == 1 })
	cancel()
	if err := <-result; err == nil {
		t.Fatal("canceled command unexpectedly succeeded")
	}
	close(release)
	waitFor(t, time.Second, func() bool {
		return len(manager.commands) == 0
	})
	manager.Stop()
	if _, ok := manager.devices[second.Serial]; ok {
		t.Fatal("canceled queued command mutated manager membership later")
	}
}

func TestClaimedManagerCommandsReturnDefinitiveResultDuringStop(t *testing.T) {
	tests := []struct {
		name    string
		initial func() map[string]*Device
		command func(context.Context, *Manager, *Device) error
		present bool
	}{
		{
			name:    "add",
			initial: func() map[string]*Device { return map[string]*Device{} },
			command: func(ctx context.Context, manager *Manager, device *Device) error {
				return manager.Add(ctx, device.Serial, device)
			},
			present: true,
		},
		{
			name: "remove",
			initial: func() map[string]*Device {
				device := testDevice(testConfig("openrgb-claimed-remove", "Remove"))
				return map[string]*Device{device.Serial: device}
			},
			command: func(ctx context.Context, manager *Manager, device *Device) error {
				return manager.Remove(ctx, device.Serial, device)
			},
			present: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			initial := test.initial()
			var device *Device
			if test.name == "add" {
				device = testDevice(testConfig("openrgb-claimed-add", "Add"))
			} else {
				for _, existing := range initial {
					device = existing
				}
			}
			manager := commandTestManager(initial)
			claimed := make(chan struct{})
			applied := make(chan struct{})
			release := make(chan struct{})
			manager.commandClaimed = func(*managerCommand) {
				close(claimed)
			}
			manager.commandBeforeResult = func(*managerCommand) {
				close(applied)
				<-release
			}
			manager.Start()

			result := make(chan error, 1)
			go func() {
				result <- test.command(context.Background(), manager, device)
			}()
			select {
			case <-claimed:
			case <-time.After(time.Second):
				t.Fatal("worker did not claim command")
			}
			select {
			case <-applied:
			case <-time.After(time.Second):
				t.Fatal("worker did not apply command")
			}
			stopDone := make(chan struct{})
			go func() {
				manager.Stop()
				close(stopDone)
			}()
			close(release)

			select {
			case err := <-result:
				if err != nil {
					t.Fatalf("claimed command reported failure during Stop: %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("claimed command caller did not terminate")
			}
			select {
			case <-stopDone:
			case <-time.After(time.Second):
				t.Fatal("manager Stop did not terminate")
			}
			_, present := manager.devices[device.Serial]
			if present != test.present {
				t.Fatalf("membership present=%v, want %v", present, test.present)
			}
		})
	}
}

func TestFinalManagerRemovalDoesNotDiscoverAndLaterAddDoes(t *testing.T) {
	first := testDevice(testConfig("openrgb-final-remove", "First"))
	second := testDevice(testConfig("openrgb-after-empty", "Second"))
	manager := commandTestManager(map[string]*Device{first.Serial: first})
	var discoveries atomic.Int32
	initialDiscovery := make(chan struct{})
	manager.discover = func(context.Context) ([]openrgb.DiscoveredController, error) {
		count := discoveries.Add(1)
		if count == 1 {
			close(initialDiscovery)
		}
		return nil, nil
	}
	manager.Start()
	<-initialDiscovery

	if err := manager.Remove(context.Background(), first.Serial, first); err != nil {
		t.Fatal(err)
	}
	afterRemoval := discoveries.Load()
	time.Sleep(30 * time.Millisecond)
	if got := discoveries.Load(); got != afterRemoval {
		t.Fatalf("final removal triggered discovery: before=%d after=%d", afterRemoval, got)
	}

	if err := manager.Add(context.Background(), second.Serial, second); err != nil {
		t.Fatal(err)
	}
	waitFor(t, time.Second, func() bool {
		return discoveries.Load() > afterRemoval
	})
	manager.Stop()
}

func TestDrainedRemoveAllThenAddReconcilesNewMembership(t *testing.T) {
	first := testDevice(testConfig("openrgb-drain-remove", "First"))
	second := testDevice(testConfig("openrgb-drain-add", "Second"))
	manager := commandTestManager(map[string]*Device{first.Serial: first})
	var discoveries atomic.Int32
	initialStarted := make(chan struct{})
	releaseInitial := make(chan struct{})
	manager.discover = func(ctx context.Context) ([]openrgb.DiscoveredController, error) {
		if discoveries.Add(1) == 1 {
			close(initialStarted)
			select {
			case <-releaseInitial:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		return nil, nil
	}
	manager.Start()
	<-initialStarted

	removeCommand := &managerCommand{
		kind:    managerCommandRemove,
		devices: map[string]*Device{first.Serial: first},
		result:  make(chan error, 1),
		ctx:     context.Background(),
	}
	addCommand := &managerCommand{
		kind:    managerCommandAdd,
		devices: map[string]*Device{second.Serial: second},
		result:  make(chan error, 1),
		ctx:     context.Background(),
	}
	manager.commands <- removeCommand
	manager.commands <- addCommand
	close(releaseInitial)
	if err := <-removeCommand.result; err != nil {
		t.Fatal(err)
	}
	if err := <-addCommand.result; err != nil {
		t.Fatal(err)
	}
	waitFor(t, time.Second, func() bool {
		return discoveries.Load() >= 2
	})
	manager.Stop()
	if len(manager.devices) != 1 || manager.devices[second.Serial] != second {
		t.Fatalf("drained membership = %#v", manager.devices)
	}
}

func TestFirstManagerStartAndLastConfiguredStopAreSingular(t *testing.T) {
	StopManager()
	setConfiguredDevices(nil)
	previousTarget := localTargetServerEnabled
	localTargetServerEnabled = func() bool { return false }
	previousFactory := managerFactory
	var factories atomic.Int32
	managerFactory = func(devices map[string]*Device, update availabilityUpdater) *Manager {
		factories.Add(1)
		manager := commandTestManager(devices)
		manager.updateAvailable = update
		return manager
	}
	t.Cleanup(func() {
		StopManager()
		setConfiguredDevices(nil)
		localTargetServerEnabled = previousTarget
		managerFactory = previousFactory
	})

	first := testDevice(testConfig("openrgb-first-live", "First"))
	second := testDevice(testConfig("openrgb-second-live", "Second"))
	if err := addConfiguredDevices(map[string]*Device{first.Serial: first}); err != nil {
		t.Fatal(err)
	}
	started, err := addManagerDevices(context.Background(), map[string]*Device{first.Serial: first})
	if err != nil || !started {
		t.Fatalf("first manager add = started %v, error %v", started, err)
	}
	if err = addConfiguredDevices(map[string]*Device{second.Serial: second}); err != nil {
		t.Fatal(err)
	}
	started, err = addManagerDevices(context.Background(), map[string]*Device{second.Serial: second})
	if err != nil || started {
		t.Fatalf("second manager add = started %v, error %v", started, err)
	}
	if factories.Load() != 1 {
		t.Fatalf("manager factories = %d, want 1", factories.Load())
	}

	if err = removeManagerDevices(context.Background(), map[string]*Device{first.Serial: first}); err != nil {
		t.Fatal(err)
	}
	if err = removeConfiguredDevices(map[string]*Device{first.Serial: first}); err != nil {
		t.Fatal(err)
	}
	if stopManagerIfNoConfigured() {
		t.Fatal("manager stopped with one configured import remaining")
	}
	if err = removeManagerDevices(context.Background(), map[string]*Device{second.Serial: second}); err != nil {
		t.Fatal(err)
	}
	if err = removeConfiguredDevices(map[string]*Device{second.Serial: second}); err != nil {
		t.Fatal(err)
	}
	if !stopManagerIfNoConfigured() {
		t.Fatal("last configured removal did not stop the manager")
	}
	activeManagerMutex.RLock()
	manager := activeManager
	activeManagerMutex.RUnlock()
	if manager != nil {
		t.Fatal("manager remained active after last removal")
	}
	state, statusErr := openrgb.GetStatus()
	if state != openrgb.StateNotConfigured || statusErr != nil {
		t.Fatalf("status = %q, %v; want Not Configured", state, statusErr)
	}
}
