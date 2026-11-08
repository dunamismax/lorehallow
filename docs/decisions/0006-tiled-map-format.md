# 0006: Maps are Tiled JSON in content/maps

- **Status:** Accepted. The map format stands. The client that imports it changes from Godot (TileMapLayers) to the Phaser browser client ([ADR 0014](0014-astro-website-and-phaser-browser-client.md), BUILD.md 2.2).
- **Date:** 2026-11-07
- **BUILD.md:** 2.2

## Context

The server needs each map's collision and spawn points; the client needs its tiles to draw. If the two read different files, they drift, and a player sees an open path the server treats as a wall. Maps also need a real editor: hand-typing hundreds of tile numbers doesn't scale to a 200×150 zone (Phase 7).

## Decision

One file per map, `content/maps/<map_id>.tmj`, in [Tiled](https://www.mapeditor.org/)'s JSON map format. The server reads it for collision and spawns (`server/internal/mapdata`); the client imports the same file, synced to `client/assets/content/maps/`, and builds its TileMapLayers from it. `make check` validates every map.

Conventions:

- **Orthogonal, finite, 32×32 tiles**, render order right-down, tile layer data as plain JSON arrays (no compression), so diffs stay readable.
- **Tilesets are external** `content/tilesets/<name>.tsj` files whose `name` is the processed art id: tileset `terrain_base` is `art/processed/tiles/terrain_base.png`. The `.tsj` image path points there for the Tiled editor; the client finds `res://assets/art/tiles/<name>.png` by name, so the relative path never matters at runtime.
- **Tile layers are drawn** bottom to top in file order, except the layer named **`collision`**: any non-empty tile in it is solid, and it is never drawn. It must be the same size as the map.
- **An object layer named `spawns`** holds point objects with type (or class) `spawn`. Their position is in pixels; the server converts to subpixels (16 per pixel). At least one is required.
- Tiled's flip flags (the top bits of a tile id) are masked off; flipped tiles render flipped on the client and don't change collision.
- Unknown layers and properties are ignored by the server, so maps can carry editor-only notes.

The validator (`lorehallow-content validate`, part of `make content-lint`) checks: the JSON parses with these conventions, every tileset file and its image exist and match the tile grid, every tile id is in range, the collision layer is present and sized, at least one spawn exists and every spawn is inside the map and not inside a solid tile.

## Alternatives considered

- **LDtk.** A good editor with a cleaner JSON format, but Tiled is more widely known, has a Godot importer ecosystem, and its format is stable and documented.
- **Our own text format** (an ASCII grid like the art sources). Fine for a 30×20 test map, painful for real zones with several layers.
- **Godot's `.tscn` TileMap as the source.** The Go server would have to parse Godot's resource format, and the map would live inside the client.
- **Embedded tilesets in each map.** Simpler to load, but every map would carry its own copy of the tileset definitions and they would drift.

## Consequences

- The owner or an agent can edit maps in Tiled; agents can also generate or edit the JSON directly.
- Map size grows the server's collision grid only (one byte per tile); a 200×150 zone is 30 KB.
- If chunked loading is needed for big maps (Phase 7 ADR), it can be done on top of this format by splitting layers into chunks at import time.
