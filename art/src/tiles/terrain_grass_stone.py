"""Grass to flagstone transition set: old flagstones (inner) in grass (outer).

Built from the base terrain's grass_a and flagstone, so it is seamless
with those tiles. Layout and lighting in transition.py.
"""


def draw(ctx):
    tr = ctx.module("transition.py")
    grass = ctx.frame("terrain_base", "grass_a")
    flag = ctx.frame("terrain_base", "flagstone")
    colors = {"shade": "moss shadow", "lit": "leaf", "lip": "slate", "lip2": "stone"}
    return tr.tileset(ctx, grass, flag, flag, colors)
