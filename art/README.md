# art/

Everything about Lorehallow's pixel art. The full process is in [AGENTS.md section 8](../AGENTS.md#8-art-pipeline).

| Path | What goes there |
|---|---|
| `style/` | The locked palette (`palette.hex`), the style guide (`STYLE.md`) and the reference sheet. |
| `src/` | Hand-drawn sources, the default ([ADR 0004](../docs/decisions/0004-hand-drawn-pixel-art.md)): `.pxl.yaml` pixel grids and `.py` draw scripts, one folder per category. Rendered by `make art-draw`. |
| `batches/` | One folder per art batch: `manifest.yaml`, and from `make art-draw` `report.md`, `preview.png` and `preview-8x.png`; for generated assets also `prompts.md`, `intake-report.md` and optional fix scripts. |
| `inbox/` | Raw generated images (the exception: large painted web art), dropped in unchanged. **Never edit anything here.** |
| `processed/` | Output of `make art-draw` and `make art-intake`: palette-locked, sliced assets. Edited only by those commands. |

After drawing or intake, `make sync` copies `processed/` into the client and the website.
