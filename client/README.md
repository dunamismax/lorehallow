# client/

The Godot 4 game client, written in statically typed GDScript.

| Path | What goes there |
|---|---|
| `addons/` | Third-party addons (GUT). Vendored, not linted. |
| `assets/` | `art/` and `content/` are synced copies (`make sync`), never edited here. `audio/` and `fonts/` hold their own sources. |
| `autoload/` | Singletons registered in `project.godot`: `Config`, `Log`, `Net`. |
| `net/` | Networking. `net/pb/` is generated protobuf code (`make proto`), never edited by hand. |
| `world/` | Maps, camera and world rendering. |
| `entities/` | Players, mobs and NPCs. |
| `ui/` | Screens, HUD, windows and the theme. |
| `scenes/dev/` | Test and debug scenes. |
| `tests/` | GUT tests, run headless by `make test`. |

The client never decides outcomes. It sends inputs and draws what the server reports, and later predicts only its own movement.

Run it with `make client`. Format and lint with `make fmt` and `make lint`. See [AGENTS.md section 7](../AGENTS.md#client-conventions-godot-4-gdscript) for conventions.

## Project settings that matter

- **Pixel rules:** base viewport 640×360, stretch mode `viewport`, integer scaling, Nearest texture filter, 2D transform and vertex snapping, and an import default of lossless, no mipmaps. Don't change these without an ADR.
- **Renderer:** Compatibility (OpenGL), for the widest hardware support. Lorehallow is 2D and needs nothing from Forward+.
- **Static typing:** `untyped_declaration` is set to **Error**, so untyped code fails to load and `make test` fails. Headless runs don't print warnings, so only an error can be enforced in CI. `unsafe_*` warnings are on as warnings for the editor.
- **Excluded from warnings:** `res://addons` and `res://net/pb` (generated).

## Build version

`Config.version` and `Config.commit` come from the `application/config/version` and `lorehallow/build/commit` project settings. `make client` writes `client/override.cfg` (not committed) with the output of `git describe` and the commit, and Godot applies it on top of `project.godot`. Without it the title screen shows `dev (unknown)`.

## Tests

`make client-test` runs GUT headless with `.gutconfig.json`. `tests/test_proto_golden.gd` decodes and re-encodes every golden protocol vector in `proto/testdata/`.
