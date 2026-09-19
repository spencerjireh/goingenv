# /// script
# requires-python = ">=3.12"
# dependencies = ["pillow>=11"]
# ///
"""Render public/bg.png, the page background of the website.

A two-colour dithered collage, Signal ink on the base colour, of four
subjects that belong to the product: a hex dump of a real archive with
its GENV header at the top-left, the pack flow as a line schematic, the
[o] logomark, and a public-domain photograph of punched paper tape.

Run with `make bg` (which is `uv run assets/dither.py`). Downloads land in
assets/.cache/ and are verified against the hashes below, the way
install.sh verifies a release. The hex dump comes from an archive the
script packs itself with a random password, so the bytes are genuine
ciphertext and nothing from this machine is in the image.

Layout is deterministic; only the ciphertext bytes change between runs.

Photo: "PaperTapes-5and8Hole.jpg" by TedColes, public domain, via
Wikimedia Commons. Font: JetBrains Mono, OFL-1.1, JetBrains.
"""

from __future__ import annotations

import hashlib
import io
import math
import os
import secrets
import subprocess
import sys
import tempfile
import urllib.request
import zipfile
from pathlib import Path

from PIL import Image, ImageChops, ImageDraw, ImageFont, ImageOps

ROOT = Path(__file__).resolve().parent.parent
CACHE = ROOT / "assets" / ".cache"
OUT = ROOT / "public" / "bg.png"

USER_AGENT = "goingenv-assets/1.0 (https://github.com/spencerjireh/goingenv)"

PHOTO_URL = "https://upload.wikimedia.org/wikipedia/commons/0/00/PaperTapes-5and8Hole.jpg"
PHOTO_SHA1 = "3df7b2ab3225f87377ad7e972bd7d2cb039df68d"

FONT_ZIP_URL = "https://github.com/JetBrains/JetBrainsMono/releases/download/v2.304/JetBrainsMono-2.304.zip"
FONT_ZIP_SHA256 = "6f6376c6ed2960ea8a963cd7387ec9d76e3f629125bc33d1fdcd7eb7012f7bbf"
FONT_MEMBER = "fonts/ttf/JetBrainsMono-Regular.ttf"

# Palette, from docs/design.md. INK is Signal pulled down towards the base
# so the margins sit below body text in brightness; the tuned value is
# recorded here rather than derived.
BASE = (0x12, 0x14, 0x17)
INK = (0x1E, 0x7A, 0x63)

# The canvas is rendered at half size and scaled up 2x with nearest
# neighbour, so every dither cell is a 2px square. Browser scaling of a
# 1px dither turns it into grey mush; 2px cells survive it.
W, H = 960, 600
SCALE = 2

# How much of the field the frame occupies at 1920px wide (64rem of 1920),
# used to keep the centre quiet and the margins loud.
FRAME_FRACTION = 1024 / 1920


def fetch(url: str, dest: Path, digest: str, algo: str) -> Path:
    if not dest.exists():
        req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
        with urllib.request.urlopen(req, timeout=60) as r:  # noqa: S310
            data = r.read()
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_bytes(data)
    got = hashlib.new(algo, dest.read_bytes()).hexdigest()
    if got != digest:
        sys.exit(f"{dest.name}: {algo} mismatch\n  want {digest}\n  got  {got}")
    return dest


def font_path() -> Path:
    ttf = CACHE / Path(FONT_MEMBER).name
    if ttf.exists():
        return ttf
    z = fetch(FONT_ZIP_URL, CACHE / "JetBrainsMono.zip", FONT_ZIP_SHA256, "sha256")
    with zipfile.ZipFile(z) as zf:
        ttf.write_bytes(zf.read(FONT_MEMBER))
    return ttf


def archive_bytes(n: int) -> bytes:
    """Pack a throwaway project and return the first n bytes of the archive."""
    with tempfile.TemporaryDirectory() as tmp:
        tmpdir = Path(tmp)
        binary = tmpdir / "goingenv"
        subprocess.run(
            ["go", "build", "-o", str(binary), "./cmd/goingenv"],
            cwd=ROOT, check=True,
        )
        project = tmpdir / "project"
        project.mkdir()
        (project / ".env").write_text("DATABASE_URL=postgres://localhost/app\nPORT=3000\n")
        (project / ".env.local").write_text("DEBUG=true\n")
        (project / ".env.production").write_text("PORT=8080\n")

        env = dict(os.environ, GOINGENV_BG_PASSWORD=secrets.token_hex(16))
        quiet = {"stdout": subprocess.DEVNULL, "stderr": subprocess.PIPE}
        subprocess.run([binary, "init"], cwd=project, env=env, check=True, **quiet)
        subprocess.run(
            [binary, "pack", "--password-env", "GOINGENV_BG_PASSWORD"],
            cwd=project, env=env, check=True, **quiet,
        )
        archives = sorted((project / ".goingenv").glob("*.enc"))
        if not archives:
            sys.exit("pack produced no archive")
        return archives[-1].read_bytes()[:n]


def hexdump(data: bytes, width: int = 16) -> list[str]:
    rows = []
    for off in range(0, len(data), width):
        chunk = data[off:off + width]
        pairs = " ".join(f"{a:02x}{b:02x}" for a, b in zip(chunk[::2], chunk[1::2]))
        text = "".join(chr(c) if 32 <= c < 127 else "." for c in chunk)
        rows.append(f"{off:08x}: {pairs:<39}  {text}")
    return rows


def layer() -> Image.Image:
    return Image.new("L", (W, H), 0)


def photo_layer() -> Image.Image:
    src = fetch(PHOTO_URL, CACHE / "papertape.jpg", PHOTO_SHA1, "sha1")
    im = Image.open(src).convert("L")
    # Inverted: the studio backdrop goes dark and disappears into the base;
    # the holes and the tape's shadow edges become the ink.
    im = ImageOps.invert(ImageOps.autocontrast(im, cutoff=1))
    # Crop to the rolls: the bottom third of the frame is empty table.
    w, h = im.size
    im = im.crop((0, int(h * 0.30), w, int(h * 0.75)))
    target_w = int(W * 0.72)
    im = im.resize((target_w, int(im.height * target_w / im.width)), Image.LANCZOS)
    # Gamma up: the mid greys (the backdrop's soft shadow) drop away and
    # only the holes and the rolls' edges keep enough ink to print.
    im = im.point(lambda v: int(255 * (v / 255) ** 1.7 * 0.95))

    # Fade the left edge so it does not end in a hard line.
    fade = Image.linear_gradient("L").rotate(90, expand=True).resize(im.size)
    fade = fade.point(lambda v: 255 if v > 140 else int(v * 255 / 140))
    im = ImageChops.multiply(im, fade)

    out = layer()
    out.paste(im, (W - im.width, H - im.height - 20))
    return out


def hex_layer(font: ImageFont.FreeTypeFont) -> Image.Image:
    rows = hexdump(archive_bytes(16 * 14))
    out = layer()
    d = ImageDraw.Draw(out)
    x, y = 28, 26
    step = 24
    for i, row in enumerate(rows):
        # Brightest at the header row, fading down the dump.
        v = int(210 - i * 12)
        d.text((x, y + i * step), row, fill=v, font=font)
    return out


def schematic_layer(font: ImageFont.FreeTypeFont) -> Image.Image:
    out = layer()
    d = ImageDraw.Draw(out)
    v = 175
    x0, y0 = 40, 392
    fw, fh, gap = 118, 40, 14
    files = [".env", ".env.local", ".env.production"]
    for i, name in enumerate(files):
        y = y0 + i * (fh + gap)
        d.rectangle((x0, y, x0 + fw, y + fh), outline=v, width=1)
        d.text((x0 + 10, y + 11), name, fill=v, font=font)
        # Connector into the bus.
        d.line((x0 + fw, y + fh // 2, x0 + fw + 34, y + fh // 2), fill=v, width=1)
    bus_x = x0 + fw + 34
    top = y0 + fh // 2
    bot = y0 + 2 * (fh + gap) + fh // 2
    mid = (top + bot) // 2
    d.line((bus_x, top, bus_x, bot), fill=v, width=1)
    ax = bus_x + 70
    d.line((bus_x, mid, ax, mid), fill=v, width=1)
    d.polygon([(ax, mid), (ax - 10, mid - 5), (ax - 10, mid + 5)], fill=v)
    # The archive.
    aw, ah = 190, 62
    d.rectangle((ax + 8, mid - ah // 2, ax + 8 + aw, mid + ah // 2), outline=v, width=1)
    d.text((ax + 20, mid - 10), "archive-<ts>.enc", fill=v, font=font)
    # Dimension line under the archive, the way a drawing sizes a part.
    dy = mid + ah // 2 + 18
    d.line((ax + 8, dy, ax + 8 + aw, dy), fill=v, width=1)
    for xx in (ax + 8, ax + 8 + aw):
        d.line((xx, dy - 5, xx, dy + 5), fill=v, width=1)
    d.text((ax + 8 + aw // 2 - 42, dy + 6), "AES-256-GCM", fill=v, font=font)
    return out


def logomark_layer() -> Image.Image:
    """The [o] mark from the SVG sprite in public/index.html, viewBox 0 0 200."""
    out = layer()
    d = ImageDraw.Draw(out)
    size = 430
    ox, oy = W - size * 0.86, -30
    s = size / 200

    def pt(x, y):
        return (ox + x * s, oy + y * s)

    left = [(30, 40), (62, 40), (62, 58), (48, 58), (48, 142), (62, 142), (62, 160), (30, 160)]
    right = [(138, 40), (170, 40), (170, 160), (138, 160), (138, 142), (152, 142), (152, 58), (138, 58)]
    v = 95
    d.polygon([pt(*p) for p in left], fill=v)
    d.polygon([pt(*p) for p in right], fill=v)
    cx, cy = pt(100, 100)
    r = 26 * s
    d.ellipse((cx - r, cy - r, cx + r, cy + r), fill=150)
    return out


def vignette(im: Image.Image) -> Image.Image:
    """Quiet the centre, where the frame sits, and leave the margins loud.

    The bottom eighth fades to nothing so the image ends cleanly when the
    page uses it as a scrolling (non-fixed) background.
    """
    half = FRAME_FRACTION / 2
    mask = Image.new("L", (W, H), 0)
    px = mask.load()
    for x in range(W):
        t = abs(x / W - 0.5)
        if t < half * 0.8:
            f = 0.42
        elif t < half:
            f = 0.42 + 0.58 * (t - half * 0.8) / (half * 0.2)
        else:
            f = 1.0
        for y in range(H):
            g = min(1.0, (H - y) / (H * 0.125))
            px[x, y] = int(255 * f * g)
    return ImageChops.multiply(im, mask)


def main() -> None:
    CACHE.mkdir(parents=True, exist_ok=True)
    mono = ImageFont.truetype(str(font_path()), 17)
    small = ImageFont.truetype(str(font_path()), 13)

    field = layer()
    for lyr in (photo_layer(), logomark_layer(), hex_layer(mono), schematic_layer(small)):
        field = ImageChops.lighter(field, lyr)
    field = vignette(field)

    bits = field.convert("1")  # Floyd-Steinberg
    out = Image.new("P", bits.size)
    out.putpalette(list(BASE) + list(INK) + [0] * (256 * 3 - 6))
    out.paste(bits.point(lambda v: 1 if v else 0, "P"))
    out = out.resize((W * SCALE, H * SCALE), Image.NEAREST)
    out.save(OUT, optimize=True)
    print(f"wrote {OUT.relative_to(ROOT)} ({OUT.stat().st_size // 1024} KB)")


if __name__ == "__main__":
    main()
