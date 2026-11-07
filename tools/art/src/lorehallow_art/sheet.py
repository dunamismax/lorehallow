"""Sheet slicing and checks: frame count, frame size and foot baseline."""

from __future__ import annotations

from dataclasses import dataclass

import numpy as np

from .manifest import Asset


@dataclass(frozen=True)
class Frame:
    index: int
    name: str
    x: int
    y: int
    w: int
    h: int
    empty: bool
    baseline: int | None  # lowest opaque row inside the frame, or None if empty
    bbox: tuple[int, int, int, int] | None  # (x0, y0, x1, y1) of content, frame-relative


def slice_frames(rgba: np.ndarray, asset: Asset) -> list[Frame]:
    """Split a tileset or sheet into its cells, in reading order."""
    assert asset.grid is not None
    fw, fh = asset.grid.cell
    names = asset.frame_names()
    total = asset.grid.columns * asset.grid.rows
    frames = []
    for k in range(total):
        r, c = divmod(k, asset.grid.columns)
        cell = rgba[r * fh : (r + 1) * fh, c * fw : (c + 1) * fw]
        opaque = cell[..., 3] > 0
        if opaque.any():
            ys, xs = np.nonzero(opaque)
            bbox = (int(xs.min()), int(ys.min()), int(xs.max()) + 1, int(ys.max()) + 1)
            baseline = int(ys.max())
        else:
            bbox, baseline = None, None
        name = names[k] if k < len(names) else f"unused_{k:02d}"
        frames.append(Frame(k, name, c * fw, r * fh, fw, fh, bbox is None, baseline, bbox))
    return frames


def check_frames(frames: list[Frame], asset: Asset) -> list[str]:
    """Problems with a sliced tileset or sheet, as human-readable messages."""
    problems: list[str] = []
    assert asset.grid is not None
    expected = asset.frames.count if asset.frames else len(frames)
    used = frames[:expected]
    extra = [f for f in frames[expected:] if not f.empty]
    empty = [f for f in used if f.empty]
    if empty:
        names = ", ".join(f.name for f in empty)
        problems.append(f"{len(empty)} of {expected} expected frames are empty: {names}")
    if extra:
        names = ", ".join(str(f.index) for f in extra)
        problems.append(
            f"found content in {len(extra)} cells past the expected {expected} frames (cells {names})"
        )
    # Content touching every edge of its cell usually means sprites overlap
    # their neighbors or the grid is off.
    fw, _fh = asset.grid.cell
    if asset.kind == "sheet":
        for f in used:
            if f.bbox and f.bbox[0] == 0 and f.bbox[2] == fw and f.bbox[1] == 0:
                problems.append(
                    f"frame {f.name} fills its whole cell width and top; check the grid"
                )
        baselines = {f.name: f.baseline for f in used if f.baseline is not None}
        if baselines:
            want = (
                asset.frames.baseline
                if asset.frames and asset.frames.baseline is not None
                else None
            )
            if want is None:
                values = list(baselines.values())
                want = max(set(values), key=values.count)
            off = {n: b for n, b in baselines.items() if b != want}
            if off:
                detail = ", ".join(f"{n} at row {b}" for n, b in sorted(off.items()))
                problems.append(f"foot baseline should be row {want} in every frame; {detail}")
    return problems


def frame_records(frames: list[Frame], asset: Asset) -> list[dict[str, object]]:
    """Frame list for the sidecar JSON."""
    expected = asset.frames.count if asset.frames else len(frames)
    return [
        {"index": f.index, "name": f.name, "x": f.x, "y": f.y, "w": f.w, "h": f.h}
        for f in frames[:expected]
    ]


def sidecar(asset: Asset, rgba: np.ndarray, source: str, batch: str) -> dict[str, object]:
    """The sidecar JSON written next to a processed tileset or sheet."""
    assert asset.grid is not None
    return {
        "id": asset.id,
        "kind": asset.kind,
        "size": list(asset.size),
        "frame_size": list(asset.grid.cell),
        "columns": asset.grid.columns,
        "rows": asset.grid.rows,
        "frames": frame_records(slice_frames(rgba, asset), asset),
        "pivot": list(asset.pivot) if asset.pivot else None,
        "baseline": asset.frames.baseline if asset.frames else None,
        "source": source,
        "batch": batch,
    }
