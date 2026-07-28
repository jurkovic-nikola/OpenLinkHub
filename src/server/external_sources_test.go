package server

import (
	"LumenForge/src/config"
	"LumenForge/src/logger"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestGetExternalSourcesReturnsOnlyIDAndName(t *testing.T) {
	root := t.TempDir()
	paths, err := config.ResolvePaths(config.PathOptions{
		Mode:             config.ServiceModeDevelopment,
		WorkingDirectory: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(config.UsePathsForTest(paths))

	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	registryData, err := json.Marshal(map[string]any{
		"sources": []map[string]any{{
			"id":         "safe-id",
			"name":       "Safe Name",
			"executable": executable,
			"args":       []string{"secret-fixed-argument"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(paths.ExternalSourcesFile, registryData, 0o600); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	getExternalSources(recorder, httptest.NewRequest("GET", "/api/external-sources", nil))

	var response struct {
		Status int              `json:"status"`
		Data   []map[string]any `json:"data"`
	}
	if err = json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != 1 || len(response.Data) != 1 {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	entry := response.Data[0]
	if len(entry) != 2 || entry["id"] != "safe-id" || entry["name"] != "Safe Name" {
		t.Fatalf("unsafe or incorrect API entry = %#v", entry)
	}
	for _, forbidden := range []string{executable, "secret-fixed-argument", "executable", "args"} {
		if strings.Contains(recorder.Body.String(), forbidden) {
			t.Fatalf("API response exposed %q: %s", forbidden, recorder.Body.String())
		}
	}
}

func TestGetExternalSourcesMissingRegistryIsEmpty(t *testing.T) {
	paths, err := config.ResolvePaths(config.PathOptions{
		Mode:             config.ServiceModeDevelopment,
		WorkingDirectory: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(config.UsePathsForTest(paths))

	recorder := httptest.NewRecorder()
	getExternalSources(recorder, httptest.NewRequest("GET", "/api/external-sources", nil))
	if !strings.Contains(recorder.Body.String(), `"status":1`) ||
		!strings.Contains(recorder.Body.String(), `"data":[]`) ||
		!strings.Contains(recorder.Body.String(), "No external sources are configured") {
		t.Fatalf("missing registry response = %s", recorder.Body.String())
	}
}

func TestGetExternalSourcesMalformedRegistryReportsUnavailable(t *testing.T) {
	paths, err := config.ResolvePaths(config.PathOptions{
		Mode:             config.ServiceModeDevelopment,
		WorkingDirectory: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(config.UsePathsForTest(paths))
	logger.Init()
	if err = os.WriteFile(paths.ExternalSourcesFile, []byte(`{"sources":[`), 0o600); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	getExternalSources(recorder, httptest.NewRequest("GET", "/api/external-sources", nil))
	body := recorder.Body.String()
	if !strings.Contains(body, `"status":0`) ||
		!strings.Contains(body, `"data":[]`) ||
		!strings.Contains(body, "registry is unavailable") {
		t.Fatalf("malformed registry response = %s", body)
	}
	if strings.Contains(body, paths.ExternalSourcesFile) {
		t.Fatalf("malformed registry response exposed filesystem details: %s", body)
	}
}
