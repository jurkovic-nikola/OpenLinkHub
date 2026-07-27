package systray

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestBuildDashboardURL(t *testing.T) {
	tests := []struct {
		name       string
		listenPort int
		want       string
	}{
		{
			name:       "default port",
			listenPort: 27003,
			want:       "http://127.0.0.1:27003",
		},
		{
			name:       "configured port",
			listenPort: 8080,
			want:       "http://127.0.0.1:8080",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildDashboardURL(test.listenPort)
			if err != nil {
				t.Fatalf("buildDashboardURL() returned error: %v", err)
			}
			if got != test.want {
				t.Fatalf("buildDashboardURL() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDashboardActivationLaunchesConfiguredURL(t *testing.T) {
	var launchedURL string
	err := activateDashboard("clicked", 28080, func(dashboardURL string) {
		launchedURL = dashboardURL
	})
	if err != nil {
		t.Fatalf("activateDashboard() returned error: %v", err)
	}
	if launchedURL != "http://127.0.0.1:28080" {
		t.Fatalf("dashboard launcher received %q, want %q", launchedURL, "http://127.0.0.1:28080")
	}
}

func TestOpenBrowserUsesXDGOpenWithoutShell(t *testing.T) {
	process := &fakeCommandProcess{}
	var commandName string
	var commandArgs []string

	err := openBrowserWithCommand("http://127.0.0.1:27003", func(name string, args ...string) commandProcess {
		commandName = name
		commandArgs = append([]string(nil), args...)
		return process
	})
	if err != nil {
		t.Fatalf("openBrowserWithCommand() returned error: %v", err)
	}
	if commandName != "xdg-open" {
		t.Fatalf("command name = %q, want %q", commandName, "xdg-open")
	}
	if want := []string{"http://127.0.0.1:27003"}; !reflect.DeepEqual(commandArgs, want) {
		t.Fatalf("command args = %#v, want %#v", commandArgs, want)
	}
	if !process.started || !process.waited {
		t.Fatalf("process lifecycle = started:%t waited:%t, want both true", process.started, process.waited)
	}
}

func TestOpenBrowserSurfacesStartError(t *testing.T) {
	startErr := errors.New("start failed")
	process := &fakeCommandProcess{startErr: startErr}

	err := openBrowserWithCommand("http://127.0.0.1:27003", func(string, ...string) commandProcess {
		return process
	})
	if !errors.Is(err, startErr) {
		t.Fatalf("openBrowserWithCommand() error = %v, want wrapped %v", err, startErr)
	}
	var openErr *dashboardOpenError
	if !errors.As(err, &openErr) || openErr.operation != "start" {
		t.Fatalf("openBrowserWithCommand() error = %#v, want start dashboardOpenError", err)
	}
	if process.waited {
		t.Fatal("Wait() was called after Start() failed")
	}
}

func TestOpenBrowserSurfacesNonZeroExit(t *testing.T) {
	waitErr := errors.New("exit status 1")
	process := &fakeCommandProcess{waitErr: waitErr}

	err := openBrowserWithCommand("http://127.0.0.1:27003", func(string, ...string) commandProcess {
		return process
	})
	if !errors.Is(err, waitErr) {
		t.Fatalf("openBrowserWithCommand() error = %v, want wrapped %v", err, waitErr)
	}
	var openErr *dashboardOpenError
	if !errors.As(err, &openErr) || openErr.operation != "wait for" {
		t.Fatalf("openBrowserWithCommand() error = %#v, want wait dashboardOpenError", err)
	}
}

func TestLaunchDashboardAsyncReportsOpenerError(t *testing.T) {
	openerErr := errors.New("opener failed")
	reported := make(chan error, 1)

	launchDashboardAsync(
		"http://127.0.0.1:27003",
		func(string) error { return openerErr },
		func(_ string, err error) { reported <- err },
	)

	select {
	case err := <-reported:
		if !errors.Is(err, openerErr) {
			t.Fatalf("reported error = %v, want %v", err, openerErr)
		}
	case <-time.After(time.Second):
		t.Fatal("asynchronous opener error was not reported")
	}
}

type fakeCommandProcess struct {
	startErr error
	waitErr  error
	started  bool
	waited   bool
}

func (p *fakeCommandProcess) Start() error {
	p.started = true
	return p.startErr
}

func (p *fakeCommandProcess) Wait() error {
	p.waited = true
	return p.waitErr
}
