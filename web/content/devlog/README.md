# web/content/devlog/

Devlog posts for lorehallow.com/devlog. One Markdown file per post. The file name is the URL slug (`laying-the-foundations.md` becomes `/devlog/laying-the-foundations/`), so use lowercase words joined by hyphens.

Every post starts with this front matter. All six fields are required, and the build fails with the line number if one is missing or malformed:

```yaml
---
title: "Laying the foundations"
date: 2026-11-06
summary: "One or two sentences for the post list, feeds and link previews."
tags: [phase-0, tooling]
cover: null
draft: true
---
```

- `date` is `YYYY-MM-DD`. Posts are listed newest first.
- `tags` are lowercase words joined by hyphens. Each tag gets a page at `/devlog/tags/<tag>/`.
- `cover` is `null` or an art path under `web/assets/art/`, for example `web/devlog_header.png`.
- `draft: true` posts appear only in `make web-dev`, with a Draft badge. `make web-build` and the deploy leave them out.

Front matter is a small, strict subset of YAML: `key: value` lines, where a value is a string (quote it if it contains a colon followed by a space, `#`, brackets or other YAML punctuation), `true`, `false`, `null`, or a `[flow, list]`.

The body is Markdown: GitHub Flavored Markdown with smart quotes and heading anchors. Raw HTML is dropped, so a post can't carry a script.

Agents write and publish posts in the owner's voice (AGENTS.md section 12): first person, plain, warm and specific, 200 to 500 words, one per real milestone, every claim backed by commits, measurements or screenshots. To publish, set `draft: false`, run `make check`, commit, and run `make deploy-web`.

The planned Astro site (BUILD.md 1.5) reads this folder in place as a content collection, with the same front matter, slugs and rules: drafts in the local preview only, never in any public page, feed or sitemap, and raw HTML still dropped. Posts don't move and don't change.
