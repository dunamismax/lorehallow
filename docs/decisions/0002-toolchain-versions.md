# 0002: Toolchain versions and how they are pinned

- **Status:** Accepted. Node.js was later removed from the toolchain by [ADR 0011](0011-go-website.md), and returns for frontend development, builds and CI under [ADR 0014](0014-astro-website-and-phaser-browser-client.md), pinned in BUILD.md 1.2. Godot, gdtoolkit, GUT and godobuf leave when the Godot client is removed (BUILD.md 2.10). The rest stands.
- **Date:** 2026-11-06
- **BUILD.md:** 0.3

## Context

Agents and CI must build with the same tools, or "works on my machine" bugs creep in. BUILD.md 0.3 asks for the latest stable release of each tool, pinned, and checked by `make doctor`.

## Decision

Pin these versions, the latest stable releases on 2026-11-06:

| Tool | Version | How it is installed |
|---|---|---|
| Go | 1.27.3 | System (Homebrew, or the go.dev tarball) |
| Godot | 4.7.2-stable | System (Homebrew cask, or the official binary) |
| Node.js | 26.13.0 (LTS "Lithium") | System (Homebrew, mise or nvm); used from Phase 1 |
| Python | 3.14.9 | Managed by uv |
| uv | 0.13.7 | Standalone installer |
| buf | 1.75.0 | System |
| golangci-lint | 2.15.0 | System (official binary, not `go tool`, as its docs recommend) |
| gdtoolkit | 4.5.0 | `uv tool install` |
| GUT | 9.7.1 | `make setup`, from the release tarball, SHA-256 checked |
| godobuf | 0.7.0 | `make setup`, from the release tarball, SHA-256 checked |
| protoc-gen-go | 1.36.12 | `tool` directive in `server/go.mod` |
| govulncheck | 1.9.0 | `tool` directive in `server/go.mod` |

The single source of truth is [`toolchain.env`](../../toolchain.env) at the repository root. The Makefile includes it, `tools/scripts/doctor.sh` checks against it, `tools/scripts/install-addons.sh` installs from it and CI reads it. AGENTS.md section 13 mirrors it for humans.

`make doctor` accepts a tool when the major and minor versions match the pin and the patch is the same or newer (Node: same major, same or newer). Patch releases are bug fixes, and refusing them would make every machine fail the day a patch ships. A minor bump is a deliberate change: update `toolchain.env`, AGENTS.md section 13 and this table's successor ADR in one commit.

GUT 9.7.x is the line built for Godot 4.7 (9.7.0 adapted to Godot 4.7's stricter return-type checks), so it is pinned rather than the 9.6 line.

Go tools that are libraries of the server module (`protoc-gen-go`, `govulncheck`) use `go.mod` `tool` directives, so `go.sum` pins them exactly and `go tool <name>` runs them with no global install.

## Alternatives considered

- **mise or asdf for everything:** one tool manager is tidy, but it adds a dependency the owner may not use, and Godot and GUT are not well served by it.
- **Exact version matches in `make doctor`:** strictest, but breaks on every patch release with no real benefit.
- **Vendoring GUT and godobuf in git:** simpler fresh clones, but adds megabytes of third-party code to every diff and history. A checksummed download is just as reproducible.
- **Nix or a dev container:** fully reproducible, but heavy for a hobby project on macOS with a GUI engine.

## Consequences

- One file to edit for any version bump.
- CI must install the same versions; it reads `toolchain.env` rather than hard-coding them.
- Python is pinned to 3.14 in `tools/art/.python-version` and `pyproject.toml`.
