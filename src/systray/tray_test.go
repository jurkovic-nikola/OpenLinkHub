package systray

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBatterySyncStopsWhileWaitingForReady(t *testing.T) {
	ready := make(chan struct{})
	stop := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(1)
	var syncs atomic.Int32

	go func() {
		defer workers.Done()
		runBatterySync(ready, stop, time.Hour, func() { syncs.Add(1) })
	}()

	stopReturned := make(chan struct{})
	go func() {
		stopLocalTray(stop, &workers)
		close(stopReturned)
	}()
	select {
	case <-stopReturned:
	case <-time.After(time.Second):
		t.Fatal("Stop did not release worker waiting for ready")
	}
	if got := syncs.Load(); got != 0 {
		t.Fatalf("battery sync ran %d times, want 0", got)
	}
}

func TestBatterySyncStopsAfterReady(t *testing.T) {
	ready := make(chan struct{})
	stop := make(chan struct{})
	started := make(chan struct{})
	release := make(chan struct{})
	stopReturned := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(1)
	var syncs atomic.Int32

	go func() {
		defer workers.Done()
		runBatterySync(ready, stop, time.Hour, func() {
			if syncs.Add(1) == 1 {
				close(started)
				<-release
			}
		})
	}()
	close(ready)

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("initial battery sync did not run")
	}
	go func() {
		stopLocalTray(stop, &workers)
		close(stopReturned)
	}()
	premature := false
	select {
	case <-stopReturned:
		premature = true
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	if !premature {
		select {
		case <-stopReturned:
		case <-time.After(time.Second):
			t.Fatal("Stop did not wait for battery worker to exit")
		}
	}
	if premature {
		t.Fatal("Stop returned while battery worker was running")
	}
}

func TestStopWaitsForBothTrackedWorkers(t *testing.T) {
	stop := make(chan struct{})
	firstRelease := make(chan struct{})
	secondRelease := make(chan struct{})
	stopReturned := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(2)

	go func() {
		defer workers.Done()
		<-firstRelease
	}()
	go func() {
		defer workers.Done()
		<-secondRelease
	}()
	go func() {
		stopLocalTray(stop, &workers)
		close(stopReturned)
	}()

	close(firstRelease)
	premature := false
	select {
	case <-stopReturned:
		premature = true
	case <-time.After(20 * time.Millisecond):
	}
	close(secondRelease)
	if !premature {
		select {
		case <-stopReturned:
		case <-time.After(time.Second):
			t.Fatal("Stop did not return after both workers exited")
		}
	}
	if premature {
		t.Fatal("Stop returned before both workers exited")
	}
}

func stopLocalTray(stop chan struct{}, workers *sync.WaitGroup) {
	var stateMutex sync.Mutex
	var stopped bool
	var stopOnce sync.Once
	stopTray(&stateMutex, &stopped, &stopOnce, stop, func() {}, workers)
}
