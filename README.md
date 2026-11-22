# DOS Capsule

一个用 Go 写的 DOS 游戏启动器。

市面上的 DOS 游戏前端总有些地方不合心意——有的要联网、有的要装一堆东西、
有的把你的游戏库当成它自己的资源库。与其将就，不如自己写一个：把你的 DOS 游戏
目录扫成一面墙，封面、年份、类型都在，点一下就能玩。

![DOS Capsule](docs/images/screenshot.jpg)

## 安装

DOSBox Staging 是必需的模拟器。**下面的包管理器安装会自动带上它**，无需手动装。

**macOS（Homebrew）**
```bash
brew tap bytecapsule/doscapsule https://github.com/bytecapsule/doscapsule
brew trust bytecapsule/doscapsule
brew install bytecapsule/doscapsule/doscapsule
```

**Windows 10/11（Scoop）**
```powershell
scoop bucket add bytecapsule https://github.com/bytecapsule/doscapsule
scoop install doscapsule
```

**其它方式**：从 [Releases](https://github.com/bytecapsule/doscapsule/releases)
下载对应平台的 zip，自行安装 `dosbox-staging`
（macOS `brew install dosbox-staging`；Windows `scoop install dosbox-staging`；
或从 <https://dosbox-staging.github.io/> 下载）。

## 开始使用

**三步：装好 → 运行 → 选一个装着 DOS 游戏的文件夹。**

```bash
doscapsule              # 自动开浏览器
```

然后点顶栏的 **扫描游戏文件夹**，选中你的 DOS 游戏目录即可。
扫描结果、收藏和游玩记录会自动保存，下次打开直接就在。

## 游戏目录要准备什么

启动器不会动你的游戏文件，但每个游戏目录里需要几个**说明文件**，
告诉它这游戏叫什么、怎么启动：

```
你的游戏目录/
└── DUKE3D/
    ├── @launcher.json      ← 必须有，至少写一个 title
    ├── play.bat            ← 必须有，一行主程序名
    ├── @dosbox.conf        ← 可选，需要特殊配置时才写
    ├── @cover.jpg          ← 可选，封面
    └── DUKE3D.EXE …        ← 游戏本体，原样不动
```

### 这两个是必填的文件，要你自己新建

**第一个文件：`@launcher.json`** —— 游戏叫什么、哪一年、什么类型

在游戏目录里新建一个文本文件，命名为 `@launcher.json`（注意前面的 `@`），内容：

```json
{ "title": "Duke Nukem 3D", "year": 1996, "genre": "FPS" }
```

**第二个文件：`play.bat`** —— 告诉 DOSBox 要运行哪个程序

同样在游戏目录里新建，命名为 `play.bat`，内容：

```bat
@echo off
duke3d
```

这里的 `duke3d` 就是游戏主程序的**文件名，去掉 `.exe`**。
比如主程序叫 `PRINCE.EXE`，就写 `prince`；叫 `WOLF3D.EXE`，就写 `wolf3d`。

> 新建时注意：记事本可能自动存成 `.json.txt` / `.bat.txt`，请确认扩展名正确。
> macOS 的「文本编辑」默认是 RTF 格式，建议用 VS Code 或干脆让 AI 帮你生成。

**完整规范、字段说明和打包案例见 [docs/packaging.md](docs/packaging.md)。**

> 💡 这篇文档整篇丢给 AI，让它帮你把散乱的游戏目录整理成上面的结构就行。
> 应用内「打包指南」里也有一个 **复制简洁版（给 AI）** 按钮。

## 鼠标被游戏吃掉了怎么办

DOS 游戏一旦捕获鼠标，光标就出不来，这是 DOSBox 的行为，不是卡住了：

| 操作 | 热键 |
|---|---|
| **释放 / 重新捕获鼠标** | **⌘F10**（macOS）· **Ctrl+F10**（Windows/Linux）· 鼠标中键 |
| 全屏切换 | Alt+Enter |
| 退出 DOSBox | ⌘F9 / Ctrl+F9，或直接关掉 DOSBox 窗口 |

关掉 DOSBox 窗口后，卡片上的「运行中」徽标会在几秒内自动消失。

## 功能

- 扫描本地 DOS 游戏目录，自动读出封面、年份、类型、简介
- 搜索、按类型筛选；排序可按标题、年份、类型或「最近游玩」分区展示
- 收藏与最近游玩记录，关掉再开还在
- 点一下卡片就启动，DOSBox 自动调起
- 界面里直接编辑 DOSBox 配置，不用去翻文件
- 运行状态一目了然，随时可以停止
- 中 / 英文界面切换

## 开发

技术栈：Go（后端，前端资源通过 `go:embed` 打进同一个二进制）
+ SolidJS / TypeScript / Vite（界面）。

```bash
make dist      # 构建前端（输出到 cmd/doscapsule/dist）
make build     # 构建当前平台二进制
make test      # 跑测试
make run       # 构建并运行
make zip-arm   # 打 macOS arm64 包
make zip-x64   # 打 macOS x64 包
make help      # 查看全部目标
```

架构、API、发布流程见 [docs/technical.md](docs/technical.md)。

## 许可

[MIT](LICENSE)
