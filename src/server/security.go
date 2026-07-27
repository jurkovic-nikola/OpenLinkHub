package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	requestProofCookie = "lumenforge_request_proof"
	requestProofHeader = "X-LumenForge-Request-Proof"
	proofFailureHeader = "X-LumenForge-Request-Proof-Failure"
)

type localAPIProtection struct {
	listenPort int
	proof      string
}

func newLocalAPIProtection(listenPort int) *localAPIProtection {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		panic("unable to generate local API request proof: " + err.Error())
	}
	return &localAPIProtection{
		listenPort: listenPort,
		proof:      base64.RawURLEncoding.EncodeToString(random),
	}
}

func (protection *localAPIProtection) tokenHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Token string `json:"token"`
	}{Token: protection.proof})
}

func (protection *localAPIProtection) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestHost, ok := parseLocalHost(r.Host, protection.listenPort)
		if !ok {
			http.Error(w, "LumenForge only accepts local Host values", http.StatusMisdirectedRequest)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     requestProofCookie,
			Value:    protection.proof,
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
		})

		if r.Method == http.MethodOptions {
			http.Error(w, "CORS preflight requests are not supported", http.StatusForbidden)
			return
		}

		if isMutationMethod(r.Method) {
			if !protection.validMutationOrigin(r, requestHost) {
				http.Error(w, "Cross-origin requests are not allowed", http.StatusForbidden)
				return
			}
			if !protection.validRequestProof(r) {
				w.Header().Set(proofFailureHeader, "invalid")
				http.Error(w, "Missing or invalid LumenForge request proof; reload the dashboard", http.StatusForbidden)
				return
			}
			if !validMutationContentType(r) {
				http.Error(w, "Unsupported request Content-Type", http.StatusUnsupportedMediaType)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func isMutationMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func parseLocalHost(value string, listenPort int) (string, bool) {
	if value == "" || strings.TrimSpace(value) != value || strings.ContainsAny(value, "/\\@?#") {
		return "", false
	}

	host := value
	port := ""
	hasPort := false
	if !strings.EqualFold(value, "localhost") && value != "127.0.0.1" {
		var err error
		host, port, err = net.SplitHostPort(value)
		hasPort = true
		if err != nil || port == "" || strings.HasPrefix(value, "[") {
			return "", false
		}
	}

	if !strings.EqualFold(host, "localhost") && host != "127.0.0.1" {
		return "", false
	}
	host = strings.ToLower(host)
	if !hasPort {
		return host, true
	}
	for _, character := range port {
		if character < '0' || character > '9' {
			return "", false
		}
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return "", false
	}
	if listenPort > 0 && portNumber != listenPort {
		return "", false
	}
	return net.JoinHostPort(host, strconv.Itoa(portNumber)), true
}

func (protection *localAPIProtection) validMutationOrigin(r *http.Request, requestHost string) bool {
	origins := r.Header.Values("Origin")
	if len(origins) == 0 {
		return true
	}
	if len(origins) != 1 {
		return false
	}
	if strings.ContainsAny(origins[0], "?#") {
		return false
	}

	origin, err := url.Parse(origins[0])
	if err != nil ||
		origin.Scheme != "http" ||
		origin.Opaque != "" ||
		origin.User != nil ||
		origin.Path != "" ||
		origin.RawQuery != "" ||
		origin.Fragment != "" {
		return false
	}
	originHost, ok := parseLocalHost(origin.Host, protection.listenPort)
	return ok && originHost == requestHost
}

func (protection *localAPIProtection) validRequestProof(r *http.Request) bool {
	values := r.Header.Values(requestProofHeader)
	if len(values) != 1 || len(values[0]) != len(protection.proof) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(values[0]), []byte(protection.proof)) == 1
}

func validMutationContentType(r *http.Request) bool {
	contentTypes := r.Header.Values("Content-Type")
	expected := "application/json"
	switch {
	case r.URL.Path == "/api/openrgbimport/discover",
		r.URL.Path == "/api/openrgbimport/refresh",
		strings.HasPrefix(r.URL.Path, "/api/media/"):
		return r.ContentLength == 0 && len(contentTypes) == 0
	case r.URL.Path == "/api/restore", r.URL.Path == "/api/lcd/upload":
		expected = "multipart/form-data"
	case r.URL.Path == "/api/temperatures/update":
		expected = "application/x-www-form-urlencoded"
	}

	if len(contentTypes) != 1 {
		return false
	}
	mediaType, parameters, err := mime.ParseMediaType(contentTypes[0])
	if err != nil || mediaType != expected {
		return false
	}
	if expected == "multipart/form-data" && parameters["boundary"] == "" {
		return false
	}
	return true
}
