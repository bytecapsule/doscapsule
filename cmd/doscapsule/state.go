package main

// Persisted user state (state.json): selected games root, favorites and
// last-played timestamps — the same semantics as the Tauri version.

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
// The whole point is portability: when the DOSBox bundle sits beside the
// executable (binaries/dosbox/), state is kept in <exe>/data, so copying the
// project folder to another machine carries the library, the settings and the
// emulator in one piece. Without that bundle (running via `go run`, or an
// install that only has the binary) it falls back to the OS config dir.
func stateDir() (string, error) {
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

func (s *Store) Get() PersistedState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.st.clone()
}

func (s *Store) SetGamesRoot(root string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := root
	s.st.GamesRoot = &r
	s.saveLocked()
}

func (s *Store) ToggleFavorite(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	if s.st.LastPlayed == nil {
		s.st.LastPlayed = map[string]int64{}
	}
	s.st.LastPlayed[id] = time.Now().Unix()
	s.saveLocked()
}

func (s *Store) saveLocked() {
	dir, err := stateDir()
	if err != nil {
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	b, _ := json.MarshalIndent(s.st, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "state.json"), b, 0o644)
}

func (s PersistedState) clone() PersistedState {
	c := PersistedState{GamesRoot: s.GamesRoot, Favorites: append([]string(nil), s.Favorites...), LastPlayed: map[string]int64{}}
	for k, v := range s.LastPlayed {
		c.LastPlayed[k] = v
	}
	return c
}
