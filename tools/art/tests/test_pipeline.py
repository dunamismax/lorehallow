"""End to end: synthetic generator-style images must come back as the exact
logical art, with the key color transparent."""

import numpy as np
import pytest
from conftest import TEST_COLORS
from PIL import Image
from synth import KEY, random_art, render

from lorehallow_art.grid import fixed_grid
from lorehallow_art.intake import process_image
from lorehallow_art.manifest import parse_manifest
from lorehallow_art.sampling import FitError, downsample, fit, quantize, remove_key


def expected_rgba(art: np.ndarray) -> np.ndarray:
    key = np.all(art == KEY, axis=-1)
    out = np.zeros((*art.shape[:2], 4), dtype=np.uint8)
    out[..., :3] = np.where(key[..., None], 0, art)
    out[..., 3] = np.where(key, 0, 255)
    return out


def make_asset(kind, size, **extra):
    raw = {"id": "thing", "target": "tiles/thing.png", "kind": kind, "size": list(size), **extra}
    return parse_manifest({"batch": "99", "assets": [raw]}).assets[0]


@pytest.mark.parametrize(
    ("w", "h", "scale", "opts"),
    [
        pytest.param(32, 32, 32.0, {}, id="clean"),
        pytest.param(32, 48, 21.33, {"jitter": 2.0, "blur": 1.5, "noise": 6}, id="jittered"),
        pytest.param(64, 64, 16.0, {"offset": (11, 6), "pad": (4, 13)}, id="off-grid"),
        pytest.param(32, 32, 30.0, {"shift": 9, "noise": 5}, id="off-palette"),
        pytest.param(96, 64, 10.667, {"jitter": 1.0, "blur": 1.0, "noise": 4}, id="tileset-scale"),
        pytest.param(192, 192, 5.333, {"jitter": 0.5, "blur": 0.4, "noise": 3}, id="sheet-scale"),
        # A 3:1 wordmark centered on a 3:2 canvas with tall key margins.
        pytest.param(48, 16, 32.0, {"offset": (0, 128), "pad": (0, 128)}, id="wide-on-canvas"),
    ],
)
def test_recovers_exact_art(rng, palette, w, h, scale, opts):
    art = random_art(rng, w, h, TEST_COLORS)
    # Keep content off the very edge so margins can be told apart from art.
    art[0, :] = KEY
    art[-1, :] = KEY
    art[:, 0] = KEY
    art[:, -1] = KEY
    img = render(art, scale, rng=rng, **opts)
    res = process_image(Image.fromarray(img), make_asset("sprite", (w, h)), palette)
    assert res.problems == []
    assert res.status == "accepted"
    np.testing.assert_array_equal(res.image, expected_rgba(art))
    if opts.get("shift"):
        assert res.changed > 0


def test_integer_downsample_is_exact(rng, palette):
    art = random_art(rng, 16, 16, TEST_COLORS)
    img = np.repeat(np.repeat(art, 4, axis=0), 4, axis=1)
    rgb, alpha = downsample(img, fixed_grid(4, 4))
    np.testing.assert_array_equal(rgb, art)
    assert alpha.all()


def test_remove_key_catches_fringe(palette):
    # Pure key, a magenta-tinted blend (fringe) and a real palette red.
    rgb = np.array([[[255, 0, 255], [220, 40, 200], [232, 59, 59]]], dtype=np.uint8)
    res = remove_key(rgb, np.ones((1, 3), bool), (255, 0, 255), palette, tolerance=0.12)
    assert res.opaque.tolist() == [[False, False, True]]
    assert res.keyed == 2


def test_quantize_reports_changes(palette):
    rgb = np.array([[[24, 20, 37], [30, 25, 40]]], dtype=np.uint8)
    q = quantize(rgb, np.ones((1, 2), bool), palette)
    assert q.changed == 1
    assert q.rgba[0, 1, :3].tolist() == [24, 20, 37]
    assert q.rgba[..., 3].tolist() == [[255, 255]]


def test_fit_trims_transparent_margins_and_pads_one():
    a = np.zeros((10, 12, 4), dtype=np.uint8)
    a[3:7, 2:9, 3] = 255
    out = fit(a, 8, 8)
    assert out.rgba.shape == (8, 8, 4)
    assert out.trimmed == (4, 2)
    assert (out.rgba[..., 3] > 0).sum() == 28
    b = np.zeros((7, 8, 4), dtype=np.uint8)
    assert fit(b, 8, 8).padded == (0, 1)


def test_fit_refuses_content_too_big_or_grid_too_coarse():
    a = np.full((10, 10, 4), 255, dtype=np.uint8)
    with pytest.raises(FitError, match="larger than the logical"):
        fit(a, 8, 8)
    with pytest.raises(FitError, match="cells along"):
        fit(np.zeros((5, 5, 4), dtype=np.uint8), 8, 8)


def sheet_art(baselines: list[int], fw=8, fh=12) -> np.ndarray:
    """A 2x2 sheet of little figures whose feet sit on the given rows."""
    art = np.zeros((fh * 2, fw * 2, 3), dtype=np.uint8)
    art[:] = KEY
    for k, base in enumerate(baselines):
        r, c = divmod(k, 2)
        x0, y0 = c * fw, r * fh
        art[y0 + base - 6 : y0 + base + 1, x0 + 2 : x0 + 6] = TEST_COLORS[3]
        art[y0 + base - 8 : y0 + base - 6, x0 + 3 : x0 + 5] = TEST_COLORS[5]
    return art


def sheet_asset(baseline=None):
    frames = {"count": 4, "rows": ["down", "up"], "columns": ["idle_0", "idle_1"]}
    if baseline is not None:
        frames["baseline"] = baseline
    return make_asset(
        "sheet", (16, 24), grid={"cell": [8, 12], "columns": 2, "rows": 2}, frames=frames
    )


def test_aligned_sheet_is_accepted(rng, palette):
    img = render(sheet_art([10, 10, 10, 10]), 40.0, rng=rng, blur=1.0)
    res = process_image(Image.fromarray(img), sheet_asset(baseline=10), palette)
    assert res.status == "accepted", res.problems


def test_misaligned_sheet_fails_clearly(rng, palette):
    img = render(sheet_art([10, 10, 9, 10]), 40.0, rng=rng)
    res = process_image(Image.fromarray(img), sheet_asset(), palette)
    assert res.status == "regenerate"
    assert any("up_idle_0 at row 9" in p for p in res.problems), res.problems


def test_missing_frame_fails(rng, palette):
    art = sheet_art([10, 10, 10, 10])
    art[12:, 8:] = KEY  # blank the last frame
    res = process_image(Image.fromarray(render(art, 40.0, rng=rng)), sheet_asset(), palette)
    assert res.status == "regenerate"
    assert any("empty" in p and "up_idle_1" in p for p in res.problems)


def test_fit_anchors_a_sprite_on_its_pivot():
    rgba = np.zeros((60, 40, 4), dtype=np.uint8)
    rgba[10:30, 5:14] = (*TEST_COLORS[1], 255)  # 9 wide, rows 10..29
    out = fit(rgba, 32, 48, anchor=(16, 45)).rgba
    rows = np.flatnonzero((out[..., 3] > 0).any(axis=1))
    cols = np.flatnonzero((out[..., 3] > 0).any(axis=0))
    assert rows[-1] == 45
    assert (cols[0], cols[-1]) == (12, 20)  # 9 wide: the middle column is 16


def test_small_sprite_on_a_big_canvas(rng, palette):
    """Generators draw a small sprite in the middle of a large key canvas.
    The drawn content bounds the block size from below, and the sprite
    lands on its pivot whatever its margins."""
    figure = random_art(rng, 18, 30, TEST_COLORS)
    figure[0, :] = TEST_COLORS[0]  # make the full height and width count
    figure[-1, :] = TEST_COLORS[0]
    canvas = np.zeros((77, 51, 3), dtype=np.uint8)
    canvas[:] = KEY
    canvas[40:70, 17:35] = figure
    img = render(canvas, 20.08, rng=rng, blur=0.5, noise=2)
    asset = make_asset("sprite", (32, 48), pivot=[16, 45])
    res = process_image(Image.fromarray(img), asset, palette)
    assert res.status == "accepted", res.problems
    assert abs(res.grid.x.period - 20.08) < 0.2 and abs(res.grid.y.period - 20.08) < 0.2
    body = res.image[45 - 29 : 46, 16 - 9 : 16 + 9]  # 18 wide: symmetric about x = 16
    assert np.array_equal(body, expected_rgba(figure))


def test_ambiguous_axis_is_made_square(rng, palette):
    """When every block is drawn twice as wide in x, the x edges alone
    suggest double the period. Art pixels are square, so the y period wins."""
    art = random_art(rng, 24, 24, TEST_COLORS)
    art = np.repeat(art[:, ::2], 2, axis=1)  # columns come in identical pairs
    asset = make_asset("tile", (24, 24))
    res = process_image(Image.fromarray(render(art, 13.0, rng=rng)), asset, palette)
    assert abs(res.grid.x.period - res.grid.y.period) < 0.5
    assert res.status == "accepted", res.problems
    assert np.array_equal(res.image, expected_rgba(art))


def test_fixed_grid_mode_takes_the_canvas_as_the_layout(rng, palette):
    art = random_art(rng, 30, 20, TEST_COLORS)
    art[art.sum(axis=-1) == sum(KEY)] = TEST_COLORS[2]  # full-bleed, no key
    img = render(art, 6.4, rng=rng, jitter=0.6)
    asset = make_asset("ui", (30, 20))
    res = process_image(Image.fromarray(img), asset, palette, grid_mode="fixed")
    assert res.status == "accepted", res.problems
    assert np.array_equal(res.image, expected_rgba(art))
