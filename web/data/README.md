# web/data/

Data files the website builds pages from.

## releases.json

Drives lorehallow.com/downloads. The release workflow (BUILD.md 15) appends to it; until then it's empty and the page says so. The build fails on any entry that doesn't match this schema.

```json
{
  "releases": [
    {
      "version": "v0.2.0",
      "date": "2026-12-01",
      "channel": "alpha",
      "notes": "/devlog/v0-2-0/",
      "files": [
        {
          "platform": "linux",
          "url": "https://github.com/dunamismax/lorehallow/releases/download/v0.2.0/lorehallow-v0.2.0-linux-amd64.tar.gz",
          "size": 48234112,
          "sha256": "64 lowercase hex characters"
        }
      ]
    }
  ]
}
```

| Field | Rules |
|---|---|
| `version` | `vX.Y.Z` |
| `date` | `YYYY-MM-DD` |
| `channel` | `alpha`, `beta` or `stable` |
| `notes` | Link to the release notes (a devlog post or `docs/releases/`), or `""` |
| `files` | At least one |
| `files[].platform` | `windows`, `macos` or `linux` |
| `files[].url` | `https://` only |
| `files[].size` | Bytes, greater than zero |
| `files[].sha256` | 64 lowercase hex characters |

Newest release first. For each platform, the downloads page shows the first release that has a file for it. The system requirements and "verify your download" sections appear once any release exists.
