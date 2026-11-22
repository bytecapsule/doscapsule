# 游戏接入规范

往游戏目录里放几个说明文件，启动器就能认出它。

DOS Capsule 只做一件事：**扫描一个目录，把里面的游戏列出来并启动**。
它不修改你的游戏文件，也不需要安装、不联网、不建库——你按下面的约定整理目录，
点"扫描游戏文件夹"就能用。

> 这篇文档可以直接整篇丢给 AI，让它帮你把散乱的游戏目录整理成下面的结构。
> 应用内的"游戏接入规范"对话框里还有一个**复制简洁版（给 AI）**按钮，内容与此一致。

---

## 1. 目录结构

每个游戏一个目录，目录名（或相对路径）就是游戏 id，必须唯一：

```
games_root/
├── DUKE3D/
│   ├── @launcher.json      ← 必须（title 必填）
│   ├── play.bat            ← 必须（8.3 文件名）
│   ├── @dosbox.conf        ← 可选
│   ├── @cover.jpg          ← 可选
│   └── DUKE3D.EXE …        ← 游戏本体，原样保留
└── collections/puzzle/MYST/
    └── …
```

嵌套目录完全可以，id 就是相对 `games_root` 的路径（如 `collections/puzzle/MYST`），
所以不同子目录里出现同名文件夹不会冲突。

**除了下面这 4 个文件，其余文件原样不动。** 不要移动、重命名、重新压缩游戏本体。

---

## 2. `@launcher.json`（必须）

唯一必填字段是 `title`，其余都可省略。UTF-8 编码。

```json
{
  "title": "Duke Nukem 3D",
  "sort_title": "duke nukem 3d",
  "platform": "DOS",
  "year": 1996,
  "genre": "FPS",
  "developer": "3D Realms",
  "publisher": "3D Realms",
  "description": "可选：简介，会显示在详情面板里",
  "notes": "可选：备注",
  "launch": {
    "command": "call play.bat",
    "config": "@dosbox.conf"
  },
  "media": {
    "cover": "@cover.jpg"
  }
}
```

最小可用版本只有一行：

```json
{ "title": "Duke Nukem 3D" }
```

`launch.config` 和 `media` 可以完全省略——省略时启动器会自动探测目录里的
`@dosbox.conf` 和 `@cover.jpg`。

| 字段 | 说明 |
|---|---|
| `title` | **必填**，卡片上显示的名字 |
| `sort_title` | 排序用，省略时用 `title` |
| `platform` | 一般是 `"DOS"` |
| `year` | 数字，用于按年份排序 |
| `genre` | 用于按类型筛选，如 `FPS`、`RPG`、`策略` |
| `developer` / `publisher` | 显示在详情面板 |
| `description` / `notes` | 纯文本，可选 |

---

## 3. `play.bat`（必须，8.3 文件名）

内容就是 `@echo off` 加上主程序名（**不带扩展名**的 DOS 命令）：

```bat
@echo off
duke3d
```

**为什么必须 8.3 文件名**：DOSBox 内部的 DOS 环境对长文件名支持有限，
`PLAY.BAT` 这种短名在所有配置下都可靠。主程序同理（`DUKE3D` 而不是 `Duke Nukem 3D.exe`）。

游戏有多个启动步骤（换盘、先跑 SETUP、挂载 CD）时，按顺序把完整命令写全：

```bat
@echo off
imgmount d .\CD\DUKE3D.iso -t cdrom
setup
duke3d
```

---

## 4. `@dosbox.conf`（可选）

**只在需要特殊机器类型、内存、声卡或额外盘符时才写。**
不写的话，启动器会生成等价的默认配置（挂载游戏目录 + `call play.bat`）。

`@` 前缀是为了让你在一堆零碎 DOS 文件里一眼认出来。

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

**用相对挂载 `mount c .`**：启动器以游戏目录为工作目录启动 DOSBox，
所以写 `.` 而不是绝对路径，整个目录树搬到哪里都能用。

### ⚠️ 接入前务必删掉残留的 `dosbox.conf`

很多现成的游戏整合包里自带一个 `dosbox.conf`。DOSBox 会**按工作目录自动加载**它，
并覆盖启动器传入的配置——典型症状是 cycles 被锁成 5000、画面卡成幻灯片、
或者声音出不来。接入前请删除，或改名为 `dosbox.conf.bak`。

（注意区分：`@dosbox.conf` 带 `@` 前缀，是我们自己的约定，不会触发这个问题。）

---

## 5. `@cover.jpg`（可选）

- 文件名 `@cover.jpg`，与其它元数据文件保持一致的 `@` 前缀
- 规格：**4:3 横版，400×300 JPEG**，用原版盒图居中裁切
- 缺失时自动使用内置的通用 DOS 封面，卡片依然完整

批量标准化已有封面：

```bash
python3 scripts/normalize_cover.py 图片路径
python3 scripts/normalize_cover.py --dir games_root     # 批量
```

---

## 6. 扫描与识别规则

- **递归扫描**所选目录，跳过隐藏目录（`.git`、`.app` 等），不跟随符号链接
- 识别为游戏的条件（满足其一即可）：
  1. 目录内有 `@launcher.json`
  2. 没有清单，但有 `@dosbox.conf` **或**可启动文件（`play.bat`、`run.bat`、任意 `.bat`/`.exe`/`.com`）
- 收藏、最近游玩、所选目录都记在应用自己的 `state.json` 里，**不写进游戏目录**
- 扫描结果重启后自动恢复，不需要每次重扫

---

## 7. 完整案例：带 CD 的 3D 游戏

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

`@launcher.json`：

```json
{
  "title": "Duke Nukem 3D",
  "year": 1996,
  "genre": "FPS",
  "developer": "3D Realms",
  "publisher": "3D Realms",
  "description": "1996 年的经典第一人称射击游戏。"
}
```

`play.bat`：

```bat
@echo off
imgmount d .\CD\DUKE3D.iso -t cdrom
duke3d
```

`@dosbox.conf`：

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

## 8. 完整案例：只有实模式主程序的老游戏

```
games_root/
└── PRINCE/
    ├── @launcher.json
    ├── play.bat
    ├── PRINCE.EXE
    └── …
```

```json
{ "title": "Prince of Persia", "year": 1989, "genre": "平台动作" }
```

```bat
@echo off
prince
```

不需要 `@dosbox.conf`——默认配置就能跑。

---

## 9. 快速检查清单

- [ ] 每个游戏一个独立目录
- [ ] `@launcher.json` 里有 `title`
- [ ] `play.bat` 能启动主程序（8.3 文件名）
- [ ] 删掉了残留的 `dosbox.conf`
- [ ] （可选）`@dosbox.conf` 用相对挂载 `mount c .`
- [ ] （可选）`@cover.jpg` 是 4:3 400×300 JPEG

---

## 10. 性能微调（按需）

启动器已内置一套合理的默认值，只有遇到问题时才需要改：

| 症状 | 调整 |
|---|---|
| 保护模式 3D 游戏节奏过快 | `[cpu] cpu_cycles_protected` 从 `max` 改成固定值（如 `60000`），更接近当年手感 |
| 16 位实模式游戏太慢 | `[cpu] cpu_cycles` 默认 `3000`（约 3 MHz），可试 `20000`–`40000` |
| 老游戏自检误判内存 | `[dosbox] memsize` 默认 `30`，改成 `16` |
| 掉帧 / 想最高帧率 | `[sdl] output = texture`（不用 CRT 着色器）；`[render] shader = sharp` 最轻 |
| 音频爆音、断续 | 把 `cpu_cycles_protected` 固定成 `60000` 之类的值 |
| 加载慢 | 大 GRP/数据文件受模拟 I/O 限制，提高 cycles 最有效 |
