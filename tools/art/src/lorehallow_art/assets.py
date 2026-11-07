"""Files generated from the palette and prompt templates.

- art/style/palette.png: swatches, 8 per row, 16x16 each
- art/style/palette.gpl: GIMP/Aseprite/Krita palette
- art/batches/<batch>/prompts.md from prompts.template.md, with the
  palette and key color filled in, so every prompt always carries the
  current palette.
- art/batches/<batch>/guides/<asset>.png layout guides (guides.py).
"""

from __future__ import annotations

import io
from pathlib import Path

import numpy as np
from PIL import Image

from .color import rgb_to_hex, srgb_to_oklab
from .guides import guide_outputs
from .palette import Palette

SWATCH = 16
PER_ROW = 8
KEY_HEX = "#FF00FF"
MIN_KEY_DISTANCE = 0.25


def palette_names(text: str) -> list[str]:
    """The comment after each color line (up to a colon) is its name."""
    names = []
    for raw in text.splitlines():
        line, _, comment = raw.partition(";")
        line = line.strip()
        if line and not line.startswith("//"):
            names.append(comment.strip().split(":")[0] or line.lstrip("#").upper())
    return names


def swatch_image(palette: Palette) -> Image.Image:
    rows = -(-len(palette) // PER_ROW)
    arr = np.zeros((rows * SWATCH, PER_ROW * SWATCH, 4), dtype=np.uint8)
    for k, c in enumerate(palette.colors):
        r, col = divmod(k, PER_ROW)
        arr[r * SWATCH : (r + 1) * SWATCH, col * SWATCH : (col + 1) * SWATCH] = (*c, 255)
    return Image.fromarray(arr, "RGBA")


def gpl_text(palette: Palette, names: list[str]) -> str:
    lines = ["GIMP Palette", "Name: Lorehallow", f"Columns: {PER_ROW}", "#"]
    for c, n in zip(palette.colors, names, strict=True):
        lines.append(f"{c[0]:3d} {c[1]:3d} {c[2]:3d}\t{n}")
    return "\n".join(lines) + "\n"


def palette_outputs(root: Path) -> dict[Path, bytes]:
    """Expected generated palette files and their exact contents."""
    src = root / "art" / "style" / "palette.hex"
    text = src.read_text()
    pal = Palette.from_hex_lines(text)
    png = root / "art" / "style" / "palette.png"
    buf = io.BytesIO()
    swatch_image(pal).save(buf, format="PNG", optimize=True)
    return {
        png: buf.getvalue(),
        root / "art" / "style" / "palette.gpl": gpl_text(pal, palette_names(text)).encode(),
    }


def palette_problems(palette: Palette) -> list[str]:
    """Palette colors the pipeline could confuse with the key color."""
    problems = []
    key = srgb_to_oklab(np.array((255, 0, 255)))
    d = np.linalg.norm(palette.lab - key, axis=1)
    for i in np.flatnonzero(d < MIN_KEY_DISTANCE):
        problems.append(f"{rgb_to_hex(palette.colors[i])} is too close to the key color {KEY_HEX}")
    return problems


def render_prompts(template: str, palette: Palette) -> str:
    hexes = ", ".join(palette.hex_list())
    out = (
        template.replace("{{PALETTE}}", hexes)
        .replace("{{PALETTE_COUNT}}", str(len(palette)))
        .replace("{{KEY}}", KEY_HEX)
    )
    header = (
        "<!-- Generated from prompts.template.md by `make art-prompts`. "
        "Edit the template, not this file. -->\n\n"
    )
    return header + out


def prompt_outputs(root: Path) -> dict[Path, bytes]:
    pal = Palette.load(root / "art" / "style" / "palette.hex")
    out = {}
    for tpl in sorted((root / "art" / "batches").glob("batch-*/prompts.template.md")):
        out[tpl.with_name("prompts.md")] = render_prompts(tpl.read_text(), pal).encode()
    return out


def generated_outputs(root: Path) -> dict[Path, bytes]:
    return {**palette_outputs(root), **prompt_outputs(root), **guide_outputs(root)}


def write_generated(root: Path) -> list[Path]:
    written = []
    for path, data in generated_outputs(root).items():
        if not path.exists() or path.read_bytes() != data:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(data)
            written.append(path)
    return written


def stale_generated(root: Path) -> list[Path]:
    files = generated_outputs(root)
    return [p for p, data in files.items() if not p.exists() or p.read_bytes() != data]
