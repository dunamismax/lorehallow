"""A mossy boulder, 32×32: form-lit stone with a moss cap draping over
its lit top, a few grass blades at its foot, ink outline."""

STONE_CUTS = [0.2, 0.45, 0.75]
MOSS_CUTS = [0.35, 0.6, 0.85]


def draw(ctx):
    sh = ctx.module("shading.py")
    c = ctx.canvas()
    body = [(16, 19, 12, 9), (11, 15, 7, 7), (20, 13, 8, 7)]
    for cx, cy, rx, ry in body:
        sh.clump(c, cx, cy, rx, ry, sh.STONE, STONE_CUTS)
    # moss cap: the same forms, only above a wavy line, drawn in moss
    edge = {x: 12 + (2 if x in range(9, 13) else 0) + (3 if x in range(14, 17) else 0)
            + (1 if x in range(22, 26) else 0) for x in range(32)}  # fmt: skip
    for cx, cy, rx, ry in body:
        sh.clump(c, cx, cy, rx, ry, sh.MOSS, MOSS_CUTS, mask=lambda x, y: y <= edge[x])
    # drips of moss below the cap, and a crack in the stone
    c.points([(10, 15), (10, 16), (15, 16), (15, 17), (23, 14), (23, 15)], "moss")
    c.line(18, 20, 20, 24, "slate")
    c.px(17, 20, "pale stone")
    sh.prune(c)
    c.outline("ink")
    # grass blades at the foot, in front of the outline
    for x, y in ((6, 27), (7, 26), (25, 27), (26, 26), (27, 27)):
        c.px(x, y, "leaf")
    c.px(7, 27, "moss")
    c.px(26, 27, "moss")
    return c
