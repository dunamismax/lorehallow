"""Turn a detected grid into true pixel art: sample one color per cell,
make the key color transparent, and snap every color to the palette."""

from __future__ import annotations

from dataclasses import dataclass

import numpy as np

from .color import RGB, srgb_to_oklab
from .grid import Grid
from .palette import Palette

# Cells sample only their middle: edges are where blur and drift live.
INNER = 0.5
# "mode" sampling needs the most common color to cover this share of the
# inner pixels; otherwise the per-channel median is used.
MODE_SHARE = 0.3


def downsample(
    img: np.ndarray,
    grid: Grid,
    method: str = "mode",
    palette: Palette | None = None,
    key: RGB | None = None,
) -> tuple[np.ndarray, np.ndarray]:
    """Sample one color per grid cell; see sample_cells."""
    s = sample_cells(img, grid, method, palette, key)
    return s.rgb, s.alpha


@dataclass
class Sampled:
    rgb: np.ndarray
    alpha: np.ndarray
    # Mean share of each art cell's inner pixels that voted for the winning
    # color (palette "mode" only, else 1.0; key-colored cells don't count).
    # A grid that lines up with the art gives nearly uniform cells; a
    # misaligned one mixes neighbors.
    purity: float


def sample_cells(
    img: np.ndarray,
    grid: Grid,
    method: str = "mode",
    palette: Palette | None = None,
    key: RGB | None = None,
) -> Sampled:
    """Sample one color per grid cell.

    img is (H, W, 3) or (H, W, 4) uint8. Returns (rgb, alpha): an (rows,
    cols, 3) uint8 color array and an (rows, cols) bool array that is False
    where the source was transparent.

    "mode" with a palette: every inner pixel votes for its nearest palette
    color (or the key color), and the cell takes the average of the pixels
    in the winning group. Noise makes exact colors unique, so voting by
    class is what makes the mode robust. Without a palette, "mode" uses
    the most common exact color, falling back to the median. "center"
    takes the middle pixel.
    """
    h, w = img.shape[:2]
    cols, rows = grid.shape(w, h)
    if cols <= 0 or rows <= 0:
        raise ValueError(f"grid {grid} leaves no cells in a {w}x{h} image")
    has_alpha = img.shape[2] == 4
    classes = None
    if method == "mode" and palette is not None:
        targets = (
            palette.lab if key is None else np.vstack([palette.lab, srgb_to_oklab(np.array(key))])
        )
        lab = srgb_to_oklab(img[..., :3])
        classes = np.empty((h, w), dtype=np.int16)
        for y in range(0, h, 64):  # in strips to bound memory
            d = np.linalg.norm(lab[y : y + 64, :, None, :] - targets, axis=-1)
            classes[y : y + 64] = np.argmin(d, axis=-1)
    out = np.zeros((rows, cols, 3), dtype=np.uint8)
    alpha = np.ones((rows, cols), dtype=bool)
    shares: list[float] = []
    xr = [_inner_range(grid.x.start(i), grid.x.period, w) for i in range(cols)]
    yr = [_inner_range(grid.y.start(j), grid.y.period, h) for j in range(rows)]
    for j, (y0, y1) in enumerate(yr):
        for i, (x0, x1) in enumerate(xr):
            block = img[y0:y1, x0:x1].reshape(-1, img.shape[2])
            cls = classes[y0:y1, x0:x1].reshape(-1) if classes is not None else None
            if has_alpha:
                opaque = block[:, 3] >= 128
                if opaque.mean() < 0.5:
                    alpha[j, i] = False
                    continue
                block = block[opaque]
                cls = cls[opaque] if cls is not None else None
            if cls is not None:
                votes = np.bincount(cls)
                winner = votes.argmax()
                if key is None or winner < len(palette.colors):
                    shares.append(float(votes[winner]) / len(cls))
                out[j, i] = block[cls == winner, :3].mean(axis=0).round().astype(np.uint8)
            else:
                out[j, i] = _sample(block[:, :3], method)
    return Sampled(out, alpha, float(np.mean(shares)) if shares else 1.0)


def _inner_range(start: float, period: float, limit: int) -> tuple[int, int]:
    """Pixel range of the middle part of a cell, clamped to the image."""
    margin = period * (1 - INNER) / 2
    a = int(np.floor(start + margin))
    b = int(np.ceil(start + period - margin))
    a, b = max(0, a), min(limit, b)
    if b <= a:  # very small cells: use the nearest pixel to the center
        c = int(np.clip(np.floor(start + period / 2), 0, limit - 1))
        a, b = c, c + 1
    return a, b


def _sample(pixels: np.ndarray, method: str) -> np.ndarray:
    if method == "center":
        return pixels[len(pixels) // 2]
    if method != "mode":
        raise ValueError(f"unknown downsample method {method!r}")
    colors, counts = np.unique(pixels, axis=0, return_counts=True)
    k = int(np.argmax(counts))
    if counts[k] >= MODE_SHARE * len(pixels):
        return colors[k]
    return np.median(pixels, axis=0).round().astype(np.uint8)


@dataclass(frozen=True)
class KeyResult:
    opaque: np.ndarray  # (rows, cols) bool
    keyed: int  # cells made transparent by the key color


def remove_key(
    rgb: np.ndarray, alpha: np.ndarray, key: RGB, palette: Palette, tolerance: float
) -> KeyResult:
    """Make the key color and its fringe transparent.

    A cell is key when it is within `tolerance` of the key color in OKLab,
    or when it is closer to the key than to any palette color (the pinkish
    fringe left where the generator blended a sprite into the background).
    """
    lab = srgb_to_oklab(rgb)
    key_lab = srgb_to_oklab(np.array(key))
    d_key = np.linalg.norm(lab - key_lab, axis=-1)
    d_pal = np.linalg.norm(lab[..., None, :] - palette.lab, axis=-1).min(axis=-1)
    is_key = (d_key <= tolerance) | (d_key < d_pal)
    opaque = alpha & ~is_key
    return KeyResult(opaque=opaque, keyed=int((alpha & is_key).sum()))


@dataclass(frozen=True)
class QuantizeResult:
    rgba: np.ndarray  # (rows, cols, 4) uint8
    changed: int  # opaque cells whose color was not already in the palette
    max_distance: float  # largest OKLab move made, a sign of how far off the art was


def quantize(rgb: np.ndarray, opaque: np.ndarray, palette: Palette) -> QuantizeResult:
    """Snap every opaque cell to the nearest palette color. No dithering."""
    idx = palette.nearest(rgb)
    snapped = palette.as_array()[idx]
    changed_mask = opaque & np.any(snapped != rgb, axis=-1)
    dist = np.linalg.norm(srgb_to_oklab(rgb) - palette.lab[idx], axis=-1)
    rgba = np.zeros((*rgb.shape[:2], 4), dtype=np.uint8)
    rgba[..., :3] = np.where(opaque[..., None], snapped, 0)
    rgba[..., 3] = np.where(opaque, 255, 0)
    return QuantizeResult(
        rgba=rgba,
        changed=int(changed_mask.sum()),
        max_distance=float(dist[opaque].max()) if opaque.any() else 0.0,
    )


@dataclass(frozen=True)
class FitResult:
    rgba: np.ndarray
    trimmed: tuple[int, int]  # (columns, rows) of margin removed
    padded: tuple[int, int]  # (columns, rows) of transparent padding added


class FitError(ValueError):
    pass


def fit(
    rgba: np.ndarray, width: int, height: int, anchor: tuple[int, int] | None = None
) -> FitResult:
    """Fit the sampled art to the logical size.

    Generators often leave a margin of key color, so the grid finds a few
    more cells than the art has. Extra rows and columns are trimmed only
    where they are fully transparent, keeping the content centered, or,
    with an anchor (x, y), putting the content's middle column on x and its
    lowest row on y (a sprite's pivot). One missing row or column (a lost
    partial edge cell) is padded back; more than that means the grid or the
    image is wrong.
    """
    h, w = rgba.shape[:2]
    if anchor is not None:
        return _fit_anchored(rgba, width, height, anchor)
    out = rgba
    trim = [0, 0]
    pad = [0, 0]
    for axis, (have, want) in enumerate(((w, width), (h, height))):
        if have == want:
            continue
        opaque = out[..., 3] > 0
        line = opaque.any(axis=0) if axis == 0 else opaque.any(axis=1)
        if have > want:
            nz = np.flatnonzero(line)
            lo, hi = (int(nz[0]), int(nz[-1]) + 1) if len(nz) else (0, 0)
            if hi - lo > want:
                raise FitError(
                    f"content is {hi - lo} px along {'xy'[axis]}, larger than the logical {want} px"
                )
            spare = want - (hi - lo)
            # Any start in [smin, smax] keeps all content; prefer centered.
            smin, smax = max(0, hi - want), min(lo, have - want)
            start = int(np.clip(lo - spare // 2, smin, smax))
            out = out[:, start : start + want] if axis == 0 else out[start : start + want]
            trim[axis] = have - want
        else:
            if want - have > 1:
                raise FitError(f"found {have} cells along {'xy'[axis]}, expected {want}")
            shape = list(out.shape)
            shape[1 - axis] = want
            padded = np.zeros(shape, dtype=out.dtype)
            if axis == 0:
                padded[:, :have] = out
            else:
                padded[:have] = out
            out = padded
            pad[axis] = want - have
    return FitResult(rgba=out, trimmed=(trim[0], trim[1]), padded=(pad[0], pad[1]))


def _fit_anchored(rgba: np.ndarray, width: int, height: int, anchor: tuple[int, int]) -> FitResult:
    """Place the content's bounding box so that it is centered on anchor x
    (symmetric about the line x = anchor[0]) with its lowest row on anchor y."""
    h, w = rgba.shape[:2]
    opaque = rgba[..., 3] > 0
    if not opaque.any():
        return fit(rgba, width, height)
    ys, xs = np.nonzero(opaque)
    y0, y1, x0, x1 = int(ys.min()), int(ys.max()) + 1, int(xs.min()), int(xs.max()) + 1
    bw, bh = x1 - x0, y1 - y0
    for axis, (size, want) in enumerate(((bw, width), (bh, height))):
        if size > want:
            raise FitError(
                f"content is {size} px along {'xy'[axis]}, larger than the logical {want} px"
            )
    left = int(np.clip(anchor[0] - bw // 2, 0, width - bw))
    top = int(np.clip(anchor[1] + 1 - bh, 0, height - bh))
    out = np.zeros((height, width, 4), dtype=rgba.dtype)
    out[top : top + bh, left : left + bw] = rgba[y0:y1, x0:x1]
    return FitResult(
        rgba=out,
        trimmed=(max(0, w - width), max(0, h - height)),
        padded=(max(0, width - w), max(0, height - h)),
    )
