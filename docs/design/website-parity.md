# Website parity inventory

> **History.** This inventory guided the Astro migration (BUILD.md Phase 1). The Go-built site and the parity check that compared it with the Astro build were removed once the cutover was verified; the URL, redirect and metadata rules below still hold, and the output tests and the deploy health check still enforce them.

Everything lorehallow.com publishes today, from the Go-built site that is live (`web/dist/` from `make web-go-build`, the Caddy site file `deploy/caddy/lorehallow.caddy`, and the live responses), and how the Astro build is checked against each item before the cutover (BUILD.md 1.2, 1.8 and 1.9, [ADR 0014](../decisions/0014-astro-website-and-phaser-browser-client.md)).

How to read the "Checked by" column:

- **parity** is the parity check (BUILD.md 1.8): it builds both sites from the same commit and compares them.
- **output** is one of the output tests ported to TypeScript, run on the Astro build in `make web-test`.
- **deploy** is the deploy's live health check (`deploy/web/deploy.sh`), run against the host after every switch.
- **unit** is a unit test of a TypeScript module.

Taken on 2026-11-07 at commit `3e7c872`.

## URLs

Every URL below keeps answering exactly as it does now. A page that moves gets a permanent redirect in the Caddy site file.

| URL | What | Checked by |
|---|---|---|
| `/` | Landing page | parity, output, deploy |
| `/world/` | The world, with the art gallery | parity, output |
| `/progress/` | Progress tracker: XP bar, quest map, waiting on the owner, achievement feed | parity, output, deploy |
| `/progress/phase-0/` to `/progress/phase-20/` | One page per BUILD.md phase (21 pages) | parity, output |
| `/devlog/` | Devlog index | parity, output |
| `/devlog/<slug>/` | One page per published post (none published yet; three drafts) | parity, output |
| `/devlog/tags/<tag>/` | One page per tag of a published post | parity, output |
| `/devlog/rss.xml` | RSS 2.0 feed | parity, output, deploy |
| `/devlog/atom.xml` | Atom feed | parity, output, deploy |
| `/about/` | About, with `#how-its-made` | parity, output |
| `/faq/` | FAQ | parity, output |
| `/404.html` | The 404 page, served by Caddy for any missing path with HTTP 404 | parity, output, deploy |
| `/progress.json` | Machine-readable progress (README badge, uptime check) | parity, output, deploy |
| `/progress.svg` | Progress badge in the pixel font (README) | parity, deploy |
| `/sitemap.xml` | Every page except the 404 | parity, output |
| `/robots.txt` | Allows everything, points at the sitemap | parity |
| `/favicon.ico` | 16 and 32 px PNGs packed in an ICO | parity (byte for byte) |
| `/apple-touch-icon.png` | The 180 px favicon | parity (byte for byte) |
| `/assets/<name>.<hash10>.<ext>` | Fingerprinted CSS, fonts, art and share cards (below) | parity, output, deploy |

Changed during the migration, in both builds at once so the parity check still compares them (BUILD.md 1.6):

| URL | What | Checked by |
|---|---|---|
| `/play/` | Replaces Downloads: says plainly whether the game is playable, from `web/data/releases.json` | parity, output, deploy |
| `/downloads` and `/downloads/` | 301 to `/play/`, in the Caddy site file, kept for good | deploy |

## Redirects and status codes

All served by Caddy, not by the build. They don't change at the cutover except for the `/downloads/` redirect, added in 1.6.

| Request | Response | Checked by |
|---|---|---|
| `http://lorehallow.com/...` | Redirect to `https://` (Caddy automatic HTTPS) | deploy |
| `https://www.lorehallow.com/<uri>` | 301 to `https://lorehallow.com/<uri>` | deploy |
| A directory without its slash, such as `/about` | 308 to `/about/` (Caddy `file_server`) | deploy |
| A missing path | HTTP 404 with the body of `/404.html` | deploy |
| `/assets/*` | `Cache-Control: public, max-age=31536000, immutable` | deploy |
| Everything else | `Cache-Control: public, max-age=0, must-revalidate` | deploy |
| Every response | HSTS, `nosniff`, `X-Frame-Options: DENY`, Referrer-Policy, Permissions-Policy, and the CSP with `script-src 'none'` and `style-src 'self'` | deploy |

The Astro build therefore must write directories with `index.html` (`build.format: "directory"`, `trailingSlash: "always"`) and a top-level `404.html`.

## Generated assets

The Go generator names every asset `<readable name>.<first 10 hex digits of its SHA-256>.<ext>` under `/assets/`. The Astro build keeps that scheme and the same readable names, so the parity check pairs assets by readable name.

| Asset | Today | Astro build | Checked by |
|---|---|---|---|
| `site.<hash>.css` | `web/static/css/site.css` (a template) plus palette tokens, color roles, the quest-map layout and the responsive pixel-art rules for the images the pages used | Same CSS, same order, built from the same source | parity (identical after normalizing whitespace) |
| Art PNGs copied as they are (`hero_scene`, `emblem`, `favicon_*`, `og_card`, props, ...) | Copied byte for byte from `web/assets/art/` | Copied byte for byte | parity (same bytes, so the same hash) |
| Cropped frames (`section_icons-<frame>`, `progress_landmarks-<frame>`, `terrain_base-grass-a`) | Cropped from sheets by their sidecar JSON, re-encoded as PNG | Same crops, encoded by the TypeScript PNG writer | parity (identical decoded pixels; the bytes and hash may differ) |
| Mirrored pairs (`parallax_*-pair`) | The image followed by its mirror image | Same | parity (identical decoded pixels) |
| `lorehallow-pixel.<hash>.woff` | TrueType compiled from `web/static/fonts/lorehallow-pixel.txt`, wrapped in WOFF | The compiler ported to TypeScript (1.2 fonts ADR) | parity (identical TrueType; WOFF tables decompress to the same bytes) |
| `go-regular`, `go-italic`, `go-bold`, `go-mono` `.woff` | The Go fonts' TrueType files wrapped in WOFF | Same | parity (WOFF tables decompress to the same bytes) |
| `og-<slug>.<hash>.png` (one per distinct title) | 1200×630 share card: `og_card` at 5x with a plaque and the title in the pixel font | The card renderer ported to TypeScript | parity (identical decoded pixels) |

Deflate output differs between Go's and Node's zlib, so PNG and WOFF bytes, and the hashes of those files, may differ. Pixels and font tables must not.

## Meta tags on every page

From `web/templates/layout.html`. Every page keeps all of these with the same values, apart from asset URLs whose hashes change as above.

| Tag | Value | Checked by |
|---|---|---|
| `<html lang="en">` | | parity |
| `<meta charset="utf-8">`, `<meta name="viewport" ...>` | | parity |
| `<title>` | `<Title> · Lorehallow`, or `Lorehallow: a small pixel-art MMO with a shared world` on the home page | parity, output |
| `<meta name="description">` | Per page | parity, output |
| `<link rel="canonical">` | `https://lorehallow.com` + path | parity, output |
| `color-scheme` and `theme-color` | `dark`, and the `ink` palette color | parity |
| Font preloads | The pixel font and Go Regular, `as="font" type="font/woff" crossorigin` | parity |
| Stylesheet | One `<link rel="stylesheet">` to the fingerprinted CSS; never inlined | parity, output |
| Icons | 32, 16 and 192 px PNG icons and the 180 px apple-touch icon | parity |
| Feed links | RSS and Atom `rel="alternate"` | parity |
| Open Graph | `og:site_name`, `og:type` (`website`, or `article` on posts), `og:title`, `og:description`, `og:url`, `og:image` (absolute), `og:image:width` 1200, `og:image:height` 630, `og:image:alt` | parity, output |
| `article:published_time` | Posts only | parity |
| `twitter:card` | `summary_large_image` | parity |
| `generator` | `lorehallow-web <short commit>` | parity |
| `<body class="page-<section>">` | | parity |
| Header, nav and footer | Skip link, emblem and wordmark, the six nav links (Play in place of Downloads since 1.6) with `aria-current`, footer links to GitHub, Codeberg, BUILD.md, RSS, FAQ, both licenses and How it's made, and the build commit and date | parity, output |

## Output tests and budgets

Every check in `web/internal/site/site_test.go` and the parser and font tests is ported to TypeScript and runs on the Astro build (BUILD.md 1.8).

| Go test | What it guards | Astro equivalent |
|---|---|---|
| `TestNoScriptsAnywhere` | No `<script>`, `<iframe>`, inline `on*=` handlers, `style` attributes or `javascript:` URLs on any page | output |
| `TestImagesHaveAltAndWholeNumberSizes` | Every `<img>` has `alt` and `width`/`height` that are whole-number multiples of the image's real size | output |
| `TestFitRulesAreWholeNumberScales` | Every responsive pixel-art rule in the CSS scales by a whole number | output |
| `TestInternalLinksResolve` | Every internal `href`, `src`, fragment and CSS `url()` points at a file (and an `id`) that exists | output |
| `TestNoThirdPartyRequests` | No page or stylesheet loads anything from another origin | output |
| `TestLandingPageUnderOneMegabyte` | The landing page with every image, font and stylesheet it can load stays under 1 MB | output |
| `TestContrastMeetsWCAGAA` | Every text and background role pair meets WCAG AA (4.5:1, or 3:1 for large text) | unit |
| `TestStylesheetUsesColorRoles` | The stylesheet sets `color` and `background-color` only through role variables | output |
| `TestEveryPageHasMetadata` | Title, description, canonical URL, Open Graph tags with a 1200×630 card, GitHub and Codeberg links on every page | output |
| `TestFeedsSitemapAndProgressData` | RSS, Atom and the sitemap are valid XML; `/progress.json` has the right shape and totals | output |
| `TestDraftsOnlyInServe` | Drafts never reach the build | output (extended: no draft in any page, tag page, feed, sitemap, share card or any file) |
| `TestPlayWithARelease` (was `TestDownloadsWithARelease`) | Release data renders and validates | unit (`parseReleases`), and the play page in parity |
| `TestParseAchievements` | The git-history achievement feed pairs checks, rewords and moves correctly | unit, same fixtures |
| `TestFrontMatter` | Front matter errors name the file and field | unit |
| `TestPaletteParsing` | `palette.hex` parsing and its errors | unit |
| `TestImgWithoutAltFails` | An image without alt text fails the build | unit |
| `buildmd` tests and `FuzzParse` | The format contract, problems with line numbers, CRLF, the real BUILD.md | unit, plus the parser parity test against the Go parser (1.4) |
| `pixelfont` tests | The font covers every heading character, TrueType is reproducible, contours, parse errors, measuring | unit |

## Content that must match

| What | Rule | Checked by |
|---|---|---|
| Page text | The visible text of every page is the same, apart from the accepted differences below | parity |
| Inline Markdown in BUILD.md items | Rendered the same way: code spans, links (relative links point at GitHub), emphasis, smart punctuation | parity |
| Devlog posts | Same slugs, titles, dates, summaries, tags, body text, reading time and newer/older links; raw HTML dropped | parity |
| Feed entries | Same entry ids (`https://lorehallow.com/devlog/<slug>/`), links, dates, titles, summaries and categories, so readers see no duplicates | parity |
| `/progress.json` | Same document for the same commit, apart from `generated` | parity |
| `/progress.svg` | Same SVG | parity (byte for byte) |
| Sitemap | Same URL set and `lastmod` values | parity |

## Accepted differences

Recorded here as the parity check finds them. Anything not listed fails the check.

- Hashes in the names of re-encoded PNGs and of WOFF files, and therefore every URL and CSS `url()` that points at them, as explained under [Generated assets](#generated-assets).
- Whitespace between tags. The parity check compares pages after collapsing whitespace.
- `/progress.json`'s `generated` timestamp (it's the build time).
- The About page's "This website" line, which names the generator that built the page (the Astro build says Astro from the cutover on).
