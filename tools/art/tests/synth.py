"""Synthetic fixtures: known logical pixel art, rendered the way image
generators render it (non-integer upscaling, offsets, jitter, soft edges,
noise and colors slightly off the palette)."""

from __future__ import annotations

import numpy as np
from PIL import Image, ImageFilter

KEY = (255, 0, 255)


def random_art(
    rng: np.random.Generator, w: int, h: int, colors: list[tuple[int, int, int]]
) -> np.ndarray:
    """Blocky random art: rectangles of palette colors on the key color."""
    art = np.zeros((h, w, 3), dtype=np.uint8)
    art[:] = KEY
    for _ in range(max(4, (w * h) // 24)):
        x, y = rng.integers(0, w), rng.integers(0, h)
        rw, rh = rng.integers(1, max(2, w // 4)), rng.integers(1, max(2, h // 4))
        art[y : y + rh, x : x + rw] = colors[rng.integers(0, len(colors))]
    return art


def render(
    art: np.ndarray,
    scale: float,
    *,
    offset: tuple[int, int] = (0, 0),
    pad: tuple[int, int] = (0, 0),
    jitter: float = 0.0,
    blur: float = 0.0,
    noise: float = 0.0,
    shift: int = 0,
    rng: np.random.Generator | None = None,
) -> np.ndarray:
    """Upscale art by a (possibly non-integer) scale like a generator would.

    offset: key-colored margin before the art (x, y), in screen pixels.
    pad: extra key-colored margin after the art.
    jitter: random displacement of each block boundary, in screen pixels.
    blur: Gaussian blur radius (soft, anti-aliased edges).
    noise: standard deviation of per-pixel noise.
    shift: constant added to every channel (off-palette colors).
    """
    rng = rng or np.random.default_rng(0)
    h, w, _ = art.shape
    bx = np.arange(w + 1, dtype=np.float64) * scale
    by = np.arange(h + 1, dtype=np.float64) * scale
    if jitter:
        bx[1:-1] += rng.uniform(-jitter, jitter, w - 1)
        by[1:-1] += rng.uniform(-jitter, jitter, h - 1)
    W = int(np.ceil(w * scale)) + offset[0] + pad[0]
    H = int(np.ceil(h * scale)) + offset[1] + pad[1]
    xs = np.arange(W) - offset[0] + 0.5
    ys = np.arange(H) - offset[1] + 0.5
    ci = np.searchsorted(bx, xs, side="right") - 1
    ri = np.searchsorted(by, ys, side="right") - 1
    out = np.zeros((H, W, 3), dtype=np.float64)
    out[:] = KEY
    vx = (ci >= 0) & (ci < w)
    vy = (ri >= 0) & (ri < h)
    sub = art[np.clip(ri, 0, h - 1)][:, np.clip(ci, 0, w - 1)]
    mask = vy[:, None] & vx[None, :]
    out[mask] = sub[mask]
    out += shift
    if noise:
        out += rng.normal(0, noise, out.shape)
    out = np.clip(out, 0, 255).astype(np.uint8)
    if blur:
        out = np.asarray(Image.fromarray(out).filter(ImageFilter.GaussianBlur(blur)))
    return out
