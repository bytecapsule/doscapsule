# DOS Capsule

纯本地 DOS 游戏启动器 —— 一个独立的 Go 可执行文件，无数据库、无 torrent、无运行时依赖。
内置 SolidJS 前端（`go:embed` 嵌入二进制），启动后自动打开系统默认浏览器。

一个把 DOS 老游戏装进"胶囊"里的怀旧前端：扫描本地游戏目录，点开即玩。

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

## 运行

```bash
doscapsule              # 自动开浏览器，监听 127.0.0.1 的可用端口
```

DOSBox 会自动在 PATH 中查找（brew / scoop 装好即可，无需配置）。
只有当 DOSBox 不在 PATH 里时才需要手动指定：

```bash
DOSCAPSULE_DOSBOX=/path/to/dosbox-staging doscapsule
```

首次启动后在界面上选择游戏库根目录，之后自动记忆。

## 功能

- 递归扫描 + `@launcher.json` manifest（`title` 唯一必填）
- 搜索、流派筛选、排序（标题 / 年份 / 流派 / 最近游玩）
- 收藏、最近游玩，扫描结果持久化（`state.json`），重启自动恢复
- 点击卡片启动；悬停轮播截图；无封面用内置兜底图
- DOSBox 配置可视化编辑；运行中徽标 + 停止；相对路径 id 防路径穿越

## 开发

```bash
make dist      # 构建前端（输出到 cmd/doscapsule/dist）
make build     # 构建当前平台二进制
make test      # 跑测试
make run       # 构建并运行
make zip-arm   # 打 macOS arm64 包
make zip-x64   # 打 macOS x64 包
make help      # 查看全部目标
```

文档见 [docs/](docs/README.md)。

## 许可

[MIT](LICENSE)
