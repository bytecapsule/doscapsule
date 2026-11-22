# 构建

推荐用 **Makefile** 统一入口（跨平台，含 macOS arm64/x64、Linux、Windows）。

```bash
make dist      # 只构建前端（输出到 cmd/doscapsule/dist）
make build     # 构建当前平台二进制 doscapsule
make test      # 跑后端测试
make run       # 构建并运行
make clean     # 清理产物
```

## 手动构建

### 1) 前端（SolidJS + Vite）

```bash
cd frontend
pnpm install
pnpm build     # 产物输出到 ../cmd/doscapsule/dist，被 go:embed 嵌入
cd ..
```

### 2) 后端（Go）

```bash
go build -o doscapsule ./cmd/doscapsule
```

`//go:embed` 会把你刚构建的 `cmd/doscapsule/dist/` 和
`cmd/doscapsule/resources/fallback-cover.jpg` 一起编译进二进制。

### 3) 运行

```bash
./doscapsule
```

## 交叉编译（macOS x64 / arm64）

同一台 Mac 上即可产出两个架构：

```bash
GOOS=darwin GOARCH=arm64 go build -o doscapsule-arm64 ./cmd/doscapsule
GOOS=darwin GOARCH=amd64 go build -o doscapsule-amd64 ./cmd/doscapsule
```

## 前提

- Go 1.27+
- Node.js / pnpm（构建前端用）
- （打包发布时才需要）对应平台的 DOSBox Staging 二进制

发布流程见 [packaging.md](packaging.md)。
