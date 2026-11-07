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

Agents write posts in the owner's voice as drafts (`draft: true`). The owner edits them and publishes by setting `draft: false`.
