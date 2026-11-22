# 打包与发布

## 概述

发布目标是 GitHub Release。每次成功构建按时间戳打 tag 发一个 Release：macOS arm64 和 x64 各一个 zip。zip 内含：启动器二进制（前端资源已嵌入）+ README +docs，**不含 DOSBox**。

DOSBox Staging 由用户自行安装（brew 或官方 dmg）。用户 brew install doscapsule 时，formula 会自动装上 dosbox-staging 作为依赖，开箱即用。

## 本地打包

在能跑 make 的平台上：

    make zip          # 当前平台：build dist + go build + 打 zip
    make zip-arm      # macOS arm64
    make zip-x64      # macOS amd64
    make zip-all      # 双架构

zip 布局：

    doscapsule/
    |-- doscapsule      # 可执行文件
    |-- README.md
    |-- INSTALL-DOSBox.txt    # 提示自行安装 DOSBox
    `-- docs/usage.md 和 configuration.md

## 发布（GitHub Actions）

- 矩阵：macos-arm64、macos-x64
- 步骤：setup go/node -> pnpm build -> go build（对应 arch）-> scripts/zip.sh -> 上传
- 版本/标签：时间戳（date +%Y%m%d-%H%M%S）
- 模式（workflow_dispatch 手动触发）：
  - release: true（默认）：打 tag、发 GitHub Release
  - release: false：只上传构建 artifact，不发 Release（试构建）
- 触发：GitHub 仓库 -> Actions -> build-release -> Run workflow

## DOSBox 安装（用户侧）

- macOS（推荐）：brew install dosbox-staging
- 或官网下载：https://dosbox-staging.github.io/

启动器运行时自动在 PATH 查找 dosbox-staging，无需额外配置。

## Homebrew

Formula/doscapsule.rb 是 Homebrew formula，bucket/doscapsule.json 是 Scoop manifest，两者都把 dosbox-staging 声明为强制依赖，安装时自动带上。它们直接放在本仓库里，因此不需要额外的 tap/bucket 仓库：brew tap bytecapsule/doscapsule 与 scoop bucket add bytecapsule https://github.com/bytecapsule/doscapsule 都指向这个仓库。
