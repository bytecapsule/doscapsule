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

文档见 [docs/](docs/README.md)。

## 许可

[MIT](LICENSE)
