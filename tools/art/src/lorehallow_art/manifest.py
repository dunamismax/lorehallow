"""Batch manifests (art/batches/<batch>/manifest.yaml) and inbox maps
(art/inbox/<batch>/map.yaml): loading and validation.

Manifest shape::

    batch: "01"
    title: Style foundation
    palette: art/style/palette.hex     # default for every asset
    key_color: "#FF00FF"               # default for every asset
    assets:
      - id: player_base                # snake_case, unique
        target: chars/player_base.png  # under art/processed/
        kind: sheet                    # tile | tileset | sprite | sheet | ui | web
        size: [192, 192]               # logical size in art pixels (width, height)
        grid:                          # tileset and sheet only
          cell: [32, 48]               # frame or tile size
          columns: 6
          rows: 4
        frames:                        # required for sheets, optional for tilesets
          count: 24                    # frames in reading order; trailing cells may be empty
          rows: [down, up, left, right]          # optional row names
          columns: [idle_0, idle_1, walk_0, walk_1, walk_2, walk_3]  # optional
          names: [a, b, c]             # optional explicit names, one per frame
          baseline: 45                 # sheets only: row of the lowest foot pixel in every frame
        pivot: [16, 45]                # optional, in frame (or sprite) coordinates
        downsample: mode               # mode | center
        key_tolerance: 0.12            # optional, OKLab distance
        palette: art/style/palette.hex # optional per-asset override
        key_color: "#FF00FF"           # optional per-asset override
        prompt: 03-01                  # optional: prompts.md section for this asset
        notes: free text

Map shape::

    files:
      "ChatGPT Image Nov 6, 2026.png": player_base
      "another.png":
        asset: terrain_basic
        crop: [0, 0, 1024, 768]        # optional (x, y, width, height) in the raw image
        grid: fixed                    # optional: detect (default) or fixed, see intake.py
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import yaml

from .color import RGB, hex_to_rgb

KINDS = ("tile", "tileset", "sprite", "sheet", "ui", "web")
GRID_KINDS = ("tileset", "sheet")
ID_RE = re.compile(r"^[a-z][a-z0-9_]*$")
TARGET_RE = re.compile(r"^(tiles|chars|mobs|npcs|items|ui|vfx|icons|web|style)/[a-z0-9_]+\.png$")
DEFAULT_KEY = "#FF00FF"
DEFAULT_PALETTE = "art/style/palette.hex"
DEFAULT_TOLERANCE = 0.12


class ManifestError(ValueError):
    pass


@dataclass(frozen=True)
class GridSpec:
    cell: tuple[int, int]
    columns: int
    rows: int


@dataclass(frozen=True)
class FrameSpec:
    count: int
    row_names: tuple[str, ...] = ()
    column_names: tuple[str, ...] = ()
    names: tuple[str, ...] = ()
    baseline: int | None = None


@dataclass(frozen=True)
class Asset:
    id: str
    target: str
    kind: str
    size: tuple[int, int]
    palette: str
    key_color: RGB
    grid: GridSpec | None = None
    frames: FrameSpec | None = None
    pivot: tuple[int, int] | None = None
    downsample: str = "mode"
    key_tolerance: float = DEFAULT_TOLERANCE
    prompt: str | None = None
    notes: str = ""

    def frame_names(self) -> list[str]:
        """Names of each frame (or tile) in reading order."""
        if not self.grid:
            return [self.id]
        n = self.frames.count if self.frames else self.grid.columns * self.grid.rows
        if self.frames and self.frames.names:
            return list(self.frames.names)
        rn = self.frames.row_names if self.frames else ()
        cn = self.frames.column_names if self.frames else ()
        names = []
        for k in range(n):
            r, c = divmod(k, self.grid.columns)
            if rn and cn:
                names.append(f"{rn[r]}_{cn[c]}")
            else:
                names.append(f"{self.id}_{k:02d}")
        return names


@dataclass(frozen=True)
class Manifest:
    batch: str
    title: str
    assets: tuple[Asset, ...]
    path: Path | None = None

    def asset(self, asset_id: str) -> Asset:
        for a in self.assets:
            if a.id == asset_id:
                return a
        raise KeyError(asset_id)


@dataclass(frozen=True)
class MapEntry:
    file: str
    asset: str
    crop: tuple[int, int, int, int] | None = None
    grid: str = "detect"


GRID_MODES = ("detect", "fixed")


@dataclass(frozen=True)
class InboxMap:
    entries: tuple[MapEntry, ...] = field(default_factory=tuple)


def _pair(v: Any, what: str) -> tuple[int, int]:
    if not (isinstance(v, list) and len(v) == 2 and all(isinstance(x, int) and x > 0 for x in v)):
        raise ManifestError(f"{what} must be [width, height] of positive integers, got {v!r}")
    return (v[0], v[1])


def _parse_asset(raw: dict[str, Any], defaults: dict[str, Any]) -> Asset:
    where = f"asset {raw.get('id', '?')!r}"

    def err(msg: str) -> ManifestError:
        return ManifestError(f"{where}: {msg}")

    known = {
        "id", "target", "kind", "size", "grid", "frames", "pivot", "downsample",
        "key_tolerance", "palette", "key_color", "prompt", "notes",
    }  # fmt: skip
    unknown = set(raw) - known
    if unknown:
        raise err(f"unknown keys {sorted(unknown)}")
    aid = raw.get("id")
    if not isinstance(aid, str) or not ID_RE.match(aid):
        raise err("id must be snake_case")
    target = raw.get("target")
    if not isinstance(target, str) or not TARGET_RE.match(target):
        raise err(f"target must look like <category>/<snake_case>.png, got {target!r}")
    kind = raw.get("kind")
    if kind not in KINDS:
        raise err(f"kind must be one of {KINDS}, got {kind!r}")
    size = _pair(raw.get("size"), f"{where} size")

    grid = None
    if kind in GRID_KINDS:
        g = raw.get("grid")
        if not isinstance(g, dict):
            raise err(f"kind {kind} needs a grid")
        cell = _pair(g.get("cell"), f"{where} grid.cell")
        cols, rows = g.get("columns"), g.get("rows")
        if not (isinstance(cols, int) and isinstance(rows, int) and cols > 0 and rows > 0):
            raise err("grid.columns and grid.rows must be positive integers")
        if (cell[0] * cols, cell[1] * rows) != size:
            raise err(
                f"grid {cols}x{rows} of {cell[0]}x{cell[1]} is not the size {size[0]}x{size[1]}"
            )
        grid = GridSpec(cell, cols, rows)
    elif "grid" in raw:
        raise err(f"kind {kind} takes no grid")

    frames = None
    if "frames" in raw:
        if grid is None:
            raise err("only sheets and tilesets have frames")
        f = raw["frames"]
        count = f.get("count")
        if not (isinstance(count, int) and 0 < count <= grid.columns * grid.rows):
            raise err(f"frames.count must be 1..{grid.columns * grid.rows}")
        rn = tuple(f.get("rows", ()))
        cn = tuple(f.get("columns", ()))
        if rn and len(rn) != grid.rows:
            raise err(f"frames.rows names {len(rn)} rows, grid has {grid.rows}")
        if cn and len(cn) != grid.columns:
            raise err(f"frames.columns names {len(cn)} columns, grid has {grid.columns}")
        names = tuple(f.get("names", ()))
        if names and (len(names) != count or len(set(names)) != count):
            raise err(f"frames.names must list {count} unique names")
        if names and not all(isinstance(n, str) and ID_RE.match(n) for n in names):
            raise err("frames.names must be snake_case")
        baseline = f.get("baseline")
        if baseline is not None and kind != "sheet":
            raise err("only sheets have a baseline")
        if baseline is not None and not (
            isinstance(baseline, int) and 0 <= baseline < grid.cell[1]
        ):
            raise err("frames.baseline must be a row inside the frame")
        frames = FrameSpec(count, rn, cn, names, baseline)
    elif kind == "sheet":
        raise err("sheets need frames")

    pivot = None
    if "pivot" in raw:
        pv = raw["pivot"]
        if not (
            isinstance(pv, list) and len(pv) == 2 and all(isinstance(x, int) and x >= 0 for x in pv)
        ):
            raise err("pivot must be [x, y]")
        pivot = (pv[0], pv[1])

    downsample = raw.get("downsample", "mode")
    if downsample not in ("mode", "center"):
        raise err("downsample must be mode or center")
    tol = raw.get("key_tolerance", DEFAULT_TOLERANCE)
    if not isinstance(tol, (int, float)) or not 0 <= tol <= 1:
        raise err("key_tolerance must be between 0 and 1")
    try:
        key = hex_to_rgb(str(raw.get("key_color", defaults["key_color"])))
    except ValueError as e:
        raise err(str(e)) from e
    return Asset(
        id=aid,
        target=target,
        kind=kind,
        size=size,
        palette=str(raw.get("palette", defaults["palette"])),
        key_color=key,
        grid=grid,
        frames=frames,
        pivot=pivot,
        downsample=downsample,
        key_tolerance=float(tol),
        prompt=str(raw["prompt"]) if "prompt" in raw else None,
        notes=str(raw.get("notes", "")),
    )


def parse_manifest(data: Any, path: Path | None = None) -> Manifest:
    if not isinstance(data, dict):
        raise ManifestError("manifest must be a mapping")
    batch = data.get("batch")
    if not isinstance(batch, str) or not re.fullmatch(r"\d{2}", batch):
        raise ManifestError('batch must be a two-digit string like "01"')
    defaults = {
        "palette": data.get("palette", DEFAULT_PALETTE),
        "key_color": data.get("key_color", DEFAULT_KEY),
    }
    raw_assets = data.get("assets")
    if not isinstance(raw_assets, list) or not raw_assets:
        raise ManifestError("assets must be a non-empty list")
    assets = tuple(_parse_asset(a, defaults) for a in raw_assets)
    ids = [a.id for a in assets]
    targets = [a.target for a in assets]
    for name, values in (("id", ids), ("target", targets)):
        dupes = sorted({v for v in values if values.count(v) > 1})
        if dupes:
            raise ManifestError(f"duplicate asset {name}s: {dupes}")
    return Manifest(batch=batch, title=str(data.get("title", "")), assets=assets, path=path)


def load_manifest(path: Path) -> Manifest:
    try:
        return parse_manifest(yaml.safe_load(path.read_text()), path)
    except ManifestError as e:
        raise ManifestError(f"{path}: {e}") from e


def parse_map(data: Any, manifest: Manifest) -> InboxMap:
    if data is None:
        return InboxMap()
    files = data.get("files") if isinstance(data, dict) else None
    if not isinstance(files, dict):
        raise ManifestError("map.yaml must have a 'files' mapping")
    ids = {a.id for a in manifest.assets}
    entries = []
    for file, v in files.items():
        grid = "detect"
        if isinstance(v, str):
            asset, crop = v, None
        elif isinstance(v, dict) and "asset" in v:
            asset = v["asset"]
            grid = v.get("grid", "detect")
            if grid not in GRID_MODES:
                raise ManifestError(f"map.yaml: grid for {file!r} must be one of {GRID_MODES}")
            c = v.get("crop")
            if c is not None and not (
                isinstance(c, list)
                and len(c) == 4
                and all(isinstance(x, int) and x >= 0 for x in c)
            ):
                raise ManifestError(f"map.yaml: crop for {file!r} must be [x, y, width, height]")
            crop = tuple(c) if c else None
        else:
            raise ManifestError(f"map.yaml: entry for {file!r} must be an asset id or a mapping")
        if asset not in ids:
            raise ManifestError(f"map.yaml: {file!r} maps to unknown asset {asset!r}")
        entries.append(MapEntry(file=str(file), asset=asset, crop=crop, grid=grid))  # type: ignore[arg-type]
    return InboxMap(tuple(entries))


def load_map(path: Path, manifest: Manifest) -> InboxMap:
    if not path.exists():
        return InboxMap()
    return parse_map(yaml.safe_load(path.read_text()), manifest)
