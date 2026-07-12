// Package cleanup coordinates once-only component cleanup without depending on
// the components themselves.
package cleanup

import "sync"

type component struct {
	mutex     sync.Mutex
	stop      func()
	requested bool
	once      sync.Once
}

func (c *component) register(stop func()) {
	c.mutex.Lock()
	c.stop = stop
	requested := c.requested
	c.mutex.Unlock()

	if requested {
		c.run()
	}
}

func (c *component) run() {
	c.mutex.Lock()
	c.requested = true
	stop := c.stop
	c.mutex.Unlock()

	if stop != nil {
		c.once.Do(stop)
	}
}

// Coordinator owns the once-only cleanup routes for suspend-sensitive components.
type Coordinator struct {
	devices component
	input   component
}

// RegisterDevices registers device cleanup after successful initialization.
func (c *Coordinator) RegisterDevices(stop func()) {
	c.devices.register(stop)
}

// RegisterInput registers virtual-input cleanup after successful initialization.
func (c *Coordinator) RegisterInput(stop func()) {
	c.input.register(stop)
}

// StopDevices executes registered device cleanup at most once.
func (c *Coordinator) StopDevices() {
	c.devices.run()
}

// StopInput executes registered virtual-input cleanup at most once.
func (c *Coordinator) StopInput() {
	c.input.run()
}

var application Coordinator

// RegisterDevices registers application device cleanup.
func RegisterDevices(stop func()) {
	application.RegisterDevices(stop)
}

// RegisterInput registers application virtual-input cleanup.
func RegisterInput(stop func()) {
	application.RegisterInput(stop)
}

// StopDevices executes application device cleanup at most once.
func StopDevices() {
	application.StopDevices()
}

// StopInput executes application virtual-input cleanup at most once.
func StopInput() {
	application.StopInput()
}
