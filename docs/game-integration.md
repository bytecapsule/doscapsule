# Game Integration

**English** · [中文](game-integration-CN.md)

Drop a couple of metadata files into a game folder and the launcher recognises
it.

DOS Capsule does one thing: **scan a folder and list the DOS games in it so you
can launch them.** It never modifies your game files, needs no install, no
network and no database — arrange the folders as described here, click
**Scan game folder**, and you are done.

> This whole document can be handed to an AI to convert a messy games folder
> into the layout below. The in-app **Game Integration** dialog has a
> **Copy AI spec** button with the same content in short form.

---

## 1. Folder layout

One directory per game; the directory name (or relative path) is the game id and
must be unique:

```
games_root/
├── DUKE3D/
│   ├── @launcher.json      ← required (title is the only mandatory field)
│   ├── play.bat            ← required (8.3 filename)
│   ├── @dosbox.conf        ← optional
│   ├── @cover.jpg          ← optional
│   └── DUKE3D.EXE …        ← the game itself, untouched
└── collections/puzzle/MYST/
    └── …
```

Nesting is fine — the id is the path relative to `games_root`
(e.g. `collections/puzzle/MYST`), so two games may share a folder name in
different subtrees.

**Nothing but those four files changes.** Do not move, rename or re-archive the
game files.

---

## 2. `@launcher.json` (required)

`title` is the only mandatory field. Everything else is optional. UTF-8.

```json
{
  "title": "Duke Nukem 3D",
  "sort_title": "duke nukem 3d",
  "platform": "DOS",
  "year": 1996,
  "genre": "FPS",
  "developer": "3D Realms",
  "publisher": "3D Realms",
  "description": "Optional: shown in the detail panel",
  "notes": "Optional: free-form note",
  "launch": {
    "command": "call play.bat",
    "config": "@dosbox.conf"
  },
  "media": {
    "cover": "@cover.jpg"
  }
}
```

The minimum viable file is one line:

```json
{ "title": "Duke Nukem 3D" }
```

`launch.config` and `media` may be omitted entirely — the launcher then picks up
`@dosbox.conf` and `@cover.jpg` from the folder automatically.

| Field | Meaning |
|---|---|
| `title` | **Required.** The name on the card |
| `sort_title` | Used for sorting; falls back to `title` |
| `platform` | Usually `"DOS"` |
| `year` | Number, used by the year sort |
| `genre` | Used by the genre filter, e.g. `FPS`, `RPG` |
| `developer` / `publisher` | Shown in the detail panel |
| `description` / `notes` | Plain text, optional |

---

## 3. `play.bat` (required, 8.3 filename)

The content is `@echo off` followed by the program name (**without** the
extension, as a DOS command):

```bat
@echo off
duke3d
```

**Why 8.3 names**: DOSBox's internal DOS has limited long-filename support, and
short names like `PLAY.BAT` are reliable under every configuration. Same for the
program itself — `DUKE3D`, not `Duke Nukem 3D.exe`.

When a game needs several steps (swap discs, run SETUP first, mount a CD), write
them all in order:

```bat
@echo off
imgmount d .\CD\DUKE3D.iso -t cdrom
setup
duke3d
```

---

## 4. `@dosbox.conf` (optional)

**Only write one when the game needs a special machine type, memory size, sound
card or extra drives.** Without it the launcher generates an equivalent default
config (mount the game folder, `call play.bat`).

The `@` prefix is there so you can spot it among a pile of loose DOS files.

```ini
[sdl]
fullscreen=false

[dosbox]
memsize=16

[autoexec]
@echo off
mount c .
c:
call play.bat
exit
```

**Mount relatively, `mount c .`**: the launcher starts DOSBox with the game
folder as the working directory, so `.` keeps the whole tree relocatable.

### A stale `dosbox.conf` in the folder is harmless

Many repacked releases ship their own `dosbox.conf`. DOSBox would normally load
it from the working directory — but the launcher starts DOSBox with
`--nolocalconf`, so only the built-in defaults and your `@dosbox.conf` are
read. You do not have to find and delete anything.

(`@dosbox.conf` is our own file and is passed explicitly, so it is unaffected.)

---

## 5. `@cover.jpg` (optional)

- Named `@cover.jpg`, matching the `@` prefix of the other metadata files
- **4:3 landscape, 400×300 JPEG**, original box art centre-cropped
- When missing, a built-in generic DOS cover is used and the card still looks
  complete

Normalise existing cover art:

```bash
python3 scripts/normalize_cover.py path/to/image
python3 scripts/normalize_cover.py --dir games_root     # batch
```

---

## 6. Scan and detection rules

- **Recursive** scan of the chosen folder; hidden directories (`.git`, `.app`)
  are skipped and symlinks are not followed
- A folder is a game when either is true:
  1. it contains `@launcher.json`, or
  2. it has no manifest but does contain `@dosbox.conf` **or** a launchable file
     (`play.bat`, `run.bat`, any `.bat` / `.exe` / `.com`)
- Favourites, play history and the chosen folder live in the app's own
  `state.json` — **nothing is written into the game folders**
- The scan result is restored on the next start; no rescan needed

---

## 7. Worked example: a CD-based 3D game

```
games_root/
└── DUKE3D/
    ├── @launcher.json
    ├── play.bat
    ├── @dosbox.conf
    ├── @cover.jpg
    ├── DUKE3D.EXE
    ├── DUKE3D.GRP
    └── CD/
        └── DUKE3D.iso
```

`@launcher.json`:

```json
{
  "title": "Duke Nukem 3D",
  "year": 1996,
  "genre": "FPS",
  "developer": "3D Realms",
  "publisher": "3D Realms",
  "description": "The 1996 classic first-person shooter."
}
```

`play.bat`:

```bat
@echo off
imgmount d .\CD\DUKE3D.iso -t cdrom
duke3d
```

`@dosbox.conf`:

```ini
[sdl]
fullscreen=false

[dosbox]
memsize=32

[cpu]
cpu_cycles_protected=60000

[autoexec]
@echo off
mount c .
c:
call play.bat
exit
```

## 8. Worked example: a bare real-mode program

```
games_root/
└── PRINCE/
    ├── @launcher.json
    ├── play.bat
    ├── PRINCE.EXE
    └── …
```

```json
{ "title": "Prince of Persia", "year": 1989, "genre": "Platformer" }
```

```bat
@echo off
prince
```

No `@dosbox.conf` needed — the defaults run it.

---

## 9. Checklist

- [ ] One directory per game
- [ ] `@launcher.json` contains a `title`
- [ ] `play.bat` launches the program (8.3 filenames)
- [ ] (optional) `@dosbox.conf` mounts relatively: `mount c .`
- [ ] (optional) `@cover.jpg` is a 4:3 400×300 JPEG

---

## 10. Performance tuning (only if something is off)

The launcher ships sensible defaults; change these only when a game misbehaves:

| Symptom | Adjustment |
|---|---|
| Protected-mode 3D game runs too fast | Set `[cpu] cpu_cycles_protected` from `max` to a fixed value (e.g. `60000`) for a more period-accurate feel |
| 16-bit real-mode game is too slow | `[cpu] cpu_cycles` defaults to `3000` (about 3 MHz); try `20000`–`40000` |
| Old game misdetects memory at POST | `[dosbox] memsize` defaults to `30`; set it to `16` |
| Low frame rate / want max FPS | `[sdl] output = texture` (no CRT shader); `[render] shader = sharp` is the lightest shader |
| Crackling or stuttering audio | Pin `cpu_cycles_protected` to a fixed value such as `60000` |
| Slow loading | Large GRP/data files are limited by emulated I/O; raising cycles helps most |
