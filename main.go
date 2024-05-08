package main

import (
	"OpenICUELinkHub/src/controller"
	"os"
	"os/signal"
	"syscall"
)

// WaitForExit listens for a program termination and switches the device back to hardware mode
func WaitForExit() {
	terminateSignals := make(chan os.Signal, 1)
	signal.Notify(terminateSignals, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	for {
		select {
		case <-terminateSignals:
			controller.Stop() // Back to hardware mode
			os.Exit(0)
		}
	}
}

// main entry point
func main() {
	go WaitForExit()
	controller.Start()
}
