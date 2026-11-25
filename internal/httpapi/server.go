package httpapi

// HTTP transport for the App core. It owns request parsing, status codes and
// the localhost security wrapper; all business logic lives in internal/app.
// The embedded UI is just another client of this API.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bytecapsule/doscapsule/internal/app"
)

type Options struct {
	Port        int
	OpenBrowser bool
	Logf        func(format string, args ...any)
}

type server struct {
	app      *app.App
	fallback []byte
}

// Serve starts the localhost HTTP server and blocks until it fails. It never
// returns nil.
func Serve(a *app.App, ui fs.FS, fallbackCover []byte, opts Options) error {
	if opts.Port == 0 {
		opts.Port = 8420
	}

	addr := fmt.Sprintf("127.0.0.1:%d", opts.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		// Port busy: fall back to a random free port.
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}
	}
	// The listener's real address (127.0.0.1:<port>, after any fallback) is the
	// only Host we accept. DNS rebinding or a wrong host is refused outright.
	host := ln.Addr().String()
	handler := newHandler(a, ui, fallbackCover, host)

	url := fmt.Sprintf("http://%s", host)
	if opts.Logf != nil {
		opts.Logf("DOS Capsule running at %s", url)
	}
	if opts.OpenBrowser {
		go openBrowser(url, opts.Logf)
	}
	return http.Serve(ln, handler)
}

// newHandler builds the routed, host-checked, CSRF-protected handler. It is
// separate from Serve so tests can exercise the security wrapper without
// binding a real port.
func newHandler(a *app.App, ui fs.FS, fallbackCover []byte, host string) http.Handler {
	s := &server{app: a, fallback: fallbackCover}

	mux := http.NewServeMux()
	// The HTML shell is never cached: after a rebuild+restart the browser must
	// pick up the new hashed asset names instead of a stale page.
	mux.Handle("/", noStoreHTML(http.FileServer(http.FS(ui))))
	mux.HandleFunc("/media", s.handleMedia)
	mux.HandleFunc("/api/scan", s.handleScan)
	mux.HandleFunc("/api/games", s.handleGames)
	mux.HandleFunc("/api/genres", s.handleGenres)
	mux.HandleFunc("/api/remembered_root", s.handleRememberedRoot)
	mux.HandleFunc("/api/launch", s.handleLaunch)
	mux.HandleFunc("/api/stop", s.handleStop)
	mux.HandleFunc("/api/running", s.handleRunning)
	mux.HandleFunc("/api/favorite", s.handleFavorite)
	mux.HandleFunc("/api/conf", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			s.handleConfSave(w, r)
		} else {
			s.handleConfGet(w, r)
		}
	})
	mux.HandleFunc("/api/open", s.handleOpenDir)
	mux.HandleFunc("/api/pick_folder", s.handlePickFolder)
	mux.HandleFunc("/api/clear", s.handleClear)

	// CrossOriginProtection rejects non-safe cross-site browser requests (CSRF):
	// a malicious page can reach 127.0.0.1 but must not trigger launches, writes
	// or other state changes. Safe GETs pass, and curl/CLI (no Origin/Sec-Fetch)
	// stays usable. Localhost binding alone is not security.
	csrf := http.NewCrossOriginProtection().Handler(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != host {
			http.Error(w, "invalid Host", http.StatusForbidden)
			return
		}
		csrf.ServeHTTP(w, r)
	})
}

// ── handlers ────────────────────────────────────────────────────────────────

func (s *server) handleScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Root string `json:"root"`
	}
	if err := decodeBody(r, &req); err != nil || req.Root == "" {
		http.Error(w, "missing root", http.StatusBadRequest)
		return
	}
	n, err := s.app.Scan(req.Root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]int{"count": n})
}

func (s *server) handleGames(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	writeJSON(w, s.app.Games(app.ListOptions{
		Query:         q.Get("query"),
		Genre:         q.Get("genre"),
		Sort:          q.Get("sortBy"),
		FavoritesOnly: q.Get("favoritesOnly") == "1",
	}))
}

func (s *server) handleGenres(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.app.Genres())
}

func (s *server) handleRememberedRoot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.app.RememberedRoot())
}

func (s *server) handleLaunch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := decodeBody(r, &req); err != nil || req.ID == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	p, err := s.app.StartGame("", req.ID)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	s.app.Track(p)
	writeJSON(w, map[string]string{"message": "Launched " + p.Title})
}

func (s *server) handleStop(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := decodeBody(r, &req); err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	if err := s.app.Stop(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *server) handleRunning(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.app.Running())
}

func (s *server) handleFavorite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := decodeBody(r, &req); err != nil || req.ID == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]bool{"favorited": s.app.Favorite(req.ID)})
}

func (s *server) handleConfGet(w http.ResponseWriter, r *http.Request) {
	content, ok, err := s.app.ConfGet(r.URL.Query().Get("id"))
	if err != nil {
		writeAppErr(w, err)
		return
	}
	if !ok {
		writeJSON(w, map[string]any{"content": nil})
		return
	}
	writeJSON(w, map[string]any{"content": content})
}

func (s *server) handleConfSave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}
	if err := decodeBody(r, &req); err != nil || req.ID == "" {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	if err := s.app.ConfSave(req.ID, req.Content); err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *server) handleOpenDir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := decodeBody(r, &req); err != nil || req.ID == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	if err := s.app.OpenDir(req.ID); err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// handlePickFolder opens the OS-native folder chooser and returns the path.
func (s *server) handlePickFolder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
	}
	_ = decodeBody(r, &req)
	if req.Title == "" {
		req.Title = "Scan a games folder"
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`POSIX path of (choose folder with prompt %q)`, req.Title)
		cmd = exec.Command("osascript", "-e", script)
	case "windows":
		ps := fmt.Sprintf(
			`Add-Type -AssemblyName System.Windows.Forms; $f = New-Object System.Windows.Forms.FolderBrowserDialog; $f.Description = %q; if ($f.ShowDialog() -eq 'OK') { Write-Output $f.SelectedPath }`,
			req.Title)
		cmd = exec.Command("powershell", "-NoProfile", "-Command", ps)
	default:
		cmd = exec.Command("zenity", "--file-selection", "--directory", "--title", req.Title)
	}
	out, err := cmd.Output()
	if err != nil {
		// Cancelled (or no chooser available): null.
		writeJSON(w, map[string]any{"path": nil})
		return
	}
	p := strings.TrimSpace(string(out))
	p = strings.TrimSuffix(p, "/") // osascript returns a trailing slash
	writeJSON(w, map[string]string{"path": filepath.Clean(p)})
}

// /media?path=... serves cover/screenshot bytes. Allowed: anything under the
// selected games root, plus the embedded fallback cover's virtual path.
func (s *server) handleMedia(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	if p == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}
	if p == app.FallbackCoverPath {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(s.fallback)
		return
	}
	root := s.app.Root()
	if root == "" {
		http.Error(w, "no games root", http.StatusForbidden)
		return
	}
	abs, _ := filepath.Abs(p)
	rootAbs, _ := filepath.Abs(root)
	if !strings.HasPrefix(abs, rootAbs+string(os.PathSeparator)) {
		http.Error(w, "path outside the games root", http.StatusForbidden)
		return
	}
	// Covers change when the user re-packs a game; a reload must pick them up.
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, abs)
}

func (s *server) handleClear(w http.ResponseWriter, r *http.Request) {
	s.app.Clear()
	writeJSON(w, map[string]bool{"ok": true})
}

// ── helpers ─────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func decodeBody(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// writeAppErr maps the core's sentinel errors onto the API's status codes.
func writeAppErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, app.ErrNoRoot), errors.Is(err, app.ErrGameNotFound):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, app.ErrAlreadyRunning):
		http.Error(w, "game is already running", http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func noStoreHTML(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, ".html") {
			w.Header().Set("Cache-Control", "no-store")
		}
		h.ServeHTTP(w, r)
	})
}

func openBrowser(url string, logf func(string, ...any)) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil && logf != nil {
		logf("could not open browser: %v", err)
	}
}
