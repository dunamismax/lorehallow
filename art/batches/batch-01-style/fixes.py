"""Match the outermost pixels of base terrain variants so they can tile."""

from collections import Counter

from PIL import Image

TILE_SIZE = 32
LAST = TILE_SIZE - 1
MATERIAL_GROUPS = (
    (0, 1, 2, 3, 4, 15),  # Grass, including the flower, tuft and stone variants.
    (5, 6, 7),  # Dirt.
    (8, 9, 10, 11),  # Stone.
    (12, 13),  # Cobblestone.
    (14,),  # Flagstone.
)


def fix(asset_id: str, image: Image.Image) -> Image.Image:
    if asset_id != "terrain_base":
        return image

    corrected = image.copy()
    corners = ((0, 0), (0, LAST), (LAST, 0), (LAST, LAST))
    for group in MATERIAL_GROUPS:
        origins = [(index % 4 * TILE_SIZE, index // 4 * TILE_SIZE) for index in group]
        tiles = [image.crop((x, y, x + TILE_SIZE, y + TILE_SIZE)) for x, y in origins]

        # Use colors already common along each boundary. Keep tile interiors intact.
        corner = Counter(tile.getpixel(point) for tile in tiles for point in corners)
        corner_color = corner.most_common(1)[0][0]
        for index in range(1, LAST):
            horizontal = ((index, 0), (index, LAST))
            vertical = ((0, index), (LAST, index))
            for boundary in (horizontal, vertical):
                votes = Counter(tile.getpixel(point) for tile in tiles for point in boundary)
                color = votes.most_common(1)[0][0]
                for tile in tiles:
                    for point in boundary:
                        tile.putpixel(point, color)

        for tile, origin in zip(tiles, origins, strict=True):
            for point in corners:
                tile.putpixel(point, corner_color)
            corrected.paste(tile, origin)

    return corrected
