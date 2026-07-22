package server

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const uiDir = "./ui"

// spaHandler serves the SvelteKit static build with SPA fallback to index.html.
type spaHandler struct {
	root http.FileSystem
	fs   http.Handler
}

func newSPAHandler(dir string) http.Handler {
	return &spaHandler{
		root: http.Dir(dir),
		fs:   http.FileServer(http.Dir(dir)),
	}
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	urlPath := r.URL.Path
	if strings.HasPrefix(urlPath, "/api/") || urlPath == "/api" {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(urlPath, "/static/") {
		http.NotFound(w, r)
		return
	}

	clean := path.Clean(urlPath)
	if clean == "/" || clean == "." {
		h.serveIndex(w, r)
		return
	}

	full := filepath.Join(uiDir, filepath.FromSlash(clean))
	if info, err := os.Stat(full); err == nil && !info.IsDir() {
		h.fs.ServeHTTP(w, r)
		return
	}

	// Try as directory with index (SvelteKit assets under _app/)
	if info, err := os.Stat(full); err == nil && info.IsDir() {
		h.fs.ServeHTTP(w, r)
		return
	}

	h.serveIndex(w, r)
}

func (h *spaHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	index := filepath.Join(uiDir, "index.html")
	if _, err := os.Stat(index); err != nil {
		http.Error(w, "WebUI not built. Run: make frontend-build", http.StatusServiceUnavailable)
		return
	}
	http.ServeFile(w, r, index)
}
