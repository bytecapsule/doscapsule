import { createSignal, Show, For, onMount, onCleanup } from "solid-js";
import { open, convertFileSrc } from "../api/client";
import {
  localSelectRoot,
  localListGames,
  localLaunchGame,
  localGenres,
  localRememberedRoot,
  localToggleFavorite,
  localRunningGames,
  localStopGame,
  localGetConf,
  localSaveConf,
  localOpenGameDir,
  type LocalGame,
} from "../api/client";
import { Button } from "../components/Button";
import {
  BookOpen, FolderOpen, Joystick, Keyboard, Languages, Play, Search, Settings, Square, Star,
} from "lucide-solid";
import { toggleLang, tr } from "../i18n";

/** Convert an absolute file path (backend) into a src the webview can load. */
function assetSrc(path: string): string {
  return convertFileSrc(path);
}

// Stable keys for synthetic sections. They must not depend on the active
// language, otherwise switching language would re-group the list.
const UNKNOWN_YEAR_KEY = "__unknown_year__";
const OTHER_GENRE_KEY = "__other_genre__";

/** One grid card. Hovering cycles through the game's screenshots (4:3 stills,
 *  exactly how DOSBox aspect-corrects 320x200); leaving returns to the cover.
 *  Games with no cover fall back to their first screenshot; click launches. */
function LocalCard(props: {
  g: LocalGame;
  launching: boolean;
  running: boolean;
  onLaunch: () => void;
  onFav: () => void;
  onStop: () => void;
  onMenu: (x: number, y: number) => void;
}) {
  const shots = () => props.g.screenshot_paths ?? [];
  // One fixed image per card: the cover, else the first screenshot.
  const src = () =>
    props.g.cover_path
      ? assetSrc(props.g.cover_path)
      : shots().length > 0 ? assetSrc(shots()[0]) : null;
  return (
    <button
      class="game-card"
      data-testid="local-card"
      onClick={props.onLaunch}
      title={props.g.title}
      onContextMenu={(e) => { e.preventDefault(); props.onMenu(e.clientX, e.clientY); }}
    >
      <Show when={src()}>
        <img class="game-card-thumb" src={src()!} alt="" loading="lazy" />
      </Show>
      <span
        class={`fav-star${props.g.favorited ? " on" : ""}`}
        title={props.g.favorited ? tr().favRemove : tr().favAdd}
        onClick={(e) => { e.stopPropagation(); props.onFav(); }}
      >{props.g.favorited ? "★" : "☆"}</span>
      <div class="game-card-body">
        <div class="game-card-title">{props.g.title}</div>
        <div class="game-card-meta">
          <Show when={props.g.year}><span>{props.g.year}</span></Show>
          <Show when={props.g.genre}><span class="genre">{props.g.genre}</span></Show>
          <Show when={props.launching}><span>{tr().launching}</span></Show>
          <Show when={props.running}>
            <span
              class="run-badge"
              title={tr().stopEmu}
              onClick={(e) => { e.stopPropagation(); props.onStop(); }}
            ><Square class="meta-ico" />{tr().running}</span>
          </Show>
        </div>
      </div>
    </button>
  );
}

/** The M2 pure-local frontend: pick a games root, browse the directory-managed
 *  games as cards, launch one through DOSBox. No DB, no torrent. */
export function LocalLibrary() {
  const [root, setRoot] = createSignal<string | null>(null);
  const [games, setGames] = createSignal<LocalGame[]>([]);
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const [launching, setLaunching] = createSignal<string | null>(null);
  const [toast, setToast] = createSignal<string | null>(null);
  const [query, setQuery] = createSignal("");
  const [genre, setGenre] = createSignal("");
  const [sortBy, setSortBy] = createSignal("recent");
  const [genres, setGenres] = createSignal<string[]>([]);
  const [favOnly, setFavOnly] = createSignal(false);
  const [running, setRunning] = createSignal<string[]>([]);
  /** The game whose @dosbox.conf is being edited, or null. */
  const [confFor, setConfFor] = createSignal<{ id: string; title: string } | null>(null);
  const [confText, setConfText] = createSignal("");
  const [confIsNew, setConfIsNew] = createSignal(false);
  /** Right-click context menu target + cursor position, or null. */
  const [menu, setMenu] = createSignal<{ x: number; y: number; g: LocalGame } | null>(null);
  const [showHelp, setShowHelp] = createSignal(false);
  const [showSpec, setShowSpec] = createSignal(false);

  /** Group games into labelled sections for the current sort mode.
   *  - recent / title: a single unlabelled stream.
   *  - year: newest year at the top, oldest at the bottom; within a year the
   *    newest first (the backend already orders games; here we just split).
   *  - genre: one section per genre, alphabetically; within a genre by title.
   *  Backend returns a flat list sorted by (sortBy); for the sectioned modes we
   *  re-split into [{ key, label, games }] so the UI can draw a header per block.
   */
  const grouped = () => {
    const list = games();
    if (sortBy() !== "year" && sortBy() !== "genre") {
      return [{ key: "", label: "", games: list }];
    }
    const sections: { key: string; label: string; games: LocalGame[] }[] = [];
    const index = new Map<string, number>();
    for (const g of list) {
      const key = sortBy() === "year"
        ? (g.year != null ? String(g.year) : UNKNOWN_YEAR_KEY)
        : (g.genre || OTHER_GENRE_KEY);
      const label = key === UNKNOWN_YEAR_KEY ? tr().unknownYear
        : key === OTHER_GENRE_KEY ? tr().otherGenre
        : key;
      let i = index.get(key);
      if (i === undefined) {
        i = sections.length;
        index.set(key, i);
        sections.push({ key, label, games: [] });
      }
      sections[i].games.push(g);
    }
    // year: newest first, undated last; genre: alphabetical, "other" last.
    if (sortBy() === "year") {
      sections.sort((a, b) => {
        const ay = a.key === UNKNOWN_YEAR_KEY ? -1 : parseInt(a.key, 10);
        const by = b.key === UNKNOWN_YEAR_KEY ? -1 : parseInt(b.key, 10);
        return by - ay;
      });
    } else {
      sections.sort((a, b) => {
        if (a.key === OTHER_GENRE_KEY) { return 1; }
        if (b.key === OTHER_GENRE_KEY) { return -1; }
        return a.label < b.label ? -1 : a.label > b.label ? 1 : 0;
      });
    }
    return sections;
  };


  // Restore the previous library on startup (auto-reload without re-picking).
  const restoreRoot = async () => {
    const remembered = await localRememberedRoot();
    if (!remembered) return;
    setError(null);
    setLoading(true);
    try {
      await localSelectRoot(remembered);
      setRoot(remembered);
      // Pass the current sort/filter so the initial view respects the active
      // sort (e.g. the "recent" default) instead of silently falling back to title.
      setGames(await localListGames(query() || undefined, genre() || undefined, sortBy() || undefined, favOnly()));
      setGenres(await localGenres());
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  };

  onMount(() => {
    void restoreRoot();
    // Keep the Running badges honest; a stopped emulator drops its badge.
    const refreshRunning = () => { void localRunningGames().then(setRunning).catch(() => {}); };
    refreshRunning();
    const t = window.setInterval(refreshRunning, 4000);
    onCleanup(() => window.clearInterval(t));
    const closeMenu = () => setMenu(null);
    window.addEventListener("click", closeMenu);
    window.addEventListener("keydown", closeMenu);
    onCleanup(() => {
      window.removeEventListener("click", closeMenu);
      window.removeEventListener("keydown", closeMenu);
    });
  });

  const copySpec = async () => {
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(tr().specSummary.join("\n"));
      } else {
        const ta = document.createElement("textarea");
        ta.value = tr().specSummary.join("\n");
        ta.style.position = "fixed";
        ta.style.opacity = "0";
        document.body.appendChild(ta);
        ta.select();
        document.execCommand("copy");
        ta.remove();
      }
      flashToast(tr().copied);
    } catch (e) {
      setError(e instanceof Error ? tr().copyFailed(e.message) : tr().copyFailed(String(e)));
    }
  };

  const CONF_TEMPLATE = [
    "[sdl]",
    "fullscreen=false",
    "",
    "[dosbox]",
    "memsize=16",
    "",
    "[autoexec]",
    "@echo off",
    "mount c .",
    "c:",
    "call play.bat",
    "exit",
    "",
  ].join("\n");

  const flashToast = (msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(null), 3000);
  };

  const reload = async () => {
    setLoading(true);
    setError(null);
    try {
      setGames(await localListGames(query() || undefined, genre() || undefined, sortBy() || undefined, favOnly()));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  };

  /** Search keystrokes debounce 200 ms; the backend list is a cached filter,
   *  so this is cheap — the debounce just avoids flicker while typing. */
  let searchTimer: number | undefined;
  const onSearch = (v: string) => {
    setQuery(v);
    if (searchTimer !== undefined) { window.clearTimeout(searchTimer); }
    searchTimer = window.setTimeout(() => void reload(), 200);
  };

  const toggleFav = async (g: LocalGame) => {
    try { await localToggleFavorite(g.id); } finally { void reload(); }
  };

  const stopGame = async (g: LocalGame) => {
    try {
      await localStopGame(g.id);
      setRunning(await localRunningGames());
    } catch (e) { setError(e instanceof Error ? e.message : String(e)); }
  };

  const openFolder = async (g: LocalGame) => {
    try { await localOpenGameDir(g.id); } catch (e) { setError(e instanceof Error ? e.message : String(e)); }
  };

  const openConf = async (g: LocalGame) => {
    setConfFor({ id: g.id, title: g.title });
    try {
      const existing = await localGetConf(g.id);
      setConfIsNew(existing === null);
      setConfText(existing ?? CONF_TEMPLATE);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const saveConf = async () => {
    const c = confFor();
    if (!c) { return; }
    try {
      await localSaveConf(c.id, confText());
      flashToast(tr().confSaved(c.title));
      setConfFor(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  // The single toolbar action: pick a folder, scan it, show the games below.
  // Cancelling the picker changes nothing; a failed scan keeps the current
  // library on screen instead of blanking it.
  const scanFolder = async () => {
    const selected = await open({ title: tr().btnScan, directory: true });
    if (!selected) return;
    setError(null);
    setLoading(true);
    try {
      const count = await localSelectRoot(selected as string);
      setRoot(selected as string);
      setGenres(await localGenres());
      setGames(await localListGames(query() || undefined, genre() || undefined, sortBy() || undefined, favOnly()));
      flashToast(tr().scanned(count));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  };

  const launch = async (g: LocalGame) => {
    // A running / already-launching game must not spawn a second emulator.
    if (launching() === g.id) { return; }
    if (running().includes(g.id)) {
      flashToast(tr().alreadyRunning);
      return;
    }
    setLaunching(g.id);
    setError(null);
    try {
      await localLaunchGame(g.id);
      flashToast(tr().launched(g.title));
    } catch (e) {
      if ((e as { status?: number })?.status === 409) {
        flashToast(tr().alreadyRunning);
        void localRunningGames().then(setRunning).catch(() => {});
      } else {
        setError(e instanceof Error ? e.message : String(e));
      }
    } finally {
      setLaunching(null);
    }
  };

  return (
    <div class="app-shell">
      <header class="app-nav">
        <span class="app-brand">
          <Joystick class="app-brand-ico" />
          DOS <span class="app-brand-dim">Capsule</span>
        </span>
        <span class="nav-status">
          {loading() ? tr().statusScanning
            : root() ? tr().statusCount(games().length)
            : tr().statusNone}
        </span>
        <div class="nav-right">
          <button class="nav-btn" onClick={() => setShowSpec(true)}><BookOpen class="ico" />{tr().btnSpec}</button>
          <button class="nav-btn" title={tr().helpTitle} onClick={() => setShowHelp(true)}><Keyboard class="ico" />{tr().btnHelp}</button>
          <button class="nav-btn nav-btn-strong" onClick={() => void scanFolder()}><FolderOpen class="ico" />{tr().btnScan}</button>
          <a
            class="nav-btn"
            href="https://github.com/bytecapsule/doscapsule"
            target="_blank"
            rel="noopener noreferrer"
            title="GitHub"
            aria-label="GitHub repository"
          ><svg
            class="ico"
            viewBox="0 0 24 24"
            fill="currentColor"
            aria-hidden="true"
          ><path d="M12 .3a12 12 0 0 0-3.8 23.4c.6.1.8-.3.8-.6v-2c-3.3.7-4-1.6-4-1.6-.6-1.4-1.4-1.8-1.4-1.8-1-.7.1-.7.1-.7 1.2.1 1.8 1.2 1.8 1.2 1 1.8 2.8 1.3 3.5 1 .1-.8.4-1.3.7-1.6-2.7-.3-5.5-1.3-5.5-5.9 0-1.3.5-2.4 1.2-3.2-.1-.3-.5-1.6.1-3.2 0 0 1-.3 3.3 1.2a11.5 11.5 0 0 1 6 0c2.3-1.5 3.3-1.2 3.3-1.2.6 1.6.2 2.9.1 3.2.8.8 1.2 1.9 1.2 3.2 0 4.6-2.8 5.6-5.5 5.9.4.4.8 1.1.8 2.2v3.3c0 .3.2.7.8.6A12 12 0 0 0 12 .3Z"/></svg></a>
          <button
            class="nav-btn"
            title={tr().langTitle}
            aria-label={tr().langTitle}
            onClick={toggleLang}
          ><Languages class="ico" />{tr().btnLang}</button>
        </div>
      </header>

      <main class="app-main">
        <div class="app-content">
          <Show when={root()}>
            <div class="toolbar">
              <div class="tb-search-wrap">
                <Search class="tb-search-ico" />
                <input
                  class="tb-search"
                  placeholder={tr().searchPlaceholder}
                  value={query()}
                  onInput={(e) => onSearch(e.currentTarget.value)}
                />
              </div>
              <select
                class="tb-select"
                value={genre()}
                onChange={(e) => { setGenre(e.currentTarget.value); void reload(); }}
              >
                <option value="">{tr().allGenres}</option>
                <For each={genres()}>{(g) => <option value={g}>{g}</option>}</For>
              </select>
              <select
                class="tb-select"
                value={sortBy()}
                onChange={(e) => { setSortBy(e.currentTarget.value); void reload(); }}
              >
                <option value="title">{tr().sortTitle}</option>
                <option value="year">{tr().sortYear}</option>
                <option value="genre">{tr().sortGenre}</option>
                <option value="recent">{tr().sortRecent}</option>
              </select>
              <label class="tb-chip" title={tr().favoritesOnly}>
                <input
                  type="checkbox"
                  checked={favOnly()}
                  onChange={(e) => { setFavOnly(e.currentTarget.checked); void reload(); }}
                />
                <Star class={`chip-ico${favOnly() ? " on" : ""}`} />
                {tr().favoritesOnly}
              </label>
            </div>
          </Show>
        <Show when={!root()}>
          <div class="local-empty">
            <FolderOpen class="empty-ico" />
            <h2>{tr().emptyTitle}</h2>
            <p>{tr().emptyHint}</p>
            <p class="local-spec">{tr().emptySpec}</p>
          </div>
        </Show>

        <Show when={error()}>
          <div class="local-error">{error()}</div>
        </Show>

        <Show when={root()}>
          <div class="game-sections">
            <For each={grouped()}>
              {(sec) => (
                <section class="game-section">
                  <Show when={sec.key !== ""}>
                    <h3 class="section-title">
                      <span class="section-label">{sec.label}</span>
                      <span class="section-count">{sec.games.length}</span>
                    </h3>
                  </Show>
                  <div class="game-grid">
                    <For each={sec.games}>
                      {(g) => (
                        <LocalCard
                          g={g}
                          launching={launching() === g.id}
                          running={running().includes(g.id)}
                          onLaunch={() => void launch(g)}
                          onFav={() => void toggleFav(g)}
                          onStop={() => void stopGame(g)}
                          onMenu={(x, y) => setMenu({ x, y, g })}
                        />
                      )}
                    </For>
                  </div>
                </section>
              )}
            </For>
          </div>
        </Show>
        </div>
      </main>

      <Show when={toast()}>
        <div class="local-toast">{toast()}</div>
      </Show>

      <Show when={menu()}>
        <div
          class="ctx-menu"
          style={{
            left: `${Math.min(menu()!.x, window.innerWidth - 230)}px`,
            top: `${Math.min(menu()!.y, window.innerHeight - 190)}px`,
          }}
        >
          <button onClick={() => { void launch(menu()!.g); setMenu(null); }}><Play class="mi" />{tr().play}</button>
          <button onClick={() => { void openConf(menu()!.g); setMenu(null); }}><Settings class="mi" />{tr().editConf}</button>
          <button onClick={() => { void openFolder(menu()!.g); setMenu(null); }}><FolderOpen class="mi" />{tr().openFolder}</button>
          <button onClick={() => { void toggleFav(menu()!.g); setMenu(null); }}><Star class="mi" />
            {menu()!.g.favorited ? tr().favRemove : tr().favAdd}
          </button>
          <Show when={running().includes(menu()!.g.id)}>
            <button class="danger" onClick={() => { void stopGame(menu()!.g); setMenu(null); }}><Square class="mi" />{tr().stopEmu}</button>
          </Show>
        </div>
      </Show>

      <Show when={showSpec()}>
        <div class="local-modal-backdrop" onClick={() => setShowSpec(false)}>
          <div class="local-modal local-guide" onClick={(e) => e.stopPropagation()}>
            <h3><BookOpen class="h-ico" />{tr().specTitle}</h3>
            <p class="local-spec">{tr().specIntro}</p>
            <div class="guide">
              <For each={tr().guide}>
                {(sec) => (
                  <>
                    <h4>{sec.h}</h4>
                    {sec.p && <For each={sec.p}>{(p) => <p>{p}</p>}</For>}
                    {sec.ul && (
                      <ul>
                        <For each={sec.ul}>{(item) => <li>{item}</li>}</For>
                      </ul>
                    )}
                    {sec.code && <pre>{sec.code}</pre>}
                  </>
                )}
              </For>
            </div>
            <div class="local-modal-actions">
              <Button variant="secondary" onClick={() => void copySpec()}>{tr().copySummary}</Button>
              <Button variant="secondary" onClick={() => setShowSpec(false)}>{tr().close}</Button>
            </div>
          </div>
        </div>
      </Show>

      <Show when={showHelp()}>
        <div class="local-modal-backdrop" onClick={() => setShowHelp(false)}>
          <div class="local-modal local-help" onClick={(e) => e.stopPropagation()}>
            <h3><Keyboard class="h-ico" />{tr().helpTitle}</h3>
            <table class="help-table">
              <tbody>
                <For each={tr().helpRows}>
                  {(row) => (
                    <tr><td>{row[0]}</td><td>{row[1]}</td></tr>
                  )}
                </For>
              </tbody>
            </table>
            <p class="local-spec">{tr().helpNote}</p>
            <div class="local-modal-actions">
              <Button variant="secondary" onClick={() => setShowHelp(false)}>{tr().close}</Button>
            </div>
          </div>
        </div>
      </Show>

      <Show when={confFor()}>
        <div class="local-modal-backdrop" onClick={() => setConfFor(null)}>
          <div class="local-modal" onClick={(e) => e.stopPropagation()}>
            <h3><Settings class="h-ico" />{tr().confTitle(confFor()!.title)}</h3>
            <Show when={confIsNew()}>
              <p class="local-spec">{tr().confNewNote}</p>
            </Show>
            <textarea
              class="local-conf-editor"
              spellcheck={false}
              rows={18}
              value={confText()}
              onInput={(e) => setConfText(e.currentTarget.value)}
            />
            <div class="local-modal-actions">
              <Button variant="secondary" onClick={() => setConfFor(null)}>{tr().cancel}</Button>
              <Button variant="primary" onClick={() => void saveConf()}>{tr().save}</Button>
            </div>
          </div>
        </div>
      </Show>
    </div>
  );
}