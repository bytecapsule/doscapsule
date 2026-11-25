package cli

// Command-line transport for the App core. Each command is a single process:
// no daemon is required, and `run` blocks until the game exits.
//
// Contract for scripts and agents:
//   - stdout carries only data (table or JSON)
//   - stderr carries diagnostics and progress
//   - exit codes: 0 ok, 1 runtime error, 2 usage, 130 interrupted

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/bytecapsule/doscapsule/internal/app"
)

const (
	exitOK          = 0
	exitErr         = 1
	exitUsage       = 2
	exitInterrupted = 130
)

type ServeOptions struct {
	Port        int
	OpenBrowser bool
}

// Deps is the composition root's contribution: the core app and the one
// operation the CLI cannot implement by itself (starting the HTTP server).
type Deps struct {
	Version string
	App     *app.App
	Serve   func(*app.App, ServeOptions) error
	Out     io.Writer
	Err     io.Writer
}

// Run dispatches one command and returns the process exit code.
func Run(args []string, deps Deps) int {
	out, errw := deps.Out, deps.Err
	if out == nil {
		out = os.Stdout
	}
	if errw == nil {
		errw = os.Stderr
	}
	if deps.App == nil {
		deps.App = app.New()
	}

	// Top-level help is the full command list, not the legacy flag usage.
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		printUsage(out)
		return exitOK
	}
	// Legacy flags: `doscapsule -port 1234`, `-open-browser=false`, `-version`.
	// Keeping these means existing scripts and docs do not break.
	if len(args) > 0 && strings.HasPrefix(args[0], "-") {
		return legacy(args, deps, out, errw)
	}
	if len(args) == 0 {
		return serve(deps, ServeOptions{Port: 8420, OpenBrowser: true}, errw)
	}

	switch args[0] {
	case "serve":
		return serveCmd(args[1:], deps, errw)
	case "scan":
		return scanCmd(args[1:], deps, out, errw)
	case "list":
		return listCmd(args[1:], deps, out, errw)
	case "search":
		return searchCmd(args[1:], deps, out, errw)
	case "run":
		return runCmd(args[1:], deps, errw)
	case "version":
		fmt.Fprintf(out, "DOS Capsule %s\n", deps.Version)
		return exitOK
	case "help", "-h", "--help":
		printUsage(out)
		return exitOK
	default:
		fmt.Fprintf(errw, "error: unknown command %q\n\n", args[0])
		printUsage(errw)
		return exitUsage
	}
}

func legacy(args []string, deps Deps, out, errw io.Writer) int {
	fs := newFlagSet("doscapsule", "usage: doscapsule [-port N] [-open-browser=true|false] [-version]", errw)
	port := fs.Int("port", 8420, "port to listen on")
	open := fs.Bool("open-browser", true, "open the UI in the default browser")
	ver := fs.Bool("version", false, "print version and exit")
	flagArgs, pos := splitFlags(args, map[string]bool{"port": true})
	if err := fs.Parse(flagArgs); err != nil {
		return flagExit(err)
	}
	if len(pos) != 0 {
		return usageErr(errw, "doscapsule takes no positional arguments")
	}
	if *ver {
		fmt.Fprintf(out, "DOS Capsule %s\n", deps.Version)
		return exitOK
	}
	return serve(deps, ServeOptions{Port: *port, OpenBrowser: *open}, errw)
}

func serveCmd(args []string, deps Deps, errw io.Writer) int {
	fs := newFlagSet("serve", "usage: doscapsule serve [--port N] [--no-open]", errw)
	port := fs.Int("port", 8420, "port to listen on")
	noOpen := fs.Bool("no-open", false, "do not open the browser")
	flagArgs, pos := splitFlags(args, map[string]bool{"port": true})
	if err := fs.Parse(flagArgs); err != nil {
		return flagExit(err)
	}
	if len(pos) != 0 {
		return usageErr(errw, "serve takes no positional arguments")
	}
	return serve(deps, ServeOptions{Port: *port, OpenBrowser: !*noOpen}, errw)
}

func serve(deps Deps, opts ServeOptions, errw io.Writer) int {
	if deps.Serve == nil {
		fmt.Fprintln(errw, "error: no server configured")
		return exitErr
	}
	if err := deps.Serve(deps.App, opts); err != nil {
		fmt.Fprintln(errw, "error:", err)
		return exitErr
	}
	return exitOK
}

func scanCmd(args []string, deps Deps, out, errw io.Writer) int {
	fs := newFlagSet("scan", "usage: doscapsule scan <games-dir>", errw)
	flagArgs, pos := splitFlags(args, nil)
	if err := fs.Parse(flagArgs); err != nil {
		return flagExit(err)
	}
	if len(pos) != 1 {
		return usageErr(errw, "usage: doscapsule scan <games-dir>")
	}
	root := pos[0]
	n, err := deps.App.Scan(root)
	if err != nil {
		fmt.Fprintln(errw, "error:", err)
		return exitErr
	}
	// One human-readable line; the exit code is the machine signal. Use
	// `list --json` when a caller needs structured data.
	fmt.Fprintf(out, "Scanned %d games in %s\n", n, root)
	return exitOK
}

func listCmd(args []string, deps Deps, out, errw io.Writer) int {
	fs := newFlagSet("list", "usage: doscapsule list [--root DIR] [--json] [--sort title|year|genre|recent] [--genre G] [--favorites]", errw)
	root := fs.String("root", "", "games folder (default: remembered)")
	asJSON := fs.Bool("json", false, "machine-readable output")
	sortBy := fs.String("sort", "title", "title | year | genre | recent")
	genre := fs.String("genre", "", "only this genre")
	favorites := fs.Bool("favorites", false, "only favourites")
	flagArgs, pos := splitFlags(args, map[string]bool{"root": true, "sort": true, "genre": true})
	if err := fs.Parse(flagArgs); err != nil {
		return flagExit(err)
	}
	if len(pos) != 0 {
		return usageErr(errw, "usage: doscapsule list [--root DIR] [--json] [--sort ...] [--genre G] [--favorites]")
	}
	if !validSort(*sortBy) {
		return usageErr(errw, "invalid --sort "+strconv.Quote(*sortBy)+" (title|year|genre|recent)")
	}
	games, code := loadGames(deps, *root, app.ListOptions{
		Genre: *genre, Sort: *sortBy, FavoritesOnly: *favorites,
	}, errw)
	if code != exitOK {
		return code
	}
	return emitGames(games, *asJSON, out, errw)
}

func searchCmd(args []string, deps Deps, out, errw io.Writer) int {
	fs := newFlagSet("search", "usage: doscapsule search <query> [--root DIR] [--json] [--genre G] [--favorites]", errw)
	root := fs.String("root", "", "games folder (default: remembered)")
	asJSON := fs.Bool("json", false, "machine-readable output")
	genre := fs.String("genre", "", "only this genre")
	favorites := fs.Bool("favorites", false, "only favourites")
	flagArgs, pos := splitFlags(args, map[string]bool{"root": true, "genre": true})
	if err := fs.Parse(flagArgs); err != nil {
		return flagExit(err)
	}
	if len(pos) == 0 {
		return usageErr(errw, "usage: doscapsule search <query> [--root DIR] [--json]")
	}
	query := strings.Join(pos, " ")
	games, code := loadGames(deps, *root, app.ListOptions{
		Query: query, Genre: *genre, Sort: "title", FavoritesOnly: *favorites,
	}, errw)
	if code != exitOK {
		return code
	}
	return emitGames(games, *asJSON, out, errw)
}

func runCmd(args []string, deps Deps, errw io.Writer) int {
	fs := newFlagSet("run", "usage: doscapsule run <game-id> [--root DIR]", errw)
	root := fs.String("root", "", "games folder (default: remembered)")
	flagArgs, pos := splitFlags(args, map[string]bool{"root": true})
	if err := fs.Parse(flagArgs); err != nil {
		return flagExit(err)
	}
	if len(pos) != 1 {
		return usageErr(errw, "usage: doscapsule run <game-id> [--root DIR]")
	}
	id := pos[0]
	p, err := deps.App.StartGame(*root, id)
	if err != nil {
		fmt.Fprintln(errw, "error:", err)
		return exitErr
	}
	fmt.Fprintf(errw, "Launching %s… (Ctrl+C to stop)\n", p.Title)

	// Foreground model: this process is DOSBox's parent, so Ctrl+C stops the
	// game and closing the terminal takes it down too. No daemon, no orphaned
	// running state.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, terminationSignals()...)
	defer signal.Stop(sig)

	wait := make(chan error, 1)
	go func() { wait <- p.Wait() }()

	select {
	case err := <-wait:
		return exitFromWait(err, errw)
	case <-sig:
		_ = p.Kill()
		<-wait
		fmt.Fprintln(errw, "Stopped.")
		return exitInterrupted
	}
}

// exitFromWait turns DOSBox's exit status into the CLI's exit code.
func exitFromWait(err error, errw io.Writer) int {
	if err == nil {
		return exitOK
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if code := exit.ExitCode(); code > 0 {
			return code
		}
		return exitErr
	}
	fmt.Fprintln(errw, "error:", err)
	return exitErr
}

func loadGames(deps Deps, root string, opts app.ListOptions, errw io.Writer) ([]app.LocalGame, int) {
	if root == "" {
		root = deps.App.Root()
	}
	if root == "" {
		fmt.Fprintln(errw, "error: no games folder selected; run `doscapsule scan <dir>` first")
		return nil, exitErr
	}
	games, err := deps.App.ListIn(root, opts)
	if err != nil {
		fmt.Fprintln(errw, "error:", err)
		return nil, exitErr
	}
	return games, exitOK
}

func emitGames(games []app.LocalGame, asJSON bool, out, errw io.Writer) int {
	if asJSON {
		encodeJSON(out, games)
		return exitOK
	}
	if len(games) == 0 {
		fmt.Fprintln(errw, "No games match.")
		return exitOK
	}
	tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tTITLE\tYEAR\tGENRE")
	for _, g := range games {
		year := "-"
		if g.Year != nil {
			year = strconv.Itoa(*g.Year)
		}
		genre := "-"
		if g.Genre != nil && *g.Genre != "" {
			genre = *g.Genre
		}
		title := g.Title
		if g.Favorited {
			title = "★ " + title
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", g.ID, title, year, genre)
	}
	_ = tw.Flush()
	return exitOK
}

func encodeJSON(w io.Writer, v any) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func validSort(s string) bool {
	switch s {
	case "", "title", "year", "genre", "recent":
		return true
	}
	return false
}

// splitFlags separates flags from positional arguments so flags may appear
// after a positional one (`scan <dir> --json`), which the stdlib flag parser
// stops at by itself. flags that take a value are declared in takesValue.
func splitFlags(args []string, takesValue map[string]bool) (flags, pos []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			return append(flags, "--"), append(pos, args[i+1:]...)
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			pos = append(pos, a)
			continue
		}
		flags = append(flags, a)
		name := strings.TrimLeft(a, "-")
		if strings.ContainsRune(name, '=') {
			continue
		}
		if takesValue[name] && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return flags, pos
}

func newFlagSet(name, usage string, errw io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(errw)
	fs.Usage = func() { fmt.Fprintln(errw, usage) }
	return fs
}

func flagExit(err error) int {
	if errors.Is(err, flag.ErrHelp) {
		return exitOK
	}
	return exitUsage
}

func usageErr(w io.Writer, msg string) int {
	fmt.Fprintln(w, "error:", msg)
	return exitUsage
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `DOS Capsule — local DOS game launcher

Usage:
  doscapsule                         start the UI and open the browser
  doscapsule serve [--port N] [--no-open]
  doscapsule scan <games-dir>
  doscapsule list [--root DIR] [--json] [--sort title|year|genre|recent]
                  [--genre G] [--favorites]
  doscapsule search <query> [--root DIR] [--json]
  doscapsule run <game-id> [--root DIR]
  doscapsule version
  doscapsule help

run blocks until the game exits; Ctrl+C stops it. scan remembers the
folder; list/search read it (or --root) without changing it.
list --json is the machine-readable form for scripts and agents.
`)
}
