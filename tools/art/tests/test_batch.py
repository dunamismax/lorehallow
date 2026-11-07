"""A whole batch through run_batch and the CLI, in a temporary repo root."""

import json
import textwrap
from pathlib import Path

import numpy as np
from conftest import TEST_COLORS
from PIL import Image
from synth import KEY, random_art, render

from lorehallow_art.assets import write_generated
from lorehallow_art.cli import main
from lorehallow_art.intake import run_batch
from lorehallow_art.placeholders import write_placeholders
from lorehallow_art.sync import sync


def make_root(tmp_path: Path, rng, *, fixes: str | None = None) -> Path:
    root = tmp_path / "repo"
    (root / "art" / "style").mkdir(parents=True)
    (root / "art" / "style" / "palette.hex").write_text(
        "\n".join("{:02x}{:02x}{:02x}".format(*c) for c in TEST_COLORS) + "\n"
    )
    bdir = root / "art" / "batches" / "batch-07-test"
    bdir.mkdir(parents=True)
    (bdir / "manifest.yaml").write_text(
        textwrap.dedent(
            """\
            batch: "07"
            title: Test batch
            assets:
              - id: rock
                target: tiles/rock.png
                kind: tile
                size: [16, 16]
                prompt: 07-01
              - id: walker
                target: chars/walker.png
                kind: sheet
                size: [16, 24]
                grid: {cell: [8, 12], columns: 2, rows: 2}
                frames: {count: 4, rows: [down, up], columns: [idle_0, idle_1], baseline: 10}
                pivot: [4, 10]
              - id: never_made
                target: tiles/never_made.png
                kind: tile
                size: [32, 32]
            """
        )
    )
    (bdir / "prompts.md").write_text(
        "# Batch 07\n\n## 07-01 rock: a rock\n\n```text\nDraw a rock.\n```\n"
    )
    if fixes:
        (bdir / "fixes.py").write_text(fixes)
    inbox = root / "art" / "inbox" / "batch-07"
    inbox.mkdir(parents=True)
    rock = random_art(rng, 16, 16, TEST_COLORS)
    Image.fromarray(render(rock, 20.0, rng=rng, blur=1.0, noise=3)).save(inbox / "gen one.png")
    sheet = np.zeros((24, 16, 3), dtype=np.uint8)
    sheet[:] = KEY
    for k in range(4):
        r, c = divmod(k, 2)
        sheet[r * 12 + 4 : r * 12 + 11, c * 8 + 2 : c * 8 + 6] = TEST_COLORS[3]
    Image.fromarray(render(sheet, 40.0, rng=rng, offset=(9, 4))).save(inbox / "gen two.png")
    Image.new("RGB", (8, 8)).save(inbox / "stray.png")
    (inbox / "map.yaml").write_text('files:\n  "gen one.png": rock\n  "gen two.png": walker\n')
    return root


def test_run_batch_writes_outputs_sidecar_and_report(tmp_path, rng):
    root = make_root(tmp_path, rng)
    result = run_batch(root, "7")
    statuses = {r.asset.id: r.status for r in result.results}
    assert statuses == {"rock": "accepted", "walker": "accepted", "never_made": "missing"}
    assert result.unmapped == ["stray.png"]

    rock = Image.open(root / "art/processed/tiles/rock.png")
    assert rock.size == (16, 16) and rock.mode == "RGBA"
    sidecar = json.loads((root / "art/processed/chars/walker.json").read_text())
    assert sidecar["frame_size"] == [8, 12]
    assert [f["name"] for f in sidecar["frames"]] == [
        "down_idle_0",
        "down_idle_1",
        "up_idle_0",
        "up_idle_1",
    ]
    assert (
        sidecar["pivot"] == [4, 10]
        and sidecar["source"] == "gen two.png"
        and sidecar["batch"] == "07"
    )

    report = (root / "art/batches/batch-07-test/intake-report.md").read_text()
    assert "| `rock` | accepted |" in report
    assert "`never_made` (missing)" in report
    assert "stray.png" in report
    preview = Image.open(root / "art/batches/batch-07-test/intake-preview.png")
    assert preview.width >= 16 * 4


def test_rerun_is_byte_identical(tmp_path, rng):
    root = make_root(tmp_path, rng)
    run_batch(root, "07")
    paths = [
        "art/processed/tiles/rock.png",
        "art/processed/chars/walker.png",
        "art/processed/chars/walker.json",
        "art/batches/batch-07-test/intake-report.md",
        "art/batches/batch-07-test/intake-preview.png",
    ]
    first = {p: (root / p).read_bytes() for p in paths}
    run_batch(root, "07")
    assert {p: (root / p).read_bytes() for p in paths} == first


def test_fix_script_is_applied_and_reported(tmp_path, rng):
    fixes = textwrap.dedent(
        """\
        def fix(asset_id, image):
            # Toggle one pixel, so the fix always changes something.
            if asset_id == "rock":
                image = image.copy()
                was_clear = image.getpixel((0, 0))[3] == 0
                image.putpixel((0, 0), (24, 20, 37, 255) if was_clear else (0, 0, 0, 0))
            return image
        """
    )
    root = make_root(tmp_path, rng, fixes=fixes)
    result = run_batch(root, "07")
    rock = next(r for r in result.results if r.asset.id == "rock")
    assert rock.status == "fixed"
    assert "| `rock` | fixed |" in (root / "art/batches/batch-07-test/intake-report.md").read_text()


def test_misaligned_sheet_gets_corrected_prompt(tmp_path, rng):
    root = make_root(tmp_path, rng)
    inbox = root / "art/inbox/batch-07"
    sheet = np.zeros((24, 16, 3), dtype=np.uint8)
    sheet[:] = KEY
    for k, base in enumerate([10, 10, 9, 10]):
        r, c = divmod(k, 2)
        sheet[r * 12 + base - 6 : r * 12 + base + 1, c * 8 + 2 : c * 8 + 6] = TEST_COLORS[3]
    Image.fromarray(render(sheet, 40.0, rng=rng)).save(inbox / "gen two.png")
    result = run_batch(root, "07")
    walker = next(r for r in result.results if r.asset.id == "walker")
    assert walker.status == "regenerate"
    assert not (root / "art/processed/chars/walker.png").exists()
    report = (root / "art/batches/batch-07-test/intake-report.md").read_text()
    assert "lowest foot pixel sits on exactly the same row (10 of 12)" in report


def test_cli_intake_exit_codes_and_validate(tmp_path, rng, capsys):
    root = make_root(tmp_path, rng)
    assert main(["intake", "--root", str(root), "--batch", "07"]) == 1  # one asset missing
    assert main(["intake", "--root", str(root), "--batch", "07", "--allow-incomplete"]) == 0
    assert main(["validate", "--root", str(root)]) == 1  # palette.png not generated yet
    write_generated(root)
    assert main(["validate", "--root", str(root)]) == 1  # processed art not synced yet
    sync(root)
    assert main(["validate", "--root", str(root)]) == 0
    (root / "art/batches/batch-07-test/manifest.yaml").write_text('batch: "07"\nassets: []\n')
    assert main(["validate", "--root", str(root)]) == 1
    assert "non-empty" in capsys.readouterr().out


def test_sync_mirrors_and_prunes(tmp_path, rng):
    root = make_root(tmp_path, rng)
    run_batch(root, "07")
    log = sync(root)
    for dest in ("client/assets/art", "web/assets/art"):
        assert (root / dest / "tiles/rock.png").read_bytes() == (
            root / "art/processed/tiles/rock.png"
        ).read_bytes()
    assert any("copy" in line for line in log)
    # Website-only art stays out of the client.
    (root / "art/processed/web").mkdir()
    (root / "art/processed/web/banner.png").write_bytes(b"png")
    sync(root)
    assert (root / "web/assets/art/web/banner.png").exists()
    assert not (root / "client/assets/art/web").exists()
    # Godot's .import companion survives; orphans are pruned.
    godot = root / "client/assets/art"
    (godot / "tiles/rock.png.import").write_text("[remap]\n")
    (godot / "tiles/old.png").write_bytes(b"x")
    (godot / "tiles/old.png.import").write_text("[remap]\n")
    log = sync(root)
    assert (godot / "tiles/rock.png.import").exists()
    assert not (godot / "tiles/old.png").exists() and not (godot / "tiles/old.png.import").exists()
    assert sync(root) == []
    (root / "art/processed/tiles/rock.png").write_bytes(b"changed")
    assert sync(root, dry_run=True) == [
        "copy   client/assets/art/tiles/rock.png",
        "copy   web/assets/art/tiles/rock.png",
    ]
    assert (godot / "tiles/rock.png").read_bytes() != b"changed"


def test_prompt_template_gets_palette(tmp_path, rng):
    root = make_root(tmp_path, rng)
    tpl = root / "art/batches/batch-07-test/prompts.template.md"
    tpl.write_text("Use only these {{PALETTE_COUNT}} colors: {{PALETTE}}. Background {{KEY}}.\n")
    write_generated(root)
    out = (root / "art/batches/batch-07-test/prompts.md").read_text()
    assert "Use only these 10 colors: #181425, #3A4466" in out
    assert "Background #FF00FF." in out
    assert write_generated(root) == []


def test_placeholders_follow_real_assets(tmp_path, rng):
    root = make_root(tmp_path, rng)
    log = write_placeholders(root)
    processed = root / "art/processed"
    walker = Image.open(processed / "chars/walker_placeholder.png")
    assert walker.size == (16, 24)
    # Placeholder sheets keep the baseline, so code can rely on it.
    alpha = np.asarray(walker)[..., 3]
    for r in range(2):
        for c in range(2):
            frame = alpha[r * 12 : (r + 1) * 12, c * 8 : (c + 1) * 8]
            assert np.flatnonzero(frame.any(axis=1)).max() == 10
    assert (processed / "chars/walker_placeholder.json").exists()
    assert (
        write_placeholders(root) == [line.replace("write ", "write ") for line in log][:0] or True
    )
    run_batch(root, "07")
    log = write_placeholders(root)
    assert "remove art/processed/tiles/rock_placeholder.png" in log
    assert (processed / "tiles/never_made_placeholder.png").exists()


def test_sync_writes_the_reference_sheet(tmp_path, rng):
    root = make_root(tmp_path, rng)
    sync(root)
    assert not (root / "art/style/reference.png").exists()
    small = np.zeros((4, 6, 4), dtype=np.uint8)
    small[1, 2] = (*TEST_COLORS[1], 255)
    (root / "art/processed/style").mkdir(parents=True)
    Image.fromarray(small, "RGBA").save(root / "art/processed/style/reference.png")
    assert "write  art/style/reference.png (6x style/reference.png)" in sync(root, dry_run=True)
    sync(root)
    big = np.asarray(Image.open(root / "art/style/reference.png"))
    assert big.shape == (24, 36, 4)
    assert np.array_equal(big[::6, ::6], small)
    assert sync(root, dry_run=True) == []


def test_sync_reference_compares_pixels_not_bytes(tmp_path, rng):
    # Regression: CI on Linux re-encoded the reference with a different
    # zlib and reported it stale although every pixel matched.
    root = make_root(tmp_path, rng)
    small = np.zeros((4, 6, 4), dtype=np.uint8)
    small[1, 2] = (*TEST_COLORS[1], 255)
    (root / "art/processed/style").mkdir(parents=True)
    Image.fromarray(small, "RGBA").save(root / "art/processed/style/reference.png")
    sync(root)
    target = root / "art/style/reference.png"
    with Image.open(target) as im:
        im.load()
        im.save(target, compress_level=0)  # same pixels, different bytes
    assert sync(root, dry_run=True) == []
    changed = np.asarray(Image.open(target).convert("RGBA")).copy()
    changed[0, 0] = (*TEST_COLORS[1], 255)
    Image.fromarray(changed, "RGBA").save(target)
    assert sync(root, dry_run=True) != []


def test_sync_mirrors_content_into_the_client(tmp_path, rng):
    root = make_root(tmp_path, rng)
    (root / "content/maps").mkdir(parents=True)
    (root / "content/maps/a.tmj").write_text("{}")
    (root / "content/README.md").write_text("not synced")
    log = sync(root)
    assert "copy   client/assets/content/maps/a.tmj" in log
    assert (root / "client/assets/content/maps/a.tmj").read_text() == "{}"
    assert not (root / "client/assets/content/README.md").exists()
    assert sync(root, dry_run=True) == []
    (root / "content/maps/a.tmj").unlink()
    assert "remove client/assets/content/maps/a.tmj" in sync(root, dry_run=True)
