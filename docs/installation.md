# 安装

DOSBox Staging 是必需的模拟器。**用包管理器安装时会自动带上它**，无需手动准备。

## macOS（Homebrew）

本仓库自带 Homebrew formula（`Formula/doscapsule.rb`），可以直接 tap 这个仓库：

```bash
brew tap bytecapsule/doscapsule
brew install doscapsule
```

`dosbox-staging` 声明为强制依赖，brew 会自动安装（Apple Silicon 取 arm64、Intel 取 x64，
两边都是原生二进制）。

升级 / 卸载：

```bash
brew upgrade doscapsule
brew uninstall doscapsule
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

## 没有 DOSBox 会怎样

启动器运行时不强制要求 DOSBox 存在；只有点"启动游戏"时才需要它。
若找不到，会尝试依次查找：

1. 环境变量 `DOSCAPSULE_DOSBOX`（兼容旧名 `DOSGAME_DOSBOX`）
2. 可执行文件旁的 `binaries/dosbox/`、`dosbox-bin/`
3. PATH 中的 `dosbox-staging`
