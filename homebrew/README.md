# Homebrew 支持

让 `brew install doscapsule` 可用。本仓库（或其 fork）作为 Homebrew tap，
formula 从 GitHub Release 下载 zip 安装，并**自动安装 DOSBox Staging**。

## 架构支持

每个 Release 发布两个 zip：

- `doscapsule-arm64-<TAG>.zip`  —— Apple Silicon (M1/M2/...)
- `doscapsule-x64-<TAG>.zip`    —— Intel Mac

formula 用 `Hardware::CPU.arm?` 自动选：

- Apple Silicon Mac → arm64 zip
- Intel Mac → x64 zip

两版都是各自的原生二进制（Go 交叉编译，无 Rosetta 翻译损耗）。

## DOSBox 自动安装

formula 把 `dosbox-staging` 声明为**强制依赖**：

- `brew install doscapsule` 时，brew 先自动装 dosbox-staging（按主机架构，Intel -> x86_64，Apple Silicon -> arm64）。
- 装完启动器在 PATH 找到 dosbox-staging，直接可用。

## 前提

- GitHub 仓库 **public**
- 仓库名 `homebrew-tap`（`brew tap bytecapsule/homebrew-tap` 对应 `bytecapsule/homebrew-tap`）

## 步骤

1. 把 `homebrew/doscapsule.rb` 放进 `homebrew-tap` 仓库根目录。
2. 填 formula 占位符：`bytecapsule/doscapsule`、`<TAG>`、`<SHA256_ARM64>`、`<SHA256_X64>`。
3. 发布 Release（CI 出 arm64 + x64 两个 zip）。
4. 用户执行：
   ```bash
   brew tap bytecapsule/homebrew-tap
   brew install doscapsule
   ```

## 注意

- 每次 Release 后更新 `<TAG>` 和两个 SHA-256。
- SHA-256：`shasum -a 256 doscapsule-arm64-*.zip` 和 `...-x64-*.zip`。
