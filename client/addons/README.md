# client/addons/

Third-party Godot addons. They are installed by `make setup` at the versions and checksums pinned in `toolchain.env`, and are not committed.

| Addon | Used for |
|---|---|
| `gut/` | Unit tests (`make test`) |
| `godobuf/` | GDScript protobuf code generation (`make proto`), see ADR 0003 |

Addons are excluded from formatting, linting and GDScript warnings.
