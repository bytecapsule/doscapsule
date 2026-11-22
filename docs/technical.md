# 技术文档

写给要改代码或发版的人。只想放游戏进去玩，看
[game-integration.md](game-integration.md) 就够了。

---

## 1. 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.27+，单一 `package main` |
| 前端 | SolidJS + TypeScript + Vite |
| 体裁 | 前端构建产物通过 `//go:embed` 打进同一个二进制 |
| 包管理 | pnpm |
| 模拟器 | DOSBox Staging（**不打包进 zip**，由包管理器作为依赖安装） |

没有数据库、没有网络请求、没有 cgo。运行时只监听 `127.0.0.1`。

---

## 2. 目录结构

```
doscapsule/
├── cmd/doscapsule/        Go 源码 + go:embed 资源
│   ├── main.go            embed、HTTP server、/api/* 路由、开浏览器
│   ├── launcher.go        DOSBox 路径解析、defaults.conf、exec 启动
│   ├── catalog.go         游戏扫描、LocalGame、manifest 读写
│   ├── state.go           state.json、便携/系统目录解析
│   ├── util.go            resolveGameDir（防路径穿越）等
│   ├── dist/              前端构建产物（gitignore，embed 源）
│   └── resources/         fallback-cover.jpg
├── frontend/              SolidJS + Vite
├── Formula/               Homebrew formula
├── bucket/                Scoop manifest
├── scripts/               zip.sh、update-packaging.sh、normalize_cover.py
├── docs/                  本文档 + game-integration.md
├── .github/workflows/     build.yml
└── Makefile
```

采用 Go 社区标准的 `cmd/` 布局。源码约 1200 行，刻意保持**单一 package**——
拆成多个 internal 包在这个体量上收益小、反而增加循环依赖和导出噪音。

---

## 3. 关键设计决策

### `go:embed` 单文件分发
前端 `vite build` 直接输出到 `cmd/doscapsule/dist`，Go 用
`//go:embed all:dist` 把它编进二进制。对外只有一个可执行文件，没有资源目录要拷。

### 便携模式
可执行文件旁存在 `binaries/dosbox/`（即自带 DOSBox）时，状态写到旁边的 `data/`，
整个文件夹可以拷来拷去。否则回退到系统用户配置目录：

| 平台 | 路径 |
|---|---|
| macOS | `~/Library/Application Support/doscapsule` |
| Linux | `~/.config/doscapsule` |
| Windows | `%AppData%\doscapsule` |

### 配置合并顺序
启动 DOSBox 时先传启动器生成的 `defaults.conf`，再传游戏自己的 `@dosbox.conf`：

```
dosbox-staging -conf <data>/launch/defaults.conf -conf <game>/@dosbox.conf
```

DOSBox 后传的配置胜出，所以游戏配置永远可以覆盖默认值，而默认值补上游戏没写的部分。

### DOSBox 查找顺序
```
1. 环境变量 DOSCAPSULE_DOSBOX（旧名 DOSGAME_DOSBOX 仍兼容）
2. 可执行文件旁 binaries/dosbox/
3. 可执行文件旁 dosbox-bin/、resources/dosbox-bin/
4. 可执行文件旁 binaries/dosbox-staging-*（开发布局）
5. 系统 PATH 中的 dosbox-staging
```
包管理器安装走第 5 条，所以终端用户不需要任何配置。

### MIDI
macOS 的 CoreMIDI 没有内置合成器，所以默认配置写
`mididevice = fluidsynth` + `soundfonts/default.sf2`（GeneralUser GS），
否则老游戏的 MIDI 配乐在 Mac 上是静音。

### 路径安全
游戏 id 是相对 `games_root` 的路径。所有由前端传入的 id 都过
`resolveGameDir()` 做前缀校验，防止 `../` 穿越出库目录。

---

## 4. HTTP API

| 路由 | 作用 |
|---|---|
| `POST /api/scan` | 扫描指定目录 |
| `GET /api/games` | 游戏列表 |
| `GET /api/genres` | 类型列表 |
| `GET /api/remembered_root` | 上次的库目录 |
| `POST /api/launch` / `POST /api/stop` | 启动 / 停止 |
| `GET /api/running` | 运行中的游戏 |
| `POST /api/favorite` | 收藏 |
| `GET`/`POST /api/conf` | 读写 `@dosbox.conf` |
| `POST /api/open` | 打开游戏文件夹 |
| `POST /api/pick_folder` | 系统目录选择器 |
| `GET /media?path=` | 封面 / 截图 |

前端通过 `frontend/src/api/tauri.ts` 里的 fetch shim 调用（文件名是历史遗留）。

---

## 5. 构建

统一入口是 Makefile，跨平台可用：

```bash
make dist      # 只构建前端（输出到 cmd/doscapsule/dist）
make build     # 构建当前平台二进制
make build-arm # 交叉编译 macOS arm64
make build-x64 # 交叉编译 macOS x64
make test      # 跑测试
make run       # 构建并运行
make zip-arm   # 打 arm64 发布包
make zip-x64   # 打 x64 发布包
make clean     # 清理产物
make help      # 查看全部目标
```

手动构建：

```bash
cd frontend && pnpm install && pnpm build && cd ..
go build -o doscapsule ./cmd/doscapsule
```

前提：Go 1.27+、Node.js 22+、pnpm 12+。

---

## 6. 发布

### 流水线
`.github/workflows/build.yml` 是唯一的 workflow，**手动触发**（本地 `make` 才是日常验证）。

- 永远执行：前端构建 → `go test` → 三平台交叉编译 → 打 3 个 zip → 上传 artifact
- 勾选 `release` 时额外：把这些 zip 发布成 GitHub Release

全部跑在 `ubuntu-latest` 上——Go 交叉编译不需要 macOS runner，也没有 cgo 或签名需求。

### 版本号
`<UTC 日期>-<流水号>`，例如 `20261122-1`、`20261122-2`。
流水号由已有的 release 推导，同一天发两次不会撞车，跨日自动重置。

### zip 布局
**扁平结构**：二进制直接在压缩包根目录，不套一层文件夹。
这是 Homebrew 的 `bin.install "doscapsule"` 和 Scoop 的 `"bin": "doscapsule.exe"`
能工作的前提——两者都相对解压根目录解析，套一层就找不到。

### 发布后更新包管理器清单

```bash
bash scripts/update-packaging.sh <tag>
```

脚本会下载三个 zip、算 SHA-256、改写 `Formula/doscapsule.rb` 和
`bucket/doscapsule.json` 里的版本号与哈希。

### 为什么不需要额外的 tap / bucket 仓库

`Formula/` 在仓库根目录，所以本仓库自己就是一个 Homebrew tap；
`bucket/` 同理让它成为一个 Scoop bucket。用户：

```bash
brew tap bytecapsule/doscapsule https://github.com/bytecapsule/doscapsule
brew trust bytecapsule/doscapsule
brew install bytecapsule/doscapsule/doscapsule
```

```powershell
scoop bucket add bytecapsule https://github.com/bytecapsule/doscapsule
scoop install doscapsule
```

两者都在清单里把 `dosbox-staging` 声明为**强制依赖**，安装时自动带上。

---

## 7. 安全与贡献

- 所有 GitHub Actions **钉死到 commit SHA**，防止 tag 被重指向后注入代码
- 工作流默认最小权限：顶层 `contents: read`，只有 release job 有写权限
- 每次 checkout 都 `persist-credentials: false`
- PR 不自动触发流水线（初期策略），改为手动 dispatch
- `CODEOWNERS` + 分支保护：合并进 `main` 需要 owner 审批
- 漏洞上报见 [SECURITY.md](../SECURITY.md)
