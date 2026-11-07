# Phase 0 walkthrough: Foundations

Phase 0 builds no gameplay. It builds the workshop: a repository where one command checks everything, a protocol that two languages share byte for byte, and a pipeline that turns AI-generated images into true pixel art. Every later phase leans on these.

## What we built

- **A repository skeleton** with one README per top-level directory, editor and git attributes, and architecture decision records (ADRs) in `docs/decisions/`.
- **A pinned toolchain** in `toolchain.env`, checked by `make doctor` and installed (where it can be) by `make setup`.
- **A command runner**: one root `Makefile` that every human, agent and CI job uses. `make check` is the gate.
- **A Go server skeleton**: configuration, structured logging, `/healthz` and `/version`, version injection at link time, and graceful shutdown.
- **A protobuf pipeline**: one schema in `proto/`, generated Go and GDScript, and golden vectors that both sides must decode and re-encode exactly.
- **A Godot client skeleton** with the pixel-art rendering rules baked into project settings, a title screen that shows its build, and headless tests.
- **An art intake pipeline** in `tools/art/` that finds the hidden pixel grid in generated images, snaps them to the locked palette and checks sprite sheets.
- **The art direction**: a 33-color palette, a style guide and ready-to-paste prompts for the first two art batches.
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

To see the art pipeline in action without any real art, run the tests: `make art-test`. They render known pixel art the way an image generator would (odd scales, margins, blur, noise, off-palette colors) and require the pipeline to recover the original exactly.

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

### The art pipeline

Image generators can draw convincing pixel art, but the result isn't pixel art: a 32×48 character arrives as a 1024×1536 image where each "pixel" is a 32×32 blob with soft edges, slight drift and colors a few shades off. `tools/art` recovers the real thing in steps:

1. **Find the grid** (`grid.py`, below).
2. **Sample each block's middle**, where blur doesn't reach, and let its pixels vote for the nearest palette color (or the background key).
3. **Remove the key color** `#FF00FF` and the pinkish fringe around sprites.
4. **Snap to the palette** in OKLab, a color space where distance matches perceived difference.
5. **Fit** the result to the logical size, trimming empty margins.
6. **Check** sheets: frame count, nothing in unused cells, and every frame's feet on the same row.
7. **Write** the PNG, a JSON sidecar describing frames, and a report with a 4x contact sheet and corrected prompts for anything to regenerate.

A per-batch `fixes.py` handles small cleanups, so re-running intake always reproduces the same output from the untouched inbox.

## The key concept: agreeing on bytes, and finding a hidden grid

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
9. `tools/art/src/lorehallow_art/grid.py`: `edge_profile`, `fourier` and `detect_axis`.
10. `tools/art/src/lorehallow_art/sampling.py`, then `intake.py`: `process_image` is the whole pipeline in one function.
11. `tools/art/tests/synth.py`: how the synthetic "generated" images are made.
12. `art/style/STYLE.md` and `art/batches/batch-01-style/prompts.md`.

## What's next

Phase 0 finishes when the first two art batches are generated, run through intake and accepted. Phase 1 then builds lorehallow.com: hosting, TLS, the Astro site and the live progress tracker that reads this repository's `BUILD.md`.
