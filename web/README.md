# web/

The lorehallow.com website: a Go program that builds the whole site as static files, which Caddy serves from `/srv/lorehallow/web/current`. It parses [BUILD.md](../BUILD.md) to render the live progress tracker. No page ships any JavaScript ([ADR 0011](../docs/decisions/0011-go-website.md)).

## Planned: the move to Astro

This directory is being rebuilt as an Astro site ([ADR 0014](../docs/decisions/0014-astro-website-and-phaser-browser-client.md), BUILD.md Phase 1). **Everything below the next heading describes the Go generator that is in the checkout and live today.** None of the Astro project exists yet.

The plan, in order:

1. **Parity inventory** (BUILD.md 1.2): every URL, redirect, generated asset, meta tag, output test and budget of this site, written to `docs/design/website-parity.md` with how the Astro build is checked against each.
2. **A local Astro build** next to this generator: Astro components, strict TypeScript, CSS and Markdown content collections, static output, no UI framework integrations and no browser JavaScript. It writes to its own output folder, so both builds can be compared. The devlog collection reads `content/devlog/` in place.
3. **One TypeScript BUILD.md parser** shared by Astro's build and `make buildmd-lint`, checked against `internal/buildmd` on the real BUILD.md and its fixtures. `/progress.json`, `/progress.svg`, the feeds and the sitemap become build-time endpoints.
4. **Parity and output checks**: the tests listed under [Output tests](#output-tests), ported to TypeScript, plus a check that diffs the two builds. The fonts and share cards, generated in Go today (`internal/pixelfont`, `internal/woff`, `internal/site/og.go`), are ported or moved to a Go tool, as the 1.2 ADR decides.
5. **Play replaces Downloads** (1.6): `/play/` says whether the game is playable, and `/downloads/` redirects to it. `data/releases.json` is replaced by release data with no platform files.
6. **Cutover** (1.9), after the owner approves: `make deploy-web` publishes the Astro build with the same atomic release and rollback model. The Go module is deleted only after the cutover has held.

Until the cutover, the `make` targets below run this Go generator, and `make check` runs both sites' checks once the Astro project exists.

## The Go generator (current)

This is its own Go module, `github.com/dunamismax/lorehallow/web`, on the Go version in [`toolchain.env`](../toolchain.env). The server's Go conventions apply: standard library first, every new dependency justified in its commit body, and gofmt, vet, golangci-lint, govulncheck and `go test -race` all clean. The other rules are in [AGENTS.md section 7](../AGENTS.md#web-conventions-lorehallowcom): colors from the shared palette tokens, no third-party requests, pixel art at integer scales only, and CSS-only motion that stops under `prefers-reduced-motion`.

| Path | What |
|---|---|
| `cmd/lorehallow-web/` | `build` writes the site to `dist/` (gitignored); `serve` previews it at http://127.0.0.1:4321 with drafts, rebuilding on the next request after any source changes. |
| `internal/site/` | The generator: pages, the progress model and achievement feed (from `git log` of BUILD.md), devlog, feeds, sitemap, share images, fingerprinted assets, palette tokens and color roles. Its tests check the built output. |
| `internal/buildmd/` | The BUILD.md parser. It implements the format contract at the top of BUILD.md: phases, metadata, sections, items, tags and checked state, plus every format problem with its line number. Tested on malformed input, the real BUILD.md, and a fuzz target. |
| `internal/pixelfont/`, `internal/woff/` | Compile the hand-drawn heading font into TrueType and WOFF ([ADR 0012](../docs/decisions/0012-website-fonts.md)). |
| `cmd/buildmd-lint/` | `make buildmd-lint`: fails on any format problem. Part of `make check`. |
| `templates/` | `html/template` pages: `layout.html`, `partials.html`, and one file per page type in `pages/`. Embedded with `go:embed`. |
| `static/css/site.css` | The stylesheet, a `text/template` filled in with font and art URLs. Colors come from the palette tokens and roles the generator prepends. |
| `static/fonts/` | The pixel font source and the Go fonts, with their licenses. |
| `content/devlog/` | Devlog posts: Markdown with front matter ([README](content/devlog/README.md)). |
| `data/releases.json` | Release listings for /downloads ([schema](data/README.md)). |
| `assets/art/` | Processed art, mirrored from `art/processed/` by `make sync`. Never edit it by hand. |
| `placeholder/` | The holding page served before the real site's first deploy. |

```bash
make web-dev       # preview at http://127.0.0.1:4321, drafts included
make web-build     # build web/dist/ exactly as it deploys
make web-lint      # gofmt, go vet, golangci-lint, govulncheck
make web-test      # go test -race, including the output tests below
make buildmd-lint  # lint BUILD.md
```

## How it fits together

`site.Build` reads BUILD.md, `art/style/palette.hex`, the processed art, the devlog and the release data, and returns every file of the site in memory. `build` writes that to `dist/`; `serve` serves it from memory.

- **Assets.** Everything under `/assets/` has a content hash in its name, so Caddy caches it forever. That covers the stylesheet, the fonts, every image and the share cards.
- **Pixel art.** Templates draw art with `{{img "web/hero_scene.png" "alt text" ...}}`. `k=3` fixes the scale, and `max=960` makes the image responsive: the generator writes a class that steps it up through whole-number scales as the viewport grows. `sheet.png#frame` crops a named frame from a sheet's sidecar JSON, and `file.png#mirror` adds a mirrored copy so a layer tiles edge to edge. An image without alt text fails the build; pass `"decorative"` for decorative art.
- **Colors.** The generator turns `palette.hex` into `--c-<name>` custom properties, then defines the roles in `internal/site/theme.go` (`--text`, `--bg`, `--text-link`, ...). The stylesheet sets `color` and `background-color` only through roles, and every text-on-background pair it uses is listed in `Pairs` and checked for WCAG AA.
- **Progress.** `/progress/`, the phase pages, `/progress.json` and `/progress.svg` come from BUILD.md. The achievement feed walks `git log -p BUILD.md` to find the commit that checked each box, so builds need full history (a full clone, not a shallow one).
- **Share images.** Every page gets a 1200×630 Open Graph card: `og_card` at exactly 5x with the page title drawn in the pixel font.

## Output tests

`make web-test` builds the real site and checks the result:

- no `<script>`, `<iframe>`, inline event handlers, `style` attributes or `javascript:` URLs on any page
- every image has alt text and explicit dimensions that are a whole-number multiple of the art, and every responsive rule scales by whole numbers
- every internal link, fragment and CSS `url()` resolves, and no page loads anything from another origin
- the landing page, with every image, font and stylesheet it can load, stays under 1 MB
- every text color pair meets WCAG AA, and the stylesheet sets colors only through roles
- every page has a title, description, canonical URL, Open Graph tags with a 1200×630 card, and links to GitHub and Codeberg
- the feeds and sitemap are valid XML, drafts never reach `build`, and releases.json is validated

Change the parser, its tests and the contract in BUILD.md together.

```bash
make web-lint      # gofmt, go vet, golangci-lint, govulncheck
make web-test      # go test -race
make buildmd-lint  # lint BUILD.md
```
