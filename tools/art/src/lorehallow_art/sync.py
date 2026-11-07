"""`make sync`: mirror art/processed/ into the client and the website,
mirror content/ into the client, and write the prompt reference sheet
art/style/reference.png."""

from __future__ import annotations

import filecmp
import io
import shutil
from pathlib import Path

from PIL import Image

# Destination -> top-level categories it does not receive. The client has
# no use for website-only art.
DESTINATIONS: dict[str, tuple[str, ...]] = {
    "client/assets/art": ("web",),
    "web/assets/art": (),
}
# Files tools create next to synced art that must survive a sync (as long
# as the image they belong to is still there).
COMPANION_SUFFIXES = (".import",)
KEEP = {".gitkeep", "README.md"}


# The accepted style reference, upscaled for attaching to image prompts:
# at 1x a 240×160 image is too small for a generator to read the pixel grid.
REFERENCE_SOURCE = "style/reference.png"
REFERENCE_TARGET = "art/style/reference.png"
REFERENCE_SCALE = 6


def reference_bytes(src: Path) -> bytes:
    with Image.open(src) as im:
        im = im.convert("RGBA")
        big = im.resize((im.width * REFERENCE_SCALE, im.height * REFERENCE_SCALE), Image.NEAREST)
    buf = io.BytesIO()
    big.save(buf, format="PNG", optimize=True)
    return buf.getvalue()


def _same_pixels(a: bytes, b: bytes) -> bool:
    with Image.open(io.BytesIO(a)) as ia, Image.open(io.BytesIO(b)) as ib:
        ra, rb = ia.convert("RGBA"), ib.convert("RGBA")
        return ra.size == rb.size and ra.tobytes() == rb.tobytes()


def sync_reference(root: Path, dry_run: bool = False) -> list[str]:
    src = root / "art" / "processed" / REFERENCE_SOURCE
    target = root / REFERENCE_TARGET
    if not src.exists():
        return []
    data = reference_bytes(src)
    # Compare pixels, not bytes: PNG encoders (zlib builds) differ between
    # macOS and Linux, so the same image can encode to different bytes.
    if target.exists() and _same_pixels(target.read_bytes(), data):
        return []
    if not dry_run:
        target.write_bytes(data)
    return [f"write  {REFERENCE_TARGET} ({REFERENCE_SCALE}x {REFERENCE_SOURCE})"]


# Game content (maps, tilesets, later items and quests) mirrored into the
# client, which imports maps from it (ADR 0006).
CONTENT_SOURCE = "content"
CONTENT_DESTINATION = "client/assets/content"


def sync(root: Path, dry_run: bool = False) -> list[str]:
    """Mirror processed art into each destination, mirror content into the
    client and write the reference sheet. Returns a change log.

    With dry_run, nothing is written; a non-empty log means the synced
    copies are stale.
    """
    log: list[str] = sync_reference(root, dry_run)
    log += _mirror(root, root / "art" / "processed", DESTINATIONS, dry_run)
    content = root / CONTENT_SOURCE
    if content.exists():
        log += _mirror(root, content, {CONTENT_DESTINATION: ()}, dry_run)
    return log


def _mirror(
    root: Path, src: Path, destinations: dict[str, tuple[str, ...]], dry_run: bool
) -> list[str]:
    files = sorted(p.relative_to(src) for p in src.rglob("*") if p.is_file() and p.name not in KEEP)
    log: list[str] = []
    for dest_rel, excluded in destinations.items():
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
