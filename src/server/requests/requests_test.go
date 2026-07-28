package requests

import (
	"LumenForge/src/config"
	"LumenForge/src/dashboard"
	"LumenForge/src/externalsources"
	"LumenForge/src/language"
	"LumenForge/src/logger"
	"LumenForge/src/temperatures"
	"bytes"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLinkAdapterResult(t *testing.T) {
	tests := []struct {
		name        string
		result      uint64
		wantMessage string
		wantStatus  int
	}{
		{name: "success", result: 1, wantMessage: "txtLinkAdapterUpdated", wantStatus: 1},
		{name: "missing Link adapter", result: 2, wantMessage: "txtUnableToChangeRgbStripNoLink", wantStatus: 0},
		{name: "generic failure", result: 0, wantMessage: "txtUnableToChangeRgbStrip", wantStatus: 0},
		{name: "unknown result", result: 99, wantMessage: "txtUnableToChangeRgbStrip", wantStatus: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message, status := linkAdapterResult(test.result)
			if message != test.wantMessage || status != test.wantStatus {
				t.Fatalf("linkAdapterResult(%d) = (%q, %d), want (%q, %d)",
					test.result, message, status, test.wantMessage, test.wantStatus)
			}
		})
	}
}

func TestProcessNewTemperatureProfileExternalSourceValidation(t *testing.T) {
	paths, root := useRequestLocalizationTestState(t)

	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	registry := map[string]any{
		"sources": []map[string]any{{
			"id":         "registered",
			"name":       "Registered Source",
			"executable": executable,
			"args":       []string{"fixed-only"},
		}},
	}
	registryData, err := json.Marshal(registry)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(paths.ExternalSourcesFile, registryData, 0o600); err != nil {
		t.Fatal(err)
	}
	temperatures.Init()

	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus int
	}{
		{
			name: "valid registered id",
			payload: map[string]any{
				"profile":            "ValidExternal",
				"sensor":             temperatures.SensorTypeExternalExecutable,
				"externalSourceId":   "registered",
				"externalExecutable": filepath.Join(root, "untrusted"),
				"args":               []string{"request-injected"},
			},
			wantStatus: 1,
		},
		{
			name: "missing id",
			payload: map[string]any{
				"profile": "MissingExternal",
				"sensor":  temperatures.SensorTypeExternalExecutable,
			},
			wantStatus: 0,
		},
		{
			name: "unknown id",
			payload: map[string]any{
				"profile":          "UnknownExternal",
				"sensor":           temperatures.SensorTypeExternalExecutable,
				"externalSourceId": "not-registered",
			},
			wantStatus: 0,
		},
		{
			name: "legacy executable path ignored",
			payload: map[string]any{
				"profile":            "LegacyExternal",
				"sensor":             temperatures.SensorTypeExternalExecutable,
				"externalExecutable": filepath.Join(root, "untrusted"),
			},
			wantStatus: 0,
		},
		{
			name: "other sensor unchanged",
			payload: map[string]any{
				"profile":          "OrdinaryCpu",
				"sensor":           temperatures.SensorTypeCPU,
				"externalSourceId": "not-registered",
			},
			wantStatus: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, marshalErr := json.Marshal(test.payload)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			request := httptest.NewRequest("POST", "/api/temperatures/new", bytes.NewReader(body))
			response := ProcessNewTemperatureProfile(request)
			if response.Status != test.wantStatus {
				t.Fatalf("ProcessNewTemperatureProfile() = %#v", response)
			}
		})
	}

	saved, err := os.ReadFile(filepath.Join(paths.MutableTemperaturesRoot, "ValidExternal.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(saved, []byte("request-injected")) ||
		bytes.Contains(saved, []byte("fixed-only")) ||
		bytes.Contains(saved, []byte(filepath.Join(root, "untrusted"))) ||
		bytes.Contains(saved, []byte("externalExecutable")) {
		t.Fatalf("request or registry execution details entered saved profile: %s", saved)
	}

	if err = os.Remove(paths.ExternalSourcesFile); err != nil {
		t.Fatal(err)
	}
	missingRegistryBody := []byte(`{"profile":"NoRegistryExternal","sensor":7,"externalSourceId":"registered"}`)
	missingRegistryResponse := ProcessNewTemperatureProfile(
		httptest.NewRequest("POST", "/api/temperatures/new", bytes.NewReader(missingRegistryBody)),
	)
	if missingRegistryResponse.Status != 0 ||
		missingRegistryResponse.Message != language.GetValue("txtNoExternalSourcesConfigured") {
		t.Fatalf("missing registry response = %#v", missingRegistryResponse)
	}
}

func TestExternalSourceSelectionMessageUsesLanguageSystem(t *testing.T) {
	useRequestLocalizationTestState(t)

	dash := dashboard.GetDashboard()
	dash.LanguageCode = "de_DE"
	if dashboard.SaveDashboardSettings(dash, true) != 1 {
		t.Fatal("unable to select German dashboard language")
	}

	english := language.GetLanguage("en_US")
	german := language.GetLanguage("de_DE")
	if english == nil || german == nil {
		t.Fatal("expected English and German language catalogs")
	}

	tests := []struct {
		name string
		err  error
		key  string
	}{
		{
			name: "missing registry",
			err:  externalsources.ErrRegistryMissing,
			key:  "txtNoExternalSourcesConfigured",
		},
		{
			name: "invalid registry",
			err:  externalsources.ErrRegistryInvalid,
			key:  "txtExternalSourceRegistryUnavailable",
		},
		{
			name: "unknown source",
			err:  externalsources.ErrSourceUnknown,
			key:  "txtSelectRegisteredExternalSource",
		},
		{
			name: "default validation failure",
			err:  errors.New("unexpected validation failure"),
			key:  "txtUnableToValidateExternalSource",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := externalSourceSelectionMessage(test.err)
			if got != language.GetValue(test.key) || got != german.Values[test.key] {
				t.Fatalf("externalSourceSelectionMessage(%v) = %q, want localized %q",
					test.err, got, german.Values[test.key])
			}
			if got == english.Values[test.key] {
				t.Fatalf("externalSourceSelectionMessage(%v) returned hardcoded English %q",
					test.err, got)
			}
		})
	}
}

func useRequestLocalizationTestState(t *testing.T) (config.Paths, string) {
	t.Helper()

	packageDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repositoryRoot := filepath.Clean(filepath.Join(packageDirectory, "..", "..", ".."))
	root := t.TempDir()
	paths, err := config.ResolvePaths(config.PathOptions{
		Mode:             config.ServiceModeDevelopment,
		ApplicationRoot:  repositoryRoot,
		WorkingDirectory: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = config.EnsureRuntimeDirectories(paths); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(config.UsePathsForTest(paths))

	logger.Init()
	dashboard.Init()
	language.Init()
	return paths, root
}
