# Technical Notes

For anyone changing code or cutting a release. If you only want to play your
games, read [game-integration.md](game-integration.md) instead.

---

## 1. Stack

| Layer | Tech |
|---|---|
| Backend | Go 1.27+, layered packages: `internal/app` (core), `internal/httpapi`, `internal/cli` |
| Frontend | SolidJS + TypeScript + Vite |
| Delivery | The frontend build is embedded into the same binary via `//go:embed` |
| Package manager | pnpm |
| Emulator | DOSBox Staging (**not** bundled in the zip; installed as a package dependency) |

No database, no network calls, no cgo. At runtime it only listens on
`127.0.0.1`.

---

## 2. Layout

```
doscapsule/
├── cmd/doscapsule/        composition root
│   ├── main.go            go:embed + wiring only
│   ├── dist/              frontend build output (gitignored, embed source)
│   └── resources/         fallback-cover.jpg
├── internal/
│   ├── app/               core: catalog, launcher, state, path guards
│   ├── httpapi/           /api/*, embedded UI, Host/CSRF wrapper
│   └── cli/               scan / list / search / run / serve
├── frontend/              SolidJS + Vite
├── scripts/               zip.sh, normalize_cover.py
├── docs/                  this file + the integration docs
├── .github/workflows/     build.yml
├── out/                   local build output: binaries + zips (gitignored)
└── Makefile
```

`cmd/doscapsule` is only the composition root. The core is transport
independent: `internal/app` owns scanning, launching, state and filtering, with
no HTTP or CLI imports. `internal/httpapi` and `internal/cli` are thin
transports over the same `App` methods, so adding an entry point (a second UI,
an MCP adapter, a batch script) does not mean re-implementing `scan` or
`launch`.

---

## 3. Design decisions

### `go:embed` single-file delivery
`vite build` writes straight into `cmd/doscapsule/dist`, and Go embeds it with
`//go:embed all:dist`. One executable, no resource directory to ship.

### Portable mode
When a `binaries/dosbox/` folder sits next to the executable (i.e. a bundled
DOSBox), state goes into a sibling `data/` and the whole folder can be copied
around. Otherwise the system config dir is used:

| Platform | Path |
|---|---|
| macOS | `~/Library/Application Support/doscapsule` |
| Linux | `~/.config/doscapsule` |
| Windows | `%AppData%\doscapsule` |

### Config merge order
The launcher passes its generated defaults first, then the game's own config:

```
dosbox-staging --nolocalconf -conf <data>/launch/defaults.conf -conf <game>/@dosbox.conf
```

DOSBox applies the files in order, so the game config always wins while the
defaults fill in whatever it leaves out.

**`--nolocalconf` is load-bearing.** DOSBox Staging also loads a "local"
`dosbox.conf` from the working directory — which is the game folder — and it
loads it *before* the `-conf` files. Our keys would still win, but the leftover
file's `[autoexec]` section gets concatenated in, so a repack shipping its own
`dosbox.conf` could inject mount commands or launch the wrong binary. Refusing
it makes a stray file harmless instead of something the user must hunt down.

### DOSBox resolution order
```
1. DOSCAPSULE_DOSBOX env var (legacy DOSGAME_DOSBOX still honoured)
2. binaries/dosbox/ next to the executable
3. dosbox-bin/ or resources/dosbox-bin/ next to the executable
4. binaries/dosbox-staging-* next to the executable (dev layout)
5. dosbox-staging on PATH
```
A package-manager install lands on step 5, so end users configure nothing.

### MIDI
macOS CoreMIDI has no built-in synthesiser, so the defaults set
`mididevice = fluidsynth` with `soundfonts/default.sf2` (GeneralUser GS);
without it, MIDI soundtracks are silent on a Mac.

### Path safety
A game id is a path relative to `games_root`. Every id coming from the frontend
goes through `resolveGameDir()`, which validates the prefix and rejects `../`
escapes.

---

## 4. HTTP API

| Route | Purpose |
|---|---|
| `POST /api/scan` | Scan a directory |
| `GET /api/games` | Game list |
| `GET /api/genres` | Genre list |
| `GET /api/remembered_root` | Last library folder |
| `POST /api/launch` / `POST /api/stop` | Launch / stop |
| `GET /api/running` | Running games |
| `POST /api/favorite` | Toggle favourite |
| `GET`/`POST /api/conf` | Read / write `@dosbox.conf` |
| `POST /api/open` | Open the game folder |
| `POST /api/pick_folder` | System folder picker |
| `GET /media?path=` | Covers / screenshots |

The frontend calls these through the HTTP client in `frontend/src/api/client.ts`.

---

## 5. Entry points

One core (`internal/app`), two transports:

| Entry point | Package | Lifetime |
|---|---|---|
| Browser UI | `internal/httpapi` + embedded SolidJS | long-running server |
| CLI | `internal/cli` | one process per command |

### CLI

```bash
doscapsule                          # serve + open the browser (unchanged)
doscapsule serve [--port N] [--no-open]
doscapsule scan <games-dir>
doscapsule list [--root DIR] [--json] [--sort title|year|genre|recent] [--genre G] [--favorites]
doscapsule search <query> [--root DIR] [--json]
doscapsule run <game-id> [--root DIR]
doscapsule version
```

- `scan` is the only command that changes the remembered games folder.
- `list` / `search` are read-only: they scan `--root` or the remembered folder
  and print a table; `--json` prints the same `LocalGame` objects as the API.
- `run` is foreground: the CLI is DOSBox's parent, blocks until the game exits,
  and Ctrl+C stops it (exit 130). Closing the terminal takes the game down too.
  There is no daemon and no persisted running state.
- Output contract: stdout is data, stderr is diagnostics, exit codes are 0
  (ok), 1 (runtime error), 2 (usage), 130 (interrupted).
- Legacy `-port`, `-open-browser` and `-version` flags still work.

`run` does not stop games launched by the UI, and the UI does not show games
launched by `run`: the foreground process owns that lifetime.

---

## 6. Build

The Makefile is the single entry point and works on every platform:

```bash
make dist      # frontend only, into cmd/doscapsule/dist
make build     # binary for the current platform (-> out/)
make build-arm # cross-compile macOS arm64 (-> out/)
make build-x64 # cross-compile macOS x64 (-> out/)
make test      # tests
make run       # build and run
make zip-arm   # package for arm64 (-> out/*.zip)
make zip-x64   # package for x64 (-> out/*.zip)
make clean     # remove out/ + dist
make help      # list targets
```

All locally-built binaries and zips land in `out/` (gitignored), keeping the
repo root clean. By hand:

```bash
cd frontend && pnpm install && pnpm build && cd ..
mkdir -p out && go build -o out/doscapsule ./cmd/doscapsule
```

Requires Go 1.27+, Node.js 22+, pnpm 12+.

---

## 7. Release

### Pipeline
`.github/workflows/build.yml` is the only workflow and is **dispatched
manually** (day-to-day verification is `make`).

- Always: build the frontend, `go test`, cross-compile three targets, zip them,
  upload artifacts
- With the `release` input ticked: also publish those zips as a GitHub Release

Everything runs on `ubuntu-latest` — Go cross-compiles, and nothing needs cgo or
code signing, so macOS runners are unnecessary.

### Version
`<UTC date>-<serial>`, e.g. `20261122-1`, `20261122-2`. The serial is derived
from the releases that already exist for that date, so two releases in one day
do not collide and it resets daily.

### Zip layout
**Flat**: the binary sits at the archive root with no wrapper directory. This is
what makes Homebrew's `bin.install "doscapsule"` and Scoop's
`"bin": "doscapsule.exe"` work — both resolve relative to the extraction root.

Release assets are named by platform:

- `doscapsule-macos-arm64-<tag>.zip`
- `doscapsule-macos-x64-<tag>.zip`
- `doscapsule-windows-x64-<tag>.zip`

### Package index

Homebrew and Scoop pin an **exact version and checksum**, so a new GitHub
release changes nothing for package-manager users until the index is updated.
The index lives in a separate repository, `bytecapsule/homebrew-packages`:

- `Formula/doscapsule.rb` — Homebrew formula
- `bucket/doscapsule.json` — Scoop manifest
- `products/doscapsule.env` — product repository + asset names
- `scripts/update-package.sh` — downloads the release zips, computes SHA-256
- `.github/workflows/update-package.yml` — updates one product on dispatch

The release workflow mints a short-lived installation token for a GitHub App
(`actions/create-github-app-token`, Contents: write on the packages repo) and
sends a `repository_dispatch` after publishing:

```bash
gh api --method POST repos/bytecapsule/homebrew-packages/dispatches \
  -f event_type=release-published \
  -f 'client_payload[product]=doscapsule' \
  -f "client_payload[tag]=${TAG}"
```

The packages workflow then commits the new URL/hash to its own `main`. No
manual manifest bump, and no binaries are duplicated: the formula and manifest
point at this repository's release assets.

Adding a new tool happens only in the index repository: add
`Formula/<product>.rb`, `bucket/<product>.json`, and `products/<product>.env`,
then dispatch `product=<product>` from that tool's release workflow.

**On the user's side** the tap has to be refreshed before upgrading — `brew
upgrade` alone reads the tap as it was last fetched:

```bash
brew update && brew upgrade doscapsule
```

---

## 8. Security and contributions

### Runtime security
The UI is served from `127.0.0.1`, but that only limits the network entry — a
malicious web page can still reach the loopback address. So every request is
wrapped in two layers (see `httpapi.Serve`):

- **Host check**: only the exact listener address (e.g. `127.0.0.1:8420`, or the
  random port after a fallback) is accepted as `Host`; anything else returns
  `403`. This blocks DNS-rebinding.
- **CSRF**: `http.NewCrossOriginProtection().Handler(mux)` rejects non-safe
  cross-site browser requests, so an external page cannot trigger a launch,
  write a config, toggle favourites or stop a game. Safe `GET`/`HEAD` and
  CLI/curl calls (which send no `Origin`/`Sec-Fetch-Site`) still pass.

Path safety is handled per-route: game ids and `/media` paths are validated
against the games root, rejecting `../` escapes. No session/credentials are
used — the app cannot read arbitrary files, so authentication is intentionally
out of scope.

- Every GitHub Action is **pinned to a commit SHA**, so a re-pointed tag cannot
  inject code
- Least privilege: `contents: read` at the top level, write only on the release
  job
- `persist-credentials: false` on every checkout
- Pull requests do not trigger the pipeline (early-phase policy); it is
  dispatched by hand
- `CODEOWNERS` plus branch protection: merging into `main` needs the owner's
  approval
- Reporting vulnerabilities: [SECURITY.md](../SECURITY.md)
