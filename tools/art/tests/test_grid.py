import numpy as np
import pytest
from conftest import TEST_COLORS
from synth import random_art, render

from lorehallow_art.grid import detect_grid, edge_profile

CASES = [
    # (logical w, h, scale, render options)
    pytest.param(32, 32, 32.0, {}, id="tile-integer-scale"),
    pytest.param(32, 48, 21.33, {}, id="character-fractional-scale"),
    pytest.param(192, 192, 5.333, {}, id="sheet-fractional-scale"),
    pytest.param(64, 64, 16.0, {"jitter": 2.0, "blur": 1.2, "noise": 4}, id="jittered-soft"),
    pytest.param(32, 32, 8.0, {"offset": (5, 5), "pad": (9, 2)}, id="off-grid-margins"),
    pytest.param(192, 192, 5.333, {"offset": (7, 3)}, id="sheet-off-grid"),
    pytest.param(192, 192, 5.333, {"jitter": 0.8, "blur": 0.8, "noise": 5}, id="sheet-noisy"),
    pytest.param(128, 96, 8.0, {"jitter": 1.0, "blur": 0.8, "shift": 6}, id="tileset-off-palette"),
]


@pytest.mark.parametrize(("w", "h", "scale", "opts"), CASES)
def test_detects_period_and_offset(rng, w, h, scale, opts):
    art = random_art(rng, w, h, TEST_COLORS)
    img = render(art, scale, rng=rng, **opts)
    H, W = img.shape[:2]
    grid = detect_grid(img, hint=(W / w, H / h))
    assert grid.x.period == pytest.approx(scale, rel=0.003)
    assert grid.y.period == pytest.approx(scale, rel=0.003)
    ox, oy = opts.get("offset", (0, 0))
    # The detected first line is the art origin, possibly whole cells before it.
    for found, true in ((grid.x, ox), (grid.y, oy)):
        k = round((true - found.offset) / found.period)
        assert found.offset + k * found.period == pytest.approx(true, abs=0.6)


def test_edge_profile_marks_block_boundaries():
    img = np.zeros((4, 8, 3), dtype=np.uint8)
    img[:, 4:] = 200
    prof = edge_profile(img, axis=1)
    assert prof.argmax() == 4
    assert prof[0] == 0


def test_flat_image_uses_hint():
    img = np.full((64, 64, 3), 128, dtype=np.uint8)
    grid = detect_grid(img, hint=(8.0, 8.0))
    assert (grid.x.period, grid.y.period) == (8.0, 8.0)
    assert grid.shape(64, 64) == (8, 8)
