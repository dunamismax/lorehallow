"""Grass to dirt transition set: a dirt patch (inner) in grass (outer).

Built from the base terrain's grass_a, dirt_a and dirt_b, so it is
seamless with those tiles. Layout and lighting in transition.py.
"""


def draw(ctx):
    tr = ctx.module("transition.py")
    grass = ctx.frame("terrain_base", "grass_a")
    colors = {"shade": "moss shadow", "lit": "leaf", "lip": "bark", "lip2": "earth"}
    return tr.tileset(
        ctx,
        grass,
        ctx.frame("terrain_base", "dirt_a"),
        ctx.frame("terrain_base", "dirt_b"),
        colors,
    )
