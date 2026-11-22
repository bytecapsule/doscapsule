// DOS Capsule — pure-local DOS game frontend in a single Go binary.
// The SolidJS UI is embedded (go:embed) and served on 127.0.0.1; the same
// process scans game folders, launches DOSBox and manages favorites.
package main

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

//go:embed all:dist
var distFS embed.FS

//go:embed resources/fallback-cover.jpg
var fallbackCover []byte

// version is stamped at build time via -X main.version=<v> (see Makefile).
var version = "dev"

var (
	store    = LoadStore()
	launcher = NewLauncher()
	mu       sync.Mutex
	catalog  *Catalog // scanned games for the selected root
)

type Catalog struct {
	Root  string
	Games []LocalGame
}

func currentRoot() string {
	mu.Lock()
	defer mu.Unlock()
	if catalog == nil {
		if r := store.Get().GamesRoot; r != nil && isDir(*r) {
			return *r
		}
		return ""
	}
	return catalog.Root
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	http.Error(w, msg, code)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func decodeBody(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// ── API handlers ─────────────────────────────────────────────────────────────

func handleScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Root string `json:"root"`
	}
	if err := decodeBody(r, &req); err != nil || req.Root == "" {
		writeErr(w, http.StatusBadRequest, "missing root")
		return
	}
	if !isDir(req.Root) {
		writeErr(w, http.StatusBadRequest, "not a directory: "+req.Root)
		return
	}
	games := scanGames(req.Root)
	if len(games) == 0 {
		writeErr(w, http.StatusBadRequest,
			"No games found in "+req.Root+": each game needs its own subdirectory with a @launcher.json, play.bat and/or a DOSBox @dosbox.conf")
		return
	}
	mu.Lock()
	catalog = &Catalog{Root: req.Root, Games: games}
	mu.Unlock()
	store.SetGamesRoot(req.Root)
	writeJSON(w, map[string]int{"count": len(games)})
}

func handleGames(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("query"))
	genre := r.URL.Query().Get("genre")
	sortBy := r.URL.Query().Get("sortBy")
	favOnly := r.URL.Query().Get("favoritesOnly") == "1"

	mu.Lock()
	var games []LocalGame
	if catalog != nil {
		games = append([]LocalGame{}, catalog.Games...)
	}
	mu.Unlock()
	if games == nil {
		writeJSON(w, []LocalGame{})
		return
	}

	// Overlay user state at read time.
	favs := map[string]bool{}
	for _, f := range store.Get().Favorites {
		favs[f] = true
	}
	lp := store.Get().LastPlayed
	for i := range games {
		games[i].Favorited = favs[games[i].ID]
		if t, ok := lp[games[i].ID]; ok {
			tt := t
			games[i].LastPlayed = &tt
		}
	}

	if favOnly {
		kept := games[:0]
		for _, g := range games {
			if g.Favorited {
				kept = append(kept, g)
			}
		}
		games = kept
	}
	if q != "" {
		kept := games[:0]
		for _, g := range games {
			hay := []string{g.Title, g.ID, deref(g.Genre), deref(g.Developer), deref(g.Description)}
			for _, h := range hay {
				if strings.Contains(strings.ToLower(h), q) {
					kept = append(kept, g)
					break
				}
			}
		}
		games = kept
	}
	if genre != "" {
		kept := games[:0]
		for _, g := range games {
			if g.Genre != nil && *g.Genre == genre {
				kept = append(kept, g)
			}
		}
		games = kept
	}
	switch sortBy {
	case "year":
		sortGames(games, func(a, b LocalGame) bool {
			if cmpInt(a.Year, b.Year) != 0 {
				return cmpInt(a.Year, b.Year) < 0
			}
			return a.Title < b.Title
		})
	case "genre":
		sortGames(games, func(a, b LocalGame) bool {
			if deref(a.Genre) != deref(b.Genre) {
				return deref(a.Genre) < deref(b.Genre)
			}
			return a.Title < b.Title
		})
	case "recent":
		// Most-recently-played first; games never played sink to the bottom,
		// ordered by title. (cmpI64 returns -1 for nil, so flip both sides:
		// a nil LastPlayed must sort AFTER a played one.)
		sortGames(games, func(a, b LocalGame) bool {
			aPlayed := a.LastPlayed != nil
			bPlayed := b.LastPlayed != nil
			if aPlayed && bPlayed {
				if *a.LastPlayed != *b.LastPlayed {
					return *a.LastPlayed > *b.LastPlayed
				}
			} else if aPlayed != bPlayed {
				return aPlayed // played first
			}
			if a.SortTitle != b.SortTitle {
				return a.SortTitle < b.SortTitle
			}
			return a.ID < b.ID
		})
	default:
		sortGames(games, func(a, b LocalGame) bool {
			if a.SortTitle != b.SortTitle {
				return a.SortTitle < b.SortTitle
			}
			return a.ID < b.ID
		})
	}
	writeJSON(w, games)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func cmpInt(a, b *int) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return -1
	case b == nil:
		return 1
	case *a < *b:
		return -1
	case *a > *b:
		return 1
	}
	return 0
}

func cmpI64(a, b *int64) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return -1
	case b == nil:
		return 1
	case *a < *b:
		return -1
	case *a > *b:
		return 1
	}
	return 0
}

func sortGames(g []LocalGame, less func(a, b LocalGame) bool) {
	for i := 1; i < len(g); i++ {
		for j := i; j > 0 && less(g[j], g[j-1]); j-- {
			g[j], g[j-1] = g[j-1], g[j]
		}
	}
}

func handleGenres(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	var games []LocalGame
	if catalog != nil {
		games = catalog.Games
	}
	mu.Unlock()
	set := map[string]bool{}
	for _, g := range games {
		if g.Genre != nil && *g.Genre != "" {
			set[*g.Genre] = true
		}
	}
	out := []string{}
	for k := range set {
		out = append(out, k)
	}
	sortStrings(out)
	writeJSON(w, out)
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func handleRememberedRoot(w http.ResponseWriter, r *http.Request) {
	root := store.Get().GamesRoot
	if root != nil && !isDir(*root) {
		root = nil
	}
	writeJSON(w, root)
}

func handleLaunch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := decodeBody(r, &req); err != nil || req.ID == "" {
		writeErr(w, http.StatusBadRequest, "missing id")
		return
	}
	root := currentRoot()
	if root == "" {
		writeErr(w, http.StatusBadRequest, "no games root selected")
		return
	}
	msg, cmd, err := launcher.Launch(root, req.ID)
	if err != nil {
		if errors.Is(err, ErrAlreadyRunning) {
			writeErr(w, http.StatusConflict, "game is already running")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	launcher.Track(req.ID, cmd)
	store.StampPlayed(req.ID)
	writeJSON(w, map[string]string{"message": msg})
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad body")
		return
	}
	if err := launcher.Stop(req.ID); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func handleRunning(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, launcher.RunningIDs())
}

func handleFavorite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := decodeBody(r, &req); err != nil || req.ID == "" {
		writeErr(w, http.StatusBadRequest, "missing id")
		return
	}
	writeJSON(w, map[string]bool{"favorited": store.ToggleFavorite(req.ID)})
}

func handleConfGet(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	dir, err := resolveGameDir(currentRoot(), id)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	b, err := os.ReadFile(filepath.Join(dir, confName))
	if err != nil {
		writeJSON(w, map[string]any{"content": nil})
		return
	}
	writeJSON(w, map[string]any{"content": string(b)})
}

func handleConfSave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}
	if err := decodeBody(r, &req); err != nil || req.ID == "" {
		writeErr(w, http.StatusBadRequest, "bad body")
		return
	}
	dir, err := resolveGameDir(currentRoot(), req.ID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := os.WriteFile(filepath.Join(dir, confName), []byte(req.Content), 0o644); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func handleOpenDir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := decodeBody(r, &req); err != nil || req.ID == "" {
		writeErr(w, http.StatusBadRequest, "missing id")
		return
	}
	dir, err := resolveGameDir(currentRoot(), req.ID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", dir)
	case "windows":
		cmd = exec.Command("explorer", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	if err := cmd.Start(); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// handlePickFolder opens the OS-native folder chooser and returns the path.
// This keeps the browser UI on parity with the Tauri directory dialog.
func handlePickFolder(w http.ResponseWriter, r *http.Request) {
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
		// Cancelled (or no chooser available): null, like the Tauri dialog.
		writeJSON(w, map[string]any{"path": nil})
		return
	}
	p := strings.TrimSpace(string(out))
	p = strings.Trim(p, "/")
	p = "/" + p // normalize: osascript returns a trailing slash
	writeJSON(w, map[string]string{"path": filepath.Clean(p)})
}

// /media?path=... serves cover/screenshot bytes. Allowed: anything under the
// selected games root, plus the embedded fallback cover's virtual path.
func handleMedia(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	if p == "" {
		writeErr(w, http.StatusBadRequest, "missing path")
		return
	}
	if p == fallbackCoverPath {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(fallbackCover)
		return
	}
	root := currentRoot()
	if root == "" {
		writeErr(w, http.StatusForbidden, "no games root")
		return
	}
	abs, _ := filepath.Abs(p)
	rootAbs, _ := filepath.Abs(root)
	if !strings.HasPrefix(abs, rootAbs+string(os.PathSeparator)) {
		writeErr(w, http.StatusForbidden, "path outside the games root")
		return
	}
	// Covers change when the user re-packs a game; a reload must pick them up.
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, abs)
}

func noStoreHTML(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, ".html") {
			w.Header().Set("Cache-Control", "no-store")
		}
		h.ServeHTTP(w, r)
	})
}

func handleClear(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	catalog = nil
	mu.Unlock()
	writeJSON(w, map[string]bool{"ok": true})
}

// ── Server bootstrap ─────────────────────────────────────────────────────────

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("could not open browser: %v", err)
	}
}

func main() {
	port := flag.Int("port", 8420, "port to listen on")
	openBrowserFlag := flag.Bool("open-browser", true, "open the UI in the default browser")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("DOS Capsule", version)
		return
	}

	mux := http.NewServeMux()
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		log.Fatal(err)
	}
	// The HTML shell is never cached: after a rebuild+restart the browser must
	// pick up the new hashed asset names instead of a stale page.
	mux.Handle("/", noStoreHTML(http.FileServer(http.FS(sub))))
	mux.HandleFunc("/media", handleMedia)
	mux.HandleFunc("/api/scan", handleScan)
	mux.HandleFunc("/api/games", handleGames)
	mux.HandleFunc("/api/genres", handleGenres)
	mux.HandleFunc("/api/remembered_root", handleRememberedRoot)
	mux.HandleFunc("/api/launch", handleLaunch)
	mux.HandleFunc("/api/stop", handleStop)
	mux.HandleFunc("/api/running", handleRunning)
	mux.HandleFunc("/api/favorite", handleFavorite)
	mux.HandleFunc("/api/conf", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handleConfSave(w, r)
		} else {
			handleConfGet(w, r)
		}
	})
	mux.HandleFunc("/api/open", handleOpenDir)
	mux.HandleFunc("/api/pick_folder", handlePickFolder)
	mux.HandleFunc("/api/clear", handleClear)

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		// Port busy: fall back to a random free port.
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatal(err)
		}
	}
	url := fmt.Sprintf("http://127.0.0.1:%d", ln.Addr().(*net.TCPAddr).Port)
	log.Printf("DOS Capsule running at %s", url)

	if *openBrowserFlag {
		go openBrowser(url)
	}
	log.Fatal(http.Serve(ln, mux))
}

var _ = errors.New // keep import set stable for future use
