package app

// Persisted user state (state.json): selected games root, favorites and
// last-played timestamps.
//
// The CLI and the web server can run as separate processes and both write this
// file. To keep them from overwriting each other, every mutation re-reads the
// file first (read-modify-write) and saves atomically (temp file + rename).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type PersistedState struct {
	GamesRoot  *string          `json:"games_root"`
	Favorites  []string         `json:"favorites"`
	LastPlayed map[string]int64 `json:"last_played"`
}

type Store struct {
	mu sync.Mutex
	st PersistedState
}

// configDir is the OS per-user config dir for this app (used when the
// portable layout is not present). Both the current name and the older
// "dosgame-launcher" location are recognised so existing state carries over.
func configDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "doscapsule")
}

// legacyStateDir is the previous config directory name, kept so an existing
// library (favorites, last-played timestamps, chosen games root) survives the
// rename to doscapsule.
func legacyStateDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "dosgame-launcher")
}

// stateDir returns the folder holding state.json and launch/defaults.conf.
//
// DOSCAPSULE_DATA_DIR wins over everything: it lets a test or a second
// instance keep its own library instead of overwriting the real one. A dev
// build that scans a scratch folder must not clobber the games root, favorites
// and play history the user actually cares about.
//
// Otherwise the whole point is portability: when the DOSBox bundle sits beside
// the executable (binaries/dosbox/), state is kept in <exe>/data, so copying
// the project folder to another machine carries the library, the settings and
// the emulator in one piece. Without that bundle (running via `go run`, or an
// install that only has the binary) it falls back to the OS config dir.
func stateDir() (string, error) {
	if override := os.Getenv("DOSCAPSULE_DATA_DIR"); override != "" {
		if err := os.MkdirAll(override, 0o755); err != nil {
			return "", err
		}
		return override, nil
	}
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		portable := filepath.Join(base, "data")
		bundled := isDir(filepath.Join(base, "binaries", "dosbox"))
		if bundled || isDir(portable) {
			if err := os.MkdirAll(portable, 0o755); err == nil {
				migrateLegacyState(portable)
				return portable, nil
			}
		}
	}
	return configDir(), nil
}

// migrateLegacyState copies state.json from any older location once, so the
// rename to doscapsule does not look like a factory reset. It checks both the
// very old "dosgame-launcher" folder and the legacy state dir.
func migrateLegacyState(dst string) {
	target := filepath.Join(dst, "state.json")
	if fileExists(target) {
		return
	}
	for _, src := range []string{filepath.Join(legacyStateDir(), "state.json")} {
		if b, err := os.ReadFile(src); err == nil {
			_ = os.WriteFile(target, b, 0o644)
			break
		}
	}
}

func LoadStore() *Store {
	st := PersistedState{LastPlayed: map[string]int64{}}
	if dir, err := stateDir(); err == nil {
		if b, err := os.ReadFile(filepath.Join(dir, "state.json")); err == nil {
			_ = json.Unmarshal(b, &st)
			if st.LastPlayed == nil {
				st.LastPlayed = map[string]int64{}
			}
		}
	}
	return &Store{st: st}
}

// Get returns a snapshot. It re-reads state.json first so a change made by
// another process (e.g. `doscapsule scan` while the UI is open) is visible.
func (s *Store) Get() PersistedState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshLocked()
	return s.st.clone()
}

func (s *Store) SetGamesRoot(root string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshLocked()
	r := root
	s.st.GamesRoot = &r
	s.saveLocked()
}

func (s *Store) ToggleFavorite(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshLocked()
	for i, f := range s.st.Favorites {
		if f == id {
			s.st.Favorites = append(s.st.Favorites[:i], s.st.Favorites[i+1:]...)
			s.saveLocked()
			return false
		}
	}
	s.st.Favorites = append(s.st.Favorites, id)
	s.saveLocked()
	return true
}

func (s *Store) StampPlayed(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshLocked()
	if s.st.LastPlayed == nil {
		s.st.LastPlayed = map[string]int64{}
	}
	s.st.LastPlayed[id] = time.Now().Unix()
	s.saveLocked()
}

// refreshLocked replaces the in-memory state with the file on disk, if it can
// be read and parsed. A missing or corrupt file leaves the current state alone.
func (s *Store) refreshLocked() {
	dir, err := stateDir()
	if err != nil {
		return
	}
	b, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		return
	}
	var st PersistedState
	if json.Unmarshal(b, &st) != nil {
		return
	}
	if st.LastPlayed == nil {
		st.LastPlayed = map[string]int64{}
	}
	s.st = st
}

func (s *Store) saveLocked() {
	dir, err := stateDir()
	if err != nil {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	b, err := json.MarshalIndent(s.st, "", "  ")
	if err != nil {
		return
	}
	// Atomic replace: a reader (the other process) never sees a half-written
	// state.json. A unique temp name means two writers cannot clobber each
	// other's partial file; the rename is the commit point.
	f, err := os.CreateTemp(dir, "state-*.tmp")
	if err != nil {
		return
	}
	name := f.Name()
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		_ = os.Remove(name)
		return
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return
	}
	_ = os.Chmod(name, 0o644)
	if err := os.Rename(name, filepath.Join(dir, "state.json")); err != nil {
		_ = os.Remove(name)
	}
}

func (s PersistedState) clone() PersistedState {
	c := PersistedState{GamesRoot: s.GamesRoot, Favorites: append([]string(nil), s.Favorites...), LastPlayed: map[string]int64{}}
	for k, v := range s.LastPlayed {
		c.LastPlayed[k] = v
	}
	return c
}
