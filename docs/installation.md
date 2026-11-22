# 安装

DOSBox Staging 是必需的模拟器。**用包管理器安装时会自动带上它**，无需手动准备。

## macOS（Homebrew）

本仓库自带 Homebrew formula（`Formula/doscapsule.rb`），所以它本身就是一个 tap，
不需要额外的仓库。因为仓库名不是 `homebrew-*` 约定，tap 时要显式给出 URL：

```bash
brew tap bytecapsule/doscapsule https://github.com/bytecapsule/doscapsule
brew trust bytecapsule/doscapsule          # 新版 Homebrew 对第三方 tap 要求信任
brew install bytecapsule/doscapsule/doscapsule
```

`dosbox-staging` 声明为强制依赖，brew 会自动安装（Apple Silicon 取 arm64、
Intel 取 x64，两边都是原生二进制）。

之后日常使用就是普通的 `doscapsule` 命令。升级与卸载：

```bash
brew upgrade bytecapsule/doscapsule/doscapsule
brew uninstall bytecapsule/doscapsule/doscapsule
```

## Windows 10/11（Scoop）

本仓库同时是一个 Scoop bucket（`bucket/doscapsule.json`）：

```powershell
scoop bucket add bytecapsule https://github.com/bytecapsule/doscapsule
scoop install doscapsule
```

`dosbox-staging` 通过 manifest 的 `depends` 自动安装。

> 需要 Windows 10/11 + PowerShell 5.1+。Windows 7 不在支持范围内
> （DOSBox Staging 最新版本身要求 Windows 8+）。

## 手动安装（任意平台）

从 [Releases](https://github.com/bytecapsule/doscapsule/releases) 下载对应平台的 zip：

| 平台 | 文件 |
|---|---|
| macOS Apple Silicon | `doscapsule-arm64-<tag>.zip` |
| macOS Intel | `doscapsule-x64-<tag>.zip` |
| Windows x64 | `doscapsule-windows-x64-<tag>.zip` |

解压后把二进制放进 PATH，再自行安装 DOSBox Staging：

```bash
# macOS
brew install dosbox-staging
# Windows
scoop install dosbox-staging
# 或者从官网下载：https://dosbox-staging.github.io/
```

## 需要手动指定 DOSBox 吗？

**通常不需要。** 用 brew / scoop 装的 DOSBox Staging 就在 PATH 里，启动器会自动找到。

查找顺序（找到即停）：

1. 环境变量 `DOSCAPSULE_DOSBOX`（旧名 `DOSGAME_DOSBOX` 仍兼容）
2. 可执行文件旁的 `binaries/dosbox/`（便携布局）
3. 可执行文件旁的 `dosbox-bin/`、`resources/dosbox-bin/`
4. 可执行文件旁的 `binaries/dosbox-staging-*`（开发布局）
5. 系统 PATH 中的 `dosbox-staging`

只有当 DOSBox **不在 PATH 里、也不在上面那些位置**时，才需要手动指定：

```bash
DOSCAPSULE_DOSBOX=/Applications/DOSBox\ Staging.app/Contents/MacOS/dosbox-staging doscapsule
```

启动器运行时不强制要求 DOSBox 存在；只有点"启动游戏"时才需要它。
