# Lighthouse, lorehallow.com on mobile after the Astro cutover: 99 to 100 in every category

- **Date:** 2026-11-08
- **Commit:** 9950cc4 of the live site, the first Astro build deployed (`lorehallow-web 9950cc4` in each page's generator meta tag)
- **Site:** https://lorehallow.com, served by Caddy on `prod` (deploy/HOST.md) over the public internet from the owner's home connection
- **Tool:** Lighthouse 13.6.0, run once per page with `npx lighthouse@13.6.0 <url> --only-categories=performance,accessibility,best-practices,seo --chrome-flags="--headless=new"` against the local Chrome. These are Lighthouse's default mobile settings: an emulated mobile device and simulated throttling (slow 4G, 4x CPU slowdown). Lighthouse isn't in the project toolchain; it runs under Node on the machine doing the audit (BUILD.md 1.8).
- **Client:** `dev-m5` (MacBook Pro Mac17,2, Apple M5, macOS 27.2)

## Results

Budget (BUILD.md 1.8): 95 or higher in every category, on mobile.

| Page | Performance | Accessibility | Best practices | SEO | FCP | LCP | TBT | CLS | Transfer | Pass |
|---|---|---|---|---|---|---|---|---|---|---|
| `/` | 99 | 100 | 100 | 100 | 0.9 s | 2.3 s | 0 ms | 0 | 332 KiB | yes |
| `/progress/` | 100 | 100 | 100 | 100 | 0.9 s | 0.9 s | 0 ms | 0 | 252 KiB | yes |
| `/devlog/the-website-is-live/` | 100 | 100 | 100 | 100 | 0.8 s | 0.8 s | 0 ms | 0 | 244 KiB | yes |

## Notes

- The numbers match the Go-built site's ([2026-11-07](2026-11-07-lighthouse-web.md)) within a few KiB, as they should: the parity check holds the two builds to the same pages, stylesheet and pixels.
- Zero blocking time everywhere, because no page has any JavaScript.
- This is the first measurement of a devlog post page; the first posts went live on 2026-11-08.
- The landing page's LCP is still the hero (wordmark and parallax backgrounds from the stylesheet).
