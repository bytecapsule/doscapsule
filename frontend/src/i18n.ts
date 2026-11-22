// Minimal two-language UI dictionary (zh / en). No framework: one signal plus
// plain lookup tables, persisted in localStorage.

import { createSignal } from "solid-js";

export type Lang = "zh" | "en";

const STORAGE_KEY = "doscapsule-lang";
// Previous key, kept so an existing language choice survives the rename.
const LEGACY_STORAGE_KEY = "dosgame-lang";

function storedLang(): Lang {
  try {
    const v = localStorage.getItem(STORAGE_KEY) ?? localStorage.getItem(LEGACY_STORAGE_KEY);
    if (v === "en" || v === "zh") { return v; }
  } catch { /* private mode */ }
  return "zh";
}

const [lang, setLangSignal] = createSignal<Lang>(storedLang());
export { lang };

export function setLang(l: Lang) {
  setLangSignal(l);
  try { localStorage.setItem(STORAGE_KEY, l); } catch { /* ignore */ }
}

export function toggleLang() {
  setLang(lang() === "zh" ? "en" : "zh");
}

export interface GuideSection {
  h: string;
  p?: string[];
  ul?: string[];
  code?: string;
}

export interface Dict {
  statusScanning: string;
  statusCount: (n: number) => string;
  statusNone: string;
  btnSpec: string;
  btnHelp: string;
  btnScan: string;
  btnLang: string;
  langTitle: string;

  searchPlaceholder: string;
  allGenres: string;
  sortTitle: string;
  sortYear: string;
  sortGenre: string;
  sortRecent: string;
  favoritesOnly: string;

  emptyTitle: string;
  emptyHint: string;
  emptySpec: string;

  launching: string;
  running: string;
  favAdd: string;
  favRemove: string;

  play: string;
  editConf: string;
  openFolder: string;
  stopEmu: string;

  confTitle: (title: string) => string;
  confNewNote: string;
  save: string;
  cancel: string;
  close: string;

  launched: (title: string) => string;
  alreadyRunning: string;
  confSaved: (title: string) => string;
  scanned: (n: number) => string;
  copied: string;
  copyFailed: (msg: string) => string;
  copySummary: string;

  helpTitle: string;
  helpRows: [string, string][];
  helpNote: string;

  specTitle: string;
  specIntro: string;
  guide: GuideSection[];
  specSummary: string[];
}

const ZH: Dict = {
  statusScanning: "正在扫描…",
  statusCount: (n) => `当前扫描到 ${n} 个游戏`,
  statusNone: "尚未扫描游戏文件夹",
  btnSpec: "游戏接入规范",
  btnHelp: "DOSBox 操作指南",
  btnScan: "扫描游戏文件夹",
  btnLang: "中文",
  langTitle: "Switch to English",

  searchPlaceholder: "搜索游戏…",
  allGenres: "全部类型",
  sortTitle: "排序：标题",
  sortYear: "排序：年份",
  sortGenre: "排序：类型",
  sortRecent: "排序：最近游玩",
  favoritesOnly: "仅收藏",

  emptyTitle: "还没有加载游戏",
  emptyHint: "点击顶栏的 扫描游戏文件夹，选择存放游戏的目录。",
  emptySpec: "每个游戏一个目录，包含 @launcher.json、play.bat，可选的 @dosbox.conf 与 @cover.jpg。",

  launching: "启动中…",
  running: "运行中",
  favAdd: "收藏",
  favRemove: "取消收藏",

  play: "启动",
  editConf: "编辑 DOSBox 配置",
  openFolder: "打开游戏文件夹",
  stopEmu: "停止运行",

  confTitle: (title) => `编辑 DOSBox 配置 — ${title}`,
  confNewNote: "还没有 @dosbox.conf —— 保存时会按下面的模板创建一个。",
  save: "保存",
  cancel: "取消",
  close: "关闭",

  launched: (title) => `已启动 ${title}`,
  alreadyRunning: "该游戏已在运行，不能重复启动",
  confSaved: (title) => `DOSBox 配置已保存：${title}`,
  scanned: (n) => `已扫描到 ${n} 个游戏`,
  copied: "已复制到剪贴板，可直接粘贴给 AI",
  copyFailed: (msg) => `复制失败：${msg}`,
  copySummary: "复制简洁版（给 AI）",

  helpTitle: "DOSBox 常用操作",
  helpRows: [
    ["释放 / 捕获鼠标", "⌘F10（macOS）· Ctrl+F10（Win/Linux）· 鼠标中键"],
    ["全屏切换", "Alt+Enter"],
    ["退出 DOSBox", "⌘F9 / Ctrl+F9，或直接关闭 DOSBox 窗口"],
  ],
  helpNote:
    "游戏里点击后鼠标会被 DOSBox 捕获，按上面的热键释放才能操作浏览器等其它窗口。关闭 DOSBox 窗口后，卡片的“运行中”徽标会在几秒内自动消失。",

  specTitle: "游戏接入规范",
  specIntro:
    "一个 DOS 游戏要能被 DOS Capsule 加载，只需按下面的目录约定整理；你的游戏文件不会被修改。",
  guide: [
    {
      h: "1. 目录结构",
      code: `games_root/
  <game-id>/                ← 目录名（或相对路径）即游戏 id，必须唯一
    @launcher.json           ← 元数据清单（必备，title 必填）
    play.bat                 ← 启动脚本（必备，8.3 文件名）
    @dosbox.conf             ← 每游戏 DOSBox 配置（可选）
    @cover.jpg               ← 封面（可选，4:3 400x300 JPEG）
    <游戏本体>               ← .exe / .com / 数据文件`,
    },
    {
      h: "2. 必备：@launcher.json",
      p: ["唯一必填字段是 title；其余可省略。UTF-8 JSON：",
        "launch.config 与 media 均可省略：省略时自动探测目录内的 @dosbox.conf 与 @cover.jpg。"],
      code: `{
  "title": "Duke Nukem 3D",
  "sort_title": "duke nukem 3d",
  "platform": "DOS",
  "year": 1996,
  "genre": "FPS",
  "developer": "3D Realms",
  "publisher": "3D Realms",
  "description": "可选：简介",
  "notes": "可选：备注",
  "launch": {
    "command": "call play.bat",
    "config": "@dosbox.conf"
  },
  "media": { "cover": "@cover.jpg" }
}`,
    },
    {
      h: "3. 必备：play.bat（8.3 文件名）",
      p: ["内容是 @echo off 加主程序名（不带扩展名的 DOS 命令），主程序须为 8.3 文件名：",
        "若游戏有多步启动（换盘、配置程序等），把完整命令按顺序写进 play.bat 即可。"],
      code: `@echo off
duke3d`,
    },
    {
      h: "4. 可选：@dosbox.conf（每游戏配置）",
      p: ["@ 前缀便于在零碎 DOS 文件里识别。存在时它是该游戏启动的唯一配置来源；启动器以游戏目录为工作目录，所以相对挂载在整棵目录树搬移后依然有效：",
        "没有 @dosbox.conf 时，启动器生成内置配置（绝对路径挂载游戏目录 + call play.bat）。需要特殊机器类型、内存或声卡时再建 @dosbox.conf 微调。",
        "注意：如果游戏目录里残留了旧的 dosbox.conf（很多整合包会带），DOSBox 会按工作目录自动加载它并覆盖启动器配置（典型症状是 cycles 被锁成 5000、画面像幻灯片），打包前务必删除或改名。"],
      code: `[sdl]
fullscreen=false

[dosbox]
memsize=16

[autoexec]
@echo off
mount c .
c:
call play.bat
exit`,
    },
    {
      h: "5. 可选：封面（@cover.jpg）",
      ul: [
        "文件名 @cover.jpg，与其它元数据文件的 @ 前缀一致，便于识别。",
        "规格：4:3 横版 400x300 JPEG，用原版盒图或官方图居中裁切。",
        "缺失时：自动使用内置的通用 DOS 封面，卡片依然完整。",
        "标准化工具：python3 scripts/normalize_cover.py 图片路径，批量 python3 scripts/normalize_cover.py --dir games_root。",
      ],
    },
    {
      h: "6. 扫描与识别规则",
      ul: [
        "递归扫描所选目录：跳过隐藏目录（.git、.app 等），不跟随符号链接，不会陷入循环。",
        "识别为游戏：目录内有 @launcher.json；或没有清单但有 @dosbox.conf + 可启动文件（play.bat、run.bat、任意 bat/exe/com）。",
        "游戏 id = 相对 games_root 的路径（如 DUKE3D、collections/puzzle/MyGame），嵌套同名目录不会冲突。",
        "收藏、最近游玩、所选目录记在应用自己的 state.json 中，不写进游戏目录。",
      ],
    },
    {
      h: "7. 快速检查清单",
      ul: [
        "每个游戏一个独立目录",
        "@launcher.json 里有 title",
        "play.bat 能启动主程序（8.3 文件名）",
        "（可选）@dosbox.conf 用相对挂载 mount c .",
        "（可选）@cover.jpg 为 4:3 400x300 JPEG",
      ],
    },
    {
      h: "8. 性能建议（按需微调）",
      ul: [
        "保护模式 3D 游戏：启动器已默认 [cpu] cpu_cycles_protected = max；若节奏过快，改为固定值（如 60000）更接近当年的手感。",
        "16 位实模式游戏：[cpu] cpu_cycles 默认 3000（约 3 MHz），偏慢时可试 20000–40000。",
        "内存：启动器默认 [dosbox] memsize = 30；极少数老游戏对 >16 MB 敏感（自检误判），改为 16 即可。",
        "掉帧/追求最高帧率：[sdl] output = texture（不用 CRT 着色器）比 opengl 更省 GPU；[render] shader = sharp 是最轻的着色器。",
        "音频爆音/断续：把 cpu_cycles_protected 固定成 60000 之类的值，通常能消除 max 下的抖动。",
        "加载慢：大 GRP/数据文件的读取受模拟 I/O 限制，提高 cycles 是最有效的手段。",
      ],
    },
  ],
  specSummary: [
    "【任务】把指定的 DOS 游戏目录改造成 DOS Capsule 可加载的格式。",
    "只允许新增/修改下面列出的说明文件；不要修改、移动或重命名游戏本体文件。",
    "",
    "【目录结构】",
    "<games_root>/<game-id>/",
    "  @launcher.json          —— 必须，UTF-8 JSON，title 为唯一必填字段",
    "  play.bat                —— 必须，8.3 文件名，内容是 @echo off + 主程序命令",
    "  @dosbox.conf            —— 可选，每游戏 DOSBox 配置（存在即唯一启动配置）",
    "  @cover.jpg              —— 可选，4:3 横版 400x300 JPEG（原版盒图居中裁切）",
    "",
    "【@launcher.json 示例】",
    "{",
    '  "title": "游戏名",',
    '  "sort_title": "排序名（可选）",',
    '  "platform": "DOS",',
    '  "year": 1996,',
    '  "genre": "FPS",',
    '  "developer": "开发商（可选）",',
    '  "publisher": "发行商（可选）",',
    '  "description": "简介（可选）",',
    '  "launch": { "command": "call play.bat", "config": "@dosbox.conf" },',
    '  "media": { "cover": "@cover.jpg" }',
    "}",
    "launch.config 与 media 可省略：省略时自动探测 @dosbox.conf 与 @cover.jpg。",
    "",
    "【play.bat 规则】",
    "必须 8.3 文件名（PLAY.BAT）；主程序也必须是 8.3 文件名（不带扩展名）。示例：",
    "@echo off",
    "duke3d",
    "多步启动（换盘、配置程序等）按顺序写全每一行。",
    "",
    "【@dosbox.conf 规则（可选）】",
    "若提供，它是该游戏启动的唯一配置；DOSBox 以游戏目录为工作目录，因此用相对挂载，整棵目录树可搬移：",
    "[sdl]",
    "fullscreen=false",
    "[dosbox]",
    "memsize=16",
    "[autoexec]",
    "@echo off",
    "mount c .",
    "c:",
    "call play.bat",
    "exit",
    "若省略，启动器生成等价内置配置（绝对路径挂载游戏目录 + call play.bat）。",
    "确实需要特殊 machine/memsize/声卡/额外盘符（如 CD 或软驱）时，才写 @dosbox.conf。",
    "",
    "【识别规则】",
    "目录含 @launcher.json 即视为游戏；或含 @dosbox.conf + play.bat/run.bat/任意 bat/exe/com。",
    "扫描递归进行，跳过隐藏目录（.git 等），不跟随符号链接。游戏 id = 相对 games_root 的路径。",
    "打包前检查并删除游戏目录里残留的 dosbox.conf：它会被 DOSBox 按工作目录自动加载，覆盖启动器配置（常见症状是 cycles=5000、卡成幻灯片）。",
    "收藏、最近游玩等由启动器自己保管，不要写入游戏目录。",
    "",
    "【验收标准】",
    "1) 扫描后该游戏出现在卡片网格；2) 点击卡片能通过 DOSBox 启动并进入游戏；",
    "3) @cover.jpg 按 4:3 400x300 正确显示；4) 未改动任何游戏本体文件。",
    "",
    "【性能建议】",
    "1) 保护模式 3D 游戏默认 cpu_cycles_protected = max；过快就改成固定值（如 60000）。",
    "2) 16 位实模式游戏用 cpu_cycles（默认 3000，约 3 MHz）；偏慢可试 20000-40000。",
    "3) 内存默认 memsize=30；个别老游戏对 >16MB 敏感时改成 16。",
    "4) 掉帧可设 output=texture 或 shader=sharp；音频爆音用固定 cycles。",
    "5) 加载慢主要受模拟 I/O 限制，提高 cycles 最有效。",
    "",
    "【输出要求】",
    "逐游戏列出：新建或修改了哪些文件、play.bat 调用的命令、是否需要额外 mount、以及无法确定时向用户提出的问题。",
    "不要猜测游戏目录之外的文件；不要删除原文件。",
  ],
};

const EN: Dict = {
  statusScanning: "Scanning…",
  statusCount: (n) => `${n} game${n === 1 ? "" : "s"} scanned`,
  statusNone: "No games folder scanned",
  btnSpec: "Packaging Guide",
  btnHelp: "DOSBox Guide",
  btnScan: "Scan Games Folder",
  btnLang: "English",
  langTitle: "切换到中文",

  searchPlaceholder: "Search games…",
  allGenres: "All genres",
  sortTitle: "Sort: title",
  sortYear: "Sort: year",
  sortGenre: "Sort: genre",
  sortRecent: "Sort: recently played",
  favoritesOnly: "Favorites only",

  emptyTitle: "No games loaded",
  emptyHint: "Click Scan Games Folder in the top bar and choose a directory.",
  emptySpec: "Each game is one folder with @launcher.json, play.bat, optional @dosbox.conf and @cover.jpg.",

  launching: "launching…",
  running: "Running",
  favAdd: "Add to favorites",
  favRemove: "Remove from favorites",

  play: "Play",
  editConf: "Edit DOSBox Config",
  openFolder: "Open game folder",
  stopEmu: "Stop emulator",

  confTitle: (title) => `Edit DOSBox Config — ${title}`,
  confNewNote: "No @dosbox.conf yet — saving creates one from the template below.",
  save: "Save",
  cancel: "Cancel",
  close: "Close",

  launched: (title) => `Launched ${title}`,
  alreadyRunning: "This game is already running",
  confSaved: (title) => `DOSBox config saved: ${title}`,
  scanned: (n) => `Scanned ${n} game${n === 1 ? "" : "s"}`,
  copied: "Copied to clipboard — paste it into your AI",
  copyFailed: (msg) => `Copy failed: ${msg}`,
  copySummary: "Copy AI-friendly spec",

  helpTitle: "DOSBox Shortcuts",
  helpRows: [
    ["Release / capture mouse", "⌘F10 (macOS) · Ctrl+F10 (Win/Linux) · middle mouse button"],
    ["Toggle fullscreen", "Alt+Enter"],
    ["Quit DOSBox", "⌘F9 / Ctrl+F9, or just close the DOSBox window"],
  ],
  helpNote:
    "Clicking inside a game makes DOSBox capture the mouse; use the shortcut above to release it. Closing the DOSBox window clears the card's Running badge within a few seconds.",

  specTitle: "Game Packaging Guide",
  specIntro:
    "To make a DOS game loadable by DOS Capsule, organise it as described below. Your game files are never modified.",
  guide: [
    {
      h: "1. Directory layout",
      code: `games_root/
  <game-id>/                ← folder name (or relative path) is the game id, must be unique
    @launcher.json           ← metadata manifest (required; title is mandatory)
    play.bat                 ← launch script (required; 8.3 file name)
    @dosbox.conf             ← per-game DOSBox config (optional)
    @cover.jpg               ← cover art (optional; 4:3 400x300 JPEG)
    <game files>             ← .exe / .com / data files`,
    },
    {
      h: "2. Required: @launcher.json",
      p: ["The only required field is title; everything else can be omitted. UTF-8 JSON:",
        "launch.config and media are optional: when omitted, @dosbox.conf and @cover.jpg are detected automatically."],
      code: `{
  "title": "Duke Nukem 3D",
  "sort_title": "duke nukem 3d",
  "platform": "DOS",
  "year": 1996,
  "genre": "FPS",
  "developer": "3D Realms",
  "publisher": "3D Realms",
  "description": "optional",
  "notes": "optional",
  "launch": {
    "command": "call play.bat",
    "config": "@dosbox.conf"
  },
  "media": { "cover": "@cover.jpg" }
}`,
    },
    {
      h: "3. Required: play.bat (8.3 file name)",
      p: ["Content is @echo off followed by the main program name (a DOS command without extension); the executable must be an 8.3 file name:",
        "If the game needs several steps (disk swaps, setup tools), write each command in order into play.bat."],
      code: `@echo off
duke3d`,
    },
    {
      h: "4. Optional: @dosbox.conf (per-game config)",
      p: ["The @ prefix makes metadata files easy to spot among loose DOS files. When present it is the single source of truth for launching; DOSBox runs with the game folder as its working directory, so relative mounts survive moving the whole tree:",
        "Without @dosbox.conf the launcher generates an equivalent built-in config (absolute mount + call play.bat). Create one to tune machine type, memory or sound.",
        "Note: a stale dosbox.conf left in the game folder (common in repacked releases) is auto-loaded from the working directory and overrides the launcher's settings — a typical symptom is cycles pinned to 5000 and a slideshow. Remove or rename it before packaging."],
      code: `[sdl]
fullscreen=false

[dosbox]
memsize=16

[autoexec]
@echo off
mount c .
c:
call play.bat
exit`,
    },
    {
      h: "5. Optional: cover art (@cover.jpg)",
      ul: [
        "File name @cover.jpg, matching the @ prefix of the other metadata files.",
        "Spec: 4:3 landscape, 400x300 JPEG, centre-cropped from the original box art or official image.",
        "When missing, a built-in generic DOS cover is used, so cards always render a picture.",
        "Tool: python3 scripts/normalize_cover.py <image>, or bulk python3 scripts/normalize_cover.py --dir <games_root>.",
      ],
    },
    {
      h: "6. Scan and detection rules",
      ul: [
        "Recursive scan of the chosen folder: hidden folders (.git, .app, …) are skipped and symlinks are never followed, so a link loop cannot wedge it.",
        "A folder is a game when it contains @launcher.json; or has no manifest but contains @dosbox.conf plus a launcher (play.bat, run.bat, any bat/exe/com).",
        "The game id is the path relative to games_root (e.g. DUKE3D, collections/puzzle/MyGame), so nested folders with the same name never collide.",
        "Favorites, recently played and the selected folder live in the app's own state.json — never inside game folders.",
      ],
    },
    {
      h: "7. Quick checklist",
      ul: [
        "One folder per game",
        "@launcher.json exists and has a title",
        "play.bat starts the main program (8.3 file name)",
        "(optional) @dosbox.conf uses a relative mount: mount c .",
        "(optional) @cover.jpg is 4:3 400x300 JPEG",
      ],
    },
    {
      h: "8. Performance tips (optional)",
      ul: [
        "Protected-mode 3D games: the launcher already sets [cpu] cpu_cycles_protected = max; if the game runs too fast, pick a fixed value (e.g. 60000) for a period-appropriate feel.",
        "16-bit real-mode games: [cpu] cpu_cycles defaults to 3000 (≈3 MHz); try 20000–40000 when they feel sluggish.",
        "Memory: the launcher defaults to [dosbox] memsize = 30; a few very old games mis-detect >16 MB, so set 16 for those.",
        "Frame rate: [sdl] output = texture (no CRT shader) is lighter than opengl; [render] shader = sharp is the cheapest shader.",
        "Audio crackle/drop-outs: pin cpu_cycles_protected to a fixed value such as 60000; that usually removes the jitter seen at max.",
        "Slow loading: large GRP/data files are limited by emulated I/O; raising cycles is the effective lever.",
      ],
    },
  ],
  specSummary: [
    "[Task] Convert the given DOS game folder(s) into the DOS Capsule layout.",
    "Only create/modify the metadata files listed below; never modify, move or rename the game's own files.",
    "",
    "[Layout]",
    "<games_root>/<game-id>/",
    "  @launcher.json          -- required, UTF-8 JSON, title is the only mandatory field",
    "  play.bat                -- required, 8.3 file name, content: @echo off + main program command",
    "  @dosbox.conf            -- optional per-game DOSBox config (when present it is the only launch config)",
    "  @cover.jpg              -- optional, 4:3 landscape 400x300 JPEG (centre-cropped box art)",
    "",
    "[@launcher.json example]",
    "{",
    '  "title": "Game name",',
    '  "sort_title": "sort key (optional)",',
    '  "platform": "DOS",',
    '  "year": 1996,',
    '  "genre": "FPS",',
    '  "developer": "developer (optional)",',
    '  "publisher": "publisher (optional)",',
    '  "description": "summary (optional)",',
    '  "launch": { "command": "call play.bat", "config": "@dosbox.conf" },',
    '  "media": { "cover": "@cover.jpg" }',
    "}",
    "launch.config and media are optional; when omitted @dosbox.conf and @cover.jpg are auto-detected.",
    "",
    "[play.bat rules]",
    "Must be an 8.3 file name (PLAY.BAT); the executable must also be 8.3 (no extension in the command). Example:",
    "@echo off",
    "duke3d",
    "For multi-step launches (disk swaps, setup tools) write every command in order.",
    "",
    "[@dosbox.conf rules (optional)]",
    "When provided it is the only launch config; DOSBox runs with the game folder as cwd, so use relative mounts and the whole tree stays portable:",
    "[sdl]",
    "fullscreen=false",
    "[dosbox]",
    "memsize=16",
    "[autoexec]",
    "@echo off",
    "mount c .",
    "c:",
    "call play.bat",
    "exit",
    "Without it the launcher generates an equivalent built-in config (absolute mount + call play.bat).",
    "Only add @dosbox.conf when special machine/memsize/sound/extra drives (CD, floppy) are needed.",
    "",
    "[Detection rules]",
    "A folder is a game when it contains @launcher.json, or contains @dosbox.conf plus play.bat/run.bat/any bat/exe/com.",
    "Before packaging, delete any stale dosbox.conf left in the game folder: DOSBox auto-loads it from the working directory and it overrides the launcher config (typical symptom: cycles=5000, slideshow).",
    "Scanning is recursive, skips hidden folders (.git …) and never follows symlinks. Game id = path relative to games_root.",
    "Favorites/recent state is owned by the launcher; never write it into game folders.",
    "",
    "[Acceptance criteria]",
    "1) The game appears in the card grid after scanning; 2) clicking it launches through DOSBox;",
    "3) @cover.jpg renders as 4:3 400x300; 4) no game files were modified.",
    "",
    "[Performance tips]",
    "1) Protected-mode 3D games default to cpu_cycles_protected = max; use a fixed value (e.g. 60000) if too fast.",
    "2) 16-bit real-mode games use cpu_cycles (default 3000, about 3 MHz); try 20000-40000 when sluggish.",
    "3) Default memsize is 30; switch to 16 for old games that mis-detect >16 MB.",
    "4) Frame drops: output=texture or shader=sharp; audio crackle: pin a fixed cycles value.",
    "5) Slow loading is emulated-I/O bound; raising cycles is the effective lever.",
    "",
    "[Output requirements]",
    "Per game, report: which files you created/modified, what play.bat calls, whether extra mounts are required, and ask the user when something is ambiguous.",
    "Do not guess files outside the game folder; do not delete original files.",
  ],
};

const DICTS: Record<Lang, Dict> = { zh: ZH, en: EN };

/** The dictionary for the active language. */
export function tr(): Dict {
  return DICTS[lang()];
}
