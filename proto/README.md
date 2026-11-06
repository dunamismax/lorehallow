# proto/

Protobuf schemas shared by the server and the client, managed with [buf](https://buf.build).

| Path | What |
|---|---|
| `lorehallow/v1/*.proto` | The schemas, package `lorehallow.v1` |
| `PROTOCOL_VERSION` | The protocol version, a single positive integer |
| `buf.yaml` | Module, lint (`STANDARD`) and breaking (`FILE`) rules |
| `buf.gen.yaml` | Go code generation (GDScript is generated separately, see below) |
| `testdata/` | Golden vectors: `<name>.bin` (exact bytes) and `<name>.json` (what they mean) |

## Rules

- One `Envelope` per binary WebSocket message. Every message type is a case of its `payload` oneof.
- Never reuse or renumber a field number. Removed fields become `reserved`.
- Every Envelope payload needs at least one golden vector (`server/internal/protovectors`). A test fails until it has one.

## PROTOCOL_VERSION

`PROTOCOL_VERSION` holds one integer. `make proto` turns it into `pb.ProtocolVersion` (Go) and `ProtocolVersion.CURRENT` (GDScript). The server rejects a client whose version differs, with an "update required" message (Phase 2).

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
- GDScript: `client/net/pb/<file>_pb.gd` via godobuf, run headless by `tools/scripts/gen-proto.sh`. See [ADR 0003](../docs/decisions/0003-gdscript-protobuf.md).
