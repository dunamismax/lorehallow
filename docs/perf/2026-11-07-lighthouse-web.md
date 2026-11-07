# Lighthouse, lorehallow.com on mobile: 99 to 100 in every category

- **Date:** 2026-11-07
- **Commit:** a6943bedc32611b4f08bdaf68a2c4b2d39e0c336 of the live site (`lorehallow-web a6943be` in each page's generator meta tag)
- **Site:** https://lorehallow.com, served by Caddy on `prod` (deploy/HOST.md) over the public internet from the owner's home connection
- **Tool:** Lighthouse 13.6.0, run once with `npx lighthouse@13.6.0 <url> --only-categories=performance,accessibility,best-practices,seo` against the local Chrome (headless). This is the same Lighthouse that Chrome DevTools runs, with its default mobile settings: emulated mobile device and simulated throttling (slow 4G, 4x CPU slowdown). Lighthouse isn't in the project toolchain (ADR 0011).
- **Client:** `dev-m5` (MacBook Pro Mac17,2, Apple M5, macOS 27.2)

## Results

Budget (BUILD.md 1.8): 95 or higher in every category, on mobile.

| Page | Performance | Accessibility | Best practices | SEO | FCP | LCP | TBT | CLS | Transfer | Pass |
|---|---|---|---|---|---|---|---|---|---|---|
| `/` | 99 | 100 | 100 | 100 | 0.9 s | 2.3 s | 0 ms | 0 | 333 KiB | yes |
| `/progress/` | 100 | 100 | 100 | 100 | 0.9 s | 0.9 s | 0 ms | 0 | 250 KiB | yes |
| `/devlog/` | 100 | 100 | 100 | 100 | 0.8 s | 0.8 s | 0 ms | 0 | 242 KiB | yes |

A devlog post page isn't measured yet: no post is published (the two posts are drafts, which the live site leaves out). The devlog index uses the same layout, fonts and stylesheet. Measure a post when the first one goes live.

## Notes

- Zero blocking time on every page, because there's no JavaScript to run.
- The landing page's LCP is the hero (the wordmark and the parallax backgrounds, which load from the stylesheet). Lighthouse's remaining hints are informational:
  - "Render-blocking requests": the one stylesheet.
  - "Network dependency tree": fonts and backgrounds load from the stylesheet.
  - "Improve image delivery": it suggests WebP. Lossless WebP isn't in Go's standard library, and the PNGs are already a few KB each.
- To repeat from Chrome DevTools: open the page, DevTools, Lighthouse, Mode: Navigation, Device: Mobile, those four categories, Analyze.
