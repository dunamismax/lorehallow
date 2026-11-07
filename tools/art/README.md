# tools/art

The art tools. Most art is **hand-drawn as text sources** in `art/src/` and rendered by `make art-draw` ([ADR 0004](../../docs/decisions/0004-hand-drawn-pixel-art.md)). The intake pipeline is kept for the few large painted images that are generated: it turns them into true pixel art on the locked palette. Python 3.14, managed by uv (Pillow, numpy, PyYAML; pytest and ruff for development).

```bash
make art-draw BATCH=01     # render art/src sources for batch 01 into art/processed
make art-draw REVIEW=/tmp/review  # every batch, plus review images (tiles 3×3, 1x views)
make art-intake BATCH=02   # process generated images in art/inbox/batch-02
make sync                  # mirror art/processed into client/ and web/
make art-lint art-test     # ruff, manifest and source validation, pytest
```

## Drawing by hand

A manifest asset with `source: <category>/<name>.pxl.yaml` or `source: <category>/<name>.py` (relative to `art/src/`) is hand-drawn. `make art-draw` renders it into `art/processed/<target>`; intake skips it, and an inbox `map.yaml` may not name it.

### Pixel grids (`.pxl.yaml`)

```yaml
size: [16, 16]          # one frame
legend:                 # one character per color, by palette name
  ".": transparent
  K: ink
  g: lantern gold
frames:                 # one grid per frame, in reading order
  - |
    ................
    ......KKKK......   # a trailing comment needs two spaces before the #
    ...
```

Names are the comments in `art/style/palette.hex` (`ink`, `moss shadow`, `dusk violet-grey`, `lantern gold`). Every grid must be exactly `size`, and every character must be in the legend. Frames are laid out on the manifest grid in reading order; a sprite or single tile has one. A grid is also a reusable *part* that draw scripts stamp.

### Draw scripts (`.py`)

```python
def draw(ctx):
    rng = ctx.rng(1)  # seeded random.Random: the only randomness
    t = ctx.canvas(32, 32, wrap=True)  # wrap=True: strokes wrap around the edges
    t.fill("moss")
    for _ in range(12):
        t.px(rng.randrange(32), rng.randrange(32), "leaf")
    return t  # a Canvas, a PIL image, or a list of frames
```

| Helper (`draw.py`) | Does |
|---|---|
| `ctx.canvas(w, h, wrap=False)` | A new transparent canvas (default: the asset size). |
| `ctx.rng(seed)` | `random.Random` seeded from the asset id and `seed`. |
| `ctx.pxl("part.pxl.yaml")` | The frames of a pixel grid next to the script, as canvases. |
| `ctx.art(id)`, `ctx.frame(id, name)` | A finished asset (rendered from its own source, or the processed file for generated art), or one named frame of it. For composites. |
| `ctx.palette["leaf"]`, `ctx.cell`, `ctx.size` | RGB by name, frame size, asset size. |
| `c.px`, `c.points`, `c.line`, `c.polyline`, `c.rect`, `c.ellipse`, `c.fill` | Primitives; colors by name, `None` is transparent. `fill=False` draws a 1 px rim. |
| `c.outline("ink", corners=False)` | 1 px outline around every opaque shape. |
| `c.mirror()`, `c.flip()`, `c.crop(...)`, `c.swap({...})`, `c.replace(a, b)` | Mirrored, flipped, cropped and palette-swapped copies; recolor in place. |
| `c.stamp(part, x, y, mirror=False)` | Copy another canvas's opaque pixels. Wraps on a wrapping canvas. |
| `c.get(x, y)`, `c.mask()`, `c.where(pred, color)` | Read back colors and coverage; recolor by rule. |

Scripts must be deterministic: no clock, no unseeded randomness, no files other than parts and finished assets.

### Checks and outputs

Each drawn asset goes through intake's checks: logical size, palette only (no semi-transparency), frame count and empty trailing cells, the foot baseline for sheets, and the pivot (lowest pixel on the pivot row, centered on the pivot column within 2 px). Tilesets and sheets get the same sidecar `.json` as intake, with `source` set to the source path.

Per batch, `make art-draw` writes `report.md` (every asset: drawn, generated or missing, with problems) and two contact sheets: `preview.png` at 4x with every asset, and `preview-8x.png` at 8x with the hand-drawn ones. Each asset also appears at 1x on dark and light backgrounds. With `REVIEW=dir` it also writes, outside the repo, each tileset with every tile repeated 3×3 (seams show) and each asset at 1x and 3x on dark and light.

`make check` (`lorehallow-art validate`) renders every source twice and fails if a source doesn't render (unknown color name, wrong size, a script error), fails a check, renders differently twice, or doesn't match its processed file.

## Generated art (the exception)

### What intake does

For each image listed in `art/inbox/batch-NN/map.yaml`, in this order:

1. **Grid detection** (`grid.py`). Finds the real block size and offset of the "pixel art", which is rarely an integer number of screen pixels. The edge profile (color change between neighboring columns, then rows) is a pulse train with one pulse per block boundary; its Fourier component at frequency 1/p is strongest at the true block size. The manifest's logical size gives the expected block size, which picks the right peak among harmonics.
2. **Downsampling** (`sampling.downsample`). One color per block from the middle half of the block, where blur and drift don't reach. With `downsample: mode` (default) every pixel votes for its nearest palette color or the key color and the block takes the average of the winners. `center` takes the middle pixel.
3. **Key removal** (`sampling.remove_key`). Blocks within `key_tolerance` of `#FF00FF` in OKLab, or closer to the key than to any palette color (the pink fringe), become transparent.
4. **Palette quantization** (`sampling.quantize`). Every remaining block snaps to the nearest palette color in OKLab, with no dithering. The report counts how many changed and the largest move.
5. **Fit** (`sampling.fit`). Trims fully transparent margin rows and columns down to the logical size, or pads back a single lost edge row or column. Anything else is an error.
6. **Fix script** (optional). If `art/batches/<batch>/fixes.py` defines `fix(asset_id, image) -> image`, it runs here. Small cleanups (a stray pixel, nudging a frame onto the baseline) live there so re-running intake reproduces them. The asset is reported as `fixed`.
7. **Checks** (`sheet.py`). Every pixel opaque and on-palette; for tilesets and sheets, the right number of non-empty frames, nothing past them, and every frame's lowest pixel on the same foot baseline.
8. **Output.** Accepted and fixed assets go to `art/processed/<target>`, with a sidecar `.json` for tilesets and sheets (frame size, frames in order with names, pivot, baseline, source file, batch). `art/batches/<batch>/intake-report.md` gets a status table, a 4x contact sheet (`intake-preview.png`) and, for anything to regenerate, the problems, prompt additions that address them and the original prompt.

**Grid hints and modes.** Detection needs a rough block size. The canvas gives an upper bound (the art fits in it); when the drawn content is much smaller than the canvas (a sprite drawn small on a big key background), the content size gives a better estimate. Art pixels are square, so a non-square detection is re-detected near the other axis's period and kept only if it samples worse. Painted full-bleed scenes have no single block size at all (trees drawn in bigger clusters than faces): map them with `grid: fixed`, which samples on exactly the logical layout (use `crop` first if the art is a band inside the canvas).

**Sprites with a pivot** are placed so their middle column sits on the pivot column and their lowest pixel on the pivot row; intake reports any sprite that can't be.

Statuses: `accepted`, `fixed`, `regenerate` (failed a check) and `missing` (no inbox file yet). `make art-intake` exits non-zero unless everything is accepted or fixed. Re-running on the same inputs produces byte-identical outputs.

## Files

| File | Purpose |
|---|---|
| `art/batches/batch-NN-name/manifest.yaml` | What the batch should contain. Schema in `manifest.py`. |
| `art/src/<category>/<name>.pxl.yaml`, `.py` | Hand-drawn sources, named by `source:` in the manifest. |
| `art/batches/batch-NN-name/report.md`, `preview.png`, `preview-8x.png` | Written by `make art-draw`. |
| `art/batches/batch-NN-name/prompts.md` | Ready-to-paste prompts, one `## NN-MM asset_id: title` section per image. |
| `art/batches/batch-NN-name/fixes.py` | Optional fix script. |
| `art/inbox/batch-NN/map.yaml` | Which raw file is which asset id; optional `crop: [x, y, w, h]` and `grid: fixed`. |
| `art/batches/batch-NN-name/guides/<id>.png` | Layout guides for transition tilesets, generated by `make art-generate` from the manifest's `guide` colors. Attached to prompts. |

`art/inbox/` is never edited: the owner drops files there unchanged, and agents only add `map.yaml`.

## Sync

`make sync` copies `art/processed/` to `web/src/assets/art/` and, except the website-only `web/` category, to `client/assets/art/`. It also writes `art/style/reference.png`: the accepted style reference at 6x, the image attached to every later prompt. It removes files that no longer exist in the source, and keeps Godot's `.import` files for images that still exist.

## Tests

`tests/test_draw.py` covers the grid parser, the drawing helpers, the renderer and every check in a temporary repository. `tests/synth.py` renders known logical art the way generators do (non-integer scale, margins, jittered block edges, blur, noise and off-palette shifts). The tests require that the pipeline recovers the original art exactly, that misaligned or incomplete sheets fail with a clear message, and that a whole batch re-runs byte for byte.
