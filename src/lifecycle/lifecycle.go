// Package lifecycle coordinates application shutdown requests without depending
// on the packages that perform application cleanup.
package lifecycle

import "sync"

// Coordinator records the first shutdown request and wakes all waiters.
type Coordinator struct {
	once       sync.Once
	done       chan struct{}
	exitStatus int
}

// NewCoordinator creates a shutdown coordinator.
func NewCoordinator() *Coordinator {
	return &Coordinator{done: make(chan struct{})}
}

// Request records an exit status. Only the first request takes effect.
func (c *Coordinator) Request(exitStatus int) {
	c.once.Do(func() {
		c.exitStatus = exitStatus
		close(c.done)
	})
}

// Done is closed when shutdown has been requested.
func (c *Coordinator) Done() <-chan struct{} {
	return c.done
}

// Wait blocks until shutdown is requested and returns the requested exit status.
func (c *Coordinator) Wait() int {
	<-c.done
	return c.exitStatus
}

var application = NewCoordinator()

// Request asks the application to shut down with exitStatus.
func Request(exitStatus int) {
	application.Request(exitStatus)
}

// Done is closed when application shutdown has been requested.
func Done() <-chan struct{} {
	return application.Done()
}

// Wait blocks until application shutdown is requested.
func Wait() int {
	return application.Wait()
}
