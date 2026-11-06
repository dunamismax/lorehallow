"""Placeholder art, so code never waits on real art (AGENTS.md section 8).

For every manifest asset that has no processed output yet, write
`<target>_placeholder.png` (and a sidecar for tilesets and sheets) at the
right size, using palette colors. When the real asset exists, its
placeholder is removed. Output is deterministic.
"""

from __future__ import annotations

import json
from pathlib import Path

import numpy as np
from PIL import Image

from .manifest import Asset, load_manifest
from .palette import Palette
from .sheet import frame_records, slice_frames

INK = (0x1A, 0x14, 0x20)


def placeholder_path(processed: Path, asset: Asset) -> Path:
    target = processed / asset.target
    return target.with_name(target.stem + "_placeholder.png")


def _cell(w: int, h: int, color: tuple[int, int, int], index: int) -> np.ndarray:
    """A flat cell with an ink border and a diagonal mark that differs per index."""
    a = np.zeros((h, w, 4), dtype=np.uint8)
    a[...] = (*color, 255)
    a[0, :], a[-1, :], a[:, 0], a[:, -1] = (*INK, 255), (*INK, 255), (*INK, 255), (*INK, 255)
    for k in range(index % 4 + 1):
        y = 3 + k * 3
        if y < h - 2:
            a[y, 3 : min(w - 3, 3 + 2 * (k + 1))] = (*INK, 255)
    return a


def _figure(
    w: int, h: int, body: tuple[int, int, int], skin: tuple[int, int, int], baseline: int
) -> np.ndarray:
    """A simple standing figure whose outline's lowest row is the baseline."""
    a = np.zeros((h, w, 4), dtype=np.uint8)
    cx = w // 2
    bottom = baseline - 1  # last row of fill; the outline sits on the baseline
    top = max(1, bottom - 28)
    height = bottom - top + 1
    head_h = max(1, round(height * 0.45))
    leg_h = max(1, round(height * 0.15))
    half = max(1, min(7, w // 2 - 2))
    head = (slice(top, top + head_h), slice(cx - max(1, half - 1), cx + max(1, half - 1)))
    torso = (slice(top + head_h, bottom - leg_h + 1), slice(cx - half, cx + half))
    legs = (
        slice(bottom - leg_h + 1, bottom + 1),
        slice(cx - max(1, half - 2), cx + max(1, half - 2)),
    )
    for region, color in ((head, skin), (torso, body), (legs, INK)):
        a[region] = (*color, 255)
    # 1 px ink outline around the silhouette.
    opaque = a[..., 3] > 0
    edge = np.zeros_like(opaque)
    edge[1:] |= opaque[:-1]
    edge[:-1] |= opaque[1:]
    edge[:, 1:] |= opaque[:, :-1]
    edge[:, :-1] |= opaque[:, 1:]
    a[edge & ~opaque] = (*INK, 255)
    return a


def render(asset: Asset, palette: Palette) -> np.ndarray:
    w, h = asset.size
    colors = [c for c in palette.colors if c != INK] or list(palette.colors)
    if asset.grid:
        fw, fh = asset.grid.cell
        count = asset.frames.count if asset.frames else asset.grid.columns * asset.grid.rows
        out = np.zeros((h, w, 4), dtype=np.uint8)
        for k in range(count):
            r, c = divmod(k, asset.grid.columns)
            if asset.kind == "sheet":
                base = (
                    asset.frames.baseline
                    if asset.frames and asset.frames.baseline is not None
                    else fh - 3
                )
                cell = _figure(
                    fw, fh, colors[(r * 3 + 7) % len(colors)], colors[-6 % len(colors)], base
                )
            else:
                cell = _cell(fw, fh, colors[(k * 5 + 3) % len(colors)], k)
            out[r * fh : (r + 1) * fh, c * fw : (c + 1) * fw] = cell
        return out
    if asset.kind == "sprite" and h > w:
        base = asset.pivot[1] if asset.pivot else h - 3
        return _figure(w, h, colors[16 % len(colors)], colors[-6 % len(colors)], base)
    return _cell(w, h, colors[(len(asset.id) * 7) % len(colors)], 0)


def write_placeholders(root: Path) -> list[str]:
    processed = root / "art" / "processed"
    palette_cache: dict[str, Palette] = {}
    log = []
    for mpath in sorted((root / "art" / "batches").glob("batch-*/manifest.yaml")):
        manifest = load_manifest(mpath)
        for asset in manifest.assets:
            real = processed / asset.target
            ph = placeholder_path(processed, asset)
            sidecar = ph.with_suffix(".json")
            if real.exists():
                for p in (ph, sidecar):
                    if p.exists():
                        p.unlink()
                        log.append(f"remove {p.relative_to(root)}")
                continue
            if asset.palette not in palette_cache:
                palette_cache[asset.palette] = Palette.load(root / asset.palette)
            img = render(asset, palette_cache[asset.palette])
            ph.parent.mkdir(parents=True, exist_ok=True)
            Image.fromarray(img, "RGBA").save(ph, optimize=True)
            log.append(f"write  {ph.relative_to(root)}")
            if asset.grid:
                data = {
                    "id": asset.id,
                    "kind": asset.kind,
                    "size": list(asset.size),
                    "frame_size": list(asset.grid.cell),
                    "columns": asset.grid.columns,
                    "rows": asset.grid.rows,
                    "frames": frame_records(slice_frames(img, asset), asset),
                    "pivot": list(asset.pivot) if asset.pivot else None,
                    "baseline": asset.frames.baseline if asset.frames else None,
                    "source": "placeholder",
                    "batch": manifest.batch,
                }
                sidecar.write_text(json.dumps(data, indent=2) + "\n")
    return log
