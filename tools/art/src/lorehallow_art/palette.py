"""The locked Lorehallow palette and nearest-color quantization."""

from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path

import numpy as np

from .color import RGB, hex_to_rgb, rgb_to_hex, srgb_to_oklab


@dataclass(frozen=True)
class Palette:
    colors: tuple[RGB, ...]
    _lab: np.ndarray = field(init=False, repr=False, compare=False)

    def __post_init__(self) -> None:
        if not self.colors:
            raise ValueError("palette is empty")
        if len(set(self.colors)) != len(self.colors):
            raise ValueError("palette has duplicate colors")
        object.__setattr__(self, "_lab", srgb_to_oklab(np.array(self.colors)))

    @classmethod
    def from_hex_lines(cls, text: str) -> Palette:
        colors: list[RGB] = []
        for raw in text.splitlines():
            line = raw.split(";")[0].strip()
            if not line or line.startswith("//"):
                continue
            colors.append(hex_to_rgb(line))
        return cls(tuple(colors))

    @classmethod
    def load(cls, path: Path) -> Palette:
        return cls.from_hex_lines(path.read_text())

    def __len__(self) -> int:
        return len(self.colors)

    def __contains__(self, rgb: object) -> bool:
        return rgb in self.colors

    @property
    def lab(self) -> np.ndarray:
        return self._lab

    def nearest(self, rgb: np.ndarray) -> np.ndarray:
        """Index of the nearest palette color (OKLab distance) for each
        color in an (..., 3) array."""
        lab = srgb_to_oklab(rgb)
        d = np.linalg.norm(lab[..., None, :] - self._lab, axis=-1)
        return np.argmin(d, axis=-1)

    def as_array(self) -> np.ndarray:
        return np.array(self.colors, dtype=np.uint8)

    def hex_list(self) -> list[str]:
        return [rgb_to_hex(c) for c in self.colors]
