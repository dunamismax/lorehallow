"""Layout guides: flat two-color maps attached to prompts for transition tilesets.

Image generators follow a picture of the layout far better than a written
description of it. For an asset with a `guide: {outer, inner}` entry in its
manifest, `make art-generate` writes art/batches/<batch>/guides/<id>.png: the
tileset at its logical size with the outer ground (grass) in one flat color
and the inner ground (dirt, stone) in another, upscaled 8x with no grid lines
(drawn lines between tiles get copied into the art).

Tiles are recognized by the transition frame names in STYLE.md: edge_tl ...
edge_br, center, inner_tl ... inner_br, and the row 3 variants edge_t_b,
center_b and edge_b_b. Every border between the two grounds meets the tile
edges BAND pixels in, so the tiles connect.
"""

from __future__ import annotations

import io
from pathlib import Path

import numpy as np
from PIL import Image

from .color import hex_to_rgb
from .manifest import Asset, load_manifest

SCALE = 8
BAND = 10  # width of the outer ground along an edge, in art pixels
RADIUS = 5  # rounding of the patch's corners

# Which sides of the tile have a band of outer ground, for each frame name.
EDGES: dict[str, str] = {
    "edge_tl": "tl",
    "edge_t": "t",
    "edge_tr": "tr",
    "edge_l": "l",
    "center": "",
    "edge_r": "r",
    "edge_bl": "bl",
    "edge_b": "b",
    "edge_br": "br",
    "edge_t_b": "t",
    "center_b": "",
    "edge_b_b": "b",
}
# Inner corners: outer ground only in this corner.
BITES = {"inner_tl": "tl", "inner_tr": "tr", "inner_bl": "bl", "inner_br": "br"}


def _rounded_rect(n: int, x0: int, y0: int, x1: int, y1: int, r: int) -> np.ndarray:
    """Boolean mask of the rectangle [x0, x1) x [y0, y1) with corners
    rounded by r, inside an n x n tile."""
    ys, xs = np.mgrid[0:n, 0:n] + 0.5
    inside = (xs >= x0) & (xs < x1) & (ys >= y0) & (ys < y1)
    for cx, cy, sx, sy in ((x0 + r, y0 + r, -1, -1), (x1 - r, y0 + r, 1, -1),
                           (x0 + r, y1 - r, -1, 1), (x1 - r, y1 - r, 1, 1)):  # fmt: skip
        corner = (sx * (xs - cx) > 0) & (sy * (ys - cy) > 0)
        inside &= ~(corner & ((xs - cx) ** 2 + (ys - cy) ** 2 > r * r))
    return inside


def inner_mask(name: str, n: int) -> np.ndarray:
    """True where the tile shows the inner ground."""
    if name in BITES:
        side = BITES[name]
        # The outer ground's bite is the corner of a patch of outer ground
        # whose edges continue the neighboring edge tiles' bands.
        x0, x1 = (-n, BAND) if "l" in side else (n - BAND, 2 * n)
        y0, y1 = (-n, BAND) if "t" in side else (n - BAND, 2 * n)
        return ~_rounded_rect(n, x0, y0, x1, y1, RADIUS)
    if name not in EDGES:
        raise ValueError(f"no guide layout for frame {name!r}")
    side = EDGES[name]
    x0 = BAND if "l" in side else -n
    x1 = n - BAND if "r" in side else 2 * n
    y0 = BAND if "t" in side else -n
    y1 = n - BAND if "b" in side else 2 * n
    return _rounded_rect(n, x0, y0, x1, y1, RADIUS)


def render_guide(asset: Asset) -> Image.Image:
    if not asset.guide or not asset.grid:
        raise ValueError(f"{asset.id} has no guide or grid")
    fw, fh = asset.grid.cell
    if fw != fh:
        raise ValueError(f"{asset.id}: guides need square tiles")
    outer = np.array(hex_to_rgb(asset.guide[0]), dtype=np.uint8)
    inner = np.array(hex_to_rgb(asset.guide[1]), dtype=np.uint8)
    w, h = asset.size
    out = np.zeros((h, w, 3), dtype=np.uint8)
    out[:] = outer
    for k, name in enumerate(asset.frame_names()):
        r, c = divmod(k, asset.grid.columns)
        tile = out[r * fh : (r + 1) * fh, c * fw : (c + 1) * fw]
        tile[inner_mask(name, fw)] = inner
    big = Image.fromarray(out, "RGB").resize((w * SCALE, h * SCALE), Image.NEAREST)
    return big


def guide_outputs(root: Path) -> dict[Path, bytes]:
    out: dict[Path, bytes] = {}
    for mpath in sorted((root / "art" / "batches").glob("batch-*/manifest.yaml")):
        manifest = load_manifest(mpath)
        for asset in manifest.assets:
            if not asset.guide:
                continue
            buf = io.BytesIO()
            render_guide(asset).save(buf, format="PNG", optimize=True)
            out[mpath.parent / "guides" / f"{asset.id}.png"] = buf.getvalue()
    return out
