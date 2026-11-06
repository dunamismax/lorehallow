"""`make sync`: mirror art/processed/ into the client and the website."""

from __future__ import annotations

import filecmp
import shutil
from pathlib import Path

# Destination -> top-level categories it does not receive. The client has
# no use for website-only art.
DESTINATIONS: dict[str, tuple[str, ...]] = {
    "client/assets/art": ("web",),
    "web/src/assets/art": (),
}
# Files tools create next to synced art that must survive a sync (as long
# as the image they belong to is still there).
COMPANION_SUFFIXES = (".import",)
KEEP = {".gitkeep", "README.md"}


def sync(root: Path, dry_run: bool = False) -> list[str]:
    """Mirror processed art into each destination. Returns a change log.

    With dry_run, nothing is written; a non-empty log means the synced
    copies are stale.
    """
    src = root / "art" / "processed"
    files = sorted(p.relative_to(src) for p in src.rglob("*") if p.is_file() and p.name not in KEEP)
    log: list[str] = []
    for dest_rel, excluded in DESTINATIONS.items():
        dest = root / dest_rel
        if not dry_run:
            dest.mkdir(parents=True, exist_ok=True)
        mine = [f for f in files if f.parts[0] not in excluded]
        wanted = set(mine)
        for rel in mine:
            target = dest / rel
            if not target.exists() or not filecmp.cmp(src / rel, target, shallow=False):
                if not dry_run:
                    target.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copyfile(src / rel, target)
                log.append(f"copy   {dest_rel}/{rel}")
        if not dest.exists():
            continue
        for p in sorted(dest.rglob("*"), reverse=True):
            if p.is_dir():
                if not dry_run and not any(p.iterdir()):
                    p.rmdir()
                continue
            rel = p.relative_to(dest)
            if p.name in KEEP or rel in wanted:
                continue
            if p.suffix in COMPANION_SUFFIXES and rel.with_suffix("") in wanted:
                continue
            if not dry_run:
                p.unlink()
            log.append(f"remove {dest_rel}/{rel}")
    return log
