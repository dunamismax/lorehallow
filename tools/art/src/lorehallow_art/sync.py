"""`make sync`: mirror art/processed/ into the client and the website."""

from __future__ import annotations

import filecmp
import shutil
from pathlib import Path

DESTINATIONS = ("client/assets/art", "web/src/assets/art")
# Files tools create next to synced art that must survive a sync (as long
# as the image they belong to is still there).
COMPANION_SUFFIXES = (".import",)
KEEP = {".gitkeep", "README.md"}


def sync(root: Path) -> list[str]:
    """Mirror processed art into each destination. Returns a change log."""
    src = root / "art" / "processed"
    files = sorted(p.relative_to(src) for p in src.rglob("*") if p.is_file() and p.name not in KEEP)
    log: list[str] = []
    for dest_rel in DESTINATIONS:
        dest = root / dest_rel
        dest.mkdir(parents=True, exist_ok=True)
        wanted = set(files)
        for rel in files:
            target = dest / rel
            target.parent.mkdir(parents=True, exist_ok=True)
            if not target.exists() or not filecmp.cmp(src / rel, target, shallow=False):
                shutil.copyfile(src / rel, target)
                log.append(f"copy   {dest_rel}/{rel}")
        for p in sorted(dest.rglob("*"), reverse=True):
            if p.is_dir():
                if not any(p.iterdir()):
                    p.rmdir()
                continue
            rel = p.relative_to(dest)
            if p.name in KEEP or rel in wanted:
                continue
            if p.suffix in COMPANION_SUFFIXES and rel.with_suffix("") in wanted:
                continue
            p.unlink()
            log.append(f"remove {dest_rel}/{rel}")
    return log
