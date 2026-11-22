# 配置说明

## `@launcher.json`（游戏 manifest）

每个游戏目录的可选清单文件，唯一必填字段是 `title`。

```json
{
  "title": "仙剑奇侠传",
  "sort_title": "xian jian qi xia zhuan",
  "platform": "DOS",
  "developer": "大宇资讯",
  "publisher": "大宇资讯",
  "year": 1995,
  "genre": "RPG",
  "description": "...",
  "launch": {
    "command": "call play.bat",
    "config": "@dosbox.conf"
  },
  "media": {
    "cover": "@cover.jpg"
  }
}
```

- `launch.command`：游戏启动命令，缺省 `call play.bat`
- `launch.config`：游戏用的 DOSBox 配置文件名，缺省 `@dosbox.conf`
- `media.cover`：封面文件名，缺省 `@cover.jpg`

历史版本也用 `@dosapp_launcher.json`，扫描器两者都认，优先新名。

## `@dosbox.conf`（游戏的 DOSBox 配置）

启动器每次启动会先传 `defaults.conf`（见下），再传游戏自己的配置。DOSBox 按顺序合并，
所以**游戏配置里写了同名键就会覆盖默认值**。

## 启动器默认配置（`data/launch/defaults.conf`）

```ini
[dosbox]
memsize = 30

[cpu]
cpu_cycles_protected = max

[render]
crt_color_profile = p22
color_temperature = 7800

[midi]
mididevice = fluidsynth
```

要点：

- `memsize = 30`：1990 年代 DOS 游戏给 16MB 余量偏紧，30MB 对现代机器无压力。
- `cpu_cycles_protected = max`：全局默认把保护模式锁在 60 MHz，Duke3D、古墓丽影这类
  32 位（DOS4GW）3D 游戏会明显变慢。该键只作用于保护模式，16 位实模式游戏走
  `cpu_cycles`，不受影响。
- `mididevice = fluidsynth`：macOS 的 CoreMIDI 无合成器，通用 MIDI 音乐默认走
  FluidSynth（需 `soundfonts/default.sf2`）。

想给某个游戏改回去，在其 `@dosbox.conf` 写：

```ini
[cpu]
cpu_cycles_protected = 60000
```

## DOSBox 资源（音色库 / Unicode 映射）

- **音色库**：`soundfonts/default.sf2`（GeneralUser GS）。
- **Unicode 映射**：`mapping/`、`mapping-freedos.org/`、`mapping-unicode.org/`、
  `mapping-wikipedia.org/`。缺了它们，DOSBox shell 的 UTF-8 制表符会显示成 `?`。

首次使用或从零搭建 DOSBox 便携目录，运行：

```bash
./scripts/stage-dosbox-resources.sh "/Applications/DOSBox Staging.app"
```

（脚本把资源复制到 `binaries/dosbox/`。）
