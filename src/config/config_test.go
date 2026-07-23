package config

import "testing"

func TestSetSystemServiceExplicitMode(t *testing.T) {
	originalSystemService := systemService
	t.Cleanup(func() {
		systemService = originalSystemService
	})

	tests := []struct {
		name       string
		mode       string
		wantSystem bool
	}{
		{name: "system", mode: "system", wantSystem: true},
		{name: "user", mode: "user", wantSystem: false},
		{name: "desktop alias", mode: "desktop", wantSystem: false},
		{name: "normalized system", mode: " SYSTEM ", wantSystem: true},
		{name: "normalized user", mode: " USER ", wantSystem: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LUMENFORGE_SERVICE_MODE", test.mode)
			systemService = !test.wantSystem

			setSystemService()

			if systemService != test.wantSystem {
				t.Fatalf("setSystemService() with LUMENFORGE_SERVICE_MODE=%q set systemService to %t; want %t", test.mode, systemService, test.wantSystem)
			}
		})
	}
}

func TestSetSystemServiceUnrecognizedModeUsesFallback(t *testing.T) {
	originalSystemService := systemService
	t.Cleanup(func() {
		systemService = originalSystemService
	})

	t.Setenv("LUMENFORGE_SERVICE_MODE", "")
	setSystemService()
	wantSystem := systemService

	t.Setenv("LUMENFORGE_SERVICE_MODE", "unsupported")
	systemService = !wantSystem
	setSystemService()

	if systemService != wantSystem {
		t.Fatalf("unrecognized LUMENFORGE_SERVICE_MODE did not use fallback: got %t; want %t", systemService, wantSystem)
	}
}
