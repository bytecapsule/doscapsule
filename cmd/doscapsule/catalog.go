package main

// Catalog: recursive scan of a games root + @launcher.json manifests.
// Relative-path ids, hidden dirs skipped, symlinks never followed, manifest
// snake_case keys, fuzzy search across fields.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ManifestLaunch struct {
	Config     *string `json:"config"`
	Command    *string `json:"command"`
	WorkingDir *string `json:"working_dir"`
	Fullscreen *bool   `json:"fullscreen"`
	Cycles     *string `json:"cycles"`
}

type ManifestMedia struct {
	Cover       *string  `json:"cover"`
	Screenshots []string `json:"screenshots"`
	Manual      *string  `json:"manual"`
	Video       *string  `json:"video"`
	Music       *string  `json:"music"`
}

type Manifest struct {
	ID          *string         `json:"id"`
	Title       *string         `json:"title"`
	SortTitle   *string         `json:"sort_title"`
	Platform    *string         `json:"platform"`
	Developer   *string         `json:"developer"`
	Publisher   *string         `json:"publisher"`
	Year        *int            `json:"year"`
	Genre       *string         `json:"genre"`
	Series      *string         `json:"series"`
	Region      *string         `json:"region"`
	MaxPlayers  *int            `json:"max_players"`
	Rating      *float64        `json:"rating"`
	Description *string         `json:"description"`
	Notes       *string         `json:"notes"`
	Launch      *ManifestLaunch `json:"launch"`
	Media       *ManifestMedia  `json:"media"`
}

type LocalGame struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	SortTitle       string   `json:"sort_title"`
	Platform        string   `json:"platform"`
	Developer       *string  `json:"developer"`
	Publisher       *string  `json:"publisher"`
	Year            *int     `json:"year"`
	Genre           *string  `json:"genre"`
	Rating          *float64 `json:"rating"`
	Description     *string  `json:"description"`
	Notes           *string  `json:"notes"`
	Favorited       bool     `json:"favorited"`
	LastPlayed      *int64   `json:"last_played"`
	HasThumbnail    bool     `json:"has_thumbnail"`
	Installed       bool     `json:"installed"`
	CoverPath       *string  `json:"cover_path"`
	ScreenshotPaths []string `json:"screenshot_paths"`
	ManualPath      *string  `json:"manual_path"`
	GameDir         string   `json:"game_dir"`
}

const manifestName = "@launcher.json"

// manifestFile returns the manifest path for a game dir, or "" if none.
func manifestFile(dir string) string {
	if p := filepath.Join(dir, manifestName); fileExists(p) {
		return p
	}
	return ""
}

const confName = "@dosbox.conf"

// The virtual path the frontend asks /media for when a game ships no cover.
const fallbackCoverPath = "/builtin/fallback-cover.jpg"

func fileExists(p string) bool {
	return p != "" && filepath.Base(p) != "" && func() bool { st, err := os.Stat(p); return err == nil && st.Mode().IsRegular() }()
}

func isGameDir(dir string) bool {
	if manifestFile(dir) != "" {
		return true
	}
	hasConf := fileExists(filepath.Join(dir, confName))
	hasBat := fileExists(filepath.Join(dir, "play.bat")) || fileExists(filepath.Join(dir, "PLAY.BAT")) ||
		fileExists(filepath.Join(dir, "run.bat")) || hasAnyExt(dir, []string{".bat", ".exe", ".com"})
	return hasConf && hasBat
}

func hasAnyExt(dir string, exts []string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		for _, t := range exts {
			if ext == t {
				return true
			}
		}
	}
	return false
}

func loadManifest(dir string) Manifest {
	var m Manifest
	b, err := os.ReadFile(manifestFile(dir))
	if err == nil {
		_ = json.Unmarshal(b, &m)
	}
	return m
}

func firstImage(dir string, match func(name string) bool) *string {
	entries, _ := os.ReadDir(dir)
	var found []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			continue
		}
		if match(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))) {
			found = append(found, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(found)
	if len(found) == 0 {
		return nil
	}
	return &found[0]
}

func coverPathOf(dir string, m Manifest) *string {
	if m.Media != nil && m.Media.Cover != nil {
		p := filepath.Join(dir, *m.Media.Cover)
		if fileExists(p) {
			return &p
		}
	}
	// Standard: `@cover.jpg` (the @ prefix marks launcher metadata files).
	// Plain `cover.jpg` is still accepted for older folders.
	name := func(n string) bool { return strings.EqualFold(strings.TrimPrefix(n, "@"), "cover") }
	return firstImage(dir, name)
}

func screenshotPathsOf(dir string, m Manifest) []string {
	if m.Media != nil && len(m.Media.Screenshots) > 0 {
		var out []string
		for _, p := range m.Media.Screenshots {
			full := filepath.Join(dir, p)
			if fileExists(full) {
				out = append(out, full)
			}
		}
		return out
	}
	entries, _ := os.ReadDir(dir)
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			continue
		}
		stem := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		stem = strings.TrimPrefix(stem, "@")
		if strings.HasPrefix(strings.ToLower(stem), "screenshot") {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out)
	return out
}

func manualPathOf(dir string, m Manifest) *string {
	if m.Media != nil && m.Media.Manual != nil {
		p := filepath.Join(dir, *m.Media.Manual)
		if fileExists(p) {
			return &p
		}
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".pdf" || ext == ".txt" || ext == ".html" || ext == ".htm" {
			p := filepath.Join(dir, e.Name())
			return &p
		}
	}
	return nil
}

func gameFromDir(root, dir string) *LocalGame {
	if !isGameDir(dir) {
		return nil
	}
	// Unique, readable id: path relative to the games root.
	rel, err := filepath.Rel(root, dir)
	id := filepath.Base(dir)
	if err == nil && rel != "." {
		id = filepath.ToSlash(rel)
	}
	m := loadManifest(dir)
	title := id
	if m.Title != nil && *m.Title != "" {
		title = *m.Title
	}
	sortTitle := title
	if m.SortTitle != nil && *m.SortTitle != "" {
		sortTitle = *m.SortTitle
	}
	platform := "DOS"
	if m.Platform != nil && *m.Platform != "" {
		platform = *m.Platform
	}
	found := coverPathOf(dir, m)
	hasThumb := found != nil
	if found == nil {
		fb := fallbackCoverPath // virtual path served from the embedded bytes
		found = &fb
		hasThumb = true
	}
	var dev, pub, genre, desc, notes *string
	if m.Developer != nil {
		dev = m.Developer
	}
	if m.Publisher != nil {
		pub = m.Publisher
	}
	if m.Genre != nil {
		genre = m.Genre
	}
	if m.Description != nil {
		desc = m.Description
	}
	if m.Notes != nil {
		notes = m.Notes
	}
	return &LocalGame{
		ID: id, Title: title, SortTitle: strings.ToLower(sortTitle), Platform: platform,
		Developer: dev, Publisher: pub, Year: m.Year, Genre: genre, Rating: m.Rating,
		Description: desc, Notes: notes,
		Favorited: false, Installed: true, HasThumbnail: hasThumb,
		CoverPath:       found,
		ScreenshotPaths: screenshotPathsOf(dir, m),
		ManualPath:      manualPathOf(dir, m),
		GameDir:         dir,
	}
}

func scanGames(root string) []LocalGame {
	out := []LocalGame{}
	stack := []string{root}
	for len(stack) > 0 {
		dir := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if g := gameFromDir(root, dir); g != nil {
			out = append(out, *g)
			// A self-declared game is a leaf: its subfolders are data.
			if manifestFile(dir) != "" {
				continue
			}
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".") {
				continue // hidden: .git, .app bundles, OS metadata
			}
			p := filepath.Join(dir, e.Name())
			// Lstat: never follow symlinks (a link loop cannot wedge the scan).
			if fi, err := os.Lstat(p); err == nil && fi.IsDir() {
				stack = append(stack, p)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortTitle != out[j].SortTitle {
			return out[i].SortTitle < out[j].SortTitle
		}
		return out[i].ID < out[j].ID
	})
	return out
}
