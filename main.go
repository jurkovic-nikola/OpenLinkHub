package main

import (
	"LumenForge/src/controller"
	"LumenForge/src/lifecycle"
	"os"
	"os/signal"
	"syscall"
)

// run starts LumenForge and returns only after coordinated cleanup completes.
func run() int {
	terminateSignals := make(chan os.Signal, 1)
	signal.Notify(terminateSignals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(terminateSignals)

	go func() {
		select {
		case <-terminateSignals:
			lifecycle.Request(0)
		case <-lifecycle.Done():
		}
	}()

	controller.Start()
	exitStatus := lifecycle.Wait()
	controller.Stop()
	return exitStatus
}

// main entry point
func main() {
	os.Exit(run())
}
