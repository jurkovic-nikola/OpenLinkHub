package server

import (
	"LumenForge/src/inputmanager"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

func protectedRecorder(
	protection *localAPIProtection,
	handler http.Handler,
	method string,
	target string,
	host string,
	origin string,
	proof string,
	contentType string,
	body *bytes.Reader,
) *httptest.ResponseRecorder {
	var request *http.Request
	if body == nil {
		request = httptest.NewRequest(method, target, nil)
	} else {
		request = httptest.NewRequest(method, target, body)
	}
	request.Host = host
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	if proof != "" {
		request.Header.Set(requestProofHeader, proof)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	recorder := httptest.NewRecorder()
	protection.wrap(handler).ServeHTTP(recorder, request)
	return recorder
}

func addLocalRequestProtection(t *testing.T, router http.Handler, request *http.Request) {
	t.Helper()
	request.Host = "127.0.0.1"
	if !isMutationMethod(request.Method) {
		return
	}

	tokenRequest := httptest.NewRequest(http.MethodGet, "/api/security/token", nil)
	tokenRequest.Host = "127.0.0.1"
	tokenRecorder := httptest.NewRecorder()
	router.ServeHTTP(tokenRecorder, tokenRequest)
	if tokenRecorder.Code != http.StatusOK {
		t.Fatalf("request proof endpoint returned %d: %s", tokenRecorder.Code, tokenRecorder.Body.String())
	}
	var tokenResponse struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(tokenRecorder.Body.Bytes(), &tokenResponse); err != nil {
		t.Fatalf("decode request proof: %v", err)
	}
	request.Header.Set(requestProofHeader, tokenResponse.Token)
	switch {
	case request.URL.Path == "/api/openrgbimport/discover",
		request.URL.Path == "/api/openrgbimport/refresh",
		strings.HasPrefix(request.URL.Path, "/api/media/"):
	case request.URL.Path == "/api/temperatures/update":
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	default:
		request.Header.Set("Content-Type", "application/json")
	}
}

func TestLocalHostValidation(t *testing.T) {
	protection := newLocalAPIProtection(28080)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	for _, host := range []string{"127.0.0.1", "127.0.0.1:28080", "localhost", "localhost:28080", "LOCALHOST:28080"} {
		t.Run("accept "+host, func(t *testing.T) {
			recorder := protectedRecorder(protection, handler, http.MethodGet, "/api/read", host, "", "", "", nil)
			if recorder.Code != http.StatusNoContent {
				t.Fatalf("Host %q status = %d, want %d", host, recorder.Code, http.StatusNoContent)
			}
		})
	}

	for _, host := range []string{
		"", "0.0.0.0", "192.168.1.10", "100.64.0.10", "example.com",
		"localhost.example.com", "127.0.0.2", "localhost:27003", "localhost:",
		"localhost:not-a-port", "[127.0.0.1]:28080", " localhost", "localhost/path",
	} {
		t.Run("reject "+host, func(t *testing.T) {
			recorder := protectedRecorder(protection, handler, http.MethodGet, "/api/read", host, "", "", "", nil)
			if recorder.Code != http.StatusMisdirectedRequest {
				t.Fatalf("Host %q status = %d, want %d", host, recorder.Code, http.StatusMisdirectedRequest)
			}
		})
	}
}

func TestLocalMutationProtection(t *testing.T) {
	protection := newLocalAPIProtection(28080)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	jsonBody := func() *bytes.Reader { return bytes.NewReader([]byte(`{"enabled":true}`)) }

	tests := []struct {
		name        string
		method      string
		origin      string
		proof       string
		contentType string
		want        int
	}{
		{name: "same-origin 127", method: http.MethodPost, origin: "http://127.0.0.1:28080", proof: protection.proof, contentType: "application/json", want: http.StatusNoContent},
		{name: "same-origin localhost", method: http.MethodPut, origin: "http://localhost:28080", proof: protection.proof, contentType: "application/json; charset=utf-8", want: http.StatusNoContent},
		{name: "CLI without Origin", method: http.MethodDelete, proof: protection.proof, contentType: "application/json", want: http.StatusNoContent},
		{name: "cross-origin", method: http.MethodPost, origin: "https://attacker.example", proof: protection.proof, contentType: "application/json", want: http.StatusForbidden},
		{name: "local but different origin", method: http.MethodPost, origin: "http://localhost:28080", proof: protection.proof, contentType: "application/json", want: http.StatusForbidden},
		{name: "missing proof", method: http.MethodPost, origin: "http://127.0.0.1:28080", contentType: "application/json", want: http.StatusForbidden},
		{name: "invalid proof", method: http.MethodPost, origin: "http://127.0.0.1:28080", proof: "wrong", contentType: "application/json", want: http.StatusForbidden},
		{name: "simple form attack", method: http.MethodPost, origin: "https://attacker.example", contentType: "application/x-www-form-urlencoded", want: http.StatusForbidden},
		{name: "wrong content type", method: http.MethodPatch, origin: "http://127.0.0.1:28080", proof: protection.proof, contentType: "text/plain", want: http.StatusUnsupportedMediaType},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			host := "127.0.0.1:28080"
			if test.name == "same-origin localhost" {
				host = "localhost:28080"
			}
			recorder := protectedRecorder(protection, handler, test.method, "/api/future-mutation", host, test.origin, test.proof, test.contentType, jsonBody())
			if recorder.Code != test.want {
				t.Fatalf("status = %d, want %d: %s", recorder.Code, test.want, recorder.Body.String())
			}
		})
	}
}

func TestLocalMutationSpecialContentTypes(t *testing.T) {
	protection := newLocalAPIProtection(28080)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	var multipartBody bytes.Buffer
	writer := multipart.NewWriter(&multipartBody)
	part, err := writer.CreateFormFile("animationFile", "test.gif")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("GIF89a")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	multipartRequest := httptest.NewRequest(http.MethodPost, "/api/lcd/upload", bytes.NewReader(multipartBody.Bytes()))
	multipartRequest.Host = "127.0.0.1:28080"
	multipartRequest.Header.Set(requestProofHeader, protection.proof)
	multipartRequest.Header.Set("Content-Type", writer.FormDataContentType())
	multipartRecorder := httptest.NewRecorder()
	protection.wrap(handler).ServeHTTP(multipartRecorder, multipartRequest)
	if multipartRecorder.Code != http.StatusNoContent {
		t.Fatalf("multipart status = %d: %s", multipartRecorder.Code, multipartRecorder.Body.String())
	}

	formRecorder := protectedRecorder(
		protection,
		handler,
		http.MethodPut,
		"/api/temperatures/update",
		"localhost:28080",
		"",
		protection.proof,
		"application/x-www-form-urlencoded; charset=UTF-8",
		bytes.NewReader([]byte("profile=test&data=%7B%7D")),
	)
	if formRecorder.Code != http.StatusNoContent {
		t.Fatalf("form status = %d: %s", formRecorder.Code, formRecorder.Body.String())
	}

	emptyRecorder := protectedRecorder(
		protection,
		handler,
		http.MethodPost,
		"/api/openrgbimport/discover",
		"127.0.0.1:28080",
		"",
		protection.proof,
		"",
		nil,
	)
	if emptyRecorder.Code != http.StatusNoContent {
		t.Fatalf("empty-body status = %d: %s", emptyRecorder.Code, emptyRecorder.Body.String())
	}
}

func TestCORSPreflightIsRejectedWithoutCORSHeaders(t *testing.T) {
	protection := newLocalAPIProtection(28080)
	request := httptest.NewRequest(http.MethodOptions, "/api/color", nil)
	request.Host = "127.0.0.1:28080"
	request.Header.Set("Origin", "https://attacker.example")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", requestProofHeader)
	recorder := httptest.NewRecorder()
	protection.wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("preflight reached application handler")
	})).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("unexpected CORS header: %q", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestProductionMutationRoutesCannotBypassProtection(t *testing.T) {
	source, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	routePattern := regexp.MustCompile(`handleFunc\(r, "([^"]+)", http\.Method(Post|Put|Patch|Delete),`)
	matches := routePattern.FindAllStringSubmatch(string(source), -1)
	if len(matches) < 100 {
		t.Fatalf("found only %d production mutation routes", len(matches))
	}

	router := setRoutes()
	for _, match := range matches {
		method := strings.ToUpper(match[2])
		path := match[1]
		t.Run(method+" "+path, func(t *testing.T) {
			request := httptest.NewRequest(method, path, nil)
			request.Host = "127.0.0.1"
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
			}
		})
	}
}

func TestExistingReadOnlyRouteRemainsAvailable(t *testing.T) {
	router := setRoutes()
	request := httptest.NewRequest(http.MethodGet, "/api/openrgb/status", nil)
	request.Host = "localhost"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestMediaControlUsesProtectedPost(t *testing.T) {
	previousMediaInputControl := mediaInputControl
	var controlType uint16
	var hold bool
	mediaInputControl = func(gotControlType uint16, gotHold bool) {
		controlType = gotControlType
		hold = gotHold
	}
	t.Cleanup(func() {
		mediaInputControl = previousMediaInputControl
	})

	router := setRoutes()
	getRequest := httptest.NewRequest(http.MethodGet, "/api/media/previous", nil)
	addLocalRequestProtection(t, router, getRequest)
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d, want %d", getRecorder.Code, http.StatusMethodNotAllowed)
	}

	for action, expectedControlType := range map[string]uint16{
		"previous":   inputmanager.MediaPrev,
		"stop":       inputmanager.MediaStop,
		"play":       inputmanager.MediaPlayPause,
		"next":       inputmanager.MediaNext,
		"volumeDown": inputmanager.VolumeDown,
		"volumeUp":   inputmanager.VolumeUp,
		"mute":       inputmanager.VolumeMute,
	} {
		t.Run(action, func(t *testing.T) {
			controlType = 0
			hold = true
			postRequest := httptest.NewRequest(http.MethodPost, "/api/media/"+action, nil)
			addLocalRequestProtection(t, router, postRequest)
			postRecorder := httptest.NewRecorder()
			router.ServeHTTP(postRecorder, postRequest)
			if postRecorder.Code != http.StatusOK {
				t.Fatalf("POST status = %d: %s", postRecorder.Code, postRecorder.Body.String())
			}
			if controlType != expectedControlType || hold {
				t.Fatalf("media control = (%d, %t), want (%d, false)", controlType, hold, expectedControlType)
			}
		})
	}
}
