# Technical Notes

For anyone changing code or cutting a release. If you only want to play your
games, read [game-integration.md](game-integration.md) instead.

---

## 1. Stack

| Layer | Tech |
|---|---|
| Backend | Go 1.27+, a single `package main` |
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
├── cmd/doscapsule/        Go sources + go:embed resources
│   ├── main.go            embed, HTTP server, /api/* routes, browser launch
│   ├── launcher.go        DOSBox resolution, defaults.conf, exec
│   ├── catalog.go         scanning, LocalGame, manifest I/O
│   ├── state.go           state.json, portable vs system config dir
│   ├── util.go            resolveGameDir (path traversal guard) etc.
│   ├── dist/              frontend build output (gitignored, embed source)
│   └── resources/         fallback-cover.jpg
├── frontend/              SolidJS + Vite
├── Formula/               Homebrew formula
├── bucket/                Scoop manifest
├── scripts/               zip.sh, update-packaging.sh, normalize_cover.py
├── docs/                  this file + the integration docs
├── .github/workflows/     build.yml
├── out/                   local build output: binaries + zips (gitignored)
└── Makefile
```

Standard Go `cmd/` layout. The sources stay a **single package** (~1,200 lines):
splitting into several internal packages at this size buys little and adds
export noise and import cycles.

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

## 5. Build

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
make clean     # remove build/out + dist
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

## 6. Release

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

### Updating the package manifests after a release — do not skip this

Homebrew and Scoop pin an **exact version and checksum**, so a new GitHub
release changes nothing for package-manager users until the manifests are
bumped too. A stale manifest makes `brew upgrade` answer "already installed and
up-to-date" for a release that exists.

```bash
make manifest          # points Formula/ and bucket/ at the latest release
git add Formula bucket && git commit -m "packaging: ..." && git push
```

`make manifest-check` compares the manifest version against the latest release
and exits non-zero when they differ, so a forgotten bump is detectable.

Commit that bump with **`[skip ci]` in the message**. The manifests live in this
repository, so the bump is a commit on `main`; if a push trigger is ever enabled,
that commit would start a build which bumps the manifests again — a loop that
also burns Actions minutes. Two independent guards cover it:

1. `[skip ci]` in the bump commit, and
2. a `paths-ignore: [Formula/**, bucket/**]` filter in the push trigger (already
   written into the commented block in `build.yml`)

The only way this bites is if the manifests move out of this repo into a
separate tap/bucket repository, which is the conventional fix — worth it only
once releases become automatic.

The underlying script is `scripts/update-packaging.sh <tag>`: it downloads the
three zips, computes SHA-256 and rewrites the tag and hashes. It rewrites the
values in place, so it can be run for every release.

**On the user's side** the tap has to be refreshed before upgrading — `brew
upgrade` alone reads the tap as it was last fetched:

```bash
brew update && brew upgrade doscapsule
```

### Why there is no separate tap/bucket repository

`Formula/` at the repo root makes this repository a Homebrew tap, and `bucket/`
makes it a Scoop bucket, so no second repository is needed:

```bash
brew tap bytecapsule/doscapsule https://github.com/bytecapsule/doscapsule
brew trust bytecapsule/doscapsule
brew install bytecapsule/doscapsule/doscapsule
```

```powershell
scoop bucket add bytecapsule https://github.com/bytecapsule/doscapsule
scoop install doscapsule
```

Both declare `dosbox-staging` as a hard dependency, so installing pulls DOSBox
in automatically.

---

## 7. Security and contributions

### Runtime security
The UI is served from `127.0.0.1`, but that only limits the network entry — a
malicious web page can still reach the loopback address. So every request is
wrapped in two layers (see `main()`):

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
