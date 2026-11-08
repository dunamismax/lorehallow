# 0001: The Lorehallow stack

- **Status:** Accepted. The client row is superseded by [ADR 0014](0014-astro-website-and-phaser-browser-client.md): the game is a Phaser and TypeScript browser client, and native desktop exports are retired. The website, never listed here, is covered by ADRs 0011 and 0014.
- **Date:** 2026-11-06
- **BUILD.md:** 0.2

## Context

The project brief and the owner settled the core technology before any code existed. This record writes those choices down in one place so later ADRs can refer to them. Each one is also a row in the [AGENTS.md decision log](../../AGENTS.md#14-decision-log) (rows 1 to 9 and 18 to 19).

## Decision

| Area | Choice | Why |
|---|---|---|
| Client | Godot 4 with statically typed GDScript | Fast iteration, excellent 2D, simple exports to macOS, Windows and Linux. |
| Server | Go | Built for network servers, safe concurrency, fast builds, single-binary deploys. |
| Transport | WebSocket over TLS (`wss://`) | Reliable and ordered like WoW's TCP model; leaves room for a web client. |
| Protocol | Protobuf, schemas in `proto/`, managed by buf | One versioned source of truth for every message. |
| Persistence | PostgreSQL, from Phase 4 | Characters, inventory and world state need real transactions. |
| Hosting | Self-hosted on the owner's server behind lorehallow.com | Learn deployment, TLS, supervision, logs and backups for real. |
| Network model | Server-authoritative, 20 Hz tick, ~100 ms interpolation, prediction for own movement only | Cheat resistance and a clear model to learn from. |
| Art scale | 32×32 tiles, 32×48 characters, ~32-color palette, 640×360 base viewport, integer scaling | Readable pixel art at 2x, 3x and 4x on common displays. |
| Phase order | 0 scaffold, 1 website, 2+ game | Public, followable progress from day one. |
| Versions | `v0.<phase>.<patch>` until 1.0 | Version numbers map to BUILD.md phases. |
| Git | `main` only, commit and push after every verified step, no AI attribution | Owner's rules. |

## Alternatives considered

- **Unity or a custom engine for the client:** heavier, slower to iterate, or far more work for 2D pixel art.
- **Rust or Node for the server:** Rust is slower to iterate on for a learning project; Node makes CPU-bound tick loops and concurrency harder to reason about.
- **UDP with a custom reliability layer:** teaches a lot, but a slow-paced tab-target MMO doesn't need it, and WebSocket keeps a browser client possible.
- **JSON messages:** easier to read, but no schema, no versioning discipline and larger on the wire.

## Consequences

- There is no official protobuf support for GDScript, so Phase 0 has to find or build a generator (BUILD.md 0.5).
- Two languages share one protocol, so golden round-trip vectors are required for every message.
- Self-hosting means the owner's host details shape the deploy work in Phase 1.
