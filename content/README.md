# content/

Canonical game data, read by the server and synced into the client (`make sync`).

- YAML for items, mobs, quests and so on; Tiled JSON (`.tmj`) for maps unless an ADR says otherwise.
- IDs are `snake_case`, stable and never reused.
- Every content type gets a validator that runs in `make check`.
