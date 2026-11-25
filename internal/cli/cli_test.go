package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/bytecapsule/doscapsule/internal/app"
)

func writeGame(t *testing.T, root, rel, title string, year int, genre string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"title":"` + title + `"`
	if year > 0 {
		manifest += `,"year":` + strconv.Itoa(year)
	}
	if genre != "" {
		manifest += `,"genre":"` + genre + `"`
	}
	manifest += `}`
	if err := os.WriteFile(filepath.Join(dir, "@launcher.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "play.bat"), []byte("@echo off\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// run executes one command against its own app instance (like a fresh CLI
// process) and returns the exit code plus captured streams.
func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(args, Deps{
		Version: "test",
		App:     app.New(),
		Out:     &out,
		Err:     &errb,
	})
	return code, out.String(), errb.String()
}

func setup(t *testing.T) string {
	t.Helper()
	t.Setenv("DOSCAPSULE_DATA_DIR", filepath.Join(t.TempDir(), "state"))
	root := filepath.Join(t.TempDir(), "games")
	writeGame(t, root, "Alpha", "Alpha", 1993, "FPS")
	writeGame(t, root, "Beta", "Beta", 2001, "Puzzle")
	return root
}

func TestScanThenListFromAnotherProcess(t *testing.T) {
	root := setup(t)

	code, out, errb := run(t, "scan", root)
	if code != 0 {
		t.Fatalf("scan exit=%d stderr=%q", code, errb)
	}
	if !strings.Contains(out, "Scanned 2 games") {
		t.Fatalf("unexpected scan output: %q", out)
	}

	// A separate app instance only knows state.json; list must still work.
	code, out, errb = run(t, "list", "--json")
	if code != 0 {
		t.Fatalf("list exit=%d stderr=%q", code, errb)
	}
	var games []app.LocalGame
	if err := json.Unmarshal([]byte(out), &games); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(games) != 2 {
		t.Fatalf("expected 2 games, got %d", len(games))
	}
}

func TestListSortByYear(t *testing.T) {
	root := setup(t)
	if code, _, errb := run(t, "scan", root); code != 0 {
		t.Fatalf("scan failed: %s", errb)
	}
	code, out, errb := run(t, "list", "--sort", "year", "--json")
	if code != 0 {
		t.Fatalf("list exit=%d stderr=%q", code, errb)
	}
	var games []app.LocalGame
	if err := json.Unmarshal([]byte(out), &games); err != nil {
		t.Fatal(err)
	}
	if len(games) != 2 {
		t.Fatalf("expected 2 games, got %d", len(games))
	}
	if games[0].ID != "Alpha" || games[1].ID != "Beta" {
		t.Fatalf("expected Alpha then Beta, got %q then %q", games[0].ID, games[1].ID)
	}
}

func TestSearchJSONWithTrailingFlag(t *testing.T) {
	root := setup(t)
	if code, _, errb := run(t, "scan", root); code != 0 {
		t.Fatalf("scan failed: %s", errb)
	}
	code, out, errb := run(t, "search", "alpha", "--json")
	if code != 0 {
		t.Fatalf("search exit=%d stderr=%q", code, errb)
	}
	var games []app.LocalGame
	if err := json.Unmarshal([]byte(out), &games); err != nil {
		t.Fatal(err)
	}
	if len(games) != 1 || games[0].ID != "Alpha" {
		t.Fatalf("expected Alpha only, got %+v", games)
	}
}

func TestScanRejectsJSONFlag(t *testing.T) {
	root := setup(t)
	if code, _, _ := run(t, "scan", root, "--json"); code != exitUsage {
		t.Fatalf("scan --json should be a usage error, got %d", code)
	}
}

func TestErrorsAndExitCodes(t *testing.T) {
	t.Setenv("DOSCAPSULE_DATA_DIR", filepath.Join(t.TempDir(), "state"))

	if code, _, errb := run(t, "list"); code != exitErr || !strings.Contains(errb, "no games folder") {
		t.Fatalf("list without root: exit=%d stderr=%q", code, errb)
	}
	if code, _, _ := run(t, "scan"); code != exitUsage {
		t.Fatalf("scan without dir should be usage error, got %d", code)
	}
	if code, _, _ := run(t, "nonsense"); code != exitUsage {
		t.Fatalf("unknown command should be usage error, got %d", code)
	}
	if code, out, _ := run(t, "help"); code != 0 || !strings.Contains(out, "DOS Capsule") {
		t.Fatalf("help: exit=%d out=%q", code, out)
	}
}
