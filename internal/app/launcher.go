package app

// DOSBox resolution, per-game conf handling, launch / stop / running tracking.
// A per-game @dosbox.conf is the single source of truth (DOSBox cwd = game
// dir, relative mounts work); otherwise a
// built-in conf is generated into the app config dir.

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

type Launcher struct {
	mu       sync.Mutex
	running  map[string]*exec.Cmd // game id -> emulator process
	launchMu sync.Mutex           // serialises launch attempts so a double-click cannot start two instances
}

// ErrAlreadyRunning is returned when the game's emulator is already running.
var ErrAlreadyRunning = errors.New("game is already running")

func NewLauncher() *Launcher { return &Launcher{running: map[string]*exec.Cmd{}} }

func dosboxBinaryName() string {
	if runtime.GOOS == "windows" {
		return "dosbox-staging.exe"
	}
	return "dosbox-staging"
}

func resolveDosbox() string {
	for _, key := range []string{"DOSCAPSULE_DOSBOX", "DOSGAME_DOSBOX"} {
		if p := os.Getenv(key); p != "" && fileExists(p) {
			return p
		}
	}
	exe, err := os.Executable()
	if err == nil {
		base := filepath.Dir(exe)
		// Portable bundle: binaries/dosbox/ holds the emulator AND everything it
		// reads at runtime (dosbox-staging.conf as the portable marker,
		// soundfonts, shaders, keyboard layouts, code pages, mapping tables).
		// Copying the folder to another machine therefore needs no setup.
		if p := dosboxInDir(filepath.Join(base, "binaries", "dosbox")); p != "" {
			return p
		}
		for _, sub := range []string{"dosbox-bin", filepath.Join("resources", "dosbox-bin")} {
			c := filepath.Join(base, sub, dosboxBinaryName())
			if fileExists(c) {
				return c
			}
		}
		// Dev layout: binaries/dosbox-staging-<target-triple>
		bin, _ := os.ReadDir(filepath.Join(base, "binaries"))
		var cands []string
		for _, e := range bin {
			if strings.HasPrefix(e.Name(), "dosbox-staging") {
				cands = append(cands, filepath.Join(base, "binaries", e.Name()))
			}
		}
		sort.Strings(cands)
		if len(cands) > 0 {
			return cands[0]
		}
	}
	// Also try the working directory (go run / dev checkout).
	wd, _ := os.Getwd()
	for _, sub := range []string{filepath.Join("binaries", dosboxBinaryName()), filepath.Join("dosbox-bin", dosboxBinaryName())} {
		c := filepath.Join(wd, sub)
		if fileExists(c) {
			return c
		}
	}
	if bin, _ := exec.LookPath("dosbox-staging"); bin != "" {
		return bin
	}
	return dosboxBinaryName()
}

// dosboxInDir finds the emulator binary inside a portable bundle directory.
// The '.conf' file that switches DOSBox to its portable layout shares the
// prefix, so it has to be excluded explicitly.
func dosboxInDir(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var cands []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "dosbox-staging") {
			continue
		}
		if strings.HasSuffix(e.Name(), ".conf") {
			continue
		}
		cands = append(cands, filepath.Join(dir, e.Name()))
	}
	sort.Strings(cands)
	if len(cands) > 0 {
		return cands[0]
	}
	return ""
}

func sanitizeID(id string) string {
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && 'Z' >= r) || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func buildConf(autoexec string, fullscreen bool, cycles string) string {
	s := "[sdl]\nfullscreen=" + map[bool]string{true: "true", false: "false"}[fullscreen] + "\n"
	if cycles != "" {
		s += "\n[cpu]\ncycles=" + cycles + "\n"
	}
	s += "\n[autoexec]\n" + autoexec
	return s
}

// Launch spawns DOSBox for the game; cmd is tracked for Running/Stop.
// defaultsConfBody is passed to EVERY launch, before the game's own conf.
// DOSBox merges -conf files in order, so a game that sets these keys itself
// still wins. CPUs: 32-bit protected-mode games get max cycles (the global
// default caps them at 60 MHz, which made Duke3D/Tomb Raider crawl); real-mode
// 16-bit games are unaffected because they use `cpu_cycles` instead.
const defaultsConfBody = `[sdl]
# Present every frame in step with the display. Staging otherwise presents at
# the DOS rate "without vsync" (its own log line), which shows up as tearing -
# a momentary torn/ripped picture that heals itself on the next frame.
vsync = on

[dosbox]
# Comfortable for 1990s DOS games (the stock 16 MB left little headroom).
# 30 MB stays below Staging's >31 MB advisory. Override per game with
# [dosbox] memsize = <value> in its @dosbox.conf.
memsize = 30

[cpu]
# Launcher default: lift the 60 MHz protected-mode cap. Override per game in
# its @dosbox.conf ([cpu] cpu_cycles_protected = <value>).
cpu_cycles_protected = max

[render]
# Pin what 'auto' would otherwise re-decide mid-game. With shader = crt-auto,
# 'auto' re-picks the phosphor profile and the white point whenever the picture
# content changes, and each switch is briefly visible ("the screen tears and
# then fixes itself"). p22/7800K are the values that mode settles on for VGA
# games, so pinning them keeps the look and drops the flicker.
crt_color_profile = p22
color_temperature = 7800

[midi]
# Staging's default device is the host MIDI port, and macOS has no synth
# attached to CoreMIDI - so every General MIDI game (阿猫阿狗, 英雄传说3,
# 仙剑, ...) played its sound effects but no music. FluidSynth renders the
# MIDI in-process using soundfonts/default.sf2 from DOSBox's config dir; when
# that file is missing Staging logs "Error loading SoundFont" and falls back
# to the host port, so this is safe on a machine without a soundfont. A game
# that picks its own device still wins (later -conf overrides this one).
mididevice = fluidsynth
`

func writeDefaultsConf(dir string) (string, error) {
	p := filepath.Join(dir, "defaults.conf")
	if err := os.WriteFile(p, []byte(defaultsConfBody), 0o644); err != nil {
		return "", err
	}
	return p, nil
}

// Start spawns DOSBox for the game and returns a handle to it. The caller
// decides how to wait: the HTTP server calls Track (background), the CLI calls
// Wait (foreground). Registration happens inside launchMu so a second
// concurrent start for the same id sees it as busy.
func (l *Launcher) Start(root, id string) (*Process, error) {
	// One launch at a time: a second click (or a rapid double-click) must not
	// start another emulator for the same game.
	l.launchMu.Lock()
	defer l.launchMu.Unlock()
	l.mu.Lock()
	_, busy := l.running[id]
	l.mu.Unlock()
	if busy {
		return nil, ErrAlreadyRunning
	}
	dir, err := resolveGameDir(root, id)
	if err != nil {
		return nil, err
	}
	m := loadManifest(dir)

	confArg := ""
	cmdDir := root
	if m.Launch != nil && m.Launch.Config != nil {
		p := filepath.Join(dir, *m.Launch.Config)
		if fileExists(p) {
			confArg = p
			cmdDir = dir // relative mounts resolve against the game dir
		}
	}
	if confArg == "" && fileExists(filepath.Join(dir, confName)) {
		confArg = filepath.Join(dir, confName)
		cmdDir = dir
	}
	confDir, err := launchConfDir()
	if err != nil {
		return nil, err
	}
	if confArg == "" {
		// Built-in conf: absolute mount, cwd = games root.
		fullscreen := m.Launch != nil && m.Launch.Fullscreen != nil && *m.Launch.Fullscreen
		cycles := ""
		if m.Launch != nil && m.Launch.Cycles != nil {
			cycles = *m.Launch.Cycles
		}
		command := "call play.bat"
		if m.Launch != nil && m.Launch.Command != nil && *m.Launch.Command != "" {
			command = *m.Launch.Command
		}
		gameDirFwd := filepath.ToSlash(dir)
		autoexec := fmt.Sprintf("@echo off\nmount c \"%s\"\nc:\n%s\nexit\n", gameDirFwd, command)
		confPath := filepath.Join(confDir, "local-"+sanitizeID(id)+".conf")
		if err := os.WriteFile(confPath, []byte(buildConf(autoexec, fullscreen, cycles)), 0o644); err != nil {
			return nil, err
		}
		confArg = confPath
	}

	defaults, err := writeDefaultsConf(confDir)
	if err != nil {
		return nil, err
	}
	dosbox := resolveDosbox()
	// Order matters: defaults first, the game's conf last so it can override.
	//
	// --nolocalconf is load-bearing. DOSBox Staging also loads a "local"
	// dosbox.conf from the working directory (we set that to the game folder),
	// and it loads it BEFORE these -conf files. Our keys would win, but the
	// leftover file's [autoexec] section still gets concatenated in, so a
	// repack that ships its own dosbox.conf can inject mount commands or
	// launch the wrong binary. Refusing the local conf makes a stray file
	// harmless instead of something the user has to hunt down and delete.
	cmd := exec.Command(dosbox, "--nolocalconf", "-conf", defaults, "-conf", confArg)
	cmd.Dir = cmdDir
	// Keep DOSBox's own output next to the state file: when a game renders
	// text wrongly this log says which code page / layout / warnings applied.
	if dir, err := stateDir(); err == nil {
		if f, err := os.OpenFile(filepath.Join(dir, "dosbox.log"),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
			fmt.Fprintf(f, "\n=== launch %s (%s) ===\n", id, confArg)
			cmd.Stdout = f
			cmd.Stderr = f
		}
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start DOSBox %s: %v", dosbox, err)
	}
	title := id
	if m.Title != nil && *m.Title != "" {
		title = *m.Title
	}
	p := &Process{ID: id, Title: title, cmd: cmd, l: l}
	// Register now (inside launchMu) so a concurrent request sees it as busy
	// before it can reach its own check.
	l.mu.Lock()
	l.running[id] = cmd
	l.mu.Unlock()
	return p, nil
}

// Process is a running emulator. It carries just enough for an entry point to
// wait for it (CLI foreground) or leave it running (HTTP background).
type Process struct {
	ID    string
	Title string
	cmd   *exec.Cmd
	l     *Launcher
}

func (p *Process) Pid() int {
	if p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

func (p *Process) Kill() error {
	if p.cmd.Process == nil {
		return errors.New("not running")
	}
	return p.cmd.Process.Kill()
}

// Wait blocks until the emulator exits, then forgets it. Use this from a
// foreground entry point; do not combine it with Track.
func (p *Process) Wait() error {
	err := p.cmd.Wait()
	p.l.forget(p.ID, p.cmd)
	return err
}

// Track reaps the child in the background and removes it from the running map
// when it exits. Use this from a long-lived server; do not combine it with Wait.
func (l *Launcher) Track(p *Process) {
	go func() {
		_ = p.cmd.Wait()
		l.forget(p.ID, p.cmd)
	}()
}

func (l *Launcher) forget(id string, cmd *exec.Cmd) {
	l.mu.Lock()
	if cur, ok := l.running[id]; ok && cur == cmd {
		delete(l.running, id)
	}
	l.mu.Unlock()
}

func (l *Launcher) RunningIDs() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := []string{}
	for id := range l.running {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (l *Launcher) Stop(id string) error {
	l.mu.Lock()
	cmd, ok := l.running[id]
	if ok {
		delete(l.running, id)
	}
	l.mu.Unlock()
	if !ok || cmd.Process == nil {
		return fmt.Errorf("not running")
	}
	return cmd.Process.Kill()
}
