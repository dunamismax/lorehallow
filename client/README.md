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
