#!/usr/bin/env python3
"""Normalize a cover image to the DOSGame Launcher cover standard.

Standard: 4:3 landscape, 400x300, JPEG q90. Any input (any size/ratio/
format Pillow can read) is center-cropped to 4:3 and scaled to 400x300.

Usage:
    python3 normalize_cover.py <image> [more images...]
    python3 normalize_cover.py --dir <games_root>   # every */cover.jpg under root

The result overwrites <game_dir>/cover.jpg in place.
"""
import io
import sys
from pathlib import Path

from PIL import Image

W, H = 400, 300  # 4:3 landscape


def normalize_bytes(data: bytes) -> Image.Image:
    im = Image.open(io.BytesIO(data)).convert("RGB")
    iw, ih = im.size
    tr = W / H
    if iw / ih > tr:  # too wide: crop the width
        nw = int(ih * tr)
        x0 = (iw - nw) // 2
        im = im.crop((x0, 0, x0 + nw, ih))
    else:             # too tall: crop the height
        nh = int(iw / tr)
        y0 = (ih - nh) // 2
        im = im.crop((0, y0, iw, y0 + nh))
    return im.resize((W, H), Image.LANCZOS)


def normalize_file(path: Path) -> None:
    data = path.read_bytes()
    is_shot = path.name.lstrip("@").lower().startswith("screenshot")
    im = Image.open(io.BytesIO(data)).convert("RGB")
    if is_shot:
        # Screenshots: stretch to 4:3 (what DOSBox aspect correction shows).
        im = im.resize((W, H), Image.LANCZOS)
        out = path  # keep @screenshot-N.jpg name
    else:
        im = normalize_bytes(data)  # covers: center-crop to 4:3
        out = path.with_name("@cover.jpg")
    im.save(out, "JPEG", quality=90)
    print(f"{path} -> {out} (400x300)")


def main() -> None:
    args = sys.argv[1:]
    if not args:
        print(__doc__)
        sys.exit(1)
    targets: list[Path] = []
    if args[0] == "--dir":
        root = Path(args[1])
        targets = sorted(list(root.glob("*/@cover.jpg")) + list(root.glob("*/cover.jpg")))
    else:
        targets = [Path(a) for a in args]
    for t in targets:
        try:
            normalize_file(t)
        except Exception as e:
            print(f"{t}: FAILED ({e})")


if __name__ == "__main__":
    main()
