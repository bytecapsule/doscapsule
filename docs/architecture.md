# 架构

## 目录结构

- cmd/doscapsule/     Go 源码 + go:embed 资源（main/launcher/catalog/state/util + test）
- frontend/        SolidJS + Vite 前端
- scripts/         构建/资源脚本
- docs/            文档
- .github/workflows/  CI/CD
- Makefile
- go.mod
- README.md

采用 Go 社区标准的 cmd/ 布局。源码保持单一 package main，靠 cmd/ 目录归档，
未拆成多个 internal 包（项目约 1200 行，拆包收益小、风险高）。

## 模块职责

- main.go:      go:embed、HTTP server、/api/* 路由、打开浏览器
- launcher.go:  DOSBox 路径解析、写 defaults.conf、exec 启动、运行中注册
- catalog.go:   Catalog 扫描、LocalGame 结构、manifest 读写
- state.go:     state.json 读写、stateDir() 便携/系统目录解析
- util.go:      resolveGameDir（防路径穿越）、isDir 等

## 关键设计决策

- go:embed 前端：把前端构建产物嵌进二进制，单文件分发。产物路径 cmd/doscapsule/dist。
- 便携模式：可执行文件旁有 binaries/dosbox/ 时，状态存到旁的 data/，整体可拷贝；
  否则回退到系统用户配置目录。
- 默认 DOSBox 配置：先传 defaults.conf，再传游戏自己的 @dosbox.conf，后者覆盖前者。
- MIDI 音乐：macOS CoreMIDI 无合成器，mididevice=fluidsynth 让通用 MIDI 走音色库。
  音色库 soundfonts/default.sf2 随便携目录分发。

## API

/api/scan|games|genres|remembered_root|launch|stop|running|favorite|conf|open|pick_folder|clear
+ /media?path=。前端通过 frontend/src/api/tauri.ts 的 fetch shim 调用。
