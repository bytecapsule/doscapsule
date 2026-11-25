package app

import (
	"path/filepath"
	"testing"
)

func TestListInFavorites(t *testing.T) {
	t.Setenv("DOSCAPSULE_DATA_DIR", filepath.Join(t.TempDir(), "state"))
	root := t.TempDir()
	makeGame(t, root, "A", "Alpha")
	makeGame(t, root, "B", "Beta")

	a := New()
	if !a.Favorite("A") {
		t.Fatal("Favorite(A) should add it")
	}
	games, err := a.ListIn(root, ListOptions{FavoritesOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 1 || games[0].ID != "A" {
		t.Fatalf("expected only A, got %+v", games)
	}
	if !games[0].Favorited {
		t.Fatal("favorited flag not overlaid")
	}
}

func TestScanStoresAbsoluteRoot(t *testing.T) {
	t.Setenv("DOSCAPSULE_DATA_DIR", filepath.Join(t.TempDir(), "state"))
	base := t.TempDir()
	root := filepath.Join(base, "games")
	makeGame(t, root, "A", "Alpha")
	t.Chdir(base)

	a := New()
	if _, err := a.Scan("games"); err != nil {
		t.Fatal(err)
	}
	if got := a.Root(); !filepath.IsAbs(got) {
		t.Fatalf("root should be absolute, got %q", got)
	}
}

func TestConfRoundTrip(t *testing.T) {
	t.Setenv("DOSCAPSULE_DATA_DIR", filepath.Join(t.TempDir(), "state"))
	root := t.TempDir()
	makeGame(t, root, "A", "Alpha")

	a := New()
	if _, err := a.Scan(root); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := a.ConfGet("A"); err != nil || ok {
		t.Fatalf("missing conf: ok=%v err=%v", ok, err)
	}
	const body = "[sdl]\nfullscreen=false\n"
	if err := a.ConfSave("A", body); err != nil {
		t.Fatal(err)
	}
	got, ok, err := a.ConfGet("A")
	if err != nil || !ok {
		t.Fatalf("saved conf not readable: ok=%v err=%v", ok, err)
	}
	if got != body {
		t.Fatalf("content = %q, want %q", got, body)
	}
}
