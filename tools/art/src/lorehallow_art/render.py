"""`make art-draw`: render hand-drawn sources into art/processed.

Every manifest asset with a `source:` is drawn from `art/src/<source>`:

- `.pxl.yaml` pixel grids (pxl.py): frames laid out on the manifest grid
  in reading order. A single sprite or tile has one frame.
- `.py` draw scripts: `draw(ctx)` returns a Canvas, a PIL image or a list
  of frames (draw.py). Scripts may compose other finished assets, which
  are rendered from their own sources first.

The result goes through intake's checks (size, palette only, frame count,
empty trailing cells, foot baseline, pivot) and is written with a sidecar
for tilesets and sheets. Each batch gets a report.md and a contact sheet
at 4x (preview.png) and 8x (preview-8x.png). Rendering is deterministic,
and `validate` fails when a processed file no longer matches its source.
"""

from __future__ import annotations

import importlib.util
import io
import json
from collections.abc import Callable
from dataclasses import dataclass, field
from pathlib import Path

import numpy as np
from PIL import Image, ImageDraw

from .draw import Canvas, Ctx
from .manifest import Asset, Manifest, load_manifest
from .palette import NamedPalette
from .pxl import SourceError, load_pxl
from .sheet import check_frames, sidecar, slice_frames

SOURCE_DIR = Path("art") / "src"
SCALES = (4, 8)
# A sprite counts as centered on its pivot column when the middle of its
# bounding box is at most this far from it (asymmetric gear allowed).
CENTER_TOLERANCE = 2.0
DARK = (0x2E, 0x24, 0x33, 255)  # night plum
LIGHT = (0xF4, 0xEC, 0xD8, 255)  # parchment white


@dataclass
class DrawResult:
    asset: Asset
    batch: str
    status: str  # drawn | problems | generated | missing
    image: np.ndarray | None = None
    source: str | None = None
    problems: list[str] = field(default_factory=list)


class Library:
    """Every manifest asset in the repository, rendered on demand and cached."""

    def __init__(self, root: Path) -> None:
        self.root = root
        self.manifests: list[Manifest] = [
            load_manifest(p)
            for p in sorted((root / "art" / "batches").glob("batch-*/manifest.yaml"))
        ]
        self.assets: dict[str, tuple[Manifest, Asset]] = {}
        for m in self.manifests:
            for a in m.assets:
                if a.id in self.assets:
                    raise SourceError(f"asset id {a.id!r} is used by more than one batch")
                self.assets[a.id] = (m, a)
        self._palettes: dict[str, NamedPalette] = {}
        self._cache: dict[str, np.ndarray] = {}
        self._stack: list[str] = []

    def manifest(self, batch: str) -> Manifest:
        for m in self.manifests:
            if m.batch == batch.zfill(2):
                return m
        raise SourceError(f"no batch {batch}")

    def palette(self, asset: Asset) -> NamedPalette:
        if asset.palette not in self._palettes:
            self._palettes[asset.palette] = NamedPalette.load(self.root / asset.palette)
        return self._palettes[asset.palette]

    def source_path(self, asset: Asset) -> Path:
        assert asset.source
        return self.root / SOURCE_DIR / asset.source

    def render(self, asset_id: str) -> np.ndarray:
        """The asset rendered fresh from its source (cached per Library)."""
        if asset_id in self._cache:
            return self._cache[asset_id]
        if asset_id not in self.assets:
            raise SourceError(f"unknown asset {asset_id!r}")
        _, asset = self.assets[asset_id]
        if not asset.source:
            raise SourceError(f"{asset_id} has no source")
        if asset_id in self._stack:
            raise SourceError("assets compose each other in a loop: " + " → ".join(self._stack))
        self._stack.append(asset_id)
        try:
            arr = render_source(asset, self.palette(asset), self.source_path(asset), self.finished)
        finally:
            self._stack.pop()
        self._cache[asset_id] = arr
        return arr

    def finished(self, asset_id: str) -> tuple[np.ndarray, Asset]:
        """A finished asset for composites: rendered from source if it has
        one, otherwise the processed file (generated art)."""
        if asset_id not in self.assets:
            raise SourceError(f"unknown asset {asset_id!r}")
        _, asset = self.assets[asset_id]
        if asset.source:
            return self.render(asset_id), asset
        path = self.root / "art" / "processed" / asset.target
        if not path.exists():
            raise SourceError(f"{asset_id} is not drawn or processed yet")
        with Image.open(path) as im:
            return np.asarray(im.convert("RGBA")).copy(), asset


def _layout(frames: list[np.ndarray], asset: Asset, where: str) -> np.ndarray:
    cell = asset.grid.cell if asset.grid else asset.size
    cols = asset.grid.columns if asset.grid else 1
    cap = asset.grid.columns * asset.grid.rows if asset.grid else 1
    if len(frames) > cap:
        raise SourceError(f"{where}: {len(frames)} frames, but the grid holds {cap}")
    out = np.zeros((asset.size[1], asset.size[0], 4), dtype=np.uint8)
    for k, f in enumerate(frames):
        if (f.shape[1], f.shape[0]) != cell:
            raise SourceError(
                f"{where}: frame {k} is {f.shape[1]}x{f.shape[0]}, the manifest frame is "
                f"{cell[0]}x{cell[1]}"
            )
        r, c = divmod(k, cols)
        out[r * cell[1] : (r + 1) * cell[1], c * cell[0] : (c + 1) * cell[0]] = f
    return out


def _as_array(obj: object, where: str) -> np.ndarray:
    if isinstance(obj, Canvas):
        return obj.a.copy()
    if isinstance(obj, Image.Image):
        return np.asarray(obj.convert("RGBA")).copy()
    if isinstance(obj, np.ndarray) and obj.ndim == 3 and obj.shape[2] == 4:
        return obj.astype(np.uint8)
    raise SourceError(f"{where}: draw() must return a Canvas, an image or a list of frames")


def render_source(
    asset: Asset,
    palette: NamedPalette,
    src: Path,
    finished: Callable[[str], tuple[np.ndarray, Asset]],
) -> np.ndarray:
    """Render one source to an RGBA array at the asset's logical size."""
    where = str(src)
    if not src.exists():
        raise SourceError(f"{asset.id}: source {src} does not exist")
    if src.name.endswith(".pxl.yaml"):
        pxl = load_pxl(src, palette)
        out = _layout(list(pxl.frames), asset, where)
    else:
        spec = importlib.util.spec_from_file_location(f"lorehallow_src_{asset.id}", src)
        assert spec and spec.loader
        module = importlib.util.module_from_spec(spec)
        try:
            spec.loader.exec_module(module)
        except SourceError:
            raise
        except Exception as e:
            raise SourceError(f"{where}: {type(e).__name__}: {e}") from e
        fn = getattr(module, "draw", None)
        if not callable(fn):
            raise SourceError(f"{where} must define draw(ctx)")
        try:
            result = fn(Ctx(asset, palette, src, finished))
        except SourceError:
            raise
        except KeyError as e:
            raise SourceError(f"{where}: {e.args[0] if e.args else e}") from e
        except Exception as e:
            raise SourceError(f"{where}: {type(e).__name__}: {e}") from e
        if isinstance(result, (list, tuple)):
            out = _layout([_as_array(f, where) for f in result], asset, where)
        else:
            out = _as_array(result, where)
    if (out.shape[1], out.shape[0]) != asset.size:
        raise SourceError(
            f"{where}: drew {out.shape[1]}x{out.shape[0]}, the manifest size is "
            f"{asset.size[0]}x{asset.size[1]}"
        )
    out = out.copy()
    out[out[..., 3] == 0] = 0
    return out


def palette_problems(rgba: np.ndarray, palette: NamedPalette) -> list[str]:
    opaque = rgba[..., 3] > 0
    if (rgba[..., 3][opaque] != 255).any():
        return ["semi-transparent pixels present"]
    colors = {tuple(int(v) for v in c) for c in rgba[opaque][:, :3]}
    off = colors - set(palette.palette.colors)
    return [f"{len(off)} colors not in the palette"] if off else []


def _pivot_problem(cell: np.ndarray, pivot: tuple[int, int], label: str) -> list[str]:
    opaque = cell[..., 3] > 0
    if not opaque.any():
        return []
    ys, xs = np.nonzero(opaque)
    problems = []
    center = (xs.min() + xs.max() + 1) / 2
    if abs(center - pivot[0]) > CENTER_TOLERANCE:
        problems.append(f"{label} is centered on column {center:g}, the pivot is {pivot[0]}")
    if int(ys.max()) != pivot[1]:
        problems.append(f"{label}'s lowest pixel is on row {int(ys.max())}, expected {pivot[1]}")
    return problems


def check(rgba: np.ndarray, asset: Asset, palette: NamedPalette) -> list[str]:
    """Intake's checks for a drawn asset."""
    problems = palette_problems(rgba, palette)
    if not (rgba[..., 3] > 0).any():
        problems.append("the image is empty")
    if asset.grid:
        frames = slice_frames(rgba, asset)
        problems.extend(check_frames(frames, asset))
        if asset.pivot and asset.kind == "sheet":
            for f in frames:
                if not f.empty:
                    cell = rgba[f.y : f.y + f.h, f.x : f.x + f.w]
                    # The baseline is check_frames' job; only centering here.
                    problems.extend(
                        p for p in _pivot_problem(cell, asset.pivot, f.name) if "centered" in p
                    )
    elif asset.pivot:
        problems.extend(_pivot_problem(rgba, asset.pivot, asset.id))
    return problems


def draw_asset(lib: Library, asset_id: str) -> DrawResult:
    m, asset = lib.assets[asset_id]
    src = (SOURCE_DIR / asset.source).as_posix() if asset.source else None
    try:
        img = lib.render(asset_id)
    except SourceError as e:
        return DrawResult(asset, m.batch, "problems", None, src, [str(e)])
    problems = check(img, asset, lib.palette(asset))
    return DrawResult(asset, m.batch, "problems" if problems else "drawn", img, src, problems)


def outputs(r: DrawResult) -> dict[str, bytes]:
    """Processed files for a drawn asset: the PNG and, for grids, a sidecar."""
    a = r.asset
    assert r.image is not None
    buf = io.BytesIO()
    Image.fromarray(r.image, "RGBA").save(buf, format="PNG", optimize=True)
    out = {a.target: buf.getvalue()}
    if a.grid:
        data = sidecar(a, r.image, r.source or "", r.batch)
        out[str(Path(a.target).with_suffix(".json"))] = (json.dumps(data, indent=2) + "\n").encode()
    return out


def run_draw(root: Path, batch: str | None = None, write: bool = True) -> list[DrawResult]:
    """Render every hand-drawn asset (of one batch, or all), write the
    processed files, and write each batch's report and contact sheets.
    Generated assets are listed in the report from their processed files."""
    lib = Library(root)
    manifests = [lib.manifest(batch)] if batch else lib.manifests
    results: list[DrawResult] = []
    for m in manifests:
        batch_results = []
        for a in m.assets:
            if a.source:
                r = draw_asset(lib, a.id)
            else:
                path = root / "art" / "processed" / a.target
                r = DrawResult(a, m.batch, "generated" if path.exists() else "missing")
                if path.exists():
                    with Image.open(path) as im:
                        r.image = np.asarray(im.convert("RGBA")).copy()
                r.source = "intake"
            batch_results.append(r)
        if write and any(a.source for a in m.assets):
            for r in batch_results:
                if r.asset.source and r.status == "drawn":
                    for rel, data in outputs(r).items():
                        dest = root / "art" / "processed" / rel
                        dest.parent.mkdir(parents=True, exist_ok=True)
                        if not dest.exists() or dest.read_bytes() != data:
                            dest.write_bytes(data)
            write_report(root, m, batch_results)
        results.extend(batch_results)
    return results


def _decode(data: bytes) -> np.ndarray:
    with Image.open(io.BytesIO(data)) as im:
        a = np.asarray(im.convert("RGBA")).copy()
    a[a[..., 3] == 0] = 0
    return a


def stale_drawn(root: Path) -> list[str]:
    """Problems with hand-drawn assets: sources that don't render or fail a
    check, renders that aren't deterministic, and processed files that
    don't match a fresh render of their source."""
    lib, again = Library(root), Library(root)
    problems = []
    for asset_id, (_m, a) in lib.assets.items():
        if not a.source:
            continue
        r = draw_asset(lib, asset_id)
        if r.image is None or r.problems:
            problems.extend(f"{asset_id}: {p}" for p in r.problems)
            continue
        try:
            second = again.render(asset_id)
        except SourceError as e:
            problems.append(f"{asset_id}: {e}")
            continue
        if not np.array_equal(second, r.image):
            problems.append(f"{asset_id}: two renders differ; the source is not deterministic")
            continue
        for rel, data in outputs(r).items():
            path = root / "art" / "processed" / rel
            if not path.exists():
                problems.append(f"{asset_id}: art/processed/{rel} is missing; run make art-draw")
            elif rel.endswith(".png"):
                if not np.array_equal(_decode(path.read_bytes()), r.image):
                    problems.append(
                        f"{asset_id}: art/processed/{rel} doesn't match its source; "
                        "run make art-draw"
                    )
            elif json.loads(path.read_bytes()) != json.loads(data):
                problems.append(
                    f"{asset_id}: art/processed/{rel} doesn't match its source; run make art-draw"
                )
    return problems


# -- report and contact sheets ----------------------------------------------


def _checker(w: int, h: int, size: int = 8) -> Image.Image:
    yy, xx = np.mgrid[0:h, 0:w]
    on = ((xx // size + yy // size) % 2).astype(bool)
    arr = np.zeros((h, w, 4), dtype=np.uint8)
    arr[...] = (70, 66, 78, 255)
    arr[on] = (90, 86, 98, 255)
    return Image.fromarray(arr, "RGBA")


def contact_sheet(results: list[DrawResult], scale: int) -> Image.Image:
    """Each asset at `scale` on a checkerboard, with 1x copies on dark and
    light backgrounds beside it, labeled, one per row."""
    shown = [r for r in results if r.image is not None]
    pad, label_h = 8, 14
    rows = []
    for r in shown:
        img = Image.fromarray(r.image, "RGBA")
        big = img.resize((img.width * scale, img.height * scale), Image.NEAREST)
        rows.append((r, img, big))
    width = max([big.width + 2 * img.width + 4 * pad for _, img, big in rows] + [320]) + 2 * pad
    height = pad + sum(big.height + label_h + pad for _, _, big in rows)
    sheet = Image.new("RGBA", (width, max(height, 40)), (34, 30, 40, 255))
    draw = ImageDraw.Draw(sheet)
    y = pad
    for r, img, big in rows:
        draw.text((pad, y), f"{r.asset.id} [{r.status}]", fill=(240, 230, 210, 255))
        y += label_h
        checker = _checker(big.width, big.height)
        checker.alpha_composite(big)
        sheet.alpha_composite(checker, (pad, y))
        x = pad + big.width + pad
        for bg in (DARK, LIGHT):
            tile = Image.new("RGBA", (img.width + 2 * pad, img.height + 2 * pad), bg)
            tile.alpha_composite(img, (pad, pad))
            sheet.alpha_composite(tile, (x, y))
            x += tile.width
        y += big.height + pad
    if not shown:
        draw.text((pad, pad), "nothing drawn yet", fill=(240, 230, 210, 255))
    return sheet


def _png(img: Image.Image) -> bytes:
    buf = io.BytesIO()
    img.save(buf, format="PNG", optimize=True)
    return buf.getvalue()


def write_report(root: Path, manifest: Manifest, results: list[DrawResult]) -> None:
    assert manifest.path is not None
    bdir = manifest.path.parent
    previews = {4: "preview.png", 8: "preview-8x.png"}
    for scale, name in previews.items():
        # The 8x sheet shows only hand-drawn art; large painted images at
        # 8x would make it many megabytes for no review value.
        subset = results if scale == 4 else [r for r in results if r.asset.source]
        data = _png(contact_sheet(subset, scale))
        path = bdir / name
        if not path.exists() or path.read_bytes() != data:
            path.write_bytes(data)
    counts: dict[str, int] = {}
    for r in results:
        counts[r.status] = counts.get(r.status, 0) + 1
    lines = [
        f"# Batch {manifest.batch} report" + (f": {manifest.title}" if manifest.title else ""),
        "",
        f"Generated by `make art-draw BATCH={manifest.batch}`. Do not edit by hand.",
        "Hand-drawn assets are rendered from `art/src/`; generated ones come from",
        "intake (`intake-report.md` has their details, when there are any).",
        "",
        "Summary: " + ", ".join(f"{n} {s}" for s, n in sorted(counts.items())),
        "",
        "| Asset | Kind | Size | Source | Status | Problems |",
        "|---|---|---|---|---|---|",
    ]
    for r in results:
        a = r.asset
        src = f"`{r.source}`" if r.source and r.source != "intake" else (r.source or "")
        lines.append(
            f"| `{a.id}` | {a.kind} | {a.size[0]}×{a.size[1]} | {src} | {r.status} | "
            + "; ".join(r.problems).replace("|", "/")
            + " |"
        )
    lines += [
        "",
        "Contact sheets: [4x, every asset](preview.png) and [8x, hand-drawn](preview-8x.png).",
        "Each asset is also shown at 1x on dark and light backgrounds.",
        "",
        "![Contact sheet at 4x](preview.png)",
        "",
    ]
    text = "\n".join(lines)
    path = bdir / "report.md"
    if not path.exists() or path.read_text() != text:
        path.write_text(text)


# -- review images (not committed) -------------------------------------------


def write_review(results: list[DrawResult], out: Path, scale: int = 3) -> list[Path]:
    """Extra views for looking at art: every tile of a tileset repeated
    3×3 (seams show), and each asset at 1x and `scale` on dark and light."""
    out.mkdir(parents=True, exist_ok=True)
    written = []
    for r in results:
        if r.image is None:
            continue
        img = Image.fromarray(r.image, "RGBA")
        a = r.asset
        views = []
        for bg in (DARK, LIGHT):
            for s in (1, scale):
                tile = Image.new("RGBA", (img.width * s + 16, img.height * s + 16), bg)
                tile.alpha_composite(
                    img.resize((img.width * s, img.height * s), Image.NEAREST), (8, 8)
                )
                views.append(tile)
        sheet = Image.new("RGBA", (sum(v.width for v in views), max(v.height for v in views)), DARK)
        x = 0
        for v in views:
            sheet.alpha_composite(v, (x, 0))
            x += v.width
        p = out / f"{a.id}-views.png"
        sheet.save(p)
        written.append(p)
        if a.kind == "tileset" and a.grid:
            fw, fh = a.grid.cell
            gap = 4
            canvas = Image.new(
                "RGBA",
                (a.grid.columns * (3 * fw + gap) + gap, a.grid.rows * (3 * fh + gap) + gap),
                DARK,
            )
            for f in slice_frames(r.image, a):
                if f.empty:
                    continue
                cell = img.crop((f.x, f.y, f.x + f.w, f.y + f.h))
                ox = gap + (f.index % a.grid.columns) * (3 * fw + gap)
                oy = gap + (f.index // a.grid.columns) * (3 * fh + gap)
                for j in range(3):
                    for i in range(3):
                        canvas.alpha_composite(cell, (ox + i * fw, oy + j * fh))
            canvas = canvas.resize((canvas.width * 2, canvas.height * 2), Image.NEAREST)
            p = out / f"{a.id}-tiled3x3.png"
            canvas.save(p)
            written.append(p)
    return written
