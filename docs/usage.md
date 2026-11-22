# 使用指南

## 运行

```bash
./doscapsule              # 监听 127.0.0.1:8420（占用则随机端口），自动开浏览器
./doscapsule -port=9000   # 指定端口
DOSCAPSULE_DOSBOX=/path/dosbox-staging ./doscapsule   # 指定 DOSBox
```

## 游戏库目录

每个游戏占用一个子目录，内含：

- `@launcher.json` —— 必须，至少含 `title`（见 [configuration.md](configuration.md)）
- `play.bat` —— 游戏启动脚本
- `@dosbox.conf` —— （可选）游戏的 DOSBox 配置，缺省用启动器默认配置
- `@cover.jpg` —— （可选）封面，4:3 / 400×300；缺省用内置兜底图

第一次启动后在界面上选择游戏库根目录，之后自动记忆（`state.json`），重启自动恢复。

## 常用操作

- **启动游戏**：点击卡片
- **右键菜单**：Play / 编辑 DOSBox 配置 / 打开目录 / 收藏 / 停止（全部操作入口）
- **搜索 / 筛选 / 排序**：顶部工具栏
- **编辑 DOSBox 配置**：右键 → 编辑，写入游戏目录的 `@dosbox.conf`

## DOSBox 查找顺序

`$DOSCAPSULE_DOSBOX` → 二进制旁 `dosbox-bin/`、`binaries/dosbox/`、`binaries/dosbox-staging-*`
→ 系统 PATH 的 `dosbox-staging`。

## 数据与配置存放

- 状态（收藏、最近游玩、游戏库根）：`data/state.json`
- 启动器默认配置：`data/launch/defaults.conf`
- 运行日志：`data/dosbox.log`

> 便携模式：当 `binaries/dosbox/` 与可执行文件同目录时，`data/` 位于可执行文件旁，
> 整个项目文件夹可整体拷贝到别的机器直接运行。未检测到便携 DOSBox 时回退到系统用户配置目录。
