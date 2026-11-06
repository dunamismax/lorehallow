"""Favicon set and client app icon from the processed emblem (BUILD.md 0.9).

Pixel art scales only by whole numbers, so each size is an integer
multiple of the 32×32 emblem, centered on an opaque background where the
size isn't an exact multiple (180 = 5×32 + 20). The 16 px favicon uses the
hand-reduced 16×16 emblem, since halving pixel art destroys it.

Until the real emblem is processed, the placeholders are used and every
output name gets a `_placeholder` suffix.
"""

from __future__ import annotations

from pathlib import Path

from PIL import Image

# Opaque background for icons that platforms show without transparency.
BACKGROUND = (0x1F, 0x2A, 0x44, 255)  # deep dusk
# size -> (source, scale, opaque background)
SIZES: dict[int, tuple[str, int, bool]] = {
    16: ("emblem_16", 1, False),
    32: ("emblem", 1, False),
    180: ("emblem", 5, True),  # apple-touch-icon
    192: ("emblem", 6, False),
    512: ("emblem", 16, False),
}
APP_ICON_SCALE = 8  # 256×256 for the Godot client


def _source(icons: Path, name: str) -> tuple[Path, bool] | None:
    real = icons / f"{name}.png"
    if real.exists():
        return real, False
    ph = icons / f"{name}_placeholder.png"
    return (ph, True) if ph.exists() else None


def _scaled(img: Image.Image, scale: int, size: int, opaque: bool) -> Image.Image:
    big = img.resize((img.width * scale, img.height * scale), Image.NEAREST)
    canvas = Image.new("RGBA", (size, size), BACKGROUND if opaque else (0, 0, 0, 0))
    off = ((size - big.width) // 2, (size - big.height) // 2)
    canvas.alpha_composite(big, off)
    return canvas


def export(root: Path) -> list[str]:
    icons = root / "art" / "processed" / "icons"
    log: list[str] = []
    written: list[Path] = []
    for size, (name, scale, opaque) in SIZES.items():
        src = _source(icons, name)
        if src is None:
            log.append(f"skip   favicon_{size}: no {name} art or placeholder")
            continue
        path, placeholder = src
        with Image.open(path) as img:
            out = _scaled(img.convert("RGBA"), scale, size, opaque)
        dest = icons / f"favicon_{size}{'_placeholder' if placeholder else ''}.png"
        out.save(dest, optimize=True)
        written.append(dest)
    src = _source(icons, "emblem")
    if src:
        path, placeholder = src
        with Image.open(path) as img:
            n = img.width * APP_ICON_SCALE
            out = _scaled(img.convert("RGBA"), APP_ICON_SCALE, n, False)
        dest = icons / f"app_icon{'_placeholder' if placeholder else ''}.png"
        out.save(dest, optimize=True)
        written.append(dest)
    # Drop placeholder outputs once the real ones exist.
    for p in sorted(icons.glob("*_placeholder.png")):
        if p.stem.startswith(("favicon_", "app_icon")):
            real = p.with_name(p.name.replace("_placeholder", ""))
            if real in written:
                p.unlink()
                log.append(f"remove {p.relative_to(root)}")
    log.extend(f"write  {p.relative_to(root)}" for p in written)
    return log
