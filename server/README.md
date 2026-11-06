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

## Conventions

See [AGENTS.md section 7](../AGENTS.md#server-conventions-go): pure, deterministic simulation; one goroutine owns a zone's state; table-driven tests with `-race`; no sleeps in tests.
