# web/

The lorehallow.com website: a Go program that builds the whole site as static files, which Caddy serves from `/srv/lorehallow/web/current`. It parses [BUILD.md](../BUILD.md) to render the live progress tracker. No page ships any JavaScript ([ADR 0011](../docs/decisions/0011-go-website.md)).

This is its own Go module, `github.com/dunamismax/lorehallow/web`, on the Go version in [`toolchain.env`](../toolchain.env). The server's Go conventions apply: standard library first, every new dependency justified in its commit body, and gofmt, vet, golangci-lint, govulncheck and `go test -race` all clean. The other rules are in [AGENTS.md section 7](../AGENTS.md#web-conventions-lorehallowcom): colors from the shared palette tokens, no third-party requests, pixel art at integer scales only, and CSS-only motion that stops under `prefers-reduced-motion`.

| Path | What |
|---|---|
| `internal/buildmd/` | The BUILD.md parser. It implements the format contract at the top of BUILD.md: phases, metadata, sections, items, tags and checked state, plus every format problem with its line number. Tested on malformed input, the real BUILD.md, and a fuzz target. |
| `cmd/buildmd-lint/` | `make buildmd-lint`: fails on any format problem. Part of `make check`. |
| `assets/art/` | Processed art, mirrored from `art/processed/` by `make sync`. Never edit it by hand. |
| `placeholder/` | The holding page served before the real site's first deploy. |

Built in Phase 1 (BUILD.md 1.2 to 1.9):

| Path | What |
|---|---|
| `cmd/lorehallow-web/` | `build` writes the site to `dist/` (gitignored); `serve` previews it locally with drafts and rebuilds on change. |
| `templates/`, `static/` | `html/template` pages, CSS and self-hosted fonts, embedded with `go:embed`. |
| `content/devlog/` | Devlog posts: Markdown with front matter, rendered with goldmark. |
| `data/releases.json` | Release listings for /downloads, written later by the release workflow. |

Change the parser, its tests and the contract in BUILD.md together.

```bash
make web-lint      # gofmt, go vet, golangci-lint, govulncheck
make web-test      # go test -race
make buildmd-lint  # lint BUILD.md
```
