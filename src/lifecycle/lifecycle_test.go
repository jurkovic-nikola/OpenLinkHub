package lifecycle

import (
	"sync"
	"testing"
)

func TestCoordinatorFirstRequestWins(t *testing.T) {
	coordinator := NewCoordinator()
	coordinator.Request(1)
	coordinator.Request(0)

	if status := coordinator.Wait(); status != 1 {
		t.Fatalf("Wait() returned %d, want 1", status)
	}
}

func TestCoordinatorConcurrentRequestsCloseDoneOnce(t *testing.T) {
	coordinator := NewCoordinator()
	var requests sync.WaitGroup

	for i := 0; i < 20; i++ {
		requests.Add(1)
		go func(status int) {
			defer requests.Done()
			coordinator.Request(status)
		}(i % 2)
	}

	requests.Wait()
	select {
	case <-coordinator.Done():
	default:
		t.Fatal("Done() was not closed")
	}

	status := coordinator.Wait()
	if status != 0 && status != 1 {
		t.Fatalf("Wait() returned unexpected status %d", status)
	}
}
