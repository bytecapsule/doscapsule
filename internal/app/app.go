package app

// App is the transport-independent core of DOS Capsule. The HTTP server and
// the CLI both call these methods; neither owns business logic of its own.
//
// Adding a new entry point (MCP, a second UI, a batch script) means calling
// App, not re-implementing scanning, launching or state handling.

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

var (
	// ErrNoRoot means no games folder has been selected yet.
	ErrNoRoot = errors.New("no games root selected")
	// ErrGameNotFound means the id does not resolve to a directory under the
	// current (or supplied) games root.
	ErrGameNotFound = errors.New("game directory not found")
)

// ListOptions is the filter/sort used by both `GET /api/games` and
// `doscapsule list`.
type ListOptions struct {
	Query         string
	Genre         string
	Sort          string // "", "title", "year", "genre", "recent"
	FavoritesOnly bool
}

type Catalog struct {
	Root  string
	Games []LocalGame
}

type App struct {
	mu       sync.Mutex
	catalog  *Catalog
	store    *Store
	launcher *Launcher
}

func New() *App {
	return &App{store: LoadStore(), launcher: NewLauncher()}
}

// Root returns the active games folder: the scanned one if this process has
// one, otherwise the remembered folder if it still exists.
func (a *App) Root() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.catalog != nil {
		return a.catalog.Root
	}
	if r := a.store.Get().GamesRoot; r != nil && isDir(*r) {
		return *r
	}
	return ""
}

// RememberedRoot is the persisted games folder, or nil when it is unset or no
// longer a directory.
func (a *App) RememberedRoot() *string {
	root := a.store.Get().GamesRoot
	if root != nil && !isDir(*root) {
		return nil
	}
	return root
}

// Scan selects root, scans it and caches the catalog. It is the only entry
// point that changes the remembered games folder.
func (a *App) Scan(root string) (int, error) {
	if root == "" {
		return 0, errors.New("missing root")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return 0, err
	}
	if !isDir(abs) {
		return 0, fmt.Errorf("not a directory: %s", root)
	}
	games := scanGames(abs)
	if len(games) == 0 {
		return 0, fmt.Errorf("No games found in %s: each game needs its own subdirectory with a @launcher.json, play.bat and/or a DOSBox @dosbox.conf", root)
	}
	a.mu.Lock()
	a.catalog = &Catalog{Root: abs, Games: games}
	a.mu.Unlock()
	a.store.SetGamesRoot(abs)
	return len(games), nil
}

// Games returns the cached catalog (as scanned by this process), filtered and
// sorted. The HTTP server uses this; a fresh process has an empty catalog and
// should use ListIn instead.
func (a *App) Games(opts ListOptions) []LocalGame {
	a.mu.Lock()
	var games []LocalGame
	if a.catalog != nil {
		games = append([]LocalGame{}, a.catalog.Games...)
	}
	a.mu.Unlock()
	return a.refine(games, opts)
}

// ListIn scans root (without persisting it) and returns the filtered list.
// This is the stateless path used by the CLI.
func (a *App) ListIn(root string, opts ListOptions) ([]LocalGame, error) {
	if root == "" {
		return nil, ErrNoRoot
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if !isDir(abs) {
		return nil, fmt.Errorf("not a directory: %s", root)
	}
	return a.refine(scanGames(abs), opts), nil
}

// Genres lists the distinct genres in the cached catalog.
func (a *App) Genres() []string {
	a.mu.Lock()
	var games []LocalGame
	if a.catalog != nil {
		games = a.catalog.Games
	}
	a.mu.Unlock()
	set := map[string]bool{}
	for _, g := range games {
		if g.Genre != nil && *g.Genre != "" {
			set[*g.Genre] = true
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// StartGame starts the emulator. root "" means the current games folder. The
// caller owns the Process: call Track for a background launch (HTTP) or Wait
// for a foreground one (CLI).
func (a *App) StartGame(root, id string) (*Process, error) {
	if id == "" {
		return nil, errors.New("missing id")
	}
	if root == "" {
		root = a.Root()
	}
	if root == "" {
		return nil, ErrNoRoot
	}
	p, err := a.launcher.Start(root, id)
	if err != nil {
		return nil, err
	}
	a.store.StampPlayed(id)
	return p, nil
}

// Track keeps a background launch visible to Running/Stop.
func (a *App) Track(p *Process) { a.launcher.Track(p) }

func (a *App) Stop(id string) error { return a.launcher.Stop(id) }

func (a *App) Running() []string { return a.launcher.RunningIDs() }

func (a *App) Favorite(id string) bool { return a.store.ToggleFavorite(id) }

// ConfGet returns the game's @dosbox.conf. ok is false when the file does not
// exist yet.
func (a *App) ConfGet(id string) (content string, ok bool, err error) {
	dir, err := a.gameDir(id)
	if err != nil {
		return "", false, err
	}
	b, err := os.ReadFile(filepath.Join(dir, confName))
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return string(b), true, nil
}

func (a *App) ConfSave(id, content string) error {
	dir, err := a.gameDir(id)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, confName), []byte(content), 0o644)
}

// OpenDir reveals the game folder in the OS file manager.
func (a *App) OpenDir(id string) error {
	dir, err := a.gameDir(id)
	if err != nil {
		return err
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
	return cmd.Start()
}

// Clear drops the in-memory catalog; the remembered root is kept.
func (a *App) Clear() {
	a.mu.Lock()
	a.catalog = nil
	a.mu.Unlock()
}

func (a *App) gameDir(id string) (string, error) {
	root := a.Root()
	if root == "" {
		return "", ErrNoRoot
	}
	return resolveGameDir(root, id)
}

// refine overlays user state, applies filters and sorts in the same order for
// every entry point.
func (a *App) refine(games []LocalGame, opts ListOptions) []LocalGame {
	out := append([]LocalGame{}, games...)
	st := a.store.Get()
	favs := make(map[string]bool, len(st.Favorites))
	for _, f := range st.Favorites {
		favs[f] = true
	}
	for i := range out {
		out[i].Favorited = favs[out[i].ID]
		if t, ok := st.LastPlayed[out[i].ID]; ok {
			tt := t
			out[i].LastPlayed = &tt
		}
	}

	if opts.FavoritesOnly {
		kept := out[:0]
		for _, g := range out {
			if g.Favorited {
				kept = append(kept, g)
			}
		}
		out = kept
	}
	if q := strings.ToLower(opts.Query); q != "" {
		kept := out[:0]
		for _, g := range out {
			hay := []string{g.Title, g.ID, deref(g.Genre), deref(g.Developer), deref(g.Description)}
			for _, h := range hay {
				if strings.Contains(strings.ToLower(h), q) {
					kept = append(kept, g)
					break
				}
			}
		}
		out = kept
	}
	if opts.Genre != "" {
		kept := out[:0]
		for _, g := range out {
			if g.Genre != nil && *g.Genre == opts.Genre {
				kept = append(kept, g)
			}
		}
		out = kept
	}

	switch opts.Sort {
	case "year":
		sort.Slice(out, func(i, j int) bool {
			if c := cmpInt(out[i].Year, out[j].Year); c != 0 {
				return c < 0
			}
			return out[i].Title < out[j].Title
		})
	case "genre":
		sort.Slice(out, func(i, j int) bool {
			if deref(out[i].Genre) != deref(out[j].Genre) {
				return deref(out[i].Genre) < deref(out[j].Genre)
			}
			return out[i].Title < out[j].Title
		})
	case "recent":
		// Most-recently-played first; never played sinks to the bottom. A nil
		// LastPlayed must sort after a played one.
		sort.Slice(out, func(i, j int) bool {
			aPlayed := out[i].LastPlayed != nil
			bPlayed := out[j].LastPlayed != nil
			if aPlayed && bPlayed {
				if *out[i].LastPlayed != *out[j].LastPlayed {
					return *out[i].LastPlayed > *out[j].LastPlayed
				}
			} else if aPlayed != bPlayed {
				return aPlayed
			}
			if out[i].SortTitle != out[j].SortTitle {
				return out[i].SortTitle < out[j].SortTitle
			}
			return out[i].ID < out[j].ID
		})
	default:
		sort.Slice(out, func(i, j int) bool {
			if out[i].SortTitle != out[j].SortTitle {
				return out[i].SortTitle < out[j].SortTitle
			}
			return out[i].ID < out[j].ID
		})
	}
	return out
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
