"""Command-line entry point.

lorehallow-art draw --root . [--batch 01] [--review DIR]
lorehallow-art intake --root . --batch 01
lorehallow-art sync --root .
lorehallow-art validate --root .
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

from .assets import palette_problems, stale_generated, write_generated
from .favicons import export as export_favicons
from .intake import INTAKE_REPORT, run_batch
from .manifest import ManifestError, load_manifest, load_map
from .palette import Palette
from .placeholders import write_placeholders
from .pxl import SourceError
from .render import run_draw, stale_drawn, write_review
from .sync import sync


def _intake(args: argparse.Namespace) -> int:
    result = run_batch(Path(args.root), args.batch, write=not args.dry_run)
    for r in result.results:
        extra = f"  ({'; '.join(r.problems)})" if r.problems else ""
        print(f"{r.status:10s} {r.asset.id}{extra}")
    if result.unmapped:
        print("unmapped inbox files: " + ", ".join(result.unmapped))
    print(f"report: {result.batch_dir / INTAKE_REPORT}")
    return 0 if result.ok or args.allow_incomplete else 1


def _draw(args: argparse.Namespace) -> int:
    try:
        results = run_draw(Path(args.root), args.batch)
    except (ManifestError, SourceError) as e:
        print(e)
        return 1
    bad = 0
    for r in results:
        if not r.asset.source:
            continue
        extra = f"  ({'; '.join(r.problems)})" if r.problems else ""
        print(f"{r.status:10s} {r.asset.id}{extra}")
        bad += r.status != "drawn"
    if args.review:
        for p in write_review(results, Path(args.review)):
            print(f"review     {p}")
    print("draw: ok" if not bad else f"draw: {bad} asset(s) with problems")
    return 1 if bad else 0


def _sync(args: argparse.Namespace) -> int:
    for line in sync(Path(args.root)):
        print(line)
    print("sync: done")
    return 0


def _validate(args: argparse.Namespace) -> int:
    """Check every batch manifest, inbox map and the palette parse cleanly,
    and that hand-drawn art renders, passes its checks and is up to date."""
    root = Path(args.root)
    errors = 0
    palette = root / "art" / "style" / "palette.hex"
    if palette.exists():
        try:
            for problem in palette_problems(Palette.load(palette)):
                print(f"{palette}: {problem}")
                errors += 1
            for path in stale_generated(root):
                print(f"{path} is stale; run make art-generate")
                errors += 1
        except ValueError as e:
            print(f"{palette}: {e}")
            errors += 1
    try:
        for problem in stale_drawn(root):
            print(problem)
            errors += 1
    except (ManifestError, SourceError) as e:
        print(e)
        errors += 1
    stale = sync(root, dry_run=True)
    if stale:
        print(f"synced art is stale ({len(stale)} changes); run make sync")
        for line in stale:
            print(f"  {line}")
        errors += 1
    for path in sorted((root / "art" / "batches").glob("batch-*/manifest.yaml")):
        try:
            m = load_manifest(path)
            expected = f"batch-{m.batch}-"
            if not path.parent.name.startswith(expected):
                raise ManifestError(f"{path}: folder name must start with {expected}")
            load_map(root / "art" / "inbox" / f"batch-{m.batch}" / "map.yaml", m)
            for a in m.assets:
                if not (root / a.palette).exists():
                    raise ManifestError(f"{path}: asset {a.id} uses missing palette {a.palette}")
        except (ManifestError, ValueError) as e:
            print(e)
            errors += 1
    print("validate: ok" if not errors else f"validate: {errors} problem(s)")
    return 1 if errors else 0


def _generate(args: argparse.Namespace) -> int:
    for path in write_generated(Path(args.root)):
        print(f"wrote {path}")
    print("generate: done")
    return 0


def _placeholders(args: argparse.Namespace) -> int:
    for line in write_placeholders(Path(args.root)):
        print(line)
    print("placeholders: done")
    return 0


def _favicons(args: argparse.Namespace) -> int:
    for line in export_favicons(Path(args.root)):
        print(line)
    print("favicons: done")
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="lorehallow-art", description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)

    p = sub.add_parser("draw", help="render hand-drawn sources from art/src into art/processed")
    p.add_argument("--root", default=".", help="repository root")
    p.add_argument("--batch", help="batch number, e.g. 01 (default: every batch)")
    p.add_argument("--review", help="also write review images (tiles 3×3, 1x views) here")
    p.set_defaults(func=_draw)

    p = sub.add_parser("intake", help="process one batch from art/inbox into art/processed")
    p.add_argument("--root", default=".", help="repository root")
    p.add_argument("--batch", required=True, help="batch number, e.g. 01")
    p.add_argument("--dry-run", action="store_true", help="process and report, write nothing")
    p.add_argument(
        "--allow-incomplete", action="store_true", help="exit 0 even if assets need regeneration"
    )
    p.set_defaults(func=_intake)

    p = sub.add_parser("sync", help="mirror art/processed into the client and website")
    p.add_argument("--root", default=".")
    p.set_defaults(func=_sync)

    p = sub.add_parser("validate", help="check manifests, maps and the palette")
    p.add_argument("--root", default=".")
    p.set_defaults(func=_validate)

    p = sub.add_parser("generate", help="rebuild palette.png, palette.gpl and prompts.md files")
    p.add_argument("--root", default=".")
    p.set_defaults(func=_generate)

    p = sub.add_parser("placeholders", help="write placeholder art for assets not processed yet")
    p.add_argument("--root", default=".")
    p.set_defaults(func=_placeholders)

    p = sub.add_parser("favicons", help="export the favicon set and app icon from the emblem")
    p.add_argument("--root", default=".")
    p.set_defaults(func=_favicons)

    args = parser.parse_args(argv)
    return int(args.func(args))


if __name__ == "__main__":
    sys.exit(main())
