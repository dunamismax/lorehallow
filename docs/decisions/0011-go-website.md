# 0011: The website is a Go static site generator, with no JavaScript

- **Status:** Superseded by [ADR 0014](0014-astro-website-and-phaser-browser-client.md) (2026-11-07): the website moves to Astro with strict TypeScript, still with no browser JavaScript. This generator stays live until the Astro build passes its parity checks and is cut over (BUILD.md 1.9). It had superseded the Astro and TypeScript choices in decision log rows 10 and 32.
- **Date:** 2026-11-07
- **BUILD.md:** 1.2

## Context

Phase 1 planned lorehallow.com as an Astro static site in TypeScript, and the BUILD.md parser and linter were written in TypeScript and run by Node. Nothing else in the repository used Node.

The owner is moving every project they maintain to Go and dropping Node and browser JavaScript wherever possible. On the shared host, each Astro/Node site uses 52 to 86 MB of memory, while Lorehallow's Go game server uses 3 MB. Lorehallow's server is already Go, and Caddy itself is written in Go.

The site has no need for Node. It is static: pages built from Markdown, BUILD.md, a few data files and processed art, served by Caddy from `/srv/lorehallow/web/current`. Every planned feature works without JavaScript, including the progress tracker, quest map, devlog, feeds, downloads page and social cards. The one piece the plan called "progressive enhancement" was ambient motion (parallax, fireflies, lantern flicker), and CSS does all of it.

## Decision

**lorehallow.com is built by a Go program in `web/`, and it ships no JavaScript.**

- **Module.** `web/` is its own Go module, `github.com/dunamismax/lorehallow/web`, on the Go version pinned in `toolchain.env`, separate from the server so the web and server lanes don't share a `go.mod`.
- **Generator.** `cmd/lorehallow-web` has two subcommands. `build` writes the finished site to `web/dist/`. `serve` builds into memory and serves it locally with a rebuild on change (`make web-dev`). The output is plain files that Caddy serves, so there is no web process to run, supervise or patch on the host.
- **Templates.** `html/template` from the standard library, with the templates, CSS and fonts in the binary through `go:embed`. It escapes by context and needs no code generation step. templ was considered and isn't needed for a static site this size.
- **Markdown.** Devlog posts are Markdown files with YAML front matter in `web/content/devlog/`, rendered by [goldmark](https://github.com/yuin/goldmark). It is the site's one expected third-party dependency, because Markdown parsing isn't in the standard library. Anything else needs a reason in its commit body, as on the server.
- **BUILD.md.** The parser is `web/internal/buildmd`, a line-for-line port of the TypeScript one. It gave identical results on the real BUILD.md and on 200 randomly broken copies of it. The generator and `cmd/buildmd-lint` (in `make check`) share it.
- **Feeds and metadata.** RSS and Atom are written with `encoding/xml`, and so are `sitemap.xml` and the `/progress.svg` badge. `/progress.json` is written with `encoding/json`.
- **Images.** The generator reads each image's size with `image.DecodeConfig` and writes explicit `width` and `height`, always a whole-number multiple of the art.
- **Fingerprinting.** CSS, fonts and images are written to `/assets/` under names that include a content hash, so Caddy caches them forever (`deploy/caddy/lorehallow.caddy`).
- **No JavaScript.** No page includes a `<script>`. Caddy's Content-Security-Policy says `script-src 'none'`, and a generator test fails the build if any page has a script tag or an inline event handler. Motion is CSS animation only, and it is switched off under `prefers-reduced-motion`.
- **Quality checks without Node.** Go tests enforce what can be measured from the output:
  - the page-weight budget (landing page under 1 MB)
  - no scripts
  - alt text on every image
  - explicit image dimensions
  - internal links that resolve
  - WCAG AA contrast for the palette pairs the CSS uses

  Lighthouse is a Node tool, so it isn't in the toolchain. It runs from Chrome DevTools against the live site, and the results are recorded in `docs/perf/`.

## Alternatives considered

- **Astro (the old plan).** It's a good static site generator, but it keeps Node, npm, a lockfile, `node_modules` and npm audits in the toolchain for one site, and nothing in the site needs them.
- **Hugo.** It's written in Go and is a single binary. But its templates are another language, and the progress tracker needs the BUILD.md parser, quest-map layout and achievement feed as real code. That code would end up as Go anyway, outside Hugo.
- **A Go web server that renders each request.** Nothing on the site changes between pushes except the progress data, and that is rebuilt at every deploy. Static files are faster, cache better, and leave no process on the host.
- **templ.** It gives type-safe components, but adds a code generator and a dependency. `html/template` covers a site this size and is in the standard library.

## Consequences

- Node, npm and TypeScript leave the toolchain: `toolchain.env`, `make doctor`, `make setup`, CI and the AGENTS.md toolchain table. The whole repository builds with Go, Python (the art tools) and Godot.
- `make web-lint` is gofmt, `go vet`, golangci-lint and govulncheck, and `make web-test` is `go test -race`, the same gates as the server. `npm audit` goes away; govulncheck covers both modules.
- Synced art moves from `web/src/assets/art/` to `web/assets/art/`, and devlog posts and release data will live in `web/content/devlog/` and `web/data/releases.json`.
- The site can't use client-side features: no search box, no theme toggle that needs a script, no live counters. A dark/light theme follows `prefers-color-scheme`. Live numbers, such as players online, are built into the page at build time or left to a later decision.
- Godot's web export (planned in Phase 15) is the game client, not the website, and needs its own loader script. When that lands, it gets its own page and a CSP exception, decided in its own ADR.
