# web/

The lorehallow.com website: an Astro project that builds the whole site as static files, which Caddy serves from `/srv/lorehallow/web/current`. Astro components, strict TypeScript, CSS and a Markdown content collection; static output, no UI framework integrations and no browser JavaScript ([ADR 0014](../docs/decisions/0014-astro-website-and-phaser-browser-client.md)). It parses [BUILD.md](../BUILD.md) to render the live progress tracker.

Until the Astro cutover (BUILD.md 1.9) the site was built by a Go program; [docs/walkthroughs/phase-01-website-migration.md](../docs/walkthroughs/phase-01-website-migration.md) tells how it moved without changing a URL.

```bash
make web-dev       # http://127.0.0.1:4321, drafts included, rebuilt on change
make web-build     # dist/, exactly what deploys (no drafts)
make web-lint      # Biome, astro check, tsc --noEmit, npm audit
make web-test      # node --test: the parser, unit tests and the output tests on a real build
make buildmd-lint  # lint BUILD.md
make deploy-web    # build, upload a release, switch, check it live, roll back on failure
```

Deploys and rollbacks: [docs/ops/website.md](../docs/ops/website.md). The rules for the site: [AGENTS.md section 7](../AGENTS.md#web-conventions-lorehallowcom).

## Layout

| Path | What |
|---|---|
| `astro.config.ts` | Static output into `dist/` (`LOREHALLOW_ASTRO_OUT` overrides it), `trailingSlash: "always"` and directory URLs, stylesheets never inlined, prefetch and the dev toolbar off, and Sätteri Markdown with raw HTML dropped. |
| `src/pages/` | One page or build-time endpoint per URL: `progress/[phase].astro`, `devlog/[slug].astro`, `devlog/tags/[tag].astro`, `play/index.astro`, and `progress.json.ts`, `progress.svg.ts`, `devlog/rss.xml.ts`, `devlog/atom.xml.ts`, `sitemap.xml.ts`, `robots.txt.ts`, `favicon.ico.ts`, `apple-touch-icon.png.ts`. |
| `src/layouts/`, `src/components/` | The layout (head metadata, header, nav, footer) and the shared pieces: `Img` (pixel art), `XpBar`, `PostCard`, `SectionHead`, `TagBadge`, `Checkmark`. |
| `src/content.config.ts`, `src/lib/devlog-schema.ts` | The devlog collection, reading `content/devlog/` in place. A bad front matter field fails the build naming the file and the field. |
| `src/lib/buildmd.ts` | The BUILD.md parser, shared by the build and `make buildmd-lint` (`scripts/buildmd-lint.ts`, run by Node directly). |
| `src/lib/progress.ts` | The progress model, the git-history achievement feed, `/progress.json` and the badge. |
| `src/lib/assets.ts`, `png.ts` | Content-hashed `/assets/`, frame crops, mirrored pairs, and the responsive whole-number-scale rules. Pixel art is never resampled. |
| `src/lib/pixelfont.ts`, `woff.ts`, `og.ts` | The hand-drawn heading font compiled to TrueType and WOFF, and the 1200×630 share cards ([ADR 0016](../docs/decisions/0016-fonts-and-share-cards-in-typescript.md)). |
| `src/lib/theme.ts`, `sitecss.ts` | Palette tokens and color roles from `art/style/palette.hex`, and the stylesheet built from `static/css/site.css`. |
| `src/lib/state.ts`, `site.ts`, `feeds.ts`, `markdown.ts`, `releases.ts` | Shared build state, site data, feeds and sitemap, Markdown, and release data for /play. |
| `src/integration.ts` | Once every page is rendered, writes the assets the pages registered and the stylesheet, and points each page at it. In `astro dev` it serves them from memory. |
| `static/css/site.css` | The stylesheet, with placeholders `src/lib/sitecss.ts` fills in (fonts, art URLs, the quest map, fireflies, progress widths). |
| `static/fonts/` | The pixel font source and the Go fonts, with their licenses. |
| `content/devlog/` | Devlog posts ([README](content/devlog/README.md)). |
| `data/releases.json` | Versions of the browser game, for /play ([schema](data/README.md)). |
| `assets/art/` | Processed art, mirrored from `art/processed/` by `make sync`. Never edit it by hand. |
| `placeholder/` | The holding page `deploy/bootstrap/app.sh` installs on a host before the first deploy. |
| `tests/` | `node --test` suites: the parser, unit tests, and the output tests. |

## How it fits together

- **Shared state.** Pages call `state()` (`src/lib/state.ts`) for the palette, fonts, icons and the asset registry, and `site()` (`src/lib/site.ts`) for BUILD.md progress and the devlog. Both live on `globalThis`, so every bundled page and the integration share one build.
- **Pixel art.** Pages draw art with `<Img src="web/hero_scene.png" alt="..." k={3} />`. `k` fixes the scale; `max` makes the image responsive, stepping through whole-number scales as the viewport grows. `sheet.png#frame` crops a named frame from a sheet's sidecar JSON, and `file.png#mirror` adds a mirrored copy so a layer tiles edge to edge. An image without alt text fails the build; pass `"decorative"` for decorative art.
- **Colors.** `palette.hex` becomes `--c-<name>` custom properties and the roles in `src/lib/theme.ts` (`--text`, `--bg`, `--text-link`, ...). The stylesheet sets `color` and `background-color` only through roles, and every text-on-background pair it uses is in `PAIRS` and checked for WCAG AA.
- **Progress.** `/progress/`, the phase pages, `/progress.json` and `/progress.svg` come from BUILD.md. The achievement feed walks `git log -p BUILD.md`, so builds need full history (a full clone, not a shallow one).
- **The stylesheet comes last.** It carries a responsive rule for every image the pages drew, so the integration writes it in `astro:build:done` and swaps its hashed URL into every page.

## Output tests

`make web-test` builds the real site (with an extra draft post) and checks:

- no `<script>`, `<iframe>`, inline event handlers, `style` attributes or `javascript:` URLs on any page, and no JavaScript file in the build
- every image has alt text and dimensions that are a whole-number multiple of the art, and every responsive rule scales by whole numbers
- every internal link, fragment and CSS `url()` resolves, and no page loads anything from another origin
- the landing page, with every image, font and stylesheet it can load, stays under 1 MB
- every page has a title, description, canonical URL, Open Graph tags with a 1200×630 card, and links to GitHub and Codeberg
- the feeds and sitemap are well formed, and the draft appears in no file of the build

Unit tests cover the palette and contrast, color roles in the stylesheet, alt text, PNG round trips, the pixel font, the achievement feed, devlog front matter and release data. Change the parser, its tests and the contract in BUILD.md together.
