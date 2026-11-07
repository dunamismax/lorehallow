"""Hand-drawn art: .pxl.yaml grids, draw-script helpers, the renderer, its
checks, `make art-draw` and the staleness check in `validate`."""

import json
import textwrap
from pathlib import Path

import numpy as np
import pytest
from PIL import Image

from lorehallow_art.cli import main
from lorehallow_art.draw import Canvas, ellipse_mask, line_points
from lorehallow_art.intake import run_batch
from lorehallow_art.manifest import ManifestError, load_manifest, load_map
from lorehallow_art.palette import NamedPalette, palette_names
from lorehallow_art.pxl import SourceError, parse_pxl
from lorehallow_art.render import (
    Library,
    _pivot_problem,
    check,
    run_draw,
    stale_drawn,
    write_review,
)

PALETTE_HEX = textwrap.dedent(
    """\
    ; test palette
    1a1420 ; ink: outline
    2f4f2f ; moss shadow
    7a8e3c ; leaf
    abb84e ; sunlit leaf
    965a3a ; earth
    f2c45a ; lantern gold
    """
)
INK, MOSS, LEAF, SUN, EARTH, GOLD = (
    (0x1A, 0x14, 0x20),
    (0x2F, 0x4F, 0x2F),
    (0x7A, 0x8E, 0x3C),
    (0xAB, 0xB8, 0x4E),
    (0x96, 0x5A, 0x3A),
    (0xF2, 0xC4, 0x5A),
)


@pytest.fixture
def named() -> NamedPalette:
    return NamedPalette.from_text(PALETTE_HEX)


# -- palette names -----------------------------------------------------------


def test_palette_names_come_from_comments(named):
    assert palette_names(PALETTE_HEX)[:2] == ["ink", "moss shadow"]
    assert named["leaf"] == LEAF
    assert named.name_of(GOLD) == "lantern gold"
    assert "leaf" in named and "magenta" not in named
    with pytest.raises(KeyError, match="unknown palette color 'magenta'"):
        named["magenta"]


def test_palette_names_must_be_unique():
    with pytest.raises(ValueError, match="duplicate"):
        NamedPalette.from_text("111111 ; a\n222222 ; a\n")


# -- pixel grids ---------------------------------------------------------------

GRID = {
    "size": [3, 2],
    "legend": {".": "transparent", "K": "ink", "g": "leaf"},
    "frames": ["K.g\ngKK\n", ".K.\nK.K  # a comment\n"],
}


def test_parse_pxl_frames(named):
    p = parse_pxl(GRID, named)
    assert p.size == (3, 2) and len(p.frames) == 2
    f0 = p.frames[0]
    assert f0.shape == (2, 3, 4)
    assert tuple(f0[0, 0]) == (*INK, 255)
    assert f0[0, 1, 3] == 0
    assert tuple(f0[0, 2]) == (*LEAF, 255)
    assert tuple(p.frames[1][1, 2]) == (*INK, 255)


@pytest.mark.parametrize(
    ("change", "message"),
    [
        ({"legend": {".": "transparent", "K": "ink", "g": "magenta"}}, "unknown palette color"),
        ({"frames": ["K.x\ngKK\n"]}, "'x' is not in the legend"),
        ({"frames": ["K.g\n"]}, "has 1 rows, expected 2"),
        ({"frames": ["K.gg\ngKK\n"]}, "row 0 has 4 columns, expected 3"),
        ({"size": [3]}, "size must be"),
        ({"legend": {"KK": "ink"}}, "one non-space character"),
        ({"frames": []}, "non-empty list"),
        ({"colour": 1}, "unknown keys"),
    ],
)
def test_parse_pxl_errors(named, change, message):
    with pytest.raises(SourceError, match=message):
        parse_pxl({**GRID, **change}, named)


# -- drawing helpers -----------------------------------------------------------


def test_canvas_pixels_by_name_and_transparency(named):
    c = Canvas(named, 4, 4)
    c.px(1, 2, "leaf")
    assert c.get(1, 2) == "leaf" and c.get(0, 0) is None
    c.px(9, 9, "leaf")  # outside, clipped
    assert c.mask().sum() == 1
    c.px(1, 2, None)
    assert c.mask().sum() == 0
    with pytest.raises(KeyError):
        c.px(0, 0, "magenta")


def test_wrapping_canvas_wraps_every_primitive(named):
    c = Canvas(named, 8, 8, wrap=True)
    c.px(-1, 8, "ink")
    assert c.get(7, 0) == "ink"
    c.line(6, 3, 9, 3, "leaf")
    assert [c.get(x, 3) for x in (6, 7, 0, 1)] == ["leaf"] * 4
    stamp = Canvas(named, 2, 2)
    stamp.fill("earth")
    c.stamp(stamp, 7, 7)
    assert {c.get(7, 7), c.get(0, 7), c.get(7, 0), c.get(0, 0)} == {"earth"}


def test_line_rect_and_ellipse(named):
    assert line_points(0, 0, 3, 1) == [(0, 0), (1, 0), (2, 1), (3, 1)]
    c = Canvas(named, 6, 6)
    c.rect(1, 1, 4, 3, "leaf", fill=False)
    assert c.mask().sum() == 10 and c.get(2, 2) is None
    m = ellipse_mask(7, 5)
    assert (m == m[:, ::-1]).all() and (m == m[::-1]).all()
    assert m[2].all() and not m[0, 0]
    e = Canvas(named, 7, 7)
    e.ellipse(0, 0, 7, 7, "ink", fill=False)
    assert e.get(3, 3) is None and e.get(3, 0) == "ink"


def test_outline_mirror_swap_and_crop(named):
    c = Canvas(named, 5, 5)
    c.px(2, 2, "leaf")
    c.outline("ink")
    assert c.mask().sum() == 5 and c.get(2, 1) == "ink" and c.get(1, 1) is None
    c.outline("ink", corners=True)
    assert c.mask().sum() == 21  # the 3x3 diamond grows to 5x5 minus corners
    d = Canvas(named, 3, 1)
    d.px(0, 0, "ink")
    assert d.mirror().get(2, 0) == "ink"
    assert d.flip().get(0, 0) == "ink"
    assert d.swap({"ink": "lantern gold"}).get(0, 0) == "lantern gold"
    assert d.crop(0, 0, 2, 1).get(0, 0) == "ink"
    d.replace("ink", "leaf")
    assert d.get(0, 0) == "leaf"


def test_outline_on_a_wrapping_canvas_wraps(named):
    c = Canvas(named, 4, 4, wrap=True)
    c.px(0, 0, "leaf")
    c.outline("ink")
    assert c.get(3, 0) == "ink" and c.get(0, 3) == "ink"


# -- the renderer, in a temporary repository --------------------------------------

MANIFEST = """\
batch: "05"
title: Draw test
assets:
  - id: gem
    target: items/gem.png
    kind: sprite
    size: [5, 5]
    source: items/gem.pxl.yaml
  - id: walker
    target: chars/walker.png
    kind: sheet
    size: [10, 6]
    grid: {cell: [5, 6], columns: 2, rows: 1}
    frames: {count: 2, columns: [a, b], rows: [down], baseline: 4}
    pivot: [2, 4]
    source: chars/walker.py
  - id: patch
    target: tiles/patch.png
    kind: tileset
    size: [8, 4]
    grid: {cell: [4, 4], columns: 2, rows: 1}
    frames: {count: 2, names: [plain, dotted]}
    source: tiles/patch.py
  - id: scene
    target: ui/scene.png
    kind: ui
    size: [8, 8]
    source: ui/scene.py
  - id: painted
    target: web/painted.png
    kind: web
    size: [4, 4]
"""

GEM = """\
size: [5, 5]
legend: {".": transparent, K: ink, g: lantern gold}
frames:
  - |
    ..K..
    .KgK.
    KgggK
    .KgK.
    ..K..
"""

WALKER = """\
def draw(ctx):
    body = ctx.pxl("walker_body.pxl.yaml")[0]
    frames = []
    for k in range(2):
        f = ctx.canvas(*ctx.cell)
        f.stamp(body, 1, 2 - k)  # bobs one row, feet stay
        f.rect(1, 4, 3, 1, "earth")
        frames.append(f)
    return frames
"""

WALKER_BODY = """\
size: [3, 2]
legend: {K: ink, l: leaf}
frames:
  - |
    KlK
    KlK
"""

PATCH = """\
def draw(ctx):
    rng = ctx.rng(7)
    out = ctx.canvas()
    for k in range(2):
        t = ctx.canvas(4, 4, wrap=True)
        t.fill("leaf")
        if k:
            t.px(rng.randrange(4), rng.randrange(4), "sunlit leaf")
        out.stamp(t, 4 * k, 0)
    return out
"""

SCENE = """\
def draw(ctx):
    c = ctx.canvas()
    ctx.module("helper.py").dot(c)
    c.stamp(ctx.frame("patch", "plain"), 0, 0)
    c.stamp(ctx.art("gem"), 3, 3)
    c.stamp(ctx.art("painted"), 0, 4)
    return c
"""


def make_root(tmp_path: Path) -> Path:
    root = tmp_path / "repo"
    (root / "art/style").mkdir(parents=True)
    (root / "art/style/palette.hex").write_text(PALETTE_HEX)
    bdir = root / "art/batches/batch-05-draw"
    bdir.mkdir(parents=True)
    (bdir / "manifest.yaml").write_text(MANIFEST)
    src = root / "art/src"
    for rel, text in {
        "items/gem.pxl.yaml": GEM,
        "chars/walker.py": WALKER,
        "chars/walker_body.pxl.yaml": WALKER_BODY,
        "tiles/patch.py": PATCH,
        "ui/scene.py": SCENE,
        "ui/helper.py": "def dot(c):\n    c.px(7, 0, 'leaf')\n",
    }.items():
        (src / rel).parent.mkdir(parents=True, exist_ok=True)
        (src / rel).write_text(text)
    painted = root / "art/processed/web/painted.png"
    painted.parent.mkdir(parents=True)
    arr = np.zeros((4, 4, 4), dtype=np.uint8)
    arr[...] = (*EARTH, 255)
    Image.fromarray(arr, "RGBA").save(painted)
    return root


def test_run_draw_renders_writes_and_reports(tmp_path):
    root = make_root(tmp_path)
    results = {r.asset.id: r for r in run_draw(root, "05")}
    assert {k: r.status for k, r in results.items()} == {
        "gem": "drawn",
        "walker": "drawn",
        "patch": "drawn",
        "scene": "drawn",
        "painted": "generated",
    }
    gem = np.asarray(Image.open(root / "art/processed/items/gem.png").convert("RGBA"))
    assert tuple(gem[2, 2]) == (*GOLD, 255) and gem[0, 0, 3] == 0
    side = json.loads((root / "art/processed/chars/walker.json").read_text())
    assert side["source"] == "art/src/chars/walker.py"
    assert [f["name"] for f in side["frames"]] == ["down_a", "down_b"]
    assert side["baseline"] == 4 and side["batch"] == "05"
    scene = np.asarray(Image.open(root / "art/processed/ui/scene.png").convert("RGBA"))
    assert tuple(scene[5, 5]) == (*GOLD, 255)  # the gem, stamped at (3, 3)
    assert tuple(scene[7, 0]) == (*EARTH, 255)  # generated art from art/processed
    assert tuple(scene[0, 7]) == (*LEAF, 255)  # drawn by the helper module
    bdir = root / "art/batches/batch-05-draw"
    report = (bdir / "report.md").read_text()
    assert "| `walker` | sheet | 10×6 | `art/src/chars/walker.py` | drawn |" in report
    assert "| `painted` | web | 4×4 | intake | generated |" in report
    p4, p8 = Image.open(bdir / "preview.png"), Image.open(bdir / "preview-8x.png")
    assert p4.width >= 10 * 4 and p8.width >= 10 * 8
    assert stale_drawn(root) == []


def test_rendering_is_deterministic(tmp_path):
    root = make_root(tmp_path)
    a, b = Library(root), Library(root)
    for asset_id in ("gem", "walker", "patch", "scene"):
        assert np.array_equal(a.render(asset_id), b.render(asset_id))


def test_validate_fails_when_processed_art_is_stale(tmp_path):
    root = make_root(tmp_path)
    run_draw(root)
    gem = root / "art/src/items/gem.pxl.yaml"
    gem.write_text(gem.read_text().replace("KgggK", "KgKgK"))
    problems = stale_drawn(root)
    # The scene composites the gem, so it is stale too.
    assert problems == [
        "gem: art/processed/items/gem.png doesn't match its source; run make art-draw",
        "scene: art/processed/ui/scene.png doesn't match its source; run make art-draw",
    ]
    (root / "art/processed/chars/walker.json").unlink()
    assert any("walker.json is missing" in p for p in stale_drawn(root))
    assert main(["validate", "--root", str(root)]) == 1
    run_draw(root)
    assert stale_drawn(root) == []


def test_validate_catches_bad_sources(tmp_path):
    root = make_root(tmp_path)
    run_draw(root)
    gem = root / "art/src/items/gem.pxl.yaml"
    gem.write_text(gem.read_text().replace("lantern gold", "gold leaf"))
    assert any("unknown palette color 'gold leaf'" in p for p in stale_drawn(root))
    gem.write_text(GEM.replace("size: [5, 5]", "size: [5, 4]"))
    assert any("has 5 rows, expected 4" in p for p in stale_drawn(root))
    gem.write_text(GEM)
    walker = root / "art/src/chars/walker.py"
    walker.write_text(WALKER.replace("ctx.canvas(*ctx.cell)", "ctx.canvas(6, 6)"))
    assert any("frame 0 is 6x6, the manifest frame is 5x6" in p for p in stale_drawn(root))
    walker.write_text("import random\ndef draw(ctx):\n    c = ctx.canvas()\n"
                      "    c.px(random.randrange(10), 4, 'ink')\n    return c\n")  # fmt: skip
    problems = stale_drawn(root)
    assert any("not deterministic" in p or "doesn't match" in p or "frame" in p for p in problems)
    walker.write_text("def draw(ctx):\n    return 3\n")
    assert any("must return a Canvas" in p for p in stale_drawn(root))
    walker.write_text(
        "def draw(ctx):\n    c = ctx.canvas()\n    c.px(0, 0, 'mauve')\n    return c\n"
    )
    assert any("unknown palette color 'mauve'" in p for p in stale_drawn(root))


def test_checks_baseline_pivot_frames_and_palette(tmp_path, named):
    root = make_root(tmp_path)
    lib = Library(root)
    _, walker = lib.assets["walker"]
    good = lib.render("walker")
    assert check(good, walker, named) == []
    raised = np.zeros_like(good)
    raised[:-1] = good[1:]  # every foot one row up
    assert any("foot baseline should be row 4" in p for p in check(raised, walker, named))
    wide = np.zeros((6, 12, 4), np.uint8)
    wide[2:5, 8:11] = (*LEAF, 255)
    assert _pivot_problem(wide, (5, 4), "w") == ["w is centered on column 9.5, the pivot is 5"]
    assert _pivot_problem(wide, (8, 4), "w") == []
    half = good.copy()
    half[:, 5:] = 0
    assert any("expected frames are empty" in p for p in check(half, walker, named))
    off = good.copy()
    off[4, 1] = (255, 0, 255, 255)
    assert "1 colors not in the palette" in check(off, walker, named)
    _, gem = lib.assets["gem"]
    assert check(np.zeros((5, 5, 4), np.uint8), gem, named) == ["the image is empty"]


def test_sprite_pivot_check(tmp_path, named):
    root = make_root(tmp_path)
    lib = Library(root)
    _, gem = lib.assets["gem"]
    object.__setattr__(gem, "pivot", (2, 4))
    assert check(lib.render("gem"), gem, named) == []
    object.__setattr__(gem, "pivot", (2, 3))
    assert check(lib.render("gem"), gem, named) == ["gem's lowest pixel is on row 4, expected 3"]


def test_extra_frames_and_loops_are_errors(tmp_path):
    root = make_root(tmp_path)
    gem = root / "art/src/items/gem.pxl.yaml"
    gem.write_text(GEM + "  - |\n" + "".join("    .....\n" for _ in range(5)))
    assert any("2 frames, but the grid holds 1" in p for p in stale_drawn(root))
    gem.write_text(GEM)
    (root / "art/src/ui/scene.py").write_text("def draw(ctx):\n    return ctx.art('scene')\n")
    assert any("loop" in p for p in stale_drawn(root))


def test_intake_skips_hand_drawn_assets_and_maps_cannot_name_them(tmp_path):
    root = make_root(tmp_path)
    (root / "art/inbox/batch-05").mkdir(parents=True)
    result = run_batch(root, "05")
    assert [r.asset.id for r in result.results] == ["painted"]
    m = load_manifest(root / "art/batches/batch-05-draw/manifest.yaml")
    bad = root / "art/inbox/batch-05/map.yaml"
    bad.write_text("files:\n  x.png: gem\n")
    with pytest.raises(ManifestError, match="hand-drawn"):
        load_map(bad, m)


def test_manifest_source_must_be_a_pxl_or_py_path(tmp_path):
    root = make_root(tmp_path)
    mp = root / "art/batches/batch-05-draw/manifest.yaml"
    mp.write_text(MANIFEST.replace("items/gem.pxl.yaml", "items/gem.png"))
    with pytest.raises(ManifestError, match="source must look like"):
        load_manifest(mp)


def test_cli_draw_and_review(tmp_path, capsys):
    root = make_root(tmp_path)
    review = tmp_path / "review"
    assert main(["draw", "--root", str(root), "--batch", "05", "--review", str(review)]) == 0
    out = capsys.readouterr().out
    assert "drawn      walker" in out and "draw: ok" in out
    assert (review / "patch-tiled3x3.png").exists()
    assert (review / "gem-views.png").exists()
    assert write_review([], review) == []
    (root / "art/src/items/gem.pxl.yaml").write_text(GEM.replace("K: ink", "K: inky"))
    assert main(["draw", "--root", str(root)]) == 1
    assert "unknown palette color 'inky'" in capsys.readouterr().out
