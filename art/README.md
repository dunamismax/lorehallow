# art/

Everything about Lorehallow's pixel art. The full process is in [AGENTS.md section 8](../AGENTS.md#8-art-pipeline).

| Path | What goes there |
|---|---|
| `style/` | The locked palette (`palette.hex`), the style guide (`STYLE.md`) and the reference sheet. |
| `batches/` | One folder per art batch: `prompts.md`, `manifest.yaml`, `report.md` and optional fix scripts. |
| `inbox/` | Raw generated images, dropped in unchanged by the owner. **Never edit anything here.** |
| `processed/` | Output of `make art-intake`: grid-snapped, palette-locked, sliced assets. Edited only by scripts. |

After intake, `make sync` copies `processed/` into the client and the website.
