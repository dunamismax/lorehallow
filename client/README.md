# client/

The Godot 4 game client skeleton, written in statically typed GDScript. **It is being replaced.**

## Next: the Phaser client

Lorehallow becomes a browser game at play.lorehallow.com, built with Phaser, strict TypeScript and Vite ([ADR 0014](../docs/decisions/0014-astro-website-and-phaser-browser-client.md)). None of it exists yet; it is the next work in the plan, starting at BUILD.md 2.6. These notes are for whoever starts it. Everything they mention was checked on 2026-11-08; check versions again before pinning.

**Order** (BUILD.md 2.6, then 2.2, 2.7, 2.8, 2.9, 2.10): the scaffold and `make client-dev`/`client-build`; TypeScript protobuf bindings passing every golden vector; integer pixel scaling and the title screen; the dev art scene and its headless pixel check; the connect screen and Net module; input, interpolation and animation; the debug overlay and network simulation; and the two-browser smoke test against the real Go server.

**Toolchain.** The npm workspace already exists ([ADR 0015](../docs/decisions/0015-frontend-toolchain.md)): add `"client"` to `workspaces` in the root `package.json` and give `client/` its own `package.json` (`@lorehallow/client`), `tsconfig.json` (extend the same strict settings as `web/`, plus `noUncheckedIndexedAccess`) and `vite.config.ts`. Add `client/src/**` and `client/tests/**` to the `includes` in `biome.json`, and exclude the generated bindings. Tests use `node --test`, as in `web/`. Latest versions seen: Phaser 4.2.1, Vite 8.3.6 (Astro already depends on Vite 8), `@bufbuild/protobuf` and `@bufbuild/protoc-gen-es` 2.17.0, Playwright 1.65.0 for the headless browser.

**Protobuf bindings.** Add a `local` protoc-gen-es plugin to `proto/buf.gen.yaml` with `target=ts`, `import_extension=ts` and `erasable_syntax=true`. The last one makes enums `as const` objects instead of TypeScript `enum`s; without it Node's type stripping can't run the generated code in tests, and `erasableSyntaxOnly` fails. Write a `PROTOCOL_VERSION` constant from `proto/PROTOCOL_VERSION` in `tools/scripts/gen-proto.sh`, next to the Go and GDScript ones, and add the output folder to `proto-check`'s stale-code list in the Makefile. The golden JSON files are protojson with proto field names and enum numbers, so compare against `toJson(schema, msg, { useProtoFieldName: true, enumAsInteger: true })`.

**Pixel scaling.** Phaser's `roundPixels` only behaves at integer zoom. Choose the scale in device pixels (`floor(min(innerWidth * devicePixelRatio / 640, innerHeight * devicePixelRatio / 360))`, at least 1), keep the canvas backing store at 640×360 with `pixelArt: true` and `Scale.NONE`, set the canvas's CSS size to 640k/dpr by 360k/dpr pixels, and call `game.scale.refresh()` after a resize so input coordinates follow. A pure function for the scale choice makes the required test easy.

**Assets.** Load art and maps through Vite's asset handling so built files get content hashes, and set `build.assetsInlineLimit: 0` so nothing becomes a `data:` URL. The game's CSP (BUILD.md 2.9) allows scripts only from `'self'`.

**Text.** Phaser's canvas text is anti-aliased, which breaks the pixel rules. The hand-drawn font in `web/static/fonts/lorehallow-pixel.txt` has a parser in `web/src/lib/pixelfont.ts` (`PixelFont.parse`, `measure`, `rects`). That module imports `fillRect` from `web/src/lib/png.ts`, which imports `node:zlib`; move `RGBA`, `blank` and `fillRect` into a small module with no Node imports before using the parser in the browser.

**Local server.** `make server` runs the Go server on 127.0.0.1:7420 with `content/maps/starter.tmj`. The Vite dev server should proxy `/ws` (with `ws: true`), `/healthz`, `/version` and `/api/` to it.

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
