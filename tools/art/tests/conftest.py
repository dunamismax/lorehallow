import sys
from pathlib import Path

import numpy as np
import pytest

sys.path.insert(0, str(Path(__file__).parent))

from lorehallow_art.palette import Palette

# A small, well-spread test palette (not the real one, which may change).
TEST_COLORS = [
    (24, 20, 37),
    (58, 68, 102),
    (38, 92, 66),
    (99, 199, 77),
    (184, 111, 80),
    (228, 166, 114),
    (254, 231, 97),
    (232, 59, 59),
    (139, 155, 180),
    (255, 255, 255),
]


@pytest.fixture
def palette() -> Palette:
    return Palette(tuple(TEST_COLORS))


@pytest.fixture
def rng() -> np.random.Generator:
    return np.random.default_rng(1234)
