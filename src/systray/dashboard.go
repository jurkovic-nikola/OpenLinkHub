package systray

import (
	"LumenForge/src/logger"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
)

const dashboardOpener = "xdg-open"

type commandProcess interface {
	Start() error
	Wait() error
}

type commandFactory func(name string, args ...string) commandProcess

type dashboardOpenError struct {
	operation string
	err       error
}

func (e *dashboardOpenError) Error() string {
	return fmt.Sprintf("%s dashboard opener: %v", e.operation, e.err)
}

func (e *dashboardOpenError) Unwrap() error {
	return e.err
}

func buildDashboardURL(listenAddress string, listenPort int) (string, error) {
	if listenPort < 1 || listenPort > 65535 {
		return "", fmt.Errorf("invalid listen port %d", listenPort)
	}

	host := strings.TrimSpace(listenAddress)
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	}

	ip := net.ParseIP(host)
	switch {
	case host == "":
		host = "127.0.0.1"
	case ip != nil && ip.IsUnspecified() && strings.Contains(host, ":"):
		host = "::1"
	case ip != nil && ip.IsUnspecified():
		host = "127.0.0.1"
	}

	return (&url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, strconv.Itoa(listenPort)),
	}).String(), nil
}

func activateDashboard(eventID, listenAddress string, listenPort int, launch func(string)) error {
	logger.Log(logger.Fields{
		"eventId": eventID,
		"menuId":  101,
	}).Debug("Open Dashboard tray action received")

	dashboardURL, err := buildDashboardURL(listenAddress, listenPort)
	if err != nil {
		return err
	}

	logger.Log(logger.Fields{"url": dashboardURL}).Debug("Opening dashboard URL")
	launch(dashboardURL)
	return nil
}

func launchDashboard(dashboardURL string) {
	launchDashboardAsync(dashboardURL, openBrowser, logDashboardOpenError)
}

func launchDashboardAsync(dashboardURL string, opener func(string) error, reportError func(string, error)) {
	go func() {
		if err := opener(dashboardURL); err != nil {
			reportError(dashboardURL, err)
		}
	}()
}

func openBrowser(dashboardURL string) error {
	return openBrowserWithCommand(dashboardURL, func(name string, args ...string) commandProcess {
		return exec.Command(name, args...)
	})
}

func openBrowserWithCommand(dashboardURL string, newCommand commandFactory) error {
	cmd := newCommand(dashboardOpener, dashboardURL)
	if err := cmd.Start(); err != nil {
		return &dashboardOpenError{operation: "start", err: err}
	}
	if err := cmd.Wait(); err != nil {
		return &dashboardOpenError{operation: "wait for", err: err}
	}
	return nil
}

func logDashboardOpenError(dashboardURL string, err error) {
	fields := logger.Fields{
		"command": dashboardOpener,
		"error":   err,
		"url":     dashboardURL,
	}
	if openError, ok := err.(*dashboardOpenError); ok && openError.operation == "start" {
		logger.Log(fields).Error("Failed to start dashboard opener")
		return
	}
	logger.Log(fields).Error("Dashboard opener exited unsuccessfully")
}
