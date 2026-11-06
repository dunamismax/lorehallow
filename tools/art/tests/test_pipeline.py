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
