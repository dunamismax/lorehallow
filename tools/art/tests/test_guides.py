"""Transition layout guides: the 3x3 patch closes, bites sit in their corners,
and every border meets the tile edges at the same points."""

import numpy as np
import pytest

from lorehallow_art.guides import BAND, SCALE, inner_mask, render_guide
from lorehallow_art.manifest import ManifestError, parse_manifest

NAMES = [
    "edge_tl", "edge_t", "edge_tr", "inner_tl",
    "edge_l", "center", "edge_r", "inner_tr",
    "edge_bl", "edge_b", "edge_br", "inner_bl",
    "edge_t_b", "center_b", "edge_b_b", "inner_br",
]  # fmt: skip


def transition(**extra):
    raw = {
        "id": "grass_dirt",
        "target": "tiles/grass_dirt.png",
        "kind": "tileset",
        "size": [128, 128],
        "grid": {"cell": [32, 32], "columns": 4, "rows": 4},
        "frames": {"count": 16, "names": NAMES},
        "guide": {"outer": "#7a8e3c", "inner": "#965a3a"},
        **extra,
    }
    return parse_manifest({"batch": "99", "assets": [raw]}).assets[0]


def test_patch_closes_and_borders_line_up():
    n = 32
    patch = np.block([[inner_mask(NAMES[r * 4 + c], n) for c in range(3)] for r in range(3)])
    # One closed patch: grass all around the 3x3 block, dirt inside.
    assert not patch[0].any() and not patch[-1].any()
    assert not patch[:, 0].any() and not patch[:, -1].any()
    assert patch[BAND:-BAND, 48].all() and patch[48, BAND:-BAND].all()
    # Edge tiles meet their neighbours at the same rows and columns.
    assert np.array_equal(inner_mask("edge_t", n)[:, -1], inner_mask("edge_tr", n)[:, 0])
    assert np.array_equal(inner_mask("edge_t", n)[:, 0], inner_mask("edge_t_b", n)[:, 0])
    assert inner_mask("center", n).all()


@pytest.mark.parametrize(("name", "corner"), [("inner_tl", (0, 0)), ("inner_tr", (0, 31)),
                                              ("inner_bl", (31, 0)), ("inner_br", (31, 31))])  # fmt: skip
def test_inner_corner_bites(name, corner):
    m = inner_mask(name, 32)
    assert not m[corner]
    assert m.sum() > 32 * 32 - BAND * BAND - 1
    others = [(0, 0), (0, 31), (31, 0), (31, 31)]
    others.remove(corner)
    assert all(m[c] for c in others)


def test_render_guide_size_and_colors():
    img = np.asarray(render_guide(transition()))
    assert img.shape == (128 * SCALE, 128 * SCALE, 3)
    colors = {tuple(c) for c in img.reshape(-1, 3)}
    assert colors == {(0x7A, 0x8E, 0x3C), (0x96, 0x5A, 0x3A)}


def test_guide_validation():
    with pytest.raises(ManifestError, match="guide"):
        transition(guide={"outer": "#7a8e3c"})
    with pytest.raises(ManifestError, match="guide"):
        transition(guide={"outer": "nope", "inner": "#965a3a"})
