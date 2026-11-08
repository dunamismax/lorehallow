# client/

The Godot 4 game client skeleton, written in statically typed GDScript. **It is being replaced.**

## Planned: the Phaser browser client

Lorehallow becomes a browser game at play.lorehallow.com, built with Phaser, strict TypeScript and Vite ([ADR 0014](../docs/decisions/0014-astro-website-and-phaser-browser-client.md), BUILD.md 2.6). None of it exists yet. The first task is the scaffold, then, in BUILD.md order:

- `make client-dev` (Vite dev server, proxying `/ws` and `/api/` to the local Go server) and `make client-build` (the static build, with version and commit embedded)
- TypeScript protobuf bindings from `make proto`, passing every golden vector in `proto/testdata/` byte for byte
- the starter map from `content/maps/`, drawn at whole-number scales of 640×360 with nearest-neighbor sampling
- a dev art scene with a pixel check in a headless browser (the new `make client-art-check`)
- a browser smoke test where two sessions see each other move through the real Go server

Both clients share this folder until BUILD.md 2.10. Keep Godot out of the Phaser folders, keep the Godot checks below green, and remove Godot, its addons, the GDScript bindings and their tooling only after the Phaser checks pass in `make check`.

## The Godot skeleton (current)

| Path | What goes there |
|---|---|
| `addons/` | Third-party addons (GUT). Vendored, not linted. |
| `assets/` | `art/` and `content/` are synced copies (`make sync`), never edited here. `audio/` and `fonts/` hold their own sources. |
| `autoload/` | Singletons registered in `project.godot`: `Config`, `Log`, `Net`. |
| `net/` | Networking. `net/pb/` is generated protobuf code (`make proto`), never edited by hand. |
| `world/` | Maps, camera and world rendering. |
| `entities/` | Players, mobs and NPCs. |
| `ui/` | Screens, HUD, windows and the theme. |
| `scenes/dev/` | Test and debug scenes. `art_test.tscn` shows every synced image at 1x to 4x with a 3×3 seam patch per tileset; `make client-art-check` renders it in a window and checks every pixel against the source (`ART=all` for every image). |
| `tests/` | GUT tests, run headless by `make test`. |

The client never decides outcomes. It sends inputs and draws what the server reports, and later predicts only its own movement.

Run it with `make client`. Format and lint with `make fmt` and `make lint`. AGENTS.md section 7 describes the [Godot skeleton as it stands](../AGENTS.md#client-conventions-phaser-typescript) and the conventions for the Phaser client.

## Project settings that matter

- **Pixel rules:** base viewport 640×360, stretch mode `viewport`, integer scaling, Nearest texture filter, 2D transform and vertex snapping, and an import default of lossless, no mipmaps. Don't change these without an ADR.
- **Renderer:** Compatibility (OpenGL), for the widest hardware support. Lorehallow is 2D and needs nothing from Forward+.
- **Static typing:** `untyped_declaration` is set to **Error**, so untyped code fails to load and `make test` fails. Headless runs don't print warnings, so only an error can be enforced in CI. `unsafe_*` warnings are on as warnings for the editor.
- **Excluded from warnings:** `res://addons` and `res://net/pb` (generated).

## Build version

`Config.version` and `Config.commit` come from the `application/config/version` and `lorehallow/build/commit` project settings. `make client` writes `client/override.cfg` (not committed) with the output of `git describe` and the commit, and Godot applies it on top of `project.godot`. Without it the title screen shows `dev (unknown)`.

## Tests

`make client-test` runs GUT headless with `.gutconfig.json`. `tests/test_proto_golden.gd` decodes and re-encodes every golden protocol vector in `proto/testdata/`.
