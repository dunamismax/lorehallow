# tools/art

The art intake pipeline: it turns images generated in ChatGPT into true pixel art on the locked palette. Python 3.14, managed by uv (Pillow, numpy, PyYAML; pytest and ruff for development).

```bash
make art-intake BATCH=01   # process art/inbox/batch-01 into art/processed
make sync                  # mirror art/processed into client/ and web/
make art-lint art-test     # ruff, manifest validation, pytest
```

## What intake does

For each image listed in `art/inbox/batch-NN/map.yaml`, in this order:

1. **Grid detection** (`grid.py`). Finds the real block size and offset of the "pixel art", which is rarely an integer number of screen pixels. The edge profile (color change between neighboring columns, then rows) is a pulse train with one pulse per block boundary; its Fourier component at frequency 1/p is strongest at the true block size. The manifest's logical size gives the expected block size, which picks the right peak among harmonics.
2. **Downsampling** (`sampling.downsample`). One color per block from the middle half of the block, where blur and drift don't reach. With `downsample: mode` (default) every pixel votes for its nearest palette color or the key color and the block takes the average of the winners. `center` takes the middle pixel.
3. **Key removal** (`sampling.remove_key`). Blocks within `key_tolerance` of `#FF00FF` in OKLab, or closer to the key than to any palette color (the pink fringe), become transparent.
4. **Palette quantization** (`sampling.quantize`). Every remaining block snaps to the nearest palette color in OKLab, with no dithering. The report counts how many changed and the largest move.
5. **Fit** (`sampling.fit`). Trims fully transparent margin rows and columns down to the logical size, or pads back a single lost edge row or column. Anything else is an error.
6. **Fix script** (optional). If `art/batches/<batch>/fixes.py` defines `fix(asset_id, image) -> image`, it runs here. Small cleanups (a stray pixel, nudging a frame onto the baseline) live there so re-running intake reproduces them. The asset is reported as `fixed`.
7. **Checks** (`sheet.py`). Every pixel opaque and on-palette; for tilesets and sheets, the right number of non-empty frames, nothing past them, and every frame's lowest pixel on the same foot baseline.
8. **Output.** Accepted and fixed assets go to `art/processed/<target>`, with a sidecar `.json` for tilesets and sheets (frame size, frames in order with names, pivot, baseline, source file, batch). `art/batches/<batch>/report.md` gets a status table, a 4x contact sheet (`preview.png`) and, for anything to regenerate, the problems, prompt additions that address them and the original prompt.

Statuses: `accepted`, `fixed`, `regenerate` (failed a check) and `missing` (no inbox file yet). `make art-intake` exits non-zero unless everything is accepted or fixed. Re-running on the same inputs produces byte-identical outputs.

## Files

| File | Purpose |
|---|---|
| `art/batches/batch-NN-name/manifest.yaml` | What the batch should contain. Schema in `manifest.py`. |
| `art/batches/batch-NN-name/prompts.md` | Ready-to-paste prompts, one `## NN-MM asset_id: title` section per image. |
| `art/batches/batch-NN-name/fixes.py` | Optional fix script. |
| `art/inbox/batch-NN/map.yaml` | Which raw file is which asset id; optional `crop: [x, y, w, h]`. |

`art/inbox/` is never edited: the owner drops files there unchanged, and agents only add `map.yaml`.

## Sync

`make sync` copies `art/processed/` to `client/assets/art/` and `web/src/assets/art/`, removes files that no longer exist in the source, and keeps Godot's `.import` files for images that still exist.

## Tests

`tests/synth.py` renders known logical art the way generators do (non-integer scale, margins, jittered block edges, blur, noise and off-palette shifts). The tests require that the pipeline recovers the original art exactly, that misaligned or incomplete sheets fail with a clear message, and that a whole batch re-runs byte for byte.
