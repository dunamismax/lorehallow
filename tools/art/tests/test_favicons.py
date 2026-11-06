import numpy as np
from PIL import Image

from lorehallow_art.favicons import export


def emblem(path, size, color):
    a = np.zeros((size, size, 4), dtype=np.uint8)
    a[2:-2, 2:-2] = (*color, 255)
    path.parent.mkdir(parents=True, exist_ok=True)
    Image.fromarray(a, "RGBA").save(path)


def test_exports_integer_scaled_sizes_and_prefers_real_art(tmp_path):
    icons = tmp_path / "art/processed/icons"
    emblem(icons / "emblem_placeholder.png", 32, (10, 20, 30))
    emblem(icons / "emblem_16_placeholder.png", 16, (10, 20, 30))
    export(tmp_path)
    for size in (16, 32, 180, 192, 512):
        img = Image.open(icons / f"favicon_{size}_placeholder.png")
        assert img.size == (size, size)
    # 180 is 5x the emblem centered on an opaque background.
    a = np.asarray(Image.open(icons / "favicon_180_placeholder.png"))
    assert (a[..., 3] == 255).all()
    assert a[10 + 2 * 5, 10 + 2 * 5, :3].tolist() == [10, 20, 30]
    assert Image.open(icons / "app_icon_placeholder.png").size == (256, 256)
    # 512 is exactly 16x: every 16×16 block is one color.
    big = np.asarray(Image.open(icons / "favicon_512_placeholder.png"))
    blocks = big.reshape(32, 16, 32, 16, 4)
    assert (blocks == blocks[:, :1, :, :1]).all()

    emblem(icons / "emblem.png", 32, (200, 100, 50))
    emblem(icons / "emblem_16.png", 16, (200, 100, 50))
    log = export(tmp_path)
    assert (icons / "favicon_32.png").exists() and not (
        icons / "favicon_32_placeholder.png"
    ).exists()
    assert any("remove" in line for line in log)
