"""Color conversion: sRGB to OKLab, a perceptual space where Euclidean
distance roughly matches how different two colors look."""

from __future__ import annotations

import numpy as np

RGB = tuple[int, int, int]


def hex_to_rgb(value: str) -> RGB:
    """Parse "#RRGGBB" or "RRGGBB"."""
    v = value.strip().lstrip("#")
    if len(v) != 6:
        raise ValueError(f"expected a 6-digit hex color, got {value!r}")
    try:
        return (int(v[0:2], 16), int(v[2:4], 16), int(v[4:6], 16))
    except ValueError as e:
        raise ValueError(f"invalid hex color {value!r}") from e


def rgb_to_hex(rgb: RGB) -> str:
    return "#{:02X}{:02X}{:02X}".format(*rgb)


def srgb_to_oklab(rgb: np.ndarray) -> np.ndarray:
    """Convert an (..., 3) array of 0-255 sRGB values to OKLab."""
    c = np.asarray(rgb, dtype=np.float64) / 255.0
    lin = np.where(c <= 0.04045, c / 12.92, ((c + 0.055) / 1.055) ** 2.4)
    m1 = np.array(
        [
            [0.4122214708, 0.5363325363, 0.0514459929],
            [0.2119034982, 0.6806995451, 0.1073969566],
            [0.0883024619, 0.2817188376, 0.6299787005],
        ]
    )
    m2 = np.array(
        [
            [0.2104542553, 0.7936177850, -0.0040720468],
            [1.9779984951, -2.4285922050, 0.4505937099],
            [0.0259040371, 0.7827717662, -0.8086757660],
        ]
    )
    lms = lin @ m1.T
    return np.cbrt(lms) @ m2.T
