"""The intake pipeline for one batch: raw inbox images in, processed pixel
art, sidecars and a report out. Re-running it gives the same result."""

from __future__ import annotations

import importlib.util
import json
from collections.abc import Callable
from dataclasses import dataclass, field
from pathlib import Path

import numpy as np
from PIL import Image, ImageDraw

from .grid import Grid, detect_grid
from .manifest import Asset, InboxMap, Manifest, load_manifest, load_map
from .palette import Palette
from .sampling import FitError, downsample, fit, quantize, remove_key
from .sheet import check_frames, frame_records, slice_frames

FixFn = Callable[[str, Image.Image], Image.Image]
PREVIEW_SCALE = 4


@dataclass
class AssetResult:
    asset: Asset
    status: str = "missing"  # accepted | fixed | regenerate | missing
    source: str | None = None
    grid: Grid | None = None
    raw_size: tuple[int, int] | None = None
    cells: tuple[int, int] | None = None
    keyed: int = 0
    changed: int = 0
    max_distance: float = 0.0
    trimmed: tuple[int, int] = (0, 0)
    padded: tuple[int, int] = (0, 0)
    problems: list[str] = field(default_factory=list)
    image: np.ndarray | None = None  # final RGBA at logical size


@dataclass
class BatchResult:
    manifest: Manifest
    batch_dir: Path
    results: list[AssetResult]
    unmapped: list[str]

    @property
    def ok(self) -> bool:
        return all(r.status in ("accepted", "fixed") for r in self.results)


def find_batch_dir(root: Path, batch: str) -> Path:
    batch = batch.zfill(2)
    matches = sorted((root / "art" / "batches").glob(f"batch-{batch}-*"))
    if len(matches) != 1:
        raise FileNotFoundError(
            f"expected one art/batches/batch-{batch}-* folder, found {len(matches)}"
        )
    return matches[0]


def load_fixes(batch_dir: Path) -> FixFn | None:
    """Load the optional per-batch fix script, `fixes.py`, with a function
    fix(asset_id, image) -> image applied after quantization."""
    path = batch_dir / "fixes.py"
    if not path.exists():
        return None
    spec = importlib.util.spec_from_file_location(f"lorehallow_fixes_{batch_dir.name}", path)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    fn = getattr(module, "fix", None)
    if not callable(fn):
        raise AttributeError(f"{path} must define fix(asset_id, image)")
    return fn


def process_image(
    raw: Image.Image, asset: Asset, palette: Palette, fix: FixFn | None = None
) -> AssetResult:
    """Run one raw image through the pipeline for one asset."""
    res = AssetResult(asset=asset)
    img = np.asarray(raw.convert("RGBA"))
    if not (img[..., 3] < 255).any():
        img = img[..., :3]
    h, w = img.shape[:2]
    res.raw_size = (w, h)
    hint = (w / asset.size[0], h / asset.size[1])
    grid = detect_grid(img[..., :3], hint=hint)
    res.grid = grid
    rgb, alpha = downsample(img, grid, asset.downsample, palette, asset.key_color)
    res.cells = (rgb.shape[1], rgb.shape[0])
    key = remove_key(rgb, alpha, asset.key_color, palette, asset.key_tolerance)
    res.keyed = key.keyed
    q = quantize(rgb, key.opaque, palette)
    res.changed, res.max_distance = q.changed, q.max_distance
    try:
        fitted = fit(q.rgba, *asset.size)
    except FitError as e:
        res.status = "regenerate"
        res.problems.append(str(e))
        return res
    res.trimmed, res.padded = fitted.trimmed, fitted.padded
    out = fitted.rgba
    status = "accepted"
    if fix is not None:
        fixed = np.asarray(fix(asset.id, Image.fromarray(out, "RGBA")).convert("RGBA"))
        if fixed.shape != out.shape:
            raise ValueError(f"fix script changed the size of {asset.id}")
        if not np.array_equal(fixed, out):
            status = "fixed"
            out = fixed
    res.problems.extend(_palette_problems(out, palette))
    if asset.grid:
        res.problems.extend(check_frames(slice_frames(out, asset), asset))
    if not (out[..., 3] > 0).any():
        res.problems.append("the image is empty after removing the key color")
    res.image = out
    res.status = "regenerate" if res.problems else status
    return res


def _palette_problems(rgba: np.ndarray, palette: Palette) -> list[str]:
    opaque = rgba[..., 3] > 0
    if (rgba[..., 3][opaque] != 255).any():
        return ["semi-transparent pixels present"]
    colors = {tuple(int(v) for v in c) for c in rgba[opaque][:, :3]}
    off = colors - set(palette.colors)
    return [f"{len(off)} colors not in the palette"] if off else []


def run_batch(root: Path, batch: str, write: bool = True) -> BatchResult:
    batch_dir = find_batch_dir(root, batch)
    manifest = load_manifest(batch_dir / "manifest.yaml")
    inbox = root / "art" / "inbox" / f"batch-{manifest.batch}"
    imap: InboxMap = load_map(inbox / "map.yaml", manifest)
    fix = load_fixes(batch_dir)
    palettes: dict[str, Palette] = {}

    by_asset = {e.asset: e for e in imap.entries}
    raw_files = sorted(
        p.name for p in inbox.glob("*") if p.suffix.lower() in (".png", ".webp", ".jpg", ".jpeg")
    )
    unmapped = [f for f in raw_files if f not in {e.file for e in imap.entries}]

    results = []
    for asset in manifest.assets:
        entry = by_asset.get(asset.id)
        if entry is None:
            results.append(AssetResult(asset=asset))
            continue
        path = inbox / entry.file
        if not path.exists():
            r = AssetResult(asset=asset, status="regenerate", source=entry.file)
            r.problems.append(f"mapped file {entry.file!r} is not in the inbox")
            results.append(r)
            continue
        if asset.palette not in palettes:
            palettes[asset.palette] = Palette.load(root / asset.palette)
        with Image.open(path) as raw:
            raw.load()
            if entry.crop:
                x, y, cw, ch = entry.crop
                raw = raw.crop((x, y, x + cw, y + ch))
            r = process_image(raw, asset, palettes[asset.palette], fix)
        r.source = entry.file
        results.append(r)

    result = BatchResult(manifest, batch_dir, results, unmapped)
    if write:
        write_outputs(root, result)
        write_report(root, result)
    return result


def write_outputs(root: Path, result: BatchResult) -> None:
    processed = root / "art" / "processed"
    for r in result.results:
        if r.image is None or r.status not in ("accepted", "fixed"):
            continue
        a = r.asset
        dest = processed / a.target
        dest.parent.mkdir(parents=True, exist_ok=True)
        Image.fromarray(r.image, "RGBA").save(dest, optimize=True)
        if a.grid:
            sidecar = {
                "id": a.id,
                "kind": a.kind,
                "size": list(a.size),
                "frame_size": list(a.grid.cell),
                "columns": a.grid.columns,
                "rows": a.grid.rows,
                "frames": frame_records(slice_frames(r.image, a), a),
                "pivot": list(a.pivot) if a.pivot else None,
                "baseline": a.frames.baseline if a.frames else None,
                "source": r.source,
                "batch": result.manifest.batch,
            }
            dest.with_suffix(".json").write_text(json.dumps(sidecar, indent=2) + "\n")


def contact_sheet(results: list[AssetResult]) -> Image.Image:
    """All processed assets at 4x on a checkerboard, labeled, one per row."""
    shown = [r for r in results if r.image is not None]
    pad, label_h = 8, 14
    width = max([r.image.shape[1] * PREVIEW_SCALE for r in shown] + [320]) + 2 * pad
    height = pad + sum(r.image.shape[0] * PREVIEW_SCALE + label_h + pad for r in shown)
    sheet = Image.new("RGBA", (width, max(height, 40)), (34, 30, 40, 255))
    draw = ImageDraw.Draw(sheet)
    y = pad
    for r in shown:
        img = Image.fromarray(r.image, "RGBA")
        big = img.resize((img.width * PREVIEW_SCALE, img.height * PREVIEW_SCALE), Image.NEAREST)
        draw.text((pad, y), f"{r.asset.id} [{r.status}]", fill=(240, 230, 210, 255))
        y += label_h
        checker = _checker(big.width, big.height)
        checker.alpha_composite(big)
        sheet.alpha_composite(checker, (pad, y))
        y += big.height + pad
    if not shown:
        draw.text((pad, pad), "no processed assets yet", fill=(240, 230, 210, 255))
    return sheet


def _checker(w: int, h: int, size: int = 8) -> Image.Image:
    yy, xx = np.mgrid[0:h, 0:w]
    on = ((xx // size + yy // size) % 2).astype(bool)
    arr = np.zeros((h, w, 4), dtype=np.uint8)
    arr[...] = (70, 66, 78, 255)
    arr[on] = (90, 86, 98, 255)
    return Image.fromarray(arr, "RGBA")


def _prompt_section(batch_dir: Path, asset: Asset) -> str | None:
    """The prompts.md section for an asset (by `prompt` number or id)."""
    path = batch_dir / "prompts.md"
    if not path.exists():
        return None
    sections = path.read_text().split("\n## ")
    for s in sections[1:]:
        head = s.split("\n", 1)[0]
        if (asset.prompt and head.startswith(asset.prompt)) or f" {asset.id}" in head:
            return "## " + s.strip()
    return None


def write_report(root: Path, result: BatchResult) -> None:
    m = result.manifest
    bdir = result.batch_dir
    preview_name = "preview.png"
    contact_sheet(result.results).save(bdir / preview_name, optimize=True)
    counts: dict[str, int] = {}
    for r in result.results:
        counts[r.status] = counts.get(r.status, 0) + 1
    lines = [
        f"# Batch {m.batch} intake report" + (f": {m.title}" if m.title else ""),
        "",
        "Generated by `make art-intake BATCH=" + m.batch + "`. Do not edit by hand; re-run intake.",
        "",
        "Summary: " + ", ".join(f"{n} {s}" for s, n in sorted(counts.items())),
        "",
        "| Asset | Status | Source | Grid (px per art pixel) | Cells | Quantized | Notes |",
        "|---|---|---|---|---|---|---|",
    ]
    for r in result.results:
        grid = f"{r.grid.x.period:.2f} × {r.grid.y.period:.2f}" if r.grid else ""
        cells = f"{r.cells[0]}×{r.cells[1]}" if r.cells else ""
        quant = f"{r.changed} px (max Δ {r.max_distance:.3f})" if r.grid else ""
        notes = []
        if r.trimmed != (0, 0):
            notes.append(f"trimmed margin {r.trimmed[0]}×{r.trimmed[1]}")
        if r.padded != (0, 0):
            notes.append(f"padded {r.padded[0]}×{r.padded[1]}")
        notes.extend(r.problems)
        lines.append(
            f"| `{r.asset.id}` | {r.status} | {r.source or ''} | {grid} | {cells} | {quant} | "
            + "; ".join(notes).replace("|", "/")
            + " |"
        )
    if result.unmapped:
        lines += [
            "",
            "Inbox files not in map.yaml: " + ", ".join(f"`{f}`" for f in result.unmapped),
        ]
    lines += ["", f"![Contact sheet at {PREVIEW_SCALE}x]({preview_name})", ""]

    redo = [r for r in result.results if r.status in ("regenerate", "missing")]
    if redo:
        lines += ["## Needs (re)generation", ""]
        for r in redo:
            lines.append(f"### `{r.asset.id}` ({r.status})")
            lines.append("")
            if r.problems:
                lines += ["Problems:", ""] + [f"- {p}" for p in r.problems] + [""]
            section = _prompt_section(bdir, r.asset)
            if r.status == "regenerate":
                lines += [
                    "Corrected prompt: start from the prompt below and add the fixes that",
                    "address the problems above, then regenerate into the same inbox folder.",
                    "",
                ]
                for hint in _prompt_fixes(r):
                    lines.append(f"- Add: “{hint}”")
                lines.append("")
            if section:
                lines += [
                    "<details><summary>Original prompt</summary>",
                    "",
                    section,
                    "",
                    "</details>",
                    "",
                ]
    (bdir / "report.md").write_text("\n".join(lines).rstrip() + "\n")


def _prompt_fixes(r: AssetResult) -> list[str]:
    a = r.asset
    out = []
    for p in r.problems:
        if "baseline" in p and a.frames and a.grid:
            row = a.frames.baseline if a.frames.baseline is not None else a.grid.cell[1] - 3
            out.append(
                f"In every frame the lowest foot pixel sits on exactly the same row ({row} of "
                f"{a.grid.cell[1]}); the character never bobs below or above it."
            )
        elif "empty" in p:
            out.append("Every cell of the grid contains a sprite; no cell is left blank.")
        elif "past the expected" in p:
            out.append("Leave the cells after the last frame completely empty (flat #FF00FF).")
        elif "larger than the logical" in p or "cells along" in p:
            out.append(
                f"The whole image is exactly {a.size[0]}×{a.size[1]} art pixels, drawn as equal "
                "square blocks, with no border, margin or frame around it."
            )
        elif "palette" in p or "semi-transparent" in p:
            out.append(
                "Use only the listed palette colors, flat fills, no gradients or soft edges."
            )
    return sorted(set(out))
