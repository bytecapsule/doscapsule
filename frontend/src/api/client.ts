// The single backend client for the UI. Every call is a plain HTTP request
// against the embedded Go process (cmd/doscapsule/main.go), which serves the
// same JSON API. The local folder chooser and file->URL conversion go through
// the Go side too, so the frontend never talks to a native runtime.

export interface LocalGame {
  id: string;
  title: string;
  sort_title: string;
  platform: string;
  developer: string | null;
  publisher: string | null;
  year: number | null;
  genre: string | null;
  rating: number | null;
  description: string | null;
  notes: string | null;
  favorited: boolean;
  last_played: number | null;
  has_thumbnail: boolean;
  installed: boolean;
  cover_path: string | null;
  screenshot_paths: string[];
  manual_path: string | null;
  game_dir: string;
}

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url);
  const text = await res.text();
  if (!res.ok) {
    const err = new Error(text || res.statusText) as Error & { status?: number };
    err.status = res.status;
    throw err;
  }
  return text ? JSON.parse(text) : (undefined as T);
}

async function postJSON<T>(url: string, body?: unknown): Promise<T> {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await res.text();
  if (!res.ok) { throw new Error(text || res.statusText); }
  return text ? JSON.parse(text) : (undefined as T);
}

/** File path -> URL the webview/browser can load (served by the Go process). */
export function convertFileSrc(path: string): string {
  return `/media?path=${encodeURIComponent(path)}`;
}

/** Native OS folder chooser via the backend; null on cancel. */
export async function open(opts?: { title?: string; directory?: boolean }): Promise<string | null> {
  const r = await postJSON<{ path: string | null }>("/api/pick_folder", { title: opts?.title ?? "" });
  return r.path;
}

export async function localSelectRoot(root: string): Promise<number> {
  const r = await postJSON<{ count: number }>("/api/scan", { root });
  return r.count;
}

export async function localListGames(
  query?: string, genre?: string, sortBy?: string, favoritesOnly?: boolean,
): Promise<LocalGame[]> {
  const q = new URLSearchParams();
  if (query) { q.set("query", query); }
  if (genre) { q.set("genre", genre); }
  if (sortBy) { q.set("sortBy", sortBy); }
  if (favoritesOnly) { q.set("favoritesOnly", "1"); }
  return getJSON<LocalGame[]>(`/api/games?${q.toString()}`);
}

export async function localGenres(): Promise<string[]> {
  return (await getJSON<string[]>("/api/genres")) ?? [];
}

export async function localRememberedRoot(): Promise<string | null> {
  return getJSON<string | null>("/api/remembered_root");
}

export async function localLaunchGame(id: string): Promise<string> {
  const r = await postJSON<{ message: string }>("/api/launch", { id });
  return r.message;
}

export async function localClearRoot(): Promise<void> {
  await postJSON("/api/clear");
}

export async function localToggleFavorite(id: string): Promise<boolean> {
  const r = await postJSON<{ favorited: boolean }>("/api/favorite", { id });
  return r.favorited;
}

export async function localRunningGames(): Promise<string[]> {
  return (await getJSON<string[]>("/api/running")) ?? [];
}

export async function localStopGame(id: string): Promise<void> {
  await postJSON("/api/stop", { id });
}

export async function localGetConf(id: string): Promise<string | null> {
  const r = await getJSON<{ content: string | null }>(`/api/conf?id=${encodeURIComponent(id)}`);
  return r.content;
}

export async function localSaveConf(id: string, content: string): Promise<void> {
  await postJSON("/api/conf", { id, content });
}

export async function localOpenGameDir(id: string): Promise<void> {
  await postJSON("/api/open", { id });
}
