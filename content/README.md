# content/

Canonical game data, read by the server and synced into the client (`make sync` mirrors it to `client/assets/content/`).

- YAML for items, mobs, quests and so on; Tiled JSON (`.tmj`) for maps ([ADR 0006](../docs/decisions/0006-tiled-map-format.md)).
- IDs are `snake_case`, stable and never reused.
- Every content type gets a validator that runs in `make check` (`make content-lint`, the `lorehallow-content` command in `server/`).

| Path | What |
|---|---|
| `maps/<map_id>.tmj` | Maps: orthogonal, 32×32 tiles, uncompressed layer data. Tile layers are drawn bottom to top; the `collision` layer marks solid tiles and is never drawn; the `spawns` object layer holds point objects of type `spawn`. |
| `tilesets/<name>.tsj` | External Tiled tilesets. `<name>` is the processed art id: the image is `art/processed/tiles/<name>.png`. |

Edit maps in [Tiled](https://www.mapeditor.org/) (open the `.tmj` directly) or by hand; keep tile layer data as one map row per line so diffs stay readable. `maps/starter.tmj` is a provisional map built from the Batch 01 tiles; BUILD.md 2.8 rebuilds it with Batch 03.
