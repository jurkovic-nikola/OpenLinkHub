package cleanup

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestConcurrentStopsRunEachComponentOnce(t *testing.T) {
	var coordinator Coordinator
	var deviceStops atomic.Int32
	var inputStops atomic.Int32
	coordinator.RegisterDevices(func() { deviceStops.Add(1) })
	coordinator.RegisterInput(func() { inputStops.Add(1) })

	var calls sync.WaitGroup
	for i := 0; i < 50; i++ {
		calls.Add(2)
		go func() {
			defer calls.Done()
			coordinator.StopDevices()
		}()
		go func() {
			defer calls.Done()
			coordinator.StopInput()
		}()
	}
	calls.Wait()

	if got := deviceStops.Load(); got != 1 {
		t.Fatalf("device cleanup ran %d times, want 1", got)
	}
	if got := inputStops.Load(); got != 1 {
		t.Fatalf("input cleanup ran %d times, want 1", got)
	}
}

func TestStopBeforeRegistrationRunsAfterRegistration(t *testing.T) {
	var coordinator Coordinator
	var stops atomic.Int32

	coordinator.StopDevices()
	coordinator.RegisterDevices(func() { stops.Add(1) })
	coordinator.StopDevices()

	if got := stops.Load(); got != 1 {
		t.Fatalf("device cleanup ran %d times, want 1", got)
	}
}

func TestConcurrentRegistrationAndStopsRunOnce(t *testing.T) {
	var coordinator Coordinator
	var stops atomic.Int32
	start := make(chan struct{})
	var calls sync.WaitGroup

	calls.Add(1)
	go func() {
		defer calls.Done()
		<-start
		coordinator.RegisterInput(func() { stops.Add(1) })
	}()
	for i := 0; i < 50; i++ {
		calls.Add(1)
		go func() {
			defer calls.Done()
			<-start
			coordinator.StopInput()
		}()
	}
	close(start)
	calls.Wait()
	coordinator.StopInput()

	if got := stops.Load(); got != 1 {
		t.Fatalf("input cleanup ran %d times, want 1", got)
	}
}
