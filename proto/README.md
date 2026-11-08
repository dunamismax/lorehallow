# proto/

Protobuf schemas shared by the server and the client, managed with [buf](https://buf.build).

| Path | What |
|---|---|
| `lorehallow/v1/*.proto` | The schemas, package `lorehallow.v1` |
| `PROTOCOL_VERSION` | The protocol version, a single positive integer |
| `buf.yaml` | Module, lint (`STANDARD`) and breaking (`FILE`) rules |
| `buf.gen.yaml` | Go code generation (GDScript is generated separately, see below; TypeScript will be added here, see below) |
| `testdata/` | Golden vectors: `<name>.bin` (exact bytes) and `<name>.json` (what they mean) |

## Rules

- One `Envelope` per binary WebSocket message. Every message type is a case of its `payload` oneof.
- Never reuse or renumber a field number. Removed fields become `reserved`.
- Every Envelope payload needs at least one golden vector (`server/internal/protovectors`). A test fails until it has one.
- Hot messages (`PlayerInput`, `Snapshot`) keep field numbers 1 to 15 (one-byte tags), use `sint32` for values that can be negative, and integers instead of floats. Byte budgets per message are in [docs/design/protocol.md](../docs/design/protocol.md).
- Names godobuf can't handle: a field called `message`, a message named like a Godot class (`Input`, `Node`, `Object`, `Resource`, ...), and `reserved` statements at all, so `gen-proto.sh` strips them from the copy godobuf compiles (they generate no code; buf still enforces them). Reserve numbers only. These limits go away with the GDScript bindings (BUILD.md 2.10); existing names such as `PlayerInput` stay.
- Golden JSON descriptions use protojson with proto field names and enum numbers, so the GDScript test can compare enums directly.

## PROTOCOL_VERSION

`PROTOCOL_VERSION` holds one integer. `make proto` turns it into `pb.ProtocolVersion` (Go) and `ProtocolVersion.CURRENT` (GDScript), and later a TypeScript constant. The server rejects a client whose version differs, with an "update required" message (Phase 2).

Bump it on any incompatible change: removing or renaming a field or message, changing a field's type, or changing what a field means. Adding a new field or a new payload case is compatible and needs no bump.

`make check` runs `buf breaking` against `main`. A breaking change passes only if `PROTOCOL_VERSION` was bumped in the same change; that is what "planned" means.

## Workflow

```bash
# edit lorehallow/v1/*.proto (and server/internal/protovectors for new messages)
make proto        # regenerates Go, GDScript, version constants and golden vectors
make check        # lint, breaking check, staleness check, round-trip tests on both sides
```

Generated code is committed and never edited by hand:

- Go: `server/internal/pb/` via buf and `protoc-gen-go` (pinned in `server/go.mod`).
- GDScript: `client/net/pb/<file>_pb.gd` via godobuf, run headless by `tools/scripts/gen-proto.sh`. See [ADR 0003](../docs/decisions/0003-gdscript-protobuf.md), now superseded.

## Planned: TypeScript bindings

The Phaser browser client ([ADR 0014](../docs/decisions/0014-astro-website-and-phaser-browser-client.md)) needs TypeScript bindings. BUILD.md 2.6 adds a TypeScript plugin to `buf.gen.yaml`, pinned in the Node workspace, so `make proto` writes them into the client (committed, marked generated) with a `PROTOCOL_VERSION` constant. A test decodes and re-encodes every vector in `testdata/` byte for byte and checks its JSON description. Until 2.10 removes Godot, `make proto` generates Go, GDScript and TypeScript, and a protocol change updates all three in one commit.
