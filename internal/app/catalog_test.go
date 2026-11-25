package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeGame(t *testing.T, root, rel, title string) string {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{`
	if title != "" {
		manifest += `"title":` + `"` + title + `"`
	}
	manifest += `,"launch":{"command":"call play.bat"}}`
	if err := os.WriteFile(filepath.Join(dir, manifestName), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "play.bat"), []byte("@echo off\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestScanRelativeIDsAndNesting(t *testing.T) {
	root := t.TempDir()
	makeGame(t, root, "DUKE3D", "Duke Nukem 3D")
	makeGame(t, root, "collections/puzzle/MyGame", "我的游戏")
	os.MkdirAll(filepath.Join(root, "DUKE3D", "DATA"), 0o755) // game data subdir

	games := scanGames(root)
	if len(games) != 2 {
		t.Fatalf("expected 2 games, got %d: %+v", len(games), games)
	}
	ids := map[string]string{}
	for _, g := range games {
		ids[g.ID] = g.Title
	}
	if ids["DUKE3D"] != "Duke Nukem 3D" {
		t.Errorf("DUKE3D id missing: %+v", ids)
	}
	if ids["collections/puzzle/MyGame"] != "我的游戏" {
		t.Errorf("nested relative id missing: %+v", ids)
	}
}

func TestScanSkipsHiddenAndNonGames(t *testing.T) {
	root := t.TempDir()
	makeGame(t, root, "GOOD", "")
	os.MkdirAll(filepath.Join(root, ".hidden", "Game"), 0o755)
	makeGame(t, filepath.Join(root, ".hidden"), "Game", "")
	games := scanGames(root)
	if len(games) != 1 || games[0].ID != "GOOD" {
		t.Fatalf("expected only GOOD, got %+v", games)
	}
}

func TestManifestSnakeCase(t *testing.T) {
	root := t.TempDir()
	dir := makeGame(t, root, "GM", "")
	os.WriteFile(filepath.Join(dir, manifestName),
		[]byte(`{"title":"X","sort_title":"x","max_players":2,"launch":{"command":"call play.bat","working_dir":"."}}`), 0o644)
	games := scanGames(root)
	if len(games) != 1 {
		t.Fatal("game not scanned")
	}
	if games[0].SortTitle != "x" {
		t.Errorf("sort_title not parsed: %q", games[0].SortTitle)
	}
}

func TestResolveGameDirBlocksTraversal(t *testing.T) {
	root := t.TempDir()
	makeGame(t, root, "A", "")
	if _, err := resolveGameDir(root, "../escape"); err == nil {
		t.Fatal("traversal must be rejected")
	}
	if _, err := resolveGameDir(root, "A"); err != nil {
		t.Fatalf("valid id rejected: %v", err)
	}
}

func TestBuildConfHasAutoexec(t *testing.T) {
	c := buildConf("@echo off\nmount c \"X\"\nc:\ncall play.bat\nexit\n", true, "auto")
	if !strings.Contains(c, "[autoexec]") || !strings.Contains(c, "call play.bat") || !strings.Contains(c, "fullscreen=true") {
		t.Errorf("bad conf: %s", c)
	}
}
