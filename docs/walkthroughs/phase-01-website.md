# Phase 1 walkthrough: lorehallow.com

Phase 1 puts the project in public: a website at https://lorehallow.com that explains what Lorehallow is, tracks the build plan live, hosts a devlog and is ready to list downloads. It is a static site built by a small Go program, served by a Caddy shared with other sites on the owner's home server, and deployed by swapping a symlink.

## What we built

- **A Go static site generator** in `web/` ([ADR 0011](../decisions/0011-go-website.md)). `lorehallow-web build` writes every page, feed, image and data file to `web/dist/`; `lorehallow-web serve` previews the same thing from memory, drafts included, and rebuilds on the next request after a source file changes.
- **The pages**: a landing page with a parallax hero, the world, about, FAQ, downloads (honestly empty), a devlog with tags and feeds, a 404, and the progress tracker.
- **A live progress tracker** built from BUILD.md: an overall XP bar, a quest map with one hand-drawn landmark per phase along a winding road, a page per phase, an achievement feed from git history, a "waiting on the owner" panel, `/progress.json` and a `/progress.svg` badge.
- **A hand-drawn heading font**, compiled into a real web font by the generator, with the Go fonts for body text ([ADR 0012](../decisions/0012-website-fonts.md)).
- **No JavaScript at all.** The fireflies, the flickering lantern and the parallax are CSS. Caddy's Content-Security-Policy refuses scripts outright, and a test fails the build if one appears.
- **Output tests** that check the built site: no scripts, alt text and whole-number image sizes, working internal links, no third-party requests, contrast, metadata, and a 1 MB budget for the landing page.
- **Deploys** with timestamped release directories, an atomic switch, a health check through the host's own Caddy, automatic rollback, and `make rollback-web`, run by hand over SSH.
- **Uptime monitoring** from GitHub's runners, with one ntfy alert per outage ([ADR 0013](../decisions/0013-uptime-monitoring.md)).

## How to try it

```bash
make web-dev                 # http://127.0.0.1:4321, drafts included
make web-build               # web/dist/, exactly what deploys
make web-test                # the output tests
make deploy-web              # build, upload, switch, check (needs deploy/host.env)
make rollback-web            # back to the previous release
make rehearse-rollback-web   # prove the automatic rollback still works
```

Then check a box in BUILD.md, commit, push, and watch /progress change.

## How it works

### DNS, TLS and a shared Caddy

`lorehallow.com`, `www` and `play` are A records at Cloudflare pointing straight at the owner's home IP, not proxied, so Caddy sees real client addresses. The home router forwards 80 and 443 to the VM. Caddy was already serving other sites there, so Lorehallow is one site file, `/etc/caddy/sites/lorehallow.caddy`, imported by the main Caddyfile ([ADR 0009](../decisions/0009-caddy-on-a-shared-host.md)).

Caddy's automatic HTTPS does the TLS work. On the first request for a name it has no certificate for, it proves to Let's Encrypt that it controls the name (an HTTP-01 or TLS-ALPN-01 challenge on ports 80 and 443), gets a certificate and renews it before it expires. Plain HTTP gets a permanent redirect to HTTPS, and `www` redirects to the apex.

Because the Caddy is shared, a broken config would take down every site on the host. `make install-host` (`deploy/bootstrap/app.sh`) copies the new site file into place, validates the *whole* configuration with the same binary and user the service runs as, puts the old file back if validation fails, and only then reloads.

The headers are part of the design. `script-src 'none'` makes the no-JavaScript rule something the browser enforces. Files under `/assets/` have a content hash in their name, so they're cached for a year as `immutable`. Everything else is revalidated on every request, cheaply, thanks to ETags.

### A static site from Go

`site.Build` (`web/internal/site/site.go`) reads its inputs and returns the whole site as a `map[string][]byte`:

- BUILD.md, through the same parser `make buildmd-lint` uses
- `art/style/palette.hex`, turned into CSS custom properties (`--c-lantern-gold` and so on)
- the processed art in `web/assets/art/`
- the devlog Markdown and `web/data/releases.json`
- `git log` of BUILD.md, for the achievement feed

Keeping the site in memory makes `build` and `serve` the same code, and lets the tests inspect the output without touching the disk.

A few details are worth a look:

- **Fingerprinting.** `Assets.Add` names a file after the first ten hex digits of its SHA-256. The stylesheet is rendered last, because it includes rules generated while the pages render; pages carry a placeholder URL that is swapped for the real one at the end.
- **Pixel art at whole-number scales.** CSS can't read an image's natural size, so the generator does. `{{img "web/hero_scene.png" "alt" "max=720"}}` writes an `<img>` with explicit dimensions and a generated class that steps the image through 1x, 2x and 3x as the viewport grows. The image is never stretched to a fractional size.
- **Colors with a contract.** The stylesheet sets text and background colors only through named roles. `theme.go` lists every text-on-background pair the CSS uses, and a test checks each one meets WCAG AA.
- **Share cards.** Each page gets a 1200×630 Open Graph image: the 240×126 `og_card` art at exactly 5x, with the page title drawn on a plaque in the pixel font.

### How BUILD.md becomes the progress page

BUILD.md's format contract (phases, metadata, sections, single-line checkboxes) is what makes this possible. The parser turns it into phases and items, and `buildProgress` adds what the pages need:

- each phase's state: done, current (the first with an unchecked box), or future
- its landmark and walkthrough link
- the unchecked `HUMAN` items in the current phase

The quest map is an ordered list placed on a CSS grid. On phones it's one column; on wider screens the rows snake back and forth (4, then 7 across), with the road segments generated per breakpoint.

The achievement feed is the interesting part. `git log -p -U0 -- BUILD.md` lists every change to the file. Within each commit, a line that went from `- [ ]` to `- [x]` is a box being checked. A checked line that was removed and re-added with different text is a reword, not news, and so is a line moved verbatim. The result is a list of "quest complete" entries, each with the date and the commit that did it. This is why the site must be built from a full clone, not a shallow one.

### The deploy pipeline

`deploy/web/deploy.sh` treats the host like a small release manager:

1. Build the site at a commit and upload it to `/srv/lorehallow/web/releases/<timestamp>/`.
2. Point `current` at it with `ln -sfn` to a temporary name, then `mv -T` over the old link. A rename is atomic, so every request sees either the old release or the new one.
3. Check it through the host's own Caddy (`curl --resolve lorehallow.com:443:127.0.0.1`), so DNS and the home router aren't part of the test. The check confirms that the page is from the new commit, that its stylesheet loads, that the key pages and data load, and that a missing page is a real 404.
4. On failure, switch back to the previous release and delete the new one.

Rollback is the same switch in the other direction. `make rehearse-rollback-web` deploys a release with its stylesheet deleted and checks that the health check catches it and the previous release comes back on its own.

Deploys are run by hand with `make deploy-web`, from the owner's machine or an agent's session.

## The key concept: static sites and atomic releases

The site has no server process: Caddy hands out files. That removes a whole class of problems. There is nothing to crash, leak memory, patch or supervise, and Caddy can serve a static file to thousands of visitors at once. Everything dynamic, even the "live" progress tracker, is computed at build time and refreshed by rebuilding when the inputs change.

The price is that a deploy replaces many files at once, and a visitor mustn't see half of the old site and half of the new. The answer is old and simple: never change a release in place. Build each release in its own directory, then move one pointer.

```
/srv/lorehallow/web/
├── current -> releases/20261107T171404Z   ← one rename switches the whole site
└── releases/
    ├── 20261107T170914Z/   ← the previous release, ready for rollback
    ├── 20261107T171357Z/
    └── 20261107T171404Z/   ← live
```

Content hashes in asset names finish the job. A page from the new release links to `site.41073fbebc.css`; a browser still showing the old page keeps loading `site.3e8d8e5a43.css` from its cache. The two never mix, which is why assets can be cached forever and pages not at all.

The same pattern is in the game server's deploy (Phase 2), and it comes back at a larger scale in Phase 12's zero-downtime deploys.

## Read the code

1. `web/cmd/lorehallow-web/main.go`: `build` and `serve`, and how `serve` notices changes.
2. `web/internal/site/site.go`: `Build`, `render`, the CSS placeholder swap, `questMapCSS`.
3. `web/internal/site/assets.go`: `Add` (fingerprints), `Image` (frames and mirrored tiles), `Img` and `FitCSS` (whole-number scaling).
4. `web/internal/site/progress.go`: `buildProgress`, then `parseAchievements`.
5. `web/internal/site/theme.go`: the palette, the color roles and the contrast pairs.
6. `web/internal/pixelfont/ttf.go`: `contours` traces pixels into font outlines; `TTF` writes the tables.
7. `web/internal/site/site_test.go`: the output tests.
8. `web/static/css/site.css`: the hero, the fireflies and the parallax, all in CSS.
9. `deploy/web/deploy.sh`.
10. `deploy/caddy/lorehallow.caddy`: headers and caching.

## What's next

Phase 2 is already partly built: two players in one world, with a real connection lifecycle, backpressure, a tick loop and interpolation. Every checkbox it closes shows up on https://lorehallow.com/progress at the next deploy.
