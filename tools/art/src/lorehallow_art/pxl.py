"""Pixel grids: hand-drawn art as text (`art/src/<category>/<name>.pxl.yaml`).

Shape::

    size: [32, 48]            # one frame, width × height
    legend:                   # one character per color, by palette name
      ".": transparent
      K: ink
      S: skin mid
    frames:                   # one grid per frame, in reading order
      - |
        ................................
        ...

Every grid is exactly `size` (height lines of width characters). A
character missing from the legend, a color name missing from the palette,
or a grid of the wrong size is an error, so nothing off-palette or
misaligned can be drawn. Lines may carry a trailing `  # comment`.
"""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any

import numpy as np
import yaml

from .palette import NamedPalette

TRANSPARENT = "transparent"


class SourceError(ValueError):
    """A hand-drawn source that can't be rendered."""


@dataclass(frozen=True)
class Pxl:
    size: tuple[int, int]
    frames: tuple[np.ndarray, ...]  # RGBA, (height, width, 4) each


def _grid_lines(text: str) -> list[str]:
    lines = []
    for raw in text.splitlines():
        line = raw.split("  #", 1)[0].rstrip() if "  #" in raw else raw.rstrip()
        lines.append(line)
    while lines and not lines[-1]:
        lines.pop()
    return lines


def parse_pxl(data: Any, palette: NamedPalette, where: str = "pxl") -> Pxl:
    def err(msg: str) -> SourceError:
        return SourceError(f"{where}: {msg}")

    if not isinstance(data, dict):
        raise err("must be a mapping with size, legend and frames")
    unknown = set(data) - {"size", "legend", "frames", "notes"}
    if unknown:
        raise err(f"unknown keys {sorted(unknown)}")
    size = data.get("size")
    if not (
        isinstance(size, list)
        and len(size) == 2
        and all(isinstance(v, int) and v > 0 for v in size)
    ):
        raise err(f"size must be [width, height], got {size!r}")
    w, h = size
    legend_raw = data.get("legend")
    if not isinstance(legend_raw, dict) or not legend_raw:
        raise err("legend must map single characters to palette color names")
    legend: dict[str, tuple[int, int, int, int]] = {}
    for ch, name in legend_raw.items():
        ch = str(ch)
        if len(ch) != 1 or ch.isspace():
            raise err(f"legend key {ch!r} must be one non-space character")
        if name == TRANSPARENT:
            legend[ch] = (0, 0, 0, 0)
        elif isinstance(name, str) and name in palette:
            legend[ch] = (*palette[name], 255)
        else:
            raise err(f"legend {ch!r}: unknown palette color {name!r}")
    frames_raw = data.get("frames")
    if not isinstance(frames_raw, list) or not frames_raw:
        raise err("frames must be a non-empty list of grids")
    frames = []
    for k, grid in enumerate(frames_raw):
        if not isinstance(grid, str):
            raise err(f"frame {k} must be a block of text (use `- |`)")
        lines = _grid_lines(grid)
        if len(lines) != h:
            raise err(f"frame {k} has {len(lines)} rows, expected {h}")
        arr = np.zeros((h, w, 4), dtype=np.uint8)
        for y, line in enumerate(lines):
            if len(line) != w:
                raise err(f"frame {k} row {y} has {len(line)} columns, expected {w}")
            for x, ch in enumerate(line):
                if ch not in legend:
                    raise err(f"frame {k} row {y} column {x}: {ch!r} is not in the legend")
                arr[y, x] = legend[ch]
        frames.append(arr)
    return Pxl((w, h), tuple(frames))


def load_pxl(path: Path, palette: NamedPalette) -> Pxl:
    try:
        data = yaml.safe_load(path.read_text())
    except yaml.YAMLError as e:
        raise SourceError(f"{path}: not valid YAML: {e}") from e
    return parse_pxl(data, palette, str(path))
