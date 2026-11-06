import numpy as np
import pytest

from lorehallow_art.color import hex_to_rgb, rgb_to_hex, srgb_to_oklab
from lorehallow_art.palette import Palette


def test_hex_round_trip():
    assert hex_to_rgb("#ff00FF") == (255, 0, 255)
    assert hex_to_rgb("1a2b3c") == (26, 43, 60)
    assert rgb_to_hex((26, 43, 60)) == "#1A2B3C"


@pytest.mark.parametrize("bad", ["#fff", "zzzzzz", "", "#1234567"])
def test_hex_rejects_bad(bad):
    with pytest.raises(ValueError):
        hex_to_rgb(bad)


def test_oklab_reference_points():
    lab = srgb_to_oklab(np.array([[255, 255, 255], [0, 0, 0]]))
    assert np.allclose(lab[0], [1.0, 0.0, 0.0], atol=1e-3)
    assert np.allclose(lab[1], [0.0, 0.0, 0.0], atol=1e-6)


def test_palette_parse_ignores_comments_and_blanks():
    p = Palette.from_hex_lines("; Lorehallow\n#000000\n\nffffff ; white\n// note\n")
    assert p.colors == ((0, 0, 0), (255, 255, 255))


def test_palette_rejects_duplicates_and_empty():
    with pytest.raises(ValueError):
        Palette.from_hex_lines("000000\n000000\n")
    with pytest.raises(ValueError):
        Palette.from_hex_lines("; nothing\n")


def test_nearest_is_perceptual(palette):
    # A slightly-off green maps to the green, not to the grey with a
    # similar RGB distance.
    idx = palette.nearest(np.array([[100, 190, 85], [140, 150, 185]]))
    assert [palette.colors[i] for i in idx] == [(99, 199, 77), (139, 155, 180)]
