"""Scripted fixes for Batch 02, applied by `make art-intake BATCH=02`.

parallax_far: the painting has five one-pixel stars in its top rows. Three
of them sit in the middle of the layer, right behind the landing page's
headline, where they read as stray punctuation between words. Move those
three out to the sides of the sky (same color, still isolated), so the
layer keeps its five stars and the middle stays clear.
"""

from collections import Counter

from PIL import Image

# (from, to) in art pixels. Every target is open sky, away from the other
# stars and from the mirrored copies the website tiles beside the layer.
FAR_STAR_MOVES = [((93, 7), (28, 5)), ((84, 15), (212, 8)), ((161, 17), (229, 21))]


def _sky_around(img: Image.Image, x: int, y: int) -> tuple[int, ...]:
    around = [
        img.getpixel((x + dx, y + dy))
        for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1))
        if 0 <= x + dx < img.width and 0 <= y + dy < img.height
    ]
    return Counter(around).most_common(1)[0][0]


def fix(asset_id: str, image: Image.Image) -> Image.Image:
    if asset_id != "parallax_far":
        return image
    out = image.copy()
    for (fx, fy), (tx, ty) in FAR_STAR_MOVES:
        star = out.getpixel((fx, fy))
        sky = _sky_around(out, fx, fy)
        if star == sky:
            raise ValueError(f"parallax_far: no star at {(fx, fy)}")
        if out.getpixel((tx, ty)) != _sky_around(out, tx, ty):
            raise ValueError(f"parallax_far: {(tx, ty)} isn't open sky")
        out.putpixel((fx, fy), sky)
        out.putpixel((tx, ty), star)
    return out
