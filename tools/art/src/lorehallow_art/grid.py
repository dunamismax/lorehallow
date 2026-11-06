"""Grid detection: find the real art-pixel block size and offset in a
generated "pixel art" image.

Image generators draw pixel art as blocks of roughly N screen pixels, but
N is rarely an integer, the blocks drift a little, and the edges are soft.
We find the grid from where the color changes. Summing the color
difference between neighboring columns gives a 1-D "edge profile" with a
peak at every block boundary: a pulse train whose spacing is the block
size. A pulse train with period p has strong Fourier components at
frequency 1/p and its harmonics 2/p, 3/p, ... but not at 1/(2p). So:

1. Measure the Fourier magnitude of the profile for each candidate period.
2. The true period and its whole fractions (p/2, p/3) score high; its
   multiples do not. Blur weakens the fractions further.
3. Among near-ties keep the largest period, or with a hint (the expected
   block size from the manifest's logical size) the one nearest the hint.
4. The phase of that Fourier component gives the offset of the first line.

Rows are handled the same way.
"""

from __future__ import annotations

from dataclasses import dataclass

import numpy as np


@dataclass(frozen=True)
class Axis:
    period: float
    offset: float
    score: float

    def cells(self, length: int) -> int:
        """Number of cells whose centers fall inside an axis of this length.

        The offset may be slightly negative: the first cell can start just
        before the image edge.
        """
        return int(np.floor((length - self.offset) / self.period - 0.5)) + 1

    def start(self, i: int) -> float:
        return self.offset + i * self.period


@dataclass(frozen=True)
class Grid:
    x: Axis
    y: Axis

    def shape(self, width: int, height: int) -> tuple[int, int]:
        """(columns, rows) of whole cells in an image of this size."""
        return self.x.cells(width), self.y.cells(height)


def edge_profile(img: np.ndarray, axis: int) -> np.ndarray:
    """Color change between neighbors along an axis (1 = columns, 0 = rows).

    e[i] is the change between positions i-1 and i, so a block boundary at
    pixel i shows up at e[i]. e[0] is 0. Normalized to a maximum of 1.
    """
    a = img.astype(np.float64)
    d = np.abs(np.diff(a, axis=axis)).sum(axis=-1)
    prof = d.sum(axis=0 if axis == 1 else 1)
    prof = np.concatenate([[0.0], prof])
    m = prof.max()
    return prof / m if m > 0 else prof


def fourier(prof: np.ndarray, periods: np.ndarray) -> tuple[np.ndarray, np.ndarray]:
    """Normalized magnitude and offset of the profile's component at 1/p."""
    x = np.arange(len(prof))
    total = prof.sum()
    z = np.exp(-2j * np.pi * x[None, :] / periods[:, None]) @ prof
    mag = np.abs(z) / total if total > 0 else np.zeros(len(periods))
    offset = np.mod(-np.angle(z) / (2 * np.pi) * periods, periods)
    return mag, offset


# Candidates scoring at least this fraction of the best are ties.
TIE = 0.8


def detect_axis(
    prof: np.ndarray,
    min_period: float = 2.0,
    max_period: float | None = None,
    hint: float | None = None,
) -> Axis:
    """Find the period and offset that best explain an edge profile."""
    n = len(prof)
    if not prof.any():
        # No edges at all: trust the hint, or call the whole axis one cell.
        p = hint if hint else float(n)
        return Axis(period=p, offset=0.0, score=0.0)
    if hint:
        lo, hi = max(min_period, hint / 1.6), hint * 1.6
    else:
        lo, hi = min_period, max_period or max(min_period + 1, n / 2)
    if max_period:
        hi = min(hi, max_period)

    # A Fourier peak is about p*p/n wide, so step finely at short periods.
    periods = [lo]
    while periods[-1] < hi:
        periods.append(periods[-1] + max(0.002, periods[-1] ** 2 / (4 * n)))
    periods = np.array(periods)
    mag, _ = fourier(prof, periods)
    top = mag.max()
    # Local maxima only, so one broad peak doesn't produce many "ties".
    is_peak = np.r_[True, mag[1:] >= mag[:-1]] & np.r_[mag[:-1] >= mag[1:], True]
    ties = periods[is_peak & (mag >= TIE * top)]
    p0 = float(min(ties, key=lambda p: abs(np.log(p / hint)))) if hint else float(ties.max())

    width = max(0.01, p0 * p0 / (2 * n))
    fine = np.linspace(max(lo, p0 - width), p0 + width, 201)
    fmag, foff = fourier(prof, fine)
    i = int(np.argmax(fmag))
    period, offset = float(fine[i]), float(foff[i])
    # Put the offset in (-p/2, p/2]: a line just before 0 means the first
    # cell starts at the image edge, not one cell later.
    if offset > period / 2:
        offset -= period
    return Axis(period=round(period, 4), offset=round(offset, 3), score=float(fmag[i]))


def detect_grid(
    img: np.ndarray,
    hint: tuple[float, float] | None = None,
    max_period: float | None = None,
) -> Grid:
    """Detect the art-pixel grid of an (H, W, 3) image."""
    hx, hy = hint if hint else (None, None)
    x = detect_axis(edge_profile(img, axis=1), hint=hx, max_period=max_period)
    y = detect_axis(edge_profile(img, axis=0), hint=hy, max_period=max_period)
    return Grid(x=x, y=y)


def fixed_grid(period_x: float, period_y: float, offset_x: float = 0, offset_y: float = 0) -> Grid:
    """A grid given explicitly, for example by a manifest `grid_override`."""
    return Grid(Axis(period_x, offset_x, 0.0), Axis(period_y, offset_y, 0.0))
