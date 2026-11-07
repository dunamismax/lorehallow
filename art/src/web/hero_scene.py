"""Landing-page key art, 240×160: the generated painting of the village at
dusk (hero_scene_painted) with its four old generated adventurers painted
out and the hand-drawn ones standing in their places: the knight and the
ranger on the path, the mage beside them, and the healer with a lantern by
the well.
"""

# (box to paint out, side to mirror from, cast member, centre x, feet row).
# Painted in this order, so each box mirrors scenery that is already clean.
# The old knight's hair sits just left of the old ranger's legs, so it goes
# first, mirrored from the clear grass on its left; then the ranger and
# the rest of the knight can mirror from both sides.
FIGURES = [
    ((164, 67, 184, 100), "both", 4, 176, 101),  # healer by the well
    ((115, 87, 139, 120), "both", 2, 127, 119),  # mage
    ((64, 103, 80, 114), "left", None, 0, 0),  # the knight's hair
    ((82, 86, 107, 119), "both", 1, 94, 118),  # ranger
    ((57, 102, 82, 141), "both", 0, 70, 139),  # knight
]


def draw(ctx):
    scene = ctx.module("scene.py")
    c = ctx.art("hero_scene_painted")
    cast = scene.cast(ctx)
    for box, side, _, _, _ in FIGURES:
        scene.paint_out(c, *box, side=side)
    # stamp back to front, by the row each one stands on
    for _, _, who, x, feet in sorted(FIGURES, key=lambda f: f[4]):
        if who is not None:
            scene.stand(c, cast[who], x, feet)
    return c
