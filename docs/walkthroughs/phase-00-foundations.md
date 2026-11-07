# Phase 0 walkthrough: Foundations

Phase 0 builds no gameplay. It builds the workshop: a repository where one command checks everything, a protocol that two languages share byte for byte, and art tools that render hand-drawn pixel art from text (plus a pipeline that turns the occasional generated image into true pixel art). Every later phase leans on these.

## What we built

- **A repository skeleton** with one README per top-level directory, editor and git attributes, and architecture decision records (ADRs) in `docs/decisions/`.
- **A pinned toolchain** in `toolchain.env`, checked by `make doctor` and installed (where it can be) by `make setup`.
- **A command runner**: one root `Makefile` that every human, agent and CI job uses. `make check` is the gate.
- **A Go server skeleton**: configuration, structured logging, `/healthz` and `/version`, version injection at link time, and graceful shutdown.
- **A protobuf pipeline**: one schema in `proto/`, generated Go and GDScript, and golden vectors that both sides must decode and re-encode exactly.
- **A Godot client skeleton** with the pixel-art rendering rules baked into project settings, a title screen that shows its build, and headless tests.
- **Hand-drawn art as text** in `art/src/`: pixel grids (`.pxl.yaml`) and draw scripts (`.py`), rendered and checked by `make art-draw` ([ADR 0004](../decisions/0004-hand-drawn-pixel-art.md)). The terrain, hero, props, style reference, logo, emblems, icons, progress landmarks and character lineup are all drawn this way.
- **An art intake pipeline** in `tools/art/` that finds the hidden pixel grid in generated images, snaps them to the locked palette and checks sprite sheets. It now handles only the few large painted web images.
- **The art direction**: a 33-color palette, a style guide and a style reference scene built from the game's own tiles and sprites.
- **CI** on GitHub Actions that runs exactly `make check`.

## How to try it

```bash
make doctor     # are the pinned tools installed?
make setup      # Go modules, Godot addons (GUT, godobuf), the art tools' Python env
make check      # every formatter, linter, codegen check and test suite
make server     # run the server on 127.0.0.1:7420
curl localhost:7420/version
make client     # open the client: a title screen with the build version
make help       # everything else
```

To see the art tools at work:

```bash
make art-draw BATCH=01 REVIEW=/tmp/review   # render art/src for batch 01; review images in /tmp/review
open art/batches/batch-01-style/preview-8x.png
make client-art-check                       # render the dev art scene in Godot and check every pixel
```

Change a letter in `art/src/chars/concept_hero.pxl.yaml` and run `make check`: it fails, because the processed image no longer matches its source. `make art-test` covers the rest, including intake, whose tests render known pixel art the way an image generator would (odd scales, margins, blur, noise, off-palette colors) and require the pipeline to recover the original exactly.

## How it works

### One command runner

The `Makefile` is deliberately boring. Each component has `*-lint` and `*-test` targets; `lint`, `test` and `check` just call them. CI runs `make doctor`, `make setup` and `make check`, nothing else, so "it passed locally" and "it passed in CI" mean the same thing.

Versions live in one file, `toolchain.env`, which is both valid shell and valid Make syntax. The Makefile `include`s it, `tools/scripts/doctor.sh` `source`s it, and CI appends it to `$GITHUB_ENV`. Changing a version is a one-line edit ([ADR 0002](../decisions/0002-toolchain-versions.md)).

### The server skeleton

`server/cmd/lorehallow-server/main.go` does very little in `main`: it builds a signal-aware context and calls `run`. `run` takes its arguments, environment and output as parameters, which is what makes it testable: the test starts the real server on port 0, waits for the `onListen` callback (no sleeping), makes a real HTTP request, cancels the context and checks that shutdown is clean.

Configuration (`internal/config`) has a strict precedence: flags beat `LOREHALLOW_*` environment variables, which beat defaults. `-ldflags -X` stamps the version and commit into `internal/buildinfo` at build time, so a running binary can always tell you what it is (`GET /version`).

### Protobuf on both sides

```
proto/lorehallow/v1/envelope.proto
          │
          ├── buf + protoc-gen-go ──────▶ server/internal/pb/envelope.pb.go
          ├── godot + godobuf ─────────▶ client/net/pb/envelope_pb.gd
          └── PROTOCOL_VERSION ────────▶ pb.ProtocolVersion / ProtocolVersion.CURRENT

server/internal/protovectors ──▶ proto/testdata/*.bin + *.json
                                    │
                         ┌──────────┴──────────┐
                  Go round-trip test     GDScript round-trip test
```

`make proto` (`tools/scripts/gen-proto.sh`) regenerates all of it. The Go side uses buf with `protoc-gen-go`, pinned through a `tool` directive in `server/go.mod`. GDScript has no official protobuf support, so we evaluated godobuf, a generator written in GDScript that runs inside Godot, and kept it because it passed every golden vector ([ADR 0003](../decisions/0003-gdscript-protobuf.md)).

`make check` then guards three things:

1. **Staleness.** It regenerates everything and fails if the result differs from what is committed or staged.
2. **Compatibility.** `buf breaking` compares the schema with `main` (in CI, with the previous push). A breaking change is accepted only if `proto/PROTOCOL_VERSION` was bumped in the same change.
3. **Agreement.** Golden round-trip tests on both sides (below).

### Drawing pixel art as text

Every hand-drawn asset has a source in `art/src/` named by `source:` in its batch manifest. Two formats share one renderer (`tools/art/src/lorehallow_art/render.py`):

- A **pixel grid** is YAML: a size, a legend mapping one character to one palette color *by name*, and one text grid per frame. The hero is 48 lines of 32 characters. Because colors are names (`ink`, `moss shadow`), an off-palette pixel can't be written down at all; a typo is an "unknown palette color" error.
- A **draw script** is Python with `draw(ctx)`. It draws on a `Canvas` (`draw.py`) whose primitives (pixel, line, rect, ellipse, 1 px outline, mirror, palette swap, stamp) only accept palette names, and whose only randomness is `ctx.rng(seed)`. Scripts suit art that follows rules: textured ground, transition sets, form-lit tree canopies, composites like the style reference, and the character lineup, which recolors the hero's own head and body so all five adventurers stay on-model.

The renderer lays frames out on the manifest grid and runs intake's checks: size, palette only, frame count, empty trailing cells, the foot baseline and the pivot. It writes the processed PNG, a sidecar for sheets, and per batch a report with contact sheets at 4x and 8x. `make check` renders every source twice: it fails if the two renders differ (a script used the clock or unseeded randomness) or if `art/processed` doesn't match (someone edited a source and forgot to re-render, or edited a processed file by hand).

Looking at the art is part of the process, not an afterthought: `REVIEW=dir` adds each tileset with every tile repeated 3×3 and each asset at 1x and 3x on dark and light backgrounds. In Godot, `make client-art-check` renders the dev art scene and compares every on-screen pixel with its source at 1x to 4x.

### The intake pipeline (for generated images)

Image generators can draw convincing pixel art, but the result isn't pixel art: a 32×48 character arrives as a 1024×1536 image where each "pixel" is a 32×32 blob with soft edges, slight drift and colors a few shades off. `tools/art` recovers the real thing in steps:

1. **Find the grid** (`grid.py`, below).
2. **Sample each block's middle**, where blur doesn't reach, and let its pixels vote for the nearest palette color (or the background key).
3. **Remove the key color** `#FF00FF` and the pinkish fringe around sprites.
4. **Snap to the palette** in OKLab, a color space where distance matches perceived difference.
5. **Fit** the result to the logical size, trimming empty margins.
6. **Check** sheets: frame count, nothing in unused cells, and every frame's feet on the same row.
7. **Write** the PNG, a JSON sidecar describing frames, and a report with a 4x contact sheet and corrected prompts for anything to regenerate.

A per-batch `fixes.py` handles small cleanups, so re-running intake always reproduces the same output from the untouched inbox.

## The key concept: agreeing on bytes, finding a hidden grid, and tiles that can't have seams

### Golden vectors

Two programs in two languages must agree on every byte of every message, forever. Unit tests on each side only prove each side agrees with itself. The fix is a shared set of **golden vectors**: Go writes known messages to `proto/testdata/` as raw bytes (`ping_max.bin`) plus a readable description (`ping_max.json`). Both sides then:

1. decode the bytes,
2. compare the decoded values with the description, and
3. re-encode and require the identical bytes.

The vectors deliberately include awkward cases: an empty message inside a `oneof` (encoded as `0a 00`), `uint32` max, and `int64` min and max, which take ten-byte varints. A test fails if any `Envelope` payload type has no vector, so a new message can't skip this.

### Finding the pixel grid with a Fourier transform

How do you find the block size in an image when blocks are 5.33 screen pixels wide and slightly wobbly? Look at where the color changes.

Sum the color difference between each pair of neighboring columns and you get an **edge profile**: near zero inside blocks, a spike at every block boundary. That profile is a pulse train, spikes spaced one block apart (not every boundary has a spike, since neighboring blocks can share a color, but every spike is on a boundary).

```
edge profile:  |    |         |    |    |         |    |
               0   5.3      16.0 21.3 26.7     37.3 42.7  ...
```

A pulse train with spacing `p` has a strong Fourier component at frequency `1/p`: all the spikes line up in phase. At frequency `1/(2p)` half the spikes are out of phase and cancel. So we measure the Fourier magnitude for every candidate period and look for peaks. The true period peaks; so do its fractions (`p/2`, `p/3`), because a pulse train repeats at those too. The manifest tells us the logical size, which gives an expected block size, and we pick the peak nearest it. The **phase** of the winning component tells us where the first boundary falls, which is the grid offset. One more subtlety: peaks get narrower as periods get shorter (their width is about `p²/n` for an image `n` pixels wide), so the search steps more finely at short periods.

All of this is in `detect_axis` in `tools/art/src/lorehallow_art/grid.py`, about 40 lines.

### Seamless by construction

A ground tile is seamless when its right edge continues into its own left edge, and its bottom into its top. Drawing that by eye and checking it afterwards is fragile. Instead, ground tiles are drawn on a canvas that **wraps**: every coordinate is taken modulo 32, so a grass tuft placed at x = 30 spills its last two pixels onto x = 0 and 1. The tile is a torus; there is no edge to get wrong.

Variants are harder: `grass_b` must sit seamlessly next to `grass_a`, not just next to itself. Both are drawn from one layout of features, and a variant keeps every feature that comes within 3 px of the border and re-rolls only the ones wholly inside. Their borders are then pixel-identical, so any variant can sit next to any other.

Transition tiles (grass to dirt) use the same idea in one dimension. Each boundary is a depth profile, `depth(x) = 9 + Σ aₖ sin(2πkx/32)`: periodic over 32 px, and exactly 9 px at every tile border. An edge tile always meets its neighbor at depth 9, so edges self-tile, corners meet edges, and the 3×3 patch closes, whatever the wiggles in between look like.

```
edge_t        edge_t        edge_t
~~~~~~~~~^~~~~~~~~~~~~^~~~~~~~~~~~~^   ← depth 9 at every tile border (^)
dirt          dirt          dirt
```

## Read the code

In this order:

1. `Makefile`: the targets and how `check` is composed. Then `toolchain.env` and `tools/scripts/doctor.sh`.
2. `server/cmd/lorehallow-server/main.go`: `run`, and its test in `main_test.go`.
3. `server/internal/config/config.go`: `Load` and its table-driven tests.
4. `proto/lorehallow/v1/envelope.proto` and `proto/README.md`.
5. `tools/scripts/gen-proto.sh` and `tools/scripts/proto-breaking.sh`.
6. `server/internal/protovectors/protovectors.go`: `All`, `Write`, and the three tests in `protovectors_test.go`.
7. `client/tests/test_proto_golden.gd` and `client/tests/helpers/pb_dict.gd`: the same checks from GDScript.
8. `client/project.godot`: the pixel rules (viewport, stretch, filtering, snapping, import defaults) and the typing rules.
9. `art/style/STYLE.md`, then `art/src/chars/concept_hero.pxl.yaml`: a sprite as text.
10. `tools/art/src/lorehallow_art/pxl.py` and `render.py`: `render_source`, `check` and `stale_drawn`.
11. `tools/art/src/lorehallow_art/draw.py`: `Canvas` (note `wrap`) and `Ctx`.
12. `art/src/tiles/terrain_base.py` (`Texture.tile`) and `art/src/tiles/transition.py` (`wave`, `inner_fn`, `tile`).
13. `art/src/style/reference.py`: a scene composited from finished assets and autotiled ground.
14. `client/scenes/dev/art_verify.gd`: checking the rendered game against the sources.
15. `tools/art/src/lorehallow_art/grid.py`: `edge_profile`, `fourier` and `detect_axis`; then `sampling.py` and `intake.py`, where `process_image` is the whole intake pipeline in one function.
16. `tools/art/tests/synth.py`: how the synthetic "generated" images are made.

## What's next

Phase 0 finishes when the owner approves the logo and hero art. Phase 1 then builds lorehallow.com: hosting, TLS, the Astro site and the live progress tracker that reads this repository's `BUILD.md`.
