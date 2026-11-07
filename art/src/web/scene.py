"""Helpers for putting hand-drawn characters into a generated painting.

The large painted web scenes are generated (ADR 0004), but their figures
must match the game's hand-drawn characters. A scene script starts from
the painting, paints each old figure out with `paint_out`, and stamps a
hand-drawn character in its place with `stand`.
"""


def paint_out(c, x0, y0, x1, y1, side="both"):
    """Paint over the box [x0, x1) x [y0, y1) by mirroring the scenery
    beside it into it, row by row. Grass, paths and cobbles are textures
    that run across a figure, so mirrored rows carry them through, and the
    new figure stamped on top hides the seam.

    side "both" fills the left half from the left and the right half from
    the right; "left" or "right" fills the whole box from that side, for a
    box whose other neighbour is another old figure."""
    mid = {"both": (x0 + x1) // 2, "left": x1, "right": x0}[side]
    for y in range(y0, y1):
        for x in range(x0, mid):
            c.px(x, y, c.get(x0 - 1 - (x - x0), y))
        for x in range(mid, x1):
            c.px(x, y, c.get(x1 + (x1 - 1 - x), y))


def stand(c, sprite, x, feet):
    """Stamp a 32×48 character frame so its centre column is x and its
    feet are on row `feet` (the frame's feet are on row 45, centred on
    column 16)."""
    c.stamp(sprite, x - 16, feet - 45)


def cast(ctx):
    """The five hand-drawn adventurers (knight, ranger, mage, bard, healer)
    as 32×48 frames, built exactly as character_lineup builds them."""
    lineup = ctx.module("character_lineup.py")
    hero = ctx.art("concept_hero")
    gear = ctx.pxl("../chars/gear.pxl.yaml")
    lantern = ctx.pxl("../props/lantern.pxl.yaml")[0]
    return [lineup.adventurer(ctx, hero, gear, lantern, spec) for spec in lineup.ADVENTURERS]
