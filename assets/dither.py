# /// script
# requires-python = ">=3.12"
# dependencies = ["pillow>=11"]
# ///
"""Render public/bg.png, the page background of the website.

A two-colour dithered collage, Signal ink on the base colour, tall enough
to scroll with the page. The frame in the middle is opaque, so the image
is designed for what shows: the two margins and a faint hero. Down the
left, a hex dump of a real archive with its GENV header on the first row.
Down the right, three objects: an IBM 80-column punched card stood on
end, the pack flow as a line schematic, and a card-storage warehouse.

Run with `make bg` (which is `uv run assets/dither.py`). Downloads land in
assets/.cache/ and are verified against the hashes below, the way
install.sh verifies a release. The hex dump comes from an archive the
script packs itself with a random password, so the bytes are genuine
ciphertext and nothing from this machine is in the image.

Layout is deterministic; only the ciphertext bytes change between runs.

Photos, both public domain via Wikimedia Commons:
  "Blue-punch-card-front-horiz.png", Gwern, derivative by agr.
  "IBM card storage.NARA.jpg", US National Archives, catalog 12169529.
Font: JetBrains Mono, OFL-1.1, JetBrains.
"""

from __future__ import annotations

import hashlib
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

CARD_URL = "https://upload.wikimedia.org/wikipedia/commons/4/4c/Blue-punch-card-front-horiz.png"
CARD_SHA1 = "14addcbc3e0def4a92ff8e916c07a7731d149e54"

STORAGE_URL = "https://upload.wikimedia.org/wikipedia/commons/8/87/IBM_card_storage.NARA.jpg"
STORAGE_SHA1 = "d87513cd01396833f41ad240115d86c6f72685b9"

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
# 1px dither turns it into grey mush; 2px cells survive it. 1920x7200
# covers the page at any width from 1280 up; the bottom fades to base.
W, H = 960, 3600
SCALE = 2

# How much of the field the frame occupies at 1920px wide (64rem of 1920),
# used to keep the centre quiet and the margins loud.
FRAME_FRACTION = 1024 / 1920

HEX_ROW = 24  # canvas px per hex dump row


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


def fake_env(prefix: str, n: int) -> str:
    """Enough made-up keys that the archive runs to a few kilobytes."""
    lines = []
    for i in range(n):
        lines.append(f"{prefix}_{i:02d}_URL=https://service-{i}.internal.example/v1")
        lines.append(f"{prefix}_{i:02d}_TOKEN={secrets.token_hex(12)}")
    return "\n".join(lines) + "\n"


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
        (project / ".env").write_text(fake_env("APP", 24))
        (project / ".env.local").write_text(fake_env("LOCAL", 12))
        (project / ".env.production").write_text(fake_env("PROD", 24))

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
        data = archives[-1].read_bytes()
        if len(data) < n:
            sys.exit(f"archive is {len(data)} bytes; need {n} for the dump")
        return data[:n]


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


def load_photo(src: Path, gamma: float, invert: bool) -> Image.Image:
    im = ImageOps.autocontrast(Image.open(src).convert("L"), cutoff=1)
    if invert:
        im = ImageOps.invert(im)
    # Gamma up: mid greys (paper grain, soft shadow) drop away and only
    # the real marks keep enough ink to print.
    return im.point(lambda v: int(255 * (v / 255) ** gamma))


def fade_left(im: Image.Image, width_fraction: float = 0.45) -> Image.Image:
    """Fade the left edge into the field so it does not end in a line."""
    fade = Image.linear_gradient("L").rotate(90, expand=True).resize(im.size)
    knee = int(255 * width_fraction)
    fade = fade.point(lambda v: 255 if v > knee else int(v * 255 / knee))
    return ImageChops.multiply(im, fade)


def hex_layer(font: ImageFont.FreeTypeFont) -> Image.Image:
    """The archive, dumped down the left edge for the height of the page."""
    rows_that_fit = (H - 26) // HEX_ROW
    rows = hexdump(archive_bytes(16 * rows_that_fit))
    out = layer()
    d = ImageDraw.Draw(out)
    for i, row in enumerate(rows):
        # Brightest at the header row, settling to a steady value so the
        # column reads all the way down rather than fading to nothing.
        v = max(120, 210 - i * 8)
        d.text((28, 26 + i * HEX_ROW), row, fill=v, font=font)
    return out


def card_layer() -> Image.Image:
    """An 80-column card stood on end, flush with the right edge."""
    src = fetch(CARD_URL, CACHE / "punchcard.png", CARD_SHA1, "sha1")
    # Not inverted, and a steep gamma: the card's blue field falls to a
    # faint tone, the printed digits vanish into it, and the holes, which
    # scan white, are what prints. The scan's white surround is cropped
    # off first or it would print as a solid border.
    im = Image.open(src).convert("L")
    w, h = im.size
    im = im.crop((int(w * 0.012), int(h * 0.025), int(w * 0.988), int(h * 0.975)))
    im = ImageOps.autocontrast(im, cutoff=1).point(lambda v: int(255 * (v / 255) ** 2.4))
    im = im.rotate(90, expand=True)
    target_w = 250
    im = im.resize((target_w, int(im.height * target_w / im.width)), Image.LANCZOS)
    out = layer()
    out.paste(im, (W - im.width + 12, 170))
    return out


def schematic_layer(font: ImageFont.FreeTypeFont) -> Image.Image:
    """The pack flow, stacked so it fits the right margin: three files, a
    bus, the archive, and a dimension line, the way a drawing sizes a part."""
    out = layer()
    d = ImageDraw.Draw(out)
    v = 175
    bw, bh, gap = 134, 34, 12
    x0 = W - bw - 10
    y = 1420
    for name in (".env", ".env.local", ".env.production"):
        d.rectangle((x0, y, x0 + bw, y + bh), outline=v, width=1)
        d.text((x0 + 8, y + 9), name, fill=v, font=font)
        # Connector out of the box into the bus on its left.
        d.line((x0, y + bh // 2, x0 - 22, y + bh // 2), fill=v, width=1)
        y += bh + gap
    bus_x = x0 - 22
    top = 1420 + bh // 2
    bot = y - gap - bh // 2
    d.line((bus_x, top, bus_x, bot), fill=v, width=1)
    # Down from the bus, then back in to the archive.
    ay = y + 26
    d.line((bus_x, bot, bus_x, ay + 31), fill=v, width=1)
    d.line((bus_x, ay + 31, x0, ay + 31), fill=v, width=1)
    d.polygon([(x0, ay + 31), (x0 - 9, ay + 26), (x0 - 9, ay + 36)], fill=v)
    d.rectangle((x0, ay, x0 + bw, ay + 62), outline=v, width=1)
    d.text((x0 + 8, ay + 12), "archive-", fill=v, font=font)
    d.text((x0 + 8, ay + 32), "<ts>.enc", fill=v, font=font)
    dy = ay + 62 + 16
    d.line((x0, dy, x0 + bw, dy), fill=v, width=1)
    for xx in (x0, x0 + bw):
        d.line((xx, dy - 5, xx, dy + 5), fill=v, width=1)
    d.text((x0 + 14, dy + 6), "AES-256-GCM", fill=v, font=font)
    return out


def storage_layer() -> Image.Image:
    """Boxed cards to the horizon: an archive, lower right."""
    src = fetch(STORAGE_URL, CACHE / "cardstorage.jpg", STORAGE_SHA1, "sha1")
    im = load_photo(src, gamma=1.6, invert=False)
    w, h = im.size
    # Crop off the railing at the left and the ceiling at the top.
    im = im.crop((int(w * 0.18), int(h * 0.10), w, h))
    target_w = 520
    im = im.resize((target_w, int(im.height * target_w / im.width)), Image.LANCZOS)
    im = fade_left(im.point(lambda v: int(v * 0.9)))
    out = layer()
    out.paste(im, (W - im.width, 2320))
    return out


def vignette(im: Image.Image) -> Image.Image:
    """Quiet the centre, where the frame sits, and leave the margins loud.

    The bottom eighth fades to nothing so the image ends in the base
    colour before the page does.
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
    for lyr in (card_layer(), storage_layer(), hex_layer(mono), schematic_layer(small)):
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
