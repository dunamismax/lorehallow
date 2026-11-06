# server/

The Go game server, module `github.com/dunamismax/lorehallow/server`.

| Path | What goes there |
|---|---|
| `cmd/` | One directory per binary: `lorehallow-server` now, bots and admin tools later. |
| `internal/` | Packages. `internal/pb/` is generated protobuf code (`make proto`), never edited by hand. |
| `migrations/` | SQL migrations, from Phase 4 on. |

Run it with `make server`. Configuration is documented below once the server skeleton lands.
