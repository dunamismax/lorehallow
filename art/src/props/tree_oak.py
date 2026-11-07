"""A broad oak, 64×96: a thick flared trunk with roots and a canopy of
overlapping form-lit leaf clumps with a leafy edge, lit from the
top-left, ink outline."""

CANOPY = [  # (cx, cy, rx, ry), back to front
    (19, 22, 14, 12), (45, 20, 14, 12), (32, 12, 14, 10),
    (12, 37, 10, 10), (52, 37, 10, 10), (32, 30, 18, 15),
    (20, 50, 13, 10), (44, 50, 13, 10), (32, 21, 10, 8), (32, 56, 10, 6),
]  # fmt: skip
CUTS = [0.18, 0.42, 0.66, 0.86]


def trunk(c):
    for y in range(52, 93):
        half = 6 if y < 82 else 6 + (y - 81) // 2 + (2 if y >= 90 else 0)
        for x in range(32 - half, 32 + half):
            rel = (x - (32 - half)) / (2 * half)
            col = "earth" if rel < 0.3 else "bark" if rel < 0.75 else "dark soil"
            c.px(x, y, col)
    for x, y0, y1 in ((30, 64, 74), (35, 66, 80), (32, 76, 86)):
        c.line(x, y0, x, y1, "dark soil")
    c.line(28, 66, 28, 82, "dry path")
    for x0, x1 in ((27, 23), (37, 41)):
        c.line(x0, 86, x1, 92, "dark soil")
    c.line(32, 89, 32, 92, "dark soil")


def draw(ctx):
    sh = ctx.module("shading.py")
    c = ctx.canvas()
    trunk(c)
    for cx, cy, rx, ry in CANOPY:
        sh.clump(c, cx, cy, rx, ry, sh.FOLIAGE, CUTS)
    # the canopy's shadow on the trunk
    for x in range(24, 41):
        for y in range(52, 70):
            if c.get(x, y) in ("earth", "bark", "dry path") and c.get(x, y - 1) in sh.FOLIAGE:
                c.px(x, y, "dark soil")
                c.px(x, y + 1, "dark soil" if x > 29 else "bark")
    rng = ctx.rng("sprigs")
    for _ in range(60):
        x, y = rng.randrange(4, 60), rng.randrange(4, 62)
        here = c.get(x, y)
        if here == "leaf" and c.get(x + 1, y) == "leaf" and c.get(x, y + 1) == "leaf":
            c.points([(x, y), (x + 1, y)], "sunlit leaf")
            c.px(x + 1, y + 1, "moss")
        elif here == "moss" and c.get(x + 1, y + 1) == "moss":
            c.px(x, y, "leaf")
            c.px(x + 1, y + 1, "moss shadow")
    # warm gold glints where the light is strongest (highlights lean gold)
    for _ in range(80):
        x, y = rng.randrange(4, 60), rng.randrange(2, 40)
        if all(c.get(x + dx, y + dy) == "sunlit leaf" for dx, dy in ((0, 0), (1, 0), (0, 1), (-1, 0))):
            c.px(x, y, "lantern gold")
    sh.bumps(c, ctx.rng("bumps"), sh.FOLIAGE, 14)
    sh.prune(c)
    c.outline("ink")
    return c
