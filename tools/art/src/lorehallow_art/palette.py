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


def palette_names(text: str) -> list[str]:
    """The comment after each color line (up to a colon) is its name."""
    names = []
    for raw in text.splitlines():
        line, _, comment = raw.partition(";")
        line = line.strip()
        if line and not line.startswith("//"):
            names.append(comment.strip().split(":")[0].strip() or line.lstrip("#").upper())
    return names


class NamedPalette:
    """Palette colors by name, as hand-drawn sources refer to them.

    Names come from the comments in palette.hex (`7a8e3c ; leaf`). A file
    without names falls back to the upper-case hex value (`7A8E3C`).
    """

    def __init__(self, palette: Palette, names: list[str]) -> None:
        if len(names) != len(palette):
            raise ValueError("one name per palette color is required")
        dupes = sorted({n for n in names if names.count(n) > 1})
        if dupes:
            raise ValueError(f"duplicate palette names: {dupes}")
        self.palette = palette
        self.names = tuple(names)
        self._by_name = dict(zip(names, palette.colors, strict=True))
        self._by_rgb = {c: n for n, c in self._by_name.items()}

    @classmethod
    def from_text(cls, text: str) -> NamedPalette:
        return cls(Palette.from_hex_lines(text), palette_names(text))

    @classmethod
    def load(cls, path: Path) -> NamedPalette:
        return cls.from_text(path.read_text())

    def __contains__(self, name: object) -> bool:
        return name in self._by_name

    def __getitem__(self, name: str) -> RGB:
        try:
            return self._by_name[name]
        except KeyError:
            raise KeyError(f"unknown palette color {name!r}") from None

    def name_of(self, rgb: RGB) -> str | None:
        return self._by_rgb.get(rgb)
