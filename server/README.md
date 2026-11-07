# server/

The Go game server, module `github.com/dunamismax/lorehallow/server`.

| Path | What goes there |
|---|---|
| `cmd/lorehallow-server/` | The game server binary. |
| `cmd/lorehallow-protovectors/` | Writes the golden protocol vectors to `proto/testdata/` (run by `make proto`). |
| `internal/config/` | Flags and `LOREHALLOW_*` environment variables. |
| `internal/logging/` | `log/slog` setup: text locally, JSON in production. |
| `internal/httpapi/` | Plain HTTP endpoints. |
| `internal/buildinfo/` | Version and commit, injected with `-ldflags`. |
| `internal/pb/` | Generated protobuf code (`make proto`). Never edit by hand. |
| `internal/protovectors/` | Golden vector definitions and round-trip tests. |
| `migrations/` | SQL migrations, from Phase 4 on. |

## Running

```bash
make server        # builds bin/lorehallow-server and runs it on 127.0.0.1:7420
curl localhost:7420/healthz
curl localhost:7420/version
```

Stop it with Ctrl-C. SIGINT and SIGTERM trigger a graceful shutdown bounded by the shutdown timeout.

## Configuration

Every setting has a flag and an environment variable. Flags win over the environment, and the environment wins over the defaults. The defaults suit local development.

| Flag | Environment | Default | Meaning |
|---|---|---|---|
| `-addr` | `LOREHALLOW_ADDR` | `127.0.0.1:7420` | Listen address for HTTP and WebSocket traffic |
| `-log-format` | `LOREHALLOW_LOG_FORMAT` | `text` | `text` locally, `json` in production |
| `-log-level` | `LOREHALLOW_LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error` |
| `-shutdown-timeout` | `LOREHALLOW_SHUTDOWN_TIMEOUT` | `10s` | How long graceful shutdown may take |

[`.env.example`](../.env.example) lists the variable names.

## Endpoints

| Endpoint | Returns |
|---|---|
| `GET /healthz` | `ok` with status 200 while the process is serving |
| `GET /version` | JSON: `version`, `commit` and `protocol_version` |

## Build identity

`make server` and `make server-build` set `buildinfo.Version` (from `git describe`) and `buildinfo.Commit` with `-ldflags -X`. A plain `go build` reports `dev` and `unknown`.

## Architecture and scale

The server is the project's main engineering achievement. It aims to host hundreds to thousands of simultaneous, actively connected players; [docs/design/scaling.md](../docs/design/scaling.md) sets the measurable targets and [ADR 0005](../docs/decisions/0005-scale-and-backend-architecture.md) the design. Thousands is a target, not a capability: only numbers measured and recorded in [docs/perf/](../docs/perf/) count.

In short, one process with many zones:

- **One goroutine owns each zone.** Its tick loop is the only code that touches the zone's state. The rules inside it (`internal/sim`) are pure and deterministic.
- **Channels at the edges.** Each connection has a reader goroutine that validates messages and puts them on the zone's bounded inbox, and a writer goroutine that drains a bounded send queue.
- **Slow clients are evicted, never waited for.** A full send queue disconnects the client; the tick loop never blocks on a socket, a database or a lock.
- **Encode once, then fan out.** Each entity's update is serialized once per tick and shared by every player who can see it.
- **Spatial grid interest management** (Phase 3), quantized delta snapshots and per-player bandwidth budgets for crowds (Phase 7).
- **Zone handoff by message** (Phase 11), so zones can later move into separate processes and machines (Phase 20) when a measurement shows one process is the limit.

`lorehallow-bot` (Phase 2) is the headless load generator, and `make loadtest SCENARIO=<name>` runs the scaling.md scenarios against a real server.

## Conventions

See [AGENTS.md section 7](../AGENTS.md#server-conventions-go) and the [backend engineering standards](../AGENTS.md#backend-engineering-standards): pure, deterministic simulation; one goroutine owns a zone's state; bounded queues everywhere; nothing blocks a tick; table-driven tests with `-race`; no sleeps in tests (fake clock or `testing/synctest`); fuzzing, allocation benchmarks and leak checks for anything on the hot path or facing clients.
