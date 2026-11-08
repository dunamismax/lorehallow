# web/

The lorehallow.com website: a Go program that builds the whole site as static files, which Caddy serves from `/srv/lorehallow/web/current`. It parses [BUILD.md](../BUILD.md) to render the live progress tracker. No page ships any JavaScript ([ADR 0011](../docs/decisions/0011-go-website.md)).

## Two builds of one site

This directory holds two builds of lorehallow.com while it moves to Astro ([ADR 0014](../docs/decisions/0014-astro-website-and-phaser-browser-client.md), BUILD.md Phase 1):

- **The Astro project** (`astro.config.ts`, `src/`, `scripts/`, `tests/`): the new site. Astro components, strict TypeScript, CSS and a Markdown content collection, static output, no UI framework integrations and no browser JavaScript. It builds into `dist-astro/` and isn't deployed yet.
- **The Go generator** (described below): the site that is live. `make deploy-web` still publishes its build until the cutover (BUILD.md 1.9), and it is deleted a week after that.

`make web-parity` builds both from the same commit and compares every file: page HTML and CSS after collapsing whitespace, feeds, `/progress.json` (apart from its timestamp), font tables, and decoded pixels for every image. Any difference not listed in [docs/design/website-parity.md](../docs/design/website-parity.md) fails `make check`. Today they match on all 153 files.

### The Astro project

| Path | What |
|---|---|
| `astro.config.ts` | Static output into `dist-astro/`, `trailingSlash: "always"` and directory URLs (the Go site's URL shape), stylesheets never inlined, prefetch and the dev toolbar off, and Sätteri Markdown with raw HTML dropped. |
| `src/pages/` | One page or build-time endpoint per URL: `progress/[phase].astro`, `devlog/[slug].astro`, `devlog/tags/[tag].astro`, and `progress.json.ts`, `progress.svg.ts`, `devlog/rss.xml.ts`, `devlog/atom.xml.ts`, `sitemap.xml.ts`, `robots.txt.ts`, `favicon.ico.ts`, `apple-touch-icon.png.ts`. |
| `src/layouts/`, `src/components/` | The layout (head metadata, header, nav, footer) and the shared pieces: `Img` (pixel art), `XpBar`, `PostCard`, `SectionHead`, `TagBadge`, `Checkmark`. |
| `src/content.config.ts` | The devlog collection, reading `content/devlog/` in place with the same front matter rules. A bad field fails the build naming the file and the field. |
| `src/lib/buildmd.ts` | The BUILD.md parser, shared by the build and `make buildmd-lint` (`scripts/buildmd-lint.ts`, run by Node directly). |
| `src/lib/` | The rest of the generator, ported from Go ([ADR 0016](../docs/decisions/0016-fonts-and-share-cards-in-typescript.md)): `progress.ts` (the progress model, the git-history achievement feed, `/progress.json`, the badge), `assets.ts` (content-hashed `/assets/`, frame crops, mirrored pairs, responsive whole-number scales), `png.ts`, `pixelfont.ts`, `woff.ts`, `og.ts` (share cards), `theme.ts` (palette and color roles), `sitecss.ts` (the stylesheet), `feeds.ts`, `markdown.ts`, `state.ts`, `site.ts`. |
| `src/integration.ts` | Once every page is rendered, writes the assets the pages registered and the stylesheet, which carries the responsive rules for exactly the images they drew, and points each page at it. In `astro dev` it serves the same files from memory. |
| `scripts/parity.ts` | The parity check. |
| `tests/` | `node --test` suites: the parser, and the parser parity test against `internal/buildmd`. |

```bash
make web-dev       # the Astro site at http://127.0.0.1:4321, drafts included
make web-build     # the Astro site into dist-astro/ (no drafts)
make web-go-dev    # the Go-built site at http://127.0.0.1:4322, drafts included
make web-go-build  # the Go-built site into dist/, exactly what deploys until 1.9
make web-lint      # both: gofmt, vet, golangci-lint, govulncheck; Biome, astro check, tsc, npm audit
make web-test      # both: go test -race; node --test; and the parity check
make buildmd-lint  # lint BUILD.md
```

## The Go generator (current)

This is its own Go module, `github.com/dunamismax/lorehallow/web`, on the Go version in [`toolchain.env`](../toolchain.env). The server's Go conventions apply: standard library first, every new dependency justified in its commit body, and gofmt, vet, golangci-lint, govulncheck and `go test -race` all clean. The other rules are in [AGENTS.md section 7](../AGENTS.md#web-conventions-lorehallowcom): colors from the shared palette tokens, no third-party requests, pixel art at integer scales only, and CSS-only motion that stops under `prefers-reduced-motion`.

| Path | What |
|---|---|
| `cmd/lorehallow-web/` | `build` writes the site to `dist/` (gitignored); `serve` previews it at http://127.0.0.1:4322 (`make web-go-dev`) with drafts, rebuilding on the next request after any source changes. |
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
make web-go-dev    # preview at http://127.0.0.1:4322, drafts included
make web-go-build  # build web/dist/ exactly as it deploys
make web-go-lint   # gofmt, go vet, golangci-lint, govulncheck
make web-go-test   # go test -race, including the output tests below
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

The Astro build gets the same output tests in TypeScript before the cutover (BUILD.md 1.8).
