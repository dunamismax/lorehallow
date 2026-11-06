# proto/

Protobuf schemas shared by the server and the client, managed with [buf](https://buf.build).

- Schemas live in `lorehallow/v1/`, package `lorehallow.v1`.
- `testdata/` holds golden vectors that both sides decode and re-encode byte for byte.
- Run `make proto` after any schema change. It regenerates Go code into `server/internal/pb/` and GDScript into `client/net/pb/`.
