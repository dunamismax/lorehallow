# web/data/

Data files the website builds pages from.

## releases.json

Every version of the browser game that has gone live at play.lorehallow.com, newest first. It drives [lorehallow.com/play](https://lorehallow.com/play/) and, from Phase 15, the patch notes. There are no platform files, sizes or checksums: the game is played in the browser and a new version is a new static build ([ADR 0014](../../docs/decisions/0014-astro-website-and-phaser-browser-client.md)).

It's empty until v0.2.0 goes live (BUILD.md 2.10), and /play says plainly that there's nothing to play yet. Add the entry in the same commit that deploys the version.

```json
{
  "releases": [
    {
      "version": "v0.2.0",
      "date": "2026-12-01",
      "channel": "alpha",
      "notes": "/devlog/v0-2-0/",
      "protocol": 3
    }
  ]
}
```

| Field | Rule |
|---|---|
| `version` | `vX.Y.Z`, as tagged in git |
| `date` | `YYYY-MM-DD`, the day it went live; entries are listed newest first |
| `channel` | `alpha`, `beta` or `stable` |
| `notes` | A site path (`/devlog/...`) or an `https://` URL, or `""` |
| `protocol` | The `PROTOCOL_VERSION` the build speaks (`proto/PROTOCOL_VERSION`) |

No other fields are allowed. The build validates it (`src/lib/releases.ts`) and fails with the release and field named.
