"""Drawing helpers for draw scripts (`art/src/<category>/<name>.py`).

A draw script defines `draw(ctx) -> image` and builds the image with a
`Canvas`. Colors are always palette names (`"ink"`, `"moss shadow"`) or
None for transparent, so nothing off-palette can be drawn. Randomness
comes only from `ctx.rng(seed)`, so every render is identical.

A canvas made with `wrap=True` wraps every operation around its edges:
a stroke that leaves the right edge comes back on the left. Ground tiles
drawn that way are seamless by construction.
"""

from __future__ import annotations

import importlib.util
import itertools
import random
from collections.abc import Callable, Iterable, Sequence
from pathlib import Path
from types import ModuleType
from typing import TYPE_CHECKING

import numpy as np
from PIL import Image

from .palette import NamedPalette
from .pxl import SourceError, load_pxl

if TYPE_CHECKING:
    from .manifest import Asset

Color = str | None


class Canvas:
    """An RGBA pixel canvas drawn with palette color names."""

    def __init__(self, palette: NamedPalette, w: int, h: int, *, wrap: bool = False) -> None:
        self.palette = palette
        self.w, self.h = w, h
        self.wrap = wrap
        self.a = np.zeros((h, w, 4), dtype=np.uint8)

    # -- construction -------------------------------------------------

    @classmethod
    def from_array(cls, palette: NamedPalette, arr: np.ndarray, *, wrap: bool = False) -> Canvas:
        c = cls(palette, arr.shape[1], arr.shape[0], wrap=wrap)
        c.a = np.array(arr, dtype=np.uint8, copy=True)
        c.a[c.a[..., 3] == 0] = 0
        return c

    def copy(self, *, wrap: bool | None = None) -> Canvas:
        return Canvas.from_array(self.palette, self.a, wrap=self.wrap if wrap is None else wrap)

    def blank(self, w: int | None = None, h: int | None = None) -> Canvas:
        return Canvas(self.palette, w or self.w, h or self.h, wrap=self.wrap)

    # -- colors and coordinates ---------------------------------------

    def rgba(self, c: Color) -> tuple[int, int, int, int]:
        if c is None:
            return (0, 0, 0, 0)
        return (*self.palette[c], 255)

    def _xy(self, x: int, y: int) -> tuple[int, int] | None:
        if self.wrap:
            return x % self.w, y % self.h
        if 0 <= x < self.w and 0 <= y < self.h:
            return x, y
        return None

    def get(self, x: int, y: int) -> Color:
        """The color name at (x, y), None if transparent or outside."""
        p = self._xy(x, y)
        if p is None:
            return None
        px = self.a[p[1], p[0]]
        if px[3] == 0:
            return None
        return self.palette.name_of((int(px[0]), int(px[1]), int(px[2])))

    def opaque(self, x: int, y: int) -> bool:
        p = self._xy(x, y)
        return p is not None and bool(self.a[p[1], p[0], 3])

    def mask(self) -> np.ndarray:
        return self.a[..., 3] > 0

    # -- primitives ---------------------------------------------------

    def px(self, x: int, y: int, c: Color) -> None:
        p = self._xy(x, y)
        if p is not None:
            self.a[p[1], p[0]] = self.rgba(c)

    def points(self, pts: Iterable[tuple[int, int]], c: Color) -> None:
        for x, y in pts:
            self.px(x, y, c)

    def fill(self, c: Color) -> None:
        self.a[...] = self.rgba(c)

    def line(self, x0: int, y0: int, x1: int, y1: int, c: Color) -> None:
        """A 1 px Bresenham line, endpoints included."""
        for x, y in line_points(x0, y0, x1, y1):
            self.px(x, y, c)

    def polyline(self, pts: Sequence[tuple[int, int]], c: Color) -> None:
        for (x0, y0), (x1, y1) in itertools.pairwise(pts):
            self.line(x0, y0, x1, y1, c)

    def rect(self, x: int, y: int, w: int, h: int, c: Color, *, fill: bool = True) -> None:
        """A rectangle with its top-left at (x, y), w × h pixels."""
        for yy in range(y, y + h):
            for xx in range(x, x + w):
                if fill or yy in (y, y + h - 1) or xx in (x, x + w - 1):
                    self.px(xx, yy, c)

    def ellipse(self, x: int, y: int, w: int, h: int, c: Color, *, fill: bool = True) -> None:
        """An ellipse filling the w × h box at (x, y). Symmetric, no
        anti-aliasing; with fill=False only its 1 px rim is drawn."""
        m = ellipse_mask(w, h)
        if not fill:
            m = m & ~_erode(m)
        for yy, xx in zip(*np.nonzero(m), strict=True):
            self.px(x + int(xx), y + int(yy), c)

    def where(self, pred: Callable[[int, int, Color], bool], c: Color) -> None:
        """Set every pixel for which pred(x, y, current_color) is true."""
        hits = [(x, y) for y in range(self.h) for x in range(self.w) if pred(x, y, self.get(x, y))]
        for x, y in hits:
            self.px(x, y, c)

    def replace(self, old: Color, new: Color) -> None:
        """Recolor every pixel of one color."""
        self.where(lambda _x, _y, cur: cur == old, new)

    def swap(self, mapping: dict[str, Color]) -> Canvas:
        """A palette-swapped copy: {"leaf": "berry"} recolors leaf pixels."""
        out = self.copy()
        for y in range(self.h):
            for x in range(self.w):
                cur = self.get(x, y)
                if cur in mapping:
                    out.px(x, y, mapping[cur])
        return out

    # -- outline, mirror, stamp ---------------------------------------

    def outline(self, c: Color = "ink", *, corners: bool = False) -> None:
        """Add a 1 px outline around every opaque shape: transparent pixels
        touching an opaque one (4-neighbors, or 8 with corners) get c."""
        m = self.mask()
        edge = _dilate(m, corners=corners, wrap=self.wrap) & ~m
        self.a[edge] = self.rgba(c)

    def mirror(self) -> Canvas:
        """A copy flipped left to right."""
        return Canvas.from_array(self.palette, self.a[:, ::-1], wrap=self.wrap)

    def flip(self) -> Canvas:
        """A copy flipped top to bottom."""
        return Canvas.from_array(self.palette, self.a[::-1], wrap=self.wrap)

    def crop(self, x: int, y: int, w: int, h: int) -> Canvas:
        out = Canvas(self.palette, w, h)
        for yy in range(h):
            for xx in range(w):
                p = self._xy(x + xx, y + yy)
                if p is not None:
                    out.a[yy, xx] = self.a[p[1], p[0]]
        return out

    def stamp(self, src: Canvas, x: int = 0, y: int = 0, *, mirror: bool = False) -> None:
        """Copy src's opaque pixels with its top-left at (x, y). Wraps on a
        wrapping canvas, clips otherwise."""
        s = src.mirror() if mirror else src
        ys, xs = np.nonzero(s.mask())
        for sy, sx in zip(ys, xs, strict=True):
            p = self._xy(x + int(sx), y + int(sy))
            if p is not None:
                self.a[p[1], p[0]] = s.a[sy, sx]

    # -- output -------------------------------------------------------

    def image(self) -> Image.Image:
        return Image.fromarray(self.a, "RGBA")


def line_points(x0: int, y0: int, x1: int, y1: int) -> list[tuple[int, int]]:
    pts = []
    dx, dy = abs(x1 - x0), -abs(y1 - y0)
    sx, sy = (1 if x0 < x1 else -1), (1 if y0 < y1 else -1)
    err = dx + dy
    x, y = x0, y0
    while True:
        pts.append((x, y))
        if x == x1 and y == y1:
            return pts
        e2 = 2 * err
        if e2 >= dy:
            err += dy
            x += sx
        if e2 <= dx:
            err += dx
            y += sy


def ellipse_mask(w: int, h: int) -> np.ndarray:
    """Pixels inside an ellipse inscribed in a w × h box, sampled at pixel
    centers. Symmetric left-right and top-bottom."""
    yy, xx = np.mgrid[0:h, 0:w]
    cx, cy = (w - 1) / 2, (h - 1) / 2
    rx, ry = w / 2, h / 2
    return ((xx - cx) / rx) ** 2 + ((yy - cy) / ry) ** 2 <= 1.0 + 1e-9


def _shift(m: np.ndarray, dy: int, dx: int, wrap: bool) -> np.ndarray:
    if wrap:
        return np.roll(m, (dy, dx), axis=(0, 1))
    out = np.zeros_like(m)
    h, w = m.shape
    ys = slice(max(dy, 0), h + min(dy, 0))
    xs = slice(max(dx, 0), w + min(dx, 0))
    yo = slice(max(-dy, 0), h + min(-dy, 0))
    xo = slice(max(-dx, 0), w + min(-dx, 0))
    out[ys, xs] = m[yo, xo]
    return out


NEIGHBORS_4 = ((0, 1), (0, -1), (1, 0), (-1, 0))
NEIGHBORS_8 = (*NEIGHBORS_4, (1, 1), (1, -1), (-1, 1), (-1, -1))


def _dilate(m: np.ndarray, *, corners: bool = False, wrap: bool = False) -> np.ndarray:
    out = m.copy()
    for dy, dx in NEIGHBORS_8 if corners else NEIGHBORS_4:
        out |= _shift(m, dy, dx, wrap)
    return out


def _erode(m: np.ndarray) -> np.ndarray:
    out = m.copy()
    for dy, dx in NEIGHBORS_4:
        out &= _shift(m, dy, dx, False)
    return out


class Ctx:
    """What a draw script's `draw(ctx)` gets.

    - `ctx.palette["leaf"]`: an RGB color by name
    - `ctx.canvas(w, h, wrap=False)`: a new canvas (default: the asset size)
    - `ctx.rng(seed)`: a seeded random.Random, the only randomness allowed
    - `ctx.pxl("hero_head.pxl.yaml")`: frames of a pixel grid as canvases
      (path relative to the script's folder)
    - `ctx.module("ground.py")`: a helper module next to the script
    - `ctx.asset("terrain_base")`, `ctx.frame("terrain_base", "grass_a")`:
      finished assets, for composites
    """

    def __init__(
        self,
        asset: Asset,
        palette: NamedPalette,
        src: Path,
        finished: Callable[[str], tuple[np.ndarray, Asset]],
    ) -> None:
        self.asset = asset
        self.palette = palette
        self.src = src
        self._finished = finished
        self.size = asset.size
        self.cell = asset.grid.cell if asset.grid else asset.size

    def canvas(self, w: int | None = None, h: int | None = None, *, wrap: bool = False) -> Canvas:
        return Canvas(self.palette, w or self.size[0], h or self.size[1], wrap=wrap)

    def rng(self, seed: int | str = 0) -> random.Random:
        return random.Random(f"{self.asset.id}:{seed}")

    def pxl(self, name: str) -> list[Canvas]:
        path = (self.src.parent / name).resolve()
        if not path.exists():
            raise SourceError(f"{self.src}: part {name!r} not found at {path}")
        return [Canvas.from_array(self.palette, f) for f in load_pxl(path, self.palette).frames]

    def module(self, name: str) -> ModuleType:
        """A helper module next to the script (shared code for several
        draw scripts), loaded fresh for every render."""
        path = (self.src.parent / name).resolve()
        if not path.exists():
            raise SourceError(f"{self.src}: helper {name!r} not found at {path}")
        spec = importlib.util.spec_from_file_location(f"lorehallow_helper_{path.stem}", path)
        assert spec and spec.loader
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)
        return mod

    def asset_canvas(self, asset_id: str) -> Canvas:
        arr, _ = self._finished(asset_id)
        return Canvas.from_array(self.palette, arr)

    # Short alias: ctx.asset is the manifest entry, so finished art is ctx.art().
    art = asset_canvas

    def frame(self, asset_id: str, which: str | int) -> Canvas:
        arr, a = self._finished(asset_id)
        if a.grid is None:
            raise SourceError(f"{asset_id} has no frames")
        names = a.frame_names()
        k = names.index(which) if isinstance(which, str) else which
        fw, fh = a.grid.cell
        r, c = divmod(k, a.grid.columns)
        return Canvas.from_array(self.palette, arr[r * fh : (r + 1) * fh, c * fw : (c + 1) * fw])
