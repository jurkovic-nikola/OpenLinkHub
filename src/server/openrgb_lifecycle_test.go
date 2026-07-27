package server

import (
	"LumenForge/src/devices/openrgbimport"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func decodeLifecycleResponse(t *testing.T, recorder *httptest.ResponseRecorder) *Response {
	t.Helper()
	var response Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return &response
}

func TestOpenRGBImportLifecycleRoutesAndResponseCompatibility(t *testing.T) {
	previousDiscover := discoverOpenRGBImports
	previousImport := importOpenRGBImports
	previousRemove := removeOpenRGBImports
	previousRefresh := refreshOpenRGBImports
	t.Cleanup(func() {
		discoverOpenRGBImports = previousDiscover
		importOpenRGBImports = previousImport
		removeOpenRGBImports = previousRemove
		refreshOpenRGBImports = previousRefresh
	})

	discoverOpenRGBImports = func(context.Context) (openrgbimport.DiscoveryPreview, error) {
		return openrgbimport.DiscoveryPreview{
			DiscoveryState: "available",
			Configured:     []openrgbimport.ConfiguredImportSummary{},
			Controllers:    []openrgbimport.ControllerPreview{},
		}, nil
	}
	importOpenRGBImports = func(_ context.Context, keys []string) (openrgbimport.ImportResult, error) {
		return openrgbimport.ImportResult{
			ConfiguredSerials: []string{"openrgb-hash-test"},
			Configured: []openrgbimport.ConfiguredImportSummary{
				{Serial: "openrgb-hash-test", Product: strings.Join(keys, ",")},
			},
		}, nil
	}
	removeOpenRGBImports = func(_ context.Context, serials []string) (openrgbimport.RemoveResult, error) {
		return openrgbimport.RemoveResult{RemovedSerials: append([]string(nil), serials...)}, nil
	}
	refreshOpenRGBImports = func(context.Context) error { return nil }

	router := setRoutes()
	tests := []struct {
		path string
		body string
	}{
		{path: "/api/openrgbimport/discover"},
		{path: "/api/openrgbimport/import", body: `{"keys":["orgb-v1-test"]}`},
		{path: "/api/openrgbimport/remove", body: `{"serials":["openrgb-hash-test"]}`},
		{path: "/api/openrgbimport/refresh"},
	}
	for _, test := range tests {
		request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
		addLocalRequestProtection(t, router, request)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s HTTP status = %d", test.path, recorder.Code)
		}
		response := decodeLifecycleResponse(t, recorder)
		if response.Code != http.StatusOK || response.Status != 1 || response.Message == "" {
			t.Fatalf("%s response = %#v", test.path, response)
		}
		if strings.Contains(recorder.Body.String(), `"instance"`) || strings.Contains(recorder.Body.String(), `"controllerId"`) {
			t.Fatalf("%s serialized live/internal data: %s", test.path, recorder.Body.String())
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/api/openrgbimport/discover", nil)
	addLocalRequestProtection(t, router, request)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET method status = %d, want 405", recorder.Code)
	}
}

func TestOpenRGBImportLifecycleHandlersRejectInvalidRequests(t *testing.T) {
	router := setRoutes()
	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "malformed import", path: "/api/openrgbimport/import", body: `{"keys":`},
		{name: "unknown import field", path: "/api/openrgbimport/import", body: `{"keys":["x"],"instance":{}}`},
		{name: "empty import", path: "/api/openrgbimport/import", body: `{"keys":[]}`},
		{name: "malformed removal", path: "/api/openrgbimport/remove", body: `not-json`},
		{name: "empty removal", path: "/api/openrgbimport/remove", body: `{"serials":[]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
			addLocalRequestProtection(t, router, request)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			response := decodeLifecycleResponse(t, recorder)
			if recorder.Code != http.StatusOK || response.Code != http.StatusOK || response.Status != 0 || response.Message == "" {
				t.Fatalf("response = %#v, HTTP %d", response, recorder.Code)
			}
		})
	}

	keys := make([]string, openRGBImportBatchLimit+1)
	serials := make([]string, openRGBImportBatchLimit+1)
	for index := range keys {
		keys[index] = "key"
		serials[index] = "serial"
	}
	for path, body := range map[string]any{
		"/api/openrgbimport/import": map[string]any{"keys": keys},
		"/api/openrgbimport/remove": map[string]any{"serials": serials},
	} {
		data, _ := json.Marshal(body)
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(data)))
		addLocalRequestProtection(t, router, request)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		response := decodeLifecycleResponse(t, recorder)
		if response.Status != 0 || !strings.Contains(strings.ToLower(response.Message), "maximum") {
			t.Fatalf("%s excessive batch response = %#v", path, response)
		}
	}
}

func TestOpenRGBImportDiscoveryFailureKeepsUsefulData(t *testing.T) {
	previousDiscover := discoverOpenRGBImports
	t.Cleanup(func() { discoverOpenRGBImports = previousDiscover })
	discoverOpenRGBImports = func(context.Context) (openrgbimport.DiscoveryPreview, error) {
		data := openrgbimport.DiscoveryPreview{
			DiscoveryState: "offline",
			Error:          "SDK unavailable",
			Configured: []openrgbimport.ConfiguredImportSummary{
				{Serial: "openrgb-mobo-1", Product: "Configured"},
			},
			Controllers: []openrgbimport.ControllerPreview{},
		}
		return data, errors.New("SDK unavailable")
	}

	router := setRoutes()
	request := httptest.NewRequest(http.MethodPost, "/api/openrgbimport/discover", nil)
	addLocalRequestProtection(t, router, request)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	response := decodeLifecycleResponse(t, recorder)
	if recorder.Code != http.StatusOK || response.Status != 0 || response.Data == nil {
		t.Fatalf("failure response = %#v", response)
	}
	if !strings.Contains(recorder.Body.String(), "openrgb-mobo-1") {
		t.Fatalf("configured summaries missing from failure response: %s", recorder.Body.String())
	}
}
