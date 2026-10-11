import argparse
import sys
from pathlib import Path

from PIL import Image, ImageChops, ImageDraw


def round_and_border(png, dst, line, radius=24, border=3, supersample=4):
    img = Image.open(png).convert("RGB")
    w, h = img.size

    def mask(inset, r):
        m = Image.new("L", (w * supersample, h * supersample), 0)
        ImageDraw.Draw(m).rounded_rectangle(
            (inset * supersample, inset * supersample,
             (w - inset) * supersample - 1, (h - inset) * supersample - 1),
            radius=r * supersample, fill=255)
        return m.resize((w, h), Image.LANCZOS)

    outer, inner = mask(0, radius), mask(border, radius - border)
    out = img.convert("RGBA")
    out.paste(Image.new("RGBA", (w, h), line + (255,)), (0, 0), ImageChops.subtract(outer, inner))
    out.putalpha(outer)
    out.save(dst, "WEBP", lossless=True, method=6)
    return w, h


def hex_rgb(value):
    value = value.lstrip("#")
    return tuple(int(value[i:i + 2], 16) for i in (0, 2, 4))


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("src")
    ap.add_argument("out")
    ap.add_argument("--light", default="#c8d5e2")
    ap.add_argument("--dark", default="#314154")
    ap.add_argument("--max-kb", type=int, default=300)
    a = ap.parse_args()

    src, out = Path(a.src), Path(a.out)
    out.mkdir(parents=True, exist_ok=True)
    lines = {"light": hex_rgb(a.light), "dark": hex_rgb(a.dark)}
    problems = 0
    for png in sorted(src.glob("*.png")):
        item, _, theme = png.stem.rpartition("-")
        if theme not in lines:
            continue
        dst = out / (f"{item}.webp" if theme == "light" else f"{item}-{theme}.webp")
        w, h = round_and_border(png, dst, lines[theme])
        kb = dst.stat().st_size / 1024
        flag = "" if kb <= a.max_kb else f"  WARN over {a.max_kb} KB"
        problems += bool(flag)
        print(f"{dst.name}  {w}x{h}  {kb:.0f} KB{flag}")
    sys.exit(1 if problems else 0)


if __name__ == "__main__":
    main()
